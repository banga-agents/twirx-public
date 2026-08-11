// Package snapshotruntime loads and queries one verified immutable Semantic
// Snapshot. It has no network, mutation, browser, model, or action path.
package snapshotruntime

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/archiveacquire"
	"github.com/typed-web-commons/typed-web/internal/archiveimport"
	"github.com/typed-web-commons/typed-web/internal/archiveprofile"
	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/e2format"
	"github.com/typed-web-commons/typed-web/internal/observation"
	"github.com/typed-web-commons/typed-web/internal/proofbundle"
	"github.com/typed-web-commons/typed-web/internal/scalefixture"
	"github.com/typed-web-commons/typed-web/internal/snapshotartifact"
)

var (
	ErrInvalidSnapshot  = errors.New("snapshotruntime: invalid snapshot")
	ErrUnsupportedQuery = errors.New("snapshotruntime: query feature is not available in the read-only Genesis profile")
)

const MaxRuntimePackets = 100000

type Options struct {
	ExpectedID      dataplane.Digest
	IncludeFixtures bool
}

type Runtime struct {
	directory       string
	manifest        dataplane.SnapshotManifest
	id              dataplane.Digest
	manifestSize    uint64
	includeFixtures bool
	packets         []indexedPacket
	byDigest        map[dataplane.Digest]indexedPacket
	deltas          []indexedDelta
	byDeltaDigest   map[dataplane.Digest]indexedDelta
	proofs          map[dataplane.Digest]snapshotartifact.ProofEntry
	archiveKeys     map[dataplane.Digest]struct{}
	archiveBatches  map[dataplane.Digest]struct{}
	origins         []OriginDescription
	originByID      map[string]OriginDescription
	concepts        []string
	mappings        []snapshotartifact.Mapping
	views           []snapshotartifact.View
	report          snapshotartifact.BuildReport
	semanticLexical map[dataplane.Digest]string
}

type indexedPacket struct {
	Sequence      uint64
	Digest        dataplane.Digest
	Bytes         []byte
	Packet        dataplane.Packet
	EvidenceClass string
}

type indexedDelta struct {
	Sequence uint64
	Digest   dataplane.Digest
	Bytes    []byte
	Delta    dataplane.Delta
}

type verifiedBundle struct {
	Result               e2format.Result
	RepresentationDigest dataplane.Digest
	ArtifactDigests      map[string]dataplane.Digest
	ArtifactSizes        map[string]uint64
}

type verifiedScaleFixture struct {
	Profile              scalefixture.Profile
	Values               map[string]string
	ObservationDigest    dataplane.Digest
	RepresentationDigest dataplane.Digest
	ProfileDigest        dataplane.Digest
	ArtifactDigests      map[string]dataplane.Digest
	ArtifactSizes        map[string]uint64
}

type Description struct {
	SnapshotID      string                        `json:"snapshot_id"`
	Channel         string                        `json:"channel"`
	CreatedAt       string                        `json:"created_at"`
	SourceRevision  string                        `json:"source_revision"`
	Counts          dataplane.SnapshotCounts      `json:"counts"`
	EvidenceClasses []string                      `json:"evidence_classes"`
	Actual          snapshotartifact.ActualCounts `json:"actual"`
	Target          snapshotartifact.TargetCounts `json:"funding_demo_target"`
	Limitations     []string                      `json:"limitations"`
	Execution       string                        `json:"execution"`
}

type Trace struct {
	PacketDigest  string                      `json:"packet_digest"`
	Sequence      uint64                      `json:"sequence"`
	EvidenceClass string                      `json:"evidence_class"`
	Packet        dataplane.Packet            `json:"packet"`
	Proof         snapshotartifact.ProofEntry `json:"proof"`
}

type ViewDescription struct {
	ID              string   `json:"id"`
	Definition      string   `json:"definition"`
	ThroughSequence uint64   `json:"through_sequence"`
	PublicOnly      bool     `json:"public_only"`
	EvidenceClasses []string `json:"evidence_classes"`
	RowCount        uint64   `json:"row_count"`
}

type DeltaDescription struct {
	Sequence                   uint64 `json:"sequence"`
	Digest                     string `json:"digest"`
	Class                      string `json:"class"`
	Kind                       string `json:"kind"`
	SemanticKeyDigest          string `json:"semantic_key_digest"`
	BeforePacketDigest         string `json:"before_packet_digest,omitempty"`
	AfterPacketDigest          string `json:"after_packet_digest,omitempty"`
	BeforeSourceEvidenceDigest string `json:"before_source_evidence_digest,omitempty"`
	AfterSourceEvidenceDigest  string `json:"after_source_evidence_digest,omitempty"`
	OriginID                   string `json:"origin_id"`
	OccurredAt                 string `json:"occurred_at"`
	BatchID                    string `json:"batch_id"`
	CanonVersion               string `json:"canon_version"`
	ReasonCode                 string `json:"reason_code"`
}

// OriginDescription is an immutable Atlas identity carried by the admitted
// snapshot. PacketCount counts public packets in this snapshot; zero does not
// imply that the origin was fetched, reviewed, denied, or found empty.
type OriginDescription struct {
	ID              string `json:"origin_id"`
	CanonicalOrigin string `json:"canonical_origin"`
	CanonicalHost   string `json:"canonical_host"`
	DomainFamily    string `json:"domain_family"`
	CatalogState    string `json:"catalog_state"`
	PacketCount     uint64 `json:"public_packet_count"`
	PacketState     string `json:"packet_state"`
}

type QueryExecution struct {
	Result        dataplane.QueryResult   `json:"result"`
	EconomicEvent dataplane.EconomicEvent `json:"economic_event"`
	Plan          Plan                    `json:"plan"`
}

type Plan struct {
	Mode             string   `json:"mode"`
	SnapshotID       string   `json:"snapshot_id"`
	QueryDigest      string   `json:"query_digest"`
	ScannedPackets   uint64   `json:"scanned_packets"`
	MatchedPackets   uint64   `json:"matched_packets"`
	ReturnedPackets  uint64   `json:"returned_packets"`
	ExcludedFixtures uint64   `json:"excluded_fixtures"`
	Truncated        bool     `json:"truncated"`
	Unsupported      []string `json:"unsupported"`
	NetworkRequests  uint64   `json:"network_requests"`
}

func Open(directory string, options Options) (*Runtime, error) {
	manifest, id, err := dataplane.VerifySnapshotDirectory(directory, options.ExpectedID)
	if err != nil {
		return nil, err
	}
	if manifest.Counts.Packets > MaxRuntimePackets {
		return nil, fmt.Errorf("%w: packet count exceeds local runtime limit", ErrInvalidSnapshot)
	}
	for _, artifact := range manifest.Artifacts {
		if artifact.Size > snapshotartifact.MaxArtifactBytes {
			return nil, fmt.Errorf("%w: artifact %q exceeds local runtime limit", ErrInvalidSnapshot, artifact.Path)
		}
	}
	manifestInfo, err := os.Lstat(filepath.Join(directory, "manifest.cbor"))
	if err != nil || !manifestInfo.Mode().IsRegular() || manifestInfo.Size() <= 0 {
		return nil, fmt.Errorf("%w: manifest changed after verification", ErrInvalidSnapshot)
	}
	r := &Runtime{directory: directory, manifest: manifest, id: id, manifestSize: uint64(manifestInfo.Size()), includeFixtures: options.IncludeFixtures, byDigest: make(map[dataplane.Digest]indexedPacket), byDeltaDigest: make(map[dataplane.Digest]indexedDelta), proofs: make(map[dataplane.Digest]snapshotartifact.ProofEntry), archiveKeys: make(map[dataplane.Digest]struct{}), archiveBatches: make(map[dataplane.Digest]struct{}), originByID: make(map[string]OriginDescription), semanticLexical: make(map[dataplane.Digest]string)}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Runtime) load() error {
	roles := make(map[string][]dataplane.SnapshotArtifact)
	for _, artifact := range r.manifest.Artifacts {
		roles[artifact.Role] = append(roles[artifact.Role], artifact)
	}
	if len(roles["origin_catalog"]) != 1 || len(roles["concepts"]) != 1 || len(roles["mappings"]) != 1 || len(roles["build_report"]) != 1 {
		return fmt.Errorf("%w: singleton artifact roles do not reconcile", ErrInvalidSnapshot)
	}
	selectionPath := filepath.Join(r.directory, filepath.FromSlash(roles["origin_catalog"][0].Path))
	selection, err := atlas.LoadSelection(selectionPath)
	selectionDigest, selectionDigestErr := snapshotartifact.ParseDigest(selection.DigestReference())
	if err != nil || selectionDigestErr != nil || selectionDigest != r.manifest.AtlasSelectionDigest || uint64(len(selection.Candidates)) != r.manifest.Counts.Origins {
		return fmt.Errorf("%w: origin catalog: %v", ErrInvalidSnapshot, err)
	}
	selectionIDs := make(map[string]struct{}, len(selection.Candidates))
	for _, candidate := range selection.Candidates {
		selectionIDs[candidate.ID] = struct{}{}
		description := OriginDescription{ID: candidate.ID, CanonicalOrigin: candidate.CanonicalOrigin, CanonicalHost: candidate.CanonicalHost, DomainFamily: candidate.DomainFamily, CatalogState: string(candidate.Catalog.State), PacketState: "no_public_packets"}
		r.origins = append(r.origins, description)
		r.originByID[description.ID] = description
	}

	conceptBytes, err := r.readArtifact(roles["concepts"][0])
	if err != nil {
		return err
	}
	var concepts snapshotartifact.Concepts
	if err := snapshotartifact.Decode(conceptBytes, &concepts); err != nil {
		return fmt.Errorf("%w: concept catalog: %v", ErrInvalidSnapshot, err)
	}
	moduleDigest, moduleErr := snapshotartifact.ModuleSetDigest(concepts.Modules)
	if concepts.Format != snapshotartifact.ConceptsFormat || !strictSortedUnique(concepts.Concepts) || !strictSortedUnique(concepts.Modules) || moduleErr != nil || moduleDigest != r.manifest.CanonModuleSetDigest || uint64(len(concepts.Concepts)) != r.manifest.Counts.Concepts {
		return fmt.Errorf("%w: concept catalog", ErrInvalidSnapshot)
	}
	r.concepts = concepts.Concepts

	mappingBytes, err := r.readArtifact(roles["mappings"][0])
	if err != nil {
		return err
	}
	var mappings snapshotartifact.Mappings
	if err := snapshotartifact.Decode(mappingBytes, &mappings); err != nil || mappings.Format != snapshotartifact.MappingsFormat || uint64(len(mappings.Mappings)) != r.manifest.Counts.Mappings {
		return fmt.Errorf("%w: mapping catalog", ErrInvalidSnapshot)
	}
	for i, mapping := range mappings.Mappings {
		if mapping.ID == "" || mapping.NativeTerm == "" || mapping.SemanticTerm == "" || mapping.Status != "reviewed" || mapping.EvidenceClass == "" || (i > 0 && mappings.Mappings[i-1].ID >= mapping.ID) {
			return fmt.Errorf("%w: mapping entry %d", ErrInvalidSnapshot, i)
		}
	}
	r.mappings = mappings.Mappings

	for _, artifact := range roles["packet_batch"] {
		data, readErr := r.readArtifact(artifact)
		if readErr != nil {
			return readErr
		}
		segment, decodeErr := snapshotartifact.UnmarshalPacketSegment(data)
		if decodeErr != nil {
			return decodeErr
		}
		for _, entry := range segment.Entries {
			digest, _ := snapshotartifact.ParseDigest(entry.Digest)
			if _, exists := r.byDigest[digest]; exists {
				return fmt.Errorf("%w: duplicate packet", ErrInvalidSnapshot)
			}
			packet, _ := dataplane.UnmarshalPacket(entry.CBOR)
			indexed := indexedPacket{Sequence: entry.Sequence, Digest: digest, Bytes: entry.CBOR, Packet: packet}
			r.byDigest[digest] = indexed
			r.packets = append(r.packets, indexed)
		}
	}
	sort.Slice(r.packets, func(i, j int) bool { return r.packets[i].Sequence < r.packets[j].Sequence })
	if uint64(len(r.packets)) != r.manifest.Counts.Packets || uint64(len(r.packets)) != r.manifest.HighestPacketSequence {
		return fmt.Errorf("%w: packet counts do not reconcile", ErrInvalidSnapshot)
	}
	for index, packet := range r.packets {
		if packet.Sequence != uint64(index+1) {
			return fmt.Errorf("%w: packet sequence has a gap", ErrInvalidSnapshot)
		}
	}

	var proofIndexes int
	var proofEntries int
	bundles := make(map[string]verifiedBundle)
	scaleFixtures := make(map[string]verifiedScaleFixture)
	previousProof := ""
	for _, artifact := range roles["proof_index"] {
		if artifact.Path != "proof/index.json" && !(strings.HasPrefix(artifact.Path, "proof/index-") && strings.HasSuffix(artifact.Path, ".json")) {
			continue
		}
		proofIndexes++
		data, readErr := r.readArtifact(artifact)
		if readErr != nil {
			return readErr
		}
		index, decodeErr := snapshotartifact.UnmarshalProofIndex(data)
		if decodeErr != nil {
			return fmt.Errorf("%w: proof index", ErrInvalidSnapshot)
		}
		for _, entry := range index.Entries {
			if entry.PacketDigest <= previousProof {
				return fmt.Errorf("%w: malformed proof entry", ErrInvalidSnapshot)
			}
			digest, parseErr := snapshotartifact.ParseDigest(entry.PacketDigest)
			if parseErr != nil {
				return parseErr
			}
			packet, exists := r.byDigest[digest]
			if !exists {
				return fmt.Errorf("%w: proof references unknown packet", ErrInvalidSnapshot)
			}
			for _, proof := range entry.Artifacts {
				manifestArtifact, found := r.findArtifact(proof.Path)
				proofDigest, digestErr := snapshotartifact.ParseDigest(proof.Digest)
				if !found || digestErr != nil || manifestArtifact.Role != "proof_index" || manifestArtifact.Digest != proofDigest || manifestArtifact.Size != proof.Size {
					return fmt.Errorf("%w: proof artifact mismatch", ErrInvalidSnapshot)
				}
			}
			if err := r.reconcileProof(entry, packet, bundles, scaleFixtures); err != nil {
				return err
			}
			if entry.EvidenceClass == "test_fixture" {
				if packet.Packet.Epistemic.AuthorityClass != "controlled_test_fixture" {
					return fmt.Errorf("%w: fixture authority mismatch", ErrInvalidSnapshot)
				}
			} else {
				if _, selected := selectionIDs[packet.Packet.Source.OriginID]; !selected || (entry.EvidenceClass == "recorded_offline_replay" && packet.Packet.Epistemic.AuthorityClass != "project_recorded_origin_fixture") || (entry.EvidenceClass == "archive_observation" && packet.Packet.Epistemic.AuthorityClass != "common_crawl_archive_observation") {
					return fmt.Errorf("%w: public packet origin or authority mismatch", ErrInvalidSnapshot)
				}
			}
			packet.EvidenceClass = entry.EvidenceClass
			r.byDigest[digest] = packet
			r.proofs[digest] = entry
			previousProof = entry.PacketDigest
			proofEntries++
		}
	}
	if proofIndexes == 0 || proofEntries != len(r.packets) || uint64(len(roles["proof_index"])-proofIndexes) != r.manifest.Counts.ProofArtifacts {
		return fmt.Errorf("%w: proof artifact count does not reconcile", ErrInvalidSnapshot)
	}
	for index := range r.packets {
		r.packets[index].EvidenceClass = r.byDigest[r.packets[index].Digest].EvidenceClass
		packet := r.packets[index].Packet
		if packet.Derivation.CompilerContractDigest != r.manifest.CompilerContractDigest || packet.Epistemic.FreshnessStatus != "stale" || packet.Lifecycle.State != "stale" || !packetSemanticIndexesReconcile(packet, concepts.Concepts, mappings.Mappings) {
			return fmt.Errorf("%w: packet semantic indexes do not reconcile", ErrInvalidSnapshot)
		}
		if r.packets[index].EvidenceClass != "test_fixture" {
			description, selected := r.originByID[packet.Source.OriginID]
			if !selected {
				return fmt.Errorf("%w: public packet origin absent from Atlas", ErrInvalidSnapshot)
			}
			description.PacketCount++
			description.PacketState = "public_packets_available"
			r.originByID[description.ID] = description
		}
	}
	for index := range r.origins {
		r.origins[index] = r.originByID[r.origins[index].ID]
	}

	for _, artifact := range roles["delta_batch"] {
		data, readErr := r.readArtifact(artifact)
		if readErr != nil {
			return readErr
		}
		segment, decodeErr := snapshotartifact.UnmarshalDeltaSegment(data)
		if decodeErr != nil {
			return decodeErr
		}
		for _, entry := range segment.Entries {
			digest, _ := snapshotartifact.ParseDigest(entry.Digest)
			if _, exists := r.byDeltaDigest[digest]; exists {
				return fmt.Errorf("%w: duplicate delta", ErrInvalidSnapshot)
			}
			delta, _ := dataplane.UnmarshalDelta(entry.CBOR)
			indexed := indexedDelta{Sequence: entry.Sequence, Digest: digest, Bytes: entry.CBOR, Delta: delta}
			if err := r.reconcileDelta(indexed); err != nil {
				return err
			}
			r.byDeltaDigest[digest] = indexed
			r.deltas = append(r.deltas, indexed)
		}
	}
	sort.Slice(r.deltas, func(i, j int) bool { return r.deltas[i].Sequence < r.deltas[j].Sequence })
	if uint64(len(r.deltas)) != r.manifest.Counts.Deltas || uint64(len(r.deltas)) != r.manifest.HighestDeltaSequence {
		return fmt.Errorf("%w: delta counts do not reconcile", ErrInvalidSnapshot)
	}
	for index, delta := range r.deltas {
		if delta.Sequence != uint64(index+1) {
			return fmt.Errorf("%w: delta sequence has a gap", ErrInvalidSnapshot)
		}
	}

	for _, descriptor := range r.manifest.Views {
		artifact, found := findArtifactDigest(roles["materialized_view"], descriptor.ArtifactDigest)
		if !found {
			return fmt.Errorf("%w: materialized view artifact absent", ErrInvalidSnapshot)
		}
		data, readErr := r.readArtifact(artifact)
		if readErr != nil {
			return readErr
		}
		var view snapshotartifact.View
		if err := snapshotartifact.Decode(data, &view); err != nil || view.Format != snapshotartifact.ViewFormat || view.ID != descriptor.ID || uint64(len(view.Rows)) != descriptor.RowCount || view.ThroughSequence != descriptor.ThroughSequence {
			return fmt.Errorf("%w: view %q does not reconcile", ErrInvalidSnapshot, descriptor.ID)
		}
		for _, row := range view.Rows {
			digest, parseErr := snapshotartifact.ParseDigest(row.PacketDigest)
			packet, exists := r.byDigest[digest]
			if parseErr != nil || !exists || (view.PublicOnly && packet.EvidenceClass == "test_fixture") || !viewRowMatches(row, packet.Packet, r.semanticLexical[digest]) {
				return fmt.Errorf("%w: view %q row", ErrInvalidSnapshot, descriptor.ID)
			}
		}
		r.views = append(r.views, view)
	}

	reportBytes, err := r.readArtifact(roles["build_report"][0])
	if err != nil {
		return err
	}
	if err := snapshotartifact.Decode(reportBytes, &r.report); err != nil || r.report.Format != snapshotartifact.BuildReportFormat || r.report.SourceRevision != r.manifest.SourceRevision || r.report.BuiltAt != r.manifest.CreatedAt || r.report.NetworkRequests != 0 || r.report.CurrentClaimsMade || r.report.FixtureCountedPublic || !r.reportReconciles() {
		return fmt.Errorf("%w: build report does not reconcile", ErrInvalidSnapshot)
	}
	return nil
}

func (r *Runtime) reconcileDelta(indexed indexedDelta) error {
	delta := indexed.Delta
	if delta.Class != "origin" || delta.Kind != "modified" || !delta.BeforePacketDigest.Present || !delta.AfterPacketDigest.Present || !delta.BeforeSourceEvidenceDigest.Present || !delta.AfterSourceEvidenceDigest.Present {
		return fmt.Errorf("%w: unsupported delta profile", ErrInvalidSnapshot)
	}
	before, beforeOK := r.byDigest[delta.BeforePacketDigest.Value]
	after, afterOK := r.byDigest[delta.AfterPacketDigest.Value]
	_, keyOK := r.archiveKeys[delta.SemanticKeyDigest]
	_, batchOK := r.archiveBatches[delta.BatchID]
	if !beforeOK || !afterOK || !keyOK || !batchOK || before.EvidenceClass != "archive_observation" || after.EvidenceClass != "archive_observation" || before.Packet.Source.OriginID != delta.OriginID || after.Packet.Source.OriginID != delta.OriginID || before.Packet.Subject.Native != after.Packet.Subject.Native || before.Packet.Predicate.Native != after.Packet.Predicate.Native || before.Packet.Object.NativeLexical == after.Packet.Object.NativeLexical || before.Packet.Source.RepresentationDigest != delta.BeforeSourceEvidenceDigest.Value || after.Packet.Source.RepresentationDigest != delta.AfterSourceEvidenceDigest.Value || after.Packet.Time.ObservedAt != delta.OccurredAt || delta.CanonVersion != "tw:canon@0.1" || delta.ReasonCode != "source_native_title_changed" {
		return fmt.Errorf("%w: archive origin delta does not reconcile", ErrInvalidSnapshot)
	}
	return nil
}

func (r *Runtime) reconcileProof(entry snapshotartifact.ProofEntry, packet indexedPacket, bundles map[string]verifiedBundle, scaleFixtures map[string]verifiedScaleFixture) error {
	switch entry.ProofType {
	case "e2_bundle":
		return r.reconcileE2Proof(entry, packet, bundles)
	case "controlled_scale_fixture":
		return r.reconcileScaleFixtureProof(entry, packet, scaleFixtures)
	case "archive_capture":
		return r.reconcileArchiveProof(entry, packet)
	default:
		return fmt.Errorf("%w: unsupported proof type %q", ErrInvalidSnapshot, entry.ProofType)
	}
}

type archiveKey struct {
	Format    string `json:"format"`
	OriginID  string `json:"origin_id"`
	Subject   string `json:"native_subject"`
	Predicate string `json:"native_predicate"`
	Scope     string `json:"scope"`
}

func (r *Runtime) reconcileArchiveProof(entry snapshotartifact.ProofEntry, indexed indexedPacket) error {
	artifacts := make(map[string]dataplane.SnapshotArtifact, len(entry.Artifacts))
	bytesByName := make(map[string][]byte, len(entry.Artifacts))
	for _, proof := range entry.Artifacts {
		artifact, found := r.findArtifact(proof.Path)
		if !found {
			return fmt.Errorf("%w: archive proof artifact absent", ErrInvalidSnapshot)
		}
		data, err := r.readArtifact(artifact)
		if err != nil {
			return err
		}
		artifacts[proof.Name], bytesByName[proof.Name] = artifact, data
	}
	needed := []string{"acquisition-manifest.json", "adapter.json", "capture.json", "extraction-plan.json", "representation.body", "semantic-key.json", "spool-manifest.json"}
	for _, name := range needed {
		if len(bytesByName[name]) == 0 {
			return fmt.Errorf("%w: archive proof omits %s", ErrInvalidSnapshot, name)
		}
	}
	acquisitionPath := filepath.Join(r.directory, filepath.FromSlash(artifacts["acquisition-manifest.json"].Path))
	acquisitionRoot := filepath.Dir(acquisitionPath)
	manifest, err := archiveacquire.Verify(acquisitionRoot)
	if err != nil {
		return fmt.Errorf("%w: archive acquisition: %v", ErrInvalidSnapshot, err)
	}
	spoolManifestPath := filepath.Join(r.directory, filepath.FromSlash(artifacts["spool-manifest.json"].Path))
	spoolRoot := filepath.Dir(spoolManifestPath)
	relativeSpool, err := filepath.Rel(acquisitionRoot, spoolRoot)
	if err != nil || relativeSpool == "." || strings.HasPrefix(relativeSpool, "..") || filepath.IsAbs(relativeSpool) {
		return fmt.Errorf("%w: archive spool escaped acquisition", ErrInvalidSnapshot)
	}
	evidence, err := archiveimport.VerifySpool(spoolRoot)
	if err != nil {
		return fmt.Errorf("%w: archive spool: %v", ErrInvalidSnapshot, err)
	}
	acquisitionMatch := false
	for _, capture := range manifest.Captures {
		if filepath.FromSlash(capture.SpoolPath) == relativeSpool && capture.CaptureTimestamp == evidence.CaptureTimestamp && capture.RepresentationDigest == evidence.RepresentationDigest {
			acquisitionMatch = true
			break
		}
	}
	if !acquisitionMatch {
		return fmt.Errorf("%w: archive spool is not admitted by acquisition manifest", ErrInvalidSnapshot)
	}
	profile, err := archiveprofile.ParseSpec(bytesByName["adapter.json"])
	if err != nil {
		return fmt.Errorf("%w: archive profile: %v", ErrInvalidSnapshot, err)
	}
	expectedPlan, err := archiveprofile.PlanBytes(profile)
	if err != nil || !bytes.Equal(expectedPlan, bytesByName["extraction-plan.json"]) {
		return fmt.Errorf("%w: archive extraction plan", ErrInvalidSnapshot)
	}
	statement, err := archiveprofile.ExtractTitle(bytesByName["representation.body"])
	if err != nil {
		return fmt.Errorf("%w: archive native extraction: %v", ErrInvalidSnapshot, err)
	}
	var key archiveKey
	if err := snapshotartifact.Decode(bytesByName["semantic-key.json"], &key); err != nil || key.Format != "tw.semantic-key/0.1" || key.OriginID != profile.OriginID || key.Subject != profile.Subject || key.Predicate != profile.Predicate || key.Scope != "historical_archive" {
		return fmt.Errorf("%w: archive semantic key", ErrInvalidSnapshot)
	}
	semanticKeyDigest := dataplane.DigestBytes(bytesByName["semantic-key.json"])
	r.archiveKeys[semanticKeyDigest] = struct{}{}
	acquisitionID := dataplane.DigestBytes(bytesByName["acquisition-manifest.json"])
	r.archiveBatches[acquisitionID] = struct{}{}
	representationDigest, err := snapshotartifact.ParseDigest(evidence.RepresentationDigest)
	if err != nil || representationDigest != dataplane.DigestBytes(bytesByName["representation.body"]) {
		return fmt.Errorf("%w: archive representation identity", ErrInvalidSnapshot)
	}
	observed, err := time.Parse("20060102150405", evidence.CaptureTimestamp)
	if err != nil {
		return fmt.Errorf("%w: archive capture time", ErrInvalidSnapshot)
	}
	p := indexed.Packet
	if snapshotartifact.DigestReference(dataplane.DigestBytes(bytesByName["capture.json"])) != entry.EvidenceID || entry.ExecutionOriginID != profile.OriginID || entry.OperationID != profile.OperationID || entry.FieldID != profile.Predicate || evidence.OriginID != profile.OriginID || evidence.CurrentPublisherStatement || p.Kind != profile.Kind || p.Subject.Native != profile.Subject || len(p.Subject.CanonicalCandidates) != 0 || p.Predicate.Native != profile.Predicate || p.Predicate.Semantic.Present || p.Object.NativeStatus != "resolved" || p.Object.NativeLexical != statement.NativeLexical || !p.Object.MediaType.Present || p.Object.MediaType.Value != profile.MediaType || p.Object.Typed != nil || p.Context.Scope.Value != "historical_archive" || p.Time.ObservedAt != observed.UTC().Format(time.RFC3339) || p.Source.OriginID != profile.OriginID || p.Source.RepresentationDigest != representationDigest || p.Source.Locator != statement.Locator || !p.Source.NativeSchemaRef.Present || p.Source.NativeSchemaRef.Value != "archive:html-title@0.1" || p.Derivation.ObservationDigest != dataplane.DigestBytes(bytesByName["capture.json"]) || p.Derivation.TransportDigest.Present || p.Derivation.AdapterDigest != dataplane.DigestBytes(bytesByName["adapter.json"]) || p.Derivation.ExtractionPlanDigest != dataplane.DigestBytes(expectedPlan) || len(p.Derivation.TransformationIDs) != 0 || len(p.Derivation.MappingIDs) != 0 || p.Derivation.SemanticClosureDigest.Present || p.Epistemic.Lane != "observed_native" || p.Epistemic.ExtractionStatus != "deterministic" || p.Epistemic.MappingStatus != "none" || p.Epistemic.AuthorityClass != "common_crawl_archive_observation" || p.Epistemic.FreshnessStatus != "stale" || p.Lifecycle.State != "stale" || p.Retention != "public_archival" || p.Disclosure != "public" {
		return fmt.Errorf("%w: archive packet derivation", ErrInvalidSnapshot)
	}
	r.semanticLexical[indexed.Digest] = ""
	return nil
}

func (r *Runtime) reconcileE2Proof(entry snapshotartifact.ProofEntry, packet indexedPacket, cache map[string]verifiedBundle) error {
	bundle, exists := cache[entry.EvidenceID]
	if !exists {
		artifactDigests := make(map[string]dataplane.Digest, len(entry.Artifacts))
		artifactSizes := make(map[string]uint64, len(entry.Artifacts))
		artifactBytes := make(map[string][]byte, len(entry.Artifacts))
		previous := ""
		for _, proof := range entry.Artifacts {
			if proof.Name <= previous {
				return fmt.Errorf("%w: proof artifacts are not strictly sorted", ErrInvalidSnapshot)
			}
			manifestArtifact, found := r.findArtifact(proof.Path)
			if !found {
				return fmt.Errorf("%w: proof artifact absent", ErrInvalidSnapshot)
			}
			data, err := r.readArtifact(manifestArtifact)
			if err != nil {
				return err
			}
			digest := dataplane.DigestBytes(data)
			artifactDigests[proof.Name] = digest
			artifactSizes[proof.Name] = uint64(len(data))
			artifactBytes[proof.Name] = data
			previous = proof.Name
		}
		manifestBytes, ok := artifactBytes[proofbundle.ManifestName]
		if !ok || snapshotartifact.DigestReference(dataplane.DigestBytes(manifestBytes)) != entry.EvidenceID {
			return fmt.Errorf("%w: proof bundle identity mismatch", ErrInvalidSnapshot)
		}
		manifest, err := proofbundle.UnmarshalManifest(manifestBytes)
		if err != nil || len(manifest.Entries)+1 != len(entry.Artifacts) {
			return fmt.Errorf("%w: proof bundle manifest: %v", ErrInvalidSnapshot, err)
		}
		for _, manifestEntry := range manifest.Entries {
			if artifactDigests[manifestEntry.Name] != dataplane.Digest(manifestEntry.Digest) || artifactSizes[manifestEntry.Name] != manifestEntry.Size {
				return fmt.Errorf("%w: proof bundle constituent mismatch", ErrInvalidSnapshot)
			}
		}
		resultBytes, ok := artifactBytes["result.cbor"]
		if !ok || manifest.ResultID != snapshotartifact.DigestReference(dataplane.DigestBytes(resultBytes)) {
			return fmt.Errorf("%w: proof result identity mismatch", ErrInvalidSnapshot)
		}
		result, err := e2format.UnmarshalResult(resultBytes)
		if err != nil {
			return fmt.Errorf("%w: proof result: %v", ErrInvalidSnapshot, err)
		}
		checks := []struct {
			name     string
			expected dataplane.Digest
		}{
			{"observation.cbor", dataplane.Digest(result.ObservationDigest)},
			{"transport.cbor", dataplane.Digest(result.TransportDigest)},
			{"adapter.cbor", dataplane.Digest(result.AdapterDigest)},
			{"contract.cbor", dataplane.Digest(result.ContractDigest)},
			{"semantic-closure.cbor", dataplane.Digest(result.SemanticClosureDigest)},
		}
		for _, check := range checks {
			if artifactDigests[check.name] != check.expected {
				return fmt.Errorf("%w: %s is not bound by result", ErrInvalidSnapshot, check.name)
			}
		}
		representation, ok := artifactBytes["representation.body"]
		if !ok {
			return fmt.Errorf("%w: representation body absent", ErrInvalidSnapshot)
		}
		bundle = verifiedBundle{Result: result, RepresentationDigest: dataplane.DigestBytes(representation), ArtifactDigests: artifactDigests, ArtifactSizes: artifactSizes}
		cache[entry.EvidenceID] = bundle
	} else {
		if len(entry.Artifacts) != len(bundle.ArtifactDigests) {
			return fmt.Errorf("%w: inconsistent repeated proof bundle", ErrInvalidSnapshot)
		}
		for _, proof := range entry.Artifacts {
			digest, err := snapshotartifact.ParseDigest(proof.Digest)
			if err != nil || bundle.ArtifactDigests[proof.Name] != digest || bundle.ArtifactSizes[proof.Name] != proof.Size {
				return fmt.Errorf("%w: repeated proof reference mismatch", ErrInvalidSnapshot)
			}
		}
	}
	if bundle.Result.OriginID != entry.ExecutionOriginID || bundle.Result.OperationID != entry.OperationID || bundle.Result.ObservedAt != packet.Packet.Time.ObservedAt || packet.Packet.Source.RepresentationDigest != bundle.RepresentationDigest || packet.Packet.Derivation.ObservationDigest != dataplane.Digest(bundle.Result.ObservationDigest) || !packet.Packet.Derivation.TransportDigest.Present || packet.Packet.Derivation.TransportDigest.Value != dataplane.Digest(bundle.Result.TransportDigest) || packet.Packet.Derivation.AdapterDigest != dataplane.Digest(bundle.Result.AdapterDigest) || !packet.Packet.Derivation.SemanticClosureDigest.Present || packet.Packet.Derivation.SemanticClosureDigest.Value != dataplane.Digest(bundle.Result.SemanticClosureDigest) {
		return fmt.Errorf("%w: packet derivation does not match proof result", ErrInvalidSnapshot)
	}
	var field *e2format.Field
	for index := range bundle.Result.Fields {
		if bundle.Result.Fields[index].ID == entry.FieldID {
			field = &bundle.Result.Fields[index]
			break
		}
	}
	if field == nil || packet.Packet.Predicate.Native != field.NativeTerm || !packet.Packet.Predicate.Semantic.Present || packet.Packet.Predicate.Semantic.Value != field.SemanticTerm || packet.Packet.Source.Locator != field.NativeLocator || packet.Packet.Object.NativeStatus != field.Status || packet.Packet.Object.NativeLexical != field.NativeLexical || !sameStrings(packet.Packet.Derivation.TransformationIDs, field.Transforms) || len(packet.Packet.Derivation.MappingIDs) != 1 || packet.Packet.Derivation.MappingIDs[0] != field.Mapping {
		return fmt.Errorf("%w: packet field does not match proof result", ErrInvalidSnapshot)
	}
	if packet.Packet.Object.Typed != nil && packet.Packet.Object.Typed.Lexical != field.SemanticLexical {
		return fmt.Errorf("%w: typed packet lexical does not match proof result", ErrInvalidSnapshot)
	}
	r.semanticLexical[packet.Digest] = field.SemanticLexical
	return nil
}

func (r *Runtime) reconcileScaleFixtureProof(entry snapshotartifact.ProofEntry, packet indexedPacket, cache map[string]verifiedScaleFixture) error {
	verified, exists := cache[entry.EvidenceID]
	if !exists {
		artifactDigests := make(map[string]dataplane.Digest, len(entry.Artifacts))
		artifactSizes := make(map[string]uint64, len(entry.Artifacts))
		artifactBytes := make(map[string][]byte, len(entry.Artifacts))
		for _, proof := range entry.Artifacts {
			manifestArtifact, found := r.findArtifact(proof.Path)
			if !found {
				return fmt.Errorf("%w: scale fixture proof artifact absent", ErrInvalidSnapshot)
			}
			data, err := r.readArtifact(manifestArtifact)
			if err != nil {
				return err
			}
			artifactDigests[proof.Name] = dataplane.DigestBytes(data)
			artifactSizes[proof.Name] = uint64(len(data))
			artifactBytes[proof.Name] = data
		}
		observationBytes, observationOK := artifactBytes["observation.cbor"]
		body, bodyOK := artifactBytes["representation.body"]
		profileBytes, profileOK := artifactBytes["scale-profile.json"]
		if !observationOK || !bodyOK || !profileOK || len(artifactBytes) != 3 {
			return fmt.Errorf("%w: scale fixture proof constituents", ErrInvalidSnapshot)
		}
		envelope, err := observation.UnmarshalCBOR(observationBytes)
		if err != nil {
			return fmt.Errorf("%w: scale fixture observation: %v", ErrInvalidSnapshot, err)
		}
		observationDigest := dataplane.DigestBytes(observationBytes)
		if snapshotartifact.DigestReference(observationDigest) != entry.EvidenceID || envelope.BodySHA256 != dataplane.DigestBytes(body) || envelope.BodySize != uint64(len(body)) || envelope.RequestURL != scalefixture.RequestURL || envelope.FinalURL != scalefixture.RequestURL || envelope.ObserverID != scalefixture.ObserverID || envelope.PolicyID != scalefixture.PolicyID || envelope.MediaType != "application/json" || envelope.Status != 200 {
			return fmt.Errorf("%w: scale fixture observation binding", ErrInvalidSnapshot)
		}
		profile, err := scalefixture.UnmarshalProfile(profileBytes)
		if err != nil || profile.ObservedAt != envelope.RetrievedAt {
			return fmt.Errorf("%w: scale fixture profile: %v", ErrInvalidSnapshot, err)
		}
		fields, err := scalefixture.ParseBody(body, profile)
		if err != nil {
			return fmt.Errorf("%w: scale fixture representation: %v", ErrInvalidSnapshot, err)
		}
		values := make(map[string]string, len(fields))
		for _, field := range fields {
			values[field[0]] = field[1]
		}
		verified = verifiedScaleFixture{Profile: profile, Values: values, ObservationDigest: observationDigest, RepresentationDigest: dataplane.DigestBytes(body), ProfileDigest: dataplane.DigestBytes(profileBytes), ArtifactDigests: artifactDigests, ArtifactSizes: artifactSizes}
		cache[entry.EvidenceID] = verified
	} else {
		if len(entry.Artifacts) != len(verified.ArtifactDigests) {
			return fmt.Errorf("%w: inconsistent repeated scale proof", ErrInvalidSnapshot)
		}
		for _, proof := range entry.Artifacts {
			digest, err := snapshotartifact.ParseDigest(proof.Digest)
			if err != nil || verified.ArtifactDigests[proof.Name] != digest || verified.ArtifactSizes[proof.Name] != proof.Size {
				return fmt.Errorf("%w: repeated scale proof reference mismatch", ErrInvalidSnapshot)
			}
		}
	}
	value, ok := verified.Values[entry.FieldID]
	planBytes, planErr := scalefixture.PlanBytes(entry.FieldID)
	p := packet.Packet
	if !ok || planErr != nil || entry.ExecutionOriginID != scalefixture.OriginID || entry.OperationID != scalefixture.OperationID || p.Source.OriginID != scalefixture.OriginID || p.Subject.Native != scalefixture.SubjectID || p.Predicate.Native != entry.FieldID || p.Predicate.Semantic.Present || p.Source.Locator != scalefixture.Locator(entry.FieldID) || p.Object.NativeStatus != "resolved" || p.Object.NativeLexical != value || p.Time.ObservedAt != verified.Profile.ObservedAt || p.Source.RepresentationDigest != verified.RepresentationDigest || p.Derivation.ObservationDigest != verified.ObservationDigest || p.Derivation.TransportDigest.Present || p.Derivation.AdapterDigest != verified.ProfileDigest || p.Derivation.ExtractionPlanDigest != dataplane.DigestBytes(planBytes) || len(p.Derivation.TransformationIDs) != 0 || len(p.Derivation.MappingIDs) != 0 || p.Derivation.SemanticClosureDigest.Present || p.Epistemic.Lane != "observed_native" || p.Epistemic.MappingStatus != "none" || p.Epistemic.AuthorityClass != "controlled_test_fixture" {
		return fmt.Errorf("%w: scale fixture packet derivation", ErrInvalidSnapshot)
	}
	r.semanticLexical[packet.Digest] = ""
	return nil
}

func (r *Runtime) reportReconciles() bool {
	publicOrigins := make(map[string]struct{})
	fixtureOrigins := make(map[string]struct{})
	archiveOrigins := make(map[string]struct{})
	operations := make(map[string]struct{})
	hasScaleFixture := false
	var resolved, unresolved, publicPackets, fixturePackets uint64
	for _, packet := range r.packets {
		proof := r.proofs[packet.Digest]
		operations[proof.OperationID] = struct{}{}
		if proof.ProofType == "controlled_scale_fixture" {
			hasScaleFixture = true
		}
		if packet.EvidenceClass == "test_fixture" {
			fixtureOrigins[packet.Packet.Source.OriginID] = struct{}{}
			fixturePackets++
		} else {
			publicOrigins[packet.Packet.Source.OriginID] = struct{}{}
			publicPackets++
			if packet.EvidenceClass == "archive_observation" {
				archiveOrigins[packet.Packet.Source.OriginID] = struct{}{}
			}
		}
		if packet.Packet.Object.NativeStatus == "resolved" {
			resolved++
		} else {
			unresolved++
		}
	}
	actual := r.report.Actual
	target := r.report.FundingDemoTarget
	hasArchive := len(archiveOrigins) > 0
	expectedMode := "offline_e2_replay"
	expectedEvidence := []string{"recorded_offline_replay", "test_fixture"}
	if hasArchive {
		expectedMode = "offline_e2_replay_with_archive_observations"
		expectedEvidence = []string{"archive_observation", "recorded_offline_replay", "test_fixture"}
	}
	if hasScaleFixture && hasArchive {
		expectedMode = "offline_e2_replay_with_archive_observations_and_controlled_scale_fixture"
	} else if hasScaleFixture {
		expectedMode = "offline_e2_replay_with_controlled_scale_fixture"
	}
	return r.report.Mode == expectedMode && validLimitations(r.report.Limitations) && sameStrings(r.manifest.EvidenceClasses, expectedEvidence) && target == (snapshotartifact.TargetCounts{AtlasIdentities: 500, ArchiveProfiles: 100, CurrentObserved: 25, Packets: 25000, MaterializedViews: 2}) && actual.AtlasIdentities == r.manifest.Counts.Origins && actual.ArchiveProfiles == uint64(len(archiveOrigins)) && actual.PublicOrigins == uint64(len(publicOrigins)) && actual.FixtureOrigins == uint64(len(fixtureOrigins)) && actual.Operations == uint64(len(operations)) && actual.Packets == r.manifest.Counts.Packets && actual.PublicPackets == publicPackets && actual.FixturePackets == fixturePackets && actual.ResolvedPackets == resolved && actual.UnresolvedPackets == unresolved && actual.Deltas == r.manifest.Counts.Deltas && actual.MaterializedViews == r.manifest.Counts.Views && actual.ProofArtifacts == r.manifest.Counts.ProofArtifacts
}

func validLimitations(values []string) bool {
	if len(values) == 0 || len(values) > 32 {
		return false
	}
	for _, value := range values {
		if value == "" || len(value) > 2048 || strings.ContainsRune(value, '\x00') {
			return false
		}
	}
	return true
}

func viewRowMatches(row snapshotartifact.ViewRow, packet dataplane.Packet, semanticLexical string) bool {
	return row.OriginID == packet.Source.OriginID && row.SubjectID == packet.Subject.Native && row.NativeTerm == packet.Predicate.Native && row.NativeLexical == packet.Object.NativeLexical && row.NativeStatus == packet.Object.NativeStatus && row.SemanticTerm == packet.Predicate.Semantic.Value && row.SemanticLexical == semanticLexical && row.Lane == packet.Epistemic.Lane && row.ObservedAt == packet.Time.ObservedAt
}

func mappingExists(mappings []snapshotartifact.Mapping, native, semantic, status string) bool {
	for _, mapping := range mappings {
		if mapping.NativeTerm == native && mapping.SemanticTerm == semantic && mapping.Status == status {
			return true
		}
	}
	return false
}

func packetSemanticIndexesReconcile(packet dataplane.Packet, concepts []string, mappings []snapshotartifact.Mapping) bool {
	if !allContained(concepts, packet.Subject.CanonicalCandidates) {
		return false
	}
	if packet.Epistemic.Lane == "observed_native" {
		return !packet.Predicate.Semantic.Present && len(packet.Derivation.MappingIDs) == 0 && packet.Epistemic.MappingStatus == "none"
	}
	return packet.Predicate.Semantic.Present && contains(concepts, packet.Predicate.Semantic.Value) && mappingExists(mappings, packet.Predicate.Native, packet.Predicate.Semantic.Value, packet.Epistemic.MappingStatus)
}

func allContained(values, candidates []string) bool {
	for _, candidate := range candidates {
		if !contains(values, candidate) {
			return false
		}
	}
	return true
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (r *Runtime) Describe() Description {
	return Description{SnapshotID: snapshotartifact.DigestReference(r.id), Channel: r.manifest.Channel, CreatedAt: r.manifest.CreatedAt, SourceRevision: r.manifest.SourceRevision, Counts: r.manifest.Counts, EvidenceClasses: append([]string(nil), r.manifest.EvidenceClasses...), Actual: r.report.Actual, Target: r.report.FundingDemoTarget, Limitations: append([]string(nil), r.report.Limitations...), Execution: "immutable_materialized_snapshot_only"}
}

func (r *Runtime) Concepts() []string { return append([]string(nil), r.concepts...) }

func (r *Runtime) ConceptPage(offset, limit int) ([]string, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(r.concepts) {
		offset = len(r.concepts)
	}
	if limit < 0 {
		limit = 0
	}
	end := offset + limit
	if end > len(r.concepts) {
		end = len(r.concepts)
	}
	return append([]string(nil), r.concepts[offset:end]...), len(r.concepts)
}

// OriginPage returns immutable Atlas identities in their canonical selection
// order. It never profiles, retrieves, invokes, or changes an origin.
func (r *Runtime) OriginPage(offset, limit int) ([]OriginDescription, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(r.origins) {
		offset = len(r.origins)
	}
	if limit < 0 {
		limit = 0
	}
	end := offset + limit
	if end > len(r.origins) {
		end = len(r.origins)
	}
	return append([]OriginDescription(nil), r.origins[offset:end]...), len(r.origins)
}

// DescribeOrigin returns one immutable Atlas identity by its exact canonical
// ID. The boolean is false for any value outside the admitted selection.
func (r *Runtime) DescribeOrigin(id string) (OriginDescription, bool) {
	description, found := r.originByID[id]
	return description, found
}

func (r *Runtime) DescribeViews() []ViewDescription {
	result := make([]ViewDescription, 0, len(r.views))
	for _, view := range r.views {
		result = append(result, ViewDescription{ID: view.ID, Definition: view.Definition, ThroughSequence: view.ThroughSequence, PublicOnly: view.PublicOnly, EvidenceClasses: append([]string(nil), view.EvidenceClasses...), RowCount: uint64(len(view.Rows))})
	}
	return result
}

// DeltaPage returns a bounded copy of the immutable delta stream. The stream
// is ordered by snapshot sequence and never performs origin access.
func (r *Runtime) DeltaPage(offset, limit int) ([]DeltaDescription, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(r.deltas) {
		offset = len(r.deltas)
	}
	if limit < 0 {
		limit = 0
	}
	end := offset + limit
	if end > len(r.deltas) {
		end = len(r.deltas)
	}
	result := make([]DeltaDescription, 0, end-offset)
	for _, delta := range r.deltas[offset:end] {
		value := delta.Delta
		result = append(result, DeltaDescription{Sequence: delta.Sequence, Digest: snapshotartifact.DigestReference(delta.Digest), Class: value.Class, Kind: value.Kind, SemanticKeyDigest: snapshotartifact.DigestReference(value.SemanticKeyDigest), BeforePacketDigest: optionalDigestReference(value.BeforePacketDigest), AfterPacketDigest: optionalDigestReference(value.AfterPacketDigest), BeforeSourceEvidenceDigest: optionalDigestReference(value.BeforeSourceEvidenceDigest), AfterSourceEvidenceDigest: optionalDigestReference(value.AfterSourceEvidenceDigest), OriginID: value.OriginID, OccurredAt: value.OccurredAt, BatchID: snapshotartifact.DigestReference(value.BatchID), CanonVersion: value.CanonVersion, ReasonCode: value.ReasonCode})
	}
	return result, len(r.deltas)
}

func optionalDigestReference(value dataplane.OptionalDigest) string {
	if !value.Present {
		return ""
	}
	return snapshotartifact.DigestReference(value.Value)
}

// DeltaCBOR returns the exact canonical delta core identified by reference.
func (r *Runtime) DeltaCBOR(reference string) ([]byte, error) {
	digest, err := snapshotartifact.ParseDigest(reference)
	if err != nil {
		return nil, err
	}
	delta, exists := r.byDeltaDigest[digest]
	if !exists {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), delta.Bytes...), nil
}

func (r *Runtime) Trace(reference string) (Trace, error) {
	digest, err := snapshotartifact.ParseDigest(reference)
	if err != nil {
		return Trace{}, err
	}
	packet, exists := r.byDigest[digest]
	if !exists || (!r.includeFixtures && packet.EvidenceClass == "test_fixture") {
		return Trace{}, os.ErrNotExist
	}
	return Trace{PacketDigest: reference, Sequence: packet.Sequence, EvidenceClass: packet.EvidenceClass, Packet: packet.Packet, Proof: r.proofs[digest]}, nil
}

func (r *Runtime) PacketCBOR(reference string) ([]byte, error) {
	digest, err := snapshotartifact.ParseDigest(reference)
	if err != nil {
		return nil, err
	}
	packet, exists := r.byDigest[digest]
	if !exists || (!r.includeFixtures && packet.EvidenceClass == "test_fixture") {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), packet.Bytes...), nil
}

func (r *Runtime) ManifestCBOR() ([]byte, error) {
	path := filepath.Join(r.directory, "manifest.cbor")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || uint64(info.Size()) != r.manifestSize {
		return nil, fmt.Errorf("%w: manifest changed after admission", ErrInvalidSnapshot)
	}
	data, err := os.ReadFile(path)
	if err != nil || dataplane.DigestBytes(data) != r.id {
		return nil, fmt.Errorf("%w: manifest identity changed after admission", ErrInvalidSnapshot)
	}
	return data, nil
}

func (r *Runtime) ProofArtifact(reference, name string) ([]byte, string, error) {
	digest, err := snapshotartifact.ParseDigest(reference)
	if err != nil {
		return nil, "", err
	}
	packet, exists := r.byDigest[digest]
	if !exists || (!r.includeFixtures && packet.EvidenceClass == "test_fixture") {
		return nil, "", os.ErrNotExist
	}
	proof := r.proofs[digest]
	for _, candidate := range proof.Artifacts {
		if candidate.Name != name {
			continue
		}
		artifact, found := r.findArtifact(candidate.Path)
		if !found {
			return nil, "", fmt.Errorf("%w: admitted proof artifact disappeared", ErrInvalidSnapshot)
		}
		data, err := r.readArtifact(artifact)
		return data, artifact.MediaType, err
	}
	return nil, "", os.ErrNotExist
}

func (r *Runtime) Query(query dataplane.Query) (QueryExecution, error) {
	return r.QueryContext(context.Background(), query)
}

func (r *Runtime) QueryContext(parent context.Context, query dataplane.Query) (QueryExecution, error) {
	var execution QueryExecution
	if err := query.Validate(); err != nil {
		return execution, err
	}
	if query.Execution.AllowLiveRefresh || query.Execution.MaximumLiveOrigins != 0 || !query.Execution.AllowMaterializedState {
		return execution, fmt.Errorf("%w: live refresh", ErrUnsupportedQuery)
	}
	unsupported := make([]string, 0, 4)
	if query.Ontology.MaximumDepth != 0 || query.Ontology.MaximumPathCostMillionths != 0 {
		unsupported = append(unsupported, "ontology_expansion")
	}
	if len(query.Dimensions) != 0 {
		unsupported = append(unsupported, "dimension_relations")
	}
	if query.Economics.MaximumPrice.Present || len(query.Economics.AllowedFundingClasses) != 0 {
		unsupported = append(unsupported, "economic_filtering")
	}
	if query.Proof.Level != "packet" {
		unsupported = append(unsupported, "non_packet_proof_level")
	}
	if query.Conflicts != "preserve_sources" {
		unsupported = append(unsupported, "conflict_resolution")
	}
	if query.Preference != "highest_proof" {
		unsupported = append(unsupported, "preference_ranking")
	}
	if len(unsupported) != 0 {
		return execution, fmt.Errorf("%w: %s", ErrUnsupportedQuery, strings.Join(unsupported, ","))
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(query.Execution.DeadlineMilliseconds)*time.Millisecond)
	defer cancel()
	queryBytes, _ := dataplane.MarshalQuery(query)
	queryDigest := dataplane.DigestBytes(queryBytes)
	plan := Plan{Mode: "materialized_snapshot", SnapshotID: snapshotartifact.DigestReference(r.id), QueryDigest: snapshotartifact.DigestReference(queryDigest), ScannedPackets: uint64(len(r.packets)), Unsupported: []string{}, NetworkRequests: 0}
	matched := make([]indexedPacket, 0)
	for index, packet := range r.packets {
		if index%128 == 0 {
			select {
			case <-ctx.Done():
				return execution, fmt.Errorf("snapshotruntime: query deadline: %w", ctx.Err())
			default:
			}
		}
		if !r.includeFixtures && packet.EvidenceClass == "test_fixture" {
			plan.ExcludedFixtures++
			continue
		}
		if matches(query, packet.Packet, r.manifest.CreatedAt) {
			matched = append(matched, packet)
		}
	}
	plan.MatchedPackets = uint64(len(matched))
	maximum := query.Limits.MaximumResults
	if query.Limits.MaximumPackets < maximum {
		maximum = query.Limits.MaximumPackets
	}
	if uint64(len(matched)) > maximum {
		matched = matched[:maximum]
		plan.Truncated = true
	}
	rows := make([]dataplane.QueryResultRow, 0, len(matched))
	proofs := []dataplane.DigestSizeEntry{{Digest: r.id, Size: r.manifestSize}}
	var proofBytes uint64 = r.manifestSize
	origins := make(map[string]struct{})
	for _, candidate := range matched {
		if proofBytes+uint64(len(candidate.Bytes)) > query.Limits.MaximumProofBytes {
			plan.Truncated = true
			break
		}
		packet := candidate.Packet
		predicate := packet.Predicate.Native
		if packet.Predicate.Semantic.Present {
			predicate = packet.Predicate.Semantic.Value
		}
		rows = append(rows, dataplane.QueryResultRow{SubjectID: packet.Subject.Native, PredicateID: predicate, Status: packet.Object.NativeStatus, NativeTerm: packet.Predicate.Native, NativeLocator: packet.Source.Locator, NativeLexical: packet.Object.NativeLexical, SemanticTerm: packet.Predicate.Semantic, Typed: packet.Object.Typed, OriginID: packet.Source.OriginID, PacketDigest: candidate.Digest, ObservationDigest: packet.Derivation.ObservationDigest, Lane: packet.Epistemic.Lane, ObservedAt: packet.Time.ObservedAt})
		proofs = append(proofs, dataplane.DigestSizeEntry{Digest: candidate.Digest, Size: uint64(len(candidate.Bytes))})
		proofBytes += uint64(len(candidate.Bytes))
		origins[packet.Source.OriginID] = struct{}{}
	}
	plan.ReturnedPackets = uint64(len(rows))
	sortRows(rows)
	sort.Slice(proofs, func(i, j int) bool { return bytes.Compare(proofs[i].Digest[:], proofs[j].Digest[:]) < 0 })
	status := "resolved"
	if len(rows) == 0 || uint64(len(origins)) < query.Sources.MinimumDistinctOrigins {
		rows = nil
		proofs = []dataplane.DigestSizeEntry{{Digest: r.id, Size: r.manifestSize}}
		status = "unresolved"
	} else if plan.Truncated {
		status = "partial"
	}
	planBytes, _ := snapshotartifact.Marshal(plan)
	planDigest := dataplane.DigestBytes(planBytes)
	economic := dataplane.EconomicEvent{Version: dataplane.EconomicEventVersion, EventID: "query-" + hex.EncodeToString(queryDigest[:8]), OccurredAt: r.manifest.CreatedAt, WorkType: "snapshot_query", QueryDigest: dataplane.OptionalDigest{Present: true, Value: queryDigest}, Resources: dataplane.EconomicResources{ProofBytesReturned: proofBytes}, FundingClass: "public_commons", Cost: dataplane.EconomicMoney{Currency: "USD", Amount: "0", Class: "not_measured"}, Revenue: dataplane.EconomicMoney{Currency: "USD", Amount: "0", Class: "not_measured"}, MeasurementMethod: "deterministic_snapshot_accounting"}
	economicBytes, err := dataplane.MarshalEconomicEvent(economic)
	if err != nil {
		return execution, err
	}
	result := dataplane.QueryResult{Version: dataplane.QueryResultVersion, QueryDigest: queryDigest, PlanDigest: planDigest, Preference: query.Preference, SnapshotSequence: r.manifest.HighestPacketSequence, Status: status, Rows: rows, ProofArtifacts: proofs, EconomicEventDigest: dataplane.DigestBytes(economicBytes), GeneratedAt: r.manifest.CreatedAt}
	if _, err := dataplane.MarshalQueryResult(result); err != nil {
		return execution, err
	}
	return QueryExecution{Result: result, EconomicEvent: economic, Plan: plan}, nil
}

func matches(query dataplane.Query, packet dataplane.Packet, snapshotTime string) bool {
	if !containsAny(query.Select, packet.Predicate.Native, packet.Predicate.Semantic.Value) {
		return false
	}
	if query.Subject.Concept.Present && !contains(packet.Subject.CanonicalCandidates, query.Subject.Concept.Value) {
		return false
	}
	if len(query.Subject.IDs) != 0 && !containsAny(query.Subject.IDs, append([]string{packet.Subject.Native}, packet.Subject.CanonicalCandidates...)...) {
		return false
	}
	if len(query.Sources.AllowedOriginIDs) != 0 && !contains(query.Sources.AllowedOriginIDs, packet.Source.OriginID) {
		return false
	}
	if len(query.Sources.AllowedAuthorityClasses) != 0 && !contains(query.Sources.AllowedAuthorityClasses, packet.Epistemic.AuthorityClass) {
		return false
	}
	if !contains(query.Trust.AllowedLanes, packet.Epistemic.Lane) || !contains(query.Trust.AllowedMappingStatuses, packet.Epistemic.MappingStatus) {
		return false
	}
	if !matchTime(query.Time, packet.Time.ObservedAt, packet.Lifecycle.State) {
		return false
	}
	if packet.Epistemic.FreshnessStatus == "stale" && query.Freshness.StaleBehavior == "exclude" {
		return false
	}
	if query.Freshness.MaximumAgeSeconds != nil {
		snapshot, _ := time.Parse(time.RFC3339, snapshotTime)
		observed, _ := time.Parse(time.RFC3339, packet.Time.ObservedAt)
		if snapshot.Sub(observed) > time.Duration(*query.Freshness.MaximumAgeSeconds)*time.Second {
			return false
		}
	}
	return true
}

func matchTime(query dataplane.QueryTime, observedAt, lifecycle string) bool {
	switch query.Mode {
	case "current":
		return lifecycle == "current" || (lifecycle == "stale")
	case "as_of":
		return observedAt <= query.Until.Value
	case "between":
		return observedAt >= query.From.Value && observedAt <= query.Until.Value
	case "history":
		return (!query.From.Present || observedAt >= query.From.Value) && (!query.Until.Present || observedAt <= query.Until.Value)
	}
	return false
}

func (r *Runtime) readArtifact(artifact dataplane.SnapshotArtifact) ([]byte, error) {
	path := filepath.Join(r.directory, filepath.FromSlash(artifact.Path))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || uint64(info.Size()) != artifact.Size {
		return nil, fmt.Errorf("%w: artifact %q changed after verification", ErrInvalidSnapshot, artifact.Path)
	}
	data, err := os.ReadFile(path)
	if err != nil || dataplane.DigestBytes(data) != artifact.Digest {
		return nil, fmt.Errorf("%w: artifact %q digest changed", ErrInvalidSnapshot, artifact.Path)
	}
	return data, nil
}

func (r *Runtime) findArtifact(path string) (dataplane.SnapshotArtifact, bool) {
	for _, artifact := range r.manifest.Artifacts {
		if artifact.Path == path {
			return artifact, true
		}
	}
	return dataplane.SnapshotArtifact{}, false
}

func findArtifactDigest(artifacts []dataplane.SnapshotArtifact, digest dataplane.Digest) (dataplane.SnapshotArtifact, bool) {
	for _, artifact := range artifacts {
		if artifact.Digest == digest {
			return artifact, true
		}
	}
	return dataplane.SnapshotArtifact{}, false
}

func strictSortedUnique(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for index, value := range values {
		if value == "" || strings.ContainsRune(value, '\x00') || (index > 0 && values[index-1] >= value) {
			return false
		}
	}
	return true
}

func contains(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func containsAny(values []string, candidates ...string) bool {
	for _, candidate := range candidates {
		if candidate != "" && contains(values, candidate) {
			return true
		}
	}
	return false
}

func sortRows(rows []dataplane.QueryResultRow) {
	sort.Slice(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		for _, values := range [][2]string{{left.SubjectID, right.SubjectID}, {left.PredicateID, right.PredicateID}, {left.OriginID, right.OriginID}, {left.ObservedAt, right.ObservedAt}} {
			if values[0] != values[1] {
				return values[0] < values[1]
			}
		}
		return bytes.Compare(left.PacketDigest[:], right.PacketDigest[:]) < 0
	})
}
