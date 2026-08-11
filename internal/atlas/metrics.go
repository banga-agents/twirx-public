package atlas

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
)

const MetricsFormat = "tw.atlas-metrics/0.3"

type Metrics struct {
	Format          string           `json:"format"`
	GeneratedAt     *string          `json:"generated_at"`
	Commit          *string          `json:"commit"`
	SelectionDigest string           `json:"selection_digest"`
	Statement       string           `json:"statement"`
	Atlas           AtlasCounts      `json:"atlas"`
	Selection       SelectionCounts  `json:"selection"`
	Coverage        CoverageCounts   `json:"coverage"`
	Capabilities    CapabilityCounts `json:"capabilities"`
	Semantics       SemanticCounts   `json:"semantics"`
	Runtime         RuntimeCounts    `json:"runtime"`
	Learning        LearningCounts   `json:"learning"`
}

type AtlasCounts struct {
	SelectedCandidates       int            `json:"selected_candidates"`
	CanonicalRegistryRecords int            `json:"canonical_registry_records"`
	GenesisPublicRecords     int            `json:"genesis_public_records"`
	TestFixtureRecords       int            `json:"test_fixture_records"`
	PolicyRecords            int            `json:"policy_records"`
	CatalogState             map[string]int `json:"catalog_state"`
	PolicyReviewState        map[string]int `json:"policy_review_state"`
	PolicyDecision           map[string]int `json:"policy_decision"`
	TechnicalStage           map[string]int `json:"technical_stage"`
	TechnicalAtOrBeyond      map[string]int `json:"technical_at_or_beyond"`
	PublisherStatus          map[string]int `json:"publisher_status"`
	HealthStatus             map[string]int `json:"health_status"`
	AdapterTrust             map[string]int `json:"adapter_trust"`
	MappingTrust             map[string]int `json:"mapping_trust"`
}

type SelectionCounts struct {
	Total          int            `json:"total"`
	DomainFamilies map[string]int `json:"domain_families"`
}

type CoverageCounts struct {
	CountriesOrTerritories int            `json:"countries_or_territories"`
	Languages              int            `json:"languages"`
	DomainFamilies         map[string]int `json:"domain_families"`
	AuthorityClasses       map[string]int `json:"authority_classes"`
}

type SemanticCounts struct {
	CanonicalConcepts       int `json:"canonical_concepts"`
	NativeTerms             int `json:"native_terms"`
	ReviewedMappings        int `json:"reviewed_mappings"`
	DisputedMappings        int `json:"disputed_mappings"`
	EquivalentOperationSets int `json:"equivalent_operation_sets"`
}

type RuntimeCounts struct {
	Invocations24Hours        int      `json:"invocations_24h"`
	SuccessRate24Hours        *float64 `json:"success_rate_24h"`
	ProvenanceCompleteness    *float64 `json:"provenance_completeness"`
	GoCAgreement              *float64 `json:"go_c_agreement"`
	P50InternalOverheadMillis *float64 `json:"p50_internal_overhead_ms"`
	P95InternalOverheadMillis *float64 `json:"p95_internal_overhead_ms"`
}

type LearningCounts struct {
	FieldOrOperationObservations int  `json:"field_or_operation_observations"`
	AdjudicatedMappings          int  `json:"adjudicated_mappings"`
	HardNegatives                int  `json:"hard_negatives"`
	HeldOutOrigins               int  `json:"held_out_origins"`
	WSIMSeedReady                bool `json:"wsim_seed_ready"`
}

func BuildMetrics(selection *Selection, registry *Registry, policies *PolicySet) (Metrics, error) {
	if selection == nil || registry == nil || policies == nil {
		return Metrics{}, errors.New("atlas: selection, registry, and policies are required")
	}
	if err := selection.Validate(); err != nil {
		return Metrics{}, err
	}
	if err := policies.Validate(selection); err != nil {
		return Metrics{}, err
	}
	if err := registry.Validate(selection, policies); err != nil {
		return Metrics{}, err
	}

	counts := AtlasCounts{
		SelectedCandidates: len(selection.Candidates), CanonicalRegistryRecords: len(registry.Origins), PolicyRecords: len(policies.Policies),
		CatalogState:      map[string]int{string(CatalogCandidate): len(selection.Candidates), string(CatalogCataloged): 0},
		PolicyReviewState: map[string]int{string(PolicyPending): len(selection.Candidates), string(PolicyCompleted): 0},
		PolicyDecision:    map[string]int{string(DecisionPermitLive): 0, string(DecisionPermitWithConstraints): 0, string(DecisionProfileOnly): 0, string(DecisionCatalogOnly): 0, string(DecisionDeny): 0, string(DecisionUncertain): len(selection.Candidates)},
		TechnicalStage:    map[string]int{}, TechnicalAtOrBeyond: map[string]int{},
		PublisherStatus: map[string]int{string(PublisherUnclaimed): len(selection.Candidates), string(PublisherDomainVerified): 0, string(PublisherApproved): 0, string(PublisherSigned): 0},
		HealthStatus:    map[string]int{string(HealthUnknown): len(selection.Candidates), string(HealthHealthy): 0, string(HealthDegraded): 0, string(HealthStale): 0, string(HealthSuspended): 0, string(HealthRevoked): 0},
		AdapterTrust:    map[string]int{string(AdapterTrustNone): len(selection.Candidates), string(AdapterTrustCandidate): 0, string(AdapterTrustReviewed): 0, string(AdapterTrustConformant): 0, string(AdapterTrustRevoked): 0},
		MappingTrust:    map[string]int{string(MappingTrustNone): len(selection.Candidates), string(MappingTrustCandidate): 0, string(MappingTrustReviewed): 0, string(MappingTrustDisputed): 0, string(MappingTrustRevoked): 0},
	}
	for _, stage := range technicalOrder {
		counts.TechnicalStage[string(stage)] = 0
		counts.TechnicalAtOrBeyond[string(stage)] = 0
	}
	counts.TechnicalStage[string(TechnicalUnprofiled)] = len(selection.Candidates)
	counts.TechnicalAtOrBeyond[string(TechnicalUnprofiled)] = len(selection.Candidates)

	countries := make(map[string]struct{})
	languages := make(map[string]struct{})
	families := make(map[string]int)
	authorities := make(map[string]int)
	concepts := make(map[string]struct{})
	nativeModules := make(map[string]struct{})
	mappings := make(map[string]struct{})
	capabilities := NewCapabilityCounts(len(selection.Candidates))
	for index := range registry.Origins {
		record := &registry.Origins[index]
		if record.Scope == ScopeTestFixture {
			counts.TestFixtureRecords++
			continue
		}
		counts.GenesisPublicRecords++
		capabilities.Add(record, true)
		counts.CatalogState[string(CatalogCandidate)]--
		counts.CatalogState[string(record.Catalog.State)]++
		if record.Policy.ReviewState != PolicyPending {
			counts.PolicyReviewState[string(PolicyPending)]--
			counts.PolicyReviewState[string(record.Policy.ReviewState)]++
		}
		if record.Policy.Decision != DecisionUncertain {
			counts.PolicyDecision[string(DecisionUncertain)]--
			counts.PolicyDecision[string(record.Policy.Decision)]++
		}
		counts.TechnicalStage[string(TechnicalUnprofiled)]--
		counts.TechnicalStage[string(record.Technical.Stage)]++
		for stageIndex := 1; stageIndex <= record.Technical.Stage.Index(); stageIndex++ {
			counts.TechnicalAtOrBeyond[string(technicalOrder[stageIndex])]++
		}
		if record.Publisher.Status != PublisherUnclaimed {
			counts.PublisherStatus[string(PublisherUnclaimed)]--
			counts.PublisherStatus[string(record.Publisher.Status)]++
		}
		if record.Health.Status != HealthUnknown {
			counts.HealthStatus[string(HealthUnknown)]--
			counts.HealthStatus[string(record.Health.Status)]++
		}
		if record.AdapterTrust.Status != AdapterTrustNone {
			counts.AdapterTrust[string(AdapterTrustNone)]--
			counts.AdapterTrust[string(record.AdapterTrust.Status)]++
		}
		if record.MappingTrust.Status != MappingTrustNone {
			counts.MappingTrust[string(MappingTrustNone)]--
			counts.MappingTrust[string(record.MappingTrust.Status)]++
		}
		if record.Jurisdiction != "not_established" {
			countries[record.Jurisdiction] = struct{}{}
		}
		for _, language := range record.Languages {
			languages[language] = struct{}{}
		}
		for _, family := range record.DomainFamilies {
			families[family]++
		}
		authorities[record.AuthorityClass]++
		if record.Semantics.NativeModule != nil {
			nativeModules[*record.Semantics.NativeModule] = struct{}{}
		}
		for _, concept := range record.Semantics.ConceptIDs {
			concepts[concept] = struct{}{}
		}
		for _, mapping := range record.Semantics.MappingModules {
			mappings[mapping] = struct{}{}
		}
	}
	metrics := Metrics{
		Format: MetricsFormat, SelectionDigest: selection.DigestReference(),
		Statement: "Counts are derived from validated repository artifacts across orthogonal state dimensions. Pending policy review is never counted as completed. Controlled fixtures are separately counted and excluded from every Genesis-500 public-origin state, coverage, and semantic counter.",
		Atlas:     counts, Selection: SelectionCounts{Total: len(selection.Candidates), DomainFamilies: copyCounts(selection.FamilyQuotas)},
		Coverage:     CoverageCounts{CountriesOrTerritories: len(countries), Languages: len(languages), DomainFamilies: families, AuthorityClasses: authorities},
		Capabilities: capabilities,
		Semantics:    SemanticCounts{CanonicalConcepts: len(concepts), NativeTerms: len(nativeModules), ReviewedMappings: len(mappings)},
		Runtime:      RuntimeCounts{}, Learning: LearningCounts{},
	}
	metrics.Learning.WSIMSeedReady = metrics.Learning.FieldOrOperationObservations >= 10000 && metrics.Learning.AdjudicatedMappings >= 2000 && metrics.Learning.HardNegatives >= 1000 && metrics.Learning.HeldOutOrigins >= 50
	return metrics, nil
}

func MarshalMetrics(metrics Metrics) ([]byte, error) {
	if metrics.Format != MetricsFormat || metrics.SelectionDigest == "" || metrics.Atlas.SelectedCandidates != RequiredCandidates {
		return nil, errors.New("atlas: invalid metrics")
	}
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("atlas: encode metrics: %w", err)
	}
	return append(data, '\n'), nil
}

func WriteMetrics(path string, metrics Metrics) error {
	data, err := MarshalMetrics(metrics)
	if err != nil {
		return err
	}
	return atomicfile.Write(path, data, 1<<20, 0o640)
}

func copyCounts(input map[string]int) map[string]int {
	result := make(map[string]int, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
