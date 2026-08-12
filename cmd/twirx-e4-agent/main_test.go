package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRunControlledAgent(t *testing.T) {
	root := repositoryRoot(t)
	var stdout bytes.Buffer
	if err := run([]string{"--root", root, "--scenario", "world-state.controlled-development"}, &stdout); err != nil {
		t.Fatal(err)
	}
	var result output
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Format != "tw.e4-agent-controlled-demo/0.1" || result.EvidenceClass != "test_fixture" || result.CurrentClaimsMade || result.FixtureCountPublic || result.Execution.Status != "resolved" || result.Execution.ResultCount != 1 {
		t.Fatalf("controlled output = %+v", result)
	}
}

func TestRunListsAndRejectsUnknownScenario(t *testing.T) {
	var stdout bytes.Buffer
	if err := run([]string{"--list"}, &stdout); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"world-state.controlled-development"`)) {
		t.Fatal("scenario registry missing from list")
	}
	if err := run([]string{"--root", repositoryRoot(t), "--scenario", "unknown"}, &stdout); err == nil {
		t.Fatal("unknown scenario accepted")
	}
	if err := run([]string{"--release", "missing"}, &stdout); err == nil {
		t.Fatal("public release without a trusted manifest digest was accepted")
	}
	if err := run([]string{"--benchmark-iterations", "1"}, &stdout); err == nil {
		t.Fatal("benchmarking a controlled runtime was accepted")
	}
}

func TestSummarizeDurations(t *testing.T) {
	stats := summarizeDurations([]time.Duration{10 * time.Nanosecond, time.Nanosecond, 5 * time.Nanosecond, 3 * time.Nanosecond})
	if stats.Minimum != 1 || stats.Median != 5 || stats.P95 != 10 || stats.Maximum != 10 || stats.Mean != 4 {
		t.Fatalf("duration stats = %+v", stats)
	}
}

func TestRunControlledInvestigation(t *testing.T) {
	root := repositoryRoot(t)
	var stdout bytes.Buffer
	if err := run([]string{"--root", root, "--investigation", "utility.controlled-world-and-opportunity"}, &stdout); err != nil {
		t.Fatal(err)
	}
	var result investigationOutput
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Format != "tw.e4-agent-controlled-investigation/0.1" || result.EvidenceClass != "test_fixture" || result.CurrentClaimsMade || result.FixtureCountPublic || result.Investigation.Status != "resolved" || result.Investigation.ResultCount != 2 {
		t.Fatalf("controlled investigation output = %+v", result)
	}
	stdout.Reset()
	if err := run([]string{"--list-investigations"}, &stdout); err != nil || !bytes.Contains(stdout.Bytes(), []byte(`"utility.controlled-world-and-opportunity"`)) {
		t.Fatalf("investigation registry output = %q err=%v", stdout.String(), err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
