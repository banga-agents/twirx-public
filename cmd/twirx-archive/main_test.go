package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/archiveimport"
)

func TestPlanAndInspectAreOfflineAndBoundToWorkOrder(t *testing.T) {
	root := t.TempDir()
	writeCommandWorkOrder(t, root)
	var output bytes.Buffer
	if err := run([]string{"plan", "--work-orders", root, "--id", "example-archive"}, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var plan planOutput
	if err := json.Unmarshal(output.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.NetworkRequestsMade != 0 || len(plan.Entries) != 1 || !strings.HasPrefix(plan.Entries[0].IndexURL, "https://index.commoncrawl.org/CC-MAIN-2026-30-index?") {
		t.Fatalf("unexpected offline plan: %+v", plan)
	}

	line := []byte(`{"url":"https://example.org/data","timestamp":"20260801010203","status":"200","mime":"text/plain","digest":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","filename":"crawl-data/CC-MAIN-2026-30/segments/1/warc/example.warc.gz","offset":"1","length":"8"}` + "\n")
	indexPath := filepath.Join(root, "index.jsonl")
	if err := os.WriteFile(indexPath, line, 0o600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := run([]string{"inspect-index", "--work-orders", root, "--id", "example-archive", "--collection", "CC-MAIN-2026-30", "--route", "https://example.org/data", "--response", indexPath}, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var inspection indexOutput
	if err := json.Unmarshal(output.Bytes(), &inspection); err != nil {
		t.Fatal(err)
	}
	if inspection.NetworkRequestsMade != 0 || len(inspection.Captures) != 1 || inspection.Captures[0].Length != 8 {
		t.Fatalf("unexpected inspection: %+v", inspection)
	}
}

func TestImportPreservesMalformedRawEvidenceBeforeParserFailure(t *testing.T) {
	root := t.TempDir()
	writeCommandWorkOrder(t, root)
	indexPath := filepath.Join(root, "index.jsonl")
	line := []byte(`{"url":"https://example.org/data","timestamp":"20260801010203","status":"200","mime":"text/plain","digest":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","filename":"crawl-data/CC-MAIN-2026-30/segments/1/warc/example.warc.gz","offset":"1","length":"8"}` + "\n")
	if err := os.WriteFile(indexPath, line, 0o600); err != nil {
		t.Fatal(err)
	}
	warcPath := filepath.Join(root, "range.gz")
	if err := os.WriteFile(warcPath, []byte("not-gzip"), 0o600); err != nil {
		t.Fatal(err)
	}
	spool := filepath.Join(root, "spool")
	err := run([]string{"import", "--work-orders", root, "--id", "example-archive", "--collection", "CC-MAIN-2026-30", "--route", "https://example.org/data", "--response", indexPath, "--capture", "0", "--warc", warcPath, "--http-status", "206", "--content-range", "bytes 1-8/100", "--out", spool}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("malformed WARC unexpectedly imported")
	}
	if _, statErr := os.Stat(filepath.Join(spool, "evidence-manifest.json")); statErr != nil {
		t.Fatalf("raw evidence was not sealed before parsing failed: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(spool, "manifest.json")); !os.IsNotExist(statErr) {
		t.Fatal("failed import published a final manifest")
	}
}

func writeCommandWorkOrder(t *testing.T, root string) {
	t.Helper()
	order := archiveimport.WorkOrder{
		Format: archiveimport.WorkOrderFormat, ID: "example-archive", OriginID: "example-org", CanonicalOrigin: "https://example.org",
		PermittedRoutes: []string{"https://example.org/data"}, CollectionIDs: []string{"CC-MAIN-2026-30"},
		SelectedCaptures:      []archiveimport.CaptureSelection{{CollectionID: "CC-MAIN-2026-30", Route: "https://example.org/data", Timestamp: "20260801010203", ProviderDigest: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", Filename: "crawl-data/CC-MAIN-2026-30/segments/1/warc/example.warc.gz", Offset: 1, Length: 8}},
		CapturesPerCollection: 1, IndexRequestBudget: 1,
		MaxIndexResponseBytes: archiveimport.MaxIndexResponse, MaxCompressedRecordBytes: archiveimport.MaxCompressed, MaxDecompressedRecordBytes: archiveimport.MaxDecompressed, MaxRetainedBodyBytes: archiveimport.MaxRetainedBody,
		PolicyReviewState: "completed", PolicyDecision: "profile_only", PolicyEvidenceDigest: "sha256:" + strings.Repeat("a", 64), DecisionDigest: "sha256:" + strings.Repeat("b", 64),
		ApprovalReference: "atlas/admissions/example-org/decision.json", ApprovedBy: "Genesis steward", ApprovedAt: "2026-08-11T00:00:00Z",
		EvidenceClass: "archive_observation", Freshness: "historical", CurrentPublisherStatement: false, ObservedBy: "common_crawl",
	}
	data, err := json.MarshalIndent(order, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, order.ID+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}
