package dataplane

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/cborlite"
)

const (
	SnapshotVersion          = "tw.semantic-snapshot-manifest/0.1"
	MaxSnapshotManifestBytes = 4 << 20
	MaxSnapshotArtifacts     = 16384
	MaxSnapshotArtifactBytes = uint64(4) << 30
	MaxSnapshotBytes         = uint64(8) << 30
)

type SnapshotArtifact struct {
	Path      string
	Digest    Digest
	Size      uint64
	MediaType string
	Role      string
}

type SnapshotView struct {
	ID               string
	DefinitionDigest Digest
	ArtifactDigest   Digest
	RowCount         uint64
	ThroughSequence  uint64
}

type SnapshotCounts struct {
	Origins        uint64
	Concepts       uint64
	Mappings       uint64
	Packets        uint64
	Deltas         uint64
	Views          uint64
	ProofArtifacts uint64
	EconomicEvents uint64
}

type SnapshotManifest struct {
	Version                string
	Channel                string
	CreatedAt              string
	SourceRevision         string
	CompilerContractDigest Digest
	CompilerVersion        string
	AtlasSelectionDigest   Digest
	CanonModuleSetDigest   Digest
	PreviousSnapshotID     OptionalDigest
	EvidenceClasses        []string
	Artifacts              []SnapshotArtifact
	Views                  []SnapshotView
	Counts                 SnapshotCounts
	HighestPacketSequence  uint64
	HighestDeltaSequence   uint64
	TotalArtifactBytes     uint64
	BuildReportDigest      Digest
}

func (m SnapshotManifest) Validate() error {
	if m.Version != SnapshotVersion {
		return fmt.Errorf("%w: snapshot version must be %q", ErrInvalid, SnapshotVersion)
	}
	identifiers := []struct{ name, value string }{
		{"channel", m.Channel},
		{"source revision", m.SourceRevision},
		{"compiler version", m.CompilerVersion},
	}
	for _, field := range identifiers {
		if err := validateIdentifier(field.name, field.value); err != nil {
			return err
		}
	}
	if err := validateUTCSecond("snapshot creation time", m.CreatedAt); err != nil {
		return err
	}
	if err := validateOptionalDigest("previous snapshot ID", m.PreviousSnapshotID); err != nil {
		return err
	}
	if len(m.EvidenceClasses) == 0 || len(m.EvidenceClasses) > 16 {
		return fmt.Errorf("%w: evidence-class count outside 1..16", ErrInvalid)
	}
	if err := validateSortedUniqueText("evidence classes", m.EvidenceClasses, 16); err != nil {
		return err
	}
	if len(m.Artifacts) == 0 || len(m.Artifacts) > MaxSnapshotArtifacts {
		return fmt.Errorf("%w: snapshot artifact count outside 1..%d", ErrInvalid, MaxSnapshotArtifacts)
	}
	required := map[string]bool{"origin_catalog": false, "concepts": false, "mappings": false, "packet_batch": false, "proof_index": false, "build_report": false}
	roleDigests := make(map[string][]Digest)
	buildReports := 0
	materializedArtifacts := 0
	var total uint64
	for i, artifact := range m.Artifacts {
		if err := ValidateSnapshotPath(artifact.Path); err != nil {
			return fmt.Errorf("artifact %d: %w", i, err)
		}
		if i > 0 && strings.Compare(m.Artifacts[i-1].Path, artifact.Path) >= 0 {
			return fmt.Errorf("%w: artifacts must be strictly sorted by path", ErrInvalid)
		}
		if artifact.Size == 0 || artifact.Size > MaxSnapshotArtifactBytes {
			return fmt.Errorf("%w: artifact %q size outside bounds", ErrInvalid, artifact.Path)
		}
		if total > MaxSnapshotBytes-artifact.Size {
			return fmt.Errorf("%w: snapshot byte total overflow or limit", ErrInvalid)
		}
		total += artifact.Size
		if err := validateText("artifact media type", artifact.MediaType, MaxShortText, false); err != nil {
			return err
		}
		if err := validateEnum("artifact role", artifact.Role, "origin_catalog", "concepts", "mappings", "packet_batch", "delta_batch", "materialized_view", "search_index", "proof_index", "economic_summary", "build_report"); err != nil {
			return err
		}
		if _, exists := required[artifact.Role]; exists {
			required[artifact.Role] = true
		}
		if artifact.Role == "build_report" {
			buildReports++
		}
		if artifact.Role == "materialized_view" {
			materializedArtifacts++
		}
		roleDigests[artifact.Role] = append(roleDigests[artifact.Role], artifact.Digest)
	}
	if total != m.TotalArtifactBytes || total == 0 || total > MaxSnapshotBytes {
		return fmt.Errorf("%w: declared snapshot bytes do not reconcile", ErrInvalid)
	}
	for _, role := range []string{"origin_catalog", "concepts", "mappings", "packet_batch", "proof_index", "build_report"} {
		if !required[role] {
			return fmt.Errorf("%w: required snapshot role %q absent", ErrInvalid, role)
		}
	}
	if buildReports != 1 || materializedArtifacts > 32 {
		return fmt.Errorf("%w: snapshot requires one build report and at most 32 materialized artifacts", ErrInvalid)
	}
	if !containsDigest(roleDigests["build_report"], m.BuildReportDigest) {
		return fmt.Errorf("%w: build-report digest is not a build_report artifact", ErrInvalid)
	}
	if len(m.Views) > 32 || m.Counts.Views != uint64(len(m.Views)) {
		return fmt.Errorf("%w: view count does not reconcile", ErrInvalid)
	}
	materializations := roleDigests["materialized_view"]
	for i, view := range m.Views {
		if err := validateIdentifier("view ID", view.ID); err != nil {
			return err
		}
		if i > 0 && strings.Compare(m.Views[i-1].ID, view.ID) >= 0 {
			return fmt.Errorf("%w: views must be strictly sorted by ID", ErrInvalid)
		}
		if view.RowCount > 10000000 {
			return fmt.Errorf("%w: view row count exceeds bound", ErrInvalid)
		}
		if !containsDigest(materializations, view.ArtifactDigest) {
			return fmt.Errorf("%w: view %q artifact is not materialized_view role", ErrInvalid, view.ID)
		}
		if view.ThroughSequence > m.HighestPacketSequence {
			return fmt.Errorf("%w: view sequence exceeds snapshot packet sequence", ErrInvalid)
		}
	}
	if m.Counts.Origins > 1000000 || m.Counts.Concepts > 10000000 || m.Counts.Mappings > 10000000 || m.Counts.Packets > 1000000000 || m.Counts.Deltas > 1000000000 || m.Counts.ProofArtifacts > 1000000000 || m.Counts.EconomicEvents > 1000000000 {
		return fmt.Errorf("%w: snapshot count exceeds protocol bound", ErrInvalid)
	}
	return nil
}

func containsDigest(values []Digest, wanted Digest) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func ValidateSnapshotPath(value string) error {
	if value == "" || len(value) > MaxShortText || !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || path.Clean(value) != value {
		return fmt.Errorf("%w: unsafe snapshot path %q", ErrInvalid, value)
	}
	if value == "manifest.cbor" || value == "manifest.json" || strings.HasPrefix(value, "channels/") {
		return fmt.Errorf("%w: reserved snapshot path %q", ErrInvalid, value)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("%w: unsafe snapshot path segment", ErrInvalid)
		}
		for _, r := range segment {
			if r < 0x20 || r == 0x7f {
				return fmt.Errorf("%w: control character in snapshot path", ErrInvalid)
			}
		}
		lower := strings.ToLower(segment)
		if strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") {
			return fmt.Errorf("%w: encoded separator in snapshot path", ErrInvalid)
		}
	}
	return nil
}

func MarshalSnapshotManifest(m SnapshotManifest) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(18)
	enc.Text(m.Version)
	enc.Text(m.Channel)
	enc.Text(m.CreatedAt)
	enc.Text(m.SourceRevision)
	encodeDigest(&enc, m.CompilerContractDigest)
	enc.Text(m.CompilerVersion)
	encodeDigest(&enc, m.AtlasSelectionDigest)
	encodeDigest(&enc, m.CanonModuleSetDigest)
	encodeOptionalDigest(&enc, m.PreviousSnapshotID)
	encodeTextSet(&enc, m.EvidenceClasses)
	enc.Array(uint64(len(m.Artifacts)))
	for _, artifact := range m.Artifacts {
		enc.Array(5)
		enc.Text(artifact.Path)
		encodeDigest(&enc, artifact.Digest)
		enc.Uint(artifact.Size)
		enc.Text(artifact.MediaType)
		enc.Text(artifact.Role)
	}
	enc.Array(uint64(len(m.Views)))
	for _, view := range m.Views {
		enc.Array(5)
		enc.Text(view.ID)
		encodeDigest(&enc, view.DefinitionDigest)
		encodeDigest(&enc, view.ArtifactDigest)
		enc.Uint(view.RowCount)
		enc.Uint(view.ThroughSequence)
	}
	enc.Array(8)
	enc.Uint(m.Counts.Origins)
	enc.Uint(m.Counts.Concepts)
	enc.Uint(m.Counts.Mappings)
	enc.Uint(m.Counts.Packets)
	enc.Uint(m.Counts.Deltas)
	enc.Uint(m.Counts.Views)
	enc.Uint(m.Counts.ProofArtifacts)
	enc.Uint(m.Counts.EconomicEvents)
	enc.Uint(m.HighestPacketSequence)
	enc.Uint(m.HighestDeltaSequence)
	enc.Uint(m.TotalArtifactBytes)
	encodeDigest(&enc, m.BuildReportDigest)
	enc.Array(0)
	encoded := enc.Bytes()
	if len(encoded) > MaxSnapshotManifestBytes {
		return nil, fmt.Errorf("%w: snapshot manifest exceeds %d bytes", ErrInvalid, MaxSnapshotManifestBytes)
	}
	return encoded, nil
}

func UnmarshalSnapshotManifest(data []byte) (SnapshotManifest, error) {
	var m SnapshotManifest
	dec, err := checkedDocument(data, MaxSnapshotManifestBytes)
	if err != nil {
		return m, err
	}
	if n, e := dec.Array(); e != nil || n != 18 {
		return m, fmt.Errorf("%w: snapshot manifest array", ErrInvalid)
	}
	if m.Version, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.Channel, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.CreatedAt, err = dec.Text(20); err != nil {
		return m, err
	}
	if m.SourceRevision, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.CompilerContractDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if m.CompilerVersion, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.AtlasSelectionDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if m.CanonModuleSetDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if m.PreviousSnapshotID, err = decodeOptionalDigest(dec); err != nil {
		return m, err
	}
	if m.EvidenceClasses, err = decodeTextSet(dec, 16, 1); err != nil {
		return m, err
	}
	artifactCount, e := dec.Array()
	if e != nil || artifactCount == 0 || artifactCount > MaxSnapshotArtifacts {
		return m, fmt.Errorf("%w: snapshot artifact count", ErrInvalid)
	}
	for i := uint64(0); i < artifactCount; i++ {
		if n, x := dec.Array(); x != nil || n != 5 {
			return m, fmt.Errorf("%w: snapshot artifact array", ErrInvalid)
		}
		var artifact SnapshotArtifact
		if artifact.Path, err = dec.Text(MaxShortText); err != nil {
			return m, err
		}
		if artifact.Digest, err = decodeDigest(dec); err != nil {
			return m, err
		}
		if artifact.Size, err = dec.Uint(); err != nil {
			return m, err
		}
		if artifact.MediaType, err = dec.Text(MaxShortText); err != nil {
			return m, err
		}
		if artifact.Role, err = dec.Text(MaxIdentifier); err != nil {
			return m, err
		}
		m.Artifacts = append(m.Artifacts, artifact)
	}
	viewCount, e := dec.Array()
	if e != nil || viewCount > 32 {
		return m, fmt.Errorf("%w: snapshot view count", ErrInvalid)
	}
	for i := uint64(0); i < viewCount; i++ {
		if n, x := dec.Array(); x != nil || n != 5 {
			return m, fmt.Errorf("%w: snapshot view array", ErrInvalid)
		}
		var view SnapshotView
		if view.ID, err = dec.Text(MaxIdentifier); err != nil {
			return m, err
		}
		if view.DefinitionDigest, err = decodeDigest(dec); err != nil {
			return m, err
		}
		if view.ArtifactDigest, err = decodeDigest(dec); err != nil {
			return m, err
		}
		if view.RowCount, err = dec.Uint(); err != nil {
			return m, err
		}
		if view.ThroughSequence, err = dec.Uint(); err != nil {
			return m, err
		}
		m.Views = append(m.Views, view)
	}
	if n, x := dec.Array(); x != nil || n != 8 {
		return m, fmt.Errorf("%w: snapshot counts array", ErrInvalid)
	}
	counts := []*uint64{&m.Counts.Origins, &m.Counts.Concepts, &m.Counts.Mappings, &m.Counts.Packets, &m.Counts.Deltas, &m.Counts.Views, &m.Counts.ProofArtifacts, &m.Counts.EconomicEvents}
	for _, target := range counts {
		if *target, err = dec.Uint(); err != nil {
			return m, err
		}
	}
	if m.HighestPacketSequence, err = dec.Uint(); err != nil {
		return m, err
	}
	if m.HighestDeltaSequence, err = dec.Uint(); err != nil {
		return m, err
	}
	if m.TotalArtifactBytes, err = dec.Uint(); err != nil {
		return m, err
	}
	if m.BuildReportDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return m, err
	}
	if err = finish(dec); err != nil {
		return m, err
	}
	return m, m.Validate()
}

// VerifySnapshotDirectory verifies a complete local snapshot without network
// access. expectedID is detached; a zero digest means the caller did not
// provide an expected identity and receives the computed identity on success.
func VerifySnapshotDirectory(directory string, expectedID Digest) (SnapshotManifest, Digest, error) {
	var manifest SnapshotManifest
	root, err := openSnapshotRoot(directory)
	if err != nil {
		return manifest, Digest{}, fmt.Errorf("open snapshot root: %w", err)
	}
	defer root.Close()
	manifestBytes, err := readBoundedRegular(root, "manifest.cbor", MaxSnapshotManifestBytes)
	if err != nil {
		return manifest, Digest{}, fmt.Errorf("read manifest: %w", err)
	}
	computedID := DigestBytes(manifestBytes)
	if expectedID != (Digest{}) && expectedID != computedID {
		return manifest, computedID, fmt.Errorf("%w: detached snapshot ID mismatch", ErrInvalid)
	}
	manifest, err = UnmarshalSnapshotManifest(manifestBytes)
	if err != nil {
		return manifest, computedID, err
	}
	for _, artifact := range manifest.Artifacts {
		file, info, openErr := root.OpenRegular(artifact.Path)
		if openErr != nil {
			return manifest, computedID, fmt.Errorf("open artifact %q: %w", artifact.Path, openErr)
		}
		if uint64(info.Size()) != artifact.Size {
			file.Close()
			return manifest, computedID, fmt.Errorf("%w: artifact %q type or size mismatch", ErrInvalid, artifact.Path)
		}
		hash := sha256.New()
		copied, copyErr := io.CopyN(hash, file, int64(artifact.Size))
		var trailing [1]byte
		extra, trailingErr := file.Read(trailing[:])
		closeErr := file.Close()
		if copyErr != nil || copied != int64(artifact.Size) || (trailingErr != io.EOF && trailingErr != nil) || extra != 0 || closeErr != nil {
			return manifest, computedID, fmt.Errorf("%w: artifact %q read mismatch", ErrInvalid, artifact.Path)
		}
		if !bytes.Equal(hash.Sum(nil), artifact.Digest[:]) {
			return manifest, computedID, fmt.Errorf("%w: artifact %q digest mismatch", ErrInvalid, artifact.Path)
		}
	}
	return manifest, computedID, nil
}

type snapshotRoot interface {
	OpenRegular(name string) (*os.File, os.FileInfo, error)
	Close() error
}

func readBoundedRegular(root snapshotRoot, name string, maximum int) ([]byte, error) {
	file, info, err := root.OpenRegular(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if info.Size() <= 0 || info.Size() > int64(maximum) {
		return nil, fmt.Errorf("%w: %q is not a bounded regular file", ErrInvalid, name)
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil || len(data) != int(info.Size()) {
		return nil, fmt.Errorf("%w: failed bounded read of %q", ErrInvalid, name)
	}
	return data, nil
}

func validateSnapshotOpenPath(name string) error {
	if err := ValidateSnapshotPath(name); err != nil && name != "manifest.cbor" {
		return err
	}
	return nil
}
