package admission

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/atlas"
)

func repositorySelection(t *testing.T) *atlas.Selection {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	selection, err := atlas.LoadSelection(filepath.Join(filepath.Dir(file), "..", "..", "atlas", "genesis-500", "selection.json"))
	if err != nil {
		t.Fatal(err)
	}
	return selection
}

func pendingArtifacts(t *testing.T, selection *atlas.Selection, id string) (atlas.OriginRecord, PolicyArtifact, DecisionArtifact) {
	t.Helper()
	candidate, err := selection.Find(id)
	if err != nil {
		t.Fatal(err)
	}
	reviewed, reviewer, policyID := "2026-08-11T00:00:00Z", "agent-prepared catalog dossier; founder review pending", id
	host := strings.TrimPrefix(candidate.CanonicalOrigin, "https://")
	record := atlas.OriginRecord{
		ID: id, Scope: atlas.ScopeGenesisPublic, CanonicalOrigin: candidate.CanonicalOrigin, RegistrableDomain: host,
		ExecutionCatalogIDs: []string{},
		Catalog:             atlas.CatalogDimension{State: atlas.CatalogCataloged, ReviewedAt: &reviewed, Reviewer: &reviewer, EvidenceRefs: []string{"atlas/genesis-500/selection.json"}},
		Policy:              atlas.PolicyDimension{ReviewState: atlas.PolicyPending, Decision: atlas.DecisionUncertain, PolicyID: &policyID, Attribution: "pending human review", Authentication: "pending human review", RatePolicy: "pending human review", RetentionPolicy: "pending human review", TermsReference: "pending human review", RiskState: "pending", ReviewerNotes: "No policy decision is claimed."},
		Technical:           atlas.TechnicalDimension{Stage: atlas.TechnicalUnprofiled},
		Publisher:           atlas.Publisher{Name: host, Kind: "candidate publisher", Status: atlas.PublisherUnclaimed, EvidenceRefs: []string{}},
		Health:              atlas.HealthDimension{Status: atlas.HealthUnknown, EvidenceRefs: []string{}},
		AdapterTrust:        atlas.AdapterTrustDimension{Status: atlas.AdapterTrustNone, EvidenceRefs: []string{}},
		MappingTrust:        atlas.MappingTrustDimension{Status: atlas.MappingTrustNone, EvidenceRefs: []string{}},
		Jurisdiction:        "not_established", Languages: []string{"und"}, DomainFamilies: []string{candidate.DomainFamily}, AuthorityClass: "candidate-unreviewed",
		Discovery:  atlas.Discovery{Sources: []string{}, SitemapURLs: []string{}, FeedURLs: []string{}, APIDescriptions: []string{}, StructuredData: []string{}, PublicEndpoints: []string{}},
		Interfaces: []atlas.InterfaceDeclaration{}, Capabilities: []atlas.CapabilityCandidate{},
		Access:    atlas.AccessMetadata{Class: atlas.AccessUnknown, AssessmentStatus: atlas.AccessNotAssessed, LicenseOrTermsRefs: []string{}, Attribution: "not_assessed", RatePolicy: "not_assessed", PaymentProtocolCandidates: []string{}, PriceDiscoveryStatus: atlas.PriceNotAssessed, OfferCandidates: []atlas.OfferCandidate{}},
		Economics: atlas.EconomicsMetadata{FundingClass: atlas.FundingUnknown}, PublisherReadiness: atlas.PublisherReadiness{Signals: []atlas.PublisherReadinessSignal{}},
		Runtime:   atlas.RuntimeRecord{RefreshClass: "disabled", Scheduler: atlas.SchedulerState{State: "disabled"}},
		Semantics: atlas.SemanticRecord{MappingModules: []string{}, ConceptIDs: []string{}, OperationIDs: []string{}, SemanticClosureDigests: []string{}},
		Evidence:  atlas.EvidenceRecord{ObservationIDs: []string{}, Artifacts: []atlas.ArtifactReference{}},
	}
	policy := PolicyArtifact{Format: PolicyFormat, Policy: atlas.OriginPolicy{
		ID: id, OriginID: id, CanonicalOrigin: candidate.CanonicalOrigin, ReviewState: atlas.PolicyPending, Decision: atlas.DecisionUncertain,
		Robots:     atlas.RobotsAssessment{State: "not_observed", URL: candidate.CanonicalOrigin + "/robots.txt", ProductToken: atlas.CrawlerToken},
		TermsState: "pending", TermsReference: "pending human review", Attribution: "pending human review", Authentication: "pending human review", RatePolicy: "pending human review", RetentionPolicy: "pending human review", RiskState: "pending", ReviewerNotes: "No policy decision is claimed.", EvidenceRefs: []string{"evidence/catalog-proposal.txt"},
	}, Artifacts: []EvidenceArtifact{{Path: "evidence/catalog-proposal.txt", Digest: digest([]byte("founder review required\n"))}}}
	decision := DecisionArtifact{Format: DecisionFormat, OriginID: id, ReviewState: ReviewPending, ReviewerType: "none", CatalogReviewedAt: reviewed, CatalogReviewer: reviewer, PolicyReviewState: atlas.PolicyPending, PolicyDecision: atlas.DecisionUncertain, Rationale: "Prepared for explicit founder policy review; no admission is claimed.", Constraints: []string{}, EvidenceRefs: []string{"atlas/genesis-500/selection.json"}}
	return record, policy, decision
}

func writeOrigin(t *testing.T, root string, record atlas.OriginRecord, policy PolicyArtifact, decision DecisionArtifact) {
	t.Helper()
	directory := filepath.Join(root, record.ID)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "evidence"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "evidence", "catalog-proposal.txt"), []byte("founder review required\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]any{"record.json": record, "policy-evidence.json": policy, "decision.json": decision} {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o640); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPendingDossierDoesNotBecomeCanonicalAdmission(t *testing.T) {
	selection := repositorySelection(t)
	record, policy, decision := pendingArtifacts(t, selection, "crossref-org")
	root := t.TempDir()
	writeOrigin(t, root, record, policy, decision)
	sources, err := Load(root, selection)
	if err != nil {
		t.Fatal(err)
	}
	_, _, registry, _, batch, err := Render(sources, nil, selection, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Origins) != 0 || batch.Dossiers != 1 || batch.PendingHumanReview != 1 || batch.CanonicalAdmissions != 0 || batch.PolicyCompleted != 0 {
		t.Fatalf("pending proposal changed canonical state: %#v %#v", registry, batch)
	}
}

func TestCompletedHumanDecisionIsRequiredForRendering(t *testing.T) {
	selection := repositorySelection(t)
	record, policy, decision := pendingArtifacts(t, selection, "crossref-org")
	decision.ReviewState = ReviewCompleted
	decision.AdmitToRegistry = true
	root := t.TempDir()
	writeOrigin(t, root, record, policy, decision)
	if _, err := Load(root, selection); err == nil || !strings.Contains(err.Error(), "explicit human") {
		t.Fatalf("false completed decision was accepted: %v", err)
	}
}

func TestExplicitCompletedHumanDecisionRendersDeterministically(t *testing.T) {
	selection := repositorySelection(t)
	record, policy, decision := pendingArtifacts(t, selection, "crossref-org")
	reviewed, reviewer := "2026-08-11T01:00:00Z", "founder"
	policy.Policy.ReviewState, policy.Policy.Decision = atlas.PolicyCompleted, atlas.DecisionCatalogOnly
	policy.Policy.ReviewedAt, policy.Policy.Reviewer = &reviewed, &reviewer
	policy.Policy.TermsState, policy.Policy.RiskState = "not_found", "accepted"
	record.Policy.ReviewState, record.Policy.Decision = policy.Policy.ReviewState, policy.Policy.Decision
	record.Policy.ReviewedAt, record.Policy.Reviewer = &reviewed, &reviewer
	record.Policy.TermsReference, record.Policy.RiskState = policy.Policy.TermsReference, policy.Policy.RiskState
	decision.ReviewState, decision.AdmitToRegistry, decision.ReviewerType = ReviewCompleted, true, "human"
	decision.ApprovalReference = "founder-review:test"
	decision.PolicyReviewState, decision.PolicyDecision = policy.Policy.ReviewState, policy.Policy.Decision
	decision.PolicyReviewedAt, decision.PolicyReviewer = &reviewed, &reviewer
	root := t.TempDir()
	writeOrigin(t, root, record, policy, decision)
	sources, err := Load(root, selection)
	if err != nil {
		t.Fatal(err)
	}
	_, firstPolicies, _, firstRegistry, batch, err := Render(sources, nil, selection, "test")
	if err != nil {
		t.Fatal(err)
	}
	_, secondPolicies, _, secondRegistry, _, err := Render(sources, nil, selection, "test")
	if err != nil {
		t.Fatal(err)
	}
	if string(firstPolicies) != string(secondPolicies) || string(firstRegistry) != string(secondRegistry) || batch.CanonicalAdmissions != 1 || batch.PolicyCompleted != 1 {
		t.Fatal("explicit decision did not render deterministically")
	}
}

func TestDuplicateExecutionAliasFailsClosed(t *testing.T) {
	selection := repositorySelection(t)
	first, firstPolicy, firstDecision := pendingArtifacts(t, selection, "crossref-org")
	second, secondPolicy, secondDecision := pendingArtifacts(t, selection, "data-gov")
	first.ExecutionCatalogIDs, second.ExecutionCatalogIDs = []string{"shared-alias"}, []string{"shared-alias"}
	root := t.TempDir()
	writeOrigin(t, root, first, firstPolicy, firstDecision)
	writeOrigin(t, root, second, secondPolicy, secondDecision)
	if _, err := Load(root, selection); err == nil || !strings.Contains(err.Error(), "duplicate alias") {
		t.Fatalf("duplicate alias accepted: %v", err)
	}
}

func TestRepositoryPilotHasFiveProvisionalCommercialCandidates(t *testing.T) {
	selection := repositorySelection(t)
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	sources, err := Load(filepath.Join(root, "atlas", "admissions"), selection)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, batch, err := Render(sources, nil, selection, "test")
	if err != nil {
		t.Fatal(err)
	}
	if batch.Dossiers != 25 || batch.Capabilities.CommercialOfferCandidates != 5 || batch.Capabilities.AccessClasses["subscription"] != 5 || batch.Capabilities.MachineReadablePaymentDeclarations != 0 || batch.PolicyCompleted != 3 || batch.Capabilities.AdmittedPublicReadOperations != 4 {
		t.Fatalf("unexpected future-compatible pilot counters: %#v", batch)
	}
}

func FuzzDecisionJSON(f *testing.F) {
	f.Add([]byte(`{"format":"tw.admission-decision/0.1","origin_id":"example-org","review_state":"pending"}`))
	f.Add([]byte(`{"format":"x","format":"y"}`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		var decision DecisionArtifact
		_ = decode(data, &decision)
	})
}

func BenchmarkAdmissionLoadAndRender25(b *testing.B) {
	_, file, _, _ := runtime.Caller(0)
	repository := filepath.Join(filepath.Dir(file), "..", "..")
	selection, err := atlas.LoadSelection(filepath.Join(repository, "atlas", "genesis-500", "selection.json"))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		sources, err := Load(filepath.Join(repository, "atlas", "admissions"), selection)
		if err != nil {
			b.Fatal(err)
		}
		fixtures, err := LoadFixtures(filepath.Join(repository, "atlas", "fixtures"))
		if err != nil {
			b.Fatal(err)
		}
		if _, _, _, _, _, err := Render(sources, fixtures, selection, "benchmark"); err != nil {
			b.Fatal(err)
		}
	}
}
