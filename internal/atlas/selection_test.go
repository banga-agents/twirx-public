package atlas

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func loadTestSelection(t *testing.T) *Selection {
	t.Helper()
	selection, err := LoadSelection(filepath.Join(repositoryRoot(t), "atlas", "genesis-500", "selection.json"))
	if err != nil {
		t.Fatal(err)
	}
	return selection
}

func loadTestPolicies(t *testing.T, selection *Selection) *PolicySet {
	t.Helper()
	policies, err := LoadPolicySet(filepath.Join(repositoryRoot(t), "atlas", "policies.json"), selection)
	if err != nil {
		t.Fatal(err)
	}
	return policies
}

func TestGenesisSelectionAndMetricsAreHonest(t *testing.T) {
	selection := loadTestSelection(t)
	policies := loadTestPolicies(t, selection)
	if len(selection.Candidates) != 500 || selection.DigestReference() != "sha256:a87ea65056f3bd76594674b953e1b9c60ad50313d5a9955da42f5a89eb5f5729" {
		t.Fatalf("unexpected selection identity: %d %s", len(selection.Candidates), selection.DigestReference())
	}
	registry, err := LoadRegistry(filepath.Join(repositoryRoot(t), "atlas", "registry.json"), selection, policies)
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := BuildMetrics(selection, registry, policies)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Atlas.SelectedCandidates != 500 || metrics.Atlas.PolicyRecords != 3 || metrics.Atlas.GenesisPublicRecords != 3 || metrics.Atlas.TestFixtureRecords != 1 || metrics.Atlas.CatalogState["candidate"] != 497 || metrics.Atlas.CatalogState["cataloged"] != 3 || metrics.Atlas.PolicyReviewState["completed"] != 3 || metrics.Atlas.PolicyReviewState["pending"] != 497 || metrics.Atlas.TechnicalStage["semantically_linked"] != 2 || metrics.Coverage.CountriesOrTerritories != 1 || metrics.Learning.WSIMSeedReady {
		t.Fatalf("metrics overstate evidence: %#v", metrics)
	}
	if metrics.Capabilities.CapabilityCandidates != 4 || metrics.Capabilities.AdmittedPublicReadOperations != 4 || metrics.Capabilities.CommercialOfferCandidates != 0 || metrics.Capabilities.MachineReadablePaymentDeclarations != 0 || metrics.Capabilities.AccessClasses["unknown"] != 500 || metrics.Capabilities.InterfaceKinds["mcp"] != 1 || metrics.Capabilities.InterfaceKinds["openapi"] != 1 {
		t.Fatalf("capability counters overstate canonical evidence: %#v", metrics.Capabilities)
	}
	twirx, err := registry.Find("twirx-org")
	if err != nil {
		t.Fatal(err)
	}
	if policies.DigestReference() != "sha256:0f7ba43def7fff5a3adb8cd8c91cc5a9c27b0731bc4b0008d61e772b01dc8c20" || twirx.Policy.ReviewState != PolicyCompleted || twirx.Policy.Decision != DecisionPermitLive || twirx.Runtime.Scheduler.State != "disabled" {
		t.Fatalf("unexpected TWIRX admission state: policy=%s registry=%#v", policies.DigestReference(), registry.Origins)
	}
}

func TestSelectionRejectsIdentityAndMaturityMutations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Selection)
	}{
		{"duplicate ID", func(s *Selection) { s.Candidates[1].ID = s.Candidates[0].ID }},
		{"duplicate origin", func(s *Selection) { s.Candidates[1].CanonicalOrigin = s.Candidates[0].CanonicalOrigin }},
		{"HTTP origin", func(s *Selection) {
			s.Candidates[0].CanonicalOrigin = strings.Replace(s.Candidates[0].CanonicalOrigin, "https://", "http://", 1)
		}},
		{"origin path", func(s *Selection) { s.Candidates[0].CanonicalOrigin += "/path" }},
		{"cataloged candidate", func(s *Selection) { s.Candidates[0].Catalog.State = CatalogCataloged }},
		{"identity hint", func(s *Selection) { value := "publisher"; s.Candidates[0].PublisherHint = &value }},
		{"wrong declared quota", func(s *Selection) { s.FamilyQuotas["health_public_health"]++ }},
		{"unknown family", func(s *Selection) { s.Candidates[0].DomainFamily = "unknown" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selection := loadTestSelection(t)
			test.mutate(selection)
			if err := selection.Validate(); err == nil {
				t.Fatal("invalid selection accepted")
			}
		})
	}
}

func TestCandidateIdentityNormalizesConventionalWWWHost(t *testing.T) {
	candidate := Candidate{ID: "rfc-editor-org", CanonicalOrigin: "https://www.rfc-editor.org", CanonicalHost: "www.rfc-editor.org", DomainFamily: "standards_technical_docs_open_source", Catalog: CandidateCatalog{State: CatalogCandidate}, LanguageHints: []string{}}
	if err := candidate.validate(); err != nil {
		t.Fatalf("conventional www migration changed origin identity: %v", err)
	}
	candidate.ID = "www-rfc-editor-org"
	if err := candidate.validate(); err == nil {
		t.Fatal("literal www-prefixed duplicate identity was accepted")
	}
}

func TestSelectionDecoderRejectsUnknownDuplicateAndTrailingData(t *testing.T) {
	selectionPath := filepath.Join(repositoryRoot(t), "atlas", "genesis-500", "selection.json")
	valid, err := os.ReadFile(selectionPath)
	if err != nil {
		t.Fatal(err)
	}
	unknown := strings.Replace(string(valid), `"format":`, `"unknown":true,"format":`, 1)
	duplicate := strings.Replace(string(valid), `"format":`, `"format":"duplicate","format":`, 1)
	for name, data := range map[string][]byte{"unknown": []byte(unknown), "duplicate": []byte(duplicate), "trailing": append(valid, []byte("{}")...)} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "selection.json")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadSelection(path); err == nil {
				t.Fatal("malformed selection accepted")
			}
		})
	}
}

func TestMetricsEncodingIsDeterministic(t *testing.T) {
	selection := loadTestSelection(t)
	policies := &PolicySet{Format: PolicySetFormat, Version: "test", Statement: "No synthetic policy records.", Policies: []OriginPolicy{}, digest: sha256.Sum256([]byte("empty-test-policy-set"))}
	registry := &Registry{Format: RegistryFormat, Version: "test", Origins: []OriginRecord{}}
	metrics, err := BuildMetrics(selection, registry, policies)
	if err != nil {
		t.Fatal(err)
	}
	first, err := MarshalMetrics(metrics)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarshalMetrics(metrics)
	if err != nil || string(first) != string(second) || !json.Valid(first) {
		t.Fatal("metrics encoding is not deterministic valid JSON")
	}
}

func FuzzSelectionJSON(f *testing.F) {
	f.Add([]byte(`{"format":"tw.atlas-selection/0.2"}`))
	f.Add([]byte(`{"format":"x","format":"y"}`))
	f.Add([]byte("{}{}"))
	policy := jsonbounded.Policy{MaxBytes: MaxSelectionBytes, MaxDepth: 12, MaxScalarBytes: 16 << 10, MaxContainerEntries: 1024, MaxTokens: 50000}
	f.Fuzz(func(t *testing.T, data []byte) {
		var selection Selection
		if err := jsonbounded.Decode(data, &selection, policy, true); err == nil {
			_ = selection.Validate()
		}
	})
}
