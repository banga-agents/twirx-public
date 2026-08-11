package main

import (
	"bytes"
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestValidateMetricsAndDryRunPlan(t *testing.T) {
	for _, args := range [][]string{{"validate", "--root", testRoot(t)}, {"metrics", "--root", testRoot(t)}, {"plan", "--root", testRoot(t), "--at", "2026-08-10T00:00:00Z"}} {
		var stdout, stderr bytes.Buffer
		if err := run(context.Background(), args, &stdout, &stderr); err != nil {
			t.Fatalf("%v: %v: %s", args, err, stderr.String())
		}
		if !strings.Contains(stdout.String(), `"candidate`) && !strings.Contains(stdout.String(), `"candidates`) && !strings.Contains(stdout.String(), `"network_access": "disabled"`) {
			t.Fatalf("missing count: %s", stdout.String())
		}
	}
}

func TestServeRequiresLiteralLoopback(t *testing.T) {
	if err := run(context.Background(), []string{"serve", "--root", testRoot(t), "--listen", "0.0.0.0:8092"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("public listener accepted")
	}
}
