package snapshotbuild

import (
	"encoding/json"
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
	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/snapshotartifact"
)

type compiledArchivePacket struct {
	Digest        dataplane.Digest
	Packet        dataplane.Packet
	CaptureTime   string
	AcquisitionID dataplane.Digest
	SemanticKey   dataplane.Digest
}

type archiveSemanticKey struct {
	Format    string `json:"format"`
	OriginID  string `json:"origin_id"`
	Subject   string `json:"native_subject"`
	Predicate string `json:"native_predicate"`
	Scope     string `json:"scope"`
}

func (b *builder) compileArchiveAcquisition(id string, compilerContractDigest dataplane.Digest) error {
	loaded, err := archiveimport.LoadWorkOrder(filepath.Join(b.options.Root, "atlas", "archive-work-orders"), id)
	if err != nil {
		return err
	}
	root := filepath.Join(b.options.Root, "atlas", "archive-acquisitions", id)
	manifest, err := archiveacquire.Verify(root)
	if err != nil || manifest.WorkOrderDigest != loaded.Digest || manifest.WorkOrderID != loaded.Order.ID {
		return errors.New("archive acquisition does not reconcile with its current work order")
	}
	if err := verifyPolicyAuthority(b.options.Root, loaded.Order); err != nil {
		return err
	}
	adapterBytes, err := readBoundedRegular(filepath.Join(b.options.Root, "atlas", "archive-adapters", loaded.Order.OriginID+".json"), 16<<10)
	if err != nil {
		return err
	}
	profile, err := archiveprofile.ParseSpec(adapterBytes)
	if err != nil || profile.OriginID != loaded.Order.OriginID {
		return errors.New("archive native profile does not match the admitted origin")
	}
	planBytes, err := archiveprofile.PlanBytes(profile)
	if err != nil {
		return err
	}
	keyBytes, err := json.Marshal(archiveSemanticKey{Format: "tw.semantic-key/0.1", OriginID: loaded.Order.OriginID, Subject: profile.Subject, Predicate: profile.Predicate, Scope: "historical_archive"})
	if err != nil {
		return err
	}
	keyBytes = append(keyBytes, '\n')
	semanticKey := dataplane.DigestBytes(keyBytes)
	acquisitionManifestBytes, err := readBoundedRegular(filepath.Join(root, "acquisition-manifest.json"), archiveacquire.MaxManifest)
	if err != nil {
		return err
	}
	acquisitionID := dataplane.DigestBytes(acquisitionManifestBytes)
	compiled := make([]compiledArchivePacket, 0, len(manifest.Captures))
	for _, capture := range manifest.Captures {
		spoolRoot := filepath.Join(root, filepath.FromSlash(capture.SpoolPath))
		evidence, err := archiveimport.VerifySpool(spoolRoot)
		if err != nil || evidence.OriginID != loaded.Order.OriginID || evidence.CurrentPublisherStatement || evidence.EvidenceClass != "archive_observation" {
			return errors.New("archive capture proof failed reconciliation")
		}
		body, err := readBoundedRegular(filepath.Join(spoolRoot, "representation.body"), archiveprofile.MaxBody)
		if err != nil {
			return err
		}
		statement, err := archiveprofile.ExtractTitle(body)
		if err != nil {
			return err
		}
		observed, err := time.Parse("20060102150405", evidence.CaptureTimestamp)
		if err != nil || observed.UTC().Format(time.RFC3339) > b.options.CreatedAt {
			return errors.New("archive capture occurs after snapshot creation time")
		}
		representationDigest, err := snapshotartifact.ParseDigest(evidence.RepresentationDigest)
		if err != nil || representationDigest != dataplane.DigestBytes(body) {
			return errors.New("archive representation digest mismatch")
		}
		captureBytes, err := readBoundedRegular(filepath.Join(spoolRoot, "capture.json"), archiveimport.MaxMetadata)
		if err != nil {
			return err
		}
		packet := dataplane.Packet{
			Version:    dataplane.PacketVersion,
			Kind:       profile.Kind,
			Subject:    dataplane.PacketSubject{Native: profile.Subject, CanonicalCandidates: []string{}},
			Predicate:  dataplane.PacketPredicate{Native: profile.Predicate},
			Object:     dataplane.PacketObject{NativeStatus: "resolved", NativeLexical: statement.NativeLexical, MediaType: dataplane.OptionalText{Present: true, Value: profile.MediaType}},
			Context:    dataplane.PacketContext{Scope: dataplane.OptionalText{Present: true, Value: "historical_archive"}},
			Time:       dataplane.PacketTime{ObservedAt: observed.UTC().Format(time.RFC3339)},
			Source:     dataplane.PacketSource{OriginID: loaded.Order.OriginID, RepresentationDigest: representationDigest, Locator: statement.Locator, NativeSchemaRef: dataplane.OptionalText{Present: true, Value: "archive:html-title@0.1"}},
			Derivation: dataplane.PacketDerivation{ObservationDigest: dataplane.DigestBytes(captureBytes), AdapterDigest: dataplane.DigestBytes(adapterBytes), ExtractionPlanDigest: dataplane.DigestBytes(planBytes), TransformationIDs: []string{}, MappingIDs: []string{}, CompilerContractDigest: compilerContractDigest, CompilerVersion: CompilerVersion},
			Epistemic:  dataplane.PacketEpistemic{Lane: "observed_native", ExtractionStatus: "deterministic", MappingStatus: "none", AuthorityClass: "common_crawl_archive_observation", FreshnessStatus: "stale"},
			Lifecycle:  dataplane.PacketLifecycle{State: "stale"}, Retention: "public_archival", Disclosure: "public",
		}
		encoded, err := dataplane.MarshalPacket(packet)
		if err != nil {
			return err
		}
		packetDigest := dataplane.DigestBytes(encoded)
		if _, duplicate := b.packets[packetDigest]; duplicate {
			return errors.New("duplicate archive packet digest")
		}
		proof, err := b.archiveProof(id, capture.SpoolPath, packetDigest, packet.Derivation.ObservationDigest, profile, manifest, acquisitionManifestBytes, adapterBytes, planBytes, keyBytes)
		if err != nil {
			return err
		}
		b.packets[packetDigest] = encoded
		b.packetMeta[packetDigest] = packetMetadata{}
		b.proofs = append(b.proofs, proof)
		b.resolved++
		compiled = append(compiled, compiledArchivePacket{Digest: packetDigest, Packet: packet, CaptureTime: evidence.CaptureTimestamp, AcquisitionID: acquisitionID, SemanticKey: semanticKey})
	}
	if len(compiled) == 0 {
		return errors.New("archive acquisition produced no packets")
	}
	sort.Slice(compiled, func(i, j int) bool { return compiled[i].CaptureTime < compiled[j].CaptureTime })
	for index := 1; index < len(compiled); index++ {
		before, after := compiled[index-1], compiled[index]
		if before.Packet.Object.NativeLexical == after.Packet.Object.NativeLexical {
			continue
		}
		delta := dataplane.Delta{Version: dataplane.DeltaVersion, Class: "origin", Kind: "modified", SemanticKeyDigest: after.SemanticKey, BeforePacketDigest: dataplane.OptionalDigest{Present: true, Value: before.Digest}, AfterPacketDigest: dataplane.OptionalDigest{Present: true, Value: after.Digest}, BeforeSourceEvidenceDigest: dataplane.OptionalDigest{Present: true, Value: before.Packet.Source.RepresentationDigest}, AfterSourceEvidenceDigest: dataplane.OptionalDigest{Present: true, Value: after.Packet.Source.RepresentationDigest}, OriginID: loaded.Order.OriginID, OccurredAt: after.Packet.Time.ObservedAt, BatchID: after.AcquisitionID, CanonVersion: "tw:canon@0.1", ReasonCode: "source_native_title_changed"}
		encoded, err := dataplane.MarshalDelta(delta)
		if err != nil {
			return err
		}
		digest := dataplane.DigestBytes(encoded)
		if _, duplicate := b.deltas[digest]; duplicate {
			return errors.New("duplicate archive delta digest")
		}
		b.deltas[digest] = encoded
	}
	b.publicOrigins[loaded.Order.OriginID] = struct{}{}
	b.archiveProfiles++
	b.archiveOperations++
	return nil
}

func (b *builder) archiveProof(acquisitionID, spoolPath string, packetDigest, observationDigest dataplane.Digest, profile archiveprofile.Spec, manifest *archiveacquire.Manifest, acquisitionManifest, adapter, plan, key []byte) (snapshotartifact.ProofEntry, error) {
	base := filepath.ToSlash(filepath.Join("proof", "archive", acquisitionID))
	proof := snapshotartifact.ProofEntry{ProofType: "archive_capture", PacketDigest: snapshotartifact.DigestReference(packetDigest), EvidenceID: snapshotartifact.DigestReference(observationDigest), EvidenceClass: "archive_observation", ExecutionOriginID: profile.OriginID, OperationID: profile.OperationID, FieldID: profile.Predicate}
	common := []struct {
		name, path, media string
		data              []byte
	}{
		{"acquisition-manifest.json", base + "/acquisition-manifest.json", "application/json", acquisitionManifest},
		{"adapter.json", base + "/adapter.json", "application/json", adapter},
		{"extraction-plan.json", base + "/extraction-plan.json", "application/json", plan},
		{"semantic-key.json", base + "/semantic-key.json", "application/json", key},
	}
	for _, artifact := range common {
		entry, err := b.ensureProofArtifact(artifact.name, artifact.path, artifact.media, artifact.data)
		if err != nil {
			return proof, err
		}
		proof.Artifacts = append(proof.Artifacts, entry)
	}
	for _, source := range manifest.Artifacts {
		if strings.HasPrefix(source.Path, "captures/") {
			continue
		}
		data, err := readBoundedRegular(filepath.Join(b.options.Root, "atlas", "archive-acquisitions", acquisitionID, filepath.FromSlash(source.Path)), snapshotartifact.MaxArtifactBytes)
		if err != nil {
			return proof, err
		}
		name := "acquisition-" + strings.NewReplacer("/", "-", ".", "-").Replace(source.Path)
		entry, err := b.ensureProofArtifact(name, base+"/"+source.Path, proofMediaType(source.Path), data)
		if err != nil {
			return proof, err
		}
		proof.Artifacts = append(proof.Artifacts, entry)
	}
	spoolNames := []struct{ name, file, media string }{
		{"capture.json", "capture.json", "application/json"},
		{"evidence-manifest.json", "evidence-manifest.json", "application/json"},
		{"index-record.json", "index-record.json", "application/json"},
		{"range-response.json", "range-response.json", "application/json"},
		{"representation.body", "representation.body", "text/html"},
		{"spool-manifest.json", "manifest.json", "application/json"},
		{"warc-record.gz", "warc-record.gz", "application/gzip"},
		{"work-order.json", "work-order.json", "application/json"},
	}
	for _, item := range spoolNames {
		data, err := readBoundedRegular(filepath.Join(b.options.Root, "atlas", "archive-acquisitions", acquisitionID, filepath.FromSlash(spoolPath), item.file), snapshotartifact.MaxArtifactBytes)
		if err != nil {
			return proof, err
		}
		path := base + "/" + filepath.ToSlash(spoolPath) + "/" + item.file
		entry, err := b.ensureProofArtifact(item.name, path, item.media, data)
		if err != nil {
			return proof, err
		}
		proof.Artifacts = append(proof.Artifacts, entry)
	}
	sort.Slice(proof.Artifacts, func(i, j int) bool { return proof.Artifacts[i].Name < proof.Artifacts[j].Name })
	return proof, nil
}

func (b *builder) ensureProofArtifact(name, path, media string, data []byte) (snapshotartifact.ProofArtifact, error) {
	digest := dataplane.DigestBytes(data)
	for _, artifact := range b.artifacts {
		if artifact.Path == path {
			if artifact.Digest != digest || artifact.Size != uint64(len(data)) || artifact.MediaType != media || artifact.Role != "proof_index" {
				return snapshotartifact.ProofArtifact{}, errors.New("archive proof artifact path collision")
			}
			return snapshotartifact.ProofArtifact{Name: name, Path: path, Digest: snapshotartifact.DigestReference(digest), Size: uint64(len(data))}, nil
		}
	}
	if _, err := b.addArtifact(path, media, "proof_index", data); err != nil {
		return snapshotartifact.ProofArtifact{}, err
	}
	b.proofCount++
	return snapshotartifact.ProofArtifact{Name: name, Path: path, Digest: snapshotartifact.DigestReference(digest), Size: uint64(len(data))}, nil
}

func verifyPolicyAuthority(root string, order archiveimport.WorkOrder) error {
	for _, item := range []struct{ file, expected string }{{"policy-evidence.json", order.PolicyEvidenceDigest}, {"decision.json", order.DecisionDigest}} {
		data, err := readBoundedRegular(filepath.Join(root, "atlas", "admissions", order.OriginID, item.file), 1<<20)
		if err != nil || snapshotartifact.DigestReference(dataplane.DigestBytes(data)) != item.expected {
			return fmt.Errorf("archive policy authority changed for %s", item.file)
		}
	}
	return nil
}

func readBoundedRegular(path string, maximum int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > int64(maximum) {
		return nil, errors.New("snapshotbuild: input is not a bounded regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || len(data) > maximum {
		return nil, errors.New("snapshotbuild: input changed or exceeded its bound")
	}
	return data, nil
}

func strictSortedUnique(values []string) bool {
	for index, value := range values {
		if value == "" || strings.ContainsRune(value, '\x00') || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}
