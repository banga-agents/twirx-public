package snapshotartifact

import (
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

func TestPacketSegmentRejectsDigestMismatch(t *testing.T) {
	segment := PacketSegment{Format: PacketSegmentFormat, StartSequence: 1, Entries: []PacketEntry{{Sequence: 1, Digest: DigestReference(dataplane.DigestBytes([]byte("different"))), CBOR: []byte{0x80}}}}
	if err := segment.Validate(); err == nil {
		t.Fatal("expected digest mismatch rejection")
	}
}

func TestDecodeRejectsUnknownField(t *testing.T) {
	data := []byte(`{"format":"tw.snapshot-packet-segment/0.1","start_sequence":1,"entries":[],"extra":true}`)
	if _, err := UnmarshalPacketSegment(data); err == nil {
		t.Fatal("expected unknown-field rejection")
	}
}

func TestParseDigestRejectsMalformed(t *testing.T) {
	for _, value := range []string{"", "sha256:00", "sha1:0000000000000000000000000000000000000000", "sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		if _, err := ParseDigest(value); err == nil {
			t.Fatalf("accepted malformed digest %q", value)
		}
	}
}

func TestProofIndexRejectsMalformedAndUnknownFields(t *testing.T) {
	valid := ProofIndex{Format: ProofIndexFormat, Entries: []ProofEntry{validProofEntry("packet-one")}}
	encoded, err := Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UnmarshalProofIndex(encoded); err != nil {
		t.Fatalf("valid proof index rejected: %v", err)
	}
	if _, err := UnmarshalProofIndex([]byte(`{"format":"tw.snapshot-proof-index/0.1","entries":[],"extra":true}`)); err == nil {
		t.Fatal("unknown proof-index field accepted")
	}
	mutations := []func(*ProofIndex){
		func(index *ProofIndex) { index.Entries[0].ProofType = "automatic_canon" },
		func(index *ProofIndex) { index.Entries[0].EvidenceID = "sha256:00" },
		func(index *ProofIndex) { index.Entries[0].Artifacts[0].Path = "../private" },
		func(index *ProofIndex) { index.Entries[0].Artifacts[0].Size = 0 },
		func(index *ProofIndex) { index.Entries = append(index.Entries, index.Entries[0]) },
	}
	for mutationIndex, mutate := range mutations {
		index := valid
		index.Entries = append([]ProofEntry(nil), valid.Entries...)
		index.Entries[0].Artifacts = append([]ProofArtifact(nil), valid.Entries[0].Artifacts...)
		mutate(&index)
		if err := index.Validate(); err == nil {
			t.Fatalf("proof-index mutation %d accepted", mutationIndex)
		}
	}
}

func validProofEntry(seed string) ProofEntry {
	packet := DigestReference(dataplane.DigestBytes([]byte(seed)))
	evidence := DigestReference(dataplane.DigestBytes([]byte("evidence-" + seed)))
	artifact := DigestReference(dataplane.DigestBytes([]byte("artifact-" + seed)))
	return ProofEntry{ProofType: "controlled_scale_fixture", PacketDigest: packet, EvidenceID: evidence, EvidenceClass: "test_fixture", ExecutionOriginID: "fixture-origin", OperationID: "fixture.operation", FieldID: "field", Artifacts: []ProofArtifact{{Name: "observation.cbor", Path: "proof/fixture/observation.cbor", Digest: artifact, Size: 1}}}
}

func FuzzPacketSegmentJSON(f *testing.F) {
	f.Add([]byte(`{"format":"tw.snapshot-packet-segment/0.1","start_sequence":1,"entries":[]}`))
	f.Add([]byte{0xff, 0x00, 0x01})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = UnmarshalPacketSegment(data)
	})
}

func FuzzProofIndexJSON(f *testing.F) {
	seed, err := Marshal(ProofIndex{Format: ProofIndexFormat, Entries: []ProofEntry{validProofEntry("fuzz")}})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(`{"format":"tw.snapshot-proof-index/0.1","entries":[]}`))
	f.Add([]byte{0xff, 0x00, 0x01})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = UnmarshalProofIndex(data)
	})
}
