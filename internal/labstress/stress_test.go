package labstress_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/labapi"
	"github.com/typed-web-commons/typed-web/internal/labengine"
	"github.com/typed-web-commons/typed-web/internal/labstress"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func fiveOperationWorkload() labstress.Workload {
	return labstress.Workload{Format: labstress.WorkloadFormat, Scenarios: []labstress.Scenario{
		{ID: "offer", OriginID: "controlled-origin-lab", OperationID: "fixture.getOffer", Input: map[string]any{"product_id": "demo-1"}, Weight: 1},
		{ID: "status", OriginID: "twirx-project", OperationID: "project.getStatus", Input: map[string]any{}, Weight: 1},
		{ID: "gate", OriginID: "twirx-project", OperationID: "project.getEngineeringGateReport", Input: map[string]any{}, Weight: 1},
		{ID: "risks", OriginID: "twirx-project", OperationID: "project.listUnresolvedRisks", Input: map[string]any{}, Weight: 1},
		{ID: "indicator", OriginID: "world-bank-indicators", OperationID: "development.getIndicator", Input: map[string]any{"country": "CHL", "indicator": "SP.POP.TOTL", "year": json.Number("2024")}, Weight: 1},
	}}
}

func TestRunMixedReplayWorkloadAndVerifyEveryBundle(t *testing.T) {
	root := repositoryRoot(t)
	engine, err := labengine.New(root, filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := labapi.New(labapi.Config{Engine: engine, StaticDir: filepath.Join(root, "lab", "static"), PerIPPerMinute: 600, PerIPBurst: 100, AuditWriter: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	report, err := labstress.Run(context.Background(), labstress.Config{BaseURL: server.URL, Requests: 5, Concurrency: 5, SimulatedClients: 5, ProofSamples: 32, Timeout: 10 * time.Second, Workload: fiveOperationWorkload()})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Pass || report.Counts.Succeeded != 5 || report.Counts.Attempted != 5 || report.PreflightChecks != 3 {
		t.Fatalf("unexpected report: %#v", report)
	}
	if len(report.Results) != 5 || report.Proof.BundlesRehashed != 5 || report.Proof.ProvenanceViews != 5 {
		t.Fatalf("proof coverage: %#v", report.Proof)
	}
	for _, operation := range report.Operations {
		if operation.Counts.Succeeded != 1 || operation.Latency.Samples != 1 {
			t.Fatalf("operation summary: %#v", operation)
		}
	}
}

func TestTargetAndWorkloadBoundaries(t *testing.T) {
	workload := fiveOperationWorkload()
	invalidTargets := []string{
		"https://example.com",
		"http://169.254.169.254",
		"file:///tmp/lab",
		"http://user:password@127.0.0.1:8090",
		"http://127.0.0.1:8090/path",
	}
	for _, base := range invalidTargets {
		if _, err := labstress.Run(context.Background(), labstress.Config{BaseURL: base, Requests: 5, Concurrency: 1, ProofSamples: 1, Timeout: time.Second, Workload: workload}); err == nil {
			t.Fatalf("accepted invalid target %q", base)
		}
	}
	duplicate := workload
	duplicate.Scenarios = append([]labstress.Scenario(nil), workload.Scenarios...)
	duplicate.Scenarios[1].ID = duplicate.Scenarios[0].ID
	if err := duplicate.Validate(); err == nil {
		t.Fatal("accepted duplicate scenario")
	}
	nested := fiveOperationWorkload()
	nested.Scenarios[0].Input["product_id"] = map[string]any{"url": "http://127.0.0.1"}
	if err := nested.Validate(); err == nil {
		t.Fatal("accepted nested workload input")
	}
	if _, err := labstress.Run(context.Background(), labstress.Config{BaseURL: "https://lab.twirx.org", Requests: 5, Concurrency: 1, SimulatedClients: 1, ProofSamples: 1, Timeout: time.Second, Workload: workload}); err == nil {
		t.Fatal("accepted simulated forwarding clients for public target")
	}
}
