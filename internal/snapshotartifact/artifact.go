// Package snapshotartifact defines deterministic, bounded containers used by
// the read-only Genesis snapshot runtime. The canonical protocol objects
// inside these containers remain the deterministic-CBOR dataplane objects.
package snapshotartifact

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	PacketSegmentFormat = "tw.snapshot-packet-segment/0.1"
	DeltaSegmentFormat  = "tw.snapshot-delta-segment/0.1"
	ConceptsFormat      = "tw.snapshot-concepts/0.1"
	MappingsFormat      = "tw.snapshot-mappings/0.1"
	ProofIndexFormat    = "tw.snapshot-proof-index/0.1"
	ViewFormat          = "tw.snapshot-view/0.1"
	BuildReportFormat   = "tw.snapshot-build-report/0.1"

	MaxArtifactBytes  = 16 << 20
	MaxSegmentEntries = 32768
	MaxIndexEntries   = 32768
)

var ErrInvalid = errors.New("snapshotartifact: invalid artifact")

type PacketEntry struct {
	Sequence uint64 `json:"sequence"`
	Digest   string `json:"digest"`
	CBOR     []byte `json:"cbor"`
}

type PacketSegment struct {
	Format        string        `json:"format"`
	StartSequence uint64        `json:"start_sequence"`
	Entries       []PacketEntry `json:"entries"`
}

type DeltaEntry struct {
	Sequence uint64 `json:"sequence"`
	Digest   string `json:"digest"`
	CBOR     []byte `json:"cbor"`
}

type DeltaSegment struct {
	Format        string       `json:"format"`
	StartSequence uint64       `json:"start_sequence"`
	Entries       []DeltaEntry `json:"entries"`
}

type Concepts struct {
	Format   string   `json:"format"`
	Concepts []string `json:"concepts"`
	Modules  []string `json:"modules"`
}

type Mapping struct {
	ID            string `json:"id"`
	NativeTerm    string `json:"native_term"`
	SemanticTerm  string `json:"semantic_term"`
	Status        string `json:"status"`
	EvidenceClass string `json:"evidence_class"`
}

type Mappings struct {
	Format   string    `json:"format"`
	Mappings []Mapping `json:"mappings"`
}

type ProofArtifact struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   uint64 `json:"size"`
}

type ProofEntry struct {
	ProofType         string          `json:"proof_type"`
	PacketDigest      string          `json:"packet_digest"`
	EvidenceID        string          `json:"evidence_id"`
	EvidenceClass     string          `json:"evidence_class"`
	ExecutionOriginID string          `json:"execution_origin_id"`
	OperationID       string          `json:"operation_id"`
	FieldID           string          `json:"field_id"`
	Artifacts         []ProofArtifact `json:"artifacts"`
}

type ProofIndex struct {
	Format  string       `json:"format"`
	Entries []ProofEntry `json:"entries"`
}

type ViewRow struct {
	PacketDigest    string `json:"packet_digest"`
	OriginID        string `json:"origin_id"`
	SubjectID       string `json:"subject_id"`
	NativeTerm      string `json:"native_term"`
	NativeLexical   string `json:"native_lexical"`
	NativeStatus    string `json:"native_status"`
	SemanticTerm    string `json:"semantic_term"`
	SemanticLexical string `json:"semantic_lexical"`
	Lane            string `json:"lane"`
	ObservedAt      string `json:"observed_at"`
}

type View struct {
	Format          string    `json:"format"`
	ID              string    `json:"id"`
	Definition      string    `json:"definition"`
	ThroughSequence uint64    `json:"through_sequence"`
	PublicOnly      bool      `json:"public_only"`
	EvidenceClasses []string  `json:"evidence_classes"`
	Rows            []ViewRow `json:"rows"`
}

type ActualCounts struct {
	AtlasIdentities   uint64 `json:"atlas_identities"`
	ArchiveProfiles   uint64 `json:"archive_profiles"`
	PublicOrigins     uint64 `json:"public_origins_with_packets"`
	FixtureOrigins    uint64 `json:"fixture_origins_with_packets"`
	Operations        uint64 `json:"operations"`
	Packets           uint64 `json:"packets"`
	PublicPackets     uint64 `json:"public_packets"`
	FixturePackets    uint64 `json:"fixture_packets"`
	ResolvedPackets   uint64 `json:"resolved_packets"`
	UnresolvedPackets uint64 `json:"unresolved_packets"`
	Deltas            uint64 `json:"deltas"`
	MaterializedViews uint64 `json:"materialized_views"`
	ProofArtifacts    uint64 `json:"proof_artifacts"`
}

type TargetCounts struct {
	AtlasIdentities   uint64 `json:"atlas_identities"`
	ArchiveProfiles   uint64 `json:"archive_profiles"`
	CurrentObserved   uint64 `json:"current_observations"`
	Packets           uint64 `json:"packets"`
	MaterializedViews uint64 `json:"materialized_views"`
}

type BuildReport struct {
	Format               string       `json:"format"`
	SourceRevision       string       `json:"source_revision"`
	BuiltAt              string       `json:"built_at"`
	Mode                 string       `json:"mode"`
	NetworkRequests      uint64       `json:"network_requests"`
	CurrentClaimsMade    bool         `json:"current_claims_made"`
	FixtureCountedPublic bool         `json:"fixture_counted_public"`
	Actual               ActualCounts `json:"actual"`
	FundingDemoTarget    TargetCounts `json:"funding_demo_target"`
	Limitations          []string     `json:"limitations"`
}

func Marshal(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) == 0 || len(data) > MaxArtifactBytes {
		return nil, fmt.Errorf("%w: encoded size outside bounds", ErrInvalid)
	}
	return data, nil
}

func Decode(data []byte, target any) error {
	if len(data) == 0 || len(data) > MaxArtifactBytes {
		return fmt.Errorf("%w: byte length outside bounds", ErrInvalid)
	}
	policy := jsonbounded.Policy{MaxBytes: MaxArtifactBytes, MaxDepth: 16, MaxScalarBytes: 4 << 20, MaxContainerEntries: MaxSegmentEntries * 16, MaxTokens: 1000000}
	if err := jsonbounded.Decode(data, target, policy, true); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return nil
}

func (s PacketSegment) Validate() error {
	if s.Format != PacketSegmentFormat || s.StartSequence == 0 || len(s.Entries) == 0 || len(s.Entries) > MaxSegmentEntries {
		return fmt.Errorf("%w: packet segment metadata", ErrInvalid)
	}
	var prior dataplane.Digest
	for i, entry := range s.Entries {
		if entry.Sequence != s.StartSequence+uint64(i) || len(entry.CBOR) == 0 || len(entry.CBOR) > dataplane.MaxDocumentBytes {
			return fmt.Errorf("%w: packet entry %d bounds or sequence", ErrInvalid, i)
		}
		digest, err := ParseDigest(entry.Digest)
		if err != nil || digest != dataplane.DigestBytes(entry.CBOR) {
			return fmt.Errorf("%w: packet entry %d digest", ErrInvalid, i)
		}
		if i > 0 && bytes.Compare(prior[:], digest[:]) >= 0 {
			return fmt.Errorf("%w: packet entries are not digest-sorted", ErrInvalid)
		}
		if _, err := dataplane.UnmarshalPacket(entry.CBOR); err != nil {
			return fmt.Errorf("%w: packet entry %d: %v", ErrInvalid, i, err)
		}
		prior = digest
	}
	return nil
}

func UnmarshalPacketSegment(data []byte) (PacketSegment, error) {
	var segment PacketSegment
	if err := Decode(data, &segment); err != nil {
		return segment, err
	}
	return segment, segment.Validate()
}

func (s DeltaSegment) Validate() error {
	if s.Format != DeltaSegmentFormat || (len(s.Entries) > 0 && s.StartSequence == 0) || len(s.Entries) > MaxSegmentEntries {
		return fmt.Errorf("%w: delta segment metadata", ErrInvalid)
	}
	var prior dataplane.Digest
	for i, entry := range s.Entries {
		if entry.Sequence != s.StartSequence+uint64(i) || len(entry.CBOR) == 0 || len(entry.CBOR) > dataplane.MaxDocumentBytes {
			return fmt.Errorf("%w: delta entry %d bounds or sequence", ErrInvalid, i)
		}
		digest, err := ParseDigest(entry.Digest)
		if err != nil || digest != dataplane.DigestBytes(entry.CBOR) {
			return fmt.Errorf("%w: delta entry %d digest", ErrInvalid, i)
		}
		if i > 0 && bytes.Compare(prior[:], digest[:]) >= 0 {
			return fmt.Errorf("%w: delta entries are not digest-sorted", ErrInvalid)
		}
		if _, err := dataplane.UnmarshalDelta(entry.CBOR); err != nil {
			return fmt.Errorf("%w: delta entry %d: %v", ErrInvalid, i, err)
		}
		prior = digest
	}
	return nil
}

func UnmarshalDeltaSegment(data []byte) (DeltaSegment, error) {
	var segment DeltaSegment
	if err := Decode(data, &segment); err != nil {
		return segment, err
	}
	return segment, segment.Validate()
}

func (p ProofIndex) Validate() error {
	if p.Format != ProofIndexFormat || len(p.Entries) == 0 || len(p.Entries) > MaxIndexEntries {
		return fmt.Errorf("%w: proof index metadata", ErrInvalid)
	}
	previousPacket := ""
	for index, entry := range p.Entries {
		if entry.ProofType != "e2_bundle" && entry.ProofType != "controlled_scale_fixture" && entry.ProofType != "archive_capture" {
			return fmt.Errorf("%w: proof entry %d type", ErrInvalid, index)
		}
		if _, err := ParseDigest(entry.PacketDigest); err != nil || entry.PacketDigest <= previousPacket {
			return fmt.Errorf("%w: proof entry %d packet identity or order", ErrInvalid, index)
		}
		if _, err := ParseDigest(entry.EvidenceID); err != nil || entry.EvidenceClass == "" || entry.ExecutionOriginID == "" || entry.OperationID == "" || entry.FieldID == "" || len(entry.Artifacts) == 0 || len(entry.Artifacts) > 32 {
			return fmt.Errorf("%w: proof entry %d identity", ErrInvalid, index)
		}
		previousArtifact := ""
		for artifactIndex, artifact := range entry.Artifacts {
			if artifact.Name == "" || artifact.Name <= previousArtifact || artifact.Size == 0 {
				return fmt.Errorf("%w: proof entry %d artifact %d metadata", ErrInvalid, index, artifactIndex)
			}
			if err := dataplane.ValidateSnapshotPath(artifact.Path); err != nil {
				return fmt.Errorf("%w: proof entry %d artifact path", ErrInvalid, index)
			}
			if _, err := ParseDigest(artifact.Digest); err != nil {
				return fmt.Errorf("%w: proof entry %d artifact digest", ErrInvalid, index)
			}
			previousArtifact = artifact.Name
		}
		previousPacket = entry.PacketDigest
	}
	return nil
}

func UnmarshalProofIndex(data []byte) (ProofIndex, error) {
	var index ProofIndex
	if err := Decode(data, &index); err != nil {
		return index, err
	}
	return index, index.Validate()
}

func ParseDigest(value string) (dataplane.Digest, error) {
	var digest dataplane.Digest
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return digest, fmt.Errorf("%w: malformed digest reference", ErrInvalid)
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	if err != nil || len(decoded) != len(digest) || value != "sha256:"+hex.EncodeToString(decoded) {
		return digest, fmt.Errorf("%w: malformed digest reference", ErrInvalid)
	}
	copy(digest[:], decoded)
	return digest, nil
}

func DigestReference(digest dataplane.Digest) string {
	return "sha256:" + hex.EncodeToString(digest[:])
}

func ModuleSetDigest(modules []string) (dataplane.Digest, error) {
	if len(modules) == 0 {
		return dataplane.Digest{}, fmt.Errorf("%w: empty module set", ErrInvalid)
	}
	for index, module := range modules {
		if module == "" || strings.ContainsRune(module, '\x00') || (index > 0 && modules[index-1] >= module) {
			return dataplane.Digest{}, fmt.Errorf("%w: module set is not strictly sorted", ErrInvalid)
		}
	}
	encoded, err := json.Marshal(modules)
	if err != nil {
		return dataplane.Digest{}, err
	}
	return dataplane.DigestBytes(encoded), nil
}

func SortedPacketEntries(packets map[dataplane.Digest][]byte) []PacketEntry {
	digests := make([]dataplane.Digest, 0, len(packets))
	for digest := range packets {
		digests = append(digests, digest)
	}
	sort.Slice(digests, func(i, j int) bool { return bytes.Compare(digests[i][:], digests[j][:]) < 0 })
	entries := make([]PacketEntry, 0, len(digests))
	for i, digest := range digests {
		entries = append(entries, PacketEntry{Sequence: uint64(i + 1), Digest: DigestReference(digest), CBOR: packets[digest]})
	}
	return entries
}

func SortedDeltaEntries(deltas map[dataplane.Digest][]byte) []DeltaEntry {
	digests := make([]dataplane.Digest, 0, len(deltas))
	for digest := range deltas {
		digests = append(digests, digest)
	}
	sort.Slice(digests, func(i, j int) bool { return bytes.Compare(digests[i][:], digests[j][:]) < 0 })
	entries := make([]DeltaEntry, 0, len(digests))
	for i, digest := range digests {
		entries = append(entries, DeltaEntry{Sequence: uint64(i + 1), Digest: DigestReference(digest), CBOR: deltas[digest]})
	}
	return entries
}
