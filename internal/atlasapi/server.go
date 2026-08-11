// Package atlasapi exposes the offline, read-only E3 Atlas control plane.
// It has no network client and cannot invoke an origin.
package atlasapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/atlas"
)

const (
	maxLimit     = 100
	defaultLimit = 50
)

type Server struct {
	Selection  *atlas.Selection
	Registry   *atlas.Registry
	Metrics    atlas.Metrics
	records    map[string]*atlas.OriginRecord
	candidates []atlas.Candidate
}

type OriginView struct {
	ID                  string                       `json:"id"`
	Scope               atlas.OriginScope            `json:"scope"`
	CanonicalOrigin     string                       `json:"canonical_origin"`
	DomainFamily        string                       `json:"domain_family"`
	ExecutionCatalogIDs []string                     `json:"execution_catalog_ids"`
	Catalog             CatalogView                  `json:"catalog"`
	Policy              PolicyView                   `json:"policy"`
	Technical           TechnicalView                `json:"technical"`
	Publisher           PublisherView                `json:"publisher"`
	Health              HealthView                   `json:"health"`
	AdapterTrust        AdapterTrustView             `json:"adapter_trust"`
	MappingTrust        MappingTrustView             `json:"mapping_trust"`
	Interfaces          []atlas.InterfaceDeclaration `json:"interfaces"`
	Capabilities        []atlas.CapabilityCandidate  `json:"capabilities"`
	Access              atlas.AccessMetadata         `json:"access"`
	Economics           atlas.EconomicsMetadata      `json:"economics"`
	PublisherReadiness  atlas.PublisherReadiness     `json:"publisher_readiness"`
	Jurisdiction        *string                      `json:"jurisdiction"`
	Languages           []string                     `json:"languages"`
	AuthorityClass      *string                      `json:"authority_class"`
	Statement           string                       `json:"statement"`
}

type CatalogView struct {
	State atlas.CatalogState `json:"state"`
}

type PolicyView struct {
	ReviewState atlas.PolicyReviewState `json:"review_state"`
	Decision    atlas.PolicyDecision    `json:"decision"`
}

type TechnicalView struct {
	Stage atlas.TechnicalStage `json:"stage"`
}

type PublisherView struct {
	Status atlas.PublisherStatus `json:"status"`
	Name   *string               `json:"name"`
	Kind   *string               `json:"kind"`
}

type HealthView struct {
	Status atlas.HealthStatus `json:"status"`
}

type AdapterTrustView struct {
	Status atlas.AdapterTrustStatus `json:"status"`
}

type MappingTrustView struct {
	Status atlas.MappingTrustStatus `json:"status"`
}

func New(selection *atlas.Selection, registry *atlas.Registry, policies *atlas.PolicySet) (*Server, error) {
	metrics, err := atlas.BuildMetrics(selection, registry, policies)
	if err != nil {
		return nil, err
	}
	server := &Server{Selection: selection, Registry: registry, Metrics: metrics, records: make(map[string]*atlas.OriginRecord), candidates: append([]atlas.Candidate(nil), selection.Candidates...)}
	for index := range registry.Origins {
		if registry.Origins[index].Scope == atlas.ScopeGenesisPublic {
			server.records[registry.Origins[index].ID] = &registry.Origins[index]
		}
	}
	sort.Slice(server.candidates, func(i, j int) bool { return server.candidates[i].ID < server.candidates[j].ID })
	return server, nil
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", "GET")
		s.writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Atlas control plane is read-only")
		return
	}
	switch {
	case request.URL.Path == "/api/v1/atlas/status" || request.URL.Path == "/api/v1/atlas/metrics":
		s.writeJSON(writer, http.StatusOK, s.Metrics)
	case request.URL.Path == "/api/v1/atlas/origins":
		s.listOrigins(writer, request)
	case strings.HasPrefix(request.URL.Path, "/api/v1/atlas/origins/"):
		s.describeOrigin(writer, strings.TrimPrefix(request.URL.Path, "/api/v1/atlas/origins/"))
	default:
		s.writeError(writer, http.StatusNotFound, "not_found", "route not found")
	}
}

func (s *Server) listOrigins(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	allowed := map[string]bool{
		"family": true, "catalog_state": true, "policy_review_state": true,
		"policy_decision": true, "technical_stage": true, "publisher_status": true,
		"health_status": true, "adapter_trust": true, "mapping_trust": true,
		"limit": true, "cursor": true,
	}
	for key, values := range query {
		if !allowed[key] || len(values) != 1 {
			s.writeError(writer, http.StatusBadRequest, "invalid_query", "unsupported, duplicate, or multi-valued query parameter")
			return
		}
	}
	if family := query.Get("family"); family != "" {
		if _, exists := atlas.RequiredFamilyQuotas[family]; !exists {
			s.writeError(writer, http.StatusBadRequest, "invalid_family", "unknown domain family")
			return
		}
	}
	if err := validateStateFilters(query); err != nil {
		s.writeError(writer, http.StatusBadRequest, "invalid_state", err.Error())
		return
	}
	limit, err := boundedInteger(query.Get("limit"), defaultLimit, 1, maxLimit)
	if err != nil {
		s.writeError(writer, http.StatusBadRequest, "invalid_limit", err.Error())
		return
	}
	cursor, err := boundedInteger(query.Get("cursor"), 0, 0, atlas.RequiredCandidates)
	if err != nil {
		s.writeError(writer, http.StatusBadRequest, "invalid_cursor", err.Error())
		return
	}
	matched := make([]OriginView, 0, limit)
	total := 0
	for _, candidate := range s.candidates {
		view := s.view(candidate)
		if !matches(query, view) {
			continue
		}
		if total >= cursor && len(matched) < limit {
			matched = append(matched, view)
		}
		total++
	}
	var next *int
	if cursor+len(matched) < total {
		value := cursor + len(matched)
		next = &value
	}
	s.writeJSON(writer, http.StatusOK, map[string]any{
		"origins": matched, "total": total, "cursor": cursor, "next_cursor": next,
		"statement": "State dimensions are independent. Candidate selection is not network authority, and pending policy review is not completed assessment.",
	})
}

func validateStateFilters(query mapValues) error {
	checks := []struct {
		key     string
		allowed map[string]bool
	}{
		{"catalog_state", values("candidate", "cataloged")},
		{"policy_review_state", values("pending", "completed")},
		{"policy_decision", values("permit_live", "permit_with_constraints", "profile_only", "catalog_only", "deny", "uncertain")},
		{"technical_stage", values("unprofiled", "profiled", "observed", "native_schema", "compiled", "semantically_linked", "live")},
		{"publisher_status", values("unclaimed", "domain_verified", "publisher_approved", "publisher_signed")},
		{"health_status", values("unknown", "healthy", "degraded", "stale", "suspended", "revoked")},
		{"adapter_trust", values("none", "candidate", "reviewed", "conformant", "revoked")},
		{"mapping_trust", values("none", "candidate", "reviewed", "disputed", "revoked")},
	}
	for _, check := range checks {
		if value := query.Get(check.key); value != "" && !check.allowed[value] {
			return fmt.Errorf("invalid %s", check.key)
		}
	}
	return nil
}

// mapValues is the subset of url.Values used by filter validation.
type mapValues interface {
	Get(string) string
}

func values(items ...string) map[string]bool {
	result := make(map[string]bool, len(items))
	for _, item := range items {
		result[item] = true
	}
	return result
}

func matches(query mapValues, view OriginView) bool {
	return (query.Get("family") == "" || query.Get("family") == view.DomainFamily) &&
		(query.Get("catalog_state") == "" || query.Get("catalog_state") == string(view.Catalog.State)) &&
		(query.Get("policy_review_state") == "" || query.Get("policy_review_state") == string(view.Policy.ReviewState)) &&
		(query.Get("policy_decision") == "" || query.Get("policy_decision") == string(view.Policy.Decision)) &&
		(query.Get("technical_stage") == "" || query.Get("technical_stage") == string(view.Technical.Stage)) &&
		(query.Get("publisher_status") == "" || query.Get("publisher_status") == string(view.Publisher.Status)) &&
		(query.Get("health_status") == "" || query.Get("health_status") == string(view.Health.Status)) &&
		(query.Get("adapter_trust") == "" || query.Get("adapter_trust") == string(view.AdapterTrust.Status)) &&
		(query.Get("mapping_trust") == "" || query.Get("mapping_trust") == string(view.MappingTrust.Status))
}

func (s *Server) describeOrigin(writer http.ResponseWriter, id string) {
	if id == "" || strings.Contains(id, "/") {
		s.writeError(writer, http.StatusNotFound, "not_found", "origin not found")
		return
	}
	candidate, err := s.Selection.Find(id)
	if err != nil {
		s.writeError(writer, http.StatusNotFound, "not_found", "origin not found")
		return
	}
	s.writeJSON(writer, http.StatusOK, s.view(*candidate))
}

func (s *Server) view(candidate atlas.Candidate) OriginView {
	view := OriginView{
		ID: candidate.ID, Scope: atlas.ScopeGenesisPublic, CanonicalOrigin: candidate.CanonicalOrigin, DomainFamily: candidate.DomainFamily,
		ExecutionCatalogIDs: []string{}, Catalog: CatalogView{State: atlas.CatalogCandidate},
		Policy:    PolicyView{ReviewState: atlas.PolicyPending, Decision: atlas.DecisionUncertain},
		Technical: TechnicalView{Stage: atlas.TechnicalUnprofiled}, Publisher: PublisherView{Status: atlas.PublisherUnclaimed},
		Health: HealthView{Status: atlas.HealthUnknown}, AdapterTrust: AdapterTrustView{Status: atlas.AdapterTrustNone}, MappingTrust: MappingTrustView{Status: atlas.MappingTrustNone},
		Interfaces: []atlas.InterfaceDeclaration{}, Capabilities: []atlas.CapabilityCandidate{},
		Access:    atlas.AccessMetadata{Class: atlas.AccessUnknown, AssessmentStatus: atlas.AccessNotAssessed, LicenseOrTermsRefs: []string{}, Attribution: "not_assessed", RatePolicy: "not_assessed", PaymentProtocolCandidates: []string{}, PriceDiscoveryStatus: atlas.PriceNotAssessed, OfferCandidates: []atlas.OfferCandidate{}},
		Economics: atlas.EconomicsMetadata{FundingClass: atlas.FundingUnknown}, PublisherReadiness: atlas.PublisherReadiness{Signals: []atlas.PublisherReadinessSignal{}},
		Languages: []string{}, Statement: "Selected candidate only; no completed policy, technical, publisher, health, adapter-trust, mapping-trust, or live claim.",
	}
	if record, exists := s.records[candidate.ID]; exists {
		publisherName, publisherKind := record.Publisher.Name, record.Publisher.Kind
		jurisdiction, authority := record.Jurisdiction, record.AuthorityClass
		view.ExecutionCatalogIDs = append([]string(nil), record.ExecutionCatalogIDs...)
		view.Catalog.State = record.Catalog.State
		view.Policy = PolicyView{ReviewState: record.Policy.ReviewState, Decision: record.Policy.Decision}
		view.Technical.Stage = record.Technical.Stage
		view.Publisher = PublisherView{Status: record.Publisher.Status, Name: &publisherName, Kind: &publisherKind}
		view.Health.Status = record.Health.Status
		view.AdapterTrust.Status = record.AdapterTrust.Status
		view.MappingTrust.Status = record.MappingTrust.Status
		view.Interfaces = append([]atlas.InterfaceDeclaration(nil), record.Interfaces...)
		view.Capabilities = append([]atlas.CapabilityCandidate(nil), record.Capabilities...)
		view.Access = record.Access
		view.Access.LicenseOrTermsRefs = append([]string(nil), record.Access.LicenseOrTermsRefs...)
		view.Access.PaymentProtocolCandidates = append([]string(nil), record.Access.PaymentProtocolCandidates...)
		view.Access.OfferCandidates = append([]atlas.OfferCandidate(nil), record.Access.OfferCandidates...)
		view.Economics = record.Economics
		view.PublisherReadiness.Signals = append([]atlas.PublisherReadinessSignal(nil), record.PublisherReadiness.Signals...)
		view.Jurisdiction, view.AuthorityClass = &jurisdiction, &authority
		view.Languages = append([]string(nil), record.Languages...)
		view.Statement = "Each state is supported and counted independently; provider content is represented, not promoted to objective truth."
	}
	return view
}

func boundedInteger(raw string, fallback, minimum, maximum int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("value must be between %d and %d", minimum, maximum)
	}
	return value, nil
}

func (s *Server) writeError(writer http.ResponseWriter, status int, code, detail string) {
	s.writeJSON(writer, status, map[string]any{"error": map[string]string{"code": code, "detail": detail}})
}

func (s *Server) writeJSON(writer http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(writer, `{"error":{"code":"internal","detail":"encoding failed"}}`, http.StatusInternalServerError)
		return
	}
	if len(data) > 1<<20 {
		http.Error(writer, `{"error":{"code":"response_too_large","detail":"response exceeded bound"}}`, http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(status)
	_, _ = writer.Write(append(data, '\n'))
}

var _ http.Handler = (*Server)(nil)
