package labapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/labengine"
)

func apiRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
func testServer(t *testing.T, mutate func(*Config)) (*Server, *labengine.Engine) {
	t.Helper()
	root := apiRoot(t)
	engine, err := labengine.New(root, filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Engine: engine, StaticDir: filepath.Join(root, "lab", "static"), PerIPPerMinute: 600, PerIPBurst: 100, AuditWriter: io.Discard}
	if mutate != nil {
		mutate(&config)
	}
	server, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	return server, engine
}
func request(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "198.51.100.10:1234"
	return req
}

func TestStatusCatalogAndSecurityHeaders(t *testing.T) {
	server, _ := testServer(t, nil)
	for _, path := range []string{"/api/v1/status", "/api/v1/origins", "/.well-known/twirx"} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request(http.MethodGet, path, ""))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, recorder.Code, recorder.Body.String())
		}
		if recorder.Header().Get("Content-Security-Policy") == "" || recorder.Header().Get("Access-Control-Allow-Origin") != "" || recorder.Header().Get("Set-Cookie") != "" {
			t.Fatalf("unsafe headers: %#v", recorder.Header())
		}
		if path == "/api/v1/origins" && (!strings.Contains(recorder.Body.String(), `"admission_status":"reviewed_e2"`) || !strings.Contains(recorder.Body.String(), `"health_state":"not_probed"`)) {
			t.Fatalf("origin assurance metadata missing: %s", recorder.Body.String())
		}
		if path == "/api/v1/status" && (!strings.Contains(recorder.Body.String(), `"execution_mode":"replay_only"`) || !strings.Contains(recorder.Body.String(), `"fresh_origin_access":false`)) {
			t.Fatalf("public execution mode missing: %s", recorder.Body.String())
		}
	}
}

func TestTypedInvokeResultProvenanceAndBundle(t *testing.T) {
	var audit bytes.Buffer
	server, engine := testServer(t, func(config *Config) { config.AuditWriter = &audit })
	payload := `{"origin_id":"world-bank-indicators","operation_id":"development.getIndicator","mode":"replay","input":{"country":"CHL","indicator":"SP.POP.TOTL","year":2024}}`
	req := request(http.MethodPost, "/api/v1/invoke", payload)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("invoke: %d %s", recorder.Code, recorder.Body.String())
	}
	var view labengine.ResultView
	if err := json.Unmarshal(recorder.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.OperationID != "development.getIndicator" || len(view.Fields) != 4 || view.Fields[3].Native.Lexical == nil || *view.Fields[3].Native.Lexical != "19764771" {
		t.Fatalf("unexpected view %#v", view)
	}
	if strings.Contains(audit.String(), "198.51.100.10") {
		t.Fatalf("raw client IP retained: %s", audit.String())
	}
	base := "/api/v1/results/" + view.ResultID
	for _, suffix := range []string{"", "/provenance"} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request(http.MethodGet, base+suffix, ""))
		if response.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", suffix, response.Code, response.Body.String())
		}
	}
	archiveResponse := httptest.NewRecorder()
	server.ServeHTTP(archiveResponse, request(http.MethodGet, base+"/bundle", ""))
	if archiveResponse.Code != http.StatusOK || archiveResponse.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("bundle: %d %s", archiveResponse.Code, archiveResponse.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(archiveResponse.Body.Bytes()), int64(archiveResponse.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool)
	for _, file := range archive.File {
		names[file.Name] = true
	}
	for _, name := range []string{"manifest.cbor", "result.cbor", "representation.body", "observation.cbor"} {
		if !names[name] {
			t.Fatalf("bundle missing %s", name)
		}
	}
	bundleDir := filepath.Join(engine.ResultsDir, strings.TrimPrefix(view.ResultID, "sha256:"))
	if err := os.WriteFile(filepath.Join(bundleDir, "transcript.json"), []byte("{}"), 0o640); err != nil {
		t.Fatal(err)
	}
	tampered := httptest.NewRecorder()
	server.writeBundle(tampered, bundleDir, view.ResultID)
	if tampered.Code != http.StatusNotFound || !strings.Contains(tampered.Body.String(), "failed verification") {
		t.Fatalf("tampered bundle emitted: %d %s", tampered.Code, tampered.Body.String())
	}
}

func TestArbitraryURLsAndMalformedRequestsFailClosed(t *testing.T) {
	server, _ := testServer(t, nil)
	tests := []struct {
		contentType, body string
		status            int
	}{
		{"application/json", `{"origin_id":"controlled-origin-lab","operation_id":"fixture.getOffer","mode":"replay","input":{"product_id":"demo-1","url":"http://127.0.0.1/private"}}`, http.StatusBadRequest},
		{"application/json", `{"origin_id":"controlled-origin-lab","operation_id":"fixture.getOffer","input":{"product_id":"demo-1"},"url":"https://example.com"}`, http.StatusBadRequest},
		{"text/plain", `{}`, http.StatusUnsupportedMediaType},
		{"application/json", `{"origin_id":"x","origin_id":"y"}`, http.StatusBadRequest},
		{"application/json", `{"origin_id":"world-bank-indicators","operation_id":"development.getIndicator","mode":"fresh","input":{"country":"CHL","indicator":"SP.POP.TOTL","year":2024}}`, http.StatusBadRequest},
	}
	for _, test := range tests {
		req := request(http.MethodPost, "/api/v1/invoke", test.body)
		req.Header.Set("Content-Type", test.contentType)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, req)
		if response.Code != test.status {
			t.Fatalf("got %d want %d: %s", response.Code, test.status, response.Body.String())
		}
	}
}

func TestRateConcurrencyAndOriginQuotas(t *testing.T) {
	fixed := time.Unix(1000, 0)
	server, _ := testServer(t, func(config *Config) { config.PerIPPerMinute = 1; config.PerIPBurst = 1; config.GlobalConcurrency = 1 })
	server.now = func() time.Time { return fixed }
	first := httptest.NewRecorder()
	server.ServeHTTP(first, request(http.MethodGet, "/api/v1/status", ""))
	second := httptest.NewRecorder()
	server.ServeHTTP(second, request(http.MethodGet, "/api/v1/status", ""))
	if first.Code != 200 || second.Code != 429 {
		t.Fatalf("rate codes %d %d", first.Code, second.Code)
	}
	concurrent, _ := testServer(t, func(config *Config) { config.GlobalConcurrency = 1 })
	concurrent.now = func() time.Time { return fixed }
	concurrent.global <- struct{}{}
	invokeBody := `{"origin_id":"controlled-origin-lab","operation_id":"fixture.getOffer","mode":"replay","input":{"product_id":"demo-1"}}`
	req := request(http.MethodPost, "/api/v1/invoke", invokeBody)
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	concurrent.ServeHTTP(response, req)
	<-concurrent.global
	if response.Code != 429 || !strings.Contains(response.Body.String(), "concurrency_limited") {
		t.Fatalf("concurrency: %d %s", response.Code, response.Body.String())
	}
	originLimited, _ := testServer(t, nil)
	originLimited.now = func() time.Time { return fixed }
	originLimited.perOrigin.buckets["controlled-origin-lab"] = bucket{tokens: 0, last: fixed}
	req = request(http.MethodPost, "/api/v1/invoke", invokeBody)
	req.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	originLimited.ServeHTTP(response, req)
	if response.Code != 429 || !strings.Contains(response.Body.String(), "origin_rate_limited") {
		t.Fatalf("origin quota: %d %s", response.Code, response.Body.String())
	}
}

func TestStaticSurfaceCannotExposeRepository(t *testing.T) {
	server, _ := testServer(t, nil)
	for _, path := range []string{"/.git/config", "/reports/public-readiness.md", "/../AGENTS.md", "/var/e2/results"} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request(http.MethodGet, path, ""))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s returned %d", path, response.Code)
		}
	}
	nonProxy := request(http.MethodGet, "/api/v1/status", "")
	nonProxy.Header.Set("X-Forwarded-For", "203.0.113.8")
	if got := server.clientKey(nonProxy); got != "198.51.100.10" {
		t.Fatalf("trusted spoofed forwarding header: %s", got)
	}
	proxy := request(http.MethodGet, "/api/v1/status", "")
	proxy.RemoteAddr = "127.0.0.1:9000"
	proxy.Header.Set("X-Forwarded-For", "203.0.113.8")
	if got := server.clientKey(proxy); got != "203.0.113.8" {
		t.Fatalf("proxy client: %s", got)
	}
	asset := httptest.NewRecorder()
	server.ServeHTTP(asset, request(http.MethodGet, "/assets/app.js", ""))
	if asset.Code != http.StatusOK || !strings.Contains(asset.Body.String(), `credentials: "omit"`) || strings.Contains(asset.Body.String(), `credentials: "same-origin"`) {
		t.Fatalf("Lab client did not omit credentials")
	}
}
