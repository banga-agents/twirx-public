package main

import (
	"bytes"
	"context"
	"testing"
)

func TestCommandRejectsMissingAuthorityAndUnknownSubcommands(t *testing.T) {
	for _, arguments := range [][]string{{}, {"unknown"}, {"acquire"}, {"verify-acquisition"}, {"project"}, {"verify-projection"}, {"build-release"}, {"verify-release"}, {"export-c-sample"}} {
		if err := run(context.Background(), arguments, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatalf("unsafe or incomplete arguments were accepted: %v", arguments)
		}
	}
}
