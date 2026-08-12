package opportunitypilot

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
)

const (
	ProjectionFormat         = "tw.e4-opportunity-approved-projection/0.1"
	ProjectionReportFormat   = "tw.e4-opportunity-projection-report/0.1"
	ProjectionManifestFormat = "tw.e4-opportunity-projection-manifest/0.1"
	MaxZIPEntries            = 16
	MaxRecordScalar          = 18000
)

type OpportunityRecord struct {
	RecordKind                            string   `json:"record_kind"`
	SourceLocator                         string   `json:"source_locator"`
	OpportunityID                         string   `json:"opportunity_id"`
	OpportunityNumber                     string   `json:"opportunity_number"`
	OpportunityTitle                      string   `json:"opportunity_title"`
	OpportunityCategory                   string   `json:"opportunity_category,omitempty"`
	CategoryOfFundingActivity             []string `json:"category_of_funding_activity"`
	CFDANumbers                           []string `json:"cfda_numbers"`
	EligibleApplicants                    []string `json:"eligible_applicants"`
	AdditionalInformationOnEligibility    string   `json:"additional_information_on_eligibility,omitempty"`
	AgencyCode                            string   `json:"agency_code"`
	AgencyName                            string   `json:"agency_name"`
	PostDate                              string   `json:"post_date,omitempty"`
	CloseDate                             string   `json:"close_date,omitempty"`
	LastUpdatedDate                       string   `json:"last_updated_date"`
	EstimatedSynopsisPostDate             string   `json:"estimated_synopsis_post_date,omitempty"`
	EstimatedSynopsisCloseDate            string   `json:"estimated_synopsis_close_date,omitempty"`
	EstimatedSynopsisCloseDateExplanation string   `json:"estimated_synopsis_close_date_explanation,omitempty"`
	EstimatedAwardDate                    string   `json:"estimated_award_date,omitempty"`
	EstimatedProjectStartDate             string   `json:"estimated_project_start_date,omitempty"`
	ExpectedNumberOfAwards                string   `json:"expected_number_of_awards,omitempty"`
	EstimatedTotalProgramFunding          string   `json:"estimated_total_program_funding,omitempty"`
	AwardCeiling                          string   `json:"award_ceiling,omitempty"`
	AwardFloor                            string   `json:"award_floor,omitempty"`
	ArchiveDate                           string   `json:"archive_date,omitempty"`
	CostSharingOrMatchingRequirement      string   `json:"cost_sharing_or_matching_requirement,omitempty"`
	Version                               string   `json:"version,omitempty"`
}

type FieldCount struct {
	Field string `json:"field"`
	Count uint64 `json:"count"`
}

type ProjectionReport struct {
	Format                    string       `json:"format"`
	SourceRecordsSeen         uint64       `json:"source_records_seen"`
	RecordsAccepted           uint64       `json:"records_accepted"`
	RecordsRejected           uint64       `json:"records_rejected"`
	ContactFieldsExcluded     uint64       `json:"contact_fields_excluded"`
	DescriptionFieldsExcluded uint64       `json:"description_fields_excluded"`
	ExcludedFields            []FieldCount `json:"excluded_fields"`
	UnknownFields             []FieldCount `json:"unknown_fields"`
}

type ProjectionManifest struct {
	Format            string           `json:"format"`
	WorkOrderDigest   string           `json:"work_order_digest"`
	AcquisitionDigest string           `json:"acquisition_manifest_digest"`
	ArchiveDigest     string           `json:"archive_digest"`
	XMLPath           string           `json:"xml_path"`
	XMLDigest         string           `json:"xml_digest"`
	XMLSize           uint64           `json:"xml_size"`
	ProjectionDigest  string           `json:"projection_digest"`
	ProjectionSize    uint64           `json:"projection_size"`
	ReportDigest      string           `json:"report_digest"`
	Report            ProjectionReport `json:"report"`
	RawEvidencePublic bool             `json:"raw_evidence_public"`
	ContactsProjected bool             `json:"contacts_projected"`
}

var excludedContactFields = map[string]bool{
	"GrantorContactEmail":            true,
	"GrantorContactEmailDescription": true,
	"GrantorContactName":             true,
	"GrantorContactPhoneNumber":      true,
	"GrantorContactText":             true,
}

var approvedFields = map[string]bool{
	"OpportunityID": true, "OpportunityNumber": true, "OpportunityTitle": true,
	"OpportunityCategory": true, "CategoryOfFundingActivity": true, "CFDANumbers": true,
	"EligibleApplicants": true, "AdditionalInformationOnEligibility": true,
	"AgencyCode": true, "AgencyName": true, "PostDate": true, "CloseDate": true,
	"LastUpdatedDate": true, "EstimatedSynopsisPostDate": true,
	"EstimatedSynopsisCloseDate": true, "EstimatedSynopsisCloseDateExplanation": true,
	"EstimatedAwardDate": true, "EstimatedProjectStartDate": true,
	"ExpectedNumberOfAwards": true, "EstimatedTotalProgramFunding": true,
	"AwardCeiling": true, "AwardFloor": true, "ArchiveDate": true,
	"CostSharingOrMatchingRequirement": true, "Version": true,
}

func ProjectAcquisition(acquisitionRoot string, loaded *LoadedWorkOrder, output string) (ProjectionManifest, error) {
	var result ProjectionManifest
	if loaded == nil || !loaded.AuthorityVerified {
		return result, errors.New("opportunity pilot: verified human authority is required")
	}
	acquisition, err := VerifyAcquisition(acquisitionRoot, loaded)
	if err != nil {
		return result, err
	}
	archiveBytes, err := readRegular(filepath.Join(acquisitionRoot, SourceFilename), int64(MaximumArchive))
	if err != nil || digest(archiveBytes) != acquisition.ArchiveDigest {
		return result, errors.New("opportunity pilot: verified archive is unavailable")
	}
	xmlPath, xmlBytes, err := extractApprovedXML(archiveBytes, loaded.Order.MaximumExpandedXMLBytes)
	if err != nil {
		return result, err
	}
	projectionBytes, reportBytes, report, err := ProjectVerifiedXML(xmlPath, xmlBytes, digest(xmlBytes), loaded.Order.MaximumRecords)
	if err != nil {
		return result, err
	}
	root, err := createOutput(output)
	if err != nil {
		return result, err
	}
	if err := atomicfile.Write(filepath.Join(root, "approved-projection.json"), projectionBytes, int(loaded.Order.MaximumExpandedXMLBytes), 0o440); err != nil {
		return ProjectionManifest{}, err
	}
	if err := atomicfile.Write(filepath.Join(root, "projection-report.json"), reportBytes, MaxManifest, 0o440); err != nil {
		return ProjectionManifest{}, err
	}
	acquisitionManifestBytes, err := readRegular(filepath.Join(acquisitionRoot, "acquisition-manifest.json"), MaxManifest)
	if err != nil {
		return ProjectionManifest{}, err
	}
	result = ProjectionManifest{Format: ProjectionManifestFormat, WorkOrderDigest: loaded.Digest, AcquisitionDigest: digest(acquisitionManifestBytes), ArchiveDigest: acquisition.ArchiveDigest, XMLPath: xmlPath, XMLDigest: digest(xmlBytes), XMLSize: uint64(len(xmlBytes)), ProjectionDigest: digest(projectionBytes), ProjectionSize: uint64(len(projectionBytes)), ReportDigest: digest(reportBytes), Report: report, RawEvidencePublic: false, ContactsProjected: false}
	manifestBytes, err := marshalJSON(result, MaxManifest)
	if err != nil {
		return ProjectionManifest{}, err
	}
	// The projection manifest is written last. Raw XML, contacts and the ZIP
	// remain only in the separately verified private acquisition directory.
	if err := atomicfile.Write(filepath.Join(root, "projection-manifest.json"), manifestBytes, MaxManifest, 0o440); err != nil {
		return ProjectionManifest{}, err
	}
	return result, nil
}

// ProjectVerifiedXML applies the same approved-field projection to caller-
// supplied XML bytes only after the caller has verified and digest-bound the
// private source representation. It is an offline helper for tests and
// separated worker processes; it grants no acquisition authority.
func ProjectVerifiedXML(xmlPath string, xmlBytes []byte, xmlDigest string, maximumRecords uint64) ([]byte, []byte, ProjectionReport, error) {
	var report ProjectionReport
	if xmlPath == "" || filepath.Base(xmlPath) != xmlPath || !strings.EqualFold(filepath.Ext(xmlPath), ".xml") || !validDigest(xmlDigest) || digest(xmlBytes) != xmlDigest {
		return nil, nil, report, errors.New("opportunity pilot: verified XML identity is invalid")
	}
	records, report, err := ProjectXML(xmlBytes, maximumRecords)
	if err != nil {
		return nil, nil, report, err
	}
	projectionBytes, err := marshalJSON(struct {
		Format  string              `json:"format"`
		Records []OpportunityRecord `json:"records"`
	}{Format: ProjectionFormat, Records: records}, int(MaximumExpandedXML))
	if err != nil {
		return nil, nil, report, err
	}
	reportBytes, err := marshalJSON(report, MaxManifest)
	if err != nil {
		return nil, nil, report, err
	}
	return projectionBytes, reportBytes, report, nil
}

// VerifyProjection reconciles a public approved-field projection against the
// exact private acquisition without exposing the raw ZIP or XML. Callers use
// the returned projection bytes as the only admissible compiler input.
func VerifyProjection(projectionRoot, acquisitionRoot string, loaded *LoadedWorkOrder) (ProjectionManifest, []byte, error) {
	var manifest ProjectionManifest
	acquisition, err := VerifyAcquisition(acquisitionRoot, loaded)
	if err != nil {
		return manifest, nil, err
	}
	entries, err := os.ReadDir(projectionRoot)
	if err != nil || len(entries) != 3 {
		return manifest, nil, errors.New("opportunity pilot: projection directory must contain exactly three artifacts")
	}
	expected := map[string]bool{"approved-projection.json": true, "projection-manifest.json": true, "projection-report.json": true}
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil || !expected[entry.Name()] || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return ProjectionManifest{}, nil, errors.New("opportunity pilot: projection directory contains an unexpected artifact")
		}
	}
	manifestBytes, err := readRegular(filepath.Join(projectionRoot, "projection-manifest.json"), MaxManifest)
	if err != nil || decode(manifestBytes, &manifest, MaxManifest) != nil {
		return ProjectionManifest{}, nil, errors.New("opportunity pilot: projection manifest is unavailable or invalid")
	}
	acquisitionManifestBytes, err := readRegular(filepath.Join(acquisitionRoot, "acquisition-manifest.json"), MaxManifest)
	if err != nil || manifest.Format != ProjectionManifestFormat || manifest.WorkOrderDigest != loaded.Digest || manifest.AcquisitionDigest != digest(acquisitionManifestBytes) || manifest.ArchiveDigest != acquisition.ArchiveDigest || manifest.XMLPath == "" || filepath.Base(manifest.XMLPath) != manifest.XMLPath || !strings.EqualFold(filepath.Ext(manifest.XMLPath), ".xml") || !validDigest(manifest.XMLDigest) || manifest.XMLSize == 0 || manifest.XMLSize > loaded.Order.MaximumExpandedXMLBytes || manifest.RawEvidencePublic || manifest.ContactsProjected {
		return ProjectionManifest{}, nil, errors.New("opportunity pilot: projection manifest authority or source identity is invalid")
	}
	projectionBytes, err := readRegular(filepath.Join(projectionRoot, "approved-projection.json"), int64(loaded.Order.MaximumExpandedXMLBytes))
	if err != nil || uint64(len(projectionBytes)) != manifest.ProjectionSize || digest(projectionBytes) != manifest.ProjectionDigest {
		return ProjectionManifest{}, nil, errors.New("opportunity pilot: approved projection does not verify")
	}
	reportBytes, err := readRegular(filepath.Join(projectionRoot, "projection-report.json"), MaxManifest)
	var report ProjectionReport
	if err != nil || digest(reportBytes) != manifest.ReportDigest || decode(reportBytes, &report, MaxManifest) != nil || report.Format != ProjectionReportFormat || !reflect.DeepEqual(report, manifest.Report) || report.SourceRecordsSeen != report.RecordsAccepted+report.RecordsRejected || report.RecordsAccepted == 0 || report.RecordsAccepted > loaded.Order.MaximumRecords {
		return ProjectionManifest{}, nil, errors.New("opportunity pilot: projection report does not reconcile")
	}
	return manifest, projectionBytes, nil
}

func ProjectXML(data []byte, maximumRecords uint64) ([]OpportunityRecord, ProjectionReport, error) {
	report := ProjectionReport{Format: ProjectionReportFormat, ExcludedFields: []FieldCount{}, UnknownFields: []FieldCount{}}
	if len(data) == 0 || uint64(len(data)) > MaximumExpandedXML || maximumRecords == 0 || maximumRecords > MaximumRecords {
		return nil, report, errors.New("opportunity pilot: XML or record bound is invalid")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	var records []OpportunityRecord
	excluded := make(map[string]uint64)
	unknown := make(map[string]uint64)
	depth := 0
	rootSeen := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, report, fmt.Errorf("opportunity pilot: malformed XML: %w", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth++
			if depth > 16 {
				return nil, report, errors.New("opportunity pilot: XML depth exceeds 16")
			}
			if depth == 1 {
				if value.Name.Local != "Grants" {
					return nil, report, errors.New("opportunity pilot: unexpected XML root")
				}
				rootSeen = true
				continue
			}
			if value.Name.Local == "OpportunitySynopsisDetail_1_0" || value.Name.Local == "OpportunityForecastDetail_1_0" {
				report.SourceRecordsSeen++
				if report.SourceRecordsSeen > maximumRecords {
					return nil, report, errors.New("opportunity pilot: source record count exceeds its bound")
				}
				record, err := readOpportunityRecord(decoder, value, report.SourceRecordsSeen-1, excluded, unknown)
				depth--
				if err != nil {
					return nil, report, err
				}
				if err := record.Validate(); err != nil {
					report.RecordsRejected++
					continue
				}
				records = append(records, record)
				report.RecordsAccepted++
			}
		case xml.EndElement:
			depth--
			if depth < 0 {
				return nil, report, errors.New("opportunity pilot: XML depth underflow")
			}
		case xml.Directive:
			return nil, report, errors.New("opportunity pilot: XML directives and DTDs are forbidden")
		case xml.ProcInst:
			if value.Target != "xml" || rootSeen {
				return nil, report, errors.New("opportunity pilot: unexpected XML processing instruction")
			}
		}
	}
	if !rootSeen || depth != 0 || report.SourceRecordsSeen == 0 || report.RecordsAccepted == 0 {
		return nil, report, errors.New("opportunity pilot: XML contains no admissible opportunity records")
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].OpportunityID != records[j].OpportunityID {
			return records[i].OpportunityID < records[j].OpportunityID
		}
		return records[i].OpportunityNumber < records[j].OpportunityNumber
	})
	seenIDs := make(map[string]struct{}, len(records))
	for _, record := range records {
		_, duplicateID := seenIDs[record.OpportunityID]
		if duplicateID {
			return nil, report, errors.New("opportunity pilot: duplicate opportunity ID")
		}
		seenIDs[record.OpportunityID] = struct{}{}
	}
	report.ExcludedFields = fieldCounts(excluded)
	report.UnknownFields = fieldCounts(unknown)
	for field, count := range excluded {
		if excludedContactFields[field] {
			report.ContactFieldsExcluded += count
		} else if field == "Description" {
			report.DescriptionFieldsExcluded += count
		}
	}
	return records, report, nil
}

func readOpportunityRecord(decoder *xml.Decoder, start xml.StartElement, index uint64, excluded, unknown map[string]uint64) (OpportunityRecord, error) {
	record := OpportunityRecord{RecordKind: start.Name.Local, SourceLocator: fmt.Sprintf("/Grants/%s[%d]", start.Name.Local, index), CategoryOfFundingActivity: []string{}, CFDANumbers: []string{}, EligibleApplicants: []string{}}
	seen := make(map[string]bool)
	for {
		token, err := decoder.Token()
		if err != nil {
			return OpportunityRecord{}, fmt.Errorf("opportunity pilot: parse opportunity record: %w", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			name := value.Name.Local
			if excludedContactFields[name] || name == "Description" {
				excluded[name]++
				if err := skipBounded(decoder, value); err != nil {
					return OpportunityRecord{}, err
				}
				continue
			}
			if !approvedFields[name] {
				if len(unknown) >= 256 && unknown[name] == 0 {
					return OpportunityRecord{}, errors.New("opportunity pilot: too many distinct unapproved XML fields")
				}
				unknown[name]++
				if err := skipBounded(decoder, value); err != nil {
					return OpportunityRecord{}, err
				}
				continue
			}
			repeatable := name == "CategoryOfFundingActivity" || name == "CFDANumbers" || name == "EligibleApplicants"
			if seen[name] && !repeatable {
				return OpportunityRecord{}, fmt.Errorf("opportunity pilot: duplicate non-repeatable field %s", name)
			}
			seen[name] = true
			text, err := readScalar(decoder, value, fieldLimit(name))
			if err != nil {
				return OpportunityRecord{}, fmt.Errorf("opportunity pilot: field %s: %w", name, err)
			}
			// An explicitly empty optional scalar carries no lexical value to
			// publish. Treat it like an omitted field; required empty fields are
			// rejected later by record validation. Non-empty whitespace remains
			// part of the exact source lexical value and is never trimmed here.
			if text == "" {
				continue
			}
			if err := assignField(&record, name, text); err != nil {
				return OpportunityRecord{}, err
			}
		case xml.EndElement:
			if value.Name == start.Name {
				return record, nil
			}
			return OpportunityRecord{}, errors.New("opportunity pilot: unexpected record end element")
		case xml.Directive, xml.ProcInst:
			return OpportunityRecord{}, errors.New("opportunity pilot: directives are forbidden inside records")
		case xml.CharData:
			if strings.TrimSpace(string(value)) != "" {
				return OpportunityRecord{}, errors.New("opportunity pilot: unexpected mixed record content")
			}
		}
	}
}

func skipBounded(decoder *xml.Decoder, start xml.StartElement) error {
	depth := 1
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth++
			if depth > 16 {
				return errors.New("opportunity pilot: excluded XML field exceeds depth bound")
			}
		case xml.EndElement:
			depth--
			if depth == 0 && value.Name != start.Name {
				return errors.New("opportunity pilot: excluded XML field has a mismatched end element")
			}
		case xml.Directive, xml.ProcInst:
			return errors.New("opportunity pilot: directive in excluded XML field")
		}
	}
	return nil
}

func readScalar(decoder *xml.Decoder, start xml.StartElement, maximum int) (string, error) {
	var builder strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch value := token.(type) {
		case xml.CharData:
			if builder.Len()+len(value) > maximum {
				return "", errors.New("opportunity pilot: XML scalar exceeds source-specific bound")
			}
			builder.Write(value)
		case xml.StartElement:
			return "", errors.New("opportunity pilot: nested content in scalar field")
		case xml.EndElement:
			if value.Name != start.Name {
				return "", errors.New("opportunity pilot: mismatched scalar end element")
			}
			text := builder.String()
			if !utf8.ValidString(text) || strings.ContainsRune(text, '\x00') {
				return "", errors.New("opportunity pilot: scalar is invalid UTF-8")
			}
			if strings.TrimSpace(text) == "" {
				return "", nil
			}
			return text, nil
		case xml.Directive, xml.ProcInst:
			return "", errors.New("opportunity pilot: directive in scalar field")
		}
	}
}

func (record OpportunityRecord) Validate() error {
	if record.RecordKind != "OpportunitySynopsisDetail_1_0" && record.RecordKind != "OpportunityForecastDetail_1_0" || record.OpportunityID == "" || record.OpportunityNumber == "" || record.OpportunityTitle == "" || record.AgencyCode == "" || record.AgencyName == "" || record.LastUpdatedDate == "" {
		return errors.New("opportunity pilot: required opportunity identity, agency, title, or update date is missing")
	}
	if !digits(record.OpportunityID, 20) || len(record.OpportunityNumber) > 40 || len(record.OpportunityTitle) > 255 || len(record.AgencyCode) > 255 || len(record.AgencyName) > 255 {
		return errors.New("opportunity pilot: opportunity identity field is malformed")
	}
	for _, value := range []string{record.PostDate, record.CloseDate, record.LastUpdatedDate, record.EstimatedSynopsisPostDate, record.EstimatedSynopsisCloseDate, record.EstimatedAwardDate, record.EstimatedProjectStartDate, record.ArchiveDate} {
		if value != "" && !mmddyyyy(value) {
			return errors.New("opportunity pilot: source date is malformed")
		}
	}
	for _, value := range []string{record.ExpectedNumberOfAwards, record.EstimatedTotalProgramFunding, record.AwardCeiling, record.AwardFloor} {
		if value != "" && !digits(value, 20) {
			return errors.New("opportunity pilot: source numeric field is malformed")
		}
	}
	return nil
}

func extractApprovedXML(archive []byte, maximum uint64) (string, []byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil || len(reader.File) == 0 || len(reader.File) > MaxZIPEntries {
		return "", nil, errors.New("opportunity pilot: malformed or overpopulated ZIP archive")
	}
	var xmlFile *zip.File
	var total uint64
	seen := make(map[string]bool)
	for _, file := range reader.File {
		name := file.Name
		if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, "\\\x00") || seen[name] || file.FileInfo().IsDir() || !file.Mode().IsRegular() || file.Flags&1 != 0 || file.Method != zip.Store && file.Method != zip.Deflate {
			return "", nil, errors.New("opportunity pilot: unsafe ZIP entry")
		}
		seen[name] = true
		if file.UncompressedSize64 == 0 || file.UncompressedSize64 > maximum || total > maximum-file.UncompressedSize64 {
			return "", nil, errors.New("opportunity pilot: ZIP expanded size exceeds its bound")
		}
		total += file.UncompressedSize64
		if file.CompressedSize64 > 0 && file.UncompressedSize64 > file.CompressedSize64*200 {
			return "", nil, errors.New("opportunity pilot: ZIP compression ratio exceeds its bound")
		}
		switch strings.ToLower(filepath.Ext(name)) {
		case ".xml":
			if xmlFile != nil {
				return "", nil, errors.New("opportunity pilot: ZIP contains multiple XML data files")
			}
			xmlFile = file
		case ".xsd":
		default:
			return "", nil, errors.New("opportunity pilot: ZIP contains an unapproved entry type")
		}
	}
	if xmlFile == nil {
		return "", nil, errors.New("opportunity pilot: ZIP omits its XML data file")
	}
	handle, err := xmlFile.Open()
	if err != nil {
		return "", nil, err
	}
	defer handle.Close()
	data, err := io.ReadAll(io.LimitReader(handle, int64(maximum)+1))
	if err != nil || uint64(len(data)) != xmlFile.UncompressedSize64 || uint64(len(data)) > maximum {
		return "", nil, errors.New("opportunity pilot: XML expansion failed or exceeded its bound")
	}
	return xmlFile.Name, data, nil
}

func assignField(record *OpportunityRecord, name, value string) error {
	switch name {
	case "OpportunityID":
		record.OpportunityID = value
	case "OpportunityNumber":
		record.OpportunityNumber = value
	case "OpportunityTitle":
		record.OpportunityTitle = value
	case "OpportunityCategory":
		record.OpportunityCategory = value
	case "CategoryOfFundingActivity":
		record.CategoryOfFundingActivity = append(record.CategoryOfFundingActivity, value)
	case "CFDANumbers":
		record.CFDANumbers = append(record.CFDANumbers, value)
	case "EligibleApplicants":
		record.EligibleApplicants = append(record.EligibleApplicants, value)
	case "AdditionalInformationOnEligibility":
		record.AdditionalInformationOnEligibility = value
	case "AgencyCode":
		record.AgencyCode = value
	case "AgencyName":
		record.AgencyName = value
	case "PostDate":
		record.PostDate = value
	case "CloseDate":
		record.CloseDate = value
	case "LastUpdatedDate":
		record.LastUpdatedDate = value
	case "EstimatedSynopsisPostDate":
		record.EstimatedSynopsisPostDate = value
	case "EstimatedSynopsisCloseDate":
		record.EstimatedSynopsisCloseDate = value
	case "EstimatedSynopsisCloseDateExplanation":
		record.EstimatedSynopsisCloseDateExplanation = value
	case "EstimatedAwardDate":
		record.EstimatedAwardDate = value
	case "EstimatedProjectStartDate":
		record.EstimatedProjectStartDate = value
	case "ExpectedNumberOfAwards":
		record.ExpectedNumberOfAwards = value
	case "EstimatedTotalProgramFunding":
		record.EstimatedTotalProgramFunding = value
	case "AwardCeiling":
		record.AwardCeiling = value
	case "AwardFloor":
		record.AwardFloor = value
	case "ArchiveDate":
		record.ArchiveDate = value
	case "CostSharingOrMatchingRequirement":
		record.CostSharingOrMatchingRequirement = value
	case "Version":
		record.Version = value
	default:
		return errors.New("opportunity pilot: internal approved-field mismatch")
	}
	return nil
}

func fieldLimit(name string) int {
	switch name {
	case "AdditionalInformationOnEligibility", "EstimatedSynopsisCloseDateExplanation":
		return 4000
	case "OpportunityTitle", "AgencyCode", "AgencyName":
		return 255
	case "OpportunityNumber":
		return 40
	default:
		return MaxRecordScalar
	}
}

func fieldCounts(values map[string]uint64) []FieldCount {
	result := make([]FieldCount, 0, len(values))
	for field, count := range values {
		result = append(result, FieldCount{Field: field, Count: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Field < result[j].Field })
	return result
}

func digits(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func mmddyyyy(value string) bool {
	if len(value) != 8 || !digits(value, 8) {
		return false
	}
	month, _ := strconv.Atoi(value[:2])
	day, _ := strconv.Atoi(value[2:4])
	year, _ := strconv.Atoi(value[4:])
	return month >= 1 && month <= 12 && day >= 1 && day <= 31 && year >= 1900 && year <= 2200
}

func projectionContainsForbidden(data []byte) bool {
	lower := strings.ToLower(string(data))
	for _, term := range []string{"grantorcontact", "agencycontact", "publisheruid", "\"token\"", "description\""} {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}
