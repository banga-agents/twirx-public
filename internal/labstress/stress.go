// Package labstress drives bounded, read-only TWIRX Lab stress workloads and
// validates the returned typed results and downloaded proof bundles.
package labstress

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/typed-web-commons/typed-web/internal/e2format"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/labengine"
	"github.com/typed-web-commons/typed-web/internal/proofbundle"
)

const (
	WorkloadFormat       = "tw.lab-stress-workload/0.1"
	ReportFormat         = "tw.lab-stress-report/0.1"
	MaxWorkloadBytes     = 256 << 10
	MaxScenarios         = 32
	MaxRequests          = 1000
	MaxConcurrency       = 64
	MaxSimulatedClients  = 1000
	MaxFailureSamples    = 16
	MaxHTTPResponseBytes = 32 << 20
)

type Workload struct {
	Format    string     `json:"format"`
	Scenarios []Scenario `json:"scenarios"`
}

type Scenario struct {
	ID          string         `json:"id"`
	OriginID    string         `json:"origin_id"`
	OperationID string         `json:"operation_id"`
	Input       map[string]any `json:"input"`
	Weight      int            `json:"weight"`
}

type Config struct {
	BaseURL          string
	Requests         int
	Concurrency      int
	SimulatedClients int
	ProofSamples     int
	Timeout          time.Duration
	Workload         Workload
}

type CountSummary struct {
	Attempted       int `json:"attempted"`
	Succeeded       int `json:"succeeded"`
	RateLimited     int `json:"rate_limited"`
	UnexpectedHTTP  int `json:"unexpected_http"`
	InvalidResults  int `json:"invalid_results"`
	TransportErrors int `json:"transport_errors"`
}

type LatencySummary struct {
	Samples int   `json:"samples"`
	MinUS   int64 `json:"min_us"`
	P50US   int64 `json:"p50_us"`
	P95US   int64 `json:"p95_us"`
	P99US   int64 `json:"p99_us"`
	MaxUS   int64 `json:"max_us"`
	MeanUS  int64 `json:"mean_us"`
}

type ByteSummary struct {
	Total int64 `json:"total"`
	Mean  int64 `json:"mean"`
}

type OperationSummary struct {
	ScenarioID  string         `json:"scenario_id"`
	OriginID    string         `json:"origin_id"`
	OperationID string         `json:"operation_id"`
	Counts      CountSummary   `json:"counts"`
	Latency     LatencySummary `json:"latency"`
	Bytes       ByteSummary    `json:"response_bytes"`
}

type ResultEvidence struct {
	ResultID    string `json:"result_id"`
	BundleID    string `json:"bundle_id"`
	OriginID    string `json:"origin_id"`
	OperationID string `json:"operation_id"`
}

type ProofSummary struct {
	Selected          int `json:"selected"`
	ResultViews       int `json:"result_views_verified"`
	ProvenanceViews   int `json:"provenance_views_verified"`
	BundlesDownloaded int `json:"bundles_downloaded"`
	BundlesRehashed   int `json:"bundles_rehashed"`
}

type Report struct {
	Format             string             `json:"format"`
	StartedAt          string             `json:"started_at"`
	CompletedAt        string             `json:"completed_at"`
	TargetClass        string             `json:"target_class"`
	Mode               string             `json:"mode"`
	Statement          string             `json:"statement"`
	Requests           int                `json:"requests"`
	Concurrency        int                `json:"concurrency"`
	SimulatedClients   int                `json:"simulated_clients"`
	PreflightChecks    int                `json:"preflight_checks"`
	InvocationWallUS   int64              `json:"invocation_wall_us"`
	ThroughputMilliRPS int64              `json:"throughput_milli_requests_per_second"`
	FullRunWallUS      int64              `json:"full_run_wall_us"`
	Counts             CountSummary       `json:"counts"`
	Latency            LatencySummary     `json:"latency"`
	Bytes              ByteSummary        `json:"response_bytes"`
	Operations         []OperationSummary `json:"operations"`
	Results            []ResultEvidence   `json:"results"`
	Proof              ProofSummary       `json:"proof"`
	Pass               bool               `json:"pass"`
	FailureSamples     []string           `json:"failure_samples,omitempty"`
}

type target struct {
	base     *url.URL
	loopback bool
	class    string
}

type observation struct {
	index      int
	scenario   int
	status     int
	durationUS int64
	bytes      int64
	view       labengine.ResultView
	err        string
	transport  bool
}

type resultExpectation struct {
	view labengine.ResultView
}

// LoadWorkload loads a regular bounded workload file and rejects ambiguous or
// out-of-vocabulary input before any request is issued.
func LoadWorkload(path string) (Workload, error) {
	var workload Workload
	info, err := os.Lstat(path)
	if err != nil {
		return workload, fmt.Errorf("labstress: inspect workload: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > MaxWorkloadBytes {
		return workload, errors.New("labstress: workload must be a bounded regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return workload, fmt.Errorf("labstress: read workload: %w", err)
	}
	policy := jsonbounded.Policy{MaxBytes: MaxWorkloadBytes, MaxDepth: 12, MaxScalarBytes: 64 << 10, MaxContainerEntries: 1024, MaxTokens: 8192}
	if err := jsonbounded.Decode(data, &workload, policy, true); err != nil {
		return workload, fmt.Errorf("labstress: decode workload: %w", err)
	}
	if err := workload.Validate(); err != nil {
		return workload, err
	}
	return workload, nil
}

func (w Workload) Validate() error {
	if w.Format != WorkloadFormat || len(w.Scenarios) == 0 || len(w.Scenarios) > MaxScenarios {
		return errors.New("labstress: unsupported workload format or invalid scenario count")
	}
	seen := make(map[string]struct{}, len(w.Scenarios))
	totalWeight := 0
	for index, scenario := range w.Scenarios {
		if !safeIdentifier(scenario.ID) || !safeIdentifier(scenario.OriginID) || !safeIdentifier(scenario.OperationID) {
			return fmt.Errorf("labstress: scenario %d has an invalid identifier", index)
		}
		if _, exists := seen[scenario.ID]; exists {
			return fmt.Errorf("labstress: duplicate scenario %q", scenario.ID)
		}
		seen[scenario.ID] = struct{}{}
		if scenario.Weight < 1 || scenario.Weight > MaxRequests {
			return fmt.Errorf("labstress: scenario %q has invalid weight", scenario.ID)
		}
		if len(scenario.Input) > 32 {
			return fmt.Errorf("labstress: scenario %q has too many inputs", scenario.ID)
		}
		for key, value := range scenario.Input {
			if !safeIdentifier(key) || !scalarInput(value) {
				return fmt.Errorf("labstress: scenario %q has invalid input %q", scenario.ID, key)
			}
		}
		encoded, err := json.Marshal(scenario.Input)
		if err != nil || len(encoded) > 64<<10 {
			return fmt.Errorf("labstress: scenario %q input exceeds bounds", scenario.ID)
		}
		totalWeight += scenario.Weight
		if totalWeight > MaxRequests {
			return errors.New("labstress: workload weight exceeds bound")
		}
	}
	return nil
}

// Run invokes the workload and verifies typed responses and a bounded sample
// of every distinct downloaded proof class. It never changes server state
// other than creating the normal content-addressed replay publications.
func Run(ctx context.Context, config Config) (Report, error) {
	started := time.Now().UTC()
	report := Report{
		Format: ReportFormat, StartedAt: started.Format(time.RFC3339Nano), Mode: "replay",
		Statement: "Measures a bounded admitted replay workload; it does not assert origin truth, fresh-origin reliability, or production capacity.",
		Requests:  config.Requests, Concurrency: config.Concurrency, SimulatedClients: config.SimulatedClients,
	}
	resolved, err := validateConfig(config)
	if err != nil {
		return report, err
	}
	report.TargetClass = resolved.class
	client := newClient(config.Concurrency, config.Timeout)
	defer client.CloseIdleConnections()

	for _, path := range []string{"/.well-known/twirx", "/api/v1/status", "/api/v1/origins"} {
		if err := preflight(ctx, client, resolved, path, config.Timeout, config.SimulatedClients > 0); err != nil {
			return report, err
		}
		report.PreflightChecks++
	}

	cycle := workloadCycle(config.Workload)
	invocationsStarted := time.Now()
	jobs := make(chan int)
	completed := make(chan observation, config.Requests)
	var workers sync.WaitGroup
	for worker := 0; worker < config.Concurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				scenarioIndex := cycle[index%len(cycle)]
				completed <- invoke(ctx, client, resolved, config.Workload.Scenarios[scenarioIndex], scenarioIndex, index, config)
			}
		}()
	}
	go func() {
		for index := 0; index < config.Requests; index++ {
			jobs <- index
		}
		close(jobs)
		workers.Wait()
		close(completed)
	}()

	observations := make([]observation, 0, config.Requests)
	for item := range completed {
		observations = append(observations, item)
	}
	report.InvocationWallUS = time.Since(invocationsStarted).Microseconds()
	if report.InvocationWallUS > 0 {
		report.ThroughputMilliRPS = int64(config.Requests) * 1_000_000_000 / report.InvocationWallUS
	}
	sort.Slice(observations, func(i, j int) bool { return observations[i].index < observations[j].index })
	results := summarize(config.Workload, observations, &report)
	report.Pass = report.Counts.Succeeded == config.Requests && report.Counts.RateLimited == 0 && report.Counts.UnexpectedHTTP == 0 && report.Counts.InvalidResults == 0 && report.Counts.TransportErrors == 0

	resultIDs := make([]string, 0, len(results))
	for id := range results {
		resultIDs = append(resultIDs, id)
	}
	sort.Strings(resultIDs)
	if config.ProofSamples < len(resultIDs) {
		resultIDs = resultIDs[:config.ProofSamples]
	}
	report.Proof.Selected = len(resultIDs)
	for _, resultID := range resultIDs {
		expectation := results[resultID]
		evidence, proofErr := verifyPublishedResult(ctx, client, resolved, expectation, config.Timeout, config.SimulatedClients > 0)
		if proofErr != nil {
			report.Pass = false
			addFailure(&report, "proof "+shortID(resultID)+": "+proofErr.Error())
			continue
		}
		report.Results = append(report.Results, evidence)
		report.Proof.ResultViews++
		report.Proof.ProvenanceViews++
		report.Proof.BundlesDownloaded++
		report.Proof.BundlesRehashed++
	}
	if report.Proof.Selected == 0 || report.Proof.BundlesRehashed != report.Proof.Selected {
		report.Pass = false
	}
	report.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	report.FullRunWallUS = time.Since(started).Microseconds()
	return report, nil
}

func validateConfig(config Config) (target, error) {
	var resolved target
	if config.Requests < 1 || config.Requests > MaxRequests || config.Concurrency < 1 || config.Concurrency > MaxConcurrency || config.Concurrency > config.Requests {
		return resolved, errors.New("labstress: requests or concurrency outside bounds")
	}
	if config.SimulatedClients < 0 || config.SimulatedClients > MaxSimulatedClients || config.ProofSamples < 1 || config.ProofSamples > MaxScenarios {
		return resolved, errors.New("labstress: simulated-client or proof-sample count outside bounds")
	}
	if config.Timeout < time.Second || config.Timeout > time.Minute {
		return resolved, errors.New("labstress: timeout outside bounds")
	}
	if err := config.Workload.Validate(); err != nil {
		return resolved, err
	}
	parsed, err := url.Parse(config.BaseURL)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return resolved, errors.New("labstress: target must be a bare admitted Lab base URL")
	}
	hostname := parsed.Hostname()
	if address, parseErr := netip.ParseAddr(hostname); parseErr == nil && address.IsLoopback() {
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return resolved, errors.New("labstress: loopback target must use HTTP or HTTPS")
		}
		resolved = target{base: parsed, loopback: true, class: "literal_loopback"}
	} else if parsed.Scheme == "https" && hostname == "lab.twirx.org" && (parsed.Port() == "" || parsed.Port() == "443") {
		resolved = target{base: parsed, class: "public_lab"}
	} else {
		return resolved, errors.New("labstress: target is neither literal loopback nor https://lab.twirx.org")
	}
	if config.SimulatedClients > 0 && !resolved.loopback {
		return resolved, errors.New("labstress: simulated clients are permitted only for a literal-loopback target")
	}
	return resolved, nil
}

func newClient(concurrency int, timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:                  nil,
		DialContext:            (&net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:      true,
		MaxIdleConns:           concurrency * 2,
		MaxIdleConnsPerHost:    concurrency,
		MaxConnsPerHost:        concurrency,
		IdleConnTimeout:        30 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ResponseHeaderTimeout:  timeout,
		MaxResponseHeaderBytes: 64 << 10,
	}
	return &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return errors.New("labstress: redirects are not accepted")
	}}
}

func preflight(ctx context.Context, client *http.Client, target target, path string, timeout time.Duration, simulate bool) error {
	body, response, err := get(ctx, client, target, path, timeout, simulate, 999)
	if err != nil {
		return fmt.Errorf("labstress: preflight %s: %w", path, err)
	}
	if response.StatusCode != http.StatusOK || !secureResponse(response) || !jsonMediaType(response.Header.Get("Content-Type")) {
		return fmt.Errorf("labstress: preflight %s failed with status %d or unsafe headers", path, response.StatusCode)
	}
	var value any
	policy := jsonbounded.Policy{MaxBytes: MaxHTTPResponseBytes, MaxDepth: 24, MaxScalarBytes: 1 << 20, MaxContainerEntries: 4096, MaxTokens: 32768}
	if err := jsonbounded.Decode(body, &value, policy, false); err != nil {
		return fmt.Errorf("labstress: preflight %s response: %w", path, err)
	}
	return nil
}

func workloadCycle(workload Workload) []int {
	total := 0
	for _, scenario := range workload.Scenarios {
		total += scenario.Weight
	}
	cycle := make([]int, 0, total)
	for index, scenario := range workload.Scenarios {
		for count := 0; count < scenario.Weight; count++ {
			cycle = append(cycle, index)
		}
	}
	return cycle
}

func invoke(ctx context.Context, client *http.Client, target target, scenario Scenario, scenarioIndex, index int, config Config) observation {
	item := observation{index: index, scenario: scenarioIndex}
	payload, err := json.Marshal(map[string]any{"origin_id": scenario.OriginID, "operation_id": scenario.OperationID, "input": scenario.Input, "mode": "replay"})
	if err != nil {
		item.err = "request encoding failed"
		return item
	}
	requestCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint(target, "/api/v1/invoke"), bytes.NewReader(payload))
	if err != nil {
		item.err = "request construction failed"
		return item
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if config.SimulatedClients > 0 {
		req.Header.Set("X-Forwarded-For", simulatedAddress(index%config.SimulatedClients))
	}
	started := time.Now()
	response, err := client.Do(req)
	item.durationUS = time.Since(started).Microseconds()
	if err != nil {
		item.err = boundedError(err)
		item.transport = true
		return item
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, MaxHTTPResponseBytes+1))
	item.status = response.StatusCode
	item.bytes = int64(len(body))
	if err != nil || len(body) > MaxHTTPResponseBytes {
		item.err = "response body read failed or exceeded bound"
		return item
	}
	if !secureResponse(response) || !jsonMediaType(response.Header.Get("Content-Type")) {
		item.err = "response security or media-type invariant failed"
		return item
	}
	if response.StatusCode == http.StatusTooManyRequests {
		if response.Header.Get("Retry-After") == "" {
			item.err = "rate limit omitted Retry-After"
		}
		return item
	}
	if response.StatusCode != http.StatusOK {
		item.err = fmt.Sprintf("unexpected HTTP status %d", response.StatusCode)
		return item
	}
	policy := jsonbounded.Policy{MaxBytes: MaxHTTPResponseBytes, MaxDepth: 32, MaxScalarBytes: 1 << 20, MaxContainerEntries: 4096, MaxTokens: 65536}
	if err := jsonbounded.Decode(body, &item.view, policy, false); err != nil {
		item.err = "typed result response was malformed or ambiguous"
		return item
	}
	if err := validateView(item.view, scenario.OriginID, scenario.OperationID, "replay"); err != nil {
		item.err = err.Error()
		return item
	}
	if response.Header.Get("Location") != "/api/v1/results/"+item.view.ResultID {
		item.err = "result Location header did not bind the returned result ID"
	}
	return item
}

func summarize(workload Workload, observations []observation, report *Report) map[string]resultExpectation {
	perScenario := make([][]observation, len(workload.Scenarios))
	latencies := make([]int64, 0, len(observations))
	results := make(map[string]resultExpectation)
	for _, item := range observations {
		report.Bytes.Total += item.bytes
		latencies = append(latencies, item.durationUS)
		perScenario[item.scenario] = append(perScenario[item.scenario], item)
		classify(item, &report.Counts)
		if item.err != "" {
			addFailure(report, fmt.Sprintf("request %d/%s: %s", item.index, workload.Scenarios[item.scenario].ID, item.err))
		}
		if item.status == http.StatusOK && item.err == "" {
			if previous, exists := results[item.view.ResultID]; exists && (previous.view.OriginID != item.view.OriginID || previous.view.OperationID != item.view.OperationID || previous.view.BundleID != item.view.BundleID) {
				report.Counts.Succeeded--
				report.Counts.InvalidResults++
				addFailure(report, "one result ID referred to inconsistent operation or bundle metadata")
			} else {
				results[item.view.ResultID] = resultExpectation{view: item.view}
			}
		}
	}
	report.Latency = latency(latencies)
	if report.Counts.Attempted > 0 {
		report.Bytes.Mean = report.Bytes.Total / int64(report.Counts.Attempted)
	}
	for index, scenario := range workload.Scenarios {
		items := perScenario[index]
		summary := OperationSummary{ScenarioID: scenario.ID, OriginID: scenario.OriginID, OperationID: scenario.OperationID}
		times := make([]int64, 0, len(items))
		for _, item := range items {
			classify(item, &summary.Counts)
			summary.Bytes.Total += item.bytes
			times = append(times, item.durationUS)
		}
		summary.Latency = latency(times)
		if summary.Counts.Attempted > 0 {
			summary.Bytes.Mean = summary.Bytes.Total / int64(summary.Counts.Attempted)
		}
		report.Operations = append(report.Operations, summary)
	}
	return results
}

func classify(item observation, counts *CountSummary) {
	counts.Attempted++
	if item.transport {
		counts.TransportErrors++
		return
	}
	if item.status == http.StatusTooManyRequests && item.err == "" {
		counts.RateLimited++
		return
	}
	if item.status != http.StatusOK {
		counts.UnexpectedHTTP++
		return
	}
	if item.err != "" {
		counts.InvalidResults++
		return
	}
	counts.Succeeded++
}

func latency(values []int64) LatencySummary {
	if len(values) == 0 {
		return LatencySummary{}
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var sum int64
	for _, value := range sorted {
		sum += value
	}
	return LatencySummary{Samples: len(sorted), MinUS: sorted[0], P50US: percentile(sorted, 50), P95US: percentile(sorted, 95), P99US: percentile(sorted, 99), MaxUS: sorted[len(sorted)-1], MeanUS: sum / int64(len(sorted))}
}

func percentile(sorted []int64, percent int) int64 {
	index := (len(sorted)*percent+99)/100 - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

func validateView(view labengine.ResultView, originID, operationID, mode string) error {
	if view.Format != e2format.ResultVersion || view.ResultID != view.ResultDigest || !validDigest(view.ResultID) || !validDigest(view.BundleID) {
		return errors.New("typed result has invalid format or detached digest identifiers")
	}
	if view.OriginID != originID || view.OperationID != operationID || view.Effect != "read" || view.Mode != mode || (view.Status != "resolved" && view.Status != "partial") {
		return errors.New("typed result identity, mode, effect, or status is invalid")
	}
	for _, digest := range []string{view.Bindings.Input, view.Bindings.Observation, view.Bindings.Transport, view.Bindings.Adapter, view.Bindings.Contract, view.Bindings.SemanticClosure} {
		if !validDigest(digest) {
			return errors.New("typed result contains an invalid upstream binding")
		}
	}
	if len(view.Fields) == 0 {
		return errors.New("typed result contains no fields")
	}
	seen := make(map[string]struct{}, len(view.Fields))
	partial := false
	for _, field := range view.Fields {
		if field.ID == "" || field.Native.Term == "" || field.Native.Locator == "" || field.Semantic.Term == "" || field.Semantic.Type == "" || field.Derivation.Mapping == "" {
			return errors.New("field provenance is incomplete")
		}
		if _, exists := seen[field.ID]; exists {
			return errors.New("typed result contains duplicate fields")
		}
		seen[field.ID] = struct{}{}
		if field.Status == "resolved" {
			if field.Native.Lexical == nil || field.Semantic.Lexical == nil {
				return errors.New("resolved field omitted native or semantic lexical value")
			}
		} else if field.Status == "unresolved" {
			partial = true
			if field.Native.Lexical != nil || field.Semantic.Lexical != nil {
				return errors.New("unresolved field contained a lexical value")
			}
		} else {
			return errors.New("field resolution state is invalid")
		}
	}
	if partial != (view.Status == "partial") {
		return errors.New("result and field resolution states disagree")
	}
	return nil
}

func verifyPublishedResult(ctx context.Context, client *http.Client, target target, expectation resultExpectation, timeout time.Duration, simulate bool) (ResultEvidence, error) {
	view := expectation.view
	base := "/api/v1/results/" + view.ResultID
	resultBody, resultResponse, err := get(ctx, client, target, base, timeout, simulate, 998)
	if err != nil || resultResponse.StatusCode != http.StatusOK || !secureResponse(resultResponse) || !jsonMediaType(resultResponse.Header.Get("Content-Type")) {
		return ResultEvidence{}, errors.New("published result view is unavailable or unsafe")
	}
	var loaded labengine.ResultView
	policy := jsonbounded.Policy{MaxBytes: MaxHTTPResponseBytes, MaxDepth: 32, MaxScalarBytes: 1 << 20, MaxContainerEntries: 4096, MaxTokens: 65536}
	if err := jsonbounded.Decode(resultBody, &loaded, policy, false); err != nil || validateView(loaded, view.OriginID, view.OperationID, "") != nil || loaded.ResultID != view.ResultID || loaded.BundleID != view.BundleID {
		return ResultEvidence{}, errors.New("published result view does not match invocation")
	}

	provenanceBody, provenanceResponse, err := get(ctx, client, target, base+"/provenance", timeout, simulate, 998)
	if err != nil || provenanceResponse.StatusCode != http.StatusOK || !secureResponse(provenanceResponse) || !jsonMediaType(provenanceResponse.Header.Get("Content-Type")) {
		return ResultEvidence{}, errors.New("provenance view is unavailable or unsafe")
	}
	var provenance struct {
		ResultID  string                   `json:"result_id"`
		Bindings  labengine.DigestBindings `json:"bindings"`
		Fields    []labengine.FieldView    `json:"fields"`
		Statement string                   `json:"statement"`
	}
	if err := jsonbounded.Decode(provenanceBody, &provenance, policy, true); err != nil || provenance.ResultID != view.ResultID || len(provenance.Fields) != len(view.Fields) || provenance.Statement == "" || provenance.Bindings != view.Bindings {
		return ResultEvidence{}, errors.New("provenance view does not bind the typed result")
	}

	bundleBody, bundleResponse, err := get(ctx, client, target, base+"/bundle", timeout, simulate, 998)
	if err != nil || bundleResponse.StatusCode != http.StatusOK || !secureResponse(bundleResponse) || bundleResponse.Header.Get("Content-Type") != "application/zip" {
		return ResultEvidence{}, errors.New("proof bundle is unavailable or unsafe")
	}
	if err := verifyArchive(bundleBody, view); err != nil {
		return ResultEvidence{}, err
	}
	return ResultEvidence{ResultID: view.ResultID, BundleID: view.BundleID, OriginID: view.OriginID, OperationID: view.OperationID}, nil
}

func verifyArchive(data []byte, view labengine.ResultView) error {
	if len(data) == 0 || len(data) > MaxHTTPResponseBytes {
		return errors.New("proof archive is empty or exceeds bound")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(archive.File) == 0 || len(archive.File) > proofbundle.MaxArtifacts+1 {
		return errors.New("proof archive structure is invalid")
	}
	files := make(map[string][]byte, len(archive.File))
	for _, file := range archive.File {
		if file.Name == "" || strings.ContainsAny(file.Name, "/\\\x00") || file.UncompressedSize64 == 0 || file.UncompressedSize64 > proofbundle.MaxArtifactBytes {
			return errors.New("proof archive contains an invalid entry")
		}
		if _, exists := files[file.Name]; exists {
			return errors.New("proof archive contains duplicate entries")
		}
		reader, openErr := file.Open()
		if openErr != nil {
			return errors.New("proof archive entry could not be opened")
		}
		body, readErr := io.ReadAll(io.LimitReader(reader, proofbundle.MaxArtifactBytes+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || len(body) == 0 || len(body) > proofbundle.MaxArtifactBytes {
			return errors.New("proof archive entry failed bounded read")
		}
		files[file.Name] = body
	}
	manifestBytes, exists := files[proofbundle.ManifestName]
	if !exists {
		return errors.New("proof archive omitted the final manifest")
	}
	manifest, err := proofbundle.UnmarshalManifest(manifestBytes)
	if err != nil || manifest.ResultID != view.ResultID || len(files) != len(manifest.Entries)+1 {
		return errors.New("proof archive manifest is invalid or incomplete")
	}
	if e2format.DigestReference(sha256.Sum256(manifestBytes)) != view.BundleID {
		return errors.New("proof archive manifest digest does not match bundle ID")
	}
	for _, entry := range manifest.Entries {
		body, found := files[entry.Name]
		if !found || uint64(len(body)) != entry.Size || sha256.Sum256(body) != entry.Digest {
			return fmt.Errorf("proof archive artifact %s failed manifest verification", entry.Name)
		}
	}
	resultBytes := files["result.cbor"]
	result, err := e2format.UnmarshalResult(resultBytes)
	if err != nil || e2format.DigestReference(sha256.Sum256(resultBytes)) != view.ResultID || result.OriginID != view.OriginID || result.OperationID != view.OperationID {
		return errors.New("proof archive canonical result does not match API result")
	}
	return nil
}

func get(ctx context.Context, client *http.Client, target target, path string, timeout time.Duration, simulate bool, simulatedIndex int) ([]byte, *http.Response, error) {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint(target, path), nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json, application/zip")
	if simulate {
		req.Header.Set("X-Forwarded-For", simulatedAddress(simulatedIndex))
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, MaxHTTPResponseBytes+1))
	if err != nil || len(body) > MaxHTTPResponseBytes {
		return nil, response, errors.New("response exceeded bounded read")
	}
	return body, response, nil
}

func endpoint(target target, path string) string {
	resolved := *target.base
	resolved.Path = path
	resolved.RawPath = ""
	resolved.RawQuery = ""
	resolved.Fragment = ""
	return resolved.String()
}

func secureResponse(response *http.Response) bool {
	return response.Header.Get("Content-Security-Policy") != "" && response.Header.Get("Set-Cookie") == "" && response.Header.Get("Access-Control-Allow-Origin") == ""
}

func jsonMediaType(raw string) bool {
	mediaType, _, err := mime.ParseMediaType(raw)
	return err == nil && mediaType == "application/json"
}

func validDigest(value string) bool {
	digest, err := e2format.ParseDigestReference(value)
	return err == nil && hex.EncodeToString(digest[:]) == strings.TrimPrefix(value, "sha256:")
}

func simulatedAddress(index int) string {
	if index >= 0 && index < 250 {
		return fmt.Sprintf("198.51.100.%d", index+1)
	}
	return fmt.Sprintf("2001:db8::%x", index+1)
}

func safeIdentifier(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '.' || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func scalarInput(value any) bool {
	switch value.(type) {
	case string, bool, json.Number:
		return true
	default:
		return false
	}
}

func boundedError(err error) string {
	value := err.Error()
	if len(value) > 256 {
		return value[:256]
	}
	return value
}

func addFailure(report *Report, value string) {
	if len(report.FailureSamples) >= MaxFailureSamples {
		return
	}
	if len(value) > 512 {
		value = value[:512]
	}
	report.FailureSamples = append(report.FailureSamples, value)
}

func shortID(value string) string {
	if len(value) > 20 {
		return value[:20]
	}
	return value
}
