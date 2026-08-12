package worldstatepilot

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCommittedWorldStateReleaseVerifiesAndRebuilds(t *testing.T) {
	root := filepath.Join("..", "..")
	prepared := filepath.Join(root, "atlas", "e4-acquisitions", "world-bank-e2-matrix")
	spool := filepath.Join(prepared, "spool")
	releaseRoot := filepath.Join(root, "generated", "e4", "releases", "world-bank-e2-matrix")
	verified, err := VerifyRelease(root, releaseRoot)
	if err != nil {
		t.Fatalf("verify committed release: %v", err)
	}
	if verified.SourceRecords != 35 || verified.Packets != 175 || verified.Frames != 35 || verified.RejectedResponses != 1 {
		t.Fatalf("unexpected real-corpus counts: %+v", verified)
	}
	rebuilt := filepath.Join(t.TempDir(), "release")
	if _, err := Compile(root, prepared, spool, rebuilt, "2026-08-12T03:12:00Z"); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	for _, relative := range []string{"release-manifest.json", "proof/world-state.json", "segments/world-state.twux"} {
		expected, err := os.ReadFile(filepath.Join(releaseRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filepath.Join(rebuilt, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, expected) {
			t.Fatalf("rebuilt artifact %s differs", relative)
		}
	}
}

func TestVerifyReleaseFailsWithoutManifest(t *testing.T) {
	if _, err := VerifyRelease(filepath.Join("..", ".."), t.TempDir()); err == nil {
		t.Fatal("accepted release without manifest-last completion")
	}
}
