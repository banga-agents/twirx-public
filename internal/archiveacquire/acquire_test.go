package archiveacquire

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/base32"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/archiveimport"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

type fakeRetriever struct {
	fetchURL  string
	rangeURL  string
	rangeHead string
	fetch     *safefetch.Result
	rangeGet  *safefetch.Result
}

func (f *fakeRetriever) Fetch(_ context.Context, rawURL string) (*safefetch.Result, error) {
	f.fetchURL = rawURL
	return f.fetch, nil
}

func (f *fakeRetriever) FetchRange(_ context.Context, rawURL, requestedRange string) (*safefetch.Result, error) {
	f.rangeURL = rawURL
	f.rangeHead = requestedRange
	return f.rangeGet, nil
}

func TestAcquisitionStoresRawEvidenceBeforeParsingAndVerifies(t *testing.T) {
	order, loaded := testLoadedOrder(t)
	body := []byte("<html><title>RFC Editor</title><p>archive statement A</p></html>")
	compressed, provider := testWARC(t, order.PermittedRoutes[0], "20250708102138", body)
	indexURL, err := archiveimport.BuildIndexURL(order, order.CollectionIDs[0], order.PermittedRoutes[0])
	if err != nil {
		t.Fatal(err)
	}
	indexBody := testIndexLine(t, order, "20250708102138", provider, uint64(len(compressed)), 1024)
	dataURL := "https://" + archiveimport.OfficialDataHost + "/crawl-data/" + order.CollectionIDs[0] + "/segments/1/warc/example.warc.gz"
	end := 1024 + uint64(len(compressed)) - 1
	rangeHeader := "bytes=1024-" + strconv.FormatUint(end, 10)
	contentRange := "bytes 1024-" + strconv.FormatUint(end, 10) + "/999999"
	indexRetriever := &fakeRetriever{fetch: &safefetch.Result{RequestURL: indexURL, FinalURL: indexURL, Method: http.MethodGet, Status: http.StatusOK, MediaType: "application/json", Body: indexBody}}
	dataRetriever := &fakeRetriever{rangeGet: &safefetch.Result{RequestURL: dataURL, FinalURL: dataURL, Method: http.MethodGet, Status: http.StatusPartialContent, MediaType: "application/octet-stream", Body: compressed, RequestedRange: rangeHeader, ContentRange: contentRange}}
	runner, err := newRunner(indexRetriever, dataRetriever)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "acquisition")
	manifest, err := runner.Acquire(context.Background(), loaded, root)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.IndexRequests != 1 || manifest.RangeRequests != 1 || manifest.NetworkRequestsMade != 2 || len(manifest.Captures) != 1 || manifest.Captures[0].RepresentationDigest != digest(body) {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	if !strings.HasPrefix(indexRetriever.fetchURL, "https://"+archiveimport.OfficialIndexHost+"/") || dataRetriever.rangeURL != dataURL || dataRetriever.rangeHead != rangeHeader {
		t.Fatalf("network authority mismatch: index=%q data=%q range=%q", indexRetriever.fetchURL, dataRetriever.rangeURL, dataRetriever.rangeHead)
	}
	verified, err := Verify(root)
	if err != nil || verified.WorkOrderDigest != manifest.WorkOrderDigest || verified.Captures[0].RepresentationDigest != digest(body) {
		t.Fatalf("verification failed: %+v %v", verified, err)
	}
	if _, err := os.Stat(filepath.Join(root, "raw", "index-000.jsonl")); err != nil {
		t.Fatal("raw index evidence was not retained")
	}
	if _, err := os.Stat(filepath.Join(root, "raw", "range-000.warc.gz")); err != nil {
		t.Fatal("raw range evidence was not retained")
	}
	if err := os.Chmod(filepath.Join(root, "raw", "index-000.jsonl"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "raw", "index-000.jsonl"), []byte("tampered\n"), 0o440); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(root); err == nil {
		t.Fatal("tampered raw acquisition evidence was accepted")
	}
}

func TestMalformedIndexRemainsRawButCannotPublish(t *testing.T) {
	order, loaded := testLoadedOrder(t)
	indexURL, err := archiveimport.BuildIndexURL(order, order.CollectionIDs[0], order.PermittedRoutes[0])
	if err != nil {
		t.Fatal(err)
	}
	indexRetriever := &fakeRetriever{fetch: &safefetch.Result{RequestURL: indexURL, FinalURL: indexURL, Method: http.MethodGet, Status: http.StatusOK, MediaType: "application/json", Body: []byte("not-json\n")}}
	dataRetriever := &fakeRetriever{}
	runner, err := newRunner(indexRetriever, dataRetriever)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "malformed")
	if _, err := runner.Acquire(context.Background(), loaded, root); err == nil {
		t.Fatal("malformed index response was accepted")
	}
	if dataRetriever.rangeURL != "" {
		t.Fatal("range request occurred after index rejection")
	}
	if _, err := os.Stat(filepath.Join(root, "raw", "index-000.jsonl")); err != nil {
		t.Fatal("raw malformed index was not retained before parsing")
	}
	if _, err := os.Stat(filepath.Join(root, "acquisition-manifest.json")); !os.IsNotExist(err) {
		t.Fatal("failed acquisition published a final manifest")
	}
}

func TestAcquisitionRejectsRedirectStatusRangeAndHostSubstitution(t *testing.T) {
	order, loaded := testLoadedOrder(t)
	indexURL, _ := archiveimport.BuildIndexURL(order, order.CollectionIDs[0], order.PermittedRoutes[0])
	tests := []safefetch.Result{
		{RequestURL: indexURL, FinalURL: "https://other.example/index", Method: http.MethodGet, Status: http.StatusOK, Body: []byte("{}\n")},
		{RequestURL: indexURL, FinalURL: indexURL, Method: http.MethodPost, Status: http.StatusOK, Body: []byte("{}\n")},
		{RequestURL: indexURL, FinalURL: indexURL, Method: http.MethodGet, Status: http.StatusFound, Body: []byte("{}\n")},
		{RequestURL: indexURL, FinalURL: indexURL, Method: http.MethodGet, Status: http.StatusOK, Body: []byte("{}\n"), Redirects: []safefetch.Redirect{{FromURL: indexURL, Status: 302, ToURL: indexURL}}},
	}
	for index := range tests {
		retriever := &fakeRetriever{fetch: &tests[index]}
		runner, err := newRunner(retriever, &fakeRetriever{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Acquire(context.Background(), loaded, filepath.Join(t.TempDir(), "rejected")); err == nil {
			t.Fatalf("substituted response %d was accepted", index)
		}
	}
}

func TestAcquisitionManifestConformance(t *testing.T) {
	order, _ := testLoadedOrder(t)
	root := filepath.Join("..", "..", "conformance", "archive-acquisition")
	for _, test := range []struct {
		name  string
		valid bool
	}{
		{name: "valid-manifest.json", valid: true},
		{name: "invalid-authority.json", valid: false},
	} {
		data, err := os.ReadFile(filepath.Join(root, test.name))
		if err != nil {
			t.Fatal(err)
		}
		var manifest Manifest
		err = decodeManifest(data, &manifest)
		if err == nil {
			err = manifest.Validate(order)
		}
		if (err == nil) != test.valid {
			t.Fatalf("conformance vector %s validity=%t, error=%v", test.name, test.valid, err)
		}
	}
}

func FuzzManifestJSON(f *testing.F) {
	order, _ := testLoadedOrder(f)
	seed := Manifest{Format: ManifestFormat, WorkOrderID: order.ID, WorkOrderDigest: "sha256:" + strings.Repeat("a", 64), IndexHost: archiveimport.OfficialIndexHost, DataHost: archiveimport.OfficialDataHost, IndexRequests: 1, RangeRequests: 1, NetworkRequestsMade: 2, Artifacts: []Artifact{
		{Path: "captures/capture-000/manifest.json", Digest: "sha256:" + strings.Repeat("b", 64), Size: 1},
		{Path: "raw/index-000.jsonl", Digest: "sha256:" + strings.Repeat("c", 64), Size: 1},
		{Path: "raw/range-000.warc.gz", Digest: "sha256:" + strings.Repeat("d", 64), Size: 1},
		{Path: "work-order.json", Digest: "sha256:" + strings.Repeat("e", 64), Size: 1},
	}, Captures: []Capture{{CollectionID: order.CollectionIDs[0], Route: order.PermittedRoutes[0], CaptureTimestamp: "20250708102138", SpoolPath: "captures/capture-000", SpoolManifestDigest: "sha256:" + strings.Repeat("b", 64), RepresentationDigest: "sha256:" + strings.Repeat("f", 64), RepresentationSize: 1}}}
	encoded, _ := marshal(seed)
	f.Add(encoded)
	f.Add([]byte(`{"format":"tw.archive-acquisition-manifest/0.1"}`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		var manifest Manifest
		if err := decodeManifest(data, &manifest); err == nil {
			_ = manifest.Validate(order)
		}
	})
}

func testLoadedOrder(t testing.TB) (archiveimport.WorkOrder, *archiveimport.LoadedWorkOrder) {
	t.Helper()
	order := archiveimport.WorkOrder{Format: archiveimport.WorkOrderFormat, ID: "rfc-editor-archive", OriginID: "rfc-editor-org", CanonicalOrigin: "https://www.rfc-editor.org", PermittedRoutes: []string{"https://www.rfc-editor.org/"}, CollectionIDs: []string{"CC-MAIN-2025-30"}, CapturesPerCollection: 1, IndexRequestBudget: 1, MaxIndexResponseBytes: archiveimport.MaxIndexResponse, MaxCompressedRecordBytes: archiveimport.MaxCompressed, MaxDecompressedRecordBytes: archiveimport.MaxDecompressed, MaxRetainedBodyBytes: archiveimport.MaxRetainedBody, PolicyReviewState: "completed", PolicyDecision: "profile_only", PolicyEvidenceDigest: "sha256:" + strings.Repeat("a", 64), DecisionDigest: "sha256:" + strings.Repeat("b", 64), ApprovalReference: "atlas/admissions/rfc-editor-org/decision.json", ApprovedBy: "Genesis steward", ApprovedAt: "2026-08-11T00:00:00Z", EvidenceClass: "archive_observation", Freshness: "historical", CurrentPublisherStatement: false, ObservedBy: "common_crawl"}
	body := []byte("<html><title>RFC Editor</title><p>archive statement A</p></html>")
	compressed, provider := testWARC(t, order.PermittedRoutes[0], "20250708102138", body)
	order.SelectedCaptures = []archiveimport.CaptureSelection{{CollectionID: order.CollectionIDs[0], Route: order.PermittedRoutes[0], Timestamp: "20250708102138", ProviderDigest: provider, Filename: "crawl-data/" + order.CollectionIDs[0] + "/segments/1/warc/example.warc.gz", Offset: 1024, Length: uint64(len(compressed))}}
	data, err := json.MarshalIndent(order, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := order.Validate(); err != nil {
		t.Fatal(err)
	}
	return order, &archiveimport.LoadedWorkOrder{Order: order, Digest: digest(data), Bytes: data}
}

func testWARC(t testing.TB, target, timestamp string, body []byte) ([]byte, string) {
	t.Helper()
	hash := sha1.Sum(body)
	provider := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash[:])
	httpBlock := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + string(body))
	parsed, _ := time.Parse("20060102150405", timestamp)
	warc := []byte("WARC/1.1\r\nWARC-Type: response\r\nWARC-Target-URI: " + target + "\r\nWARC-Date: " + parsed.UTC().Format(time.RFC3339) + "\r\nWARC-Payload-Digest: sha1:" + provider + "\r\nContent-Type: application/http; msgtype=response\r\nContent-Length: " + strconv.Itoa(len(httpBlock)) + "\r\n\r\n" + string(httpBlock) + "\r\n\r\n")
	var output bytes.Buffer
	writer := gzip.NewWriter(&output)
	writer.Header.ModTime = time.Time{}
	writer.Header.OS = 255
	if _, err := writer.Write(warc); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes(), provider
}

func testIndexLine(t testing.TB, order archiveimport.WorkOrder, timestamp, provider string, length, offset uint64) []byte {
	t.Helper()
	document := map[string]any{"urlkey": "org,rfc-editor)/", "timestamp": timestamp, "url": order.PermittedRoutes[0], "mime": "text/html", "status": "200", "digest": provider, "length": strconv.FormatUint(length, 10), "offset": strconv.FormatUint(offset, 10), "filename": "crawl-data/" + order.CollectionIDs[0] + "/segments/1/warc/example.warc.gz"}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}
