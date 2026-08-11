package atlas

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	RegistryFormat   = "tw.origin-registry/0.3"
	MaxRegistryBytes = 8 << 20
	MaxTestFixtures  = 64
)

type OriginScope string

const (
	ScopeGenesisPublic OriginScope = "genesis_public"
	ScopeTestFixture   OriginScope = "test_fixture"
)

type CatalogState string

const (
	CatalogCandidate CatalogState = "candidate"
	CatalogCataloged CatalogState = "cataloged"
)

type PolicyReviewState string

const (
	PolicyPending   PolicyReviewState = "pending"
	PolicyCompleted PolicyReviewState = "completed"
)

type PolicyDecision string

const (
	DecisionPermitLive            PolicyDecision = "permit_live"
	DecisionPermitWithConstraints PolicyDecision = "permit_with_constraints"
	DecisionProfileOnly           PolicyDecision = "profile_only"
	DecisionCatalogOnly           PolicyDecision = "catalog_only"
	DecisionDeny                  PolicyDecision = "deny"
	DecisionUncertain             PolicyDecision = "uncertain"
)

type TechnicalStage string

const (
	TechnicalUnprofiled         TechnicalStage = "unprofiled"
	TechnicalProfiled           TechnicalStage = "profiled"
	TechnicalObserved           TechnicalStage = "observed"
	TechnicalNativeSchema       TechnicalStage = "native_schema"
	TechnicalCompiled           TechnicalStage = "compiled"
	TechnicalSemanticallyLinked TechnicalStage = "semantically_linked"
	TechnicalLive               TechnicalStage = "live"
)

var technicalOrder = []TechnicalStage{
	TechnicalUnprofiled,
	TechnicalProfiled,
	TechnicalObserved,
	TechnicalNativeSchema,
	TechnicalCompiled,
	TechnicalSemanticallyLinked,
	TechnicalLive,
}

func (s TechnicalStage) Index() int {
	for index, candidate := range technicalOrder {
		if candidate == s {
			return index
		}
	}
	return -1
}

func TechnicalStages() []TechnicalStage {
	return append([]TechnicalStage(nil), technicalOrder...)
}

type PublisherStatus string

const (
	PublisherUnclaimed      PublisherStatus = "unclaimed"
	PublisherDomainVerified PublisherStatus = "domain_verified"
	PublisherApproved       PublisherStatus = "publisher_approved"
	PublisherSigned         PublisherStatus = "publisher_signed"
)

type HealthStatus string

const (
	HealthUnknown   HealthStatus = "unknown"
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthStale     HealthStatus = "stale"
	HealthSuspended HealthStatus = "suspended"
	HealthRevoked   HealthStatus = "revoked"
)

type AdapterTrustStatus string

const (
	AdapterTrustNone       AdapterTrustStatus = "none"
	AdapterTrustCandidate  AdapterTrustStatus = "candidate"
	AdapterTrustReviewed   AdapterTrustStatus = "reviewed"
	AdapterTrustConformant AdapterTrustStatus = "conformant"
	AdapterTrustRevoked    AdapterTrustStatus = "revoked"
)

type MappingTrustStatus string

const (
	MappingTrustNone      MappingTrustStatus = "none"
	MappingTrustCandidate MappingTrustStatus = "candidate"
	MappingTrustReviewed  MappingTrustStatus = "reviewed"
	MappingTrustDisputed  MappingTrustStatus = "disputed"
	MappingTrustRevoked   MappingTrustStatus = "revoked"
)

type Registry struct {
	Format  string         `json:"format"`
	Version string         `json:"version"`
	Origins []OriginRecord `json:"origins"`
}

type OriginRecord struct {
	ID                  string                 `json:"id"`
	Scope               OriginScope            `json:"scope"`
	CanonicalOrigin     string                 `json:"canonical_origin"`
	RegistrableDomain   string                 `json:"registrable_domain"`
	ExecutionCatalogIDs []string               `json:"execution_catalog_ids"`
	Catalog             CatalogDimension       `json:"catalog"`
	Policy              PolicyDimension        `json:"policy"`
	Technical           TechnicalDimension     `json:"technical"`
	Publisher           Publisher              `json:"publisher"`
	Health              HealthDimension        `json:"health"`
	AdapterTrust        AdapterTrustDimension  `json:"adapter_trust"`
	MappingTrust        MappingTrustDimension  `json:"mapping_trust"`
	Jurisdiction        string                 `json:"jurisdiction"`
	Languages           []string               `json:"languages"`
	DomainFamilies      []string               `json:"domain_families"`
	AuthorityClass      string                 `json:"authority_class"`
	Discovery           Discovery              `json:"discovery"`
	Interfaces          []InterfaceDeclaration `json:"interfaces"`
	Capabilities        []CapabilityCandidate  `json:"capabilities"`
	Access              AccessMetadata         `json:"access"`
	Economics           EconomicsMetadata      `json:"economics"`
	PublisherReadiness  PublisherReadiness     `json:"publisher_readiness"`
	Runtime             RuntimeRecord          `json:"runtime"`
	Semantics           SemanticRecord         `json:"semantics"`
	Evidence            EvidenceRecord         `json:"evidence"`
}

type CatalogDimension struct {
	State        CatalogState `json:"state"`
	ReviewedAt   *string      `json:"reviewed_at"`
	Reviewer     *string      `json:"reviewer"`
	EvidenceRefs []string     `json:"evidence_refs"`
}

type PolicyDimension struct {
	ReviewState     PolicyReviewState `json:"review_state"`
	Decision        PolicyDecision    `json:"decision"`
	ReviewedAt      *string           `json:"reviewed_at"`
	Reviewer        *string           `json:"reviewer"`
	PolicyID        *string           `json:"policy_id"`
	PolicySetDigest *string           `json:"policy_set_digest"`
	RobotsDigest    *string           `json:"robots_digest"`
	Attribution     string            `json:"attribution"`
	Authentication  string            `json:"authentication"`
	RatePolicy      string            `json:"rate_policy"`
	RetentionPolicy string            `json:"retention_policy"`
	TermsReference  string            `json:"terms_reference"`
	RiskState       string            `json:"risk_state"`
	ReviewerNotes   string            `json:"reviewer_notes"`
}

type TechnicalDimension struct {
	Stage        TechnicalStage `json:"stage"`
	ReviewedAt   *string        `json:"reviewed_at"`
	Reviewer     *string        `json:"reviewer"`
	EvidenceRefs []string       `json:"evidence_refs"`
}

type Publisher struct {
	Name         string          `json:"name"`
	Kind         string          `json:"kind"`
	Status       PublisherStatus `json:"status"`
	ReviewedAt   *string         `json:"reviewed_at"`
	Reviewer     *string         `json:"reviewer"`
	EvidenceRefs []string        `json:"evidence_refs"`
	ManifestURL  *string         `json:"manifest_url"`
}

type HealthDimension struct {
	Status       HealthStatus `json:"status"`
	CheckedAt    *string      `json:"checked_at"`
	EvidenceRefs []string     `json:"evidence_refs"`
}

type AdapterTrustDimension struct {
	Status       AdapterTrustStatus `json:"status"`
	ReviewedAt   *string            `json:"reviewed_at"`
	Reviewer     *string            `json:"reviewer"`
	EvidenceRefs []string           `json:"evidence_refs"`
}

type MappingTrustDimension struct {
	Status       MappingTrustStatus `json:"status"`
	ReviewedAt   *string            `json:"reviewed_at"`
	Reviewer     *string            `json:"reviewer"`
	EvidenceRefs []string           `json:"evidence_refs"`
}

type Discovery struct {
	Sources         []string `json:"sources"`
	ProfiledAt      *string  `json:"profiled_at"`
	RobotsURL       *string  `json:"robots_url"`
	SitemapURLs     []string `json:"sitemap_urls"`
	FeedURLs        []string `json:"feed_urls"`
	APIDescriptions []string `json:"api_descriptions"`
	StructuredData  []string `json:"structured_data"`
	PublicEndpoints []string `json:"public_endpoints"`
}

type RuntimeRecord struct {
	LastSeen          *string        `json:"last_seen"`
	LastObserved      *string        `json:"last_observed"`
	RefreshClass      string         `json:"refresh_class"`
	RequestBudget     int            `json:"request_budget"`
	StorageBudgetByte int64          `json:"storage_budget_bytes"`
	Scheduler         SchedulerState `json:"scheduler"`
}

type SemanticRecord struct {
	NativeModule           *string  `json:"native_module"`
	AdapterID              *string  `json:"adapter_id"`
	ConformanceRef         *string  `json:"conformance_ref"`
	ContractSetDigest      *string  `json:"contract_set_digest"`
	MappingModules         []string `json:"mapping_modules"`
	ConceptIDs             []string `json:"concept_ids"`
	OperationIDs           []string `json:"operation_ids"`
	SemanticClosureDigests []string `json:"semantic_closure_digests"`
}

type EvidenceRecord struct {
	ObservationIDs []string            `json:"observation_ids"`
	Artifacts      []ArtifactReference `json:"artifacts"`
}

type ArtifactReference struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

func LoadRegistry(path string, selection *Selection, policies *PolicySet) (*Registry, error) {
	data, err := readBoundedRegular(path, MaxRegistryBytes)
	if err != nil {
		return nil, fmt.Errorf("atlas: read registry: %w", err)
	}
	var registry Registry
	if err := decodeStrict(data, &registry, jsonPolicy(MaxRegistryBytes)); err != nil {
		return nil, fmt.Errorf("atlas: decode registry: %w", err)
	}
	if err := registry.Validate(selection, policies); err != nil {
		return nil, err
	}
	if err := registry.ValidateArtifactFiles(filepath.Dir(filepath.Dir(path))); err != nil {
		return nil, err
	}
	return &registry, nil
}

func (r *Registry) Validate(selection *Selection, policies *PolicySet) error {
	if r == nil || r.Format != RegistryFormat || !validText(r.Version, 128) || selection == nil || policies == nil {
		return errors.New("atlas: unsupported registry metadata")
	}
	if len(r.Origins) > RequiredCandidates+MaxTestFixtures {
		return errors.New("atlas: registry exceeds bounded origin count")
	}
	if !sort.SliceIsSorted(r.Origins, func(i, j int) bool { return r.Origins[i].ID < r.Origins[j].ID }) {
		return errors.New("atlas: registry origins must be sorted by ID")
	}
	seenIDs := make(map[string]struct{}, len(r.Origins))
	seenOrigins := make(map[string]struct{}, len(r.Origins))
	seenExecutionIDs := make(map[string]struct{})
	publicRecords := make(map[string]*OriginRecord)
	for index := range r.Origins {
		record := &r.Origins[index]
		if _, exists := seenIDs[record.ID]; exists {
			return fmt.Errorf("atlas: duplicate registry origin %q", record.ID)
		}
		if _, exists := seenOrigins[record.CanonicalOrigin]; exists {
			return fmt.Errorf("atlas: duplicate registry canonical origin %q", record.CanonicalOrigin)
		}
		seenIDs[record.ID] = struct{}{}
		seenOrigins[record.CanonicalOrigin] = struct{}{}
		if err := record.Validate(); err != nil {
			return fmt.Errorf("atlas: origin %q: %w", record.ID, err)
		}
		switch record.Scope {
		case ScopeGenesisPublic:
			candidate, err := selection.Find(record.ID)
			if err != nil {
				return fmt.Errorf("atlas: public registry origin %q: %w", record.ID, err)
			}
			if record.CanonicalOrigin != candidate.CanonicalOrigin {
				return fmt.Errorf("atlas: origin %q changes selected identity", record.ID)
			}
			if !containsText(record.DomainFamilies, candidate.DomainFamily) {
				return fmt.Errorf("atlas: origin %q omits its selected domain family %q", record.ID, candidate.DomainFamily)
			}
			publicRecords[record.ID] = record
		case ScopeTestFixture:
			if _, err := selection.Find(record.ID); err == nil {
				return fmt.Errorf("atlas: test fixture %q collides with Genesis public selection", record.ID)
			}
		default:
			return fmt.Errorf("atlas: origin %q has unknown scope", record.ID)
		}
		for _, executionID := range record.ExecutionCatalogIDs {
			if _, exists := seenExecutionIDs[executionID]; exists {
				return fmt.Errorf("atlas: duplicate execution catalog ID %q", executionID)
			}
			seenExecutionIDs[executionID] = struct{}{}
		}
		if record.Policy.PolicyID != nil {
			if record.Scope != ScopeGenesisPublic {
				return fmt.Errorf("atlas: test fixture %q cannot bind a public-origin policy", record.ID)
			}
			if err := record.validatePolicyBinding(policies); err != nil {
				return fmt.Errorf("atlas: origin %q: %w", record.ID, err)
			}
		}
	}
	for index := range policies.Policies {
		if _, exists := publicRecords[policies.Policies[index].OriginID]; !exists {
			return fmt.Errorf("atlas: policy origin %q has no canonical public registry record", policies.Policies[index].OriginID)
		}
	}
	return nil
}

func (r *Registry) Find(id string) (*OriginRecord, error) {
	index := sort.Search(len(r.Origins), func(index int) bool { return r.Origins[index].ID >= id })
	if index >= len(r.Origins) || r.Origins[index].ID != id {
		return nil, fmt.Errorf("atlas: unknown canonical origin %q", id)
	}
	return &r.Origins[index], nil
}

func (r *Registry) FindExecutionCatalogID(id string) (*OriginRecord, error) {
	for index := range r.Origins {
		for _, candidate := range r.Origins[index].ExecutionCatalogIDs {
			if candidate == id {
				return &r.Origins[index], nil
			}
		}
	}
	return nil, fmt.Errorf("atlas: unknown execution catalog ID %q", id)
}

func (o *OriginRecord) Validate() error {
	parsed, err := url.Parse(o.CanonicalOrigin)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("invalid canonical origin")
	}
	for name, value := range map[string]string{
		"ID": o.ID, "registrable domain": o.RegistrableDomain, "publisher name": o.Publisher.Name,
		"publisher kind": o.Publisher.Kind, "jurisdiction": o.Jurisdiction,
		"authority class": o.AuthorityClass, "refresh class": o.Runtime.RefreshClass,
	} {
		if !validText(value, 4096) {
			return fmt.Errorf("invalid %s", name)
		}
	}
	if !idPattern.MatchString(o.ID) || !uniqueSorted(o.ExecutionCatalogIDs) {
		return errors.New("invalid origin ID or execution catalog IDs")
	}
	if o.Catalog.State != CatalogCataloged || o.Catalog.ReviewedAt == nil || o.Catalog.Reviewer == nil || !validEvidenceAttestation(*o.Catalog.ReviewedAt, *o.Catalog.Reviewer, o.Catalog.EvidenceRefs) {
		return errors.New("registry record requires evidence-attested cataloged state")
	}
	if len(o.Languages) == 0 || len(o.Languages) > 64 || !uniqueSorted(o.Languages) {
		return errors.New("languages must be explicit, sorted, unique, and bounded")
	}
	if o.Scope == ScopeGenesisPublic {
		if len(o.DomainFamilies) == 0 || len(o.DomainFamilies) > len(RequiredFamilyQuotas) || !uniqueSorted(o.DomainFamilies) {
			return errors.New("public-origin domain families must be explicit, sorted, unique, and bounded")
		}
		for _, family := range o.DomainFamilies {
			if _, exists := RequiredFamilyQuotas[family]; !exists {
				return fmt.Errorf("unknown domain family %q", family)
			}
		}
	} else if o.Scope == ScopeTestFixture && len(o.DomainFamilies) != 0 {
		return errors.New("test fixtures cannot contribute to public domain-family coverage")
	}
	if err := o.validatePolicyDimension(); err != nil {
		return err
	}
	if err := o.validateTechnicalDimension(); err != nil {
		return err
	}
	if err := o.validatePublisherDimension(); err != nil {
		return err
	}
	if err := o.validateHealthDimension(); err != nil {
		return err
	}
	if err := o.validateTrustDimensions(); err != nil {
		return err
	}
	if err := o.validateCapabilityMetadata(); err != nil {
		return err
	}
	if o.Runtime.RequestBudget < 0 || o.Runtime.RequestBudget > 1440 || o.Runtime.StorageBudgetByte < 0 || o.Runtime.StorageBudgetByte > 100<<20 {
		return errors.New("runtime budget outside E3 bounds")
	}
	if err := o.Runtime.Scheduler.Validate(o.Runtime); err != nil {
		return err
	}
	if o.Runtime.Scheduler.State != "disabled" && (o.Policy.ReviewState != PolicyCompleted || o.Policy.Decision != DecisionPermitLive && o.Policy.Decision != DecisionPermitWithConstraints) {
		return errors.New("active scheduler requires a completed live-permitting policy decision")
	}
	if (o.Health.Status == HealthSuspended || o.Health.Status == HealthRevoked || o.AdapterTrust.Status == AdapterTrustRevoked || o.MappingTrust.Status == MappingTrustRevoked) && o.Runtime.Scheduler.State != "disabled" {
		return errors.New("suspended or revoked origin state requires disabled scheduler")
	}
	return nil
}

func (o *OriginRecord) validatePolicyDimension() error {
	if !validPolicyDecision(o.Policy.Decision) {
		return errors.New("invalid policy decision")
	}
	if o.Policy.ReviewState == PolicyPending {
		if o.Policy.Decision != DecisionUncertain || o.Policy.ReviewedAt != nil || o.Policy.Reviewer != nil {
			return errors.New("pending policy must remain uncertain and cannot claim completed review metadata")
		}
	} else if o.Policy.ReviewState == PolicyCompleted {
		if o.Policy.ReviewedAt == nil || o.Policy.Reviewer == nil || o.Policy.PolicyID == nil || o.Policy.PolicySetDigest == nil || !validDigestRef(*o.Policy.PolicySetDigest) {
			return errors.New("completed policy requires reviewer, time, and exact policy binding")
		}
		if err := canonicalTime(*o.Policy.ReviewedAt); err != nil || !validText(*o.Policy.Reviewer, 4096) {
			return errors.New("completed policy review metadata is invalid")
		}
	} else {
		return errors.New("invalid policy review state")
	}
	if (o.Policy.PolicyID == nil) != (o.Policy.PolicySetDigest == nil) {
		return errors.New("policy ID and policy-set digest must be present together")
	}
	for _, value := range []string{o.Policy.Attribution, o.Policy.Authentication, o.Policy.RatePolicy, o.Policy.RetentionPolicy, o.Policy.TermsReference, o.Policy.RiskState, o.Policy.ReviewerNotes} {
		if !validText(value, 16<<10) {
			return errors.New("policy dimension contains invalid bounded text")
		}
	}
	return nil
}

func (o *OriginRecord) validateTechnicalDimension() error {
	level := o.Technical.Stage.Index()
	if level < 0 {
		return errors.New("invalid technical stage")
	}
	if level == 0 {
		if o.Technical.ReviewedAt != nil || o.Technical.Reviewer != nil || len(o.Technical.EvidenceRefs) != 0 {
			return errors.New("unprofiled stage cannot claim technical attestation")
		}
		return nil
	}
	if o.Technical.ReviewedAt == nil || o.Technical.Reviewer == nil || !validEvidenceAttestation(*o.Technical.ReviewedAt, *o.Technical.Reviewer, o.Technical.EvidenceRefs) {
		return errors.New("technical stage requires dated evidence attestation")
	}
	if o.Discovery.ProfiledAt == nil || len(o.Discovery.Sources) == 0 || !uniqueSorted(o.Discovery.Sources) {
		return errors.New("profiled-or-later technical stage requires profile evidence")
	}
	if err := canonicalTime(*o.Discovery.ProfiledAt); err != nil {
		return errors.New("invalid profile timestamp")
	}
	if level >= TechnicalObserved.Index() && len(o.Evidence.Artifacts) == 0 {
		return errors.New("observed-or-later stage requires immutable artifact evidence")
	}
	lastPath := ""
	for index, artifact := range o.Evidence.Artifacts {
		if !safeRepositoryPath(artifact.Path) || !validDigestRef(artifact.Digest) || index > 0 && artifact.Path <= lastPath {
			return errors.New("evidence artifacts must have unique sorted safe paths and SHA-256 digests")
		}
		lastPath = artifact.Path
	}
	if level >= TechnicalNativeSchema.Index() && o.Semantics.NativeModule == nil {
		return errors.New("native-schema-or-later stage requires native module")
	}
	if level >= TechnicalCompiled.Index() && (o.Semantics.AdapterID == nil || o.Semantics.ConformanceRef == nil || o.Semantics.ContractSetDigest == nil || !validDigestRef(*o.Semantics.ContractSetDigest)) {
		return errors.New("compiled-or-later stage requires adapter and conformance evidence")
	}
	if level >= TechnicalSemanticallyLinked.Index() && (len(o.Semantics.MappingModules) == 0 || len(o.Semantics.ConceptIDs) == 0 || len(o.Semantics.OperationIDs) == 0 || len(o.Semantics.SemanticClosureDigests) == 0 || !uniqueSorted(o.Semantics.SemanticClosureDigests)) {
		return errors.New("semantically-linked-or-later stage requires reviewed semantic evidence")
	}
	for _, digest := range o.Semantics.SemanticClosureDigests {
		if !validDigestRef(digest) {
			return errors.New("invalid semantic closure digest")
		}
	}
	if level >= TechnicalLive.Index() && (o.Health.Status != HealthHealthy && o.Health.Status != HealthDegraded) {
		return errors.New("live technical stage requires current health evidence")
	}
	return nil
}

func (r *Registry) ValidateArtifactFiles(root string) error {
	for originIndex := range r.Origins {
		for artifactIndex := range r.Origins[originIndex].Evidence.Artifacts {
			artifact := r.Origins[originIndex].Evidence.Artifacts[artifactIndex]
			if !safeRepositoryPath(artifact.Path) {
				return fmt.Errorf("atlas: origin %q has unsafe evidence path %q", r.Origins[originIndex].ID, artifact.Path)
			}
			data, err := readBoundedRegular(filepath.Join(root, filepath.FromSlash(artifact.Path)), MaxRegistryBytes)
			if err != nil {
				return fmt.Errorf("atlas: origin %q read evidence %q: %w", r.Origins[originIndex].ID, artifact.Path, err)
			}
			digest := sha256.Sum256(data)
			if "sha256:"+hex.EncodeToString(digest[:]) != artifact.Digest {
				return fmt.Errorf("atlas: origin %q evidence digest mismatch for %q", r.Origins[originIndex].ID, artifact.Path)
			}
		}
	}
	return nil
}

func safeRepositoryPath(path string) bool {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != filepath.FromSlash(path) {
		return false
	}
	return path != ".." && !strings.HasPrefix(path, "../") && !strings.Contains(path, "\\")
}

func (o *OriginRecord) validatePublisherDimension() error {
	switch o.Publisher.Status {
	case PublisherUnclaimed:
		if o.Publisher.ReviewedAt != nil || o.Publisher.Reviewer != nil || len(o.Publisher.EvidenceRefs) != 0 || o.Publisher.ManifestURL != nil {
			return errors.New("unclaimed publisher status cannot carry verification evidence")
		}
	case PublisherDomainVerified, PublisherApproved, PublisherSigned:
		if o.Publisher.ReviewedAt == nil || o.Publisher.Reviewer == nil || !validEvidenceAttestation(*o.Publisher.ReviewedAt, *o.Publisher.Reviewer, o.Publisher.EvidenceRefs) {
			return errors.New("verified publisher status requires dated evidence")
		}
		if o.Publisher.Status == PublisherSigned && o.Publisher.ManifestURL == nil {
			return errors.New("publisher-signed status requires manifest URL")
		}
	default:
		return errors.New("invalid publisher status")
	}
	return nil
}

func (o *OriginRecord) validateHealthDimension() error {
	switch o.Health.Status {
	case HealthUnknown:
		if o.Health.CheckedAt != nil || len(o.Health.EvidenceRefs) != 0 {
			return errors.New("unknown health cannot carry health evidence")
		}
	case HealthHealthy, HealthDegraded, HealthStale, HealthSuspended, HealthRevoked:
		if o.Health.CheckedAt == nil || len(o.Health.EvidenceRefs) == 0 || !uniqueSorted(o.Health.EvidenceRefs) || canonicalTime(*o.Health.CheckedAt) != nil {
			return errors.New("known health status requires dated evidence")
		}
	default:
		return errors.New("invalid health status")
	}
	return nil
}

func (o *OriginRecord) validateTrustDimensions() error {
	if !validAdapterTrust(o.AdapterTrust.Status) || !validMappingTrust(o.MappingTrust.Status) {
		return errors.New("invalid adapter or mapping trust status")
	}
	if err := validateTrustAttestation(string(o.AdapterTrust.Status), string(AdapterTrustNone), o.AdapterTrust.ReviewedAt, o.AdapterTrust.Reviewer, o.AdapterTrust.EvidenceRefs); err != nil {
		return fmt.Errorf("adapter trust: %w", err)
	}
	if err := validateTrustAttestation(string(o.MappingTrust.Status), string(MappingTrustNone), o.MappingTrust.ReviewedAt, o.MappingTrust.Reviewer, o.MappingTrust.EvidenceRefs); err != nil {
		return fmt.Errorf("mapping trust: %w", err)
	}
	if o.Technical.Stage.Index() >= TechnicalCompiled.Index() && o.AdapterTrust.Status == AdapterTrustNone {
		return errors.New("compiled technical stage requires explicit adapter trust")
	}
	if o.Technical.Stage.Index() >= TechnicalSemanticallyLinked.Index() && o.MappingTrust.Status == MappingTrustNone {
		return errors.New("semantically linked stage requires explicit mapping trust")
	}
	return nil
}

func validateTrustAttestation(status, zero string, reviewedAt, reviewer *string, evidence []string) error {
	if status == zero {
		if reviewedAt != nil || reviewer != nil || len(evidence) != 0 {
			return errors.New("none status cannot carry trust evidence")
		}
		return nil
	}
	if reviewedAt == nil || reviewer == nil || !validEvidenceAttestation(*reviewedAt, *reviewer, evidence) {
		return errors.New("non-none status requires dated evidence")
	}
	return nil
}

func validEvidenceAttestation(reviewedAt, reviewer string, evidence []string) bool {
	return canonicalTime(reviewedAt) == nil && validText(reviewer, 4096) && len(evidence) > 0 && uniqueSorted(evidence)
}

func (o *OriginRecord) validatePolicyBinding(policies *PolicySet) error {
	policy, err := policies.Find(o.ID)
	if err != nil {
		return err
	}
	if o.Policy.PolicyID == nil || *o.Policy.PolicyID != policy.ID || o.Policy.PolicySetDigest == nil || *o.Policy.PolicySetDigest != policies.DigestReference() {
		return errors.New("policy dimension is not bound to the exact policy artifact")
	}
	if o.Policy.ReviewState != policy.ReviewState || o.Policy.Decision != policy.Decision || !sameOptionalText(o.Policy.ReviewedAt, policy.ReviewedAt) || !sameOptionalText(o.Policy.Reviewer, policy.Reviewer) || o.Policy.Attribution != policy.Attribution || o.Policy.Authentication != policy.Authentication || o.Policy.RatePolicy != policy.RatePolicy || o.Policy.RetentionPolicy != policy.RetentionPolicy || o.Policy.TermsReference != policy.TermsReference || o.Policy.RiskState != policy.RiskState || o.Policy.ReviewerNotes != policy.ReviewerNotes {
		return errors.New("registry policy dimension disagrees with policy artifact")
	}
	if policy.Robots.ArtifactDigest == nil {
		if o.Policy.RobotsDigest != nil {
			return errors.New("registry robots digest is not present in policy")
		}
	} else if o.Policy.RobotsDigest == nil || *o.Policy.RobotsDigest != *policy.Robots.ArtifactDigest {
		return errors.New("registry robots digest disagrees with policy")
	}
	if (policy.ReviewState != PolicyCompleted || policy.Decision != DecisionPermitLive && policy.Decision != DecisionPermitWithConstraints) && o.Runtime.Scheduler.State != "disabled" {
		return errors.New("policy does not authorize an active scheduler")
	}
	return nil
}

func sameOptionalText(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func containsText(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func validPolicyDecision(value PolicyDecision) bool {
	switch value {
	case DecisionPermitLive, DecisionPermitWithConstraints, DecisionProfileOnly, DecisionCatalogOnly, DecisionDeny, DecisionUncertain:
		return true
	default:
		return false
	}
}

func validAdapterTrust(value AdapterTrustStatus) bool {
	switch value {
	case AdapterTrustNone, AdapterTrustCandidate, AdapterTrustReviewed, AdapterTrustConformant, AdapterTrustRevoked:
		return true
	default:
		return false
	}
}

func validMappingTrust(value MappingTrustStatus) bool {
	switch value {
	case MappingTrustNone, MappingTrustCandidate, MappingTrustReviewed, MappingTrustDisputed, MappingTrustRevoked:
		return true
	default:
		return false
	}
}

func canonicalTime(value string) error {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Location() != time.UTC || parsed.Format(time.RFC3339Nano) != value {
		return errors.New("timestamp is not canonical UTC")
	}
	return nil
}

func uniqueSorted(values []string) bool {
	if !sort.StringsAreSorted(values) {
		return false
	}
	for i := range values {
		if !validText(values[i], 16<<10) || i > 0 && values[i] == values[i-1] {
			return false
		}
	}
	return true
}

func allText(values ...string) bool {
	for _, value := range values {
		if !validText(value, 16<<10) {
			return false
		}
	}
	return true
}

func validDigestRef(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, char := range value[len("sha256:"):] {
		if char < '0' || char > '9' && char < 'a' || char > 'f' {
			return false
		}
	}
	return true
}
