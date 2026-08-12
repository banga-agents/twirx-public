// Package e4capacity produces explicitly controlled, non-public capacity
// fixtures for measuring the immutable E4 Universe Snapshot. It has no
// network access and its output must never be counted as source evidence.
package e4capacity

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/universesnapshot"
)

const (
	UniverseID = "tw:world-state"
	FrameType  = "world:IndicatorObservation"
)

// ControlledFrames returns deterministic test_fixture frames. The frames use
// invented SYN identifiers and carry no origin, representation, observation,
// or public-source authority.
func ControlledFrames(count int) ([]universesnapshot.SourceFrame, error) {
	if count < 1 || count > universesnapshot.MaxFrames {
		return nil, fmt.Errorf("e4capacity: frame count outside 1..%d", universesnapshot.MaxFrames)
	}
	result := make([]universesnapshot.SourceFrame, count)
	for index := 0; index < count; index++ {
		byLabel := map[string]dataplane.Digest{
			"country":   digest(fmt.Sprintf("country/%d", index)),
			"indicator": digest(fmt.Sprintf("indicator/%d", index)),
			"value":     digest(fmt.Sprintf("value/%d", index)),
			"period":    digest(fmt.Sprintf("period/%d", index)),
		}
		packetDigests := []dataplane.Digest{byLabel["country"], byLabel["indicator"], byLabel["value"], byLabel["period"]}
		sort.Slice(packetDigests, func(i, j int) bool { return bytes.Compare(packetDigests[i][:], packetDigests[j][:]) < 0 })
		country := Country(index)
		frame := dataplane.Frame{
			Version:             dataplane.FrameVersion,
			UniverseID:          UniverseID,
			FrameType:           FrameType,
			NativeKey:           fmt.Sprintf("controlled-capacity:synthetic/%06d", index),
			CanonicalCandidates: []string{fmt.Sprintf("world:synthetic/%06d", index)},
			Slots: []dataplane.FrameSlot{
				{RoleID: "world:country", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "identifier", Lexical: country}}, PacketDigests: []dataplane.Digest{byLabel["country"]}, MappingIDs: []string{"mapping:test/country@0.1"}, Conflict: "none"},
				{RoleID: "world:indicator", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "identifier", Lexical: "TEST.INDICATOR"}}, PacketDigests: []dataplane.Digest{byLabel["indicator"]}, MappingIDs: []string{"mapping:test/indicator@0.1"}, Conflict: "none"},
				{RoleID: "world:observedValue", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "decimal", Lexical: fmt.Sprintf("%d.0", index)}}, PacketDigests: []dataplane.Digest{byLabel["value"]}, MappingIDs: []string{"mapping:test/value@0.1"}, Conflict: "none"},
				{RoleID: "world:period", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "date", Lexical: "2026-08-12"}}, PacketDigests: []dataplane.Digest{byLabel["period"]}, MappingIDs: []string{"mapping:test/period@0.1"}, Conflict: "none"},
			},
			Time:      dataplane.FrameTime{ComposedAt: "2026-08-12T00:00:00Z"},
			Epistemic: dataplane.FrameEpistemic{Lane: "provisional_semantic", CompletenessMillionths: 1000000, ConflictStatus: "none"},
			Derivation: dataplane.FrameDerivation{
				PacketDigests:          packetDigests,
				ModuleSetDigest:        digest("controlled-capacity/modules"),
				MappingIDs:             []string{"mapping:test/country@0.1", "mapping:test/indicator@0.1", "mapping:test/period@0.1", "mapping:test/value@0.1"},
				CompilerContractDigest: digest("controlled-capacity/compiler"),
				CompilerVersion:        "twirx-controlled-capacity@0.1",
			},
			Lifecycle: dataplane.FrameLifecycle{State: "current"},
		}
		encoded, err := dataplane.MarshalFrame(frame)
		if err != nil {
			return nil, fmt.Errorf("e4capacity: marshal frame %d: %w", index, err)
		}
		result[index] = universesnapshot.SourceFrame{Digest: dataplane.DigestBytes(encoded), CBOR: encoded, Frame: frame}
	}
	return result, nil
}

func Country(index int) string { return fmt.Sprintf("geo:SYN%06d", index) }

func digest(label string) dataplane.Digest { return sha256.Sum256([]byte(label)) }
