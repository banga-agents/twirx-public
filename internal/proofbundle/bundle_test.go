package proofbundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/e2format"
)

func testArtifacts(t *testing.T) map[string][]byte {
	t.Helper()
	result := e2format.Result{
		Version: e2format.ResultVersion, InvocationID: "i", OriginID: "o", OriginVersion: "1",
		OperationID: "o.read", OperationVersion: "1", Effect: "read", Status: "resolved", ObservedAt: "2026-08-10T00:00:00Z",
		Fields: []e2format.Field{{ID: "f", Status: "resolved", NativeTerm: "f", NativeLocator: "/f", NativePresent: true, NativeLexical: "v", SemanticTerm: "s:f", SemanticType: "string", SemanticPresent: true, SemanticLexical: "v", Mapping: "identity"}},
	}
	encoded, err := e2format.MarshalResult(result)
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{
		"adapter.cbor": {0x80}, "contract.cbor": {0x80}, "input.cbor": {0x80}, "observation.cbor": {0x80},
		"representation.body": []byte("{}"), "result.cbor": encoded, "semantic-closure.cbor": {0x80}, "transcript.json": []byte("{}\n"), "transport.cbor": {0x80},
	}
}

func TestWriteVerifyAndTamper(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bundle")
	publication, err := Write(dir, testArtifacts(t))
	if err != nil {
		t.Fatal(err)
	}
	if publication.ResultID == "" || publication.BundleID == "" {
		t.Fatalf("bad publication: %#v", publication)
	}
	verified, err := Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if verified != publication {
		t.Fatalf("verification mismatch: %#v %#v", verified, publication)
	}
	if err := os.WriteFile(filepath.Join(dir, "adapter.cbor"), []byte{0x81}, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(dir); err == nil {
		t.Fatal("tampering accepted")
	}
}

func TestPartialAndUnsafeBundlesAreRejected(t *testing.T) {
	partial := t.TempDir()
	if _, err := Verify(partial); err == nil {
		t.Fatal("directory without manifest admitted")
	}
	artifacts := testArtifacts(t)
	artifacts["../escape"] = []byte("bad")
	if _, err := Write(filepath.Join(t.TempDir(), "bundle"), artifacts); err == nil {
		t.Fatal("unsafe path accepted")
	}
}

func TestEmptyArtifactIsRejected(t *testing.T) {
	artifacts := testArtifacts(t)
	artifacts["transcript.json"] = nil
	if _, err := Write(filepath.Join(t.TempDir(), "bundle"), artifacts); err == nil {
		t.Fatal("empty artifact accepted")
	}
}

func TestManifestRejectsZeroSizeEntry(t *testing.T) {
	entries := make([]Entry, 0, len(RequiredArtifacts))
	for _, name := range RequiredArtifacts {
		entries = append(entries, Entry{Name: name, Size: 1})
	}
	entries[0].Size = 0
	if _, err := MarshalManifest(Manifest{Version: ManifestVersion, ResultID: "sha256:0000000000000000000000000000000000000000000000000000000000000000", Entries: entries}); err == nil {
		t.Fatal("zero-size manifest entry accepted")
	}
}

func TestSymlinkArtifactIsRejected(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bundle")
	if _, err := Write(dir, testArtifacts(t)); err != nil {
		t.Fatal(err)
	}
	body := filepath.Join(dir, "representation.body")
	if err := os.Remove(body); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", body); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(dir); err == nil {
		t.Fatal("symlink artifact accepted")
	}
}

func FuzzUnmarshalManifest(f *testing.F) {
	entries := make([]Entry, 0, len(RequiredArtifacts))
	for _, name := range RequiredArtifacts {
		entries = append(entries, Entry{Name: name, Size: 1})
	}
	// RequiredArtifacts is sorted and result.cbor has an all-zero digest, so it
	// matches the result identifier below.
	encoded, err := MarshalManifest(Manifest{
		Version:  ManifestVersion,
		ResultID: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Entries:  entries,
	})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(encoded)
	f.Add([]byte{0x9f, 0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		manifest, decodeErr := UnmarshalManifest(data)
		if decodeErr == nil {
			reencoded, encodeErr := MarshalManifest(manifest)
			if encodeErr != nil || string(reencoded) != string(data) {
				t.Fatal("accepted manifest was not canonical")
			}
		}
	})
}
