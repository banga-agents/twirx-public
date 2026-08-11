package labengine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func repositoryRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestAllFiveOperationsReplayOffline(t *testing.T) {
	engine, err := New(repositoryRoot(t), filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	requests := []Request{
		{OriginID: "twirx-project", OperationID: "project.getStatus", Mode: ModeReplay, Input: map[string]string{}},
		{OriginID: "twirx-project", OperationID: "project.getEngineeringGateReport", Mode: ModeReplay, Input: map[string]string{}},
		{OriginID: "twirx-project", OperationID: "project.listUnresolvedRisks", Mode: ModeReplay, Input: map[string]string{}},
		{OriginID: "controlled-origin-lab", OperationID: "fixture.getOffer", Mode: ModeReplay, Input: map[string]string{"product_id": "demo-1"}},
		{OriginID: "world-bank-indicators", OperationID: "development.getIndicator", Mode: ModeReplay, Input: map[string]string{"country": "CHL", "indicator": "SP.POP.TOTL", "year": "2024"}},
	}
	for _, request := range requests {
		invocation, invokeErr := engine.Invoke(context.Background(), request)
		if invokeErr != nil {
			t.Fatalf("%s: %v", request.OperationID, invokeErr)
		}
		if invocation.Publication.ResultID == "" {
			t.Fatal("missing result ID")
		}
		if _, _, verifyErr := engine.Verify(invocation.Publication.Directory); verifyErr != nil {
			t.Fatalf("verify %s: %v", request.OperationID, verifyErr)
		}
	}
}

func TestInvocationFailsClosed(t *testing.T) {
	engine, err := New(repositoryRoot(t), filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []Request{
		{OriginID: "twirx-project", OperationID: "fixture.getOffer", Mode: ModeReplay, Input: map[string]string{"product_id": "demo-1"}},
		{OriginID: "world-bank-indicators", OperationID: "development.getIndicator", Mode: ModeReplay, Input: map[string]string{"country": "USA", "indicator": "SP.POP.TOTL", "year": "2024"}},
		{OriginID: "world-bank-indicators", OperationID: "development.getIndicator", Mode: "browser", Input: map[string]string{"country": "CHL", "indicator": "SP.POP.TOTL", "year": "2024"}},
	}
	for _, request := range tests {
		if _, err := engine.Invoke(context.Background(), request); err == nil {
			t.Fatalf("accepted %#v", request)
		}
	}
}

func TestVerificationRejectsBodyTampering(t *testing.T) {
	engine, err := New(repositoryRoot(t), filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	invocation, err := engine.Invoke(context.Background(), Request{OriginID: "controlled-origin-lab", OperationID: "fixture.getOffer", Mode: ModeReplay, Input: map[string]string{"product_id": "demo-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invocation.Publication.Directory, "representation.body"), []byte("{}"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, _, err := engine.Verify(invocation.Publication.Directory); err == nil {
		t.Fatal("tampered representation accepted")
	}
}

func TestConcurrentIdenticalPublication(t *testing.T) {
	engine, err := New(repositoryRoot(t), filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	request := Request{OriginID: "controlled-origin-lab", OperationID: "fixture.getOffer", Mode: ModeReplay, Input: map[string]string{"product_id": "demo-1"}}
	const workers = 16
	errorsFound := make(chan error, workers)
	var wait sync.WaitGroup
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, invokeErr := engine.Invoke(context.Background(), request)
			errorsFound <- invokeErr
		}()
	}
	wait.Wait()
	close(errorsFound)
	for invokeErr := range errorsFound {
		if invokeErr != nil {
			t.Fatal(invokeErr)
		}
	}
}

func BenchmarkReplayInvocation(b *testing.B) {
	engine, err := New(repositoryRoot(b), filepath.Join(b.TempDir(), "results"))
	if err != nil {
		b.Fatal(err)
	}
	request := Request{OriginID: "controlled-origin-lab", OperationID: "fixture.getOffer", Mode: ModeReplay, Input: map[string]string{"product_id": "demo-1"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request.Input = map[string]string{"product_id": "demo-1"}
		_, err := engine.Invoke(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}
