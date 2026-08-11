package mcpstdio

import (
	"bytes"
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/labengine"
)

func root(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestLifecycleToolsAndRealInvocation(t *testing.T) {
	engine, err := labengine.New(root(t), filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"fixture.getOffer","arguments":{"product_id":"demo-1"}}}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	server := Server{Engine: engine, Mode: labengine.ModeReplay}
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("responses %d: %s", len(lines), output.String())
	}
	if !strings.Contains(lines[0], ProtocolVersion) || !strings.Contains(lines[1], "project.getStatus") || !strings.Contains(lines[2], "structuredContent") || !strings.Contains(lines[2], `"usd"`) {
		t.Fatalf("unexpected transcript: %s", output.String())
	}
}

func TestRejectsPreInitializationAndDuplicateKeys(t *testing.T) {
	engine, err := labengine.New(root(t), filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n", `{"jsonrpc":"2.0","id":1,"id":2,"method":"initialize"}` + "\n"} {
		var output bytes.Buffer
		server := Server{Engine: engine}
		if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.String(), `"error"`) {
			t.Fatalf("accepted %s", input)
		}
	}
}
