package dataplane

import (
	"bytes"
	"reflect"
	"testing"
)

func ontologyDigest(value byte) Digest {
	var digest Digest
	for i := range digest {
		digest[i] = value
	}
	return digest
}

func validFrame() Frame {
	packetA := ontologyDigest(1)
	packetB := ontologyDigest(2)
	return Frame{
		Version:             FrameVersion,
		UniverseID:          "tw:world-state",
		FrameType:           "world:IndicatorObservation",
		NativeKey:           "world-bank:CHL/SP.POP.TOTL/2024",
		CanonicalCandidates: []string{"world:observation/CHL/SP.POP.TOTL/2024"},
		Slots: []FrameSlot{
			{
				RoleID:        "world:indicator",
				Status:        "resolved",
				Cardinality:   "one",
				Values:        []TypedValue{{Type: "identifier", Lexical: "SP.POP.TOTL"}},
				PacketDigests: []Digest{packetA},
				MappingIDs:    []string{"mapping:world-bank/indicator@0.1"},
				Conflict:      "none",
			},
			{
				RoleID:        "world:value",
				Status:        "resolved",
				Cardinality:   "one",
				Values:        []TypedValue{{Type: "integer", Lexical: "19629590", Unit: OptionalText{Present: true, Value: "unit:person"}}},
				PacketDigests: []Digest{packetB},
				MappingIDs:    []string{"mapping:world-bank/value@0.1"},
				Conflict:      "none",
			},
		},
		Context: PacketContext{
			Jurisdiction: OptionalText{Present: true, Value: "geo:CL"},
			Language:     OptionalText{Present: true, Value: "en"},
		},
		Time:      FrameTime{ComposedAt: "2026-08-12T00:00:00Z", ValidFrom: OptionalText{Present: true, Value: "2024-01-01T00:00:00Z"}},
		Epistemic: FrameEpistemic{Lane: "attested_semantic", CompletenessMillionths: 1000000, ConflictStatus: "none"},
		Derivation: FrameDerivation{
			PacketDigests:          []Digest{packetA, packetB},
			ModuleSetDigest:        ontologyDigest(3),
			MappingIDs:             []string{"mapping:world-bank/indicator@0.1", "mapping:world-bank/value@0.1"},
			CompilerContractDigest: ontologyDigest(4),
			CompilerVersion:        "twirx-ontology@0.1",
		},
		Lifecycle: FrameLifecycle{State: "current"},
	}
}

func validMappingClaim() MappingClaim {
	return MappingClaim{
		Version:   MappingVersion,
		MappingID: "mapping:world-bank/value@0.1",
		Native: MappingNative{
			OriginID:       "api-worldbank-org",
			SchemaRef:      OptionalText{Present: true, Value: "origin:worldbank@v2"},
			Term:           "value",
			LocatorPattern: OptionalText{Present: true, Value: "/1/*/value"},
		},
		Semantic: MappingSemantic{ConceptID: "world:measurementValue", RoleID: OptionalText{Present: true, Value: "world:value"}},
		Relation: "contextual",
		Scope: MappingScope{
			UniverseID:    "tw:world-state",
			Jurisdictions: []string{},
			Languages:     []string{"en"},
			ConditionIDs:  []string{"condition:world-bank-indicator-unit"},
		},
		Status:                "reviewed",
		EvidencePacketDigests: []Digest{ontologyDigest(1)},
		ModuleID:              "tw:world-state@0.1.0",
		ReviewDecisionDigest:  OptionalDigest{Present: true, Value: ontologyDigest(5)},
	}
}

func validModule() OntologyModuleManifest {
	return OntologyModuleManifest{
		Version:              ModuleVersion,
		ModuleID:             "tw:world-state",
		SemanticVersion:      "0.1.0",
		Status:               "reviewed",
		Imports:              []string{"tw:kernel@0.1.0"},
		ConceptIDs:           []string{"world:Indicator", "world:IndicatorObservation"},
		FrameTypeIDs:         []string{"world:IndicatorObservation"},
		RoleIDs:              []string{"world:indicator", "world:value"},
		MappingClaimDigests:  []Digest{ontologyDigest(6)},
		QueryTemplateIDs:     []string{"query:world-state/compare-indicators@0.1"},
		VisualizationIDs:     []string{"view:world-state/time-series@0.1"},
		SourceArtifactDigest: ontologyDigest(7),
		ReviewDecisionDigest: OptionalDigest{Present: true, Value: ontologyDigest(8)},
	}
}

func validUniverse() SemanticUniverse {
	return SemanticUniverse{
		Version:             UniverseVersion,
		UniverseID:          "tw:world-state",
		SemanticVersion:     "0.1.0",
		Title:               "World State",
		ModuleIDs:           []string{"tw:kernel@0.1.0", "tw:world-state@0.1.0"},
		FrameTypeIDs:        []string{"world:IndicatorObservation"},
		SourceOriginIDs:     []string{"api-worldbank-org"},
		MappingClaimDigests: []Digest{ontologyDigest(6)},
		MaterializedViewIDs: []string{"view:world-state/latest@0.1"},
		QueryTemplateIDs:    []string{"query:world-state/compare-indicators@0.1"},
		VisualizationIDs:    []string{"view:world-state/time-series@0.1"},
		UpdatePolicyID:      "policy:world-state/disabled@0.1",
		EvaluationSuiteID:   "eval:world-state@0.1",
		ModuleSetDigest:     ontologyDigest(9),
		CompiledAt:          "2026-08-12T00:00:00Z",
	}
}

func TestOntologyObjectsRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		value     any
		marshal   func() ([]byte, error)
		unmarshal func([]byte) (any, error)
	}{
		{"frame", KindFrame, validFrame(), func() ([]byte, error) { return MarshalFrame(validFrame()) }, func(data []byte) (any, error) { return UnmarshalFrame(data) }},
		{"mapping", KindMappingClaim, validMappingClaim(), func() ([]byte, error) { return MarshalMappingClaim(validMappingClaim()) }, func(data []byte) (any, error) { return UnmarshalMappingClaim(data) }},
		{"module", KindOntologyModule, validModule(), func() ([]byte, error) { return MarshalOntologyModule(validModule()) }, func(data []byte) (any, error) { return UnmarshalOntologyModule(data) }},
		{"universe", KindUniverse, validUniverse(), func() ([]byte, error) { return MarshalSemanticUniverse(validUniverse()) }, func(data []byte) (any, error) { return UnmarshalSemanticUniverse(data) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := test.marshal()
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := test.unmarshal(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded, test.value) {
				t.Fatalf("round trip mismatch\nwant %#v\n got %#v", test.value, decoded)
			}
			if err := ValidateDocument(test.kind, encoded); err != nil {
				t.Fatal(err)
			}
			if err := ValidateDocument(test.kind, append(encoded, 0)); err == nil {
				t.Fatal("accepted trailing byte")
			}
		})
	}
}

func TestFrameRejectsUnprovedAndUnusedPackets(t *testing.T) {
	frame := validFrame()
	frame.Slots[0].PacketDigests[0] = ontologyDigest(10)
	if err := frame.Validate(); err == nil {
		t.Fatal("accepted slot packet outside derivation")
	}
	frame = validFrame()
	frame.Slots[1].PacketDigests = []Digest{ontologyDigest(1)}
	if err := frame.Validate(); err == nil {
		t.Fatal("accepted unused derivation packet")
	}
}

func TestFrameRejectsSemanticPromotionAndUnsortedValues(t *testing.T) {
	frame := validFrame()
	frame.Epistemic.Lane = "observed_native"
	if err := frame.Validate(); err == nil {
		t.Fatal("accepted mappings in observed-native frame")
	}
	frame = validFrame()
	frame.Slots[1].Cardinality = "many"
	frame.Slots[1].Values = []TypedValue{{Type: "integer", Lexical: "2"}, {Type: "integer", Lexical: "1"}}
	if err := frame.Validate(); err == nil {
		t.Fatal("accepted unsorted frame values")
	}
}

func TestMappingAndModuleReviewBoundaries(t *testing.T) {
	mapping := validMappingClaim()
	mapping.Status = "candidate"
	if err := mapping.Validate(); err == nil {
		t.Fatal("accepted candidate mapping with review decision")
	}
	mapping = validMappingClaim()
	mapping.ReviewDecisionDigest = OptionalDigest{}
	if err := mapping.Validate(); err == nil {
		t.Fatal("accepted reviewed mapping without decision")
	}
	module := validModule()
	module.Status = "draft"
	if err := module.Validate(); err == nil {
		t.Fatal("accepted draft module with review decision")
	}
	module = validModule()
	module.SemanticVersion = "01.0.0"
	if err := module.Validate(); err == nil {
		t.Fatal("accepted non-canonical semantic version")
	}
}

func TestFrameEncodingDeterministic(t *testing.T) {
	first, err := MarshalFrame(validFrame())
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarshalFrame(validFrame())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("same frame produced unequal canonical bytes")
	}
}
