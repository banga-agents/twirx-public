package atlas

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

func validCatalogedRecord() OriginRecord {
	reviewed := "2026-08-10T00:00:00Z"
	reviewer := "maintainer"
	return OriginRecord{
		ID: "twirx-org", Scope: ScopeGenesisPublic, CanonicalOrigin: "https://twirx.org", RegistrableDomain: "twirx.org",
		ExecutionCatalogIDs: []string{"twirx-project"},
		Catalog:             CatalogDimension{State: CatalogCataloged, ReviewedAt: &reviewed, Reviewer: &reviewer, EvidenceRefs: []string{"reports/public-readiness.md"}},
		Policy:              PolicyDimension{ReviewState: PolicyPending, Decision: DecisionUncertain, Attribution: "pending", Authentication: "pending", RatePolicy: "pending", RetentionPolicy: "pending", TermsReference: "pending", RiskState: "pending", ReviewerNotes: "pending"},
		Technical:           TechnicalDimension{Stage: TechnicalUnprofiled},
		Publisher:           Publisher{Name: "TWIRX", Kind: "public-infrastructure-project", Status: PublisherUnclaimed, EvidenceRefs: []string{}},
		Health:              HealthDimension{Status: HealthUnknown, EvidenceRefs: []string{}},
		AdapterTrust:        AdapterTrustDimension{Status: AdapterTrustNone, EvidenceRefs: []string{}},
		MappingTrust:        MappingTrustDimension{Status: MappingTrustNone, EvidenceRefs: []string{}},
		Jurisdiction:        "not_established", Languages: []string{"en"}, DomainFamilies: []string{"standards_technical_docs_open_source"}, AuthorityClass: "publisher-authored",
		Discovery:  Discovery{Sources: []string{}, SitemapURLs: []string{}, FeedURLs: []string{}, APIDescriptions: []string{}, StructuredData: []string{}, PublicEndpoints: []string{}},
		Interfaces: []InterfaceDeclaration{}, Capabilities: []CapabilityCandidate{},
		Access:    AccessMetadata{Class: AccessUnknown, AssessmentStatus: AccessNotAssessed, LicenseOrTermsRefs: []string{}, Attribution: "not_assessed", RatePolicy: "not_assessed", PaymentProtocolCandidates: []string{}, PriceDiscoveryStatus: PriceNotAssessed, OfferCandidates: []OfferCandidate{}},
		Economics: EconomicsMetadata{FundingClass: FundingUnknown}, PublisherReadiness: PublisherReadiness{Signals: []PublisherReadinessSignal{}},
		Runtime:   RuntimeRecord{RefreshClass: "disabled", Scheduler: SchedulerState{State: "disabled"}},
		Semantics: SemanticRecord{MappingModules: []string{}, ConceptIDs: []string{}, OperationIDs: []string{}, SemanticClosureDigests: []string{}},
		Evidence:  EvidenceRecord{ObservationIDs: []string{}, Artifacts: []ArtifactReference{}},
	}
}

func TestOrthogonalStatesDoNotImplyOneAnother(t *testing.T) {
	record := validCatalogedRecord()
	reviewed, reviewer := "2026-08-10T00:00:00Z", "publisher-owner"
	record.Publisher.Status = PublisherApproved
	record.Publisher.ReviewedAt = &reviewed
	record.Publisher.Reviewer = &reviewer
	record.Publisher.EvidenceRefs = []string{"decisions/005-twirx-origin-admission.md"}
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	if record.Policy.ReviewState != PolicyPending || record.Technical.Stage != TechnicalUnprofiled || record.Health.Status != HealthUnknown {
		t.Fatal("publisher approval changed an independent state dimension")
	}
}

func TestPendingPolicyNeverCountsAsCompletedDecision(t *testing.T) {
	record := validCatalogedRecord()
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	record.Policy.Decision = DecisionCatalogOnly
	if err := record.Validate(); err == nil {
		t.Fatal("pending policy accepted a completed decision")
	}
}

func TestCompletedUncertainRequiresExplicitHumanReview(t *testing.T) {
	record := validCatalogedRecord()
	reviewed, reviewer, policyID := "2026-08-10T00:00:00Z", "maintainer", "twirx-org"
	digest := "sha256:" + strings.Repeat("a", 64)
	record.Policy.ReviewState = PolicyCompleted
	record.Policy.ReviewedAt = &reviewed
	record.Policy.Reviewer = &reviewer
	record.Policy.PolicyID = &policyID
	record.Policy.PolicySetDigest = &digest
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	record.Policy.Reviewer = nil
	if err := record.Validate(); err == nil {
		t.Fatal("completed uncertain policy accepted without explicit reviewer")
	}
}

func TestTechnicalStageRequiresItsOwnEvidence(t *testing.T) {
	record := validCatalogedRecord()
	record.Technical.Stage = TechnicalCompiled
	if err := record.Validate(); err == nil {
		t.Fatal("compiled stage accepted without independent technical evidence")
	}
}

func TestDescriptiveCapabilityCannotBecomeExecutionAuthority(t *testing.T) {
	record := validCatalogedRecord()
	digest := "sha256:" + strings.Repeat("a", 64)
	record.Evidence.Artifacts = []ArtifactReference{{Path: "reports/public-readiness.md", Digest: digest}}
	record.Interfaces = []InterfaceDeclaration{{ID: "webmcp-candidate", Kind: InterfaceWebMCP, DeclarationStatus: DeclarationInferred, EndpointOrLocator: "https://twirx.org", Authentication: "unknown", Health: HealthUnknown, ExecutionStatus: InterfaceNotAdmitted, EvidenceDigest: digest}}
	record.Capabilities = []CapabilityCandidate{{NativeID: "possible-action", NativeLabel: "Possible action", SemanticOperationCandidates: []string{}, EffectClass: EffectFinancial, Status: CapabilityAdmitted, InterfaceRefs: []string{"webmcp-candidate"}, EvidenceDigest: digest}}
	if err := record.Validate(); err == nil {
		t.Fatal("financial or non-admitted capability became executable authority")
	}
}

func TestUnassessedAccessCannotClaimCommercialOffer(t *testing.T) {
	record := validCatalogedRecord()
	record.Access.OfferCandidates = []OfferCandidate{{ID: "offer", NativeLabel: "unverified", Status: OfferCandidateStatus, AccessClass: AccessSubscription, TermsRef: "pending", PriceStatus: PriceNotAssessed, EvidenceDigest: "sha256:" + strings.Repeat("a", 64)}}
	if err := record.Validate(); err == nil {
		t.Fatal("unassessed access accepted an offer claim")
	}
}

func TestReadinessSignalRequiresEvidenceAndFreshness(t *testing.T) {
	record := validCatalogedRecord()
	record.PublisherReadiness.Signals = []PublisherReadinessSignal{{Kind: SignalWebMCP, Presence: SignalPresent, Source: "page metadata", ObservationClass: ObservationObserved, Freshness: "current", StandardStatus: StandardExperimental}}
	if err := record.Validate(); err == nil {
		t.Fatal("present readiness signal accepted without dated evidence")
	}
}

func TestRegistryRejectsUnselectedPublicOrigin(t *testing.T) {
	selection := loadTestSelection(t)
	record := validCatalogedRecord()
	record.ID = "not-selected"
	registry := Registry{Format: RegistryFormat, Version: "test", Origins: []OriginRecord{record}}
	policies := loadTestPolicies(t, selection)
	if err := registry.Validate(selection, policies); err == nil {
		t.Fatal("unselected public origin admitted")
	}
}

func FuzzRegistryJSON(f *testing.F) {
	f.Add([]byte(`{"format":"tw.origin-registry/0.3","version":"test","origins":[]}`))
	f.Add([]byte(`{"format":"x","format":"y"}`))
	f.Add([]byte{0xff})
	_, file, _, _ := runtime.Caller(0)
	selection, err := LoadSelection(filepath.Join(filepath.Dir(file), "..", "..", "atlas", "genesis-500", "selection.json"))
	if err != nil {
		f.Fatal(err)
	}
	policies, err := LoadPolicySet(filepath.Join(filepath.Dir(file), "..", "..", "atlas", "policies.json"), selection)
	if err != nil {
		f.Fatal(err)
	}
	decoderPolicy := jsonbounded.Policy{MaxBytes: MaxRegistryBytes, MaxDepth: 24, MaxScalarBytes: 16 << 10, MaxContainerEntries: 4096, MaxTokens: 500000}
	f.Fuzz(func(t *testing.T, data []byte) {
		var registry Registry
		if err := jsonbounded.Decode(data, &registry, decoderPolicy, true); err == nil {
			_ = registry.Validate(selection, policies)
		}
	})
}
