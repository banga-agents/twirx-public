package opportunitypilot

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validXML = `<?xml version="1.0" encoding="UTF-8"?>
<Grants xmlns="http://apply.grants.gov/system/OpportunityDetail-V1.0">
  <OpportunitySynopsisDetail_1_0>
    <OpportunityID>12345</OpportunityID>
    <OpportunityNumber>PUBLIC-001</OpportunityNumber>
    <OpportunityTitle>Open Infrastructure Research</OpportunityTitle>
    <CategoryOfFundingActivity>ST</CategoryOfFundingActivity>
    <EligibleApplicants>12</EligibleApplicants>
    <EligibleApplicants>20</EligibleApplicants>
    <AdditionalInformationOnEligibility>See source wording.</AdditionalInformationOnEligibility>
    <AgencyCode>HHS</AgencyCode>
    <AgencyName>Health and Human Services</AgencyName>
    <PostDate>08112026</PostDate>
    <CloseDate>11102026</CloseDate>
    <LastUpdatedDate>08112026</LastUpdatedDate>
    <AwardCeiling>500000</AwardCeiling>
    <AwardFloor>10000</AwardFloor>
    <GrantorContactName>Private Projection Exclusion</GrantorContactName>
    <GrantorContactEmail>excluded@example.gov</GrantorContactEmail>
    <Description>Long full description must remain private evidence.</Description>
  </OpportunitySynopsisDetail_1_0>
</Grants>`

func TestProjectXMLExcludesContactAndDescriptionValues(t *testing.T) {
	records, report, err := ProjectXML([]byte(validXML), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || report.RecordsAccepted != 1 || report.ContactFieldsExcluded != 2 || report.DescriptionFieldsExcluded != 1 {
		t.Fatalf("unexpected projection report: records=%d report=%+v", len(records), report)
	}
	encoded, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"Private Projection Exclusion", "excluded@example.gov", "Long full description"} {
		if bytes.Contains(encoded, []byte(forbidden)) {
			t.Fatalf("projection leaked %q", forbidden)
		}
	}
	if records[0].OpportunityNumber != "PUBLIC-001" || len(records[0].EligibleApplicants) != 2 || records[0].AwardCeiling != "500000" {
		t.Fatalf("approved fields were lost: %+v", records[0])
	}
}

func TestProjectVerifiedXMLRequiresExactDigest(t *testing.T) {
	if _, _, _, err := ProjectVerifiedXML("Grants.xml", []byte(validXML), digest([]byte(validXML)), 10); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ProjectVerifiedXML("Grants.xml", []byte(validXML), "sha256:"+strings.Repeat("0", 64), 10); err == nil {
		t.Fatal("projection accepted XML without its exact verified digest")
	}
}

func TestProjectXMLTreatsEmptyOptionalScalarAsNotProvided(t *testing.T) {
	input := strings.Replace(validXML, "<CloseDate>11102026</CloseDate>", "<CloseDate></CloseDate>", 1)
	input = strings.Replace(input, "<AwardFloor>10000</AwardFloor>", "<AwardFloor/>", 1)
	records, report, err := ProjectXML([]byte(input), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || report.RecordsAccepted != 1 || records[0].CloseDate != "" || records[0].AwardFloor != "" {
		t.Fatalf("empty optional scalars were not treated as absent: records=%+v report=%+v", records, report)
	}
	if _, _, err := ProjectXML([]byte(strings.Replace(validXML, "<OpportunityID>12345</OpportunityID>", "<OpportunityID></OpportunityID>", 1)), 10); err == nil {
		t.Fatal("empty required identity was admitted")
	}
}

func TestProjectXMLPreservesPaddedEligibilityLexicalValue(t *testing.T) {
	want := "\n      Eligibility is source stated.\n    "
	input := strings.Replace(validXML, "See source wording.", want, 1)
	records, _, err := ProjectXML([]byte(input), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].AdditionalInformationOnEligibility != want {
		t.Fatalf("source lexical whitespace changed: %q", records[0].AdditionalInformationOnEligibility)
	}
}

func TestProjectXMLRejectsMalformedAdversarialInputs(t *testing.T) {
	tests := []string{
		`<!DOCTYPE Grants [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><Grants><OpportunitySynopsisDetail_1_0><OpportunityID>&xxe;</OpportunityID></OpportunitySynopsisDetail_1_0></Grants>`,
		strings.Replace(validXML, "<OpportunityID>12345</OpportunityID>", "<OpportunityID>12345</OpportunityID><OpportunityID>67890</OpportunityID>", 1),
		strings.Replace(validXML, "<OpportunityTitle>Open Infrastructure Research</OpportunityTitle>", "<OpportunityTitle><nested>bad</nested></OpportunityTitle>", 1),
		strings.Replace(validXML, "<LastUpdatedDate>08112026</LastUpdatedDate>", "<LastUpdatedDate>2026-08-11</LastUpdatedDate>", 1),
		strings.Replace(validXML, "<OpportunityNumber>PUBLIC-001</OpportunityNumber>", "", 1),
		strings.Replace(validXML, "<Description>Long full description must remain private evidence.</Description>", "<Unexpected>"+strings.Repeat("<nested>", 17)+"x"+strings.Repeat("</nested>", 17)+"</Unexpected>", 1),
	}
	for index, input := range tests {
		if _, _, err := ProjectXML([]byte(input), 10); err == nil {
			t.Fatalf("adversarial XML %d was accepted", index)
		}
	}
}

func TestProjectXMLUsesOpportunityIDAsIdentity(t *testing.T) {
	record := func(id, number string) string {
		value := strings.Replace(validXML, "<OpportunityID>12345</OpportunityID>", "<OpportunityID>"+id+"</OpportunityID>", 1)
		value = strings.Replace(value, "<OpportunityNumber>PUBLIC-001</OpportunityNumber>", "<OpportunityNumber>"+number+"</OpportunityNumber>", 1)
		start := strings.Index(value, "<OpportunitySynopsisDetail_1_0>")
		end := strings.Index(value, "</OpportunitySynopsisDetail_1_0>") + len("</OpportunitySynopsisDetail_1_0>")
		return value[start:end]
	}
	input := `<?xml version="1.0" encoding="UTF-8"?><Grants>` + record("100", "DUPLICATE") + record("200", "MIDDLE") + record("300", "DUPLICATE") + `</Grants>`
	records, report, err := ProjectXML([]byte(input), 10)
	if err != nil || len(records) != 3 || report.RecordsAccepted != 3 {
		t.Fatalf("distinct source IDs sharing a display number were rejected: records=%d report=%+v err=%v", len(records), report, err)
	}
	duplicateID := strings.Replace(input, "<OpportunityID>300</OpportunityID>", "<OpportunityID>100</OpportunityID>", 1)
	if _, _, err := ProjectXML([]byte(duplicateID), 10); err == nil {
		t.Fatal("duplicate source OpportunityID was admitted")
	}
}

func TestProjectVerifiedAcquisitionKeepsRawEvidencePrivate(t *testing.T) {
	archive := buildTestArchive(t, validXML)
	if len(archive) <= RangeBytes {
		t.Fatalf("test archive is too small: %d", len(archive))
	}
	loaded := loadedTestOrder(t)
	acquisition := filepath.Join(t.TempDir(), "acquisition")
	state := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(state, 0o700); err != nil {
		t.Fatal(err)
	}
	fetcher := &memoryRangeFetcher{data: archive}
	times := []time.Time{time.Date(2026, 8, 12, 4, 1, 0, 0, time.UTC), time.Date(2026, 8, 12, 4, 2, 0, 0, time.UTC)}
	now := func() time.Time {
		value := times[0]
		if len(times) > 1 {
			times = times[1:]
		}
		return value
	}
	if _, err := acquire(context.Background(), loaded, &Control{Format: ControlFormat, Enabled: true}, acquisition, state, now, func(time.Duration) {}, fetcher); err != nil {
		t.Fatal(err)
	}
	projection := filepath.Join(t.TempDir(), "projection")
	manifest, err := ProjectAcquisition(acquisition, loaded, projection)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.RawEvidencePublic || manifest.ContactsProjected || manifest.Report.RecordsAccepted != 1 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	projected, err := os.ReadFile(filepath.Join(projection, "approved-projection.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(projected, []byte("excluded@example.gov")) || bytes.Contains(projected, []byte("Private Projection Exclusion")) {
		t.Fatal("public projection leaked contact data")
	}
	if _, err := os.Stat(filepath.Join(projection, SourceFilename)); !os.IsNotExist(err) {
		t.Fatal("raw ZIP was copied into projection output")
	}
	verified, verifiedBytes, err := VerifyProjection(projection, acquisition, loaded)
	if err != nil || verified.ProjectionDigest != digest(projected) || !bytes.Equal(verifiedBytes, projected) {
		t.Fatalf("verify projection: manifest=%+v err=%v", verified, err)
	}
	if err := os.Remove(filepath.Join(projection, "projection-manifest.json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := VerifyProjection(projection, acquisition, loaded); err == nil {
		t.Fatal("partial projection without final manifest was admitted")
	}
}

func TestVerifyProjectionRejectsUnexpectedArtifact(t *testing.T) {
	archive := buildTestArchive(t, validXML)
	loaded := loadedTestOrder(t)
	acquisition := filepath.Join(t.TempDir(), "acquisition")
	state := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(state, 0o700); err != nil {
		t.Fatal(err)
	}
	times := []time.Time{time.Date(2026, 8, 12, 4, 1, 0, 0, time.UTC), time.Date(2026, 8, 12, 4, 2, 0, 0, time.UTC)}
	now := func() time.Time {
		value := times[0]
		if len(times) > 1 {
			times = times[1:]
		}
		return value
	}
	if _, err := acquire(context.Background(), loaded, &Control{Format: ControlFormat, Enabled: true}, acquisition, state, now, func(time.Duration) {}, &memoryRangeFetcher{data: archive}); err != nil {
		t.Fatal(err)
	}
	projection := filepath.Join(t.TempDir(), "projection")
	if _, err := ProjectAcquisition(acquisition, loaded, projection); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projection, "unexpected-contact.txt"), []byte("must fail closed"), 0o440); err != nil {
		t.Fatal(err)
	}
	if _, _, err := VerifyProjection(projection, acquisition, loaded); err == nil {
		t.Fatal("projection with unexpected artifact was admitted")
	}
}

func buildTestArchive(t *testing.T, source string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	header := &zip.FileHeader{Name: "GrantsDBExtract20260811v2.xml", Method: zip.Store}
	xmlWriter, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := xmlWriter.Write([]byte(source)); err != nil {
		t.Fatal(err)
	}
	fillerHeader := &zip.FileHeader{Name: "OpportunityDetail-V1.0.xsd", Method: zip.Store}
	filler, err := writer.CreateHeader(fillerHeader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := filler.Write(bytes.Repeat([]byte("x"), 2*RangeBytes)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func FuzzProjectGrantsXML(f *testing.F) {
	f.Add([]byte(validXML))
	f.Add([]byte(`<Grants/>`))
	f.Add([]byte(`<!DOCTYPE Grants><Grants/>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		records, _, err := ProjectXML(data, 100)
		if err != nil {
			return
		}
		for _, record := range records {
			if err := record.Validate(); err != nil {
				t.Fatalf("accepted XML produced invalid record: %v", err)
			}
		}
	})
}
