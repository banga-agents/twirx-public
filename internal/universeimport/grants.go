package universeimport

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	MaxGrantsFetchResponse = 16 << 20
	GrantsGovOriginID      = "grants-gov-api"
)

type grantsFetchResponse struct {
	ErrorCode json.Number       `json:"errorcode"`
	Message   string            `json:"msg"`
	Token     string            `json:"token"`
	Data      grantsOpportunity `json:"data"`
}

type grantsOpportunity struct {
	ID                   json.Number       `json:"id"`
	Revision             json.Number       `json:"revision"`
	OpportunityNumber    string            `json:"opportunityNumber"`
	OpportunityTitle     string            `json:"opportunityTitle"`
	OwningAgencyCode     string            `json:"owningAgencyCode"`
	Listed               string            `json:"listed"`
	PublisherUID         string            `json:"publisherUid"`
	Flag2006             string            `json:"flag2006"`
	OpportunityCategory  json.RawMessage   `json:"opportunityCategory"`
	Synopsis             grantsSynopsis    `json:"synopsis"`
	AgencyDetails        json.RawMessage   `json:"agencyDetails"`
	TopAgencyDetails     json.RawMessage   `json:"topAgencyDetails"`
	SynopsisAttachments  []json.RawMessage `json:"synopsisAttachmentFolders"`
	SynopsisDocumentURLs []json.RawMessage `json:"synopsisDocumentURLs"`
	SynopsisChanges      []json.RawMessage `json:"synAttChangeComments"`
	AssistanceListings   []json.RawMessage `json:"alns"`
	OpportunityHistory   []json.RawMessage `json:"opportunityHistoryDetails"`
	OpportunityPackages  []json.RawMessage `json:"opportunityPkgs"`
	ClosedPackages       []json.RawMessage `json:"closedOpportunityPkgs"`
	OriginalDueDateDesc  string            `json:"originalDueDateDesc"`
	SynopsisModified     []json.RawMessage `json:"synopsisModifiedFields"`
	ForecastModified     []json.RawMessage `json:"forecastModifiedFields"`
	ErrorMessages        []json.RawMessage `json:"errorMessages"`
	SynopsisPostDatePast bool              `json:"synPostDateInPast"`
	DocumentType         string            `json:"docType"`
	ForecastHistoryCount json.Number       `json:"forecastHistCount"`
	SynopsisHistoryCount json.Number       `json:"synopsisHistCount"`
	AssistCompatible     bool              `json:"assistCompatible"`
	AssistURL            string            `json:"assistURL"`
	RelatedOpportunities []json.RawMessage `json:"relatedOpps"`
	DraftMode            string            `json:"draftMode"`
}

type grantsSynopsis struct {
	OpportunityID          json.Number       `json:"opportunityId"`
	Version                json.Number       `json:"version"`
	AgencyCode             string            `json:"agencyCode"`
	AgencyName             string            `json:"agencyName"`
	AgencyPhone            string            `json:"agencyPhone"`
	AgencyAddress          string            `json:"agencyAddressDesc"`
	AgencyDetails          json.RawMessage   `json:"agencyDetails"`
	TopAgencyDetails       json.RawMessage   `json:"topAgencyDetails"`
	AgencyContactPhone     string            `json:"agencyContactPhone"`
	AgencyContactName      string            `json:"agencyContactName"`
	AgencyContactDesc      string            `json:"agencyContactDesc"`
	AgencyContactEmail     string            `json:"agencyContactEmail"`
	AgencyContactEmailDesc string            `json:"agencyContactEmailDesc"`
	SynopsisDescription    string            `json:"synopsisDesc"`
	ResponseDateDesc       string            `json:"responseDateDesc"`
	PostingDate            string            `json:"postingDate"`
	CostSharing            bool              `json:"costSharing"`
	AwardCeiling           string            `json:"awardCeiling"`
	AwardCeilingFormatted  string            `json:"awardCeilingFormatted"`
	AwardFloor             string            `json:"awardFloor"`
	AwardFloorFormatted    string            `json:"awardFloorFormatted"`
	SendEmail              string            `json:"sendEmail"`
	CreateTimestamp        string            `json:"createTimeStamp"`
	CreatedDate            string            `json:"createdDate"`
	LastUpdatedDate        string            `json:"lastUpdatedDate"`
	ApplicantTypes         []grantsCodeLabel `json:"applicantTypes"`
	FundingInstruments     []grantsCodeLabel `json:"fundingInstruments"`
	FundingCategories      []grantsCodeLabel `json:"fundingActivityCategories"`
	PostingDateCanonical   string            `json:"postingDateStr"`
	CreateTimeCanonical    string            `json:"createTimeStampStr"`
}

type grantsCodeLabel struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// CompileGrantsFetch compiles an already-stored Grants.gov fetchOpportunity
// response. The publisher's documented sample currently contains a
// token-shaped response field and public contact data. Either causes a
// fail-closed rejection, so this profile is limited to redacted conformance
// fixtures until a separate reviewed acquisition/redaction design is admitted.
func CompileGrantsFetch(representation []byte, config Config) ([]RecordArtifact, error) {
	if len(representation) == 0 || len(representation) > MaxGrantsFetchResponse {
		return nil, fmt.Errorf("grants importer: response size outside 1..%d", MaxGrantsFetchResponse)
	}
	if err := config.validate(representation); err != nil {
		return nil, err
	}
	if config.OriginID != GrantsGovOriginID {
		return nil, fmt.Errorf("grants importer: origin must be provisional E4 ID %q", GrantsGovOriginID)
	}
	policy := jsonbounded.Policy{MaxBytes: MaxGrantsFetchResponse, MaxDepth: 16, MaxScalarBytes: 1 << 20, MaxContainerEntries: 200000, MaxTokens: 1000000}
	var response grantsFetchResponse
	if err := jsonbounded.Decode(representation, &response, policy, true); err != nil {
		return nil, fmt.Errorf("grants importer: %w", err)
	}
	if response.Token != "" {
		return nil, fmt.Errorf("grants importer: token-shaped response field is not admissible")
	}
	if response.Data.PublisherUID != "" || grantsContactPresent(response.Data.Synopsis) {
		return nil, fmt.Errorf("grants importer: publisher/contact personal data is not admissible in this profile")
	}
	if response.ErrorCode.String() != "0" {
		return nil, fmt.Errorf("grants importer: source error code %q", response.ErrorCode.String())
	}
	if response.Data.ID.String() == "" || response.Data.OpportunityNumber == "" || response.Data.OpportunityTitle == "" {
		return nil, fmt.Errorf("grants importer: missing opportunity identity or title")
	}
	if response.Data.Synopsis.AgencyCode == "" || response.Data.Synopsis.AgencyName == "" {
		return nil, fmt.Errorf("grants importer: missing source agency identity")
	}
	if response.Data.OwningAgencyCode != "" && response.Data.OwningAgencyCode != response.Data.Synopsis.AgencyCode {
		return nil, fmt.Errorf("grants importer: conflicting agency codes")
	}
	record, err := compileGrantOpportunity(response.Data, config)
	if err != nil {
		return nil, err
	}
	return []RecordArtifact{record}, nil
}

func grantsContactPresent(s grantsSynopsis) bool {
	return s.AgencyPhone != "" || s.AgencyAddress != "" || s.AgencyContactPhone != "" || s.AgencyContactName != "" || s.AgencyContactDesc != "" || s.AgencyContactEmail != "" || s.AgencyContactEmailDesc != ""
}

type grantField struct {
	term       string
	locator    string
	status     string
	native     string
	typed      *dataplane.TypedValue
	transforms []string
	kind       string
}

func compileGrantOpportunity(source grantsOpportunity, config Config) (RecordArtifact, error) {
	nativeKey := "grants-gov:" + source.OpportunityNumber
	subjectCandidates := []string{"opportunity:grants-gov/" + source.ID.String()}
	fields := []grantField{
		{term: "opportunityNumber", locator: "/data/opportunityNumber", status: "resolved", native: source.OpportunityNumber, typed: &dataplane.TypedValue{Type: "identifier", Lexical: source.OpportunityNumber}, kind: "offer"},
		{term: "opportunityTitle", locator: "/data/opportunityTitle", status: "resolved", native: source.OpportunityTitle, typed: &dataplane.TypedValue{Type: "text", Lexical: source.OpportunityTitle}, kind: "offer"},
		{term: "synopsis.agencyCode", locator: "/data/synopsis/agencyCode", status: "resolved", native: source.Synopsis.AgencyCode, typed: &dataplane.TypedValue{Type: "identifier", Lexical: "grants-gov:agency/" + source.Synopsis.AgencyCode}, transforms: []string{"normalize:grants-agency-identifier@0.1"}, kind: "relationship"},
	}
	deadlineStatus := "not_provided"
	deadlineNative := ""
	if source.OriginalDueDateDesc != "" {
		deadlineStatus = "resolved"
		deadlineNative = source.OriginalDueDateDesc
	}
	fields = append(fields, grantField{term: "originalDueDateDesc", locator: "/data/originalDueDateDesc", status: deadlineStatus, native: deadlineNative, kind: "claim"})
	fields = append(fields, grantAmountField("synopsis.awardCeiling", "/data/synopsis/awardCeiling", source.Synopsis.AwardCeiling))
	fields = append(fields, grantAmountField("synopsis.awardFloor", "/data/synopsis/awardFloor", source.Synopsis.AwardFloor))
	for index, applicant := range source.Synopsis.ApplicantTypes {
		if applicant.ID == "" || applicant.Description == "" {
			return RecordArtifact{}, fmt.Errorf("grants importer: incomplete applicant type")
		}
		fields = append(fields,
			grantField{term: "synopsis.applicantTypes[].id", locator: fmt.Sprintf("/data/synopsis/applicantTypes/%d/id", index), status: "resolved", native: applicant.ID, typed: &dataplane.TypedValue{Type: "identifier", Lexical: "grants-gov:applicant-type/" + applicant.ID}, transforms: []string{"normalize:grants-applicant-type@0.1"}, kind: "claim"},
			grantField{term: "synopsis.applicantTypes[].description", locator: fmt.Sprintf("/data/synopsis/applicantTypes/%d/description", index), status: "resolved", native: applicant.Description, typed: &dataplane.TypedValue{Type: "text", Lexical: applicant.Description}, kind: "claim"},
		)
	}
	allDefinitions := grantsMappings()
	definitions := mappingsForFields(allDefinitions, fields)
	mappingByTerm := make(map[string]mappingDefinition, len(definitions))
	for _, mapping := range definitions {
		mappingByTerm[mapping.NativeTerm] = mapping
	}
	packets := make([]PacketArtifact, 0, len(fields))
	for _, field := range fields {
		mapping := mappingByTerm[field.term]
		objectLanguage := dataplane.OptionalText{}
		if field.typed != nil && field.typed.Type == "text" {
			objectLanguage = dataplane.OptionalText{Present: true, Value: "en"}
		}
		packet := dataplane.Packet{
			Version: dataplane.PacketVersion, Kind: field.kind,
			Subject:    dataplane.PacketSubject{Native: nativeKey, CanonicalCandidates: subjectCandidates},
			Predicate:  dataplane.PacketPredicate{Native: field.term, Semantic: dataplane.OptionalText{Present: true, Value: mapping.ConceptID}},
			Object:     dataplane.PacketObject{NativeStatus: field.status, NativeLexical: field.native, MediaType: dataplane.OptionalText{Present: true, Value: "application/json"}, Language: objectLanguage, Typed: field.typed},
			Context:    dataplane.PacketContext{Jurisdiction: dataplane.OptionalText{Present: true, Value: "geo:US"}, Language: dataplane.OptionalText{Present: true, Value: "en"}},
			Time:       dataplane.PacketTime{ObservedAt: config.ObservedAt},
			Source:     dataplane.PacketSource{OriginID: config.OriginID, RepresentationDigest: config.RepresentationDigest, Locator: field.locator, NativeSchemaRef: dataplane.OptionalText{Present: true, Value: "origin:grants-gov-fetch-opportunity@v1"}},
			Derivation: dataplane.PacketDerivation{ObservationDigest: config.ObservationDigest, AdapterDigest: fixedDigest("tw.e4/grants-gov-fetch-importer@0.1"), ExtractionPlanDigest: fixedDigest("tw.e4/grants-gov-fetch-extraction/" + field.locator), TransformationIDs: field.transforms, MappingIDs: []string{mapping.ID}, SemanticClosureDigest: dataplane.OptionalDigest{Present: true, Value: config.ModuleSetDigest}, CompilerContractDigest: fixedDigest("tw.semantic-frame/0.1"), CompilerVersion: "twirx-universe-import@0.1"},
			Epistemic:  dataplane.PacketEpistemic{Lane: "provisional_semantic", ExtractionStatus: "deterministic", MappingStatus: "candidate", AuthorityClass: "provider-operated-official-api", FreshnessStatus: freshness(config.EvidenceClass)},
			Lifecycle:  dataplane.PacketLifecycle{State: "current"},
			Retention:  "public_versioned",
			Disclosure: "public",
		}
		artifact, err := compilePacket(packet)
		if err != nil {
			return RecordArtifact{}, err
		}
		packets = append(packets, artifact)
	}
	mappingArtifacts, err := mappingsForPackets(config.OriginID, "tw:opportunity", "tw:opportunity@0.1.0", definitions, packets)
	if err != nil {
		return RecordArtifact{}, err
	}
	byTerm := packetsByNativePredicate(packets)
	slots := make([]dataplane.FrameSlot, 0, 8)
	addSlot := func(role, term, status, cardinality string, values []dataplane.TypedValue) error {
		slot, slotErr := frameSlotMany(role, status, cardinality, mappingByTerm[term].ID, byTerm[term], values)
		if slotErr != nil {
			return slotErr
		}
		slots = append(slots, slot)
		return nil
	}
	for _, required := range []struct{ role, term string }{{"opportunity:funder", "synopsis.agencyCode"}, {"opportunity:identifier", "opportunityNumber"}, {"opportunity:title", "opportunityTitle"}} {
		if err := addSlot(required.role, required.term, "resolved", "one", typedValues(byTerm[required.term])); err != nil {
			return RecordArtifact{}, err
		}
	}
	for _, amount := range []struct{ role, term string }{{"opportunity:maximumAmount", "synopsis.awardCeiling"}, {"opportunity:minimumAmount", "synopsis.awardFloor"}} {
		status := "unresolved"
		values := typedValues(byTerm[amount.term])
		if len(values) == 1 {
			status = "resolved"
		} else if byTerm[amount.term][0].Packet.Object.NativeStatus == "not_provided" {
			status = "not_provided"
		}
		if err := addSlot(amount.role, amount.term, status, "one", values); err != nil {
			return RecordArtifact{}, err
		}
	}
	deadlineSlotStatus := "not_provided"
	if deadlineStatus == "resolved" {
		deadlineSlotStatus = "unresolved"
	}
	if err := addSlot("opportunity:deadline", "originalDueDateDesc", deadlineSlotStatus, "one", nil); err != nil {
		return RecordArtifact{}, err
	}
	if applicantPackets := byTerm["synopsis.applicantTypes[].id"]; len(applicantPackets) > 0 {
		if err := addSlot("opportunity:applicantClass", "synopsis.applicantTypes[].id", "resolved", "many", typedValues(applicantPackets)); err != nil {
			return RecordArtifact{}, err
		}
	}
	if textPackets := byTerm["synopsis.applicantTypes[].description"]; len(textPackets) > 0 {
		if err := addSlot("opportunity:eligibilityText", "synopsis.applicantTypes[].description", "resolved", "many", typedValues(textPackets)); err != nil {
			return RecordArtifact{}, err
		}
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].RoleID < slots[j].RoleID })
	mappingIDs := make([]string, len(definitions))
	for i := range definitions {
		mappingIDs[i] = definitions[i].ID
	}
	sort.Strings(mappingIDs)
	resolvedSlots := uint64(0)
	for _, slot := range slots {
		if slot.Status == "resolved" {
			resolvedSlots++
		}
	}
	frame := dataplane.Frame{
		Version: dataplane.FrameVersion, UniverseID: "tw:opportunity", FrameType: "opportunity:GrantOpportunity", NativeKey: nativeKey, CanonicalCandidates: subjectCandidates,
		Slots: slots, Context: dataplane.PacketContext{Jurisdiction: dataplane.OptionalText{Present: true, Value: "geo:US"}, Language: dataplane.OptionalText{Present: true, Value: "en"}}, Time: dataplane.FrameTime{ComposedAt: config.ObservedAt},
		Epistemic:  dataplane.FrameEpistemic{Lane: "provisional_semantic", CompletenessMillionths: resolvedSlots * 1000000 / uint64(len(slots)), ConflictStatus: "none"},
		Derivation: dataplane.FrameDerivation{PacketDigests: sortedPacketDigests(packets), ModuleSetDigest: config.ModuleSetDigest, MappingIDs: mappingIDs, CompilerContractDigest: fixedDigest("tw.semantic-frame/0.1"), CompilerVersion: "twirx-universe-import@0.1"},
		Lifecycle:  dataplane.FrameLifecycle{State: "current"},
	}
	return finishRecord(nativeKey, config.EvidenceClass, config.RepresentationDigest, packets, mappingArtifacts, frame)
}

func mappingsForFields(definitions []mappingDefinition, fields []grantField) []mappingDefinition {
	present := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		present[field.term] = struct{}{}
	}
	result := make([]mappingDefinition, 0, len(definitions))
	for _, definition := range definitions {
		if _, ok := present[definition.NativeTerm]; ok {
			result = append(result, definition)
		}
	}
	return result
}

func grantAmountField(term, locator, native string) grantField {
	field := grantField{term: term, locator: locator, status: "not_provided", kind: "offer"}
	if native == "" {
		return field
	}
	field.status = "resolved"
	field.native = native
	if decimal, ok := canonicalUnsignedDecimal(native); ok {
		field.typed = &dataplane.TypedValue{Type: "decimal", Lexical: decimal}
		field.transforms = []string{"transform:source-number-to-decimal@0.1"}
	}
	return field
}

func canonicalUnsignedDecimal(value string) (string, bool) {
	if value == "" || strings.ContainsAny(value, "+-eE, ") {
		return "", false
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts[0]) > 1 && parts[0][0] == '0') {
		return "", false
	}
	if _, err := strconv.ParseUint(parts[0], 10, 64); err != nil {
		return "", false
	}
	if len(parts) == 1 {
		return value + ".0", true
	}
	if parts[1] == "" {
		return "", false
	}
	for _, r := range parts[1] {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return value, true
}

func typedValues(packets []PacketArtifact) []dataplane.TypedValue {
	values := make([]dataplane.TypedValue, 0, len(packets))
	for _, packet := range packets {
		if packet.Packet.Object.Typed != nil {
			values = append(values, *packet.Packet.Object.Typed)
		}
	}
	return values
}

func grantsMappings() []mappingDefinition {
	return []mappingDefinition{
		{ID: "mapping:grants-gov/applicant-type@0.1", NativeTerm: "synopsis.applicantTypes[].id", SchemaRef: "origin:grants-gov-fetch-opportunity@v1", LocatorPattern: "/data/synopsis/applicantTypes/*/id", ConceptID: "opportunity:ApplicantClass", RoleID: "opportunity:applicantClass"},
		{ID: "mapping:grants-gov/applicant-type-description@0.1", NativeTerm: "synopsis.applicantTypes[].description", SchemaRef: "origin:grants-gov-fetch-opportunity@v1", LocatorPattern: "/data/synopsis/applicantTypes/*/description", ConceptID: "opportunity:EligibilityRule", RoleID: "opportunity:eligibilityText"},
		{ID: "mapping:grants-gov/award-ceiling@0.1", NativeTerm: "synopsis.awardCeiling", SchemaRef: "origin:grants-gov-fetch-opportunity@v1", LocatorPattern: "/data/synopsis/awardCeiling", ConceptID: "tw:Price", RoleID: "opportunity:maximumAmount"},
		{ID: "mapping:grants-gov/award-floor@0.1", NativeTerm: "synopsis.awardFloor", SchemaRef: "origin:grants-gov-fetch-opportunity@v1", LocatorPattern: "/data/synopsis/awardFloor", ConceptID: "tw:Price", RoleID: "opportunity:minimumAmount"},
		{ID: "mapping:grants-gov/due-date-description@0.1", NativeTerm: "originalDueDateDesc", SchemaRef: "origin:grants-gov-fetch-opportunity@v1", LocatorPattern: "/data/originalDueDateDesc", ConceptID: "tw:Instant", RoleID: "opportunity:deadline"},
		{ID: "mapping:grants-gov/opportunity-number@0.1", NativeTerm: "opportunityNumber", SchemaRef: "origin:grants-gov-fetch-opportunity@v1", LocatorPattern: "/data/opportunityNumber", ConceptID: "opportunity:GrantOpportunity", RoleID: "opportunity:identifier"},
		{ID: "mapping:grants-gov/opportunity-title@0.1", NativeTerm: "opportunityTitle", SchemaRef: "origin:grants-gov-fetch-opportunity@v1", LocatorPattern: "/data/opportunityTitle", ConceptID: "tw:Document", RoleID: "opportunity:title"},
		{ID: "mapping:grants-gov/owning-agency@0.1", NativeTerm: "synopsis.agencyCode", SchemaRef: "origin:grants-gov-fetch-opportunity@v1", LocatorPattern: "/data/synopsis/agencyCode", ConceptID: "opportunity:Funder", RoleID: "opportunity:funder"},
	}
}
