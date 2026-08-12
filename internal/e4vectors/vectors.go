// Package e4vectors constructs the E4.0 Ontology Fabric cross-implementation
// conformance corpus. It is test support, not runtime authority.
package e4vectors

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

type Vector struct {
	Name   string
	Kind   string
	Valid  bool
	Reason string
	Data   []byte
}

func digest(value byte) dataplane.Digest {
	var result dataplane.Digest
	for i := range result {
		result[i] = value
	}
	return result
}

func Corpus() ([]Vector, error) {
	packetA, packetB := digest(1), digest(2)
	frame := dataplane.Frame{
		Version: dataplane.FrameVersion, UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", NativeKey: "world-bank:CL/POP/2024",
		CanonicalCandidates: []string{"world:observation/CL/POP/2024"},
		Derivation:          dataplane.FrameDerivation{PacketDigests: []dataplane.Digest{packetA, packetB}, ModuleSetDigest: digest(3), MappingIDs: []string{"mapping:world/a@0.1", "mapping:world/b@0.1"}, CompilerContractDigest: digest(4), CompilerVersion: "twirx-ontology@0.1"},
		Slots: []dataplane.FrameSlot{
			{RoleID: "world:indicator", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "identifier", Lexical: "POP"}}, PacketDigests: []dataplane.Digest{packetA}, MappingIDs: []string{"mapping:world/a@0.1"}, Conflict: "none"},
			{RoleID: "world:value", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "integer", Lexical: "19629590", Unit: dataplane.OptionalText{Present: true, Value: "unit:person"}}}, PacketDigests: []dataplane.Digest{packetB}, MappingIDs: []string{"mapping:world/b@0.1"}, Conflict: "none"},
		},
		Context: dataplane.PacketContext{Jurisdiction: dataplane.OptionalText{Present: true, Value: "geo:CL"}, Language: dataplane.OptionalText{Present: true, Value: "en"}},
		Time:    dataplane.FrameTime{ComposedAt: "2026-08-12T00:00:00Z"}, Epistemic: dataplane.FrameEpistemic{Lane: "attested_semantic", CompletenessMillionths: 1000000, ConflictStatus: "none"}, Lifecycle: dataplane.FrameLifecycle{State: "current"},
	}
	observedFrame := frame
	observedFrame.Derivation.MappingIDs = nil
	observedFrame.Slots = append([]dataplane.FrameSlot(nil), frame.Slots...)
	for i := range observedFrame.Slots {
		observedFrame.Slots[i].MappingIDs = nil
	}
	observedFrame.Epistemic.Lane = "observed_native"

	candidateMapping := dataplane.MappingClaim{
		Version: dataplane.MappingVersion, MappingID: "mapping:world/value@0.1",
		Native:   dataplane.MappingNative{OriginID: "api-worldbank-org", SchemaRef: dataplane.OptionalText{Present: true, Value: "origin:worldbank@v2"}, Term: "value", LocatorPattern: dataplane.OptionalText{Present: true, Value: "/1/*/value"}},
		Semantic: dataplane.MappingSemantic{ConceptID: "world:measurementValue", RoleID: dataplane.OptionalText{Present: true, Value: "world:value"}}, Relation: "candidate",
		Scope: dataplane.MappingScope{UniverseID: "tw:world-state", Languages: []string{"en"}}, Status: "candidate", EvidencePacketDigests: []dataplane.Digest{packetB}, ModuleID: "tw:world-state@0.1.0",
	}
	reviewedMapping := candidateMapping
	reviewedMapping.Relation = "contextual"
	reviewedMapping.Status = "reviewed"
	reviewedMapping.ReviewDecisionDigest = dataplane.OptionalDigest{Present: true, Value: digest(5)}

	draftModule := dataplane.OntologyModuleManifest{
		Version: dataplane.ModuleVersion, ModuleID: "tw:world-state", SemanticVersion: "0.1.0", Status: "draft", Imports: []string{"tw:kernel@0.1.0"},
		ConceptIDs: []string{"world:Indicator", "world:IndicatorObservation"}, FrameTypeIDs: []string{"world:IndicatorObservation"}, RoleIDs: []string{"world:indicator", "world:value"},
		MappingClaimDigests: []dataplane.Digest{}, QueryTemplateIDs: []string{"query:world/compare@0.1"}, VisualizationIDs: []string{"view:world/time-series@0.1"}, SourceArtifactDigest: digest(6),
	}
	reviewedModule := draftModule
	reviewedModule.Status = "reviewed"
	reviewedModule.ReviewDecisionDigest = dataplane.OptionalDigest{Present: true, Value: digest(7)}

	universe := dataplane.SemanticUniverse{
		Version: dataplane.UniverseVersion, UniverseID: "tw:world-state", SemanticVersion: "0.1.0", Title: "World State",
		ModuleIDs: []string{"tw:kernel@0.1.0", "tw:world-state@0.1.0"}, FrameTypeIDs: []string{"world:IndicatorObservation"}, SourceOriginIDs: []string{"api-worldbank-org"},
		MappingClaimDigests: []dataplane.Digest{}, MaterializedViewIDs: []string{}, QueryTemplateIDs: []string{"query:world/compare@0.1"}, VisualizationIDs: []string{"view:world/time-series@0.1"},
		UpdatePolicyID: "policy:world/disabled@0.1", EvaluationSuiteID: "eval:world@0.1", ModuleSetDigest: digest(8), CompiledAt: "2026-08-12T00:00:00Z",
	}

	types := []struct {
		name string
		kind string
		make func() ([]byte, error)
	}{
		{"frame-attested", dataplane.KindFrame, func() ([]byte, error) { return dataplane.MarshalFrame(frame) }},
		{"frame-observed", dataplane.KindFrame, func() ([]byte, error) { return dataplane.MarshalFrame(observedFrame) }},
		{"mapping-candidate", dataplane.KindMappingClaim, func() ([]byte, error) { return dataplane.MarshalMappingClaim(candidateMapping) }},
		{"mapping-reviewed", dataplane.KindMappingClaim, func() ([]byte, error) { return dataplane.MarshalMappingClaim(reviewedMapping) }},
		{"module-draft", dataplane.KindOntologyModule, func() ([]byte, error) { return dataplane.MarshalOntologyModule(draftModule) }},
		{"module-reviewed", dataplane.KindOntologyModule, func() ([]byte, error) { return dataplane.MarshalOntologyModule(reviewedModule) }},
		{"universe-world-state", dataplane.KindUniverse, func() ([]byte, error) { return dataplane.MarshalSemanticUniverse(universe) }},
	}
	var vectors []Vector
	valid := make(map[string][]byte)
	for _, item := range types {
		data, err := item.make()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", item.name, err)
		}
		valid[item.name] = data
		vectors = append(vectors,
			Vector{Name: item.name, Kind: item.kind, Valid: true, Reason: "canonical valid E4.0 document", Data: data},
			Vector{Name: item.name + "-trailing", Kind: item.kind, Reason: "trailing byte", Data: append(append([]byte(nil), data...), 0)},
			Vector{Name: item.name + "-truncated", Kind: item.kind, Reason: "truncated document", Data: append([]byte(nil), data[:len(data)-1]...)},
			mutated(item.name+"-wrong-version", item.kind, data, []byte("tw."), []byte("tx."), "unsupported version"),
		)
	}

	vectors = append(vectors,
		mutatedNth("frame-slot-packet-outside-derivation", dataplane.KindFrame, valid["frame-attested"], bytes.Repeat([]byte{2}, 32), bytes.Repeat([]byte{9}, 32), 2, "slot packet is outside derivation"),
		mutated("frame-invalid-conflict", dataplane.KindFrame, valid["frame-attested"], []byte("none"), []byte("xxxx"), "unknown frame conflict status"),
		mutated("mapping-invalid-relation", dataplane.KindMappingClaim, valid["mapping-reviewed"], []byte("contextual"), []byte("xxxxxxxxxx"), "unknown mapping relation"),
		mutated("module-invalid-semver", dataplane.KindOntologyModule, valid["module-draft"], []byte("0.1.0"), []byte("01.0."), "non-canonical semantic version"),
		mutated("universe-invalid-timestamp", dataplane.KindUniverse, valid["universe-world-state"], []byte("2026-08-12T00:00:00Z"), []byte("2026-13-12T00:00:00Z"), "invalid canonical timestamp"),
	)

	sort.Slice(vectors, func(i, j int) bool { return vectors[i].Name < vectors[j].Name })
	for i := 1; i < len(vectors); i++ {
		if vectors[i-1].Name == vectors[i].Name {
			return nil, fmt.Errorf("duplicate vector %q", vectors[i].Name)
		}
	}
	return vectors, nil
}

func mutated(name, kind string, data, old, replacement []byte, reason string) Vector {
	return mutatedNth(name, kind, data, old, replacement, 1, reason)
}

func mutatedNth(name, kind string, data, old, replacement []byte, occurrence int, reason string) Vector {
	if len(old) != len(replacement) || occurrence < 1 {
		panic("invalid equal-length conformance mutation")
	}
	out := append([]byte(nil), data...)
	offset := 0
	for found := 1; found <= occurrence; found++ {
		index := bytes.Index(out[offset:], old)
		if index < 0 {
			panic("conformance mutation target absent: " + name)
		}
		offset += index
		if found == occurrence {
			copy(out[offset:offset+len(old)], replacement)
			return Vector{Name: name, Kind: kind, Reason: reason, Data: out}
		}
		offset += len(old)
	}
	panic("unreachable conformance mutation")
}
