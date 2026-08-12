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
	MaxWorldBankResponse = 64 << 20
	WorldBankOriginID    = "api-worldbank-org"
)

type worldBankMetadata struct {
	Page        int64  `json:"page"`
	Pages       int64  `json:"pages"`
	PerPage     int64  `json:"per_page"`
	Total       int64  `json:"total"`
	SourceID    string `json:"sourceid"`
	LastUpdated string `json:"lastupdated"`
}

type worldBankLabel struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type worldBankRecord struct {
	Indicator         worldBankLabel `json:"indicator"`
	Country           worldBankLabel `json:"country"`
	CountryISO3Code   string         `json:"countryiso3code"`
	Date              string         `json:"date"`
	Value             *json.Number   `json:"value"`
	Unit              string         `json:"unit"`
	ObservationStatus string         `json:"obs_status"`
	Decimal           int64          `json:"decimal"`
}

func CompileWorldBank(representation []byte, config Config) ([]RecordArtifact, error) {
	if len(representation) == 0 || len(representation) > MaxWorldBankResponse {
		return nil, fmt.Errorf("world bank importer: response size outside 1..%d", MaxWorldBankResponse)
	}
	if err := config.validate(representation); err != nil {
		return nil, err
	}
	if config.OriginID != WorldBankOriginID {
		return nil, fmt.Errorf("world bank importer: origin must be canonical Atlas ID %q", WorldBankOriginID)
	}
	policy := jsonbounded.Policy{MaxBytes: MaxWorldBankResponse, MaxDepth: 12, MaxScalarBytes: 1 << 20, MaxContainerEntries: 2000000, MaxTokens: 10000000}
	var parts []json.RawMessage
	if err := jsonbounded.Decode(representation, &parts, policy, false); err != nil {
		return nil, fmt.Errorf("world bank importer: %w", err)
	}
	if len(parts) != 2 {
		return nil, fmt.Errorf("world bank importer: expected metadata and records arrays")
	}
	var metadata worldBankMetadata
	if err := jsonbounded.Decode(parts[0], &metadata, policy, true); err != nil {
		return nil, fmt.Errorf("world bank importer: metadata: %w", err)
	}
	if metadata.Page < 1 || metadata.Pages < metadata.Page || metadata.PerPage < 1 || metadata.Total < 0 || metadata.SourceID == "" || metadata.LastUpdated == "" {
		return nil, fmt.Errorf("world bank importer: invalid pagination or source metadata")
	}
	var records []worldBankRecord
	if err := jsonbounded.Decode(parts[1], &records, policy, true); err != nil {
		return nil, fmt.Errorf("world bank importer: records: %w", err)
	}
	if int64(len(records)) > metadata.PerPage || int64(len(records)) > metadata.Total {
		return nil, fmt.Errorf("world bank importer: record count exceeds metadata")
	}
	outputs := make([]RecordArtifact, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for index, record := range records {
		output, err := compileWorldBankRecord(record, index, metadata.SourceID, config)
		if err != nil {
			return nil, fmt.Errorf("world bank importer: record %d: %w", index, err)
		}
		if _, exists := seen[output.NativeKey]; exists {
			return nil, fmt.Errorf("world bank importer: duplicate native key %q", output.NativeKey)
		}
		seen[output.NativeKey] = struct{}{}
		outputs = append(outputs, output)
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].NativeKey < outputs[j].NativeKey })
	return outputs, nil
}

func compileWorldBankRecord(record worldBankRecord, index int, sourceID string, config Config) (RecordArtifact, error) {
	if record.Indicator.ID == "" || record.CountryISO3Code == "" || record.Date == "" || record.Country.ID == "" {
		return RecordArtifact{}, fmt.Errorf("missing required native identifiers")
	}
	if len(record.Date) != 4 {
		return RecordArtifact{}, fmt.Errorf("date is not a four-digit source year")
	}
	if _, err := strconv.Atoi(record.Date); err != nil {
		return RecordArtifact{}, fmt.Errorf("invalid source year")
	}
	nativeKey := "world-bank:" + record.CountryISO3Code + "/" + record.Indicator.ID + "/" + record.Date
	subjectCandidates := []string{"world:observation/" + record.CountryISO3Code + "/" + record.Indicator.ID + "/" + record.Date}
	mappings := worldBankMappings()
	mappingByTerm := make(map[string]mappingDefinition, len(mappings))
	for _, mapping := range mappings {
		mappingByTerm[mapping.NativeTerm] = mapping
	}
	fields := []struct {
		term       string
		locator    string
		status     string
		native     string
		typed      *dataplane.TypedValue
		transforms []string
		kind       string
	}{
		{"countryiso3code", fmt.Sprintf("/1/%d/countryiso3code", index), "resolved", record.CountryISO3Code, &dataplane.TypedValue{Type: "identifier", Lexical: "geo:" + record.CountryISO3Code}, []string{"normalize:country-identifier@0.1"}, "claim"},
		{"date", fmt.Sprintf("/1/%d/date", index), "resolved", record.Date, &dataplane.TypedValue{Type: "date", Lexical: record.Date + "-01-01"}, []string{"transform:year-start-date@0.1"}, "claim"},
		{"indicator.id", fmt.Sprintf("/1/%d/indicator/id", index), "resolved", record.Indicator.ID, &dataplane.TypedValue{Type: "identifier", Lexical: record.Indicator.ID}, []string{}, "claim"},
		{"unit", fmt.Sprintf("/1/%d/unit", index), "not_provided", "", nil, []string{}, "claim"},
	}
	if record.Unit != "" {
		fields[3].status = "resolved"
		fields[3].native = record.Unit
		fields[3].typed = &dataplane.TypedValue{Type: "identifier", Lexical: "world-bank:unit/" + record.Unit}
	}
	valueNative := ""
	var valueTyped *dataplane.TypedValue
	valueStatus := "not_provided"
	valueTransforms := []string{}
	if record.Value != nil {
		valueNative = record.Value.String()
		valueStatus = "resolved"
		decimal, err := sourceNumberToDecimal(valueNative, record.Decimal)
		if err != nil {
			return RecordArtifact{}, err
		}
		valueTyped = &dataplane.TypedValue{Type: "decimal", Lexical: decimal}
		valueTransforms = []string{"transform:source-number-to-decimal@0.1"}
	}
	fields = append(fields, struct {
		term       string
		locator    string
		status     string
		native     string
		typed      *dataplane.TypedValue
		transforms []string
		kind       string
	}{"value", fmt.Sprintf("/1/%d/value", index), valueStatus, valueNative, valueTyped, valueTransforms, "measurement"})
	var packets []PacketArtifact
	for _, field := range fields {
		mapping := mappingByTerm[field.term]
		packet := dataplane.Packet{
			Version: dataplane.PacketVersion, Kind: field.kind,
			Subject:    dataplane.PacketSubject{Native: nativeKey, CanonicalCandidates: subjectCandidates},
			Predicate:  dataplane.PacketPredicate{Native: field.term, Semantic: dataplane.OptionalText{Present: true, Value: mapping.ConceptID}},
			Object:     dataplane.PacketObject{NativeStatus: field.status, NativeLexical: field.native, MediaType: dataplane.OptionalText{Present: true, Value: "application/json"}, Typed: field.typed},
			Context:    dataplane.PacketContext{Dimensions: []dataplane.ContextDimension{{Key: "world:sourceDatabase", Value: dataplane.TypedValue{Type: "identifier", Lexical: "world-bank:source/" + sourceID}}}, Jurisdiction: dataplane.OptionalText{Present: true, Value: "geo:" + record.CountryISO3Code}},
			Time:       dataplane.PacketTime{ObservedAt: config.ObservedAt, ValidFrom: dataplane.OptionalText{Present: true, Value: record.Date + "-01-01T00:00:00Z"}},
			Source:     dataplane.PacketSource{OriginID: config.OriginID, RepresentationDigest: config.RepresentationDigest, Locator: field.locator, NativeSchemaRef: dataplane.OptionalText{Present: true, Value: "origin:worldbank-indicators-v2"}},
			Derivation: dataplane.PacketDerivation{ObservationDigest: config.ObservationDigest, AdapterDigest: fixedDigest("tw.e4/world-bank-importer@0.1"), ExtractionPlanDigest: fixedDigest("tw.e4/world-bank-extraction/" + field.locator), TransformationIDs: field.transforms, MappingIDs: []string{mapping.ID}, SemanticClosureDigest: dataplane.OptionalDigest{Present: true, Value: config.ModuleSetDigest}, CompilerContractDigest: fixedDigest("tw.semantic-frame/0.1"), CompilerVersion: "twirx-universe-import@0.1"},
			Epistemic:  dataplane.PacketEpistemic{Lane: "provisional_semantic", ExtractionStatus: "deterministic", MappingStatus: "candidate", AuthorityClass: "provider-operated-official-api", FreshnessStatus: freshness(config.EvidenceClass)},
			Lifecycle:  dataplane.PacketLifecycle{State: "current"}, Retention: "public_versioned", Disclosure: "public",
		}
		artifact, err := compilePacket(packet)
		if err != nil {
			return RecordArtifact{}, err
		}
		packets = append(packets, artifact)
	}
	mappingArtifacts, err := mappingsForPackets(config.OriginID, "tw:world-state", "tw:world-state@0.1.0", mappings, packets)
	if err != nil {
		return RecordArtifact{}, err
	}
	byTerm := packetByNativePredicate(packets)
	frameSlots := []dataplane.FrameSlot{
		frameSlot("world:country", byTerm["countryiso3code"], mappingByTerm["countryiso3code"], fields[0].typed, fields[0].status),
		frameSlot("world:indicator", byTerm["indicator.id"], mappingByTerm["indicator.id"], fields[2].typed, fields[2].status),
		frameSlot("world:observedValue", byTerm["value"], mappingByTerm["value"], valueTyped, valueStatus),
		frameSlot("world:period", byTerm["date"], mappingByTerm["date"], fields[1].typed, fields[1].status),
		frameSlot("world:unit", byTerm["unit"], mappingByTerm["unit"], fields[3].typed, fields[3].status),
	}
	allMappingIDs := make([]string, len(mappings))
	for i := range mappings {
		allMappingIDs[i] = mappings[i].ID
	}
	sort.Strings(allMappingIDs)
	frame := dataplane.Frame{
		Version: dataplane.FrameVersion, UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", NativeKey: nativeKey, CanonicalCandidates: subjectCandidates,
		Slots: frameSlots, Context: dataplane.PacketContext{Jurisdiction: dataplane.OptionalText{Present: true, Value: "geo:" + record.CountryISO3Code}}, Time: dataplane.FrameTime{ComposedAt: config.ObservedAt, ValidFrom: dataplane.OptionalText{Present: true, Value: record.Date + "-01-01T00:00:00Z"}},
		Epistemic:  dataplane.FrameEpistemic{Lane: "provisional_semantic", CompletenessMillionths: 800000, ConflictStatus: "none"},
		Derivation: dataplane.FrameDerivation{PacketDigests: sortedPacketDigests(packets), ModuleSetDigest: config.ModuleSetDigest, MappingIDs: allMappingIDs, CompilerContractDigest: fixedDigest("tw.semantic-frame/0.1"), CompilerVersion: "twirx-universe-import@0.1"},
		Lifecycle:  dataplane.FrameLifecycle{State: "current"},
	}
	return finishRecord(nativeKey, config.EvidenceClass, config.RepresentationDigest, packets, mappingArtifacts, frame)
}

func worldBankMappings() []mappingDefinition {
	return []mappingDefinition{
		{ID: "mapping:world-bank/country-code@0.1", NativeTerm: "countryiso3code", SchemaRef: "origin:worldbank-indicators-v2", LocatorPattern: "/1/*/countryiso3code", ConceptID: "world:Economy", RoleID: "world:country"},
		{ID: "mapping:world-bank/year@0.1", NativeTerm: "date", SchemaRef: "origin:worldbank-indicators-v2", LocatorPattern: "/1/*/date", ConceptID: "tw:Instant", RoleID: "world:period"},
		{ID: "mapping:world-bank/indicator@0.1", NativeTerm: "indicator.id", SchemaRef: "origin:worldbank-indicators-v2", LocatorPattern: "/1/*/indicator/id", ConceptID: "world:Indicator", RoleID: "world:indicator"},
		{ID: "mapping:world-bank/unit@0.1", NativeTerm: "unit", SchemaRef: "origin:worldbank-indicators-v2", LocatorPattern: "/1/*/unit", ConceptID: "world:Unit", RoleID: "world:unit"},
		{ID: "mapping:world-bank/value@0.1", NativeTerm: "value", SchemaRef: "origin:worldbank-indicators-v2", LocatorPattern: "/1/*/value", ConceptID: "world:Measure", RoleID: "world:observedValue"},
	}
}

func frameSlot(role string, packet PacketArtifact, mapping mappingDefinition, value *dataplane.TypedValue, status string) dataplane.FrameSlot {
	values := []dataplane.TypedValue{}
	if value != nil {
		values = append(values, *value)
	}
	return dataplane.FrameSlot{RoleID: role, Status: status, Cardinality: "one", Values: values, PacketDigests: []dataplane.Digest{packet.Digest}, MappingIDs: []string{mapping.ID}, Conflict: "none"}
}

func sourceNumberToDecimal(value string, decimalPlaces int64) (string, error) {
	if decimalPlaces < 0 || decimalPlaces > 18 {
		return "", fmt.Errorf("unsupported source decimal places")
	}
	if strings.ContainsAny(value, "eE") {
		return "", fmt.Errorf("exponent-form source number is not accepted")
	}
	if strings.Contains(value, ".") {
		parts := strings.Split(value, ".")
		if len(parts) != 2 || parts[1] == "" {
			return "", fmt.Errorf("invalid source decimal")
		}
		return value, nil
	}
	return value + ".0", nil
}

func freshness(evidenceClass string) string {
	if evidenceClass == "current_observation" {
		return "current"
	}
	return "stale"
}
