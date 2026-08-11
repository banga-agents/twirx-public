package main

import (
	"bytes"
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func commandRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestInvokeAndVerifyCommands(t *testing.T) {
	root := commandRoot(t)
	results := filepath.Join(t.TempDir(), "results")
	var output, diagnostics bytes.Buffer
	args := []string{"invoke", "--root", root, "--results", results, "--origin", "controlled-origin-lab", "--operation", "fixture.getOffer", "--mode", "replay", "--input", "product_id=demo-1"}
	if err := run(context.Background(), args, strings.NewReader(""), &output, &diagnostics); err != nil {
		t.Fatalf("invoke: %v, diagnostics=%s", err, diagnostics.String())
	}
	if !strings.Contains(output.String(), `"result_id"`) || !strings.Contains(output.String(), `"USD"`) {
		t.Fatalf("unexpected output: %s", output.String())
	}
	entries, err := filepath.Glob(filepath.Join(results, "*"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("result directories: %#v %v", entries, err)
	}
	output.Reset()
	if err := run(context.Background(), []string{"verify", "--root", root, "--results", results, "--bundle", entries[0]}, strings.NewReader(""), &output, &diagnostics); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"status": "verified"`) {
		t.Fatalf("unexpected verification: %s", output.String())
	}
}

func TestSchemaCommandAndBadInput(t *testing.T) {
	root := commandRoot(t)
	var output, diagnostics bytes.Buffer
	if err := run(context.Background(), []string{"schema", "--root", root, "--operation", "development.getIndicator"}, strings.NewReader(""), &output, &diagnostics); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"country"`) || !strings.Contains(output.String(), `"additionalProperties": false`) {
		t.Fatalf("unexpected schema: %s", output.String())
	}
	if err := run(context.Background(), []string{"invoke", "--input", "broken"}, strings.NewReader(""), &output, &diagnostics); err == nil {
		t.Fatal("invalid input accepted")
	}
	if err := run(context.Background(), []string{"serve", "--root", root, "--listen", "0.0.0.0:8090"}, strings.NewReader(""), &output, &diagnostics); err == nil {
		t.Fatal("public bind accepted")
	}
}
