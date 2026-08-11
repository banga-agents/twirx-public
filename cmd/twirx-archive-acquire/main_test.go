package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommandDoesNotAcceptNetworkAuthorityFlags(t *testing.T) {
	for _, arguments := range [][]string{
		{"run", "--url", "https://example.org"},
		{"run", "--host", "example.org"},
		{"run", "--collection", "CC-MAIN-2026-25"},
		{"run", "--range", "bytes=0-1"},
		{"verify", "--root", "/tmp/example", "--url", "https://example.org"},
	} {
		var stdout, stderr bytes.Buffer
		if err := run(arguments, &stdout, &stderr); err == nil {
			t.Fatalf("network-authority arguments were accepted: %v", arguments)
		}
	}
}

func TestUsageNamesOnlySealedInputs(t *testing.T) {
	if strings.Contains(usage, "--url") || strings.Contains(usage, "--host") || strings.Contains(usage, "--range") || strings.Contains(usage, "--collection") {
		t.Fatal("usage exposes caller-selected network authority")
	}
}
