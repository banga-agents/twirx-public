package e4source

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func TestTrackedDossiersAreValidAndDisabled(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	paths, err := filepath.Glob(filepath.Join(root, "atlas", "e4-sources", "*", "dossier.json"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	if len(paths) != 8 {
		t.Fatalf("expected 8 source dossiers, found %d", len(paths))
	}
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		dossier, err := Parse(data)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if dossier.NetworkExecutionState != "disabled" {
			t.Fatalf("%s enabled network execution through a descriptive dossier", path)
		}
		if dossier.ID == "grants-gov-api-e4" {
			if dossier.Policy.E4ReviewState != "completed" || dossier.Policy.E4Decision != "permit_with_constraints" || dossier.Policy.ReviewedAt == nil || dossier.Policy.Reviewer == nil {
				t.Fatalf("%s does not carry the exact completed steward review", path)
			}
		} else if dossier.Policy.E4ReviewState != "pending" {
			t.Fatalf("%s unexpectedly left fail-closed preparation", path)
		}
		if _, exists := seen[dossier.ID]; exists {
			t.Fatalf("duplicate dossier ID %q", dossier.ID)
		}
		seen[dossier.ID] = struct{}{}
	}
}

func FuzzDossier(f *testing.F) {
	f.Add([]byte(`{"format":"tw.e4-source-dossier/0.1"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
}
