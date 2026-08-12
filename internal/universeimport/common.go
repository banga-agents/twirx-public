// Package universeimport compiles already-stored, source-specific public
// representations into Semantic Data Plane packets, candidate mapping claims
// and proof-linked E4 frames. It contains no network access.
package universeimport

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

type Config struct {
	OriginID             string
	ObservedAt           string
	RepresentationDigest dataplane.Digest
	ObservationDigest    dataplane.Digest
	ModuleSetDigest      dataplane.Digest
	PolicyDecisionDigest dataplane.OptionalDigest
	EvidenceClass        string
	EvidenceRef          string
	EvidenceStored       bool
}

type PacketArtifact struct {
	Packet dataplane.Packet
	CBOR   []byte
	Digest dataplane.Digest
}

type MappingArtifact struct {
	Claim  dataplane.MappingClaim
	CBOR   []byte
	Digest dataplane.Digest
}

type RecordArtifact struct {
	NativeKey      string
	Packets        []PacketArtifact
	Mappings       []MappingArtifact
	Frame          dataplane.Frame
	FrameCBOR      []byte
	FrameDigest    dataplane.Digest
	EvidenceClass  string
	Representation dataplane.Digest
}

func (config Config) validate(representation []byte) error {
	if !config.EvidenceStored || config.EvidenceRef == "" {
		return fmt.Errorf("universe importer: representation must be stored before parsing")
	}
	if dataplane.DigestBytes(representation) != config.RepresentationDigest {
		return fmt.Errorf("universe importer: representation digest mismatch")
	}
	if config.ObservationDigest == (dataplane.Digest{}) || config.ModuleSetDigest == (dataplane.Digest{}) {
		return fmt.Errorf("universe importer: observation and module-set evidence are required")
	}
	if config.OriginID == "" || config.ObservedAt == "" {
		return fmt.Errorf("universe importer: origin and observation time are required")
	}
	switch config.EvidenceClass {
	case "test_fixture", "recorded_offline_replay":
		if config.PolicyDecisionDigest.Present {
			return fmt.Errorf("universe importer: fixture/replay cannot imply live policy authority")
		}
	case "archive_observation", "current_observation":
		if !config.PolicyDecisionDigest.Present || config.PolicyDecisionDigest.Value == (dataplane.Digest{}) {
			return fmt.Errorf("universe importer: real evidence requires an exact policy decision")
		}
	default:
		return fmt.Errorf("universe importer: unsupported evidence class %q", config.EvidenceClass)
	}
	return nil
}

func fixedDigest(label string) dataplane.Digest { return sha256.Sum256([]byte(label)) }

func compilePacket(packet dataplane.Packet) (PacketArtifact, error) {
	encoded, err := dataplane.MarshalPacket(packet)
	if err != nil {
		return PacketArtifact{}, err
	}
	return PacketArtifact{Packet: packet, CBOR: encoded, Digest: dataplane.DigestBytes(encoded)}, nil
}

func compileMapping(claim dataplane.MappingClaim) (MappingArtifact, error) {
	encoded, err := dataplane.MarshalMappingClaim(claim)
	if err != nil {
		return MappingArtifact{}, err
	}
	return MappingArtifact{Claim: claim, CBOR: encoded, Digest: dataplane.DigestBytes(encoded)}, nil
}

func finishRecord(nativeKey, evidenceClass string, representation dataplane.Digest, packets []PacketArtifact, mappings []MappingArtifact, frame dataplane.Frame) (RecordArtifact, error) {
	sort.Slice(packets, func(i, j int) bool { return bytes.Compare(packets[i].Digest[:], packets[j].Digest[:]) < 0 })
	sort.Slice(mappings, func(i, j int) bool { return bytes.Compare(mappings[i].Digest[:], mappings[j].Digest[:]) < 0 })
	encoded, err := dataplane.MarshalFrame(frame)
	if err != nil {
		return RecordArtifact{}, err
	}
	return RecordArtifact{NativeKey: nativeKey, Packets: packets, Mappings: mappings, Frame: frame, FrameCBOR: encoded, FrameDigest: dataplane.DigestBytes(encoded), EvidenceClass: evidenceClass, Representation: representation}, nil
}

func sortedPacketDigests(packets []PacketArtifact) []dataplane.Digest {
	values := make([]dataplane.Digest, len(packets))
	for i := range packets {
		values[i] = packets[i].Digest
	}
	sort.Slice(values, func(i, j int) bool { return bytes.Compare(values[i][:], values[j][:]) < 0 })
	return values
}

func packetByNativePredicate(packets []PacketArtifact) map[string]PacketArtifact {
	result := make(map[string]PacketArtifact, len(packets))
	for _, packet := range packets {
		result[packet.Packet.Predicate.Native] = packet
	}
	return result
}

func packetsByNativePredicate(packets []PacketArtifact) map[string][]PacketArtifact {
	result := make(map[string][]PacketArtifact, len(packets))
	for _, packet := range packets {
		result[packet.Packet.Predicate.Native] = append(result[packet.Packet.Predicate.Native], packet)
	}
	for key := range result {
		sort.Slice(result[key], func(i, j int) bool {
			return bytes.Compare(result[key][i].Digest[:], result[key][j].Digest[:]) < 0
		})
	}
	return result
}

func mappingsForPackets(originID, universeID, moduleID string, definitions []mappingDefinition, packets []PacketArtifact) ([]MappingArtifact, error) {
	byPredicate := packetsByNativePredicate(packets)
	result := make([]MappingArtifact, 0, len(definitions))
	for _, definition := range definitions {
		matched, exists := byPredicate[definition.NativeTerm]
		if !exists || len(matched) == 0 {
			return nil, fmt.Errorf("universe importer: mapping %s lacks a packet", definition.ID)
		}
		evidence := make([]dataplane.Digest, len(matched))
		for i := range matched {
			evidence[i] = matched[i].Digest
		}
		claim := dataplane.MappingClaim{
			Version:   dataplane.MappingVersion,
			MappingID: definition.ID,
			Native:    dataplane.MappingNative{OriginID: originID, SchemaRef: dataplane.OptionalText{Present: true, Value: definition.SchemaRef}, Term: definition.NativeTerm, LocatorPattern: dataplane.OptionalText{Present: true, Value: definition.LocatorPattern}},
			Semantic:  dataplane.MappingSemantic{ConceptID: definition.ConceptID, RoleID: dataplane.OptionalText{Present: true, Value: definition.RoleID}},
			Relation:  "candidate",
			Scope:     dataplane.MappingScope{UniverseID: universeID, Languages: []string{}},
			Status:    "candidate", EvidencePacketDigests: evidence, ModuleID: moduleID,
		}
		artifact, err := compileMapping(claim)
		if err != nil {
			return nil, err
		}
		result = append(result, artifact)
	}
	return result, nil
}

func frameSlotMany(role, status, cardinality, mappingID string, packets []PacketArtifact, values []dataplane.TypedValue) (dataplane.FrameSlot, error) {
	digests := make([]dataplane.Digest, len(packets))
	for i := range packets {
		digests[i] = packets[i].Digest
	}
	sort.Slice(digests, func(i, j int) bool { return bytes.Compare(digests[i][:], digests[j][:]) < 0 })
	type encodedValue struct {
		value   dataplane.TypedValue
		encoded []byte
	}
	ordered := make([]encodedValue, len(values))
	for i := range values {
		encoded, err := dataplane.CanonicalTypedValueBytes(values[i])
		if err != nil {
			return dataplane.FrameSlot{}, err
		}
		ordered[i] = encodedValue{value: values[i], encoded: encoded}
	}
	sort.Slice(ordered, func(i, j int) bool { return bytes.Compare(ordered[i].encoded, ordered[j].encoded) < 0 })
	values = values[:0]
	for i := range ordered {
		if i > 0 && bytes.Equal(ordered[i-1].encoded, ordered[i].encoded) {
			continue
		}
		values = append(values, ordered[i].value)
	}
	mappings := []string{}
	if mappingID != "" {
		mappings = []string{mappingID}
	}
	return dataplane.FrameSlot{RoleID: role, Status: status, Cardinality: cardinality, Values: values, PacketDigests: digests, MappingIDs: mappings, Conflict: "none"}, nil
}

type mappingDefinition struct {
	ID             string
	NativeTerm     string
	SchemaRef      string
	LocatorPattern string
	ConceptID      string
	RoleID         string
}
