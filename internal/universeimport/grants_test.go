package universeimport

import (
	"bytes"
	"os"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

func TestCompileGrantsControlledFixture(t *testing.T) {
	representation := grantsFixture(t)
	records, err := CompileGrantsFetch(representation, grantsTestConfig(representation))
	if err != nil {
		t.Fatalf("compile controlled Grants.gov fixture: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d", len(records))
	}
	record := records[0]
	if record.NativeKey != "grants-gov:TWIRX-CONTROLLED-001" || len(record.Packets) != 10 || len(record.Mappings) != 8 || len(record.Frame.Slots) != 8 {
		t.Fatalf("record identity/counts = %q/%d/%d/%d", record.NativeKey, len(record.Packets), len(record.Mappings), len(record.Frame.Slots))
	}
	if err := record.Frame.Validate(); err != nil {
		t.Fatalf("frame validation: %v", err)
	}
	if record.Frame.Epistemic.CompletenessMillionths != 875000 {
		t.Fatalf("completeness = %d, want 875000", record.Frame.Epistemic.CompletenessMillionths)
	}
	byTerm := packetsByNativePredicate(record.Packets)
	if got := byTerm["synopsis.awardCeiling"][0].Packet.Object; got.NativeLexical != "100000" || got.Typed == nil || got.Typed.Lexical != "100000.0" {
		t.Fatalf("ceiling source/typed value = %+v", got)
	}
	if got := byTerm["originalDueDateDesc"][0].Packet.Object; got.NativeStatus != "resolved" || got.Typed != nil {
		t.Fatalf("free-form due date was treated as executable datetime: %+v", got)
	}
	var applicantMapping *dataplane.MappingClaim
	for i := range record.Mappings {
		claim := &record.Mappings[i].Claim
		if claim.MappingID == "mapping:grants-gov/applicant-type@0.1" {
			applicantMapping = claim
		}
		if claim.Status != "candidate" || claim.ReviewDecisionDigest.Present {
			t.Fatalf("mapping crossed human review boundary: %+v", claim)
		}
	}
	if applicantMapping == nil || len(applicantMapping.EvidencePacketDigests) != 2 {
		t.Fatalf("applicant mapping evidence = %+v", applicantMapping)
	}
	for _, packet := range record.Packets {
		if packet.Packet.Source.RepresentationDigest != record.Representation || packet.Packet.Source.OriginID != GrantsGovOriginID {
			t.Fatal("packet lost representation or source identity binding")
		}
	}
}

func TestCompileGrantsRejectsCredentialAndPersonalFields(t *testing.T) {
	base := grantsFixture(t)
	tests := []struct {
		name string
		old  []byte
		new  []byte
	}{
		{name: "token", old: []byte(`"token": ""`), new: []byte(`"token": "credential-shaped-value"`)},
		{name: "publisher uid", old: []byte(`"publisherUid": ""`), new: []byte(`"publisherUid": "account-name"`)},
		{name: "contact email", old: []byte(`"agencyContactEmail": ""`), new: []byte(`"agencyContactEmail": "person@example.org"`)},
		{name: "conflicting agency", old: []byte(`"owningAgencyCode": "TEST"`), new: []byte(`"owningAgencyCode": "OTHER"`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := bytes.Replace(base, test.old, test.new, 1)
			if bytes.Equal(body, base) {
				t.Fatal("test mutation did not apply")
			}
			if _, err := CompileGrantsFetch(body, grantsTestConfig(body)); err == nil {
				t.Fatal("accepted credential, personal data, or conflicting identity")
			}
		})
	}
}

func TestCompileGrantsRejectsMalformedAndUnknownData(t *testing.T) {
	base := grantsFixture(t)
	tests := [][]byte{
		bytes.Replace(base, []byte(`"errorcode": 0`), []byte(`"errorcode": 1`), 1),
		bytes.Replace(base, []byte(`"revision": 1`), []byte(`"revision": 1, "revision": 2`), 1),
		bytes.Replace(base, []byte(`"draftMode": "N"`), []byte(`"draftMode": "N", "unexpected": true`), 1),
		[]byte(`null`),
	}
	for index, body := range tests {
		if _, err := CompileGrantsFetch(body, grantsTestConfig(body)); err == nil {
			t.Fatalf("case %d accepted invalid source", index)
		}
	}
}

func TestCompileGrantsKeepsUnparseableAmountExplicit(t *testing.T) {
	base := grantsFixture(t)
	body := bytes.Replace(base, []byte(`"awardCeiling": "100000"`), []byte(`"awardCeiling": "100,000"`), 1)
	records, err := CompileGrantsFetch(body, grantsTestConfig(body))
	if err != nil {
		t.Fatal(err)
	}
	packet := packetsByNativePredicate(records[0].Packets)["synopsis.awardCeiling"][0].Packet
	if packet.Object.NativeLexical != "100,000" || packet.Object.Typed != nil {
		t.Fatalf("unparseable amount source fidelity = %+v", packet.Object)
	}
	for _, slot := range records[0].Frame.Slots {
		if slot.RoleID == "opportunity:maximumAmount" && (slot.Status != "unresolved" || len(slot.Values) != 0) {
			t.Fatalf("amount slot = %+v", slot)
		}
	}
}

func FuzzCompileGrantsFetch(f *testing.F) {
	seed, err := os.ReadFile("../../conformance/e4-importers/grants-fetch-controlled.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, representation []byte) {
		records, err := CompileGrantsFetch(representation, grantsTestConfig(representation))
		if err != nil {
			return
		}
		for _, record := range records {
			if err := record.Frame.Validate(); err != nil {
				t.Fatalf("accepted source generated invalid frame: %v", err)
			}
		}
	})
}

func grantsFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../conformance/e4-importers/grants-fetch-controlled.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func grantsTestConfig(representation []byte) Config {
	return Config{
		OriginID:             GrantsGovOriginID,
		ObservedAt:           "2026-08-12T00:00:00Z",
		RepresentationDigest: dataplane.DigestBytes(representation),
		ObservationDigest:    fixedDigest("grants-observation"),
		ModuleSetDigest:      fixedDigest("grants-module-set"),
		EvidenceClass:        "test_fixture",
		EvidenceRef:          "conformance/e4-importers/grants-fetch-controlled.json",
		EvidenceStored:       true,
	}
}
