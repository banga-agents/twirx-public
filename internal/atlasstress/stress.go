// Package atlasstress exercises the complete offline Genesis-500 Atlas
// control plane. It never opens a socket or performs origin retrieval.
package atlasstress

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/atlasapi"
)

const Format = "tw.atlas-500-runtime-stress/0.1"

type Config struct {
	Rounds  int
	Workers int
}

type Report struct {
	Format          string            `json:"format"`
	Mode            string            `json:"mode"`
	NetworkAccess   string            `json:"network_access"`
	SelectionDigest string            `json:"selection_digest"`
	Corpus          CorpusEvidence    `json:"corpus"`
	Frontier        FrontierEvidence  `json:"frontier"`
	Discovery       DiscoveryEvidence `json:"discovery"`
	Adversarial     AdversarialResult `json:"adversarial"`
	Workload        WorkloadResult    `json:"workload"`
	Performance     PerformanceResult `json:"performance"`
	Memory          MemoryResult      `json:"memory"`
	Statement       string            `json:"statement"`
}

type CorpusEvidence struct {
	SelectedOrigins int            `json:"selected_origins"`
	DomainFamilies  int            `json:"domain_families"`
	FamilyCounts    map[string]int `json:"family_counts"`
}

type FrontierEvidence struct {
	Decisions int            `json:"decisions"`
	Jobs      int            `json:"jobs"`
	Reasons   map[string]int `json:"reasons"`
}

type DiscoveryEvidence struct {
	ListPages         int    `json:"list_pages"`
	UniqueOrigins     int    `json:"unique_origins"`
	DirectLookups     int    `json:"direct_lookups"`
	FilterQueries     int    `json:"filter_queries"`
	ResponseSetDigest string `json:"response_set_digest"`
}

type AdversarialResult struct {
	Rejected int  `json:"rejected"`
	Recovery bool `json:"recovery"`
}

type WorkloadResult struct {
	Rounds        int `json:"rounds"`
	Workers       int `json:"workers"`
	Requests      int `json:"requests"`
	Successes     int `json:"successes"`
	Failures      int `json:"failures"`
	ResponseBytes int `json:"response_bytes"`
}

type PerformanceResult struct {
	WallMicroseconds  int64   `json:"wall_microseconds"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	MeanMicroseconds  float64 `json:"mean_microseconds"`
	P50Microseconds   int64   `json:"p50_microseconds"`
	P95Microseconds   int64   `json:"p95_microseconds"`
	P99Microseconds   int64   `json:"p99_microseconds"`
	MaxMicroseconds   int64   `json:"max_microseconds"`
}

type MemoryResult struct {
	TotalAllocBytes uint64 `json:"total_alloc_bytes"`
	HeapAllocBytes  uint64 `json:"heap_alloc_bytes_after"`
	HeapObjects     uint64 `json:"heap_objects_after"`
}

type listResponse struct {
	Origins    []atlasapi.OriginView `json:"origins"`
	Total      int                   `json:"total"`
	Cursor     int                   `json:"cursor"`
	NextCursor *int                  `json:"next_cursor"`
}

type response struct {
	status int
	body   []byte
}

// RunLoopback exercises a separately running Atlas HTTP service. The base
// address must be a literal loopback HTTP endpoint; hostnames and redirects
// are intentionally unsupported so the stress tool cannot become an egress
// path.
func RunLoopback(ctx context.Context, base string, selection *atlas.Selection, plan atlas.FrontierPlan, config Config) (Report, error) {
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Port() == "" {
		return Report{}, errors.New("atlasstress: base must be an exact literal-loopback HTTP origin with an explicit port")
	}
	address, err := netip.ParseAddr(parsed.Hostname())
	if err != nil || !address.IsLoopback() {
		return Report{}, errors.New("atlasstress: base host must be a literal loopback address")
	}
	dialAddress := parsed.Host
	dialer := &net.Dialer{Timeout: 2 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(dialContext context.Context, network, address string) (net.Conn, error) {
			if network != "tcp" && network != "tcp4" && network != "tcp6" || address != dialAddress {
				return nil, errors.New("atlasstress: transport attempted a non-loopback or substituted destination")
			}
			return dialer.DialContext(dialContext, network, dialAddress)
		},
		DisableCompression: true,
	}
	defer transport.CloseIdleConnections()
	proxy := &loopbackHandler{ctx: ctx, base: base, client: &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("atlasstress: redirects are forbidden")
		},
	}}
	report, err := Run(ctx, proxy, selection, plan, config)
	if err != nil {
		return Report{}, err
	}
	report.Mode = "literal_loopback_http"
	report.NetworkAccess = "literal_loopback_only"
	return report, nil
}

type loopbackHandler struct {
	ctx    context.Context
	base   string
	client *http.Client
}

func (h *loopbackHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	upstream, err := http.NewRequestWithContext(h.ctx, request.Method, h.base+request.URL.RequestURI(), nil)
	if err != nil {
		http.Error(writer, "request construction failed", http.StatusBadGateway)
		return
	}
	result, err := h.client.Do(upstream)
	if err != nil {
		http.Error(writer, "loopback request failed", http.StatusBadGateway)
		return
	}
	defer result.Body.Close()
	body, err := io.ReadAll(io.LimitReader(result.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		http.Error(writer, "loopback response exceeded bound", http.StatusBadGateway)
		return
	}
	writer.WriteHeader(result.StatusCode)
	_, _ = writer.Write(body)
}

func Run(ctx context.Context, handler http.Handler, selection *atlas.Selection, plan atlas.FrontierPlan, config Config) (Report, error) {
	if ctx == nil || handler == nil || selection == nil {
		return Report{}, errors.New("atlasstress: context, handler, and selection are required")
	}
	if config.Rounds < 1 || config.Rounds > 1000 || config.Workers < 1 || config.Workers > 128 {
		return Report{}, errors.New("atlasstress: rounds must be 1..1000 and workers must be 1..128")
	}
	if err := selection.Validate(); err != nil {
		return Report{}, err
	}
	if len(plan.Decisions) != atlas.RequiredCandidates {
		return Report{}, fmt.Errorf("atlasstress: frontier covers %d origins, want %d", len(plan.Decisions), atlas.RequiredCandidates)
	}

	views, pages, err := enumerate(handler)
	if err != nil {
		return Report{}, err
	}
	selectionByID := make(map[string]atlas.Candidate, len(selection.Candidates))
	for _, candidate := range selection.Candidates {
		selectionByID[candidate.ID] = candidate
	}
	if len(views) != len(selectionByID) {
		return Report{}, fmt.Errorf("atlasstress: enumerated %d unique origins, want %d", len(views), len(selectionByID))
	}

	baseline := make(map[string][]byte, len(selection.Candidates))
	for id, candidate := range selectionByID {
		view, exists := views[id]
		if !exists || view.CanonicalOrigin != candidate.CanonicalOrigin || view.DomainFamily != candidate.DomainFamily {
			return Report{}, fmt.Errorf("atlasstress: list view disagrees with selection for %s", id)
		}
		result, requestErr := request(handler, http.MethodGet, "/api/v1/atlas/origins/"+id)
		if requestErr != nil || result.status != http.StatusOK {
			return Report{}, fmt.Errorf("atlasstress: direct lookup %s failed with status %d: %w", id, result.status, requestErr)
		}
		var direct atlasapi.OriginView
		if err := json.Unmarshal(result.body, &direct); err != nil || !reflect.DeepEqual(direct, view) {
			return Report{}, fmt.Errorf("atlasstress: direct lookup disagrees with list view for %s", id)
		}
		baseline[id] = append([]byte(nil), result.body...)
	}

	if err := verifyFilters(handler, selection); err != nil {
		return Report{}, err
	}
	rejected, recovered, err := verifyAdversarial(handler)
	if err != nil {
		return Report{}, err
	}

	frontierReasons := make(map[string]int)
	frontierIDs := make(map[string]struct{}, len(plan.Decisions))
	for _, decision := range plan.Decisions {
		if _, duplicate := frontierIDs[decision.OriginID]; duplicate {
			return Report{}, fmt.Errorf("atlasstress: duplicate frontier decision for %s", decision.OriginID)
		}
		if _, selected := selectionByID[decision.OriginID]; !selected {
			return Report{}, fmt.Errorf("atlasstress: frontier contains unselected origin %s", decision.OriginID)
		}
		frontierIDs[decision.OriginID] = struct{}{}
		frontierReasons[decision.Reason]++
	}

	before := runtime.MemStats{}
	runtime.ReadMemStats(&before)
	durations, responseBytes, failures, wall := repeatedLookups(ctx, handler, baseline, selection.Candidates, config)
	after := runtime.MemStats{}
	runtime.ReadMemStats(&after)
	if failures != 0 {
		return Report{}, fmt.Errorf("atlasstress: %d repeated lookups failed", failures)
	}

	requests := len(durations)
	sortedDurations := append([]int64(nil), durations...)
	sort.Slice(sortedDurations, func(i, j int) bool { return sortedDurations[i] < sortedDurations[j] })
	var durationSum int64
	for _, duration := range durations {
		durationSum += duration
	}
	wallMicros := wall.Microseconds()
	if wallMicros < 1 {
		wallMicros = 1
	}

	report := Report{
		Format: Format, Mode: "offline_in_process", NetworkAccess: "disabled", SelectionDigest: selection.DigestReference(),
		Corpus:      CorpusEvidence{SelectedOrigins: len(selection.Candidates), DomainFamilies: len(selection.FamilyQuotas), FamilyCounts: copyCounts(selection.FamilyQuotas)},
		Frontier:    FrontierEvidence{Decisions: len(plan.Decisions), Jobs: len(plan.Jobs), Reasons: frontierReasons},
		Discovery:   DiscoveryEvidence{ListPages: pages, UniqueOrigins: len(views), DirectLookups: len(baseline), FilterQueries: len(atlas.RequiredFamilyQuotas) + 4, ResponseSetDigest: responseSetDigest(baseline)},
		Adversarial: AdversarialResult{Rejected: rejected, Recovery: recovered},
		Workload:    WorkloadResult{Rounds: config.Rounds, Workers: config.Workers, Requests: requests, Successes: requests, Failures: failures, ResponseBytes: responseBytes},
		Performance: PerformanceResult{WallMicroseconds: wallMicros, RequestsPerSecond: float64(requests) * 1_000_000 / float64(wallMicros), MeanMicroseconds: float64(durationSum) / float64(requests), P50Microseconds: percentile(sortedDurations, 50), P95Microseconds: percentile(sortedDurations, 95), P99Microseconds: percentile(sortedDurations, 99), MaxMicroseconds: sortedDurations[len(sortedDurations)-1]},
		Memory:      MemoryResult{TotalAllocBytes: after.TotalAlloc - before.TotalAlloc, HeapAllocBytes: after.HeapAlloc, HeapObjects: after.HeapObjects},
		Statement:   "All 500 selected origins traversed the real read-only Atlas control plane. This proves catalog-scale handling and explicit frontier outcomes; three human policy decisions are completed, while no scheduler job, live retrieval, new compilation, semantic admission, or new publisher approval is claimed.",
	}
	return report, nil
}

func enumerate(handler http.Handler) (map[string]atlasapi.OriginView, int, error) {
	views := make(map[string]atlasapi.OriginView, atlas.RequiredCandidates)
	cursor, pages := 0, 0
	for {
		result, err := request(handler, http.MethodGet, fmt.Sprintf("/api/v1/atlas/origins?limit=100&cursor=%d", cursor))
		if err != nil || result.status != http.StatusOK {
			return nil, 0, fmt.Errorf("atlasstress: list page failed with status %d: %w", result.status, err)
		}
		var page listResponse
		if err := json.Unmarshal(result.body, &page); err != nil {
			return nil, 0, fmt.Errorf("atlasstress: decode list page: %w", err)
		}
		if page.Total != atlas.RequiredCandidates || page.Cursor != cursor || len(page.Origins) == 0 {
			return nil, 0, errors.New("atlasstress: invalid list pagination metadata")
		}
		for _, view := range page.Origins {
			if _, duplicate := views[view.ID]; duplicate {
				return nil, 0, fmt.Errorf("atlasstress: duplicate list origin %s", view.ID)
			}
			views[view.ID] = view
		}
		pages++
		if page.NextCursor == nil {
			break
		}
		if *page.NextCursor <= cursor {
			return nil, 0, errors.New("atlasstress: pagination did not advance")
		}
		cursor = *page.NextCursor
	}
	return views, pages, nil
}

func verifyFilters(handler http.Handler, selection *atlas.Selection) error {
	for family, expected := range selection.FamilyQuotas {
		result, err := request(handler, http.MethodGet, "/api/v1/atlas/origins?family="+url.QueryEscape(family)+"&limit=1")
		if err != nil || result.status != http.StatusOK {
			return fmt.Errorf("atlasstress: family filter %s failed: %w", family, err)
		}
		var page listResponse
		if err := json.Unmarshal(result.body, &page); err != nil || page.Total != expected {
			return fmt.Errorf("atlasstress: family filter %s returned %d, want %d", family, page.Total, expected)
		}
	}
	for query, expected := range map[string]int{
		"catalog_state=candidate":             497,
		"catalog_state=cataloged":             3,
		"policy_review_state=pending":         497,
		"policy_review_state=completed":       3,
		"technical_stage=semantically_linked": 2,
	} {
		result, err := request(handler, http.MethodGet, "/api/v1/atlas/origins?"+query+"&limit=1")
		if err != nil || result.status != http.StatusOK {
			return fmt.Errorf("atlasstress: state filter %s failed: %w", query, err)
		}
		var page listResponse
		if err := json.Unmarshal(result.body, &page); err != nil || page.Total != expected {
			return fmt.Errorf("atlasstress: state filter %s returned %d, want %d", query, page.Total, expected)
		}
	}
	return nil
}

func verifyAdversarial(handler http.Handler) (int, bool, error) {
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/atlas/origins"},
		{http.MethodGet, "/api/v1/atlas/origins?url=https%3A%2F%2Fexample.com"},
		{http.MethodGet, "/api/v1/atlas/origins?maturity=A2"},
		{http.MethodGet, "/api/v1/atlas/origins?policy_review_state=review_required"},
		{http.MethodGet, "/api/v1/atlas/origins?limit=1000"},
		{http.MethodGet, "/api/v1/atlas/origins?cursor=501"},
		{http.MethodGet, "/api/v1/atlas/origins?limit=1&limit=2"},
		{http.MethodGet, "/api/v1/atlas/origins/twirx-org/extra"},
	}
	for _, item := range cases {
		result, err := request(handler, item.method, item.path)
		if err != nil {
			return 0, false, err
		}
		if result.status < 400 {
			return 0, false, fmt.Errorf("atlasstress: adversarial request accepted: %s %s", item.method, item.path)
		}
	}
	recovery, err := request(handler, http.MethodGet, "/api/v1/atlas/status")
	if err != nil || recovery.status != http.StatusOK {
		return len(cases), false, fmt.Errorf("atlasstress: recovery status failed: %w", err)
	}
	return len(cases), true, nil
}

func repeatedLookups(ctx context.Context, handler http.Handler, baseline map[string][]byte, candidates []atlas.Candidate, config Config) ([]int64, int, int, time.Duration) {
	total := len(candidates) * config.Rounds
	durations := make([]int64, total)
	sizes := make([]int, total)
	failed := make([]bool, total)
	tasks := make(chan int)
	var workers sync.WaitGroup
	workers.Add(config.Workers)
	start := time.Now()
	for worker := 0; worker < config.Workers; worker++ {
		go func() {
			defer workers.Done()
			for index := range tasks {
				candidate := candidates[index%len(candidates)]
				requestStart := time.Now()
				result, err := request(handler, http.MethodGet, "/api/v1/atlas/origins/"+candidate.ID)
				durations[index] = time.Since(requestStart).Microseconds()
				sizes[index] = len(result.body)
				failed[index] = err != nil || result.status != http.StatusOK || !bytes.Equal(result.body, baseline[candidate.ID])
			}
		}()
	}
	for index := 0; index < total; index++ {
		select {
		case <-ctx.Done():
			close(tasks)
			workers.Wait()
			return durations[:index], 0, total - index, time.Since(start)
		case tasks <- index:
		}
	}
	close(tasks)
	workers.Wait()
	wall := time.Since(start)
	responseBytes, failures := 0, 0
	for index := range durations {
		responseBytes += sizes[index]
		if failed[index] {
			failures++
		}
	}
	return durations, responseBytes, failures, wall
}

func request(handler http.Handler, method, target string) (response, error) {
	parsed, err := url.ParseRequestURI(target)
	if err != nil {
		return response{}, err
	}
	request := &http.Request{Method: method, URL: parsed, Header: make(http.Header), Host: "atlas.invalid"}
	writer := &captureWriter{header: make(http.Header), status: http.StatusOK}
	handler.ServeHTTP(writer, request)
	return response{status: writer.status, body: append([]byte(nil), writer.body.Bytes()...)}, nil
}

type captureWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *captureWriter) Header() http.Header { return w.header }

func (w *captureWriter) WriteHeader(status int) { w.status = status }

func (w *captureWriter) Write(data []byte) (int, error) { return w.body.Write(data) }

func responseSetDigest(responses map[string][]byte) string {
	ids := make([]string, 0, len(responses))
	for id := range responses {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	hash := sha256.New()
	for _, id := range ids {
		_, _ = hash.Write([]byte(id))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(responses[id])
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func percentile(sorted []int64, percent int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	index := (len(sorted)*percent + 99) / 100
	if index < 1 {
		index = 1
	}
	return sorted[index-1]
}

func copyCounts(input map[string]int) map[string]int {
	result := make(map[string]int, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
