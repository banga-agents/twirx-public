package atlas

import (
	"errors"
	"fmt"
	"regexp"
)

var capabilityIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,254}$`)

type InterfaceKind string

const (
	InterfaceTWIRXNative     InterfaceKind = "twirx_native"
	InterfaceMCP             InterfaceKind = "mcp"
	InterfaceOpenAPI         InterfaceKind = "openapi"
	InterfaceGraphQL         InterfaceKind = "graphql"
	InterfaceREST            InterfaceKind = "rest"
	InterfaceRSS             InterfaceKind = "rss"
	InterfaceAtom            InterfaceKind = "atom"
	InterfaceJSONLD          InterfaceKind = "jsonld"
	InterfaceMicrodata       InterfaceKind = "microdata"
	InterfaceSemanticHTML    InterfaceKind = "semantic_html"
	InterfaceWebMCP          InterfaceKind = "webmcp"
	InterfaceBrowser         InterfaceKind = "browser"
	InterfaceSearchDiscovery InterfaceKind = "search_discovery"
)

type DeclarationStatus string

const (
	DeclarationPublisherDeclared DeclarationStatus = "publisher_declared"
	DeclarationObserved          DeclarationStatus = "observed"
	DeclarationInferred          DeclarationStatus = "inferred"
	DeclarationArchiveDerived    DeclarationStatus = "archive_derived"
)

type InterfaceExecutionStatus string

const (
	InterfaceDescriptiveOnly    InterfaceExecutionStatus = "descriptive_only"
	InterfaceNotAdmitted        InterfaceExecutionStatus = "not_admitted"
	InterfaceAdmittedPublicRead InterfaceExecutionStatus = "admitted_public_read"
)

type InterfaceDeclaration struct {
	ID                string                   `json:"id"`
	Kind              InterfaceKind            `json:"kind"`
	DeclarationStatus DeclarationStatus        `json:"declaration_status"`
	EndpointOrLocator string                   `json:"endpoint_or_locator"`
	Authentication    string                   `json:"authentication"`
	Health            HealthStatus             `json:"health"`
	ExecutionStatus   InterfaceExecutionStatus `json:"execution_status"`
	EvidenceDigest    string                   `json:"evidence_digest"`
}

type EffectClass string

const (
	EffectPublicRead         EffectClass = "public_read"
	EffectAuthenticatedRead  EffectClass = "authenticated_read"
	EffectReversibleWrite    EffectClass = "reversible_write"
	EffectStatefulCommitment EffectClass = "stateful_commitment"
	EffectFinancial          EffectClass = "financial"
	EffectLegalOrMaterial    EffectClass = "legal_or_material"
	EffectDestructive        EffectClass = "destructive"
	EffectUnknown            EffectClass = "unknown"
)

type CapabilityStatus string

const (
	CapabilityObserved        CapabilityStatus = "observed"
	CapabilityCandidateStatus CapabilityStatus = "candidate"
	CapabilityReviewed        CapabilityStatus = "reviewed"
	CapabilityAdmitted        CapabilityStatus = "admitted"
)

type CapabilityCandidate struct {
	NativeID                    string           `json:"native_id"`
	NativeLabel                 string           `json:"native_label"`
	SemanticOperationCandidates []string         `json:"semantic_operation_candidates"`
	InputShapeRef               *string          `json:"input_shape_ref"`
	OutputShapeRef              *string          `json:"output_shape_ref"`
	EffectClass                 EffectClass      `json:"effect_class"`
	Status                      CapabilityStatus `json:"status"`
	InterfaceRefs               []string         `json:"interface_refs"`
	EvidenceDigest              string           `json:"evidence_digest"`
}

type AccessClass string

const (
	AccessPublicFree    AccessClass = "public_free"
	AccessAuthenticated AccessClass = "authenticated"
	AccessSubscription  AccessClass = "subscription"
	AccessMetered       AccessClass = "metered"
	AccessPaidPerUse    AccessClass = "paid_per_use"
	AccessQuoteRequired AccessClass = "quote_required"
	AccessRestricted    AccessClass = "restricted"
	AccessUnknown       AccessClass = "unknown"
)

type AccessAssessmentStatus string

const (
	AccessNotAssessed AccessAssessmentStatus = "not_assessed"
	AccessCandidate   AccessAssessmentStatus = "candidate"
	AccessObserved    AccessAssessmentStatus = "observed"
	AccessReviewed    AccessAssessmentStatus = "reviewed"
)

type PriceDiscoveryStatus string

const (
	PriceNotAssessed  PriceDiscoveryStatus = "not_assessed"
	PriceNotFound     PriceDiscoveryStatus = "not_found"
	PriceCandidate    PriceDiscoveryStatus = "candidate"
	PriceSourceStated PriceDiscoveryStatus = "source_stated"
	PriceReviewed     PriceDiscoveryStatus = "reviewed"
)

type OfferStatus string

const (
	OfferCandidateStatus OfferStatus = "candidate"
	OfferObservedStatus  OfferStatus = "observed"
	OfferReviewedStatus  OfferStatus = "reviewed"
)

type OfferCandidate struct {
	ID             string               `json:"id"`
	NativeLabel    string               `json:"native_label"`
	Status         OfferStatus          `json:"status"`
	AccessClass    AccessClass          `json:"access_class"`
	TermsRef       string               `json:"terms_ref"`
	PriceStatus    PriceDiscoveryStatus `json:"price_status"`
	EvidenceDigest string               `json:"evidence_digest"`
}

type AccessMetadata struct {
	Class                     AccessClass            `json:"class"`
	AssessmentStatus          AccessAssessmentStatus `json:"assessment_status"`
	LicenseOrTermsRefs        []string               `json:"license_or_terms_refs"`
	Attribution               string                 `json:"attribution"`
	RatePolicy                string                 `json:"rate_policy"`
	PaymentProtocolCandidates []string               `json:"payment_protocol_candidates"`
	PriceDiscoveryStatus      PriceDiscoveryStatus   `json:"price_discovery_status"`
	EvidenceDigest            *string                `json:"evidence_digest"`
	OfferCandidates           []OfferCandidate       `json:"offer_candidates"`
}

type FundingClass string

const (
	FundingCommons    FundingClass = "commons"
	FundingPublisher  FundingClass = "publisher"
	FundingSponsor    FundingClass = "sponsor"
	FundingUsage      FundingClass = "usage"
	FundingSelfHosted FundingClass = "self_hosted"
	FundingUnknown    FundingClass = "unknown"
)

type InfrastructureCostEstimate struct {
	AmountMicrounits int64  `json:"amount_microunits"`
	Currency         string `json:"currency"`
	Basis            string `json:"basis"`
}

type EconomicsMetadata struct {
	FundingClass               FundingClass                `json:"funding_class"`
	Requests                   int64                       `json:"requests"`
	TransferredBytes           int64                       `json:"transferred_bytes"`
	EvidenceBytes              int64                       `json:"evidence_bytes"`
	CPUMilliseconds            int64                       `json:"cpu_milliseconds"`
	HumanReviewSeconds         int64                       `json:"human_review_seconds"`
	InfrastructureCostEstimate *InfrastructureCostEstimate `json:"infrastructure_cost_estimate"`
}

type ReadinessSignalKind string

const (
	SignalTWIRXNative                    ReadinessSignalKind = "twirx_native"
	SignalMCP                            ReadinessSignalKind = "mcp"
	SignalOpenAPI                        ReadinessSignalKind = "openapi"
	SignalGraphQL                        ReadinessSignalKind = "graphql"
	SignalAPI                            ReadinessSignalKind = "api"
	SignalRSS                            ReadinessSignalKind = "rss"
	SignalAtom                           ReadinessSignalKind = "atom"
	SignalJSONLD                         ReadinessSignalKind = "jsonld"
	SignalWebMCP                         ReadinessSignalKind = "webmcp"
	SignalPaymentRequirement             ReadinessSignalKind = "payment_requirement"
	SignalCommercialOffer                ReadinessSignalKind = "commercial_offer"
	SignalOAuthProtectedResource         ReadinessSignalKind = "oauth_protected_resource"
	SignalAgentSkill                     ReadinessSignalKind = "agent_skill"
	SignalAlternateMachineRepresentation ReadinessSignalKind = "alternate_machine_representation"
)

type SignalPresence string

const (
	SignalPresent SignalPresence = "present"
	SignalAbsent  SignalPresence = "absent"
	SignalUnknown SignalPresence = "unknown"
)

type ObservationClass string

const (
	ObservationPublisherDeclared ObservationClass = "publisher_declared"
	ObservationObserved          ObservationClass = "observed"
	ObservationInferred          ObservationClass = "inferred"
	ObservationArchiveDerived    ObservationClass = "archive_derived"
	ObservationNotAssessed       ObservationClass = "not_assessed"
)

type StandardStatus string

const (
	StandardFinalized     StandardStatus = "finalized"
	StandardDraft         StandardStatus = "draft"
	StandardProposal      StandardStatus = "proposal"
	StandardExperimental  StandardStatus = "experimental"
	StandardNotApplicable StandardStatus = "not_applicable"
	StandardUnknown       StandardStatus = "unknown"
)

type PublisherReadinessSignal struct {
	Kind             ReadinessSignalKind `json:"kind"`
	Presence         SignalPresence      `json:"presence"`
	Source           string              `json:"source"`
	ObservationClass ObservationClass    `json:"observation_class"`
	ObservedAt       *string             `json:"observed_at"`
	Freshness        string              `json:"freshness"`
	StandardStatus   StandardStatus      `json:"standard_status"`
	EvidenceDigest   *string             `json:"evidence_digest"`
}

type PublisherReadiness struct {
	Signals []PublisherReadinessSignal `json:"signals"`
}

type CapabilityCounts struct {
	InterfaceKinds                     map[string]int   `json:"interface_kinds"`
	CapabilityCandidates               int              `json:"capability_candidates"`
	CapabilityStatus                   map[string]int   `json:"capability_status"`
	EffectClasses                      map[string]int   `json:"effect_classes"`
	AdmittedPublicReadOperations       int              `json:"admitted_public_read_operations"`
	AccessClasses                      map[string]int   `json:"access_classes"`
	PriceDiscoveryStatus               map[string]int   `json:"price_discovery_status"`
	CommercialOfferCandidates          int              `json:"commercial_offer_candidates"`
	PublisherNativeDeclarations        int              `json:"publisher_native_declarations"`
	MachineReadablePaymentDeclarations int              `json:"machine_readable_payment_declarations"`
	ReadinessSignalsPresent            map[string]int   `json:"readiness_signals_present"`
	Requests                           int64            `json:"requests"`
	TransferredBytes                   int64            `json:"transferred_bytes"`
	EvidenceBytes                      int64            `json:"evidence_bytes"`
	CPUMilliseconds                    int64            `json:"cpu_milliseconds"`
	HumanReviewSeconds                 int64            `json:"human_review_seconds"`
	InfrastructureCostMicrounits       map[string]int64 `json:"infrastructure_cost_microunits"`
}

func NewCapabilityCounts(defaultOrigins int) CapabilityCounts {
	counts := CapabilityCounts{
		InterfaceKinds: map[string]int{}, CapabilityStatus: map[string]int{}, EffectClasses: map[string]int{},
		AccessClasses: map[string]int{}, PriceDiscoveryStatus: map[string]int{}, ReadinessSignalsPresent: map[string]int{},
		InfrastructureCostMicrounits: map[string]int64{},
	}
	for _, value := range []InterfaceKind{InterfaceTWIRXNative, InterfaceMCP, InterfaceOpenAPI, InterfaceGraphQL, InterfaceREST, InterfaceRSS, InterfaceAtom, InterfaceJSONLD, InterfaceMicrodata, InterfaceSemanticHTML, InterfaceWebMCP, InterfaceBrowser, InterfaceSearchDiscovery} {
		counts.InterfaceKinds[string(value)] = 0
	}
	for _, value := range []CapabilityStatus{CapabilityObserved, CapabilityCandidateStatus, CapabilityReviewed, CapabilityAdmitted} {
		counts.CapabilityStatus[string(value)] = 0
	}
	for _, value := range []EffectClass{EffectPublicRead, EffectAuthenticatedRead, EffectReversibleWrite, EffectStatefulCommitment, EffectFinancial, EffectLegalOrMaterial, EffectDestructive, EffectUnknown} {
		counts.EffectClasses[string(value)] = 0
	}
	for _, value := range []AccessClass{AccessPublicFree, AccessAuthenticated, AccessSubscription, AccessMetered, AccessPaidPerUse, AccessQuoteRequired, AccessRestricted, AccessUnknown} {
		counts.AccessClasses[string(value)] = 0
	}
	counts.AccessClasses[string(AccessUnknown)] = defaultOrigins
	for _, value := range []PriceDiscoveryStatus{PriceNotAssessed, PriceNotFound, PriceCandidate, PriceSourceStated, PriceReviewed} {
		counts.PriceDiscoveryStatus[string(value)] = 0
	}
	counts.PriceDiscoveryStatus[string(PriceNotAssessed)] = defaultOrigins
	for _, value := range []ReadinessSignalKind{SignalTWIRXNative, SignalMCP, SignalOpenAPI, SignalGraphQL, SignalAPI, SignalRSS, SignalAtom, SignalJSONLD, SignalWebMCP, SignalPaymentRequirement, SignalCommercialOffer, SignalOAuthProtectedResource, SignalAgentSkill, SignalAlternateMachineRepresentation} {
		counts.ReadinessSignalsPresent[string(value)] = 0
	}
	return counts
}

func (c *CapabilityCounts) Add(record *OriginRecord, replacesDefault bool) {
	if replacesDefault {
		c.AccessClasses[string(AccessUnknown)]--
		c.PriceDiscoveryStatus[string(PriceNotAssessed)]--
	}
	c.AccessClasses[string(record.Access.Class)]++
	c.PriceDiscoveryStatus[string(record.Access.PriceDiscoveryStatus)]++
	for _, declaration := range record.Interfaces {
		c.InterfaceKinds[string(declaration.Kind)]++
		if declaration.Kind == InterfaceTWIRXNative && declaration.DeclarationStatus == DeclarationPublisherDeclared {
			c.PublisherNativeDeclarations++
		}
	}
	for _, capability := range record.Capabilities {
		c.CapabilityCandidates++
		c.CapabilityStatus[string(capability.Status)]++
		c.EffectClasses[string(capability.EffectClass)]++
		if capability.Status == CapabilityAdmitted && capability.EffectClass == EffectPublicRead {
			c.AdmittedPublicReadOperations++
		}
	}
	c.CommercialOfferCandidates += len(record.Access.OfferCandidates)
	for _, signal := range record.PublisherReadiness.Signals {
		if signal.Presence != SignalPresent {
			continue
		}
		c.ReadinessSignalsPresent[string(signal.Kind)]++
		if signal.Kind == SignalPaymentRequirement {
			c.MachineReadablePaymentDeclarations++
		}
	}
	c.Requests += record.Economics.Requests
	c.TransferredBytes += record.Economics.TransferredBytes
	c.EvidenceBytes += record.Economics.EvidenceBytes
	c.CPUMilliseconds += record.Economics.CPUMilliseconds
	c.HumanReviewSeconds += record.Economics.HumanReviewSeconds
	if estimate := record.Economics.InfrastructureCostEstimate; estimate != nil {
		c.InfrastructureCostMicrounits[estimate.Currency] += estimate.AmountMicrounits
	}
}

func (o *OriginRecord) validateCapabilityMetadata() error {
	interfaces := make(map[string]InterfaceDeclaration, len(o.Interfaces))
	previous := ""
	for _, declaration := range o.Interfaces {
		if declaration.ID <= previous || !idPattern.MatchString(declaration.ID) || !validInterfaceKind(declaration.Kind) || !validDeclarationStatus(declaration.DeclarationStatus) || !validInterfaceExecutionStatus(declaration.ExecutionStatus) || !allText(declaration.EndpointOrLocator, declaration.Authentication) || !validHealthStatus(declaration.Health) || !validDigestRef(declaration.EvidenceDigest) {
			return errors.New("interfaces must be sorted, unique, bounded, evidence-bound declarations")
		}
		if declaration.ExecutionStatus == InterfaceAdmittedPublicRead {
			if declaration.Kind != InterfaceREST && declaration.Kind != InterfaceTWIRXNative {
				return errors.New("E3.2 cannot admit browser, WebMCP, MCP, authenticated, or other new interface routes")
			}
			catalogBound := false
			for _, executionID := range o.ExecutionCatalogIDs {
				if declaration.EndpointOrLocator == "origins/catalog.json#"+executionID {
					catalogBound = true
					break
				}
			}
			if !catalogBound {
				return errors.New("admitted E3.2 interface must bind an existing E2 execution-catalog entry")
			}
		}
		interfaces[declaration.ID] = declaration
		previous = declaration.ID
	}
	previous = ""
	for _, capability := range o.Capabilities {
		if capability.NativeID <= previous || !capabilityIDPattern.MatchString(capability.NativeID) || !validCapabilityStatus(capability.Status) || !validEffectClass(capability.EffectClass) || !allText(capability.NativeLabel) || !uniqueSorted(capability.SemanticOperationCandidates) || !uniqueSorted(capability.InterfaceRefs) || len(capability.InterfaceRefs) == 0 || !validDigestRef(capability.EvidenceDigest) {
			return errors.New("capabilities must be sorted, unique, bounded, evidence-bound declarations")
		}
		if capability.InputShapeRef != nil && !validText(*capability.InputShapeRef, 16<<10) || capability.OutputShapeRef != nil && !validText(*capability.OutputShapeRef, 16<<10) {
			return errors.New("capability shape reference is invalid")
		}
		for _, reference := range capability.InterfaceRefs {
			declaration, exists := interfaces[reference]
			if !exists {
				return fmt.Errorf("capability references unknown interface %q", reference)
			}
			if capability.Status == CapabilityAdmitted && declaration.ExecutionStatus != InterfaceAdmittedPublicRead {
				return errors.New("admitted capability requires an admitted public-read interface")
			}
		}
		if capability.Status == CapabilityAdmitted {
			if capability.EffectClass != EffectPublicRead || o.Technical.Stage.Index() < TechnicalCompiled.Index() || !containsText(o.Semantics.OperationIDs, capability.NativeID) {
				return errors.New("E3.2 admits only existing compiled public-read operations")
			}
		}
		previous = capability.NativeID
	}
	if err := o.validateAccessMetadata(); err != nil {
		return err
	}
	if err := o.validateEconomicsMetadata(); err != nil {
		return err
	}
	if err := o.validatePublisherReadiness(); err != nil {
		return err
	}
	return o.validateDescriptorEvidenceBindings()
}

func (o *OriginRecord) validateAccessMetadata() error {
	if !validAccessClass(o.Access.Class) || !validAccessAssessmentStatus(o.Access.AssessmentStatus) || !validPriceStatus(o.Access.PriceDiscoveryStatus) || !uniqueSorted(o.Access.LicenseOrTermsRefs) || !uniqueSorted(o.Access.PaymentProtocolCandidates) || !allText(o.Access.Attribution, o.Access.RatePolicy) {
		return errors.New("invalid access metadata")
	}
	if o.Access.AssessmentStatus == AccessNotAssessed {
		if o.Access.Class != AccessUnknown || o.Access.EvidenceDigest != nil || o.Access.PriceDiscoveryStatus != PriceNotAssessed || len(o.Access.OfferCandidates) != 0 {
			return errors.New("unassessed access cannot claim class, evidence, price, or offers")
		}
	} else if o.Access.EvidenceDigest == nil || !validDigestRef(*o.Access.EvidenceDigest) {
		return errors.New("assessed access metadata requires evidence")
	}
	previous := ""
	for _, offer := range o.Access.OfferCandidates {
		if offer.ID <= previous || !idPattern.MatchString(offer.ID) || !allText(offer.NativeLabel, offer.TermsRef) || !validOfferStatus(offer.Status) || !validAccessClass(offer.AccessClass) || offer.AccessClass == AccessUnknown || !validPriceStatus(offer.PriceStatus) || !validDigestRef(offer.EvidenceDigest) {
			return errors.New("offer candidates must be sorted, unique, provisional, and evidence-bound")
		}
		previous = offer.ID
	}
	return nil
}

func (o *OriginRecord) validateEconomicsMetadata() error {
	if !validFundingClass(o.Economics.FundingClass) || o.Economics.Requests < 0 || o.Economics.TransferredBytes < 0 || o.Economics.EvidenceBytes < 0 || o.Economics.CPUMilliseconds < 0 || o.Economics.HumanReviewSeconds < 0 {
		return errors.New("invalid non-negative economics telemetry")
	}
	if estimate := o.Economics.InfrastructureCostEstimate; estimate != nil {
		if estimate.AmountMicrounits < 0 || !allText(estimate.Currency, estimate.Basis) {
			return errors.New("invalid infrastructure cost estimate")
		}
	}
	return nil
}

func (o *OriginRecord) validatePublisherReadiness() error {
	previous := ReadinessSignalKind("")
	for _, signal := range o.PublisherReadiness.Signals {
		if string(signal.Kind) <= string(previous) || !validSignalKind(signal.Kind) || !validSignalPresence(signal.Presence) || !validObservationClass(signal.ObservationClass) || !validStandardStatus(signal.StandardStatus) || !allText(signal.Source, signal.Freshness) {
			return errors.New("publisher-readiness signals must be sorted, unique, bounded declarations")
		}
		if signal.Presence == SignalUnknown {
			if signal.ObservationClass != ObservationNotAssessed || signal.ObservedAt != nil || signal.EvidenceDigest != nil {
				return errors.New("unknown readiness signal cannot claim observation evidence")
			}
		} else if signal.ObservedAt == nil || canonicalTime(*signal.ObservedAt) != nil || signal.EvidenceDigest == nil || !validDigestRef(*signal.EvidenceDigest) {
			return errors.New("present or absent readiness signal requires dated evidence")
		}
		previous = signal.Kind
	}
	return nil
}

func (o *OriginRecord) validateDescriptorEvidenceBindings() error {
	bound := make(map[string]struct{}, len(o.Evidence.Artifacts))
	for _, artifact := range o.Evidence.Artifacts {
		bound[artifact.Digest] = struct{}{}
	}
	require := func(digest string) error {
		if _, exists := bound[digest]; !exists {
			return fmt.Errorf("descriptor evidence digest %q is not bound by the origin record", digest)
		}
		return nil
	}
	for _, declaration := range o.Interfaces {
		if err := require(declaration.EvidenceDigest); err != nil {
			return err
		}
	}
	for _, capability := range o.Capabilities {
		if err := require(capability.EvidenceDigest); err != nil {
			return err
		}
	}
	if o.Access.EvidenceDigest != nil {
		if err := require(*o.Access.EvidenceDigest); err != nil {
			return err
		}
	}
	for _, offer := range o.Access.OfferCandidates {
		if err := require(offer.EvidenceDigest); err != nil {
			return err
		}
	}
	for _, signal := range o.PublisherReadiness.Signals {
		if signal.EvidenceDigest != nil {
			if err := require(*signal.EvidenceDigest); err != nil {
				return err
			}
		}
	}
	return nil
}

func validInterfaceKind(value InterfaceKind) bool {
	switch value {
	case InterfaceTWIRXNative, InterfaceMCP, InterfaceOpenAPI, InterfaceGraphQL, InterfaceREST, InterfaceRSS, InterfaceAtom, InterfaceJSONLD, InterfaceMicrodata, InterfaceSemanticHTML, InterfaceWebMCP, InterfaceBrowser, InterfaceSearchDiscovery:
		return true
	default:
		return false
	}
}

func validDeclarationStatus(value DeclarationStatus) bool {
	return value == DeclarationPublisherDeclared || value == DeclarationObserved || value == DeclarationInferred || value == DeclarationArchiveDerived
}

func validInterfaceExecutionStatus(value InterfaceExecutionStatus) bool {
	return value == InterfaceDescriptiveOnly || value == InterfaceNotAdmitted || value == InterfaceAdmittedPublicRead
}

func validHealthStatus(value HealthStatus) bool {
	switch value {
	case HealthUnknown, HealthHealthy, HealthDegraded, HealthStale, HealthSuspended, HealthRevoked:
		return true
	default:
		return false
	}
}

func validEffectClass(value EffectClass) bool {
	switch value {
	case EffectPublicRead, EffectAuthenticatedRead, EffectReversibleWrite, EffectStatefulCommitment, EffectFinancial, EffectLegalOrMaterial, EffectDestructive, EffectUnknown:
		return true
	default:
		return false
	}
}

func validCapabilityStatus(value CapabilityStatus) bool {
	return value == CapabilityObserved || value == CapabilityCandidateStatus || value == CapabilityReviewed || value == CapabilityAdmitted
}

func validAccessClass(value AccessClass) bool {
	switch value {
	case AccessPublicFree, AccessAuthenticated, AccessSubscription, AccessMetered, AccessPaidPerUse, AccessQuoteRequired, AccessRestricted, AccessUnknown:
		return true
	default:
		return false
	}
}

func validAccessAssessmentStatus(value AccessAssessmentStatus) bool {
	return value == AccessNotAssessed || value == AccessCandidate || value == AccessObserved || value == AccessReviewed
}

func validPriceStatus(value PriceDiscoveryStatus) bool {
	return value == PriceNotAssessed || value == PriceNotFound || value == PriceCandidate || value == PriceSourceStated || value == PriceReviewed
}

func validOfferStatus(value OfferStatus) bool {
	return value == OfferCandidateStatus || value == OfferObservedStatus || value == OfferReviewedStatus
}

func validFundingClass(value FundingClass) bool {
	return value == FundingCommons || value == FundingPublisher || value == FundingSponsor || value == FundingUsage || value == FundingSelfHosted || value == FundingUnknown
}

func validSignalKind(value ReadinessSignalKind) bool {
	switch value {
	case SignalTWIRXNative, SignalMCP, SignalOpenAPI, SignalGraphQL, SignalAPI, SignalRSS, SignalAtom, SignalJSONLD, SignalWebMCP, SignalPaymentRequirement, SignalCommercialOffer, SignalOAuthProtectedResource, SignalAgentSkill, SignalAlternateMachineRepresentation:
		return true
	default:
		return false
	}
}

func validSignalPresence(value SignalPresence) bool {
	return value == SignalPresent || value == SignalAbsent || value == SignalUnknown
}

func validObservationClass(value ObservationClass) bool {
	return value == ObservationPublisherDeclared || value == ObservationObserved || value == ObservationInferred || value == ObservationArchiveDerived || value == ObservationNotAssessed
}

func validStandardStatus(value StandardStatus) bool {
	return value == StandardFinalized || value == StandardDraft || value == StandardProposal || value == StandardExperimental || value == StandardNotApplicable || value == StandardUnknown
}
