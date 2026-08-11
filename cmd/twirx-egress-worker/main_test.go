package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteAcceptsIDButNeverURL(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"execute", "https://example.org"}, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "exactly --id") {
		t.Fatalf("positional URL was not rejected: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"execute", "--id", "https://example.org"}, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "invalid work-order ID") {
		t.Fatalf("URL-shaped ID was not rejected: %v", err)
	}
}

func TestUnknownSubcommandFailsClosed(t *testing.T) {
	if err := run([]string{"fetch", "--url", "https://example.org"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("arbitrary fetch subcommand was accepted")
	}
}
