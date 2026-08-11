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

func TestTrackedConformanceCorpus(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(source), "..", "..", "conformance", "e3-s1", "vectors.tsv")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 16<<20)
	line := 0
	for scanner.Scan() {
		line++
		if line == 1 {
			if scanner.Text() != "name\tkind\texpect\treason\thex" {
				t.Fatal("unexpected vector header")
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
		if parts[2] == "accept" && parseErr != nil {
			t.Errorf("%s rejected: %v", parts[0], parseErr)
		} else if parts[2] == "reject" && parseErr == nil {
			t.Errorf("%s accepted", parts[0])
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if line < 30 {
		t.Fatalf("only %d vector lines", line-1)
	}
}
