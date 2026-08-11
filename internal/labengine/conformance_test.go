package labengine

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/e2format"
	"github.com/typed-web-commons/typed-web/internal/proofbundle"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

func TestE2SharedConformanceVectors(t *testing.T) {
	root := repositoryRoot(t)
	manifest, err := os.Open(filepath.Join(root, "conformance", "e2", "vectors.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer manifest.Close()
	scanner := bufio.NewScanner(manifest)
	if !scanner.Scan() || scanner.Text() != "id\ttarget\tmutation\texpected" {
		t.Fatal("invalid vector header")
	}
	for scanner.Scan() {
		columns := strings.Split(scanner.Text(), "\t")
		if len(columns) != 4 {
			t.Fatalf("invalid vector row %q", scanner.Text())
		}
		t.Run(columns[0], func(t *testing.T) {
			engine, err := New(root, filepath.Join(t.TempDir(), "results"))
			if err != nil {
				t.Fatal(err)
			}
			invocation, err := engine.Invoke(context.Background(), Request{OriginID: "controlled-origin-lab", OperationID: "fixture.getOffer", Mode: ModeReplay, Input: map[string]string{"product_id": "demo-1"}})
			if err != nil {
				t.Fatal(err)
			}
			dir := invocation.Publication.Directory
			name := artifactName(columns[1])
			if columns[2] == "substitute-body" || columns[2] == "symlink-body" {
				name = "representation.body"
			}
			if columns[2] == "remove-adapter" {
				name = "adapter.cbor"
			}
			path := filepath.Join(dir, name)
			switch columns[2] {
			case "none":
			case "append-zero":
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if err := os.WriteFile(path, append(data, 0), 0o640); err != nil {
					t.Fatal(err)
				}
			case "substitute-body":
				if err := os.WriteFile(path, []byte("{}"), 0o640); err != nil {
					t.Fatal(err)
				}
			case "remove-manifest":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "remove-adapter":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "replace-contract-with-manifest":
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				mutated := bytes.Replace(data, []byte("contract.cbor"), []byte("manifest.cbor"), 1)
				if bytes.Equal(mutated, data) {
					t.Fatal("manifest mutation did not find contract entry")
				}
				if err := os.WriteFile(path, mutated, 0o640); err != nil {
					t.Fatal(err)
				}
			case "invalid-cbor":
				if err := os.WriteFile(path, []byte{0xff}, 0o640); err != nil {
					t.Fatal(err)
				}
			case "symlink-body":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("/etc/passwd", path); err != nil {
					t.Fatal(err)
				}
			default:
				t.Fatalf("unknown mutation %s", columns[2])
			}
			valid := validateVector(columns[1], dir, path)
			if valid != (columns[3] == "accept") {
				t.Fatalf("valid=%v expected=%s", valid, columns[3])
			}
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func artifactName(target string) string {
	switch target {
	case "bundle", "manifest":
		return "manifest.cbor"
	case "result":
		return "result.cbor"
	case "closure":
		return "semantic-closure.cbor"
	}
	return "representation.body"
}
func validateVector(target, dir, path string) bool {
	switch target {
	case "bundle":
		_, err := proofbundle.Verify(dir)
		return err == nil
	case "result":
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		_, err = e2format.UnmarshalResult(data)
		return err == nil
	case "manifest":
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		_, err = proofbundle.UnmarshalManifest(data)
		return err == nil
	case "closure":
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		return semanticClosureValid(data)
	}
	return false
}
func semanticClosureValid(data []byte) bool { // Reuse the contract generator's exact canonical outputs as the Go oracle.
	root := repositoryRootForConformance()
	set, err := New(root, filepath.Join(os.TempDir(), "twirx-conformance-unused"))
	if err != nil {
		return false
	}
	for i := range set.Contracts.Operations {
		encoded, encodeErr := twircontract.MarshalSemanticClosure(set.Contracts.Operations[i].SemanticClosure)
		if encodeErr == nil && string(encoded) == string(data) {
			return true
		}
	}
	return false
}
func repositoryRootForConformance() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
