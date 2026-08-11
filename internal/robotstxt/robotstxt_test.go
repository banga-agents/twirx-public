package robotstxt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type conformanceCorpus struct {
	Format       string `json:"format"`
	ProductToken string `json:"product_token"`
	Cases        []struct {
		Name           string `json:"name"`
		Robots         string `json:"robots"`
		PathQuery      string `json:"path_query"`
		Allowed        bool   `json:"allowed"`
		MatchedPattern string `json:"matched_pattern"`
	} `json:"cases"`
}

func TestConformanceCorpus(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "..", "..", "conformance", "robots", "v1", "cases.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var corpus conformanceCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Format != "tw.robots-conformance/0.1" || len(corpus.Cases) < 16 {
		t.Fatal("incomplete robots conformance corpus")
	}
	for _, testCase := range corpus.Cases {
		t.Run(testCase.Name, func(t *testing.T) {
			document, err := Parse([]byte(testCase.Robots))
			if err != nil {
				t.Fatal(err)
			}
			decision, err := document.Evaluate(corpus.ProductToken, testCase.PathQuery)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Allowed != testCase.Allowed || decision.Pattern != testCase.MatchedPattern {
				t.Fatalf("got allowed=%v pattern=%q, want allowed=%v pattern=%q", decision.Allowed, decision.Pattern, testCase.Allowed, testCase.MatchedPattern)
			}
		})
	}
}

func TestMalformedInputIsBoundedAndFailsClosed(t *testing.T) {
	if _, err := Parse(make([]byte, MaxBytes+1)); err == nil {
		t.Fatal("oversized document accepted")
	}
	if _, err := Parse([]byte{0xff}); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
	document, err := Parse([]byte("User-agent: *\nDisallow: /valid\nDisallow: bad\x00path\n"))
	if err != nil {
		t.Fatal(err)
	}
	if document.ParseErrors != 1 {
		t.Fatalf("parse errors = %d, want 1", document.ParseErrors)
	}
	decision, err := document.Evaluate("TWIRXBot", "/valid")
	if err != nil || decision.Allowed {
		t.Fatalf("parseable rule lost: %#v %v", decision, err)
	}
}

func TestFetchOutcomeClassificationDoesNotGrantAccess(t *testing.T) {
	tests := []struct {
		status    int
		redirects int
		failed    bool
		expected  FetchResult
	}{
		{status: 200, expected: FetchSuccessful},
		{status: 404, expected: FetchUnavailable},
		{status: 503, expected: FetchUnreachable},
		{status: 200, failed: true, expected: FetchUnreachable},
		{status: 200, redirects: 6, expected: FetchRedirectLimit},
	}
	for _, testCase := range tests {
		if result := ClassifyFetch(testCase.status, testCase.redirects, testCase.failed); result != testCase.expected {
			t.Fatalf("ClassifyFetch(%d, %d, %v) = %q, want %q", testCase.status, testCase.redirects, testCase.failed, result, testCase.expected)
		}
	}
}

func TestInvalidTargetsAndProductTokensAreRejected(t *testing.T) {
	document, err := Parse([]byte("User-agent: *\nDisallow: /\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct{ agent, target string }{{"bot/1", "/"}, {"TWIRXBot", "https://example.test/"}, {"TWIRXBot", "/bad%xx"}} {
		if _, err := document.Evaluate(testCase.agent, testCase.target); err == nil {
			t.Fatalf("unsafe input accepted: %#v", testCase)
		}
	}
	if _, err := document.Evaluate("TWIRXBot", "/"+strings.Repeat("a", MaxTargetBytes)); err == nil {
		t.Fatal("oversized target accepted")
	}
}

func FuzzParseAndEvaluate(f *testing.F) {
	f.Add([]byte("User-agent: *\nDisallow: /private\n"), "/private")
	f.Add([]byte("User-agent: TWIRXBot\r\nAllow: /\r\n"), "/")
	f.Fuzz(func(t *testing.T, data []byte, target string) {
		if len(data) > MaxBytes {
			return
		}
		document, err := Parse(data)
		if err != nil {
			return
		}
		if target == "" || !strings.HasPrefix(target, "/") || len(target) > 4096 {
			return
		}
		_, _ = document.Evaluate("TWIRXBot", target)
	})
}
