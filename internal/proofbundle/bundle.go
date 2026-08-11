// Package proofbundle publishes and verifies bounded E2 proof bundles. The
// final manifest is written last; a directory without it is not admitted.
package proofbundle

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/cborlite"
	"github.com/typed-web-commons/typed-web/internal/e2format"
)

const (
	ManifestVersion  = "tw.bundle-manifest/0.1"
	ManifestName     = "manifest.cbor"
	MaxArtifacts     = 32
	MaxArtifactBytes = 4 << 20
	MaxManifestBytes = 64 << 10
)

var RequiredArtifacts = []string{
	"adapter.cbor", "contract.cbor", "input.cbor", "observation.cbor", "representation.body",
	"result.cbor", "semantic-closure.cbor", "transcript.json", "transport.cbor",
}

type Entry struct {
	Name   string
	Digest [32]byte
	Size   uint64
}
type Manifest struct {
	Version  string
	ResultID string
	Entries  []Entry
}
type Publication struct{ ResultID, ResultDigest, BundleID, Directory string }

func MarshalManifest(manifest Manifest) ([]byte, error) {
	if err := manifest.validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(3)
	enc.Text(manifest.Version)
	enc.Text(manifest.ResultID)
	enc.Array(uint64(len(manifest.Entries)))
	for _, entry := range manifest.Entries {
		enc.Array(3)
		enc.Text(entry.Name)
		enc.Bytestring(entry.Digest[:])
		enc.Uint(entry.Size)
	}
	encoded := enc.Bytes()
	if len(encoded) > MaxManifestBytes {
		return nil, errors.New("proofbundle: manifest exceeds limit")
	}
	return encoded, nil
}

func UnmarshalManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if len(data) == 0 || len(data) > MaxManifestBytes {
		return manifest, errors.New("proofbundle: invalid manifest size")
	}
	dec := cborlite.NewDecoder(data)
	n, err := dec.Array()
	if err != nil || n != 3 {
		return manifest, errors.New("proofbundle: invalid manifest array")
	}
	if manifest.Version, err = dec.Text(128); err != nil {
		return manifest, err
	}
	if manifest.ResultID, err = dec.Text(128); err != nil {
		return manifest, err
	}
	count, err := dec.Array()
	if err != nil || count == 0 || count > MaxArtifacts {
		return manifest, errors.New("proofbundle: invalid entry count")
	}
	manifest.Entries = make([]Entry, 0, count)
	for i := uint64(0); i < count; i++ {
		entryLen, entryErr := dec.Array()
		if entryErr != nil || entryLen != 3 {
			return manifest, errors.New("proofbundle: invalid entry array")
		}
		var entry Entry
		if entry.Name, err = dec.Text(255); err != nil {
			return manifest, err
		}
		digest, digestErr := dec.Bytestring(32)
		if digestErr != nil || len(digest) != 32 {
			return manifest, errors.New("proofbundle: invalid entry digest")
		}
		copy(entry.Digest[:], digest)
		if entry.Size, err = dec.Uint(); err != nil {
			return manifest, err
		}
		manifest.Entries = append(manifest.Entries, entry)
	}
	if dec.Remaining() != 0 {
		return manifest, errors.New("proofbundle: trailing manifest bytes")
	}
	if err := manifest.validate(); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func (m Manifest) validate() error {
	if m.Version != ManifestVersion {
		return errors.New("proofbundle: unsupported manifest version")
	}
	resultDigest, err := e2format.ParseDigestReference(m.ResultID)
	if err != nil {
		return fmt.Errorf("proofbundle: result ID: %w", err)
	}
	if len(m.Entries) == 0 || len(m.Entries) > MaxArtifacts {
		return errors.New("proofbundle: invalid entry count")
	}
	previous := ""
	foundResult := false
	for _, entry := range m.Entries {
		if !validName(entry.Name) || entry.Name <= previous {
			return errors.New("proofbundle: entries must have unique sorted safe names")
		}
		if entry.Size == 0 || entry.Size > MaxArtifactBytes {
			return errors.New("proofbundle: artifact is empty or exceeds size limit")
		}
		if entry.Name == ManifestName {
			return errors.New("proofbundle: manifest cannot list itself")
		}
		if entry.Name == "result.cbor" {
			foundResult = true
			if entry.Digest != resultDigest {
				return errors.New("proofbundle: result ID does not match result entry")
			}
		}
		previous = entry.Name
	}
	if !foundResult {
		return errors.New("proofbundle: result.cbor is required")
	}
	for _, required := range RequiredArtifacts {
		if !containsEntry(m.Entries, required) {
			return fmt.Errorf("proofbundle: missing required artifact %s", required)
		}
	}
	return nil
}

func Write(dir string, artifacts map[string][]byte) (Publication, error) {
	var publication Publication
	if len(artifacts) == 0 || len(artifacts) > MaxArtifacts {
		return publication, errors.New("proofbundle: invalid artifact count")
	}
	if _, exists := artifacts[ManifestName]; exists {
		return publication, errors.New("proofbundle: caller cannot provide manifest")
	}
	for _, required := range RequiredArtifacts {
		if _, exists := artifacts[required]; !exists {
			return publication, fmt.Errorf("proofbundle: missing required artifact %s", required)
		}
	}
	if err := os.Mkdir(dir, 0o750); err != nil {
		return publication, fmt.Errorf("proofbundle: create new directory: %w", err)
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(dir)
		}
	}()
	names := make([]string, 0, len(artifacts))
	for name := range artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]Entry, 0, len(names))
	for _, name := range names {
		data := artifacts[name]
		if !validName(name) || len(data) == 0 || len(data) > MaxArtifactBytes {
			return publication, fmt.Errorf("proofbundle: invalid artifact %q", name)
		}
		digest := sha256.Sum256(data)
		entries = append(entries, Entry{Name: name, Digest: digest, Size: uint64(len(data))})
		if err := atomicfile.Write(filepath.Join(dir, name), data, MaxArtifactBytes, 0o640); err != nil {
			return publication, err
		}
	}
	resultDigest := sha256.Sum256(artifacts["result.cbor"])
	manifest := Manifest{Version: ManifestVersion, ResultID: e2format.DigestReference(resultDigest), Entries: entries}
	manifestBytes, err := MarshalManifest(manifest)
	if err != nil {
		return publication, err
	}
	if err := atomicfile.Write(filepath.Join(dir, ManifestName), manifestBytes, MaxManifestBytes, 0o640); err != nil {
		return publication, err
	}
	bundleDigest := sha256.Sum256(manifestBytes)
	failed = false
	return Publication{ResultID: manifest.ResultID, ResultDigest: e2format.DigestReference(resultDigest), BundleID: e2format.DigestReference(bundleDigest), Directory: dir}, nil
}

func Verify(dir string) (Publication, error) {
	var publication Publication
	manifestBytes, err := readRegular(filepath.Join(dir, ManifestName), MaxManifestBytes)
	if err != nil {
		return publication, fmt.Errorf("proofbundle: final manifest unavailable: %w", err)
	}
	manifest, err := UnmarshalManifest(manifestBytes)
	if err != nil {
		return publication, err
	}
	for _, entry := range manifest.Entries {
		path := filepath.Join(dir, entry.Name)
		data, readErr := readRegular(path, MaxArtifactBytes)
		if readErr != nil {
			return publication, fmt.Errorf("proofbundle: read %s: %w", entry.Name, readErr)
		}
		if uint64(len(data)) != entry.Size || sha256.Sum256(data) != entry.Digest {
			return publication, fmt.Errorf("proofbundle: %s size or digest mismatch", entry.Name)
		}
	}
	resultData, err := readRegular(filepath.Join(dir, "result.cbor"), MaxArtifactBytes)
	if err != nil {
		return publication, err
	}
	if _, err := e2format.UnmarshalResult(resultData); err != nil {
		return publication, fmt.Errorf("proofbundle: result validation: %w", err)
	}
	bundleDigest := sha256.Sum256(manifestBytes)
	resultDigest := sha256.Sum256(resultData)
	return Publication{ResultID: manifest.ResultID, ResultDigest: e2format.DigestReference(resultDigest), BundleID: e2format.DigestReference(bundleDigest), Directory: dir}, nil
}

func readRegular(path string, maximum int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > int64(maximum) {
		return nil, errors.New("proofbundle: artifact is not a bounded regular file")
	}
	return os.ReadFile(path)
}

func validName(name string) bool {
	return name != "" && name == filepath.Base(name) && !strings.ContainsAny(name, "\\/\x00")
}
func containsEntry(entries []Entry, name string) bool {
	for _, entry := range entries {
		if entry.Name == name {
			return true
		}
	}
	return false
}
