package worldstatepilot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareDerivesExactReviewedE2Matrix(t *testing.T) {
	root := filepath.Join("..", "..")
	output := filepath.Join(t.TempDir(), "prepared")
	prepared, err := Prepare(root, filepath.Join(root, "atlas", "e4-plans", "world-bank-e2-matrix.json"), output)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Orders) != 36 || prepared.SchedulerEnabled || prepared.OriginID != OriginID {
		t.Fatalf("unexpected prepared plan: %+v", prepared)
	}
	for _, order := range prepared.Orders {
		if !strings.HasPrefix(order.RequestURL, "https://api.worldbank.org/v2/country/") || strings.Contains(order.RequestURL, "/country/all/") {
			t.Fatalf("order escaped E2 route: %+v", order)
		}
	}
	if _, err := os.Stat(filepath.Join(output, "prepared-manifest.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, "manual-control.json")); err != nil {
		t.Fatal(err)
	}
}

func TestPreparedOrdersRemainBoundToReviewedSources(t *testing.T) {
	root := filepath.Join("..", "..")
	temporary := t.TempDir()
	preparedRoot := filepath.Join(temporary, "prepared")
	prepared, err := Prepare(root, filepath.Join(root, "atlas", "e4-plans", "world-bank-e2-matrix.json"), preparedRoot)
	if err != nil {
		t.Fatal(err)
	}
	plan, _, err := LoadPlan(filepath.Join(root, "atlas", "e4-plans", "world-bank-e2-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPrepared(root, plan, preparedRoot, prepared); err != nil {
		t.Fatal(err)
	}
	prepared.Orders[0].RequestURL = "https://api.worldbank.org/v2/country/all/indicator/SP.POP.TOTL?format=json"
	if err := verifyPrepared(root, plan, preparedRoot, prepared); err == nil {
		t.Fatal("accepted broadened prepared URL")
	}
}

func TestPlanRejectsBroadenedOrScheduledScope(t *testing.T) {
	base := Plan{
		Format: PlanFormat, ID: "world-bank-e2-matrix-2026-08", OperationID: OperationID, ExecutionMode: "manual_once",
		Countries: []string{"CHL"}, Indicators: []string{"SP.POP.TOTL"}, Years: []string{"2024"},
		NotBefore: "2026-08-12T03:00:00Z", ExpiresAt: "2026-08-13T03:00:00Z", RequestIntervalMillis: 5000,
		MaximumRequests: 1, MaximumTotalBytes: 262144, Retention: "public_versioned_immutable_evidence",
	}
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Plan){
		func(p *Plan) { p.SchedulerEnabled = true },
		func(p *Plan) { p.ExecutionMode = "continuous" },
		func(p *Plan) { p.MaximumRequests = 2 },
		func(p *Plan) { p.RequestIntervalMillis = 4999 },
		func(p *Plan) { p.ExpiresAt = "2026-09-12T03:00:00Z" },
	} {
		candidate := base
		mutate(&candidate)
		if err := candidate.Validate(); err == nil {
			t.Fatal("accepted broadened pilot plan")
		}
	}
}

func FuzzPlanJSON(f *testing.F) {
	seed, err := os.ReadFile(filepath.Join("..", "..", "atlas", "e4-plans", "world-bank-e2-matrix.json"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(`{"format":"tw.e4-world-state-pilot/0.1","scheduler_enabled":true}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "plan.json")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, _, _ = LoadPlan(path)
	})
}
