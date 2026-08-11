package archiveimport

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
)

const (
	EvidenceManifestFormat = "tw.archive-evidence-manifest/0.1"
	RangeEvidenceFormat    = "tw.archive-range-evidence/0.1"
	CaptureEvidenceFormat  = "tw.archive-capture-evidence/0.1"
	CaptureManifestFormat  = "tw.archive-capture-manifest/0.1"
	MaxMetadata            = 256 << 10
)

type Artifact struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   uint64 `json:"size"`
}

type EvidenceManifest struct {
	Format          string     `json:"format"`
	WorkOrderID     string     `json:"work_order_id"`
	WorkOrderDigest string     `json:"work_order_digest"`
	Artifacts       []Artifact `json:"artifacts"`
}

type RangeEvidence struct {
	Format         string `json:"format"`
	DataURL        string `json:"data_url"`
	RequestedRange string `json:"requested_range"`
	HTTPStatus     uint64 `json:"http_status"`
	ContentRange   string `json:"content_range"`
	BodyDigest     string `json:"body_digest"`
	BodySize       uint64 `json:"body_size"`
}

type CaptureEvidence struct {
	Format                    string `json:"format"`
	WorkOrderID               string `json:"work_order_id"`
	WorkOrderDigest           string `json:"work_order_digest"`
	OriginID                  string `json:"origin_id"`
	EvidenceClass             string `json:"evidence_class"`
	Freshness                 string `json:"freshness"`
	CurrentPublisherStatement bool   `json:"current_publisher_statement"`
	ObservedBy                string `json:"observed_by"`
	CollectionID              string `json:"collection_id"`
	CaptureTimestamp          string `json:"capture_timestamp"`
	TargetURL                 string `json:"target_url"`
	WARCFile                  string `json:"warc_filename"`
	WARCOffset                uint64 `json:"warc_offset"`
	WARCLength                uint64 `json:"warc_length"`
	RangeHTTPStatus           uint64 `json:"range_http_status"`
	RangeContentRange         string `json:"range_content_range"`
	ProviderPayloadDigest     string `json:"provider_payload_digest"`
	CompressedRecordDigest    string `json:"compressed_record_digest"`
	RepresentationDigest      string `json:"representation_digest"`
	RepresentationSize        uint64 `json:"representation_size"`
	HTTPStatus                uint64 `json:"http_status"`
	MediaType                 string `json:"media_type"`
	ContentEncoding           string `json:"content_encoding"`
	ContentLanguage           string `json:"content_language"`
}

type CaptureManifest struct {
	Format          string     `json:"format"`
	WorkOrderID     string     `json:"work_order_id"`
	WorkOrderDigest string     `json:"work_order_digest"`
	EvidenceDigest  string     `json:"evidence_manifest_digest"`
	Artifacts       []Artifact `json:"artifacts"`
}

// PublishCapture writes the acquired work order, exact index record, and WARC
// range first. evidence-manifest.json is published before any WARC or HTTP
// parsing. The derived representation and final manifest are published only
// after the raw evidence verifies and parses within the sealed bounds.
func PublishCapture(output string, loaded *LoadedWorkOrder, capture Capture, rangeStatus int, contentRange string, compressed []byte) (*CaptureEvidence, error) {
	if loaded == nil || loaded.Order.Validate() != nil || digest(loaded.Bytes) != loaded.Digest || !loaded.Order.permitsCollection(capture.CollectionID) || !loaded.Order.permitsRoute(capture.RequestedURL) || uint64(len(compressed)) != capture.Length {
		return nil, errors.New("archiveimport: publication inputs are not sealed and self-consistent")
	}
	if err := ValidateRangeResponse(capture, rangeStatus, contentRange, compressed, loaded.Order.MaxCompressedRecordBytes); err != nil {
		return nil, err
	}
	if len(capture.raw) == 0 || len(capture.raw) > MaxIndexLine {
		return nil, errors.New("archiveimport: exact index record is unavailable")
	}
	root, err := createOutput(output)
	if err != nil {
		return nil, err
	}
	indexBytes := append(append([]byte(nil), capture.raw...), '\n')
	dataURL, err := capture.DataURL()
	if err != nil {
		return nil, err
	}
	rangeHeader, err := capture.RangeHeader()
	if err != nil {
		return nil, err
	}
	rangeEvidence := RangeEvidence{Format: RangeEvidenceFormat, DataURL: dataURL, RequestedRange: rangeHeader, HTTPStatus: uint64(rangeStatus), ContentRange: contentRange, BodyDigest: digest(compressed), BodySize: uint64(len(compressed))}
	if err := rangeEvidence.Validate(capture, compressed, loaded.Order.MaxCompressedRecordBytes); err != nil {
		return nil, err
	}
	rangeBytes, err := marshal(rangeEvidence, MaxMetadata)
	if err != nil {
		return nil, err
	}
	for _, artifact := range []struct {
		path string
		data []byte
		max  int
	}{
		{path: "work-order.json", data: loaded.Bytes, max: MaxWorkOrder},
		{path: "index-record.json", data: indexBytes, max: MaxIndexLine + 1},
		{path: "range-response.json", data: rangeBytes, max: MaxMetadata},
		{path: "warc-record.gz", data: compressed, max: int(loaded.Order.MaxCompressedRecordBytes)},
	} {
		if err := atomicfile.Write(filepath.Join(root, artifact.path), artifact.data, artifact.max, 0o440); err != nil {
			return nil, err
		}
	}
	evidenceManifest := EvidenceManifest{Format: EvidenceManifestFormat, WorkOrderID: loaded.Order.ID, WorkOrderDigest: loaded.Digest, Artifacts: []Artifact{
		artifactFor("index-record.json", indexBytes), artifactFor("range-response.json", rangeBytes), artifactFor("warc-record.gz", compressed), artifactFor("work-order.json", loaded.Bytes),
	}}
	if err := evidenceManifest.Validate(); err != nil {
		return nil, err
	}
	evidenceBytes, err := marshal(evidenceManifest, MaxMetadata)
	if err != nil {
		return nil, err
	}
	// Evidence-manifest-last makes acquisition completion explicit before the
	// untrusted WARC record enters any parser.
	if err := atomicfile.Write(filepath.Join(root, "evidence-manifest.json"), evidenceBytes, MaxMetadata, 0o440); err != nil {
		return nil, err
	}
	verifiedOrder, verifiedCapture, verifiedCompressed, err := verifyRawEvidence(root)
	if err != nil {
		return nil, err
	}
	parsed, err := ParseCompressedRecord(verifiedCompressed, verifiedCapture, verifiedOrder.Order)
	if err != nil {
		return nil, err
	}
	if err := atomicfile.Write(filepath.Join(root, "representation.body"), parsed.Body, int(verifiedOrder.Order.MaxRetainedBodyBytes), 0o440); err != nil {
		return nil, err
	}
	captureEvidence := CaptureEvidence{Format: CaptureEvidenceFormat, WorkOrderID: verifiedOrder.Order.ID, WorkOrderDigest: verifiedOrder.Digest, OriginID: verifiedOrder.Order.OriginID, EvidenceClass: "archive_observation", Freshness: "historical", CurrentPublisherStatement: false, ObservedBy: "common_crawl", CollectionID: verifiedCapture.CollectionID, CaptureTimestamp: verifiedCapture.Timestamp, TargetURL: parsed.TargetURL, WARCFile: verifiedCapture.Filename, WARCOffset: verifiedCapture.Offset, WARCLength: verifiedCapture.Length, RangeHTTPStatus: rangeEvidence.HTTPStatus, RangeContentRange: rangeEvidence.ContentRange, ProviderPayloadDigest: parsed.ProviderDigest, CompressedRecordDigest: digest(verifiedCompressed), RepresentationDigest: parsed.BodyDigest, RepresentationSize: uint64(len(parsed.Body)), HTTPStatus: parsed.HTTPStatus, MediaType: parsed.MediaType, ContentEncoding: parsed.ContentEncoding, ContentLanguage: parsed.ContentLanguage}
	if err := captureEvidence.Validate(verifiedOrder.Order); err != nil {
		return nil, err
	}
	captureBytes, err := marshal(captureEvidence, MaxMetadata)
	if err != nil {
		return nil, err
	}
	if err := atomicfile.Write(filepath.Join(root, "capture.json"), captureBytes, MaxMetadata, 0o440); err != nil {
		return nil, err
	}
	manifest := CaptureManifest{Format: CaptureManifestFormat, WorkOrderID: verifiedOrder.Order.ID, WorkOrderDigest: verifiedOrder.Digest, EvidenceDigest: digest(evidenceBytes), Artifacts: []Artifact{
		artifactFor("capture.json", captureBytes), artifactFor("evidence-manifest.json", evidenceBytes), artifactFor("index-record.json", indexBytes), artifactFor("range-response.json", rangeBytes), artifactFor("representation.body", parsed.Body), artifactFor("warc-record.gz", verifiedCompressed), artifactFor("work-order.json", verifiedOrder.Bytes),
	}}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	manifestBytes, err := marshal(manifest, MaxMetadata)
	if err != nil {
		return nil, err
	}
	// Final-manifest-last defines a complete, independently admissible capture.
	if err := atomicfile.Write(filepath.Join(root, "manifest.json"), manifestBytes, MaxMetadata, 0o440); err != nil {
		return nil, err
	}
	return &captureEvidence, nil
}

func VerifySpool(root string) (*CaptureEvidence, error) {
	manifestBytes, err := readRegular(filepath.Join(root, "manifest.json"), MaxMetadata)
	if err != nil {
		return nil, fmt.Errorf("archiveimport: final manifest unavailable: %w", err)
	}
	var manifest CaptureManifest
	if err := decodeStrict(manifestBytes, &manifest, MaxMetadata); err != nil || manifest.Validate() != nil {
		return nil, errors.New("archiveimport: invalid final manifest")
	}
	for _, artifact := range manifest.Artifacts {
		data, err := readSpoolArtifact(root, artifact.Path)
		if err != nil || digest(data) != artifact.Digest || uint64(len(data)) != artifact.Size {
			return nil, fmt.Errorf("archiveimport: final artifact mismatch for %s", artifact.Path)
		}
	}
	loaded, capture, compressed, err := verifyRawEvidence(root)
	if err != nil || loaded.Digest != manifest.WorkOrderDigest {
		return nil, errors.New("archiveimport: raw evidence does not reconcile with the final manifest")
	}
	evidenceBytes, err := readRegular(filepath.Join(root, "evidence-manifest.json"), MaxMetadata)
	if err != nil || digest(evidenceBytes) != manifest.EvidenceDigest {
		return nil, errors.New("archiveimport: evidence manifest identity mismatch")
	}
	parsed, err := ParseCompressedRecord(compressed, capture, loaded.Order)
	if err != nil {
		return nil, err
	}
	body, err := readRegular(filepath.Join(root, "representation.body"), int64(loaded.Order.MaxRetainedBodyBytes))
	if err != nil || digest(body) != parsed.BodyDigest || string(body) != string(parsed.Body) {
		return nil, errors.New("archiveimport: derived representation does not match the verified WARC evidence")
	}
	captureBytes, err := readRegular(filepath.Join(root, "capture.json"), MaxMetadata)
	if err != nil {
		return nil, err
	}
	var evidence CaptureEvidence
	rangeBytes, rangeErr := readRegular(filepath.Join(root, "range-response.json"), MaxMetadata)
	var rangeEvidence RangeEvidence
	if rangeErr != nil || decodeStrict(rangeBytes, &rangeEvidence, MaxMetadata) != nil || rangeEvidence.Validate(capture, compressed, loaded.Order.MaxCompressedRecordBytes) != nil {
		return nil, errors.New("archiveimport: range evidence does not reconcile")
	}
	if err := decodeStrict(captureBytes, &evidence, MaxMetadata); err != nil || evidence.Validate(loaded.Order) != nil || evidence.WorkOrderDigest != loaded.Digest || evidence.OriginID != loaded.Order.OriginID || evidence.CollectionID != capture.CollectionID || evidence.CaptureTimestamp != capture.Timestamp || evidence.TargetURL != capture.CapturedURL || evidence.WARCFile != capture.Filename || evidence.WARCOffset != capture.Offset || evidence.WARCLength != capture.Length || evidence.RangeHTTPStatus != rangeEvidence.HTTPStatus || evidence.RangeContentRange != rangeEvidence.ContentRange || evidence.CompressedRecordDigest != digest(compressed) || evidence.RepresentationDigest != parsed.BodyDigest || evidence.RepresentationSize != uint64(len(parsed.Body)) || evidence.ProviderPayloadDigest != parsed.ProviderDigest || evidence.HTTPStatus != parsed.HTTPStatus || evidence.MediaType != parsed.MediaType || evidence.ContentEncoding != parsed.ContentEncoding || evidence.ContentLanguage != parsed.ContentLanguage {
		return nil, errors.New("archiveimport: capture metadata does not reconcile")
	}
	return &evidence, nil
}

func verifyRawEvidence(root string) (*LoadedWorkOrder, Capture, []byte, error) {
	evidenceBytes, err := readRegular(filepath.Join(root, "evidence-manifest.json"), MaxMetadata)
	if err != nil {
		return nil, Capture{}, nil, fmt.Errorf("archiveimport: evidence manifest unavailable: %w", err)
	}
	var manifest EvidenceManifest
	if err := decodeStrict(evidenceBytes, &manifest, MaxMetadata); err != nil || manifest.Validate() != nil {
		return nil, Capture{}, nil, errors.New("archiveimport: invalid evidence manifest")
	}
	for _, artifact := range manifest.Artifacts {
		data, err := readSpoolArtifact(root, artifact.Path)
		if err != nil || digest(data) != artifact.Digest || uint64(len(data)) != artifact.Size {
			return nil, Capture{}, nil, fmt.Errorf("archiveimport: raw evidence mismatch for %s", artifact.Path)
		}
	}
	orderBytes, err := readRegular(filepath.Join(root, "work-order.json"), MaxWorkOrder)
	if err != nil || digest(orderBytes) != manifest.WorkOrderDigest {
		return nil, Capture{}, nil, errors.New("archiveimport: spooled work order does not match its authority digest")
	}
	var order WorkOrder
	if err := decodeStrict(orderBytes, &order, MaxWorkOrder); err != nil || order.ID != manifest.WorkOrderID || order.Validate() != nil {
		return nil, Capture{}, nil, errors.New("archiveimport: invalid spooled work order")
	}
	indexBytes, err := readRegular(filepath.Join(root, "index-record.json"), MaxIndexLine+1)
	if err != nil {
		return nil, Capture{}, nil, err
	}
	var captures []Capture
	for _, collection := range order.CollectionIDs {
		for _, route := range order.PermittedRoutes {
			parsed, parseErr := ParseIndexResponse(indexBytes, order, collection, route)
			if parseErr == nil {
				captures = append(captures, parsed...)
			}
		}
	}
	if len(captures) != 1 {
		return nil, Capture{}, nil, errors.New("archiveimport: exact index record does not resolve uniquely inside the work order")
	}
	compressed, err := readRegular(filepath.Join(root, "warc-record.gz"), int64(order.MaxCompressedRecordBytes))
	if err != nil || uint64(len(compressed)) != captures[0].Length {
		return nil, Capture{}, nil, errors.New("archiveimport: compressed WARC evidence length mismatch")
	}
	rangeBytes, err := readRegular(filepath.Join(root, "range-response.json"), MaxMetadata)
	var rangeEvidence RangeEvidence
	if err != nil || decodeStrict(rangeBytes, &rangeEvidence, MaxMetadata) != nil || rangeEvidence.Validate(captures[0], compressed, order.MaxCompressedRecordBytes) != nil {
		return nil, Capture{}, nil, errors.New("archiveimport: invalid spooled range-response evidence")
	}
	return &LoadedWorkOrder{Order: order, Digest: manifest.WorkOrderDigest, Bytes: orderBytes}, captures[0], compressed, nil
}

func (m EvidenceManifest) Validate() error {
	return validateArtifacts(m.Format == EvidenceManifestFormat, m.WorkOrderID, m.WorkOrderDigest, m.Artifacts, []string{"index-record.json", "range-response.json", "warc-record.gz", "work-order.json"})
}

func (m CaptureManifest) Validate() error {
	if !validDigest(m.EvidenceDigest) {
		return errors.New("archiveimport: invalid evidence-manifest digest")
	}
	return validateArtifacts(m.Format == CaptureManifestFormat, m.WorkOrderID, m.WorkOrderDigest, m.Artifacts, []string{"capture.json", "evidence-manifest.json", "index-record.json", "range-response.json", "representation.body", "warc-record.gz", "work-order.json"})
}

func (r RangeEvidence) Validate(capture Capture, compressed []byte, maximum uint64) error {
	dataURL, dataErr := capture.DataURL()
	rangeHeader, rangeErr := capture.RangeHeader()
	if r.Format != RangeEvidenceFormat || dataErr != nil || rangeErr != nil || r.DataURL != dataURL || r.RequestedRange != rangeHeader || r.HTTPStatus != 206 || !validPlainText(r.ContentRange, 1024) || r.BodyDigest != digest(compressed) || r.BodySize != uint64(len(compressed)) || ValidateRangeResponse(capture, int(r.HTTPStatus), r.ContentRange, compressed, maximum) != nil {
		return errors.New("archiveimport: invalid range-response evidence")
	}
	return nil
}

func validateArtifacts(formatOK bool, orderID, orderDigest string, artifacts []Artifact, expected []string) error {
	if !formatOK || !idPattern.MatchString(orderID) || !validDigest(orderDigest) || len(artifacts) != len(expected) {
		return errors.New("archiveimport: invalid manifest metadata")
	}
	for index, artifact := range artifacts {
		if artifact.Path != expected[index] || !validDigest(artifact.Digest) || artifact.Size == 0 {
			return errors.New("archiveimport: manifest paths, digests, or sizes are invalid")
		}
	}
	return nil
}

func (e CaptureEvidence) Validate(order WorkOrder) error {
	parsedTime, timeErr := time.Parse("20060102150405", e.CaptureTimestamp)
	if err := order.Validate(); err != nil || e.Format != CaptureEvidenceFormat || e.WorkOrderID != order.ID || !validDigest(e.WorkOrderDigest) || e.OriginID != order.OriginID || e.EvidenceClass != "archive_observation" || e.Freshness != "historical" || e.CurrentPublisherStatement || e.ObservedBy != "common_crawl" || !order.permitsCollection(e.CollectionID) || !order.permitsRoute(e.TargetURL) || timeErr != nil || parsedTime.UTC().Format("20060102150405") != e.CaptureTimestamp || !validWARCPath(e.WARCFile, e.CollectionID) || e.WARCLength == 0 || e.WARCLength > order.MaxCompressedRecordBytes || e.WARCOffset > ^uint64(0)-(e.WARCLength-1) || e.RangeHTTPStatus != 206 || !validPlainText(e.RangeContentRange, 1024) || !providerDigestPattern.MatchString(e.ProviderPayloadDigest) || !validDigest(e.CompressedRecordDigest) || !validDigest(e.RepresentationDigest) || e.RepresentationSize == 0 || e.RepresentationSize > order.MaxRetainedBodyBytes || e.HTTPStatus != 200 || !validPlainText(e.MediaType, 255) || len(e.ContentEncoding) > 1024 || len(e.ContentLanguage) > 1024 || strings.ContainsAny(e.ContentEncoding, "\r\n\x00") || strings.ContainsAny(e.ContentLanguage, "\r\n\x00") {
		return errors.New("archiveimport: invalid capture evidence")
	}
	capture := Capture{CollectionID: e.CollectionID, RequestedURL: e.TargetURL, CapturedURL: e.TargetURL, Timestamp: e.CaptureTimestamp, Status: e.HTTPStatus, MediaType: e.MediaType, ProviderDigest: e.ProviderPayloadDigest, Filename: e.WARCFile, Offset: e.WARCOffset, Length: e.WARCLength}
	if err := validateRangeMetadata(capture, int(e.RangeHTTPStatus), e.RangeContentRange, e.WARCLength, order.MaxCompressedRecordBytes); err != nil {
		return errors.New("archiveimport: capture evidence range does not reconcile")
	}
	return nil
}

func createOutput(path string) (string, error) {
	if path == "" {
		return "", errors.New("archiveimport: output path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(absolute)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("archiveimport: output parent must be a real directory")
	}
	if _, err := os.Lstat(absolute); err == nil {
		return "", errors.New("archiveimport: immutable output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Mkdir(absolute, 0o750); err != nil {
		return "", err
	}
	return absolute, nil
}

func artifactFor(path string, data []byte) Artifact {
	return Artifact{Path: path, Digest: digest(data), Size: uint64(len(data))}
}

func readSpoolArtifact(root, relative string) ([]byte, error) {
	maximum := int64(MaxMetadata)
	switch relative {
	case "warc-record.gz":
		maximum = MaxCompressed
	case "representation.body":
		maximum = MaxRetainedBody
	case "work-order.json":
		maximum = MaxWorkOrder
	case "index-record.json":
		maximum = MaxIndexLine + 1
	}
	if strings.Contains(relative, "..") || filepath.Base(relative) != relative {
		return nil, errors.New("archiveimport: unsafe spool artifact path")
	}
	return readRegular(filepath.Join(root, relative), maximum)
}
