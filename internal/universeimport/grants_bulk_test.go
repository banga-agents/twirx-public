package universeimport

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

func TestCompileGrantsBulkProjectionPreservesNativeAndProofLinks(t *testing.T) {
	projection := grantsProjectionFixture(t)
	sourceDigest := fixedDigest("raw grants XML")
	config := grantsTestConfig(projection)
	config.EvidenceClass = "current_observation"
	config.PolicyDecisionDigest = dataplane.OptionalDigest{Present: true, Value: fixedDigest("human policy")}
	records, err := CompileGrantsBulkProjection(projection, sourceDigest, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].NativeKey != "grants-gov:opportunity/12345" || records[0].Frame.UniverseID != "tw:opportunity" {
		t.Fatalf("unexpected record: %+v", records)
	}
	byTerm := packetsByNativePredicate(records[0].Packets)
	if packet := byTerm["AwardCeiling"][0].Packet; packet.Object.NativeLexical != "500000" || packet.Object.Typed == nil || packet.Object.Typed.Lexical != "500000.0" || packet.Source.RepresentationDigest != sourceDigest {
		t.Fatalf("amount packet lost source fidelity: %+v", packet)
	}
	if packet := byTerm["CloseDate"][0].Packet; packet.Object.NativeLexical != "11102026" || packet.Object.Typed != nil {
		t.Fatalf("date-only deadline was invented as an instant: %+v", packet)
	}
	if packet := byTerm["AdditionalInformationOnEligibility"][0].Packet; packet.Object.NativeStatus != "withheld" || packet.Object.NativeLexical != "" || packet.Object.Typed != nil {
		t.Fatalf("privacy-sensitive eligibility text entered a public packet: %+v", packet)
	}
	for _, mapping := range records[0].Mappings {
		if mapping.Claim.Status != "candidate" || mapping.Claim.ReviewDecisionDigest.Present {
			t.Fatal("bulk mapping crossed review authority")
		}
	}
	for _, slot := range records[0].Frame.Slots {
		for _, packet := range slot.PacketDigests {
			if !containsDigest(records[0].Frame.Derivation.PacketDigests, packet) {
				t.Fatal("slot packet is absent from derivation")
			}
		}
	}
}

func TestCompileGrantsBulkProjectionRejectsLeakedOrUnboundData(t *testing.T) {
	base := grantsProjectionFixture(t)
	tests := [][]byte{
		bytes.Replace(base, []byte(`"agency_name":"Health and Human Services"`), []byte(`"agency_name":"Health and Human Services","grantor_contact_email":"person@example.gov"`), 1),
		bytes.Replace(base, []byte(`"source_locator":"/Grants/OpportunitySynopsisDetail_1_0[0]"`), []byte(`"source_locator":"../../private"`), 1),
		bytes.Replace(base, []byte(`"opportunity_id":"12345"`), []byte(`"opportunity_id":""`), 1),
		bytes.Replace(base, []byte(`"last_updated_date":"08112026"`), []byte(`"last_updated_date":"2026-08-11"`), 1),
		bytes.Replace(base, []byte(`"award_ceiling":"500000"`), []byte(`"award_ceiling":"500,000"`), 1),
		bytes.Replace(base, []byte(`"opportunity_number":"PUBLIC-001"`), []byte(`"opportunity_number":"PUBLIC\n001"`), 1),
		bytes.Replace(base, []byte(`"agency_code":"HHS"`), []byte(`"agency_code":"HHS\tROOT"`), 1),
	}
	for index, data := range tests {
		config := grantsTestConfig(data)
		config.EvidenceClass = "current_observation"
		config.PolicyDecisionDigest = dataplane.OptionalDigest{Present: true, Value: fixedDigest("human policy")}
		if _, err := CompileGrantsBulkProjection(data, fixedDigest("raw grants XML"), config); err == nil {
			t.Fatalf("unsafe projection %d was accepted", index)
		}
	}
	config := grantsTestConfig(base)
	config.EvidenceClass = "current_observation"
	if _, err := CompileGrantsBulkProjection(base, fixedDigest("raw grants XML"), config); err == nil {
		t.Fatal("real projection without policy decision was accepted")
	}
}

func TestCompileGrantsBulkProjectionWindowKeepsWholeFileValidation(t *testing.T) {
	base := grantsProjectionFixture(t)
	var projection map[string]any
	if err := json.Unmarshal(base, &projection); err != nil {
		t.Fatal(err)
	}
	records := projection["records"].([]any)
	second := map[string]any{}
	for key, value := range records[0].(map[string]any) {
		second[key] = value
	}
	second["source_locator"] = "/Grants/OpportunitySynopsisDetail_1_0[1]"
	second["opportunity_id"] = "12346"
	// OpportunityNumber is deliberately shared: OpportunityID is the source
	// record identity.
	projection["records"] = append(records, second)
	data, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	config := grantsTestConfig(data)
	config.EvidenceClass = "current_observation"
	config.PolicyDecisionDigest = dataplane.OptionalDigest{Present: true, Value: fixedDigest("human policy")}
	window, total, err := CompileGrantsBulkProjectionWindow(data, fixedDigest("raw grants XML"), config, 1, 1)
	if err != nil || total != 2 || len(window) != 1 || window[0].NativeKey != "grants-gov:opportunity/12346" {
		t.Fatalf("unexpected window: total=%d records=%+v err=%v", total, window, err)
	}
	bad := bytes.Replace(data, []byte(`"opportunity_id":"12346"`), []byte(`"opportunity_id":"12345"`), 1)
	badConfig := grantsTestConfig(bad)
	badConfig.EvidenceClass = "current_observation"
	badConfig.PolicyDecisionDigest = dataplane.OptionalDigest{Present: true, Value: fixedDigest("human policy")}
	if _, _, err := CompileGrantsBulkProjectionWindow(bad, fixedDigest("raw grants XML"), badConfig, 0, 1); err == nil {
		t.Fatal("window compilation ignored an invalid record outside its selected result")
	}
}

func grantsProjectionFixture(t *testing.T) []byte {
	t.Helper()
	data, err := grantsProjectionFixtureBytes()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func grantsProjectionFixtureBytes() ([]byte, error) {
	value := map[string]any{"format": GrantsProjectionFormat, "records": []map[string]any{{
		"record_kind": "OpportunitySynopsisDetail_1_0", "source_locator": "/Grants/OpportunitySynopsisDetail_1_0[0]", "opportunity_id": "12345", "opportunity_number": "PUBLIC-001", "opportunity_title": "Open Infrastructure Research", "opportunity_category": "D", "category_of_funding_activity": []string{"ST"}, "cfda_numbers": []string{"93.001"}, "eligible_applicants": []string{"12", "20"}, "additional_information_on_eligibility": "See source wording.", "agency_code": "HHS", "agency_name": "Health and Human Services", "post_date": "08112026", "close_date": "11102026", "last_updated_date": "08112026", "estimated_synopsis_post_date": "", "estimated_synopsis_close_date": "", "estimated_synopsis_close_date_explanation": "", "estimated_award_date": "", "estimated_project_start_date": "", "expected_number_of_awards": "5", "estimated_total_program_funding": "1000000", "award_ceiling": "500000", "award_floor": "10000", "archive_date": "12102026", "cost_sharing_or_matching_requirement": "No", "version": "Synopsis 1",
	}}}
	return json.Marshal(value)
}

func containsDigest(values []dataplane.Digest, target dataplane.Digest) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func FuzzCompileGrantsBulkProjection(f *testing.F) {
	seed, err := grantsProjectionFixtureBytes()
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		config := grantsTestConfig(data)
		config.EvidenceClass = "current_observation"
		config.PolicyDecisionDigest = dataplane.OptionalDigest{Present: true, Value: fixedDigest("human policy")}
		records, err := CompileGrantsBulkProjection(data, fixedDigest("raw grants XML"), config)
		if err != nil {
			return
		}
		for _, record := range records {
			if err := record.Frame.Validate(); err != nil {
				t.Fatalf("accepted projection produced invalid frame: %v", err)
			}
		}
	})
}
