package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWritePublishesCompleteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "result.json")
	if err := Write(path, []byte("complete\n"), 64, 0o640); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "complete\n" {
		t.Fatalf("data=%q", data)
	}
}

func TestWriteRejectsOversizeWithoutReplacingDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	if err := os.WriteFile(path, []byte("previous\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("too large"), 3, 0o640); err == nil {
		t.Fatal("expected size rejection")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "previous\n" {
		t.Fatalf("destination changed after rejection: %q", data)
	}
}
