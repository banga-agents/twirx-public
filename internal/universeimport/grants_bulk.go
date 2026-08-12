package universeimport

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	GrantsProjectionFormat = "tw.e4-opportunity-approved-projection/0.1"
	MaxGrantsProjection    = 512 << 20
)

var (
	grantsLocatorPattern = regexp.MustCompile(`^/Grants/(OpportunitySynopsisDetail_1_0|OpportunityForecastDetail_1_0)\[[0-9]{1,6}\]$`)
	grantsDatePattern    = regexp.MustCompile(`^(0[1-9]|1[0-2])(0[1-9]|[12][0-9]|3[01])[0-9]{4}$`)
	grantsDigitsPattern  = regexp.MustCompile(`^[0-9]{1,20}$`)
)

type grantsBulkProjection struct {
	Format  string             `json:"format"`
	Records []grantsBulkRecord `json:"records"`
}

type grantsBulkRecord struct {
	RecordKind                            string   `json:"record_kind"`
	SourceLocator                         string   `json:"source_locator"`
	OpportunityID                         string   `json:"opportunity_id"`
	OpportunityNumber                     string   `json:"opportunity_number"`
	OpportunityTitle                      string   `json:"opportunity_title"`
	OpportunityCategory                   string   `json:"opportunity_category"`
	CategoryOfFundingActivity             []string `json:"category_of_funding_activity"`
	CFDANumbers                           []string `json:"cfda_numbers"`
	EligibleApplicants                    []string `json:"eligible_applicants"`
	AdditionalInformationOnEligibility    string   `json:"additional_information_on_eligibility"`
	AgencyCode                            string   `json:"agency_code"`
	AgencyName                            string   `json:"agency_name"`
	PostDate                              string   `json:"post_date"`
	CloseDate                             string   `json:"close_date"`
	LastUpdatedDate                       string   `json:"last_updated_date"`
	EstimatedSynopsisPostDate             string   `json:"estimated_synopsis_post_date"`
	EstimatedSynopsisCloseDate            string   `json:"estimated_synopsis_close_date"`
	EstimatedSynopsisCloseDateExplanation string   `json:"estimated_synopsis_close_date_explanation"`
	EstimatedAwardDate                    string   `json:"estimated_award_date"`
	EstimatedProjectStartDate             string   `json:"estimated_project_start_date"`
	ExpectedNumberOfAwards                string   `json:"expected_number_of_awards"`
	EstimatedTotalProgramFunding          string   `json:"estimated_total_program_funding"`
	AwardCeiling                          string   `json:"award_ceiling"`
	AwardFloor                            string   `json:"award_floor"`
	ArchiveDate                           string   `json:"archive_date"`
	CostSharingOrMatchingRequirement      string   `json:"cost_sharing_or_matching_requirement"`
	Version                               string   `json:"version"`
}

// CompileGrantsBulkProjection compiles only the already-created approved-field
// projection. The packet source digest remains the original XML
// representation, while config binds and verifies the projection bytes and
// its manifest-last observation/derivation identity. This function has no
// network or archive extraction capability.
func CompileGrantsBulkProjection(projection []byte, sourceRepresentationDigest dataplane.Digest, config Config) ([]RecordArtifact, error) {
	records, _, err := CompileGrantsBulkProjectionWindow(projection, sourceRepresentationDigest, config, 0, 250000)
	return records, err
}

// CompileGrantsBulkProjectionWindow compiles a deterministic consecutive
// window while still decoding and validating the complete approved
// projection. It bounds compiler working memory without weakening whole-file
// ordering, uniqueness, source identity, or unknown-field checks.
func CompileGrantsBulkProjectionWindow(projection []byte, sourceRepresentationDigest dataplane.Digest, config Config, start, maximum uint64) ([]RecordArtifact, uint64, error) {
	if len(projection) == 0 || len(projection) > MaxGrantsProjection {
		return nil, 0, errors.New("grants bulk importer: projection exceeds its bound")
	}
	if err := config.validate(projection); err != nil {
		return nil, 0, err
	}
	if config.OriginID != GrantsGovOriginID || sourceRepresentationDigest == (dataplane.Digest{}) || maximum == 0 || maximum > 250000 {
		return nil, 0, errors.New("grants bulk importer: exact source origin, XML representation digest, and bounded window are required")
	}
	policy := jsonbounded.Policy{MaxBytes: MaxGrantsProjection, MaxDepth: 8, MaxScalarBytes: 18000, MaxContainerEntries: 4000000, MaxTokens: 16000000}
	var decoded grantsBulkProjection
	if err := jsonbounded.Decode(projection, &decoded, policy, true); err != nil {
		return nil, 0, fmt.Errorf("grants bulk importer: decode approved projection: %w", err)
	}
	if decoded.Format != GrantsProjectionFormat || len(decoded.Records) == 0 || len(decoded.Records) > 250000 {
		return nil, 0, errors.New("grants bulk importer: invalid projection format or record count")
	}
	total := uint64(len(decoded.Records))
	if start >= total {
		return nil, total, errors.New("grants bulk importer: window starts outside projection")
	}
	end := start + maximum
	if end > total {
		end = total
	}
	result := make([]RecordArtifact, 0, end-start)
	priorID := ""
	for index, source := range decoded.Records {
		if err := validateBulkRecord(source); err != nil {
			return nil, 0, err
		}
		if source.OpportunityID <= priorID {
			return nil, 0, errors.New("grants bulk importer: projection records are not sorted and uniquely identified")
		}
		if uint64(index) >= start && uint64(index) < end {
			record, err := compileBulkGrant(source, sourceRepresentationDigest, config)
			if err != nil {
				return nil, 0, err
			}
			result = append(result, record)
		}
		priorID = source.OpportunityID
	}
	return result, total, nil
}

type bulkGrantField struct {
	term        string
	locator     string
	status      string
	native      string
	typed       *dataplane.TypedValue
	transforms  []string
	kind        string
	role        string
	concept     string
	cardinality string
}

func compileBulkGrant(source grantsBulkRecord, sourceRepresentationDigest dataplane.Digest, config Config) (RecordArtifact, error) {
	base := source.SourceLocator
	nativeKey := "grants-gov:opportunity/" + source.OpportunityID
	subjectCandidates := []string{"opportunity:grants-gov/" + source.OpportunityID}
	fields := []bulkGrantField{
		{term: "OpportunityNumber", locator: base + "/OpportunityNumber", status: "resolved", native: source.OpportunityNumber, typed: &dataplane.TypedValue{Type: "identifier", Lexical: source.OpportunityNumber}, kind: "offer", role: "opportunity:identifier", concept: "opportunity:GrantOpportunity", cardinality: "one"},
		{term: "OpportunityTitle", locator: base + "/OpportunityTitle", status: "resolved", native: source.OpportunityTitle, typed: &dataplane.TypedValue{Type: "text", Lexical: source.OpportunityTitle}, kind: "offer", role: "opportunity:title", concept: "tw:Document", cardinality: "one"},
		{term: "AgencyCode", locator: base + "/AgencyCode", status: "resolved", native: source.AgencyCode, typed: &dataplane.TypedValue{Type: "identifier", Lexical: "grants-gov:agency/" + source.AgencyCode}, transforms: []string{"normalize:grants-agency-identifier@0.1"}, kind: "relationship", role: "opportunity:funder", concept: "opportunity:Funder", cardinality: "one"},
	}
	for index, code := range source.EligibleApplicants {
		fields = append(fields, bulkGrantField{term: "EligibleApplicants", locator: fmt.Sprintf("%s/EligibleApplicants[%d]", base, index), status: "resolved", native: code, typed: &dataplane.TypedValue{Type: "identifier", Lexical: "grants-gov:applicant-type/" + code}, transforms: []string{"normalize:grants-applicant-type@0.1"}, kind: "claim", role: "opportunity:applicantClass", concept: "opportunity:ApplicantClass", cardinality: "many"})
	}
	if len(source.EligibleApplicants) == 0 {
		fields = append(fields, bulkGrantField{term: "EligibleApplicants", locator: base + "/EligibleApplicants", status: "not_provided", kind: "claim", role: "opportunity:applicantClass", concept: "opportunity:ApplicantClass", cardinality: "many"})
	}
	if source.AdditionalInformationOnEligibility != "" {
		// The approved field contains contact-like data in the admitted real
		// corpus. Preserve the fact and exact locator as a withheld source
		// statement without copying its lexical value into a public artifact.
		fields = append(fields, bulkGrantField{term: "AdditionalInformationOnEligibility", locator: base + "/AdditionalInformationOnEligibility", status: "withheld", kind: "claim", role: "opportunity:eligibilityText", concept: "opportunity:EligibilityRule", cardinality: "many"})
	} else {
		fields = append(fields, bulkGrantField{term: "AdditionalInformationOnEligibility", locator: base + "/AdditionalInformationOnEligibility", status: "not_provided", kind: "claim", role: "opportunity:eligibilityText", concept: "opportunity:EligibilityRule", cardinality: "many"})
	}
	for index, code := range source.CategoryOfFundingActivity {
		fields = append(fields, bulkGrantField{term: "CategoryOfFundingActivity", locator: fmt.Sprintf("%s/CategoryOfFundingActivity[%d]", base, index), status: "resolved", native: code, typed: &dataplane.TypedValue{Type: "identifier", Lexical: "grants-gov:funding-category/" + code}, transforms: []string{"normalize:grants-funding-category@0.1"}, kind: "claim", role: "opportunity:topic", concept: "tw:Topic", cardinality: "many"})
	}
	if len(source.CategoryOfFundingActivity) == 0 {
		fields = append(fields, bulkGrantField{term: "CategoryOfFundingActivity", locator: base + "/CategoryOfFundingActivity", status: "not_provided", kind: "claim", role: "opportunity:topic", concept: "tw:Topic", cardinality: "many"})
	}
	for _, amount := range []struct{ term, native, role string }{{"AwardCeiling", source.AwardCeiling, "opportunity:maximumAmount"}, {"AwardFloor", source.AwardFloor, "opportunity:minimumAmount"}} {
		if amount.native == "" {
			fields = append(fields, bulkGrantField{term: amount.term, locator: base + "/" + amount.term, status: "not_provided", kind: "offer", role: amount.role, concept: "tw:Price", cardinality: "one"})
			continue
		}
		decimal, ok := canonicalUnsignedDecimal(amount.native)
		field := bulkGrantField{term: amount.term, locator: base + "/" + amount.term, status: "resolved", native: amount.native, kind: "offer", role: amount.role, concept: "tw:Price", cardinality: "one"}
		if ok {
			field.typed = &dataplane.TypedValue{Type: "decimal", Lexical: decimal}
			field.transforms = []string{"transform:source-number-to-decimal@0.1"}
		} else {
			field.status = "unresolved"
		}
		fields = append(fields, field)
	}
	deadlineTerm, deadline := "CloseDate", source.CloseDate
	if deadline == "" {
		deadlineTerm, deadline = "EstimatedSynopsisCloseDate", source.EstimatedSynopsisCloseDate
	}
	if deadline != "" {
		// Grants.gov publishes a date without an exact closing instant. Preserve
		// it natively and keep the deadline slot unresolved rather than inventing
		// a timezone or end-of-day instant.
		fields = append(fields, bulkGrantField{term: deadlineTerm, locator: base + "/" + deadlineTerm, status: "resolved", native: deadline, kind: "claim", role: "opportunity:deadline", concept: "tw:Instant", cardinality: "one"})
	} else {
		fields = append(fields, bulkGrantField{term: deadlineTerm, locator: base + "/" + deadlineTerm, status: "not_provided", kind: "claim", role: "opportunity:deadline", concept: "tw:Instant", cardinality: "one"})
	}
	definitions := make([]mappingDefinition, 0, len(fields))
	seenDefinitions := make(map[string]bool)
	for _, field := range fields {
		if seenDefinitions[field.term] {
			continue
		}
		seenDefinitions[field.term] = true
		definitions = append(definitions, mappingDefinition{ID: "mapping:grants-gov-bulk/" + bulkMappingName(field.term) + "@0.1", NativeTerm: field.term, SchemaRef: "origin:grants-gov-bulk-v2@20260811", LocatorPattern: "/Grants/" + source.RecordKind + "[*]/" + field.term + bulkLocatorSuffix(field.cardinality), ConceptID: field.concept, RoleID: field.role})
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].ID < definitions[j].ID })
	definitionByTerm := make(map[string]mappingDefinition, len(definitions))
	for _, definition := range definitions {
		definitionByTerm[definition.NativeTerm] = definition
	}
	packets := make([]PacketArtifact, 0, len(fields))
	for _, field := range fields {
		mapping := definitionByTerm[field.term]
		language := dataplane.OptionalText{}
		if field.typed != nil && field.typed.Type == "text" {
			language = dataplane.OptionalText{Present: true, Value: "en"}
		}
		packet := dataplane.Packet{Version: dataplane.PacketVersion, Kind: field.kind,
			Subject:    dataplane.PacketSubject{Native: nativeKey, CanonicalCandidates: subjectCandidates},
			Predicate:  dataplane.PacketPredicate{Native: field.term, Semantic: dataplane.OptionalText{Present: true, Value: mapping.ConceptID}},
			Object:     dataplane.PacketObject{NativeStatus: field.status, NativeLexical: field.native, MediaType: dataplane.OptionalText{Present: true, Value: "application/xml"}, Language: language, Typed: field.typed},
			Context:    dataplane.PacketContext{Jurisdiction: dataplane.OptionalText{Present: true, Value: "geo:US"}, Language: dataplane.OptionalText{Present: true, Value: "en"}},
			Time:       dataplane.PacketTime{ObservedAt: config.ObservedAt},
			Source:     dataplane.PacketSource{OriginID: config.OriginID, RepresentationDigest: sourceRepresentationDigest, Locator: field.locator, NativeSchemaRef: dataplane.OptionalText{Present: true, Value: "origin:grants-gov-bulk-v2@20260811"}},
			Derivation: dataplane.PacketDerivation{ObservationDigest: config.ObservationDigest, AdapterDigest: fixedDigest("tw.e4/grants-gov-bulk-importer@0.1"), ExtractionPlanDigest: fixedDigest("tw.e4/grants-gov-bulk-extraction/" + field.locator), TransformationIDs: field.transforms, MappingIDs: []string{mapping.ID}, SemanticClosureDigest: dataplane.OptionalDigest{Present: true, Value: config.ModuleSetDigest}, CompilerContractDigest: fixedDigest("tw.semantic-frame/0.1"), CompilerVersion: "twirx-universe-import@0.1"},
			Epistemic:  dataplane.PacketEpistemic{Lane: "provisional_semantic", ExtractionStatus: "deterministic", MappingStatus: "candidate", AuthorityClass: "provider-operated-official-bulk-extract", FreshnessStatus: freshness(config.EvidenceClass)},
			Lifecycle:  dataplane.PacketLifecycle{State: "current"}, Retention: "public_versioned", Disclosure: "public"}
		artifact, err := compilePacket(packet)
		if err != nil {
			return RecordArtifact{}, err
		}
		packets = append(packets, artifact)
	}
	mappings, err := mappingsForPackets(config.OriginID, "tw:opportunity", "tw:opportunity@0.1.0", definitions, packets)
	if err != nil {
		return RecordArtifact{}, err
	}
	byTerm := packetsByNativePredicate(packets)
	slots := make([]dataplane.FrameSlot, 0, 9)
	for _, item := range []struct{ role, term, cardinality string }{{"opportunity:funder", "AgencyCode", "one"}, {"opportunity:identifier", "OpportunityNumber", "one"}, {"opportunity:title", "OpportunityTitle", "one"}, {"opportunity:applicantClass", "EligibleApplicants", "many"}, {"opportunity:eligibilityText", "AdditionalInformationOnEligibility", "many"}, {"opportunity:topic", "CategoryOfFundingActivity", "many"}, {"opportunity:maximumAmount", "AwardCeiling", "one"}, {"opportunity:minimumAmount", "AwardFloor", "one"}} {
		matched := byTerm[item.term]
		if len(matched) == 0 {
			slots = append(slots, dataplane.FrameSlot{RoleID: item.role, Status: "not_provided", Cardinality: item.cardinality, Values: []dataplane.TypedValue{}, PacketDigests: []dataplane.Digest{}, MappingIDs: []string{}, Conflict: "none"})
			continue
		}
		status := matched[0].Packet.Object.NativeStatus
		values := typedValues(matched)
		if status == "resolved" && len(values) == 0 {
			status = "unresolved"
		}
		slot, err := frameSlotMany(item.role, status, item.cardinality, definitionByTerm[item.term].ID, matched, values)
		if err != nil {
			return RecordArtifact{}, err
		}
		slots = append(slots, slot)
	}
	deadlinePackets := byTerm[deadlineTerm]
	deadlineStatus := deadlinePackets[0].Packet.Object.NativeStatus
	if deadlineStatus == "resolved" {
		deadlineStatus = "unresolved"
	}
	slot, err := frameSlotMany("opportunity:deadline", deadlineStatus, "one", definitionByTerm[deadlineTerm].ID, deadlinePackets, nil)
	if err != nil {
		return RecordArtifact{}, err
	}
	slots = append(slots, slot)
	sort.Slice(slots, func(i, j int) bool { return slots[i].RoleID < slots[j].RoleID })
	mappingIDs := make([]string, len(definitions))
	for i := range definitions {
		mappingIDs[i] = definitions[i].ID
	}
	sort.Strings(mappingIDs)
	resolved := uint64(0)
	for _, slot := range slots {
		if slot.Status == "resolved" {
			resolved++
		}
	}
	frame := dataplane.Frame{Version: dataplane.FrameVersion, UniverseID: "tw:opportunity", FrameType: "opportunity:GrantOpportunity", NativeKey: nativeKey, CanonicalCandidates: subjectCandidates, Slots: slots,
		Context: dataplane.PacketContext{Jurisdiction: dataplane.OptionalText{Present: true, Value: "geo:US"}, Language: dataplane.OptionalText{Present: true, Value: "en"}}, Time: dataplane.FrameTime{ComposedAt: config.ObservedAt},
		Epistemic:  dataplane.FrameEpistemic{Lane: "provisional_semantic", CompletenessMillionths: resolved * 1000000 / uint64(len(slots)), ConflictStatus: "none"},
		Derivation: dataplane.FrameDerivation{PacketDigests: sortedPacketDigests(packets), ModuleSetDigest: config.ModuleSetDigest, MappingIDs: mappingIDs, CompilerContractDigest: fixedDigest("tw.semantic-frame/0.1"), CompilerVersion: "twirx-universe-import@0.1"}, Lifecycle: dataplane.FrameLifecycle{State: "current"}}
	return finishRecord(nativeKey, config.EvidenceClass, sourceRepresentationDigest, packets, mappings, frame)
}

func validateBulkRecord(record grantsBulkRecord) error {
	if !grantsLocatorPattern.MatchString(record.SourceLocator) || record.RecordKind != "OpportunitySynopsisDetail_1_0" && record.RecordKind != "OpportunityForecastDetail_1_0" || record.OpportunityID == "" || record.OpportunityNumber == "" || record.OpportunityTitle == "" || record.AgencyCode == "" || record.AgencyName == "" || record.LastUpdatedDate == "" {
		return errors.New("grants bulk importer: record identity or source locator is invalid")
	}
	if !grantsDigitsPattern.MatchString(record.OpportunityID) {
		return errors.New("grants bulk importer: OpportunityID is malformed")
	}
	if containsControl(record.OpportunityNumber) || len(record.OpportunityNumber) > 40 {
		return errors.New("grants bulk importer: OpportunityNumber is malformed")
	}
	if containsControl(record.AgencyCode) || len(record.AgencyCode) > 255 {
		return errors.New("grants bulk importer: AgencyCode is malformed")
	}
	if len(record.OpportunityTitle) > 255 || len(record.AgencyName) > 255 {
		return errors.New("grants bulk importer: title or agency name exceeds its bound")
	}
	if len(record.EligibleApplicants) > 32 {
		return errors.New("grants bulk importer: EligibleApplicants cardinality exceeds 32")
	}
	if len(record.CategoryOfFundingActivity) > 32 {
		return errors.New("grants bulk importer: CategoryOfFundingActivity cardinality exceeds 32")
	}
	if len(record.CFDANumbers) > 64 {
		return errors.New("grants bulk importer: CFDANumbers cardinality exceeds 64")
	}
	for _, value := range []string{record.OpportunityTitle, record.AgencyName, record.AdditionalInformationOnEligibility, record.EstimatedSynopsisCloseDateExplanation, record.CostSharingOrMatchingRequirement, record.Version} {
		if len(value) > 18000 || strings.ContainsRune(value, '\x00') {
			return errors.New("grants bulk importer: text field is malformed")
		}
	}
	for _, values := range [][]string{record.CategoryOfFundingActivity, record.CFDANumbers, record.EligibleApplicants} {
		for _, value := range values {
			if value == "" || len(value) > 255 || strings.ContainsAny(value, "\t\r\n\x00") {
				return errors.New("grants bulk importer: repeated field is malformed")
			}
		}
	}
	for _, value := range []string{record.PostDate, record.CloseDate, record.LastUpdatedDate, record.EstimatedSynopsisPostDate, record.EstimatedSynopsisCloseDate, record.EstimatedAwardDate, record.EstimatedProjectStartDate, record.ArchiveDate} {
		if value != "" && !grantsDatePattern.MatchString(value) {
			return errors.New("grants bulk importer: source date is malformed")
		}
	}
	for _, value := range []string{record.ExpectedNumberOfAwards, record.EstimatedTotalProgramFunding, record.AwardCeiling, record.AwardFloor} {
		if value != "" && !grantsDigitsPattern.MatchString(value) {
			return errors.New("grants bulk importer: source numeric field is malformed")
		}
	}
	return nil
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func bulkMappingName(term string) string {
	var builder strings.Builder
	for index, character := range term {
		if character >= 'A' && character <= 'Z' {
			if index > 0 {
				builder.WriteByte('-')
			}
			builder.WriteByte(byte(character - 'A' + 'a'))
		} else {
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func bulkLocatorSuffix(cardinality string) string {
	if cardinality == "many" {
		return "[*]"
	}
	return ""
}
