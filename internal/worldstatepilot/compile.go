package worldstatepilot

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/cas"
	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/egressworker"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/universeimport"
	"github.com/typed-web-commons/typed-web/internal/universesnapshot"
)

const (
	ReleaseFormat = "tw.e4-world-state-release/0.1"
	maxSummary    = 1 << 20
	maxRelease    = 4 << 20
)

type ReleaseArtifact struct {
	Kind      string `json:"kind"`
	NativeKey string `json:"native_key,omitempty"`
	Path      string `json:"path"`
	Digest    string `json:"digest"`
	Size      uint64 `json:"size"`
}

type Rejection struct {
	OrderID              string `json:"order_id"`
	Code                 string `json:"code"`
	HTTPStatus           int    `json:"http_status"`
	MediaType            string `json:"media_type"`
	ObservationDigest    string `json:"observation_digest"`
	RepresentationDigest string `json:"representation_digest"`
}

type ReleaseManifest struct {
	Format               string            `json:"format"`
	PlanID               string            `json:"plan_id"`
	OriginID             string            `json:"origin_id"`
	EvidenceClass        string            `json:"evidence_class"`
	CompiledAt           string            `json:"compiled_at"`
	PolicyDecisionDigest string            `json:"policy_decision_digest"`
	ModuleSetDigest      string            `json:"module_set_digest"`
	SourceResponses      int               `json:"source_responses"`
	EligibleResponses    int               `json:"eligible_responses"`
	RejectedResponses    int               `json:"rejected_responses"`
	SourceRecords        int               `json:"source_records"`
	Packets              int               `json:"packets"`
	MappingClaims        int               `json:"mapping_claims"`
	Frames               int               `json:"frames"`
	TrustLane            string            `json:"trust_lane"`
	MappingStatus        string            `json:"mapping_status"`
	SchedulerEnabled     bool              `json:"scheduler_enabled"`
	Artifacts            []ReleaseArtifact `json:"artifacts"`
	Rejections           []Rejection       `json:"rejections"`
}

type ProofEntry struct {
	NativeKey            string   `json:"native_key"`
	FrameDigest          string   `json:"frame_digest"`
	PacketDigests        []string `json:"packet_digests"`
	ObservationDigest    string   `json:"observation_digest"`
	RepresentationDigest string   `json:"representation_digest"`
	PolicyDecisionDigest string   `json:"policy_decision_digest"`
	EvidenceRoot         string   `json:"evidence_root"`
}

type ProofIndex struct {
	Format  string       `json:"format"`
	Entries []ProofEntry `json:"entries"`
}

func Compile(root, preparedRoot, spoolRoot, output, compiledAt string) (ReleaseManifest, error) {
	var release ReleaseManifest
	parsedTime, err := time.Parse(time.RFC3339, compiledAt)
	if err != nil || parsedTime.Location() != time.UTC || parsedTime.Format(time.RFC3339) != compiledAt {
		return release, errors.New("worldstatepilot: compiled_at must be canonical UTC seconds")
	}
	if _, err := os.Lstat(output); err == nil {
		return release, errors.New("worldstatepilot: release output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return release, err
	}
	summary, err := loadExecutionSummary(filepath.Join(preparedRoot, "acquisition-summary.json"))
	if err != nil {
		return release, err
	}
	prepared, err := loadPrepared(filepath.Join(preparedRoot, "prepared-manifest.json"))
	if err != nil {
		return release, err
	}
	plan, _, err := LoadPlan(filepath.Join(root, "atlas", "e4-plans", "world-bank-e2-matrix.json"))
	if err != nil || verifyPrepared(root, plan, preparedRoot, prepared) != nil {
		return release, errors.New("worldstatepilot: prepared acquisition authority no longer validates")
	}
	if summary.PlanID != prepared.PlanID || summary.OriginID != OriginID || summary.SchedulerEnabled || len(summary.Entries) != len(prepared.Orders) || summary.EligibleResponses+summary.RejectedResponses != len(summary.Entries) {
		return release, errors.New("worldstatepilot: acquisition summary disagrees with prepared plan")
	}
	moduleSetDigest, err := worldStateModuleSetDigest(root)
	if err != nil {
		return release, err
	}
	decisionDigest, err := parseDigest(prepared.DecisionDigest)
	if err != nil {
		return release, err
	}
	if err := os.Mkdir(output, 0o750); err != nil {
		return release, err
	}
	store := cas.New(filepath.Join(output, "cas"))
	var records []universeimport.RecordArtifact
	var proofs []ProofEntry
	for index, entry := range summary.Entries {
		expected := prepared.Orders[index]
		if expected.ID != entry.OrderID {
			return ReleaseManifest{}, errors.New("worldstatepilot: execution order changed")
		}
		loaded, err := egressworker.LoadWorkOrder(filepath.Join(preparedRoot, "work-orders"), entry.OrderID)
		if err != nil {
			return ReleaseManifest{}, err
		}
		spool := filepath.Join(spoolRoot, entry.OrderID, strings.TrimPrefix(loaded.Digest, "sha256:"))
		verified, err := egressworker.VerifySpool(spool, loaded.Order.MaxBodyBytes)
		if err != nil || verified.ObservationDigest != entry.ObservationDigest || verified.BodyDigest != entry.RepresentationDigest || verified.HTTPStatus != entry.HTTPStatus || verified.MediaType != entry.MediaType {
			return ReleaseManifest{}, fmt.Errorf("worldstatepilot: spool %s no longer matches execution summary", entry.OrderID)
		}
		if !entry.CompilationEligible {
			release.Rejections = append(release.Rejections, Rejection{OrderID: entry.OrderID, Code: entry.RejectionCode, HTTPStatus: entry.HTTPStatus, MediaType: entry.MediaType, ObservationDigest: entry.ObservationDigest, RepresentationDigest: entry.RepresentationDigest})
			continue
		}
		body, err := readRepresentation(spool, entry.RepresentationDigest, loaded.Order.MaxBodyBytes)
		if err != nil {
			return ReleaseManifest{}, err
		}
		observationDigest, err := parseDigest(entry.ObservationDigest)
		if err != nil {
			return ReleaseManifest{}, err
		}
		representationDigest, err := parseDigest(entry.RepresentationDigest)
		if err != nil {
			return ReleaseManifest{}, err
		}
		compiled, err := universeimport.CompileWorldBank(body, universeimport.Config{
			OriginID: OriginID, ObservedAt: canonicalSecond(entry.RetrievedAt), RepresentationDigest: representationDigest, ObservationDigest: observationDigest,
			ModuleSetDigest: moduleSetDigest, PolicyDecisionDigest: dataplane.OptionalDigest{Present: true, Value: decisionDigest}, EvidenceClass: "current_observation",
			EvidenceRef: filepath.ToSlash(filepath.Join("atlas", "e4-acquisitions", "world-bank-e2-matrix", "spool", entry.OrderID)), EvidenceStored: true,
		})
		if err != nil {
			return ReleaseManifest{}, fmt.Errorf("worldstatepilot: compile %s: %w", entry.OrderID, err)
		}
		if len(compiled) != 1 {
			return ReleaseManifest{}, fmt.Errorf("worldstatepilot: %s produced %d source records", entry.OrderID, len(compiled))
		}
		records = append(records, compiled...)
		packetDigests := make([]string, len(compiled[0].Packets))
		for packetIndex, packet := range compiled[0].Packets {
			packetDigests[packetIndex] = digestText(packet.Digest)
		}
		sort.Strings(packetDigests)
		proofs = append(proofs, ProofEntry{NativeKey: compiled[0].NativeKey, FrameDigest: digestText(compiled[0].FrameDigest), PacketDigests: packetDigests, ObservationDigest: entry.ObservationDigest, RepresentationDigest: entry.RepresentationDigest, PolicyDecisionDigest: prepared.DecisionDigest, EvidenceRoot: filepath.ToSlash(filepath.Join("atlas", "e4-acquisitions", "world-bank-e2-matrix", "spool", entry.OrderID, strings.TrimPrefix(loaded.Digest, "sha256:")))})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].NativeKey < records[j].NativeKey })
	sort.Slice(proofs, func(i, j int) bool { return proofs[i].NativeKey < proofs[j].NativeKey })
	seenNative := make(map[string]struct{}, len(records))
	var frames []universesnapshot.SourceFrame
	for _, record := range records {
		if _, exists := seenNative[record.NativeKey]; exists {
			return ReleaseManifest{}, fmt.Errorf("worldstatepilot: duplicate compiled native key %s", record.NativeKey)
		}
		seenNative[record.NativeKey] = struct{}{}
		for _, packet := range record.Packets {
			path, err := putArtifact(store, "packet", record.NativeKey, packet.CBOR, packet.Digest)
			if err != nil {
				return ReleaseManifest{}, err
			}
			release.Artifacts = append(release.Artifacts, path)
			release.Packets++
		}
		for _, mapping := range record.Mappings {
			path, err := putArtifact(store, "mapping_claim", record.NativeKey, mapping.CBOR, mapping.Digest)
			if err != nil {
				return ReleaseManifest{}, err
			}
			release.Artifacts = append(release.Artifacts, path)
			release.MappingClaims++
		}
		path, err := putArtifact(store, "frame", record.NativeKey, record.FrameCBOR, record.FrameDigest)
		if err != nil {
			return ReleaseManifest{}, err
		}
		release.Artifacts = append(release.Artifacts, path)
		frames = append(frames, universesnapshot.SourceFrame{Digest: record.FrameDigest, CBOR: record.FrameCBOR, Frame: record.Frame})
		release.Frames++
	}
	segment, segmentDigest, err := universesnapshot.BuildCompact(frames)
	if err != nil {
		return ReleaseManifest{}, err
	}
	segmentPath := filepath.Join(output, "segments", "world-state.twux")
	if err := atomicfile.Write(segmentPath, segment, universesnapshot.MaxBytes, 0o440); err != nil {
		return ReleaseManifest{}, err
	}
	release.Artifacts = append(release.Artifacts, ReleaseArtifact{Kind: "compact_frame_segment", Path: "segments/world-state.twux", Digest: digestText(segmentDigest), Size: uint64(len(segment))})
	proofBytes, err := json.MarshalIndent(ProofIndex{Format: "tw.e4-world-state-proof-index/0.1", Entries: proofs}, "", "  ")
	if err != nil {
		return ReleaseManifest{}, err
	}
	proofBytes = append(proofBytes, '\n')
	if len(proofBytes) == 0 || len(proofBytes) > maxRelease {
		return ReleaseManifest{}, errors.New("worldstatepilot: proof index exceeds bounds")
	}
	proofPath := filepath.Join(output, "proof", "world-state.json")
	if err := atomicfile.Write(proofPath, proofBytes, maxRelease, 0o440); err != nil {
		return ReleaseManifest{}, err
	}
	release.Artifacts = append(release.Artifacts, ReleaseArtifact{Kind: "proof_index", Path: "proof/world-state.json", Digest: cas.Digest(proofBytes), Size: uint64(len(proofBytes))})
	release.Format = ReleaseFormat
	release.PlanID = prepared.PlanID
	release.OriginID = OriginID
	release.EvidenceClass = "current_observation"
	release.CompiledAt = compiledAt
	release.PolicyDecisionDigest = prepared.DecisionDigest
	release.ModuleSetDigest = digestText(moduleSetDigest)
	release.SourceResponses = len(summary.Entries)
	release.EligibleResponses = summary.EligibleResponses
	release.RejectedResponses = summary.RejectedResponses
	release.SourceRecords = len(records)
	release.TrustLane = "provisional_semantic"
	release.MappingStatus = "candidate"
	release.SchedulerEnabled = false
	sort.Slice(release.Artifacts, func(i, j int) bool {
		if release.Artifacts[i].Digest != release.Artifacts[j].Digest {
			return release.Artifacts[i].Digest < release.Artifacts[j].Digest
		}
		if release.Artifacts[i].Kind != release.Artifacts[j].Kind {
			return release.Artifacts[i].Kind < release.Artifacts[j].Kind
		}
		return release.Artifacts[i].NativeKey < release.Artifacts[j].NativeKey
	})
	encoded, err := marshal(release)
	if err != nil || len(encoded) > maxRelease {
		return ReleaseManifest{}, errors.New("worldstatepilot: release manifest exceeds bounds")
	}
	// Manifest-last publication makes the compiled release visible only after
	// all canonical objects and the compact derived segment exist.
	if err := atomicfile.Write(filepath.Join(output, "release-manifest.json"), encoded, maxRelease, 0o440); err != nil {
		return ReleaseManifest{}, err
	}
	return release, nil
}

// VerifyRelease rehashes every constituent, reopens the compact segment, and
// reconciles every proof entry with both canonical artifacts and its original
// independently verified evidence spool.
func VerifyRelease(root, releaseRoot string) (ReleaseManifest, error) {
	var release ReleaseManifest
	manifestBytes, err := readRegular(filepath.Join(releaseRoot, "release-manifest.json"), maxRelease)
	if err != nil {
		return release, fmt.Errorf("worldstatepilot: release manifest unavailable: %w", err)
	}
	policy := jsonbounded.Policy{MaxBytes: maxRelease, MaxDepth: 20, MaxScalarBytes: 64 << 10, MaxContainerEntries: 200000, MaxTokens: 1000000}
	if err := jsonbounded.Decode(manifestBytes, &release, policy, true); err != nil {
		return release, err
	}
	if release.Format != ReleaseFormat || release.OriginID != OriginID || release.EvidenceClass != "current_observation" || release.SchedulerEnabled || release.SourceResponses != release.EligibleResponses+release.RejectedResponses || release.SourceRecords != release.EligibleResponses || release.Frames != release.SourceRecords || release.Packets != release.SourceRecords*5 || release.MappingClaims != release.SourceRecords*5 || release.TrustLane != "provisional_semantic" || release.MappingStatus != "candidate" || len(release.Artifacts) == 0 {
		return ReleaseManifest{}, errors.New("worldstatepilot: invalid release counts or authority state")
	}
	known := make(map[string]string, len(release.Artifacts))
	var segment ReleaseArtifact
	var proofArtifact ReleaseArtifact
	prior := ReleaseArtifact{}
	for index, artifact := range release.Artifacts {
		if artifact.Size == 0 || !safeReleasePath(artifact.Path) || artifact.Digest == "" || index > 0 && !releaseArtifactLess(prior, artifact) {
			return ReleaseManifest{}, errors.New("worldstatepilot: release artifacts are not sorted, unique, and bounded")
		}
		if _, duplicate := known[artifact.Path]; duplicate {
			return ReleaseManifest{}, errors.New("worldstatepilot: duplicate release artifact path")
		}
		known[artifact.Path] = artifact.Kind
		path := filepath.Join(releaseRoot, filepath.FromSlash(artifact.Path))
		if artifact.Kind == "compact_frame_segment" {
			segment = artifact
			prior = artifact
			continue
		}
		data, err := readRegular(path, dataplane.MaxDocumentBytes)
		if artifact.Kind == "proof_index" {
			data, err = readRegular(path, maxRelease)
			proofArtifact = artifact
		}
		if err != nil || uint64(len(data)) != artifact.Size || cas.Digest(data) != artifact.Digest {
			return ReleaseManifest{}, fmt.Errorf("worldstatepilot: release artifact %s does not verify", artifact.Path)
		}
		prior = artifact
	}
	if segment.Path == "" || proofArtifact.Path == "" {
		return ReleaseManifest{}, errors.New("worldstatepilot: release omits segment or proof index")
	}
	segmentDigest, err := parseDigest(segment.Digest)
	if err != nil {
		return ReleaseManifest{}, err
	}
	runtime, err := universesnapshot.OpenCompactFile(filepath.Join(releaseRoot, filepath.FromSlash(segment.Path)), segmentDigest)
	if err != nil {
		return ReleaseManifest{}, err
	}
	defer runtime.Close()
	if runtime.FrameCount() != uint64(release.Frames) {
		return ReleaseManifest{}, errors.New("worldstatepilot: compact frame count disagrees with manifest")
	}
	proofBytes, err := readRegular(filepath.Join(releaseRoot, filepath.FromSlash(proofArtifact.Path)), maxRelease)
	if err != nil {
		return ReleaseManifest{}, err
	}
	var proofs ProofIndex
	if err := jsonbounded.Decode(proofBytes, &proofs, policy, true); err != nil || proofs.Format != "tw.e4-world-state-proof-index/0.1" || len(proofs.Entries) != release.Frames {
		return ReleaseManifest{}, errors.New("worldstatepilot: invalid proof index")
	}
	priorKey := ""
	for _, proof := range proofs.Entries {
		if proof.NativeKey <= priorKey || proof.PolicyDecisionDigest != release.PolicyDecisionDigest || len(proof.PacketDigests) != 5 || !sort.StringsAreSorted(proof.PacketDigests) || !safeEvidenceRoot(proof.EvidenceRoot) {
			return ReleaseManifest{}, errors.New("worldstatepilot: invalid proof entry")
		}
		frameDigest, err := parseDigest(proof.FrameDigest)
		if err != nil || known[casPath(proof.FrameDigest)] != "frame" {
			return ReleaseManifest{}, errors.New("worldstatepilot: proof frame is absent")
		}
		frameBytes, err := runtime.Trace(frameDigest)
		if err != nil || dataplane.DigestBytes(frameBytes) != frameDigest {
			return ReleaseManifest{}, errors.New("worldstatepilot: compact trace and proof disagree")
		}
		for _, packet := range proof.PacketDigests {
			if known[casPath(packet)] != "packet" {
				return ReleaseManifest{}, errors.New("worldstatepilot: proof packet is absent")
			}
		}
		spool := filepath.Join(root, filepath.FromSlash(proof.EvidenceRoot))
		verified, err := egressworker.VerifySpool(spool, egressworker.MaxBody)
		if err != nil || verified.ObservationDigest != proof.ObservationDigest || verified.BodyDigest != proof.RepresentationDigest || verified.HTTPStatus != 200 || verified.MediaType != "application/json" {
			return ReleaseManifest{}, errors.New("worldstatepilot: proof source spool does not verify")
		}
		priorKey = proof.NativeKey
	}
	return release, nil
}

func loadPrepared(path string) (Prepared, error) {
	var value Prepared
	data, err := readRegular(path, maxSummary)
	if err != nil {
		return value, err
	}
	policy := jsonbounded.Policy{MaxBytes: maxSummary, MaxDepth: 16, MaxScalarBytes: 32 << 10, MaxContainerEntries: 4096, MaxTokens: 30000}
	return value, jsonbounded.Decode(data, &value, policy, true)
}

func loadExecutionSummary(path string) (ExecutionSummary, error) {
	var value ExecutionSummary
	data, err := readRegular(path, maxSummary)
	if err != nil {
		return value, err
	}
	policy := jsonbounded.Policy{MaxBytes: maxSummary, MaxDepth: 16, MaxScalarBytes: 32 << 10, MaxContainerEntries: 4096, MaxTokens: 30000}
	if err := jsonbounded.Decode(data, &value, policy, true); err != nil {
		return value, err
	}
	if value.Format != "tw.e4-world-state-execution/0.1" || value.SchedulerEnabled || len(value.Entries) == 0 || len(value.Entries) > MaxPilotOrders {
		return value, errors.New("worldstatepilot: invalid execution summary")
	}
	return value, nil
}

func worldStateModuleSetDigest(root string) (dataplane.Digest, error) {
	data, err := readRegular(filepath.Join(root, "generated", "e4", "ontology", "index.json"), 1<<20)
	if err != nil {
		return dataplane.Digest{}, err
	}
	var index struct {
		Format string `json:"format"`
		Items  []struct {
			Kind     string `json:"kind"`
			ID       string `json:"id"`
			Version  string `json:"version"`
			Digest   string `json:"digest"`
			CBORPath string `json:"cbor_path"`
			Source   string `json:"source"`
		} `json:"items"`
		Status string `json:"status"`
	}
	policy := jsonbounded.Policy{MaxBytes: 1 << 20, MaxDepth: 8, MaxScalarBytes: 4096, MaxContainerEntries: 256, MaxTokens: 4096}
	if err := jsonbounded.Decode(data, &index, policy, true); err != nil || index.Format != "tw.ontology-compile-index/0.1" {
		return dataplane.Digest{}, errors.New("worldstatepilot: invalid ontology compile index")
	}
	var digests []dataplane.Digest
	for _, item := range index.Items {
		if item.Kind == "ontology_module" && (item.ID == "tw:kernel" || item.ID == "tw:world-state") && item.Version == "0.1.0" {
			digest, err := parseDigest(item.Digest)
			if err != nil {
				return dataplane.Digest{}, err
			}
			digests = append(digests, digest)
		}
	}
	if len(digests) != 2 {
		return dataplane.Digest{}, errors.New("worldstatepilot: exact world-state module closure is absent")
	}
	sort.Slice(digests, func(i, j int) bool { return bytes.Compare(digests[i][:], digests[j][:]) < 0 })
	joined := append(append([]byte("tw.module-set/0.1\x00"), digests[0][:]...), digests[1][:]...)
	return dataplane.DigestBytes(joined), nil
}

func readRepresentation(spool, digest string, maximum int64) ([]byte, error) {
	store := cas.New(filepath.Join(spool, "cas"))
	return store.Read(digest, maximum)
}

func putArtifact(store *cas.Store, kind, nativeKey string, data []byte, expected dataplane.Digest) (ReleaseArtifact, error) {
	if dataplane.DigestBytes(data) != expected {
		return ReleaseArtifact{}, errors.New("worldstatepilot: canonical artifact digest mismatch")
	}
	digest, path, err := store.Put(data)
	if err != nil {
		return ReleaseArtifact{}, err
	}
	relative, err := filepath.Rel(filepath.Dir(store.Root), path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return ReleaseArtifact{}, errors.New("worldstatepilot: CAS artifact escaped release")
	}
	return ReleaseArtifact{Kind: kind, NativeKey: nativeKey, Path: filepath.ToSlash(relative), Digest: digest, Size: uint64(len(data))}, nil
}

func parseDigest(value string) (dataplane.Digest, error) {
	var digest dataplane.Digest
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return digest, errors.New("worldstatepilot: invalid digest syntax")
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	if err != nil || len(decoded) != len(digest) {
		return digest, errors.New("worldstatepilot: invalid digest syntax")
	}
	copy(digest[:], decoded)
	return digest, nil
}

func digestText(value dataplane.Digest) string { return "sha256:" + hex.EncodeToString(value[:]) }

func canonicalSecond(value string) string {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed.UTC().Truncate(time.Second).Format(time.RFC3339)
}

func releaseArtifactLess(left, right ReleaseArtifact) bool {
	if left.Digest != right.Digest {
		return left.Digest < right.Digest
	}
	if left.Kind != right.Kind {
		return left.Kind < right.Kind
	}
	if left.NativeKey != right.NativeKey {
		return left.NativeKey < right.NativeKey
	}
	return left.Path < right.Path
}

func safeReleasePath(value string) bool {
	converted := filepath.FromSlash(value)
	return value != "" && !filepath.IsAbs(converted) && filepath.Clean(converted) == converted && !strings.Contains(value, "\\") && value != ".." && !strings.HasPrefix(value, "../") && value != "release-manifest.json"
}

func safeEvidenceRoot(value string) bool {
	return safeReleasePath(value) && strings.HasPrefix(value, "atlas/e4-acquisitions/world-bank-e2-matrix/spool/")
}

func casPath(value string) string {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return ""
	}
	hexDigest := strings.TrimPrefix(value, "sha256:")
	return filepath.ToSlash(filepath.Join("cas", "sha256", hexDigest[:2], hexDigest[2:4], hexDigest))
}
