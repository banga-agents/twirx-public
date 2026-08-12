package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

func TestPrepareCLI(t *testing.T) {
	root := filepath.Join("..", "..")
	out := filepath.Join(t.TempDir(), "prepared")
	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), []string{"prepare", "--root", root, "--plan", filepath.Join(root, "atlas", "e4-plans", "world-bank-e2-matrix.json"), "--out", out}, &stdout, &stderr); err != nil {
		t.Fatalf("prepare: %v: %s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"scheduler_enabled": false`)) {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}
