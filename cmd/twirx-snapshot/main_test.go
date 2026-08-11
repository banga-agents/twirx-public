package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/snapshotartifact"
	"github.com/typed-web-commons/typed-web/internal/snapshotbuild"
	"github.com/typed-web-commons/typed-web/internal/snapshotruntime"
)

func TestValidateLoopback(t *testing.T) {
	for _, address := range []string{"127.0.0.1:8090", "[::1]:8090"} {
		if err := validateLoopback(address); err != nil {
			t.Fatalf("rejected %q: %v", address, err)
		}
	}
	for _, address := range []string{"0.0.0.0:8090", "[::]:8090", "localhost:8090", "127.0.0.1"} {
		if err := validateLoopback(address); err == nil {
			t.Fatalf("accepted non-literal-loopback address %q", address)
		}
	}
}

func TestQueryRequestRejectsUnknownField(t *testing.T) {
	if _, err := decodeQueryRequest([]byte(`{"select":["project:Status.phase"],"unknown":true}`)); err == nil {
		t.Fatal("expected unknown-field rejection")
	}
}

func TestQueryRequestDefaultsRemainReadOnly(t *testing.T) {
	request, err := decodeQueryRequest([]byte(`{"select":["project:Status.phase"],"subject_concept":"project:Status"}`))
	if err != nil {
		t.Fatal(err)
	}
	query, err := request.canonical()
	if err != nil {
		t.Fatal(err)
	}
	if !query.Execution.AllowMaterializedState || query.Execution.AllowLiveRefresh || query.Execution.MaximumLiveOrigins != 0 || !query.Proof.IncludeNative {
		t.Fatalf("unsafe query defaults: %+v", query)
	}
}

func TestHTTPRuntimeServesBoundedPublicQuery(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "snapshot")
	build, err := snapshotbuild.Build(context.Background(), snapshotbuild.Options{Root: filepath.Clean(filepath.Join("..", "..")), Output: directory, SourceRevision: "test-revision", CreatedAt: "2026-08-11T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := snapshotruntime.Open(directory, snapshotruntime.Options{ExpectedID: build.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(newHandler(runtime))
	defer server.Close()
	client := server.Client()

	response, err := client.Get(server.URL + "/api/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unexpected status response: %s", response.Status)
	}
	_ = response.Body.Close()

	response, err = client.Get(server.URL + "/api/v1/origins?limit=500")
	if err != nil {
		t.Fatal(err)
	}
	var origins originPage
	if err := json.NewDecoder(response.Body).Decode(&origins); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || origins.Total != 500 || len(origins.Items) != 500 {
		t.Fatalf("unexpected Atlas response: status=%s page=%+v", response.Status, origins)
	}

	response, err = client.Get(server.URL + "/api/v1/origins/api-worldbank-org")
	if err != nil {
		t.Fatal(err)
	}
	var worldBank snapshotruntime.OriginDescription
	if err := json.NewDecoder(response.Body).Decode(&worldBank); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || worldBank.PacketCount == 0 || worldBank.PacketState != "public_packets_available" {
		t.Fatalf("unexpected origin response: status=%s origin=%+v", response.Status, worldBank)
	}

	response, err = client.Get(server.URL + "/api/v1/origins/not-admitted")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected unknown-origin status: %s", response.Status)
	}
	_ = response.Body.Close()

	response, err = client.Get(server.URL + "/api/v1/origins?unexpected=1")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected origin pagination status: %s", response.Status)
	}
	_ = response.Body.Close()

	requestBody := []byte(`{"select":["development:IndicatorObservation.value"],"subject_concept":"development:IndicatorObservation","origin_ids":["api-worldbank-org"]}`)
	response, err = client.Post(server.URL+"/api/v1/query", "application/json", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result queryResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || result.Status != "resolved" || len(result.Rows) != 1 || result.Plan.NetworkRequests != 0 {
		t.Fatalf("unexpected query response: status=%s result=%+v", response.Status, result)
	}
	if result.QueryDigest != snapshotartifact.DigestReference(dataplane.DigestBytes(result.CanonicalQueryCBOR)) || result.ResultDigest != snapshotartifact.DigestReference(dataplane.DigestBytes(result.CanonicalResultCBOR)) {
		t.Fatal("display digests do not identify returned canonical query/result bytes")
	}
	_ = response.Body.Close()

	digest := strings.TrimPrefix(result.Rows[0].PacketDigest, "sha256:")
	response, err = client.Get(server.URL + "/api/v1/packets/" + digest)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/cbor" || response.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("unexpected packet download: %s", response.Status)
	}
	_ = response.Body.Close()

	response, err = client.Get(server.URL + "/api/v1/trace/" + digest)
	if err != nil {
		t.Fatal(err)
	}
	var trace snapshotruntime.Trace
	if err := json.NewDecoder(response.Body).Decode(&trace); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if len(trace.Proof.Artifacts) != 10 {
		t.Fatalf("unexpected proof index: %+v", trace.Proof)
	}
	response, err = client.Get(server.URL + "/api/v1/proof/" + digest + "/" + trace.Proof.Artifacts[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected proof download: %s", response.Status)
	}
	_ = response.Body.Close()

	response, err = client.Get(server.URL + "/api/v1/proof/" + digest + "/not-admitted")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected unknown-proof status: %s", response.Status)
	}
	_ = response.Body.Close()

	response, err = client.Get(server.URL + "/api/v1/concepts?limit=10&unexpected=1")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected pagination status: %s", response.Status)
	}
	_ = response.Body.Close()

	response, err = client.Post(server.URL+"/api/v1/query", "text/plain", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("unexpected content-type status: %s", response.Status)
	}
	_ = response.Body.Close()
}

func TestHTTPRuntimeServesImmutableArchiveDelta(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	requirePrivateArchiveEvidence(t, repositoryRoot)
	directory := filepath.Join(t.TempDir(), "snapshot")
	build, err := snapshotbuild.Build(context.Background(), snapshotbuild.Options{Root: repositoryRoot, Output: directory, SourceRevision: "test-revision", CreatedAt: "2026-08-11T12:15:50Z", ArchiveAcquisitionIDs: []string{"rfc-editor-futo-history"}})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := snapshotruntime.Open(directory, snapshotruntime.Options{ExpectedID: build.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(newHandler(runtime))
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/api/v1/deltas?limit=10")
	if err != nil {
		t.Fatal(err)
	}
	var page deltaPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("unexpected delta page: status=%s page=%+v", response.Status, page)
	}
	digest := strings.TrimPrefix(page.Items[0].Digest, "sha256:")
	response, err = server.Client().Get(server.URL + "/api/v1/deltas/" + digest)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/cbor" || response.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("unexpected delta download: %s", response.Status)
	}
}

func requirePrivateArchiveEvidence(t *testing.T, repositoryRoot string) {
	t.Helper()
	path := filepath.Join(repositoryRoot, "atlas", "archive-acquisitions", "rfc-editor-futo-history", "captures", "capture-000", "representation.body")
	if _, err := os.Stat(path); err == nil {
		return
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect private archive evidence: %v", err)
	}
	if os.Getenv("TWIRX_REQUIRE_PRIVATE_ARCHIVE_EVIDENCE") == "1" {
		t.Fatalf("required private archive evidence is absent: %s", path)
	}
	t.Skip("private third-party archive bytes are excluded from the public source profile")
}

func FuzzQueryRequestJSON(f *testing.F) {
	f.Add([]byte(`{"select":["project:Status.phase"],"subject_concept":"project:Status"}`))
	f.Add([]byte{0xff, 0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		request, err := decodeQueryRequest(data)
		if err == nil {
			_, _ = request.canonical()
		}
	})
}
