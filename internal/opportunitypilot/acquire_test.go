package opportunitypilot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

type memoryRangeFetcher struct {
	data        []byte
	calls       int
	retrievedAt time.Time
}

func (f *memoryRangeFetcher) FetchRange(_ context.Context, rawURL, requested string) (*safefetch.Result, error) {
	f.calls++
	parts := strings.Split(strings.TrimPrefix(requested, "bytes="), "-")
	start, _ := strconv.ParseUint(parts[0], 10, 64)
	end, _ := strconv.ParseUint(parts[1], 10, 64)
	if rawURL != SourceURL || end >= uint64(len(f.data)) {
		return nil, fmt.Errorf("unsealed range")
	}
	retrievedAt := f.retrievedAt
	if retrievedAt.IsZero() {
		retrievedAt = time.Date(2026, 8, 12, 4, 1, 0, 0, time.UTC)
	}
	return &safefetch.Result{RequestURL: SourceURL, FinalURL: SourceURL, Method: "GET", Status: 206, MediaType: "application/zip", RetrievedAt: retrievedAt, Body: append([]byte(nil), f.data[start:end+1]...), RequestedRange: requested, ContentRange: fmt.Sprintf("bytes %d-%d/%d", start, end, len(f.data))}, nil
}

func loadedTestOrder(t *testing.T) *LoadedWorkOrder {
	t.Helper()
	order := validWorkOrder()
	data, err := marshalJSON(order, MaxWorkOrder)
	if err != nil {
		t.Fatal(err)
	}
	return &LoadedWorkOrder{Order: order, Bytes: data, Digest: digest(data), AuthorityVerified: true}
}

func TestAcquirePublishesManifestLastAndVerifies(t *testing.T) {
	data := []byte(strings.Repeat("z", int(2*RangeBytes+31)))
	fetcher := &memoryRangeFetcher{data: data, retrievedAt: time.Date(2026, 8, 12, 4, 1, 0, 500000000, time.UTC)}
	parent := t.TempDir()
	output := filepath.Join(parent, "acquisition")
	state := filepath.Join(parent, "state")
	if err := os.Mkdir(state, 0o700); err != nil {
		t.Fatal(err)
	}
	// The range evidence retains nanoseconds while the manifest timestamps are
	// canonical seconds. Verification must admit a range observed within the
	// same second as manifest-last completion.
	times := []time.Time{time.Date(2026, 8, 12, 4, 1, 0, 100000000, time.UTC), time.Date(2026, 8, 12, 4, 1, 0, 900000000, time.UTC)}
	now := func() time.Time {
		value := times[0]
		if len(times) > 1 {
			times = times[1:]
		}
		return value
	}
	loaded := loadedTestOrder(t)
	manifest, err := acquire(context.Background(), loaded, &Control{Format: ControlFormat, Enabled: true}, output, state, now, func(time.Duration) {}, fetcher)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.NetworkRequests != 3 || manifest.TransferredBytes != uint64(len(data)) || fetcher.calls != 3 || manifest.SchedulerEnabled || manifest.RawEvidencePublic {
		t.Fatalf("unexpected manifest: %+v calls=%d", manifest, fetcher.calls)
	}
	verified, err := VerifyAcquisition(output, loaded)
	if err != nil || verified.ArchiveDigest != digest(data) {
		t.Fatalf("verify acquisition: %v %+v", err, verified)
	}
	manifestPath := filepath.Join(output, "acquisition-manifest.json")
	originalManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	verified.Artifacts[0].Digest = "sha256:" + strings.Repeat("f", 64)
	tampered, err := marshalJSON(verified, MaxManifest)
	if err != nil || os.Chmod(manifestPath, 0o640) != nil || os.WriteFile(manifestPath, tampered, 0o440) != nil {
		t.Fatal("cannot prepare tampered manifest")
	}
	if _, err := VerifyAcquisition(output, loaded); err == nil {
		t.Fatal("tampered acquisition artifact index was admitted")
	}
	if err := os.WriteFile(manifestPath, originalManifest, 0o440); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(output, "acquisition-manifest.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyAcquisition(output, loaded); err == nil {
		t.Fatal("partial acquisition without final manifest was admitted")
	}
	if _, err := acquire(context.Background(), loaded, &Control{Format: ControlFormat, Enabled: true}, filepath.Join(parent, "second-acquisition"), state, func() time.Time { return time.Date(2026, 8, 12, 4, 3, 0, 0, time.UTC) }, func(time.Duration) {}, fetcher); err == nil {
		t.Fatal("consumed manual-once work order executed twice")
	}
}

func TestAcquireFailsBeforeNetworkWhenDisabledOrExpired(t *testing.T) {
	fetcher := &memoryRangeFetcher{data: []byte(strings.Repeat("x", int(2*RangeBytes)))}
	loaded := loadedTestOrder(t)
	for _, test := range []struct {
		control Control
		now     time.Time
	}{
		{control: Control{Format: ControlFormat}, now: time.Date(2026, 8, 12, 4, 1, 0, 0, time.UTC)},
		{control: Control{Format: ControlFormat, Enabled: true, EmergencyStop: true}, now: time.Date(2026, 8, 12, 4, 1, 0, 0, time.UTC)},
		{control: Control{Format: ControlFormat, Enabled: true}, now: time.Date(2026, 8, 13, 4, 0, 0, 0, time.UTC)},
	} {
		output := filepath.Join(t.TempDir(), "acquisition")
		state := filepath.Join(t.TempDir(), "state")
		if err := os.Mkdir(state, 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := acquire(context.Background(), loaded, &test.control, output, state, func() time.Time { return test.now }, func(time.Duration) {}, fetcher); err == nil {
			t.Fatal("disabled or expired acquisition succeeded")
		}
	}
	if fetcher.calls != 0 {
		t.Fatalf("network ran before authority rejection: %d calls", fetcher.calls)
	}
}

func TestRangeResponseRejectsRedirectMismatchAndOversize(t *testing.T) {
	item := ByteRange{Index: 0, Start: 0, End: RangeBytes - 1}
	base := &safefetch.Result{RequestURL: SourceURL, FinalURL: SourceURL, Method: "GET", Status: 206, MediaType: "application/zip", RetrievedAt: time.Date(2026, 8, 12, 4, 1, 0, 0, time.UTC), Body: make([]byte, RangeBytes), RequestedRange: item.Header(), ContentRange: fmt.Sprintf("bytes 0-%d/%d", RangeBytes-1, 2*RangeBytes)}
	if _, err := validateRangeResult(base, item, 2*RangeBytes); err != nil {
		t.Fatal(err)
	}
	mutations := []func(*safefetch.Result){
		func(r *safefetch.Result) { r.FinalURL = "https://example.org/archive.zip" },
		func(r *safefetch.Result) {
			r.Redirects = []safefetch.Redirect{{FromURL: SourceURL, ToURL: SourceURL, Status: 302}}
		},
		func(r *safefetch.Result) { r.Status = 200 },
		func(r *safefetch.Result) { r.ContentRange = "bytes 0-1/2" },
		func(r *safefetch.Result) { r.Body = r.Body[:len(r.Body)-1] },
	}
	for index, mutate := range mutations {
		candidate := *base
		candidate.Body = append([]byte(nil), base.Body...)
		mutate(&candidate)
		if _, err := validateRangeResult(&candidate, item, 2*RangeBytes); err == nil {
			t.Fatalf("unsafe range mutation %d was accepted", index)
		}
	}
}
