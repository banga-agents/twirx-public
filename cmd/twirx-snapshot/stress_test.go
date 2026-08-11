package main

import (
	"errors"
	"testing"
	"time"
)

func TestLoopbackStressTransport(t *testing.T) {
	for _, base := range []string{"http://127.0.0.1:8091", "http://[::1]:8091"} {
		if _, transport, err := loopbackStressTransport(base); err != nil || transport.Proxy != nil {
			t.Fatalf("rejected safe base %q: %v", base, err)
		}
	}
	for _, base := range []string{"https://127.0.0.1:8091", "http://localhost:8091", "http://0.0.0.0:8091", "http://127.0.0.1:8091/path", "http://127.0.0.1"} {
		if _, _, err := loopbackStressTransport(base); err == nil {
			t.Fatalf("accepted unsafe base %q", base)
		}
	}
}

func TestStressReportRequiresStableIdentities(t *testing.T) {
	outcomes := []stressOutcome{
		{LatencyMicroseconds: 10, SnapshotID: "snapshot", QueryDigest: "query", ResultDigest: "result"},
		{LatencyMicroseconds: 20, SnapshotID: "snapshot", QueryDigest: "query", ResultDigest: "result"},
		{LatencyMicroseconds: 30, SnapshotID: "snapshot", QueryDigest: "query", ResultDigest: "result"},
	}
	report := buildStressReport(outcomes, 2, time.Millisecond)
	if !report.Pass || report.Workload.Successes != 3 || report.Performance.P95Microseconds != 30 {
		t.Fatalf("unexpected passing report: %+v", report)
	}
	outcomes[1].ResultDigest = "changed"
	outcomes[2].Err = errors.New("request failed")
	report = buildStressReport(outcomes, 2, time.Millisecond)
	if report.Pass || report.Workload.Failures != 2 || len(report.Errors) != 2 {
		t.Fatalf("unexpected failing report: %+v", report)
	}
}
