package safefetch

import (
	"compress/gzip"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
)

func TestLoopbackDeniedByDefault(t *testing.T) {
	policy := DefaultPolicy()
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{"127.0.0.1", "::1"} {
		if err := fetcher.validateIP(netip.MustParseAddr(address)); err == nil {
			t.Fatalf("expected loopback address %s to fail", address)
		}
	}
	if _, err := fetcher.validateURL("http://127.0.0.1/"); err != nil {
		t.Fatalf("URL syntax should pass before address policy: %v", err)
	}
	if err := fetcher.validateIP(netip.MustParseAddr("127.0.0.1")); err == nil {
		t.Fatal("expected loopback request to fail")
	}
}

func TestLoopbackExplicitlyAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	policy := DefaultPolicy()
	policy.AllowLoopback = true
	policy.AllowNonStandardPorts = true
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	result, err := fetcher.Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != http.StatusOK || result.MediaType != "application/json" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRejectsEmbeddedCredentials(t *testing.T) {
	fetcher, err := New(DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fetcher.validateURL("https://user:pass@example.com/"); err == nil {
		t.Fatal("expected embedded credentials rejection")
	}
}

func TestRejectsPrivateIPv4AndIPv6(t *testing.T) {
	fetcher, err := New(DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{
		"0.0.0.1", "10.0.0.1", "127.0.0.1", "169.254.169.254", "172.16.0.1", "192.168.0.1",
		"224.0.0.1", "240.0.0.1", "::", "::1", "64:ff9b:1::1", "100::1", "2001:2::1",
		"2001:db8::1", "3fff::1", "fd00::1", "fe80::1", "ff02::1",
	} {
		if err := fetcher.validateIP(netip.MustParseAddr(address)); err == nil {
			t.Fatalf("private address %s was accepted", address)
		}
	}
}

type sequenceResolver struct {
	mu      sync.Mutex
	answers [][]netip.Addr
}

func (r *sequenceResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.answers) == 0 {
		return nil, errors.New("no more answers")
	}
	answer := r.answers[0]
	r.answers = r.answers[1:]
	return answer, nil
}

type pipeDialer struct{}

func (pipeDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	client, server := net.Pipe()
	_ = server.Close()
	return client, nil
}

type countingDialer struct{ calls int }

func (d *countingDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	d.calls++
	return nil, errors.New("not reachable")
}

func TestDNSIsReResolvedAndRebindingToPrivateIsDenied(t *testing.T) {
	policy := DefaultPolicy()
	policy.AllowedHosts = []string{"example.org"}
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	fetcher.resolver = &sequenceResolver{answers: [][]netip.Addr{{netip.MustParseAddr("93.184.216.34")}, {netip.MustParseAddr("127.0.0.1")}}}
	dial := fetcher.safeDialContext(pipeDialer{})
	connection, err := dial(context.Background(), "tcp", "example.org:443")
	if err != nil {
		t.Fatalf("first public resolution failed: %v", err)
	}
	_ = connection.Close()
	if _, err := dial(context.Background(), "tcp", "example.org:443"); err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("private rebinding answer was not rejected: %v", err)
	}
}

func TestMixedPublicAndPrivateDNSAnswerFailsClosed(t *testing.T) {
	policy := DefaultPolicy()
	policy.AllowedHosts = []string{"example.org"}
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	fetcher.resolver = &sequenceResolver{answers: [][]netip.Addr{{netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("169.254.169.254")}}}
	dialer := &countingDialer{}
	if _, err := fetcher.safeDialContext(dialer)(context.Background(), "tcp", "example.org:443"); err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("mixed DNS answer was accepted: %v", err)
	}
	if dialer.calls != 0 {
		t.Fatal("dial occurred before every DNS address was validated")
	}
}

func TestRejectsCarrierGradeNAT(t *testing.T) {
	fetcher, err := New(DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{"100.64.0.1", "100.127.255.254"} {
		if err := fetcher.validateIP(netip.MustParseAddr(address)); err == nil {
			t.Fatalf("carrier-grade NAT address %s was accepted", address)
		}
	}
}

func TestRejectsNonStandardPortUnderPublicPolicy(t *testing.T) {
	fetcher, err := New(DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fetcher.validateURL("https://example.org:8443/"); err == nil {
		t.Fatal("non-standard public port was accepted")
	}
}

func TestRedirectDestinationIsRevalidated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://10.0.0.1/private", http.StatusFound)
	}))
	defer server.Close()

	policy := DefaultPolicy()
	policy.AllowLoopback = true
	policy.AllowNonStandardPorts = true
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fetcher.Fetch(context.Background(), server.URL); err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("redirect to private address was not rejected: %v", err)
	}
}

func TestRedirectLoopIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer server.Close()
	policy := DefaultPolicy()
	policy.AllowLoopback = true
	policy.AllowNonStandardPorts = true
	policy.MaxRedirects = 2
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fetcher.Fetch(context.Background(), server.URL+"/loop"); err == nil || !strings.Contains(err.Error(), "redirect limit") {
		t.Fatalf("redirect loop was not bounded: %v", err)
	}
}

func TestTLSCertificateIsVerified(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	policy := DefaultPolicy()
	policy.AllowLoopback = true
	policy.AllowNonStandardPorts = true
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fetcher.Fetch(context.Background(), server.URL); err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("untrusted TLS certificate was not rejected: %v", err)
	}
}

func TestAllowedHostConstrainsInitialAndRedirectURLs(t *testing.T) {
	policy := DefaultPolicy()
	policy.AllowedHosts = []string{"api.example.org"}
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fetcher.validateURL("https://api.example.org/v1/data"); err != nil {
		t.Fatalf("admitted host rejected: %v", err)
	}
	if _, err := fetcher.validateURL("https://cdn.example.org/v1/data"); err == nil || !strings.Contains(err.Error(), "not admitted") {
		t.Fatalf("unadmitted host accepted: %v", err)
	}
	policy.AllowedHosts = []string{"API.example.org"}
	if _, err := New(policy); err == nil {
		t.Fatal("non-canonical allowed host accepted")
	}
}

func TestResponseLimitAppliesAfterDecompression(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "application/json")
		writer := gzip.NewWriter(w)
		_, _ = writer.Write([]byte(`{"value":"` + strings.Repeat("a", 256) + `"}`))
		_ = writer.Close()
	}))
	defer server.Close()

	policy := DefaultPolicy()
	policy.AllowLoopback = true
	policy.AllowNonStandardPorts = true
	policy.MaxBodyBytes = 64
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fetcher.Fetch(context.Background(), server.URL); err == nil || !strings.Contains(err.Error(), "decompressed limit") {
		t.Fatalf("oversized decompressed response was not rejected: %v", err)
	}
}

func TestExactRangeUsesIdentityEncodingAndPreservesContentRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "bytes=4-7" || r.Header.Get("Accept-Encoding") != "identity" {
			t.Errorf("unexpected range request headers: range=%q encoding=%q", r.Header.Get("Range"), r.Header.Get("Accept-Encoding"))
		}
		w.Header().Set("Content-Range", "bytes 4-7/16")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("4567"))
	}))
	defer server.Close()

	policy := DefaultPolicy()
	policy.AllowLoopback = true
	policy.AllowNonStandardPorts = true
	policy.MaxBodyBytes = 8
	policy.MaxRedirects = 0
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	result, err := fetcher.FetchRange(context.Background(), server.URL, "bytes=4-7")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != http.StatusPartialContent || result.RequestedRange != "bytes=4-7" || result.ContentRange != "bytes 4-7/16" || string(result.Body) != "4567" {
		t.Fatalf("unexpected range result: %+v", result)
	}
	for _, invalid := range []string{"", "bytes=7-4", "bytes=-4", "bytes=0-8", "items=0-1", "bytes=0-1,4-5"} {
		if _, err := fetcher.FetchRange(context.Background(), server.URL, invalid); err == nil {
			t.Fatalf("invalid range %q was accepted", invalid)
		}
	}
}

func TestRedirectChainAndStrictHeaderAllowlist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/final", http.StatusTemporaryRedirect)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Language", "en")
		w.Header().Set("Set-Cookie", "private=session")
		w.Header().Set("X-Internal-Trace", "secret")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	policy := DefaultPolicy()
	policy.AllowLoopback = true
	policy.AllowNonStandardPorts = true
	fetcher, err := New(policy)
	if err != nil {
		t.Fatal(err)
	}
	result, err := fetcher.Fetch(context.Background(), server.URL+"/start")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Redirects) != 1 || result.Redirects[0].Status != http.StatusTemporaryRedirect || result.Redirects[0].ToURL != server.URL+"/final" {
		t.Fatalf("unexpected redirects: %#v", result.Redirects)
	}
	if len(result.Headers) != 2 || result.Headers[0].Name != "content-language" || result.Headers[1].Name != "content-type" {
		t.Fatalf("unexpected selected headers: %#v", result.Headers)
	}
	for _, header := range result.Headers {
		if strings.Contains(header.Value, "private") || strings.Contains(header.Value, "secret") {
			t.Fatalf("sensitive header recorded: %#v", header)
		}
	}
}
