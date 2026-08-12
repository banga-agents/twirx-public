package opportunitypilot

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func validWorkOrder() WorkOrder {
	return WorkOrder{
		Format: WorkOrderFormat, ID: "grants-gov-20260811", OriginID: OriginID,
		SourceURL: SourceURL, SourceFilename: SourceFilename, ExecutionMode: "manual_once",
		PolicyReviewState: "completed", PolicyDecision: "permit_with_constraints",
		PolicyEvidenceDigest: "sha256:" + strings.Repeat("a", 64), DecisionDigest: "sha256:" + strings.Repeat("b", 64),
		ApprovalReference: "reports/e4-opportunity-policy-decision-proposal.md", ApprovedBy: "Genesis steward", ApprovedAt: "2026-08-12T04:00:00Z",
		NotBefore: "2026-08-12T04:00:00Z", ExpiresAt: "2026-08-13T04:00:00Z",
		RangeBytes: RangeBytes, MaximumRequests: MaximumRequests, MaximumArchiveBytes: MaximumArchive,
		MaximumExpandedXMLBytes: MaximumExpandedXML, MaximumRecords: MaximumRecords,
		RequestIntervalMillis: 2000, RequestTimeoutMillis: 30000,
	}
}

func TestCommittedAuthorityReconciles(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	loaded, err := LoadWorkOrder(filepath.Join(root, "atlas", "e4-plans", "grants-gov-20260811.json"))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AuthorityVerified {
		t.Fatal("structural work-order load implied human authority")
	}
	if err := VerifyAuthority(root, loaded); err != nil {
		t.Fatal(err)
	}
	if !loaded.AuthorityVerified {
		t.Fatal("exact committed authority did not verify")
	}
}

func TestWorkOrderIsExactAndPolicyBound(t *testing.T) {
	base := validWorkOrder()
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	tests := []func(*WorkOrder){
		func(w *WorkOrder) { w.SourceURL = "https://example.org/extract.zip" },
		func(w *WorkOrder) { w.SourceURL = SourceURL + "?later=true" },
		func(w *WorkOrder) { w.PolicyReviewState = "pending" },
		func(w *WorkOrder) { w.PolicyDecision = "uncertain" },
		func(w *WorkOrder) { w.MaximumRequests++ },
		func(w *WorkOrder) { w.MaximumArchiveBytes++ },
		func(w *WorkOrder) { w.RequestIntervalMillis = 0 },
		func(w *WorkOrder) { w.Redirects = 1 },
		func(w *WorkOrder) { w.SchedulerEnabled = true },
		func(w *WorkOrder) { w.RawEvidencePublic = true },
		func(w *WorkOrder) { w.ContactProjectionEnabled = true },
	}
	for index, mutate := range tests {
		candidate := base
		mutate(&candidate)
		if err := candidate.Validate(); err == nil {
			t.Fatalf("unsafe mutation %d was accepted", index)
		}
	}
}

func TestBuildRangesIsCompleteAndBounded(t *testing.T) {
	total := uint64(77*RangeBytes + 19)
	ranges, err := BuildRanges(total)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranges) != 78 || ranges[0].Header() != "bytes=0-1048575" || ranges[len(ranges)-1].End != total-1 {
		t.Fatalf("unexpected ranges: first=%+v last=%+v count=%d", ranges[0], ranges[len(ranges)-1], len(ranges))
	}
	for index, item := range ranges {
		if item.Index != uint64(index) || item.End < item.Start || item.End-item.Start >= RangeBytes || index > 0 && item.Start != ranges[index-1].End+1 {
			t.Fatalf("range %d is not contiguous: %+v", index, item)
		}
	}
	for _, total := range []uint64{0, MaximumArchive + 1} {
		if _, err := BuildRanges(total); err == nil {
			t.Fatalf("unsafe total %d was accepted", total)
		}
	}
}

func TestControlFailsClosed(t *testing.T) {
	order := validWorkOrder()
	for _, control := range []Control{
		{Format: ControlFormat},
		{Format: ControlFormat, Enabled: true, EmergencyStop: true},
		{Format: ControlFormat, Enabled: true, RevokedOrders: []string{order.ID}},
		{Format: ControlFormat, Enabled: true, RevokedOrigins: []string{order.OriginID}},
	} {
		if err := control.Permits(order); err == nil {
			t.Fatal("disabled or revoked order was permitted")
		}
	}
	if err := (Control{Format: ControlFormat, Enabled: true}).Permits(order); err != nil {
		t.Fatal(err)
	}
}
