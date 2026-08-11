package egressworker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

type staticRetriever struct {
	result *safefetch.Result
	err    error
	calls  int
}

func (r *staticRetriever) Fetch(context.Context, string) (*safefetch.Result, error) {
	r.calls++
	return r.result, r.err
}

func validOrder(id string, now time.Time) *LoadedWorkOrder {
	order := WorkOrder{
		Format: WorkOrderFormat, ID: id, OriginID: "example-org", Purpose: "profile", AuthorityClass: "reviewed_policy", Method: "GET",
		URL: "https://example.org/data", AllowedHosts: []string{"example.org"}, PolicyDecision: "profile_only",
		PolicyEvidenceDigest: "sha256:" + strings.Repeat("a", 64), DecisionDigest: "sha256:" + strings.Repeat("b", 64),
		ApprovalReference: "atlas/admissions/example-org/decision.json", NotBefore: now.Add(-time.Minute).Format(time.RFC3339Nano),
		ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano), MaxRedirects: 2, MaxBodyBytes: 1 << 20,
		TimeoutMillis: 5000, ConnectTimeoutMillis: 1000, HeaderTimeoutMillis: 2000,
		MaxConsecutiveFailures: 2, CircuitCooldownSeconds: 300,
	}
	data, err := marshal(order)
	if err != nil {
		panic(err)
	}
	return &LoadedWorkOrder{Order: order, Digest: digest(data), bytes: data}
}

func TestWorkOrderRejectsUnsealedDestinations(t *testing.T) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*WorkOrder)
	}{
		{"private IPv4", func(w *WorkOrder) { w.URL, w.AllowedHosts = "https://10.0.0.1/data", []string{"10.0.0.1"} }},
		{"private IPv6", func(w *WorkOrder) { w.URL, w.AllowedHosts = "https://[fd00::1]/data", []string{"fd00::1"} }},
		{"decimal IPv4", func(w *WorkOrder) { w.URL, w.AllowedHosts = "https://2130706433/data", []string{"2130706433"} }},
		{"octal IPv4", func(w *WorkOrder) { w.URL, w.AllowedHosts = "https://0177.0.0.1/data", []string{"0177.0.0.1"} }},
		{"hex IPv4", func(w *WorkOrder) {
			w.URL, w.AllowedHosts = "https://0x7f.0x0.0x0.0x1/data", []string{"0x7f.0x0.0x0.0x1"}
		}},
		{"credentials", func(w *WorkOrder) { w.URL = "https://user:pass@example.org/data" }},
		{"non HTTP", func(w *WorkOrder) { w.URL = "file:///etc/passwd" }},
		{"unexpected port", func(w *WorkOrder) { w.URL = "https://example.org:8443/data" }},
		{"unadmitted host", func(w *WorkOrder) { w.URL = "https://other.example/data" }},
		{"profile policy observation", func(w *WorkOrder) { w.Purpose = "observation" }},
		{"uncertain reviewed policy", func(w *WorkOrder) { w.PolicyDecision = "uncertain" }},
		{"evidence collection beyond robots", func(w *WorkOrder) { w.AuthorityClass, w.PolicyDecision = "policy_evidence_collection", "uncertain" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			order := validOrder("test-order", now).Order
			test.mutate(&order)
			if err := order.Validate(); err == nil {
				t.Fatal("adversarial work order was accepted")
			}
		})
	}
}

func TestPendingPolicyBootstrapIsOnlyExactRobotsEvidence(t *testing.T) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	order := validOrder("robots-order", now).Order
	order.AuthorityClass, order.PolicyDecision, order.Purpose, order.URL = "policy_evidence_collection", "uncertain", "robots", "https://example.org/robots.txt"
	if err := order.Validate(); err != nil {
		t.Fatalf("bounded robots evidence order rejected: %v", err)
	}
	order.URL = "https://example.org/other"
	if err := order.Validate(); err == nil {
		t.Fatal("pending policy authorized a non-robots route")
	}
}

func TestDisabledAndRevokedOrdersFailBeforeRetrieval(t *testing.T) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	for _, control := range []*Control{
		{Format: ControlFormat, Enabled: false},
		{Format: ControlFormat, Enabled: true, EmergencyStop: true},
		{Format: ControlFormat, Enabled: true, RevokedOrigins: []string{"example-org"}},
		{Format: ControlFormat, Enabled: true, RevokedOrders: []string{"test-order"}},
	} {
		fetcher := &staticRetriever{err: errors.New("must not execute")}
		spool, state := t.TempDir(), t.TempDir()
		if _, err := execute(context.Background(), validOrder("test-order", now), control, spool, state, now, fetcher); err == nil {
			t.Fatal("disabled or revoked work order was accepted")
		}
		if fetcher.calls != 0 {
			t.Fatal("retrieval ran before control-plane rejection")
		}
		entries, err := os.ReadDir(spool)
		if err != nil || len(entries) != 0 {
			t.Fatalf("rejected work order created spool state: %v, %d", err, len(entries))
		}
	}
}

func TestInvalidControlAndSymlinkedSpoolFailBeforeRetrieval(t *testing.T) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	fetcher := &staticRetriever{err: errors.New("must not execute")}
	if _, err := execute(context.Background(), validOrder("test-order", now), &Control{Format: "invalid", Enabled: true}, t.TempDir(), t.TempDir(), now, fetcher); err == nil {
		t.Fatal("invalid control artifact was accepted")
	}
	parent, target := t.TempDir(), t.TempDir()
	spool := filepath.Join(parent, "spool")
	if err := os.Symlink(target, spool); err != nil {
		t.Fatal(err)
	}
	if _, err := execute(context.Background(), validOrder("test-order", now), &Control{Format: ControlFormat, Enabled: true}, spool, t.TempDir(), now, fetcher); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("symlinked spool root was accepted: %v", err)
	}
	if fetcher.calls != 0 {
		t.Fatal("retrieval ran before filesystem-boundary rejection")
	}
}

func TestEvidenceSpoolIsManifestLastAndIndependentlyVerified(t *testing.T) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	loaded := validOrder("evidence-order", now)
	fetcher := &staticRetriever{result: &safefetch.Result{
		RequestURL: loaded.Order.URL, FinalURL: loaded.Order.URL, Method: "GET", Status: 200,
		MediaType: "application/json", RetrievedAt: now, Body: []byte(`{"status":"public"}`),
	}}
	spool, state := t.TempDir(), t.TempDir()
	control := &Control{Format: ControlFormat, Enabled: true}
	result, err := execute(context.Background(), loaded, control, spool, state, now, fetcher)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(spool, loaded.Order.ID, strings.TrimPrefix(loaded.Digest, "sha256:"))
	verified, err := VerifySpool(root, loaded.Order.MaxBodyBytes)
	if err != nil {
		t.Fatal(err)
	}
	if verified.BodyDigest != result.BodyDigest || verified.ObservationDigest != result.ObservationDigest || fetcher.calls != 1 {
		t.Fatalf("verified result mismatch: %#v %#v", result, verified)
	}
	if err := os.Remove(filepath.Join(root, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySpool(root, loaded.Order.MaxBodyBytes); err == nil || !strings.Contains(err.Error(), "manifest unavailable") {
		t.Fatalf("partial spool was admitted: %v", err)
	}
}

func TestSpoolTamperingFailsClosed(t *testing.T) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	loaded := validOrder("tamper-order", now)
	fetcher := &staticRetriever{result: &safefetch.Result{RequestURL: loaded.Order.URL, FinalURL: loaded.Order.URL, Method: "GET", Status: 200, MediaType: "text/plain", RetrievedAt: now, Body: []byte("evidence")}}
	spool, state := t.TempDir(), t.TempDir()
	if _, err := execute(context.Background(), loaded, &Control{Format: ControlFormat, Enabled: true}, spool, state, now, fetcher); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(spool, loaded.Order.ID, strings.TrimPrefix(loaded.Digest, "sha256:"))
	resultPath := filepath.Join(root, "result.json")
	if err := os.Chmod(resultPath, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, []byte("{}\n"), 0o440); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySpool(root, loaded.Order.MaxBodyBytes); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("tampered result admitted: %v", err)
	}
}

func TestCircuitBreakerOpensAndGlobalLeaseBoundsConcurrency(t *testing.T) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	spool, state := t.TempDir(), t.TempDir()
	control := &Control{Format: ControlFormat, Enabled: true}
	fetcher := &staticRetriever{err: errors.New("network unavailable")}
	for _, id := range []string{"failure-one", "failure-two"} {
		if _, err := execute(context.Background(), validOrder(id, now), control, spool, state, now, fetcher); err == nil {
			t.Fatal("failed retrieval reported success")
		}
	}
	if _, err := execute(context.Background(), validOrder("failure-three", now), control, spool, state, now, fetcher); err == nil || !strings.Contains(err.Error(), "circuit breaker") {
		t.Fatalf("open circuit did not reject order: %v", err)
	}
	if fetcher.calls != 2 {
		t.Fatalf("retriever called after circuit opened: %d", fetcher.calls)
	}
	lease, err := acquireLease(state)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, err := acquireLease(state); err == nil {
		t.Fatal("second global concurrency lease was admitted")
	}
}

func FuzzWorkOrderJSON(f *testing.F) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	f.Add(validOrder("fuzz-order", now).bytes)
	f.Add([]byte(`{"format":"tw.egress-work-order/0.1","url":"file:///etc/passwd"}`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		var order WorkOrder
		if err := decode(data, &order, MaxWorkOrder); err == nil {
			_ = order.Validate()
		}
	})
}

func BenchmarkVerifyEvidenceSpool(b *testing.B) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	loaded := validOrder("benchmark-order", now)
	fetcher := &staticRetriever{result: &safefetch.Result{RequestURL: loaded.Order.URL, FinalURL: loaded.Order.URL, Method: "GET", Status: 200, MediaType: "application/json", RetrievedAt: now, Body: []byte(`{"status":"public"}`)}}
	spool, state := b.TempDir(), b.TempDir()
	if _, err := execute(context.Background(), loaded, &Control{Format: ControlFormat, Enabled: true}, spool, state, now, fetcher); err != nil {
		b.Fatal(err)
	}
	root := filepath.Join(spool, loaded.Order.ID, strings.TrimPrefix(loaded.Digest, "sha256:"))
	b.ReportAllocs()
	b.SetBytes(int64(len(fetcher.result.Body)))
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := VerifySpool(root, loaded.Order.MaxBodyBytes); err != nil {
			b.Fatal(err)
		}
	}
}
