package observatoryworker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteAndVerifyOffline(t *testing.T) {
	var userAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		userAgent = request.Header.Get("User-Agent")
		if request.Method != http.MethodGet || request.URL.Path != "/robots.txt" {
			t.Errorf("unexpected request %s %s", request.Method, request.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("User-agent: TWIRXBot\nDisallow: /private\nAllow: /public\n"))
	}))
	job := writeJob(t, strings.Replace(server.URL, "localhost", "127.0.0.1", 1)+"/robots.txt", "/private/report")
	out := filepath.Join(t.TempDir(), "proof")
	result, err := Execute(context.Background(), job, out)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	if result.Allowed || !result.Matched || result.MatchedPattern != "/private" || userAgent != workerUserAgent {
		server.Close()
		t.Fatalf("unexpected result=%#v user-agent=%q", result, userAgent)
	}
	server.Close()

	verified, err := Verify(out)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Status != "verified" || verified.NetworkAccess != "disabled" || verified.Allowed {
		t.Fatalf("unexpected verification: %#v", verified)
	}
}

func TestEvidenceExistsBeforeMalformedRobotsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte{0xff, 0xfe})
	}))
	defer server.Close()
	job := writeJob(t, strings.Replace(server.URL, "localhost", "127.0.0.1", 1)+"/robots.txt", "/private")
	out := filepath.Join(t.TempDir(), "proof")
	if _, err := Execute(context.Background(), job, out); err == nil || !strings.Contains(err.Error(), "after evidence publication") {
		t.Fatalf("expected post-publication parse failure, got %v", err)
	}
	for _, name := range []string{"job.json", filepath.Join("evidence", "observation.cbor"), filepath.Join("evidence", "observation.json"), filepath.Join("evidence", "body.ref")} {
		if info, err := os.Lstat(filepath.Join(out, name)); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("evidence %s missing or not regular: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "result.json")); !os.IsNotExist(err) {
		t.Fatalf("result was published after parse failure: %v", err)
	}
}

func TestRedirectAndNonLoopbackJobsFailClosed(t *testing.T) {
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, "/robots.txt", http.StatusFound)
	}))
	defer redirect.Close()
	job := writeJob(t, strings.Replace(redirect.URL, "localhost", "127.0.0.1", 1)+"/robots.txt", "/")
	if _, err := Execute(context.Background(), job, filepath.Join(t.TempDir(), "redirect")); err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("redirect did not fail closed: %v", err)
	}

	for name, rawURL := range map[string]string{
		"public":         "https://example.org/robots.txt",
		"hostname":       "http://localhost:18081/robots.txt",
		"other-loopback": "http://127.0.0.2:18081/robots.txt",
		"private":        "http://10.0.0.1:18081/robots.txt",
		"credentials":    "http://user:pass@127.0.0.1:18081/robots.txt",
		"query":          "http://127.0.0.1:18081/robots.txt?x=1",
	} {
		t.Run(name, func(t *testing.T) {
			candidate := validJob(rawURL, "/")
			if err := candidate.Validate(); err == nil {
				t.Fatal("unsafe destination accepted")
			}
		})
	}
}

func TestJobDecoderRejectsUnknownDuplicateTrailingAndSymlink(t *testing.T) {
	data := mustJobJSON(t, validJob("http://127.0.0.1:18081/robots.txt", "/"))
	unknown := strings.Replace(string(data), `"format":`, `"unknown":true,"format":`, 1)
	duplicate := strings.Replace(string(data), `"format":`, `"format":"duplicate","format":`, 1)
	for name, malformed := range map[string][]byte{
		"unknown":   []byte(unknown),
		"duplicate": []byte(duplicate),
		"trailing":  append(append([]byte(nil), data...), []byte("{}")...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeJob(malformed); err == nil {
				t.Fatal("malformed job accepted")
			}
		})
	}
	target := filepath.Join(t.TempDir(), "target.json")
	if err := os.WriteFile(target, data, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "job.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadJob(link); err == nil {
		t.Fatal("symlink job accepted")
	}
}

func TestVerificationRejectsMutatedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
	}))
	job := writeJob(t, strings.Replace(server.URL, "localhost", "127.0.0.1", 1)+"/robots.txt", "/")
	out := filepath.Join(t.TempDir(), "proof")
	if _, err := Execute(context.Background(), job, out); err != nil {
		server.Close()
		t.Fatal(err)
	}
	server.Close()
	path := filepath.Join(out, "result.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"allowed": true`, `"allowed": false`, 1))
	if err := os.WriteFile(path, data, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(out); err == nil || !strings.Contains(err.Error(), "disagrees") {
		t.Fatalf("mutated result accepted: %v", err)
	}
}

func FuzzJobJSON(f *testing.F) {
	seed, _ := json.Marshal(validJob("http://127.0.0.1:18081/robots.txt", "/private"))
	f.Add(seed)
	f.Add([]byte(`{"format":"tw.observatory-job/0.1","format":"duplicate"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 || len(data) > maxJobBytes {
			return
		}
		_, _ = decodeJob(data)
	})
}

func writeJob(t *testing.T, rawURL, target string) *LoadedJob {
	t.Helper()
	data := mustJobJSON(t, validJob(rawURL, target))
	path := filepath.Join(t.TempDir(), "job.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadJob(path)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func validJob(rawURL, target string) Job {
	return Job{Format: JobFormat, Mode: LocalMode, OriginID: "controlled-origin-fixture", ArtifactKind: ArtifactRobots, URL: rawURL, ProductToken: ProductToken, TargetPath: target, MaxBodyBytes: robotstxtMaxForTest()}
}

func robotstxtMaxForTest() int64 { return 500 * 1024 }

func mustJobJSON(t *testing.T, job Job) []byte {
	t.Helper()
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}
