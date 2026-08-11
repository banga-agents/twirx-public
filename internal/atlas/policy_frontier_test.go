package atlas

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func reviewedPolicyFixture(t *testing.T) (*Selection, *PolicySet, OriginRecord) {
	t.Helper()
	selection := loadTestSelection(t)
	reviewed, reviewer := "2026-08-10T00:00:00Z", "test-reviewer"
	artifact := "sha256:" + strings.Repeat("a", 64)
	policies := &PolicySet{
		Format: PolicySetFormat, Version: "test", Statement: "Synthetic policy evidence for offline validation tests only.",
		Policies: []OriginPolicy{{
			ID: "twirx-org", OriginID: "twirx-org", CanonicalOrigin: "https://twirx.org", ReviewState: PolicyCompleted, Decision: DecisionPermitLive,
			ReviewedAt: &reviewed, Reviewer: &reviewer,
			Robots:     RobotsAssessment{State: "successful", URL: "https://twirx.org/robots.txt", ObservedAt: &reviewed, ArtifactDigest: &artifact, ProductToken: CrawlerToken},
			TermsState: "accepted", TermsReference: "SECURITY.md", Attribution: "Project attribution required", Authentication: "none_required",
			RatePolicy: "At most one request per minute", RetentionPolicy: "Raw evidence retained for at most 30 days", RiskState: "accepted",
			ReviewerNotes: "Read-only public project surface", EvidenceRefs: []string{"SECURITY.md", "conformance/robots/v1/cases.json"},
		}},
		digest: sha256.Sum256([]byte("exact-test-policy-artifact")),
	}
	if err := policies.Validate(selection); err != nil {
		t.Fatal(err)
	}
	record := validCatalogedRecord()
	record.Runtime.RefreshClass = "warm"
	record.Runtime.RequestBudget = 20
	record.Runtime.StorageBudgetByte = 25 << 20
	record.Runtime.Scheduler = SchedulerState{State: "ready", PublicValue: 80, SemanticNovelty: 50, DemonstratedDemand: 20, ChangeProbability: 30, HealthNeed: 40, EstimatedRequestCost: 10}
	policyID, policyDigest := "twirx-org", policies.DigestReference()
	record.Policy = PolicyDimension{
		ReviewState: PolicyCompleted, Decision: DecisionPermitLive, ReviewedAt: &reviewed, Reviewer: &reviewer,
		PolicyID: &policyID, PolicySetDigest: &policyDigest, RobotsDigest: &artifact,
		Attribution: "Project attribution required", Authentication: "none_required", RatePolicy: "At most one request per minute",
		RetentionPolicy: "Raw evidence retained for at most 30 days", TermsReference: "SECURITY.md", RiskState: "accepted",
		ReviewerNotes: "Read-only public project surface",
	}
	return selection, policies, record
}

func TestRegistryBindsExactPolicyArtifact(t *testing.T) {
	selection, policies, record := reviewedPolicyFixture(t)
	registry := &Registry{Format: RegistryFormat, Version: "test", Origins: []OriginRecord{record}}
	if err := registry.Validate(selection, policies); err != nil {
		t.Fatal(err)
	}
	wrongDigest := "sha256:" + strings.Repeat("b", 64)
	registry.Origins[0].Policy.PolicySetDigest = &wrongDigest
	if err := registry.Validate(selection, policies); err == nil {
		t.Fatal("registry accepted substituted policy-set digest")
	}
	registry.Origins[0] = record
	registry.Origins[0].Policy.RatePolicy = "different"
	if err := registry.Validate(selection, policies); err == nil {
		t.Fatal("registry accepted policy fields that disagree with policy artifact")
	}
}

func TestDryRunFrontierIsDeterministicAndNetworkDisabled(t *testing.T) {
	selection, policies, record := reviewedPolicyFixture(t)
	registry := &Registry{Format: RegistryFormat, Version: "test", Origins: []OriginRecord{record}}
	at := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	first, err := BuildDryRunFrontier(selection, registry, policies, at)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildDryRunFrontier(selection, registry, policies, at)
	if err != nil {
		t.Fatal(err)
	}
	if first.NetworkAccess != "disabled" || first.Mode != "dry_run" || len(first.Decisions) != RequiredCandidates || len(second.Decisions) != RequiredCandidates || len(first.Jobs) != 1 || len(second.Jobs) != 1 || first.Jobs[0] != second.Jobs[0] {
		t.Fatalf("unexpected frontier plan: %#v", first)
	}
	if first.Jobs[0].OriginID != "twirx-org" || first.Jobs[0].Priority != 9600000 {
		t.Fatalf("unexpected job: %#v", first.Jobs[0])
	}
	last := at.Format(time.RFC3339Nano)
	registry.Origins[0].Runtime.Scheduler.LastAttempt = &last
	deferred, err := BuildDryRunFrontier(selection, registry, policies, at)
	if err != nil {
		t.Fatal(err)
	}
	twirxDecision := frontierDecision(t, deferred, "twirx-org")
	if len(deferred.Jobs) != 0 || twirxDecision.Action != "defer" || twirxDecision.Reason != "not_due" {
		t.Fatalf("origin was not deferred: %#v", deferred)
	}
}

func TestDeniedPolicyCannotSchedule(t *testing.T) {
	selection, policies, record := reviewedPolicyFixture(t)
	policies.Policies[0].Decision = DecisionDeny
	policies.digest = sha256.Sum256([]byte("exact-deny-policy-artifact"))
	record.Policy.Decision = DecisionDeny
	policyDigest := policies.DigestReference()
	record.Policy.PolicySetDigest = &policyDigest
	record.Runtime.RequestBudget = 0
	record.Runtime.StorageBudgetByte = 0
	record.Runtime.RefreshClass = "disabled"
	record.Runtime.Scheduler = SchedulerState{State: "disabled"}
	registry := &Registry{Format: RegistryFormat, Version: "test", Origins: []OriginRecord{record}}
	plan, err := BuildDryRunFrontier(selection, registry, policies, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Jobs) != 0 || len(plan.Decisions) != RequiredCandidates || frontierDecision(t, plan, "twirx-org").Reason != "policy_not_live_permitted" {
		t.Fatalf("denied origin scheduled: %#v", plan)
	}
}

func TestCommittedPoliciesRemainBlockedByDecisionAndDisabledScheduler(t *testing.T) {
	selection := loadTestSelection(t)
	policies := loadTestPolicies(t, selection)
	registry, err := LoadRegistry(filepath.Join(repositoryRoot(t), "atlas", "registry.json"), selection, policies)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildDryRunFrontier(selection, registry, policies, time.Date(2026, 8, 10, 16, 1, 26, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Jobs) != 0 || len(plan.Decisions) != RequiredCandidates {
		t.Fatalf("pending origin entered frontier: %#v", plan)
	}
	for _, decision := range plan.Decisions {
		if decision.Action != "blocked" {
			t.Fatalf("pending origin entered frontier: %#v", plan)
		}
		expected := "catalog_review_pending"
		if decision.OriginID == "twirx-org" || decision.OriginID == "api-worldbank-org" {
			expected = "scheduler_disabled"
		} else if decision.OriginID == "rfc-editor-org" {
			expected = "policy_not_live_permitted"
		}
		if decision.Reason != expected {
			t.Fatalf("origin %s has reason %s, want %s", decision.OriginID, decision.Reason, expected)
		}
	}
}

func frontierDecision(t *testing.T, plan FrontierPlan, originID string) FrontierDecision {
	t.Helper()
	for _, decision := range plan.Decisions {
		if decision.OriginID == originID {
			return decision
		}
	}
	t.Fatalf("frontier decision missing for %s", originID)
	return FrontierDecision{}
}

func TestPolicyUnknownAndUnsafeRetrievalFailClosed(t *testing.T) {
	selection, policies, _ := reviewedPolicyFixture(t)
	for _, mutate := range []func(*OriginPolicy){
		func(policy *OriginPolicy) { policy.Decision = PolicyDecision("unknown") },
		func(policy *OriginPolicy) { policy.Robots.State = "unreachable" },
		func(policy *OriginPolicy) { policy.TermsState = "pending" },
		func(policy *OriginPolicy) { policy.Authentication = "api_key" },
	} {
		copySet := *policies
		copySet.Policies = append([]OriginPolicy(nil), policies.Policies...)
		mutate(&copySet.Policies[0])
		if err := copySet.Validate(selection); err == nil {
			t.Fatal("unsafe retrieval policy accepted")
		}
	}
}

func TestPolicyDecoderRejectsUnknownDuplicateTrailingAndSymlink(t *testing.T) {
	selection := loadTestSelection(t)
	valid, err := os.ReadFile(filepath.Join(repositoryRoot(t), "atlas", "policies.json"))
	if err != nil {
		t.Fatal(err)
	}
	unknown := strings.Replace(string(valid), `"format":`, `"unknown":true,"format":`, 1)
	duplicate := strings.Replace(string(valid), `"format":`, `"format":"duplicate","format":`, 1)
	for name, data := range map[string][]byte{"unknown": []byte(unknown), "duplicate": []byte(duplicate), "trailing": append(valid, []byte("{}")...)} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "policies.json")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadPolicySet(path, selection); err == nil {
				t.Fatal("malformed policy set accepted")
			}
		})
	}
	target := filepath.Join(t.TempDir(), "target.json")
	if err := os.WriteFile(target, valid, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "policies.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPolicySet(link, selection); err == nil {
		t.Fatal("symlink policy artifact accepted")
	}
}

func FuzzPolicySetJSON(f *testing.F) {
	f.Add([]byte(`{"format":"tw.atlas-policy-set/0.2","version":"test","statement":"test","policies":[]}`))
	f.Add([]byte(`{"format":"x","format":"y"}`))
	selection := loadSelectionForFuzz(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 || len(data) > MaxPolicySetBytes {
			return
		}
		var policies PolicySet
		if err := decodeStrict(data, &policies, jsonPolicy(MaxPolicySetBytes)); err == nil {
			policies.digest = sha256.Sum256(data)
			_ = policies.Validate(selection)
		}
	})
}

func loadSelectionForFuzz(f *testing.F) *Selection {
	f.Helper()
	_, file, _, _ := runtime.Caller(0)
	selection, err := LoadSelection(filepath.Join(filepath.Dir(file), "..", "..", "atlas", "genesis-500", "selection.json"))
	if err != nil {
		f.Fatal(err)
	}
	return selection
}
