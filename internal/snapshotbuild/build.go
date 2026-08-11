// Package snapshotbuild compiles admitted E2 replay evidence into an immutable
// Semantic Snapshot. It never performs network retrieval.
package snapshotbuild

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/e2format"
	"github.com/typed-web-commons/typed-web/internal/labengine"
	"github.com/typed-web-commons/typed-web/internal/observation"
	"github.com/typed-web-commons/typed-web/internal/proofbundle"
	"github.com/typed-web-commons/typed-web/internal/scalefixture"
	"github.com/typed-web-commons/typed-web/internal/snapshotartifact"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

const CompilerVersion = "twirx-go-semantic-snapshot/0.1"

const snapshotSegmentEntries = 4096

type Options struct {
	Root           string
	Output         string
	SourceRevision string
	CreatedAt      string
	// ScaleFixturePackets adds an explicitly controlled, non-public corpus for
	// local capacity testing. It never contributes to public packet counts.
	ScaleFixturePackets uint64
	// ArchiveAcquisitionIDs names immutable acquisitions under
	// atlas/archive-acquisitions. They are verified and compiled offline.
	ArchiveAcquisitionIDs []string
}

type Result struct {
	SnapshotID dataplane.Digest
	Manifest   dataplane.SnapshotManifest
	Report     snapshotartifact.BuildReport
}

type builder struct {
	options           Options
	stage             string
	artifacts         []dataplane.SnapshotArtifact
	packets           map[dataplane.Digest][]byte
	deltas            map[dataplane.Digest][]byte
	packetMeta        map[dataplane.Digest]packetMetadata
	proofs            []snapshotartifact.ProofEntry
	concepts          map[string]struct{}
	canonModules      map[string]struct{}
	mappings          map[string]snapshotartifact.Mapping
	publicOrigins     map[string]struct{}
	fixtureOrigins    map[string]struct{}
	proofCount        uint64
	fixturePackets    uint64
	resolved          uint64
	unresolved        uint64
	archiveProfiles   uint64
	archiveOperations uint64
}

type packetMetadata struct {
	Fixture         bool
	SemanticLexical string
}

type extractionPlan struct {
	NativeTerm    string   `json:"native_term"`
	NativeLocator string   `json:"native_locator"`
	SemanticTerm  string   `json:"semantic_term"`
	SemanticType  string   `json:"semantic_type"`
	Transforms    []string `json:"transforms"`
	Mapping       string   `json:"mapping"`
}

func Build(ctx context.Context, options Options) (Result, error) {
	var result Result
	if options.Root == "" || options.Output == "" || options.SourceRevision == "" || options.CreatedAt == "" {
		return result, errors.New("snapshotbuild: root, output, source revision, and creation time are required")
	}
	if options.ScaleFixturePackets > scalefixture.MaxFields {
		return result, errors.New("snapshotbuild: controlled scale fixture exceeds bound")
	}
	if len(options.ArchiveAcquisitionIDs) > 8 || !strictSortedUnique(options.ArchiveAcquisitionIDs) {
		return result, errors.New("snapshotbuild: archive acquisition IDs must be sorted, unique, and bounded")
	}
	if strings.ContainsRune(options.SourceRevision, '\x00') || len(options.SourceRevision) > dataplane.MaxIdentifier {
		return result, errors.New("snapshotbuild: invalid source revision")
	}
	createdAt, err := time.Parse("2006-01-02T15:04:05Z", options.CreatedAt)
	if err != nil || createdAt.UTC().Format("2006-01-02T15:04:05Z") != options.CreatedAt {
		return result, errors.New("snapshotbuild: creation time must be canonical UTC seconds")
	}
	output, err := filepath.Abs(options.Output)
	if err != nil {
		return result, err
	}
	root, err := filepath.Abs(options.Root)
	if err != nil {
		return result, err
	}
	if _, err := os.Lstat(output); err == nil {
		return result, errors.New("snapshotbuild: output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	parent := filepath.Dir(output)
	parentInfo, err := os.Stat(parent)
	if err != nil || !parentInfo.IsDir() {
		return result, errors.New("snapshotbuild: output parent must be an existing directory")
	}
	if err := os.Mkdir(output, 0o750); err != nil {
		return result, err
	}
	b := &builder{options: Options{Root: root, Output: output, SourceRevision: options.SourceRevision, CreatedAt: options.CreatedAt, ScaleFixturePackets: options.ScaleFixturePackets, ArchiveAcquisitionIDs: append([]string(nil), options.ArchiveAcquisitionIDs...)}, stage: output, packets: make(map[dataplane.Digest][]byte), deltas: make(map[dataplane.Digest][]byte), packetMeta: make(map[dataplane.Digest]packetMetadata), concepts: make(map[string]struct{}), canonModules: make(map[string]struct{}), mappings: make(map[string]snapshotartifact.Mapping), publicOrigins: make(map[string]struct{}), fixtureOrigins: make(map[string]struct{})}
	result, err = b.build(ctx)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (b *builder) build(ctx context.Context) (Result, error) {
	var result Result
	selectionPath := filepath.Join(b.options.Root, "atlas", "genesis-500", "selection.json")
	selection, err := atlas.LoadSelection(selectionPath)
	if err != nil {
		return result, err
	}
	selectionBytes, err := os.ReadFile(selectionPath)
	if err != nil {
		return result, err
	}
	if len(selectionBytes) == 0 || len(selectionBytes) > atlas.MaxSelectionBytes {
		return result, errors.New("snapshotbuild: selection changed outside its bounds")
	}
	if _, err := b.addArtifact("atlas/origin-selection.json", "application/json", "origin_catalog", selectionBytes); err != nil {
		return result, err
	}
	contractBytes, err := os.ReadFile(filepath.Join(b.options.Root, "schemas", "cddl", "semantic-data-plane.cddl"))
	if err != nil {
		return result, err
	}
	if len(contractBytes) == 0 || len(contractBytes) > dataplane.MaxDocumentBytes {
		return result, errors.New("snapshotbuild: compiler contract changed outside its bounds")
	}
	compilerContractDigest := dataplane.DigestBytes(contractBytes)

	resultsDir, err := os.MkdirTemp(filepath.Dir(b.stage), ".twirx-e2-replay-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(resultsDir)
	engine, err := labengine.New(b.options.Root, resultsDir)
	if err != nil {
		return result, err
	}
	operations := append([]twircontract.Operation(nil), engine.Contracts.Operations...)
	sort.Slice(operations, func(i, j int) bool { return operations[i].ID < operations[j].ID })
	for i := range operations {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		op := &operations[i]
		origin, findErr := engine.Catalog.Find(op.OriginID)
		if findErr != nil {
			return result, findErr
		}
		invocation, invokeErr := engine.Invoke(ctx, labengine.Request{OriginID: origin.ID, OperationID: op.ID, Input: cloneInput(origin.ReplayInput), Mode: labengine.ModeReplay})
		if invokeErr != nil {
			return result, fmt.Errorf("snapshotbuild: replay %s: %w", op.ID, invokeErr)
		}
		if err := b.compileInvocation(origin.RegistryID, origin.SourceClass == "controlled_fixture", op, invocation, compilerContractDigest); err != nil {
			return result, fmt.Errorf("snapshotbuild: compile %s: %w", op.ID, err)
		}
	}
	if b.options.ScaleFixturePackets > 0 {
		if err := b.compileScaleFixture(compilerContractDigest); err != nil {
			return result, fmt.Errorf("snapshotbuild: compile controlled scale fixture: %w", err)
		}
	}
	for _, acquisitionID := range b.options.ArchiveAcquisitionIDs {
		if err := b.compileArchiveAcquisition(acquisitionID, compilerContractDigest); err != nil {
			return result, fmt.Errorf("snapshotbuild: compile archive acquisition %s: %w", acquisitionID, err)
		}
	}
	operationCount := uint64(len(operations)) + b.archiveOperations
	if b.options.ScaleFixturePackets > 0 {
		operationCount++
	}

	if err := b.addPacketSegments(); err != nil {
		return result, err
	}
	if err := b.addDeltaSegments(); err != nil {
		return result, err
	}

	conceptList := sortedKeys(b.concepts)
	moduleList := sortedKeys(b.canonModules)
	conceptBytes, err := snapshotartifact.Marshal(snapshotartifact.Concepts{Format: snapshotartifact.ConceptsFormat, Concepts: conceptList, Modules: moduleList})
	if err != nil {
		return result, err
	}
	canonDigest, err := snapshotartifact.ModuleSetDigest(moduleList)
	if err != nil {
		return result, err
	}
	if _, err := b.addArtifact("canon/concepts.json", "application/json", "concepts", conceptBytes); err != nil {
		return result, err
	}
	mappingList := make([]snapshotartifact.Mapping, 0, len(b.mappings))
	for _, mapping := range b.mappings {
		mappingList = append(mappingList, mapping)
	}
	sort.Slice(mappingList, func(i, j int) bool { return mappingList[i].ID < mappingList[j].ID })
	mappingBytes, err := snapshotartifact.Marshal(snapshotartifact.Mappings{Format: snapshotartifact.MappingsFormat, Mappings: mappingList})
	if err != nil {
		return result, err
	}
	if _, err := b.addArtifact("canon/mappings.json", "application/json", "mappings", mappingBytes); err != nil {
		return result, err
	}

	if err := b.addProofIndexes(); err != nil {
		return result, err
	}

	views, err := b.addViews()
	if err != nil {
		return result, err
	}
	mode := "offline_e2_replay"
	limitations := []string{"archive-assisted ingestion is not implemented in this snapshot", "all public semantic packets derive from committed E2 offline replay evidence", "candidate Atlas identities without admitted evidence produce no packets", "the baseline snapshot contains no semantic deltas", "the runtime cannot refresh origins or execute actions"}
	if len(b.options.ArchiveAcquisitionIDs) > 0 {
		mode = "offline_e2_replay_with_archive_observations"
		limitations = []string{"archive observations are historical and are not current publisher statements", "archive-derived packets remain observed_native without semantic mapping", "only explicitly named and verified acquisitions enter this snapshot", "the runtime cannot refresh origins or execute actions"}
	}
	if b.options.ScaleFixturePackets > 0 {
		if len(b.options.ArchiveAcquisitionIDs) > 0 {
			mode = "offline_e2_replay_with_archive_observations_and_controlled_scale_fixture"
		} else {
			mode = "offline_e2_replay_with_controlled_scale_fixture"
		}
		limitations = append(limitations, "controlled scale-fixture packets are excluded from every public count, view and runtime default")
	}
	report := snapshotartifact.BuildReport{
		Format: snapshotartifact.BuildReportFormat, SourceRevision: b.options.SourceRevision, BuiltAt: b.options.CreatedAt,
		Mode: mode, NetworkRequests: 0, CurrentClaimsMade: false, FixtureCountedPublic: false,
		Actual:            snapshotartifact.ActualCounts{AtlasIdentities: uint64(len(selection.Candidates)), ArchiveProfiles: b.archiveProfiles, PublicOrigins: uint64(len(b.publicOrigins)), FixtureOrigins: uint64(len(b.fixtureOrigins)), Operations: operationCount, Packets: uint64(len(b.packets)), PublicPackets: uint64(len(b.packets)) - b.fixturePackets, FixturePackets: b.fixturePackets, ResolvedPackets: b.resolved, UnresolvedPackets: b.unresolved, Deltas: uint64(len(b.deltas)), MaterializedViews: uint64(len(views)), ProofArtifacts: b.proofCount},
		FundingDemoTarget: snapshotartifact.TargetCounts{AtlasIdentities: 500, ArchiveProfiles: 100, CurrentObserved: 25, Packets: 25000, MaterializedViews: 2},
		Limitations:       limitations,
	}
	reportBytes, err := snapshotartifact.Marshal(report)
	if err != nil {
		return result, err
	}
	reportDigest, err := b.addArtifact("reports/build.json", "application/json", "build_report", reportBytes)
	if err != nil {
		return result, err
	}

	sort.Slice(b.artifacts, func(i, j int) bool { return b.artifacts[i].Path < b.artifacts[j].Path })
	var total uint64
	for _, artifact := range b.artifacts {
		total += artifact.Size
	}
	selectionDigest, err := snapshotartifact.ParseDigest(selection.DigestReference())
	if err != nil {
		return result, err
	}
	evidenceClasses := []string{"recorded_offline_replay", "test_fixture"}
	if len(b.options.ArchiveAcquisitionIDs) > 0 {
		evidenceClasses = []string{"archive_observation", "recorded_offline_replay", "test_fixture"}
	}
	manifest := dataplane.SnapshotManifest{
		Version: dataplane.SnapshotVersion, Channel: "grant-demo", CreatedAt: b.options.CreatedAt, SourceRevision: b.options.SourceRevision,
		CompilerContractDigest: compilerContractDigest, CompilerVersion: CompilerVersion, AtlasSelectionDigest: selectionDigest, CanonModuleSetDigest: canonDigest,
		EvidenceClasses: evidenceClasses, Artifacts: b.artifacts, Views: views,
		Counts:                dataplane.SnapshotCounts{Origins: uint64(len(selection.Candidates)), Concepts: uint64(len(conceptList)), Mappings: uint64(len(mappingList)), Packets: uint64(len(b.packets)), Deltas: uint64(len(b.deltas)), Views: uint64(len(views)), ProofArtifacts: b.proofCount, EconomicEvents: 0},
		HighestPacketSequence: uint64(len(b.packets)), HighestDeltaSequence: uint64(len(b.deltas)), TotalArtifactBytes: total, BuildReportDigest: reportDigest,
	}
	manifestBytes, err := dataplane.MarshalSnapshotManifest(manifest)
	if err != nil {
		return result, err
	}
	if err := os.WriteFile(filepath.Join(b.stage, "manifest.cbor"), manifestBytes, 0o640); err != nil {
		return result, err
	}
	snapshotID := dataplane.DigestBytes(manifestBytes)
	verified, verifiedID, err := dataplane.VerifySnapshotDirectory(b.stage, snapshotID)
	if err != nil || verifiedID != snapshotID {
		return result, fmt.Errorf("snapshotbuild: final verification: %w", err)
	}
	return Result{SnapshotID: snapshotID, Manifest: verified, Report: report}, nil
}

func (b *builder) addPacketSegments() error {
	entries := snapshotartifact.SortedPacketEntries(b.packets)
	for start, segmentNumber := 0, 1; start < len(entries); start, segmentNumber = start+snapshotSegmentEntries, segmentNumber+1 {
		end := start + snapshotSegmentEntries
		if end > len(entries) {
			end = len(entries)
		}
		segment := snapshotartifact.PacketSegment{Format: snapshotartifact.PacketSegmentFormat, StartSequence: entries[start].Sequence, Entries: entries[start:end]}
		if err := segment.Validate(); err != nil {
			return err
		}
		data, err := snapshotartifact.Marshal(segment)
		if err != nil {
			return err
		}
		path := "packets/segment-000001.json"
		if len(entries) > snapshotSegmentEntries {
			path = fmt.Sprintf("packets/segment-%06d.json", segmentNumber)
		}
		if _, err := b.addArtifact(path, "application/json", "packet_batch", data); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) addDeltaSegments() error {
	entries := snapshotartifact.SortedDeltaEntries(b.deltas)
	for start, segmentNumber := 0, 1; start < len(entries); start, segmentNumber = start+snapshotSegmentEntries, segmentNumber+1 {
		end := start + snapshotSegmentEntries
		if end > len(entries) {
			end = len(entries)
		}
		segment := snapshotartifact.DeltaSegment{Format: snapshotartifact.DeltaSegmentFormat, StartSequence: entries[start].Sequence, Entries: entries[start:end]}
		if err := segment.Validate(); err != nil {
			return err
		}
		data, err := snapshotartifact.Marshal(segment)
		if err != nil {
			return err
		}
		path := "deltas/segment-000001.json"
		if len(entries) > snapshotSegmentEntries {
			path = fmt.Sprintf("deltas/segment-%06d.json", segmentNumber)
		}
		if _, err := b.addArtifact(path, "application/json", "delta_batch", data); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) addProofIndexes() error {
	sort.Slice(b.proofs, func(i, j int) bool { return b.proofs[i].PacketDigest < b.proofs[j].PacketDigest })
	for start, segmentNumber := 0, 1; start < len(b.proofs); start, segmentNumber = start+snapshotSegmentEntries, segmentNumber+1 {
		end := start + snapshotSegmentEntries
		if end > len(b.proofs) {
			end = len(b.proofs)
		}
		index := snapshotartifact.ProofIndex{Format: snapshotartifact.ProofIndexFormat, Entries: b.proofs[start:end]}
		if err := index.Validate(); err != nil {
			return err
		}
		data, err := snapshotartifact.Marshal(index)
		if err != nil {
			return err
		}
		path := "proof/index.json"
		if len(b.proofs) > snapshotSegmentEntries {
			path = fmt.Sprintf("proof/index-%06d.json", segmentNumber)
		}
		if _, err := b.addArtifact(path, "application/json", "proof_index", data); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) compileScaleFixture(compilerContractDigest dataplane.Digest) error {
	profile, err := scalefixture.NewProfile(b.options.ScaleFixturePackets, b.options.CreatedAt)
	if err != nil {
		return err
	}
	profileBytes, err := scalefixture.MarshalProfile(profile)
	if err != nil {
		return err
	}
	body, err := scalefixture.GenerateBody(profile)
	if err != nil {
		return err
	}
	fields, err := scalefixture.ParseBody(body, profile)
	if err != nil {
		return err
	}
	bodyDigest := sha256.Sum256(body)
	envelope := &observation.Envelope{Version: observation.FormatVersion, RequestURL: scalefixture.RequestURL, FinalURL: scalefixture.RequestURL, Method: "GET", Status: 200, MediaType: "application/json", RetrievedAt: profile.ObservedAt, BodySHA256: bodyDigest, BodySize: uint64(len(body)), PolicyID: scalefixture.PolicyID, ObserverID: scalefixture.ObserverID}
	observationBytes, err := envelope.MarshalCBOR()
	if err != nil {
		return err
	}
	observationDigest := dataplane.DigestBytes(observationBytes)
	profileDigest := dataplane.DigestBytes(profileBytes)
	proofArtifacts := make([]snapshotartifact.ProofArtifact, 0, 3)
	for _, artifact := range []struct {
		name, path, mediaType string
		data                  []byte
	}{
		{name: "observation.cbor", path: "proof/scale-fixture/observation.cbor", mediaType: "application/cbor", data: observationBytes},
		{name: "representation.body", path: "proof/scale-fixture/representation.body", mediaType: "application/json", data: body},
		{name: "scale-profile.json", path: "proof/scale-fixture/scale-profile.json", mediaType: "application/json", data: profileBytes},
	} {
		digest, err := b.addArtifact(artifact.path, artifact.mediaType, "proof_index", artifact.data)
		if err != nil {
			return err
		}
		b.proofCount++
		proofArtifacts = append(proofArtifacts, snapshotartifact.ProofArtifact{Name: artifact.name, Path: artifact.path, Digest: snapshotartifact.DigestReference(digest), Size: uint64(len(artifact.data))})
	}
	sort.Slice(proofArtifacts, func(i, j int) bool { return proofArtifacts[i].Name < proofArtifacts[j].Name })
	b.fixtureOrigins[scalefixture.OriginID] = struct{}{}
	for _, field := range fields {
		planBytes, err := scalefixture.PlanBytes(field[0])
		if err != nil {
			return err
		}
		packet := dataplane.Packet{
			Version:    dataplane.PacketVersion,
			Kind:       "state",
			Subject:    dataplane.PacketSubject{Native: scalefixture.SubjectID, CanonicalCandidates: []string{}},
			Predicate:  dataplane.PacketPredicate{Native: field[0]},
			Object:     dataplane.PacketObject{NativeStatus: "resolved", NativeLexical: field[1], MediaType: dataplane.OptionalText{Present: true, Value: "application/json"}},
			Context:    dataplane.PacketContext{Scope: dataplane.OptionalText{Present: true, Value: "controlled_scale_test"}},
			Time:       dataplane.PacketTime{ObservedAt: profile.ObservedAt},
			Source:     dataplane.PacketSource{OriginID: scalefixture.OriginID, RepresentationDigest: dataplane.DigestBytes(body), Locator: scalefixture.Locator(field[0]), NativeSchemaRef: dataplane.OptionalText{Present: true, Value: "fixture:scale-corpus@0.1"}},
			Derivation: dataplane.PacketDerivation{ObservationDigest: observationDigest, AdapterDigest: profileDigest, ExtractionPlanDigest: dataplane.DigestBytes(planBytes), TransformationIDs: []string{}, MappingIDs: []string{}, CompilerContractDigest: compilerContractDigest, CompilerVersion: CompilerVersion},
			Epistemic:  dataplane.PacketEpistemic{Lane: "observed_native", ExtractionStatus: "deterministic", MappingStatus: "none", AuthorityClass: "controlled_test_fixture", FreshnessStatus: "stale"},
			Lifecycle:  dataplane.PacketLifecycle{State: "stale"}, Retention: "public_transient", Disclosure: "public",
		}
		encoded, err := dataplane.MarshalPacket(packet)
		if err != nil {
			return err
		}
		digest := dataplane.DigestBytes(encoded)
		if _, exists := b.packets[digest]; exists {
			return errors.New("snapshotbuild: duplicate controlled scale packet")
		}
		b.packets[digest] = encoded
		b.packetMeta[digest] = packetMetadata{Fixture: true, SemanticLexical: ""}
		b.proofs = append(b.proofs, snapshotartifact.ProofEntry{ProofType: "controlled_scale_fixture", PacketDigest: snapshotartifact.DigestReference(digest), EvidenceID: snapshotartifact.DigestReference(observationDigest), EvidenceClass: "test_fixture", ExecutionOriginID: scalefixture.OriginID, OperationID: scalefixture.OperationID, FieldID: field[0], Artifacts: append([]snapshotartifact.ProofArtifact(nil), proofArtifacts...)})
		b.fixturePackets++
		b.resolved++
	}
	return nil
}

func (b *builder) compileInvocation(registryID string, fixture bool, op *twircontract.Operation, invocation *labengine.Invocation, compilerContractDigest dataplane.Digest) error {
	if invocation.Result.ObservedAt > b.options.CreatedAt {
		return errors.New("observation occurs after snapshot creation time")
	}
	proofManifestBytes, err := os.ReadFile(filepath.Join(invocation.Publication.Directory, proofbundle.ManifestName))
	if err != nil {
		return err
	}
	proofManifest, err := proofbundle.UnmarshalManifest(proofManifestBytes)
	if err != nil {
		return err
	}
	evidenceClass := "recorded_offline_replay"
	authorityClass := "project_recorded_origin_fixture"
	if fixture {
		evidenceClass = "test_fixture"
		authorityClass = "controlled_test_fixture"
		b.fixtureOrigins[registryID] = struct{}{}
	} else {
		b.publicOrigins[registryID] = struct{}{}
	}
	representationBytes, err := os.ReadFile(filepath.Join(invocation.Publication.Directory, "representation.body"))
	if err != nil {
		return err
	}
	for _, concept := range append(append([]string(nil), op.SemanticClosure...), op.Resource, op.SemanticReference) {
		b.concepts[concept] = struct{}{}
	}
	for _, module := range op.SemanticClosure {
		b.canonModules[module] = struct{}{}
	}
	for index, field := range invocation.Result.Fields {
		if index >= len(op.Output) || op.Output[index].ID != field.ID {
			return errors.New("result fields do not align with admitted operation")
		}
		spec := &op.Output[index]
		planBytes, err := json.Marshal(extractionPlan{NativeTerm: spec.NativeTerm, NativeLocator: spec.NativeLocator, SemanticTerm: spec.SemanticTerm, SemanticType: spec.Type, Transforms: spec.Transforms, Mapping: spec.Mapping})
		if err != nil {
			return err
		}
		packet := dataplane.Packet{
			Version: dataplane.PacketVersion, Kind: packetKind(spec.Type),
			Subject:    dataplane.PacketSubject{Native: op.NativeReference + "/operation/" + op.ID, CanonicalCandidates: []string{op.Resource}},
			Predicate:  dataplane.PacketPredicate{Native: field.NativeTerm, Semantic: dataplane.OptionalText{Present: true, Value: field.SemanticTerm}},
			Object:     dataplane.PacketObject{NativeStatus: field.Status, NativeLexical: field.NativeLexical, Typed: typedValue(field)},
			Time:       dataplane.PacketTime{ObservedAt: invocation.Result.ObservedAt},
			Source:     dataplane.PacketSource{OriginID: registryID, RepresentationDigest: dataplane.DigestBytes(representationBytes), Locator: field.NativeLocator, NativeSchemaRef: dataplane.OptionalText{Present: true, Value: op.NativeReference}},
			Derivation: dataplane.PacketDerivation{ObservationDigest: invocation.Result.ObservationDigest, TransportDigest: dataplane.OptionalDigest{Present: true, Value: invocation.Result.TransportDigest}, AdapterDigest: invocation.Result.AdapterDigest, ExtractionPlanDigest: dataplane.DigestBytes(planBytes), TransformationIDs: append([]string(nil), field.Transforms...), MappingIDs: []string{field.Mapping}, SemanticClosureDigest: dataplane.OptionalDigest{Present: true, Value: invocation.Result.SemanticClosureDigest}, CompilerContractDigest: compilerContractDigest, CompilerVersion: CompilerVersion},
			Epistemic:  dataplane.PacketEpistemic{Lane: "attested_semantic", ExtractionStatus: "deterministic", MappingStatus: "reviewed", AuthorityClass: authorityClass, FreshnessStatus: "stale"},
			Lifecycle:  dataplane.PacketLifecycle{State: "stale"}, Retention: "public_versioned", Disclosure: "public",
		}
		if field.Status != "resolved" {
			packet.Object.NativeLexical = ""
			packet.Object.Typed = nil
			b.unresolved++
		} else {
			b.resolved++
		}
		encoded, err := dataplane.MarshalPacket(packet)
		if err != nil {
			return err
		}
		digest := dataplane.DigestBytes(encoded)
		if _, exists := b.packets[digest]; exists {
			return errors.New("duplicate semantic packet digest")
		}
		b.packets[digest] = encoded
		b.packetMeta[digest] = packetMetadata{Fixture: fixture, SemanticLexical: field.SemanticLexical}
		b.concepts[field.SemanticTerm] = struct{}{}
		mappingID := field.Mapping + ":" + field.NativeTerm + ":" + field.SemanticTerm
		b.mappings[mappingID] = snapshotartifact.Mapping{ID: mappingID, NativeTerm: field.NativeTerm, SemanticTerm: field.SemanticTerm, Status: "reviewed", EvidenceClass: evidenceClass}

		proof := snapshotartifact.ProofEntry{ProofType: "e2_bundle", PacketDigest: snapshotartifact.DigestReference(digest), EvidenceID: invocation.Publication.BundleID, EvidenceClass: evidenceClass, ExecutionOriginID: invocation.Result.OriginID, OperationID: invocation.Result.OperationID, FieldID: field.ID}
		for _, artifact := range append(proofManifest.Entries, proofbundle.Entry{Name: proofbundle.ManifestName, Digest: sha256.Sum256(proofManifestBytes), Size: uint64(len(proofManifestBytes))}) {
			data := proofManifestBytes
			if artifact.Name != proofbundle.ManifestName {
				data, err = os.ReadFile(filepath.Join(invocation.Publication.Directory, artifact.Name))
				if err != nil {
					return err
				}
			}
			resultHex := strings.TrimPrefix(invocation.Publication.ResultDigest, "sha256:")
			path := filepath.ToSlash(filepath.Join("proof", resultHex, artifact.Name))
			if !b.hasArtifact(path) {
				if _, err := b.addArtifact(path, proofMediaType(artifact.Name), "proof_index", data); err != nil {
					return err
				}
				b.proofCount++
			}
			proof.Artifacts = append(proof.Artifacts, snapshotartifact.ProofArtifact{Name: artifact.Name, Path: path, Digest: "sha256:" + hex.EncodeToString(artifact.Digest[:]), Size: artifact.Size})
		}
		sort.Slice(proof.Artifacts, func(i, j int) bool { return proof.Artifacts[i].Name < proof.Artifacts[j].Name })
		b.proofs = append(b.proofs, proof)
		if fixture {
			b.fixturePackets++
		}
	}
	return nil
}

func (b *builder) addViews() ([]dataplane.SnapshotView, error) {
	publicEvidenceClasses := []string{"recorded_offline_replay"}
	if b.archiveProfiles > 0 {
		publicEvidenceClasses = []string{"archive_observation", "recorded_offline_replay"}
	}
	allPublic := snapshotartifact.View{Format: snapshotartifact.ViewFormat, ID: "demo.public-source-statements", Definition: "all admitted non-fixture source statements; historical and not current publisher claims", ThroughSequence: uint64(len(b.packets)), PublicOnly: true, EvidenceClasses: publicEvidenceClasses}
	measurements := snapshotartifact.View{Format: snapshotartifact.ViewFormat, ID: "demo.public-measurements", Definition: "resolved measurement packets from admitted non-fixture E2 replay evidence", ThroughSequence: uint64(len(b.packets)), PublicOnly: true, EvidenceClasses: []string{"recorded_offline_replay"}}
	for digest, encoded := range b.packets {
		meta := b.packetMeta[digest]
		if meta.Fixture {
			continue
		}
		packet, err := dataplane.UnmarshalPacket(encoded)
		if err != nil {
			return nil, err
		}
		row := viewRow(digest, packet, meta.SemanticLexical)
		allPublic.Rows = append(allPublic.Rows, row)
		if packet.Kind == "measurement" && packet.Object.NativeStatus == "resolved" {
			measurements.Rows = append(measurements.Rows, row)
		}
	}
	views := []snapshotartifact.View{allPublic, measurements}
	result := make([]dataplane.SnapshotView, 0, len(views))
	for _, view := range views {
		sort.Slice(view.Rows, func(i, j int) bool { return view.Rows[i].PacketDigest < view.Rows[j].PacketDigest })
		data, err := snapshotartifact.Marshal(view)
		if err != nil {
			return nil, err
		}
		path := "views/" + view.ID + ".json"
		digest, err := b.addArtifact(path, "application/json", "materialized_view", data)
		if err != nil {
			return nil, err
		}
		definition := dataplane.DigestBytes([]byte(view.Definition))
		result = append(result, dataplane.SnapshotView{ID: view.ID, DefinitionDigest: definition, ArtifactDigest: digest, RowCount: uint64(len(view.Rows)), ThroughSequence: view.ThroughSequence})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func viewRow(digest dataplane.Digest, packet dataplane.Packet, semanticLexical string) snapshotartifact.ViewRow {
	return snapshotartifact.ViewRow{PacketDigest: snapshotartifact.DigestReference(digest), OriginID: packet.Source.OriginID, SubjectID: packet.Subject.Native, NativeTerm: packet.Predicate.Native, NativeLexical: packet.Object.NativeLexical, NativeStatus: packet.Object.NativeStatus, SemanticTerm: packet.Predicate.Semantic.Value, SemanticLexical: semanticLexical, Lane: packet.Epistemic.Lane, ObservedAt: packet.Time.ObservedAt}
}

func (b *builder) addArtifact(name, mediaType, role string, data []byte) (dataplane.Digest, error) {
	var zero dataplane.Digest
	if err := dataplane.ValidateSnapshotPath(name); err != nil {
		return zero, err
	}
	if b.hasArtifact(name) {
		return zero, errors.New("snapshotbuild: duplicate artifact path")
	}
	if len(data) == 0 || uint64(len(data)) > dataplane.MaxSnapshotArtifactBytes {
		return zero, errors.New("snapshotbuild: artifact size outside bounds")
	}
	path := filepath.Join(b.stage, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return zero, err
	}
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return zero, err
	}
	digest := dataplane.DigestBytes(data)
	b.artifacts = append(b.artifacts, dataplane.SnapshotArtifact{Path: name, Digest: digest, Size: uint64(len(data)), MediaType: mediaType, Role: role})
	return digest, nil
}

func (b *builder) hasArtifact(path string) bool {
	for _, artifact := range b.artifacts {
		if artifact.Path == path {
			return true
		}
	}
	return false
}

func typedValue(field e2format.Field) *dataplane.TypedValue {
	if field.Status != "resolved" || !field.SemanticPresent || field.SemanticLexical == "" {
		return nil
	}
	switch field.SemanticType {
	case "string", "currency_code":
		return &dataplane.TypedValue{Type: "text", Lexical: field.SemanticLexical}
	case "integer":
		return &dataplane.TypedValue{Type: "integer", Lexical: field.SemanticLexical}
	case "decimal":
		// The Semantic Packet decimal profile requires an explicit fractional
		// component. Do not silently rewrite an integer-shaped provider lexical
		// value merely because the E2 operation declared a broader decimal type.
		if strings.Contains(field.SemanticLexical, ".") {
			return &dataplane.TypedValue{Type: "decimal", Lexical: field.SemanticLexical}
		}
	}
	return nil
}

func packetKind(valueType string) string {
	if valueType == "integer" || valueType == "decimal" {
		return "measurement"
	}
	return "state"
}

func cloneInput(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func proofMediaType(name string) string {
	if strings.HasSuffix(name, ".cbor") {
		return "application/cbor"
	}
	if strings.HasSuffix(name, ".json") {
		return "application/json"
	}
	return "application/octet-stream"
}

func compareDigest(a, b dataplane.Digest) int { return bytes.Compare(a[:], b[:]) }
