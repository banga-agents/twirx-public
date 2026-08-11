package atlasstress

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/atlasapi"
)

func TestRunTraversesAll500WithoutNetworkAuthority(t *testing.T) {
	selection, policies, registry := loadArtifacts(t)
	server, err := atlasapi.New(selection, registry, policies)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := atlas.BuildDryRunFrontier(selection, registry, policies, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(context.Background(), server, selection, plan, Config{Rounds: 2, Workers: 8})
	if err != nil {
		t.Fatal(err)
	}
	if report.Corpus.SelectedOrigins != 500 || report.Frontier.Decisions != 500 || report.Frontier.Jobs != 0 || report.Discovery.UniqueOrigins != 500 || report.Discovery.DirectLookups != 500 || report.Workload.Requests != 1000 || report.Workload.Successes != 1000 || report.Workload.Failures != 0 || report.Adversarial.Rejected != 8 || !report.Adversarial.Recovery || report.NetworkAccess != "disabled" {
		t.Fatalf("incomplete Atlas-500 stress evidence: %#v", report)
	}
	if report.Frontier.Reasons["catalog_review_pending"] != 497 || report.Frontier.Reasons["scheduler_disabled"] != 2 || report.Frontier.Reasons["policy_not_live_permitted"] != 1 || report.Discovery.ResponseSetDigest == "" {
		t.Fatalf("frontier or integrity evidence is incomplete: %#v", report)
	}
}

func TestRunLoopbackTraversesAll500OverHTTP(t *testing.T) {
	selection, policies, registry := loadArtifacts(t)
	server, err := atlasapi.New(selection, registry, policies)
	if err != nil {
		t.Fatal(err)
	}
	loopback := httptest.NewServer(server)
	defer loopback.Close()
	plan, err := atlas.BuildDryRunFrontier(selection, registry, policies, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	report, err := RunLoopback(context.Background(), loopback.URL, selection, plan, Config{Rounds: 1, Workers: 8})
	if err != nil {
		t.Fatal(err)
	}
	if report.Mode != "literal_loopback_http" || report.NetworkAccess != "literal_loopback_only" || report.Workload.Successes != 500 {
		t.Fatalf("unexpected loopback report: %#v", report)
	}
}

func TestRunLoopbackRejectsNonLiteralOrNonLoopbackBases(t *testing.T) {
	for _, base := range []string{"https://example.com:443", "http://localhost:8092", "http://192.0.2.1:8092", "http://127.0.0.1", "http://127.0.0.1:8092/path", "ftp://127.0.0.1:8092"} {
		if _, err := RunLoopback(context.Background(), base, nil, atlas.FrontierPlan{}, Config{Rounds: 1, Workers: 1}); err == nil {
			t.Fatalf("unsafe base accepted: %s", base)
		}
	}
}

func TestRunRejectsInvalidBoundsAndIncompleteFrontier(t *testing.T) {
	selection, policies, registry := loadArtifacts(t)
	server, err := atlasapi.New(selection, registry, policies)
	if err != nil {
		t.Fatal(err)
	}
	for _, config := range []Config{{Rounds: 0, Workers: 1}, {Rounds: 1, Workers: 0}, {Rounds: 1001, Workers: 1}, {Rounds: 1, Workers: 129}} {
		if _, err := Run(context.Background(), server, selection, atlas.FrontierPlan{}, config); err == nil {
			t.Fatalf("invalid configuration accepted: %#v", config)
		}
	}
	if _, err := Run(context.Background(), server, selection, atlas.FrontierPlan{}, Config{Rounds: 1, Workers: 1}); err == nil {
		t.Fatal("incomplete frontier accepted")
	}
}

func loadArtifacts(t *testing.T) (*atlas.Selection, *atlas.PolicySet, *atlas.Registry) {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	selection, err := atlas.LoadSelection(filepath.Join(root, "atlas", "genesis-500", "selection.json"))
	if err != nil {
		t.Fatal(err)
	}
	policies, err := atlas.LoadPolicySet(filepath.Join(root, "atlas", "policies.json"), selection)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := atlas.LoadRegistry(filepath.Join(root, "atlas", "registry.json"), selection, policies)
	if err != nil {
		t.Fatal(err)
	}
	return selection, policies, registry
}
