package dataplane_test

import (
	"bufio"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

func TestTrackedE4OntologyConformanceCorpus(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(source), "..", "..", "conformance", "e4-ontology", "vectors.tsv")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 16<<20)
	line, accepted, rejected := 0, 0, 0
	for scanner.Scan() {
		line++
		if line == 1 {
			if scanner.Text() != "name\tkind\texpect\treason\thex" {
				t.Fatal("unexpected E4 vector header")
			}
			continue
		}
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 5 {
			t.Fatalf("line %d has %d fields", line, len(parts))
		}
		data, err := hex.DecodeString(parts[4])
		if err != nil {
			t.Fatalf("line %d: %v", line, err)
		}
		parseErr := dataplane.ValidateDocument(parts[1], data)
		switch parts[2] {
		case "accept":
			accepted++
			if parseErr != nil {
				t.Errorf("%s rejected: %v", parts[0], parseErr)
			}
		case "reject":
			rejected++
			if parseErr == nil {
				t.Errorf("%s accepted", parts[0])
			}
		default:
			t.Fatalf("line %d has unknown expectation %q", line, parts[2])
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if line-1 != 33 || accepted != 7 || rejected != 26 {
		t.Fatalf("unexpected E4 corpus counts: total=%d accepted=%d rejected=%d", line-1, accepted, rejected)
	}
}
