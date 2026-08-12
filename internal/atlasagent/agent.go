// Package atlasagent executes bounded curated Semantic Universe scenarios.
// Natural-language text is explanatory input only: the registry supplies the
// typed query, and the immutable runtime is the sole execution authority.
package atlasagent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/universesnapshot"
)

var (
	ErrUnknownScenario      = errors.New("atlasagent: unknown scenario")
	ErrUnknownInvestigation = errors.New("atlasagent: unknown investigation")
)

type Runtime interface {
	Query(universesnapshot.Query) ([]dataplane.Digest, error)
	Trace(dataplane.Digest) ([]byte, error)
	Layout() string
	FrameCount() uint64
}

type Scenario struct {
	ID          string
	Title       string
	Question    string
	Concepts    []string
	UniverseID  string
	Query       universesnapshot.Query
	Views       []string
	Description string
}

type Plan struct {
	ScenarioID         string     `json:"scenario_id"`
	Question           string     `json:"question"`
	Concepts           []string   `json:"concepts"`
	UniverseID         string     `json:"universe_id"`
	TypedQuery         TypedQuery `json:"typed_query"`
	QueryDigest        string     `json:"query_digest"`
	Layout             string     `json:"layout"`
	FramesAvailable    uint64     `json:"frames_available"`
	MaximumResults     uint32     `json:"maximum_results"`
	NetworkRequests    uint64     `json:"network_requests"`
	BrowserExecutions  uint64     `json:"browser_executions"`
	LiveSourceCalls    uint64     `json:"live_source_calls"`
	ModelAuthority     string     `json:"model_authority"`
	ExecutionAuthority string     `json:"execution_authority"`
}

type TypedQuery struct {
	UniverseID string `json:"universe_id,omitempty"`
	FrameType  string `json:"frame_type,omitempty"`
	NativeKey  string `json:"native_key,omitempty"`
	SlotRole   string `json:"slot_role,omitempty"`
	SlotValue  *Value `json:"slot_value,omitempty"`
	Limit      uint32 `json:"limit"`
}

type Value struct {
	Type     string `json:"type"`
	Lexical  string `json:"lexical"`
	Unit     string `json:"unit,omitempty"`
	Currency string `json:"currency,omitempty"`
}

type SlotResult struct {
	RoleID        string   `json:"role_id"`
	Status        string   `json:"status"`
	Values        []Value  `json:"values"`
	PacketDigests []string `json:"packet_digests"`
	MappingIDs    []string `json:"mapping_ids"`
	Conflict      string   `json:"conflict"`
}

type FrameResult struct {
	Digest              string       `json:"digest"`
	UniverseID          string       `json:"universe_id"`
	FrameType           string       `json:"frame_type"`
	NativeKey           string       `json:"native_key"`
	CanonicalCandidates []string     `json:"canonical_candidates"`
	TrustLane           string       `json:"trust_lane"`
	Completeness        uint64       `json:"completeness_millionths"`
	Slots               []SlotResult `json:"slots"`
}

type Execution struct {
	Status         string        `json:"status"`
	Plan           Plan          `json:"plan"`
	Results        []FrameResult `json:"results"`
	ResultCount    uint64        `json:"result_count"`
	ProofLinked    bool          `json:"proof_linked"`
	Limitations    []string      `json:"limitations"`
	SuggestedViews []string      `json:"suggested_views"`
}

// Investigation coordinates multiple independently typed scenario queries.
// It is deliberately not a semantic join: every result retains its own
// universe, source frame and proof path, and no cross-origin equivalence is
// inferred.
type Investigation struct {
	ID                string      `json:"id"`
	Title             string      `json:"title"`
	Question          string      `json:"question"`
	Status            string      `json:"status"`
	ScenarioIDs       []string    `json:"scenario_ids"`
	Universes         []string    `json:"universes"`
	Executions        []Execution `json:"executions"`
	ResultCount       uint64      `json:"result_count"`
	ProofLinkedCount  uint64      `json:"proof_linked_count"`
	NetworkRequests   uint64      `json:"network_requests"`
	BrowserExecutions uint64      `json:"browser_executions"`
	LiveSourceCalls   uint64      `json:"live_source_calls"`
	ModelAuthority    string      `json:"model_authority"`
	Limitations       []string    `json:"limitations"`
}

type InvestigationDefinition struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Question    string   `json:"question"`
	ScenarioIDs []string `json:"scenario_ids"`
	Description string   `json:"description"`
}

type Engine struct {
	runtime        Runtime
	scenarios      map[string]Scenario
	investigations map[string]InvestigationDefinition
}

func New(runtime Runtime) (*Engine, error) {
	if runtime == nil {
		return nil, fmt.Errorf("atlasagent: runtime is required")
	}
	scenarios := CuratedScenarios()
	registry := make(map[string]Scenario, len(scenarios))
	for _, scenario := range scenarios {
		if _, exists := registry[scenario.ID]; exists {
			return nil, fmt.Errorf("atlasagent: duplicate scenario %q", scenario.ID)
		}
		registry[scenario.ID] = scenario
	}
	investigations := CuratedInvestigations()
	investigationRegistry := make(map[string]InvestigationDefinition, len(investigations))
	for _, investigation := range investigations {
		if _, exists := investigationRegistry[investigation.ID]; exists {
			return nil, fmt.Errorf("atlasagent: duplicate investigation %q", investigation.ID)
		}
		for _, scenarioID := range investigation.ScenarioIDs {
			if _, exists := registry[scenarioID]; !exists {
				return nil, fmt.Errorf("atlasagent: investigation %q names unknown scenario %q", investigation.ID, scenarioID)
			}
		}
		investigationRegistry[investigation.ID] = investigation
	}
	return &Engine{runtime: runtime, scenarios: registry, investigations: investigationRegistry}, nil
}

func (engine *Engine) Execute(scenarioID string) (Execution, error) {
	scenario, ok := engine.scenarios[scenarioID]
	if !ok {
		return Execution{}, ErrUnknownScenario
	}
	display := displayQuery(scenario.Query)
	queryBytes, err := json.Marshal(display)
	if err != nil {
		return Execution{}, err
	}
	queryDigest := sha256.Sum256(queryBytes)
	plan := Plan{
		ScenarioID: scenario.ID, Question: scenario.Question, Concepts: append([]string(nil), scenario.Concepts...), UniverseID: scenario.UniverseID, TypedQuery: display,
		QueryDigest: digestText(queryDigest), Layout: engine.runtime.Layout(), FramesAvailable: engine.runtime.FrameCount(), MaximumResults: scenario.Query.Limit,
		NetworkRequests: 0, BrowserExecutions: 0, LiveSourceCalls: 0, ModelAuthority: "none", ExecutionAuthority: "validated_typed_query_over_immutable_snapshot",
	}
	digests, err := engine.runtime.Query(scenario.Query)
	if err != nil {
		return Execution{}, err
	}
	execution := Execution{Status: "unresolved", Plan: plan, SuggestedViews: append([]string(nil), scenario.Views...), Limitations: []string{scenario.Description, "This E4 alpha returns immutable frame and packet identities; full source-bundle retrieval remains a separate proof-index operation."}}
	for _, digest := range digests {
		encoded, traceErr := engine.runtime.Trace(digest)
		if traceErr != nil {
			return Execution{}, traceErr
		}
		frame, traceErr := dataplane.UnmarshalFrame(encoded)
		if traceErr != nil {
			return Execution{}, traceErr
		}
		result := FrameResult{Digest: digestText(digest), UniverseID: frame.UniverseID, FrameType: frame.FrameType, NativeKey: frame.NativeKey, CanonicalCandidates: append([]string(nil), frame.CanonicalCandidates...), TrustLane: frame.Epistemic.Lane, Completeness: frame.Epistemic.CompletenessMillionths}
		for _, slot := range frame.Slots {
			packetDigests := make([]string, len(slot.PacketDigests))
			for index := range slot.PacketDigests {
				packetDigests[index] = digestText(slot.PacketDigests[index])
			}
			values := make([]Value, len(slot.Values))
			for index := range slot.Values {
				values[index] = displayValue(slot.Values[index])
			}
			result.Slots = append(result.Slots, SlotResult{RoleID: slot.RoleID, Status: slot.Status, Values: values, PacketDigests: packetDigests, MappingIDs: append([]string(nil), slot.MappingIDs...), Conflict: slot.Conflict})
		}
		execution.Results = append(execution.Results, result)
	}
	execution.ResultCount = uint64(len(execution.Results))
	if len(execution.Results) > 0 {
		execution.Status = "resolved"
		execution.ProofLinked = true
	}
	return execution, nil
}

func (engine *Engine) ExecuteInvestigation(investigationID string) (Investigation, error) {
	definition, ok := engine.investigations[investigationID]
	if !ok {
		return Investigation{}, ErrUnknownInvestigation
	}
	result := Investigation{
		ID: definition.ID, Title: definition.Title, Question: definition.Question,
		Status: "unresolved", ScenarioIDs: append([]string(nil), definition.ScenarioIDs...),
		Executions: []Execution{}, Universes: []string{}, ModelAuthority: "none",
		Limitations: []string{
			"This alpha coordinates exact typed queries; it does not infer a semantic join or cross-origin equivalence.",
			"Each execution retains its own evidence class, frame identity and packet-proof path.",
		},
	}
	seenUniverses := make(map[string]struct{})
	resolved := uint64(0)
	for _, scenarioID := range definition.ScenarioIDs {
		execution, err := engine.Execute(scenarioID)
		if err != nil {
			return Investigation{}, err
		}
		result.Executions = append(result.Executions, execution)
		result.ResultCount += execution.ResultCount
		result.NetworkRequests += execution.Plan.NetworkRequests
		result.BrowserExecutions += execution.Plan.BrowserExecutions
		result.LiveSourceCalls += execution.Plan.LiveSourceCalls
		if execution.ProofLinked {
			result.ProofLinkedCount += execution.ResultCount
		}
		if execution.Status == "resolved" {
			resolved++
		}
		if _, exists := seenUniverses[execution.Plan.UniverseID]; !exists {
			seenUniverses[execution.Plan.UniverseID] = struct{}{}
			result.Universes = append(result.Universes, execution.Plan.UniverseID)
		}
	}
	sort.Strings(result.Universes)
	switch {
	case resolved == uint64(len(result.Executions)):
		result.Status = "resolved"
	case resolved > 0:
		result.Status = "partial"
	}
	return result, nil
}

func CuratedScenarios() []Scenario {
	scenarios := []Scenario{
		{ID: "agent-economy.capability-discovery", Title: "API capability discovery", Question: "Find publisher-declared API capabilities in the admitted Agent Economy universe.", Concepts: []string{"tw:Capability", "tw:Operation"}, UniverseID: "tw:agent-economy", Query: universesnapshot.Query{UniverseID: "tw:agent-economy", FrameType: "economy:Capability", Limit: 20}, Views: []string{"view:agent-economy/capability-graph@0.1"}, Description: "Returns unresolved until Agent Economy frames are admitted."},
		{ID: "opportunity.controlled-funding", Title: "Funding discovery", Question: "Inspect the controlled grant frame and preserve unresolved deadline semantics.", Concepts: []string{"opportunity:GrantOpportunity", "opportunity:Funder", "tw:Eligibility"}, UniverseID: "tw:opportunity", Query: universesnapshot.Query{UniverseID: "tw:opportunity", FrameType: "opportunity:GrantOpportunity", SlotRole: "opportunity:funder", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: "grants-gov:agency/TEST"}, Limit: 20}, Views: []string{"view:opportunity/deadline-timeline@0.1", "view:opportunity/eligibility-tree@0.1"}, Description: "Controlled fixture only; not a live funding claim."},
		{ID: "opportunity.source-records-nsf", Title: "NSF opportunity source records", Question: "Return a bounded sample of NSF records represented in the admitted Grants.gov bulk extract.", Concepts: []string{"opportunity:GrantOpportunity", "opportunity:Funder", "tw:Eligibility"}, UniverseID: "tw:opportunity", Query: universesnapshot.Query{UniverseID: "tw:opportunity", FrameType: "opportunity:GrantOpportunity", SlotRole: "opportunity:funder", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: "grants-gov:agency/NSF"}, Limit: 20}, Views: []string{"view:opportunity/deadline-timeline@0.1", "view:opportunity/eligibility-tree@0.1"}, Description: "Source-derived records from the 2026-08-11 Grants.gov bulk extract. A returned record is not a claim that the opportunity remains open, that an applicant is eligible, or that Grants.gov endorses TWIRX; eligibility prose is withheld in this public release."},
		{ID: "research.evidence-discovery", Title: "Research discovery", Question: "Find admitted research frames about agents, provenance and semantic infrastructure.", Concepts: []string{"tw:Publication", "tw:Evidence"}, UniverseID: "tw:research", Query: universesnapshot.Query{UniverseID: "tw:research", FrameType: "research:ResearchWork", Limit: 20}, Views: []string{"view:research/funder-network@0.1"}, Description: "Returns unresolved until research frames are admitted."},
		{ID: "security.vulnerability-query", Title: "Vulnerability query", Question: "Find admitted vulnerability frames affecting the selected infrastructure profile.", Concepts: []string{"security:Vulnerability", "security:AffectedProduct"}, UniverseID: "tw:security", Query: universesnapshot.Query{UniverseID: "tw:security", FrameType: "security:Vulnerability", Limit: 20}, Views: []string{"view:security/severity-matrix@0.1"}, Description: "Returns unresolved until security frames are admitted."},
		{ID: "world-state.controlled-development", Title: "Development comparison", Question: "Inspect the controlled Chile population observation through the World State frame.", Concepts: []string{"world:Economy", "world:IndicatorObservation", "world:Measure"}, UniverseID: "tw:world-state", Query: universesnapshot.Query{UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", SlotRole: "world:country", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: "geo:CHL"}, Limit: 20}, Views: []string{"view:world-state/time-series@0.1", "view:world-state/map@0.1"}, Description: "Controlled replay fixture only; not an E4 live-source count."},
		{ID: "world-state.source-development", Title: "World State development observations", Question: "Return the proof-linked Chile observations in the admitted World Bank source matrix.", Concepts: []string{"world:Economy", "world:IndicatorObservation", "world:Measure"}, UniverseID: "tw:world-state", Query: universesnapshot.Query{UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", SlotRole: "world:country", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: "geo:CHL"}, Limit: 20}, Views: []string{"view:world-state/time-series@0.1", "view:world-state/map@0.1"}, Description: "Source-derived observations from the admitted E4.2 World Bank matrix; values remain bound to their recorded source representation and observation time."},
	}
	sort.Slice(scenarios, func(i, j int) bool { return scenarios[i].ID < scenarios[j].ID })
	return scenarios
}

func CuratedInvestigations() []InvestigationDefinition {
	investigations := []InvestigationDefinition{
		{
			ID:          "utility.controlled-world-and-opportunity",
			Title:       "World State and Opportunity proof paths",
			Question:    "Show the controlled World State and Opportunity frames through one bounded agent investigation without inventing a relationship between them.",
			ScenarioIDs: []string{"world-state.controlled-development", "opportunity.controlled-funding"},
			Description: "Controlled-fixture execution path only; it proves multi-universe coordination, not real cross-origin utility.",
		},
		{
			ID:          "utility.source-world-and-opportunity",
			Title:       "World State and Opportunity source records",
			Question:    "Query two admitted public-source universes through one bounded agent interface while preserving their independent meanings and proof paths.",
			ScenarioIDs: []string{"world-state.source-development", "opportunity.source-records-nsf"},
			Description: "Coordinates two source-derived typed queries. It does not infer a semantic relationship between World Bank observations and Grants.gov records.",
		},
	}
	sort.Slice(investigations, func(i, j int) bool { return investigations[i].ID < investigations[j].ID })
	return investigations
}

func digestText(digest dataplane.Digest) string {
	return "sha256:" + hex.EncodeToString(digest[:])
}

func displayQuery(query universesnapshot.Query) TypedQuery {
	result := TypedQuery{UniverseID: query.UniverseID, FrameType: query.FrameType, NativeKey: query.NativeKey, SlotRole: query.SlotRole, Limit: query.Limit}
	if query.SlotValue != nil {
		value := displayValue(*query.SlotValue)
		result.SlotValue = &value
	}
	return result
}

func displayValue(value dataplane.TypedValue) Value {
	result := Value{Type: value.Type, Lexical: value.Lexical}
	if value.Unit.Present {
		result.Unit = value.Unit.Value
	}
	if value.Currency.Present {
		result.Currency = value.Currency.Value
	}
	return result
}
