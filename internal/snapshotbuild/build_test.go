package snapshotbuild

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildIsDeterministicAndRefusesOverwrite(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	parent := t.TempDir()
	options := Options{Root: root, Output: filepath.Join(parent, "one"), SourceRevision: "test-revision", CreatedAt: "2026-08-11T00:00:00Z"}
	first, err := Build(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	options.Output = filepath.Join(parent, "two")
	second, err := Build(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if first.SnapshotID != second.SnapshotID {
		t.Fatalf("snapshot is not deterministic: %x != %x", first.SnapshotID, second.SnapshotID)
	}
	if first.Report.NetworkRequests != 0 || first.Report.Actual.AtlasIdentities != 500 || first.Report.Actual.PublicOrigins != 2 || first.Report.Actual.FixtureOrigins != 1 || first.Report.Actual.Packets == 0 {
		t.Fatalf("unexpected build report: %+v", first.Report)
	}
	if _, err := Build(context.Background(), options); err == nil {
		t.Fatal("expected overwrite rejection")
	}
	if _, err := os.Stat(filepath.Join(options.Output, "manifest.cbor")); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}
