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

func BenchmarkMCPReplayToolCall(b *testing.B) {
	engine, err := labengine.New(rootForBenchmark(), filepath.Join(b.TempDir(), "results"))
	if err != nil {
		b.Fatal(err)
	}
	input := strings.Join([]string{`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"fixture.getOffer","arguments":{"product_id":"demo-1"}}}`}, "\n") + "\n"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var output bytes.Buffer
		server := Server{Engine: engine, Mode: labengine.ModeReplay}
		if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
			b.Fatal(err)
		}
	}
}

func rootForBenchmark() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
