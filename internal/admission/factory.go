// Package admission validates per-origin human-review artifacts and renders
// deterministic Atlas control-plane artifacts. It never fetches an origin,
// makes a policy decision, or promotes an origin automatically.
package admission

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	DecisionFormat  = "tw.admission-decision/0.1"
	PolicyFormat    = "tw.admission-policy-evidence/0.1"
	DossierFormat   = "tw.admission-dossier/0.1"
	BatchFormat     = "tw.admission-batch/0.1"
	MaxArtifact     = 1 << 20
	MaxOrigins      = atlas.RequiredCandidates
	ReviewPending   = "pending"
	ReviewCompleted = "completed"
)

type DecisionArtifact struct {
	Format            string                  `json:"format"`
	OriginID          string                  `json:"origin_id"`
	ReviewState       string                  `json:"review_state"`
	AdmitToRegistry   bool                    `json:"admit_to_registry"`
	ReviewerType      string                  `json:"reviewer_type"`
	ApprovalReference string                  `json:"approval_reference"`
	CatalogReviewedAt string                  `json:"catalog_reviewed_at"`
	CatalogReviewer   string                  `json:"catalog_reviewer"`
	PolicyReviewState atlas.PolicyReviewState `json:"policy_review_state"`
	PolicyDecision    atlas.PolicyDecision    `json:"policy_decision"`
	PolicyReviewedAt  *string                 `json:"policy_reviewed_at"`
	PolicyReviewer    *string                 `json:"policy_reviewer"`
	Rationale         string                  `json:"rationale"`
	Constraints       []string                `json:"constraints"`
	EvidenceRefs      []string                `json:"evidence_refs"`
}

type PolicyArtifact struct {
	Format    string             `json:"format"`
	Policy    atlas.OriginPolicy `json:"policy"`
	Artifacts []EvidenceArtifact `json:"artifacts"`
}

type EvidenceArtifact struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type Source struct {
	Directory      string
	Record         atlas.OriginRecord
	Policy         atlas.OriginPolicy
	Decision       DecisionArtifact
	RecordDigest   string
	PolicyDigest   string
	DecisionDigest string
}

type Dossier struct {
	Format                       string                   `json:"format"`
	OriginID                     string                   `json:"origin_id"`
	AdmissionReview              string                   `json:"admission_review_state"`
	CanonicalAdmission           bool                     `json:"canonical_admission"`
	CanonicalOrigin              string                   `json:"canonical_origin"`
	Scope                        atlas.OriginScope        `json:"scope"`
	CatalogState                 atlas.CatalogState       `json:"catalog_state"`
	PolicyReviewState            atlas.PolicyReviewState  `json:"policy_review_state"`
	PolicyDecision               atlas.PolicyDecision     `json:"policy_decision"`
	TechnicalStage               atlas.TechnicalStage     `json:"technical_stage"`
	PublisherStatus              atlas.PublisherStatus    `json:"publisher_status"`
	HealthStatus                 atlas.HealthStatus       `json:"health_status"`
	AdapterTrust                 atlas.AdapterTrustStatus `json:"adapter_trust"`
	MappingTrust                 atlas.MappingTrustStatus `json:"mapping_trust"`
	Interfaces                   int                      `json:"interfaces"`
	CapabilityCandidates         int                      `json:"capability_candidates"`
	AdmittedPublicReadOperations int                      `json:"admitted_public_read_operations"`
	AccessClass                  atlas.AccessClass        `json:"access_class"`
	CommercialOfferCandidates    int                      `json:"commercial_offer_candidates"`
	RecordDigest                 string                   `json:"record_digest"`
	PolicyDigest                 string                   `json:"policy_evidence_digest"`
	DecisionDigest               string                   `json:"decision_digest"`
	ApprovalReference            string                   `json:"approval_reference"`
	Constraints                  []string                 `json:"constraints"`
	EvidenceRefs                 []string                 `json:"evidence_refs"`
}

type Batch struct {
	Format                 string                 `json:"format"`
	Version                string                 `json:"version"`
	Statement              string                 `json:"statement"`
	Dossiers               int                    `json:"dossiers"`
	PendingHumanReview     int                    `json:"pending_human_review"`
	CanonicalAdmissions    int                    `json:"canonical_admissions"`
	TestFixtures           int                    `json:"test_fixtures"`
	Cataloged              int                    `json:"cataloged"`
	PolicyCompleted        int                    `json:"policy_completed"`
	ProfiledOrBeyond       int                    `json:"profiled_or_beyond"`
	ObservedOrBeyond       int                    `json:"observed_or_beyond"`
	Decisions              map[string]int         `json:"decisions"`
	DomainFamilies         map[string]int         `json:"domain_families"`
	CountriesOrTerritories int                    `json:"countries_or_territories"`
	Languages              int                    `json:"languages"`
	Capabilities           atlas.CapabilityCounts `json:"capabilities"`
	OriginDossiers         []Dossier              `json:"origin_dossiers"`
}

func LoadFixtures(root string) ([]atlas.OriginRecord, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("admission: read fixture root: %w", err)
	}
	if len(entries) > atlas.MaxTestFixtures {
		return nil, errors.New("admission: fixture count exceeds bound")
	}
	fixtures := make([]atlas.OriginRecord, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		data, err := readRegular(filepath.Join(root, entry.Name(), "record.json"))
		if err != nil {
			return nil, fmt.Errorf("admission: fixture %s: %w", entry.Name(), err)
		}
		var record atlas.OriginRecord
		if err := decode(data, &record); err != nil {
			return nil, fmt.Errorf("admission: fixture %s: %w", entry.Name(), err)
		}
		if record.ID != entry.Name() || record.Scope != atlas.ScopeTestFixture || record.Policy.PolicyID != nil || record.Policy.PolicySetDigest != nil {
			return nil, fmt.Errorf("admission: fixture %s changes fixture identity or binds public policy", entry.Name())
		}
		fixtures = append(fixtures, record)
	}
	sort.Slice(fixtures, func(i, j int) bool { return fixtures[i].ID < fixtures[j].ID })
	return fixtures, nil
}

func Load(root string, selection *atlas.Selection) ([]Source, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("admission: read origins root: %w", err)
	}
	if len(entries) == 0 || len(entries) > MaxOrigins {
		return nil, errors.New("admission: origin-directory count outside bounds")
	}
	sources := make([]Source, 0, len(entries))
	seenIDs := make(map[string]string)
	seenOrigins := make(map[string]string)
	seenAliases := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		directory := filepath.Join(root, entry.Name())
		recordBytes, err := readRegular(filepath.Join(directory, "record.json"))
		if err != nil {
			return nil, fmt.Errorf("admission: %s record: %w", entry.Name(), err)
		}
		policyBytes, err := readRegular(filepath.Join(directory, "policy-evidence.json"))
		if err != nil {
			return nil, fmt.Errorf("admission: %s policy: %w", entry.Name(), err)
		}
		decisionBytes, err := readRegular(filepath.Join(directory, "decision.json"))
		if err != nil {
			return nil, fmt.Errorf("admission: %s decision: %w", entry.Name(), err)
		}
		var record atlas.OriginRecord
		var policy PolicyArtifact
		var decision DecisionArtifact
		if err := decode(recordBytes, &record); err != nil {
			return nil, fmt.Errorf("admission: %s record: %w", entry.Name(), err)
		}
		if err := decode(policyBytes, &policy); err != nil {
			return nil, fmt.Errorf("admission: %s policy: %w", entry.Name(), err)
		}
		if err := decode(decisionBytes, &decision); err != nil {
			return nil, fmt.Errorf("admission: %s decision: %w", entry.Name(), err)
		}
		if entry.Name() != record.ID || policy.Format != PolicyFormat || decision.Format != DecisionFormat || policy.Policy.OriginID != record.ID || decision.OriginID != record.ID {
			return nil, fmt.Errorf("admission: %s directory identities disagree", entry.Name())
		}
		if err := validateEvidenceArtifacts(directory, policy); err != nil {
			return nil, fmt.Errorf("admission: %s policy evidence: %w", entry.Name(), err)
		}
		if record.Scope != atlas.ScopeGenesisPublic {
			return nil, fmt.Errorf("admission: %s is not a Genesis public origin", entry.Name())
		}
		candidate, err := selection.Find(record.ID)
		if err != nil || candidate.CanonicalOrigin != record.CanonicalOrigin || policy.Policy.CanonicalOrigin != record.CanonicalOrigin {
			return nil, fmt.Errorf("admission: %s changes selected identity", entry.Name())
		}
		if previous, exists := seenOrigins[record.CanonicalOrigin]; exists {
			return nil, fmt.Errorf("admission: duplicate canonical origin %s in %s and %s", record.CanonicalOrigin, previous, entry.Name())
		}
		if previous, exists := seenIDs[record.ID]; exists {
			return nil, fmt.Errorf("admission: duplicate origin ID %s in %s and %s", record.ID, previous, entry.Name())
		}
		seenIDs[record.ID] = entry.Name()
		seenOrigins[record.CanonicalOrigin] = entry.Name()
		for _, alias := range record.ExecutionCatalogIDs {
			if previous, exists := seenIDs[alias]; exists {
				return nil, fmt.Errorf("admission: alias %s in %s collides with origin ID in %s", alias, entry.Name(), previous)
			}
			if previous, exists := seenAliases[alias]; exists {
				return nil, fmt.Errorf("admission: duplicate alias %s in %s and %s", alias, previous, entry.Name())
			}
			seenAliases[alias] = entry.Name()
		}
		if previous, exists := seenAliases[record.ID]; exists {
			return nil, fmt.Errorf("admission: origin ID %s in %s collides with alias in %s", record.ID, entry.Name(), previous)
		}
		if err := validateDecision(record, policy.Policy, decision, selection); err != nil {
			return nil, fmt.Errorf("admission: %s: %w", entry.Name(), err)
		}
		sources = append(sources, Source{Directory: directory, Record: record, Policy: policy.Policy, Decision: decision, RecordDigest: digest(recordBytes), PolicyDigest: digest(policyBytes), DecisionDigest: digest(decisionBytes)})
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Record.ID < sources[j].Record.ID })
	if len(sources) == 0 {
		return nil, errors.New("admission: no origin directories found")
	}
	return sources, nil
}

func Render(sources []Source, fixtures []atlas.OriginRecord, selection *atlas.Selection, version string) (*atlas.PolicySet, []byte, *atlas.Registry, []byte, Batch, error) {
	if len(sources) == 0 {
		return nil, nil, nil, nil, Batch{}, errors.New("admission: no sources")
	}
	policies := make([]atlas.OriginPolicy, 0, len(sources))
	for _, source := range sources {
		if source.Decision.ReviewState == ReviewCompleted && source.Decision.AdmitToRegistry {
			policies = append(policies, source.Policy)
		}
	}
	policySet, policyBytes, err := atlas.BuildPolicySet(version, "Rendered from explicit per-origin E3.2 policy evidence and human decision artifacts. Rendering does not create or approve a decision.", policies, selection)
	if err != nil {
		return nil, nil, nil, nil, Batch{}, err
	}
	policyDigest := policySet.DigestReference()
	records := make([]atlas.OriginRecord, 0, len(policies)+len(fixtures))
	for _, source := range sources {
		if source.Decision.ReviewState != ReviewCompleted || !source.Decision.AdmitToRegistry {
			continue
		}
		record := source.Record
		record.Policy.PolicySetDigest = &policyDigest
		records = append(records, record)
	}
	records = append(records, fixtures...)
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	registry := &atlas.Registry{Format: atlas.RegistryFormat, Version: version, Origins: records}
	if err := registry.Validate(selection, policySet); err != nil {
		return nil, nil, nil, nil, Batch{}, err
	}
	registryBytes, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return nil, nil, nil, nil, Batch{}, err
	}
	registryBytes = append(registryBytes, '\n')
	batch := buildBatch(sources, version)
	batch.TestFixtures = len(fixtures)
	return policySet, policyBytes, registry, registryBytes, batch, nil
}

func buildBatch(sources []Source, version string) Batch {
	batch := Batch{Format: BatchFormat, Version: version, Statement: "Counts describe explicit per-origin pilot artifacts; no decision, execution authority, or canonical admission is generated automatically.", Decisions: make(map[string]int), DomainFamilies: make(map[string]int), Capabilities: atlas.NewCapabilityCounts(0), OriginDossiers: make([]Dossier, 0, len(sources))}
	countries := make(map[string]struct{})
	languages := make(map[string]struct{})
	for _, source := range sources {
		batch.Capabilities.Add(&source.Record, false)
		batch.Dossiers++
		if source.Decision.ReviewState == ReviewPending {
			batch.PendingHumanReview++
		} else if source.Decision.AdmitToRegistry {
			batch.CanonicalAdmissions++
		}
		if source.Record.Catalog.State == atlas.CatalogCataloged {
			batch.Cataloged++
		}
		if source.Policy.ReviewState == atlas.PolicyCompleted {
			batch.PolicyCompleted++
		}
		if source.Record.Technical.Stage.Index() >= atlas.TechnicalProfiled.Index() {
			batch.ProfiledOrBeyond++
		}
		if source.Record.Technical.Stage.Index() >= atlas.TechnicalObserved.Index() {
			batch.ObservedOrBeyond++
		}
		if source.Policy.ReviewState == atlas.PolicyCompleted {
			batch.Decisions[string(source.Policy.Decision)]++
		}
		for _, family := range source.Record.DomainFamilies {
			batch.DomainFamilies[family]++
		}
		if source.Record.Jurisdiction != "not_established" {
			countries[source.Record.Jurisdiction] = struct{}{}
		}
		for _, language := range source.Record.Languages {
			languages[language] = struct{}{}
		}
		admitted := 0
		for _, capability := range source.Record.Capabilities {
			if capability.Status == atlas.CapabilityAdmitted && capability.EffectClass == atlas.EffectPublicRead {
				admitted++
			}
		}
		batch.OriginDossiers = append(batch.OriginDossiers, Dossier{Format: DossierFormat, OriginID: source.Record.ID, AdmissionReview: source.Decision.ReviewState, CanonicalAdmission: source.Decision.ReviewState == ReviewCompleted && source.Decision.AdmitToRegistry, CanonicalOrigin: source.Record.CanonicalOrigin, Scope: source.Record.Scope, CatalogState: source.Record.Catalog.State, PolicyReviewState: source.Policy.ReviewState, PolicyDecision: source.Policy.Decision, TechnicalStage: source.Record.Technical.Stage, PublisherStatus: source.Record.Publisher.Status, HealthStatus: source.Record.Health.Status, AdapterTrust: source.Record.AdapterTrust.Status, MappingTrust: source.Record.MappingTrust.Status, Interfaces: len(source.Record.Interfaces), CapabilityCandidates: len(source.Record.Capabilities), AdmittedPublicReadOperations: admitted, AccessClass: source.Record.Access.Class, CommercialOfferCandidates: len(source.Record.Access.OfferCandidates), RecordDigest: source.RecordDigest, PolicyDigest: source.PolicyDigest, DecisionDigest: source.DecisionDigest, ApprovalReference: source.Decision.ApprovalReference, Constraints: append([]string(nil), source.Decision.Constraints...), EvidenceRefs: append([]string(nil), source.Decision.EvidenceRefs...)})
	}
	batch.CountriesOrTerritories = len(countries)
	batch.Languages = len(languages)
	return batch
}

func validateDecision(record atlas.OriginRecord, policy atlas.OriginPolicy, decision DecisionArtifact, selection *atlas.Selection) error {
	if decision.Rationale == "" || len(decision.EvidenceRefs) == 0 || !sortedUnique(decision.EvidenceRefs) || !sortedUnique(decision.Constraints) {
		return errors.New("decision rationale, constraints, and evidence are required")
	}
	if decision.CatalogReviewedAt != dereference(record.Catalog.ReviewedAt) || decision.CatalogReviewer != dereference(record.Catalog.Reviewer) || decision.PolicyReviewState != policy.ReviewState || decision.PolicyDecision != policy.Decision {
		return errors.New("decision artifact disagrees with catalog or policy state")
	}
	switch decision.ReviewState {
	case ReviewPending:
		if decision.AdmitToRegistry || decision.ReviewerType != "none" || decision.ApprovalReference != "" || decision.PolicyReviewedAt != nil || decision.PolicyReviewer != nil || policy.ReviewState != atlas.PolicyPending || policy.Decision != atlas.DecisionUncertain {
			return errors.New("pending proposal cannot claim admission, human approval, or completed policy review")
		}
	case ReviewCompleted:
		if !decision.AdmitToRegistry || decision.ReviewerType != "human" || decision.ApprovalReference == "" {
			return errors.New("completed admission requires explicit human approval")
		}
		if policy.ReviewState == atlas.PolicyPending {
			if policy.Decision != atlas.DecisionUncertain || decision.PolicyReviewedAt != nil || decision.PolicyReviewer != nil || policy.ReviewedAt != nil || policy.Reviewer != nil {
				return errors.New("human catalog admission cannot imply a completed policy review")
			}
		} else if policy.ReviewState == atlas.PolicyCompleted {
			if decision.PolicyReviewedAt == nil || decision.PolicyReviewer == nil || policy.ReviewedAt == nil || policy.Reviewer == nil || *decision.PolicyReviewedAt != *policy.ReviewedAt || *decision.PolicyReviewer != *policy.Reviewer {
				return errors.New("completed policy decision must match its explicit human review")
			}
		} else {
			return errors.New("invalid policy review state")
		}
	default:
		return errors.New("invalid admission review state")
	}
	if record.Policy.PolicySetDigest != nil || record.Policy.PolicyID == nil || *record.Policy.PolicyID != policy.ID || record.Policy.ReviewState != policy.ReviewState || record.Policy.Decision != policy.Decision || record.Policy.Attribution != policy.Attribution || record.Policy.Authentication != policy.Authentication || record.Policy.RatePolicy != policy.RatePolicy || record.Policy.RetentionPolicy != policy.RetentionPolicy || record.Policy.TermsReference != policy.TermsReference || record.Policy.RiskState != policy.RiskState || record.Policy.ReviewerNotes != policy.ReviewerNotes {
		return errors.New("source record policy does not match per-origin policy evidence or contains generated digest")
	}
	policySet, _, err := atlas.BuildPolicySet("source-validation", "Validates one per-origin source artifact before deterministic batch rendering.", []atlas.OriginPolicy{policy}, selection)
	if err == nil {
		copyRecord := record
		digest := policySet.DigestReference()
		copyRecord.Policy.PolicySetDigest = &digest
		registry := atlas.Registry{Format: atlas.RegistryFormat, Version: "source-validation", Origins: []atlas.OriginRecord{copyRecord}}
		err = registry.Validate(selection, policySet)
	}
	if err != nil {
		return fmt.Errorf("source record does not satisfy canonical state validation: %w", err)
	}
	return nil
}

func Marshal(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decode(data []byte, value any) error {
	return jsonbounded.Decode(data, value, jsonbounded.Policy{MaxBytes: MaxArtifact, MaxDepth: 32, MaxScalarBytes: 64 << 10, MaxContainerEntries: 4096, MaxTokens: 100000}, true)
}

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > MaxArtifact {
		return nil, errors.New("artifact is not a bounded regular file")
	}
	return os.ReadFile(path)
}

func validateEvidenceArtifacts(directory string, artifact PolicyArtifact) error {
	if len(artifact.Artifacts) == 0 || len(artifact.Artifacts) > 64 || len(artifact.Policy.EvidenceRefs) != len(artifact.Artifacts) {
		return errors.New("bounded policy evidence artifacts are required")
	}
	previous := ""
	for index, evidence := range artifact.Artifacts {
		if evidence.Path <= previous || !safeEvidencePath(evidence.Path) || !validDigest(evidence.Digest) || artifact.Policy.EvidenceRefs[index] != evidence.Path {
			return errors.New("policy evidence must be sorted, unique, safe, digest-bound, and match policy references")
		}
		data, err := readRegular(filepath.Join(directory, filepath.FromSlash(evidence.Path)))
		if err != nil {
			return err
		}
		if digest(data) != evidence.Digest {
			return fmt.Errorf("evidence digest mismatch for %s", evidence.Path)
		}
		previous = evidence.Path
	}
	return nil
}

func safeEvidencePath(value string) bool {
	converted := filepath.FromSlash(value)
	return strings.HasPrefix(value, "evidence/") && !filepath.IsAbs(converted) && filepath.Clean(converted) == converted && !strings.Contains(value, "\\") && !strings.Contains(value, "..")
}

func validDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func dereference(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sortedUnique(values []string) bool {
	if !sort.StringsAreSorted(values) {
		return false
	}
	for index, value := range values {
		if value == "" || index > 0 && value == values[index-1] {
			return false
		}
	}
	return true
}
