package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
}

func TestTrackedOntologyTreeValidatesAndCompiles(t *testing.T) {
	root := repositoryRoot(t)
	if err := validateCommand([]string{"--root", root}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "compiled")
	if err := compileCommand([]string{"--root", root, "--out", out}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"index.json",
		"modules/tw-kernel-0.1.0.cbor",
		"modules/tw-opportunity-0.1.0.cbor",
		"modules/tw-world-state-0.1.0.cbor",
		"universes/tw-opportunity-0.1.0.cbor",
		"universes/tw-world-state-0.1.0.cbor",
	} {
		if info, err := os.Stat(filepath.Join(out, filepath.FromSlash(path))); err != nil || info.Size() == 0 {
			t.Fatalf("missing compiled artifact %s: %v", path, err)
		}
	}
}
