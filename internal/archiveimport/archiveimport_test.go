package archiveimport

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/base32"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWorkOrderFailsClosedWithoutHumanPolicyAuthority(t *testing.T) {
	valid := testOrder()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid work order rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*WorkOrder)
	}{
		{"pending review", func(order *WorkOrder) { order.PolicyReviewState = "pending" }},
		{"uncertain decision", func(order *WorkOrder) { order.PolicyDecision = "uncertain" }},
		{"missing reviewer", func(order *WorkOrder) { order.ApprovedBy = "" }},
		{"reviewer control character", func(order *WorkOrder) { order.ApprovedBy = "reviewer\nforged" }},
		{"approval reference control character", func(order *WorkOrder) { order.ApprovalReference = "decision.json\x00forged" }},
		{"approval reference traversal", func(order *WorkOrder) { order.ApprovalReference = "../../decision.json" }},
		{"missing policy digest", func(order *WorkOrder) { order.PolicyEvidenceDigest = "" }},
		{"current claim", func(order *WorkOrder) { order.CurrentPublisherStatement = true }},
		{"wrong evidence class", func(order *WorkOrder) { order.EvidenceClass = "current_publisher_statement" }},
		{"cross-origin route", func(order *WorkOrder) { order.PermittedRoutes = []string{"https://other.example/data"} }},
		{"embedded credentials", func(order *WorkOrder) { order.PermittedRoutes = []string{"https://user:pass@example.org/data"} }},
		{"non HTTPS route", func(order *WorkOrder) { order.PermittedRoutes = []string{"http://example.org/data"} }},
		{"origin path", func(order *WorkOrder) { order.CanonicalOrigin = "https://example.org/private" }},
		{"unbounded periods", func(order *WorkOrder) {
			order.CollectionIDs = []string{"CC-MAIN-2026-21", "CC-MAIN-2026-25", "CC-MAIN-2026-29"}
		}},
		{"unbudgeted requests", func(order *WorkOrder) { order.IndexRequestBudget = 0 }},
		{"oversized range", func(order *WorkOrder) { order.MaxCompressedRecordBytes = MaxCompressed + 1 }},
		{"missing exact capture", func(order *WorkOrder) { order.SelectedCaptures = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			order := valid
			order.PermittedRoutes = append([]string(nil), valid.PermittedRoutes...)
			order.CollectionIDs = append([]string(nil), valid.CollectionIDs...)
			order.SelectedCaptures = append([]CaptureSelection(nil), valid.SelectedCaptures...)
			test.mutate(&order)
			if err := order.Validate(); err == nil {
				t.Fatal("adversarial work order was accepted")
			}
		})
	}
}

func TestIndexAndRangeStayInsideSealedOfficialRoutes(t *testing.T) {
	order := testOrder()
	body := []byte("public source statement")
	compressed, providerDigest := testWARC(t, order.PermittedRoutes[0], "20260801010203", body, nil)
	bindSelection(&order, "20260801010203", providerDigest, uint64(len(compressed)), 4096)
	indexURL, err := BuildIndexURL(order, order.CollectionIDs[0], order.PermittedRoutes[0])
	if err != nil || !strings.HasPrefix(indexURL, "https://index.commoncrawl.org/CC-MAIN-2026-30-index?") || !strings.Contains(indexURL, "matchType=exact") || !strings.Contains(indexURL, "timestamp%3A20260801010203") {
		t.Fatalf("unexpected index URL %q: %v", indexURL, err)
	}
	if _, err := BuildIndexURL(order, order.CollectionIDs[0], "https://example.org/unreviewed"); err == nil {
		t.Fatal("unreviewed route produced an index query")
	}
	line := testIndexLine(t, order, "20260801010203", providerDigest, uint64(len(compressed)), 4096)
	captures, err := ParseIndexResponse(line, order, order.CollectionIDs[0], order.PermittedRoutes[0])
	if err != nil || len(captures) != 1 {
		t.Fatalf("valid index record rejected: %#v %v", captures, err)
	}
	capture := captures[0]
	dataURL, err := capture.DataURL()
	if err != nil || !strings.HasPrefix(dataURL, "https://data.commoncrawl.org/crawl-data/CC-MAIN-2026-30/") {
		t.Fatalf("unexpected data URL %q: %v", dataURL, err)
	}
	rangeHeader, err := capture.RangeHeader()
	if err != nil || rangeHeader != "bytes=4096-"+uintToString(4096+uint64(len(compressed))-1) {
		t.Fatalf("unexpected range header %q: %v", rangeHeader, err)
	}
	contentRange := "bytes 4096-" + uintToString(4096+uint64(len(compressed))-1) + "/999999"
	if err := ValidateRangeResponse(capture, 206, contentRange, compressed, order.MaxCompressedRecordBytes); err != nil {
		t.Fatalf("valid range rejected: %v", err)
	}
	for _, test := range []struct {
		status     int
		rangeValue string
		body       []byte
	}{
		{200, contentRange, compressed},
		{206, "bytes 0-1/2", compressed},
		{206, contentRange, compressed[:len(compressed)-1]},
		{206, strings.TrimSuffix(contentRange, "999999") + "*", compressed},
	} {
		if err := ValidateRangeResponse(capture, test.status, test.rangeValue, test.body, order.MaxCompressedRecordBytes); err == nil {
			t.Fatalf("invalid range response accepted: %+v", test)
		}
	}
}

func TestIndexRejectsDuplicateAmbiguousAndUnreviewedCaptures(t *testing.T) {
	order := testOrder()
	body := []byte("evidence")
	compressed, provider := testWARC(t, order.PermittedRoutes[0], "20260801010203", body, nil)
	bindSelection(&order, "20260801010203", provider, uint64(len(compressed)), 1)
	valid := testIndexLine(t, order, "20260801010203", provider, uint64(len(compressed)), 1)
	duplicate := append(append([]byte(nil), valid...), valid...)
	if _, err := ParseIndexResponse(duplicate, order, order.CollectionIDs[0], order.PermittedRoutes[0]); err == nil {
		t.Fatal("duplicate capture accepted")
	}
	duplicateKey := bytes.Replace(valid, []byte(`"url":`), []byte(`"url":"https://example.org/data","url":`), 1)
	if _, err := ParseIndexResponse(duplicateKey, order, order.CollectionIDs[0], order.PermittedRoutes[0]); err == nil {
		t.Fatal("duplicate JSON key accepted")
	}
	tighter := order
	tighter.MaxCompressedRecordBytes = uint64(len(compressed) - 1)
	if _, err := ParseIndexResponse(valid, tighter, tighter.CollectionIDs[0], tighter.PermittedRoutes[0]); err == nil {
		t.Fatal("capture outside the sealed byte budget accepted")
	}
	var document map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(valid), &document); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []func(map[string]any){
		func(value map[string]any) { value["url"] = "https://other.example/data" },
		func(value map[string]any) { value["filename"] = "crawl-data/CC-MAIN-2026-29/segments/x.warc.gz" },
		func(value map[string]any) { value["filename"] = "../../private.warc.gz" },
		func(value map[string]any) { value["status"] = "302" },
		func(value map[string]any) { value["offset"] = "-1" },
		func(value map[string]any) { value["length"] = uint64(MaxCompressed + 1) },
		func(value map[string]any) { value["digest"] = "not-a-digest" },
	} {
		copy := cloneMap(document)
		mutation(copy)
		encoded, _ := json.Marshal(copy)
		if _, err := ParseIndexResponse(append(encoded, '\n'), order, order.CollectionIDs[0], order.PermittedRoutes[0]); err == nil {
			t.Fatalf("mutated index record accepted: %s", encoded)
		}
	}
	first := cloneMap(document)
	second := cloneMap(document)
	second["offset"] = "999"
	firstBytes, _ := json.Marshal(first)
	secondBytes, _ := json.Marshal(second)
	if _, err := ParseIndexResponse(append(append(firstBytes, '\n'), append(secondBytes, '\n')...), order, order.CollectionIDs[0], order.PermittedRoutes[0]); err == nil {
		t.Fatal("unselected ambiguous index response was accepted")
	}
}

func TestEvidenceIsPublishedBeforeParsingAndFinalSpoolVerifies(t *testing.T) {
	order := testOrder()
	body := []byte("source-native lexical value")
	compressed, provider := testWARC(t, order.PermittedRoutes[0], "20260801010203", body, nil)
	bindSelection(&order, "20260801010203", provider, uint64(len(compressed)), 1024)
	loaded := loadedTestOrder(t, order)
	line := testIndexLine(t, order, "20260801010203", provider, uint64(len(compressed)), 1024)
	captures, err := ParseIndexResponse(line, order, order.CollectionIDs[0], order.PermittedRoutes[0])
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "capture")
	evidence, err := PublishCapture(output, loaded, captures[0], 206, testContentRange(captures[0]), compressed)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.EvidenceClass != "archive_observation" || evidence.Freshness != "historical" || evidence.CurrentPublisherStatement || evidence.RepresentationDigest != digest(body) {
		t.Fatalf("unexpected capture evidence: %+v", evidence)
	}
	verified, err := VerifySpool(output)
	if err != nil || *verified != *evidence {
		t.Fatalf("spool verification mismatch: %#v %#v %v", verified, evidence, err)
	}
	if err := os.Chmod(filepath.Join(output, "representation.body"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "representation.body"), []byte("tampered native value"), 0o440); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySpool(output); err == nil {
		t.Fatal("tampered representation was accepted")
	}

	malformedCapture := captures[0]
	malformed := []byte("not-gzip")
	malformedCapture.Length = uint64(len(malformed))
	malformedCapture.raw = testIndexLine(t, order, malformedCapture.Timestamp, provider, uint64(len(malformed)), malformedCapture.Offset)
	malformedOutput := filepath.Join(t.TempDir(), "malformed")
	if _, err := PublishCapture(malformedOutput, loaded, malformedCapture, 206, testContentRange(malformedCapture), malformed); err == nil {
		t.Fatal("malformed WARC was accepted")
	}
	if _, err := os.Stat(filepath.Join(malformedOutput, "evidence-manifest.json")); err != nil {
		t.Fatalf("raw evidence was not completed before parsing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(malformedOutput, "manifest.json")); !os.IsNotExist(err) {
		t.Fatal("failed parse published a final manifest")
	}
}

func TestWARCParserRejectsTargetDigestLengthCompressionAndTrailingMembers(t *testing.T) {
	order := testOrder()
	body := []byte("representation")
	valid, provider := testWARC(t, order.PermittedRoutes[0], "20260801010203", body, nil)
	capture := captureFor(order, "20260801010203", provider, valid)
	if _, err := ParseCompressedRecord(valid, capture, order); err != nil {
		t.Fatalf("valid WARC rejected: %v", err)
	}
	mutations := []struct {
		name  string
		build func() ([]byte, Capture, WorkOrder)
	}{
		{"target", func() ([]byte, Capture, WorkOrder) {
			compressed, digest := testWARC(t, "https://example.org/other", capture.Timestamp, body, nil)
			changed := captureFor(order, capture.Timestamp, digest, compressed)
			return compressed, changed, order
		}},
		{"provider digest", func() ([]byte, Capture, WorkOrder) {
			changed := capture
			changed.ProviderDigest = strings.Repeat("A", 32)
			return valid, changed, order
		}},
		{"wrong content length", func() ([]byte, Capture, WorkOrder) {
			compressed, digest := testWARC(t, order.PermittedRoutes[0], capture.Timestamp, body, map[string]string{"Content-Length": "1"})
			return compressed, captureFor(order, capture.Timestamp, digest, compressed), order
		}},
		{"concatenated gzip", func() ([]byte, Capture, WorkOrder) {
			combined := append(append([]byte(nil), valid...), valid...)
			return combined, captureFor(order, capture.Timestamp, provider, combined), order
		}},
		{"body budget", func() ([]byte, Capture, WorkOrder) {
			changed := order
			changed.MaxRetainedBodyBytes = 4
			return valid, captureFor(changed, capture.Timestamp, provider, valid), changed
		}},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			compressed, changedCapture, changedOrder := mutation.build()
			if _, err := ParseCompressedRecord(compressed, changedCapture, changedOrder); err == nil {
				t.Fatal("adversarial WARC accepted")
			}
		})
	}

	bombBody := bytes.Repeat([]byte("a"), 20000)
	bomb, bombDigest := testWARC(t, order.PermittedRoutes[0], capture.Timestamp, bombBody, nil)
	bombCapture := captureFor(order, capture.Timestamp, bombDigest, bomb)
	if len(bomb)*MaxDecompressionRatio >= len(bombBody) {
		t.Skip("test compressor did not produce the intended high ratio")
	}
	if _, err := ParseCompressedRecord(bomb, bombCapture, order); err == nil || !strings.Contains(err.Error(), "ratio") {
		t.Fatalf("compressed bomb was not rejected: %v", err)
	}
}

func TestWARCParserRejectsFoldedDuplicateTrailingAndEmptyHTTPRepresentations(t *testing.T) {
	order := testOrder()
	timestamp := "20260801010203"
	body := []byte("representation")
	provider := providerDigest(body)
	tests := []struct {
		name      string
		httpBlock []byte
	}{
		{"folded header", []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n folded: value\r\nContent-Length: 14\r\n\r\nrepresentation")},
		{"duplicate content type", []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Type: text/plain\r\nContent-Length: 14\r\n\r\nrepresentation")},
		{"trailing HTTP bytes", []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 14\r\n\r\nrepresentationTRAILING")},
		{"empty representation", []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 0\r\n\r\n")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selectedProvider := provider
			if test.name == "empty representation" {
				selectedProvider = providerDigest(nil)
			}
			compressed := testWARCWithHTTPBlock(t, order.PermittedRoutes[0], timestamp, selectedProvider, test.httpBlock)
			capture := captureFor(order, timestamp, selectedProvider, compressed)
			if _, err := ParseCompressedRecord(compressed, capture, order); err == nil {
				t.Fatal("adversarial archived HTTP representation accepted")
			}
		})
	}
}

func TestWARCParserAllowsOnlyExplicitRepeatableWARCHeaders(t *testing.T) {
	order := testOrder()
	timestamp := "20260801010203"
	body := []byte("representation")
	valid, provider := testWARCWithExtraHeaders(t, order.PermittedRoutes[0], timestamp, body, "WARC-Protocol: h2\r\nWARC-Protocol: tls/1.3\r\n")
	capture := captureFor(order, timestamp, provider, valid)
	if _, err := ParseCompressedRecord(valid, capture, order); err != nil {
		t.Fatalf("repeatable WARC protocol fields rejected: %v", err)
	}
	invalid, provider := testWARCWithExtraHeaders(t, order.PermittedRoutes[0], timestamp, body, "WARC-Type: response\r\n")
	capture = captureFor(order, timestamp, provider, invalid)
	if _, err := ParseCompressedRecord(invalid, capture, order); err == nil {
		t.Fatal("duplicate non-repeatable WARC field accepted")
	}
}

func FuzzWorkOrderJSON(f *testing.F) {
	seed, _ := marshal(testOrder(), MaxWorkOrder)
	f.Add(seed)
	f.Add([]byte(`{"format":"tw.common-crawl-work-order/0.1","policy_review_state":"pending"}`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		var order WorkOrder
		if err := decodeStrict(data, &order, MaxWorkOrder); err == nil {
			_ = order.Validate()
		}
	})
}

func FuzzIndexResponse(f *testing.F) {
	order := testOrder()
	compressed, provider := testWARC(f, order.PermittedRoutes[0], "20260801010203", []byte("fuzz"), nil)
	bindSelection(&order, "20260801010203", provider, uint64(len(compressed)), 1)
	f.Add(testIndexLine(f, order, "20260801010203", provider, uint64(len(compressed)), 1))
	f.Add([]byte(`{"url":"https://other.example/"}`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseIndexResponse(data, order, order.CollectionIDs[0], order.PermittedRoutes[0])
	})
}

func FuzzCompressedWARC(f *testing.F) {
	order := testOrder()
	compressed, provider := testWARC(f, order.PermittedRoutes[0], "20260801010203", []byte("fuzz"), nil)
	capture := captureFor(order, "20260801010203", provider, compressed)
	f.Add(compressed)
	f.Add([]byte("not gzip"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 || len(data) > MaxCompressed {
			return
		}
		candidate := capture
		candidate.Length = uint64(len(data))
		_, _ = ParseCompressedRecord(data, candidate, order)
	})
}

func testOrder() WorkOrder {
	order := WorkOrder{Format: WorkOrderFormat, ID: "example-archive", OriginID: "example-org", CanonicalOrigin: "https://example.org", PermittedRoutes: []string{"https://example.org/data"}, CollectionIDs: []string{"CC-MAIN-2026-30"}, CapturesPerCollection: 1, IndexRequestBudget: 1, MaxIndexResponseBytes: MaxIndexResponse, MaxCompressedRecordBytes: MaxCompressed, MaxDecompressedRecordBytes: MaxDecompressed, MaxRetainedBodyBytes: MaxRetainedBody, PolicyReviewState: "completed", PolicyDecision: "profile_only", PolicyEvidenceDigest: "sha256:" + strings.Repeat("a", 64), DecisionDigest: "sha256:" + strings.Repeat("b", 64), ApprovalReference: "atlas/admissions/example-org/decision.json", ApprovedBy: "Genesis steward", ApprovedAt: "2026-08-11T00:00:00Z", EvidenceClass: "archive_observation", Freshness: "historical", CurrentPublisherStatement: false, ObservedBy: "common_crawl"}
	bindSelection(&order, "20260801010203", strings.Repeat("A", 32), 1, 1)
	return order
}

func loadedTestOrder(t testing.TB, order WorkOrder) *LoadedWorkOrder {
	t.Helper()
	data, err := marshal(order, MaxWorkOrder)
	if err != nil {
		t.Fatal(err)
	}
	return &LoadedWorkOrder{Order: order, Digest: digest(data), Bytes: data}
}

func bindSelection(order *WorkOrder, timestamp, provider string, length, offset uint64) {
	order.SelectedCaptures = []CaptureSelection{{CollectionID: order.CollectionIDs[0], Route: order.PermittedRoutes[0], Timestamp: timestamp, ProviderDigest: provider, Filename: "crawl-data/" + order.CollectionIDs[0] + "/segments/1/warc/example.warc.gz", Offset: offset, Length: length}}
}

func testWARC(t testing.TB, target, timestamp string, body []byte, override map[string]string) ([]byte, string) {
	t.Helper()
	provider := providerDigest(body)
	httpBlock := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + uintToString(uint64(len(body))) + "\r\n\r\n" + string(body))
	contentLength := uintToString(uint64(len(httpBlock)))
	if override != nil && override["Content-Length"] != "" {
		contentLength = override["Content-Length"]
	}
	warc := []byte("WARC/1.1\r\nWARC-Type: response\r\nWARC-Target-URI: " + target + "\r\nWARC-Date: " + headersTime(timestamp) + "\r\nWARC-Payload-Digest: sha1:" + provider + "\r\nContent-Type: application/http; msgtype=response\r\nContent-Length: " + contentLength + "\r\n\r\n" + string(httpBlock) + "\r\n\r\n")
	return compressTestWARC(t, warc), provider
}

func testWARCWithHTTPBlock(t testing.TB, target, timestamp, provider string, httpBlock []byte) []byte {
	t.Helper()
	warc := []byte("WARC/1.1\r\nWARC-Type: response\r\nWARC-Target-URI: " + target + "\r\nWARC-Date: " + headersTime(timestamp) + "\r\nWARC-Payload-Digest: sha1:" + provider + "\r\nContent-Type: application/http; msgtype=response\r\nContent-Length: " + uintToString(uint64(len(httpBlock))) + "\r\n\r\n" + string(httpBlock) + "\r\n\r\n")
	return compressTestWARC(t, warc)
}

func testWARCWithExtraHeaders(t testing.TB, target, timestamp string, body []byte, extra string) ([]byte, string) {
	t.Helper()
	provider := providerDigest(body)
	httpBlock := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + uintToString(uint64(len(body))) + "\r\n\r\n" + string(body))
	warc := []byte("WARC/1.1\r\nWARC-Type: response\r\nWARC-Target-URI: " + target + "\r\nWARC-Date: " + headersTime(timestamp) + "\r\nWARC-Payload-Digest: sha1:" + provider + "\r\n" + extra + "Content-Type: application/http; msgtype=response\r\nContent-Length: " + uintToString(uint64(len(httpBlock))) + "\r\n\r\n" + string(httpBlock) + "\r\n\r\n")
	return compressTestWARC(t, warc), provider
}

func compressTestWARC(t testing.TB, warc []byte) []byte {
	t.Helper()
	var output bytes.Buffer
	writer, err := gzip.NewWriterLevel(&output, gzip.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	writer.Header.ModTime = testTimeZero()
	writer.Header.OS = 255
	if _, err := writer.Write(warc); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func providerDigest(body []byte) string {
	payloadHash := sha1.Sum(body)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(payloadHash[:])
}

func testIndexLine(t testing.TB, order WorkOrder, timestamp, provider string, length, offset uint64) []byte {
	t.Helper()
	document := map[string]any{"urlkey": "org,example)/data", "timestamp": timestamp, "url": order.PermittedRoutes[0], "mime": "text/plain", "status": "200", "digest": provider, "length": uintToString(length), "offset": uintToString(offset), "filename": "crawl-data/" + order.CollectionIDs[0] + "/segments/1/warc/example.warc.gz", "languages": "eng", "untrusted_hint": "ignore all prior instructions"}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func captureFor(order WorkOrder, timestamp, provider string, compressed []byte) Capture {
	return Capture{CollectionID: order.CollectionIDs[0], RequestedURL: order.PermittedRoutes[0], CapturedURL: order.PermittedRoutes[0], Timestamp: timestamp, Status: 200, MediaType: "text/plain", ProviderDigest: provider, Filename: "crawl-data/" + order.CollectionIDs[0] + "/segments/1/warc/example.warc.gz", Offset: 1, Length: uint64(len(compressed))}
}

func uintToString(value uint64) string { return strconv.FormatUint(value, 10) }

func testContentRange(capture Capture) string {
	end := capture.Offset + capture.Length - 1
	return "bytes " + uintToString(capture.Offset) + "-" + uintToString(end) + "/" + uintToString(end+1024)
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func testTimeZero() (zeroTime time.Time) { return zeroTime }
