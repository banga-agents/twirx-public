package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/snapshotartifact"
)

const maxStressResponseBytes = 8 << 20

type stressReport struct {
	Format       string            `json:"format"`
	Pass         bool              `json:"pass"`
	SnapshotID   string            `json:"snapshot_id"`
	QueryDigest  string            `json:"query_digest"`
	ResultDigest string            `json:"result_digest"`
	Workload     stressWorkload    `json:"workload"`
	Performance  stressPerformance `json:"performance"`
	Errors       []string          `json:"errors"`
}

type stressWorkload struct {
	Requests                     uint64 `json:"requests"`
	Concurrency                  uint64 `json:"concurrency"`
	Successes                    uint64 `json:"successes"`
	Failures                     uint64 `json:"failures"`
	RuntimeOriginNetworkRequests uint64 `json:"runtime_origin_network_requests"`
}

type stressPerformance struct {
	DurationMicroseconds        uint64 `json:"duration_microseconds"`
	RequestsPerSecondMillionths uint64 `json:"requests_per_second_millionths"`
	P50Microseconds             uint64 `json:"p50_microseconds"`
	P95Microseconds             uint64 `json:"p95_microseconds"`
	P99Microseconds             uint64 `json:"p99_microseconds"`
}

type stressOutcome struct {
	LatencyMicroseconds uint64
	SnapshotID          string
	QueryDigest         string
	ResultDigest        string
	OriginRequests      uint64
	Err                 error
}

func runStress(arguments []string) error {
	flags := flag.NewFlagSet("stress", flag.ContinueOnError)
	base := flags.String("base", "http://127.0.0.1:8091", "exact literal-loopback runtime base")
	queryFile := flags.String("query", "", "bounded JSON query request")
	requestCount := flags.Int("requests", 5000, "total requests (1..100000)")
	concurrency := flags.Int("concurrency", 8, "workers (1..64)")
	timeout := flags.Duration("timeout", 10*time.Second, "per-request timeout (1s..60s)")
	out := flags.String("out", "", "optional new/replace atomic JSON report")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 || *requestCount < 1 || *requestCount > 100000 || *concurrency < 1 || *concurrency > 64 || *timeout < time.Second || *timeout > 60*time.Second {
		return errors.New("snapshot stress arguments are outside bounds")
	}
	endpoint, transport, err := loopbackStressTransport(*base)
	if err != nil {
		return err
	}
	request, err := readQueryFile(*queryFile)
	if err != nil {
		return err
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}
	client := &http.Client{Transport: transport, Timeout: *timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("redirects are forbidden") }}
	outcomes := make([]stressOutcome, *requestCount)
	jobs := make(chan int)
	var workers sync.WaitGroup
	for worker := 0; worker < *concurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				outcomes[index] = executeStressRequest(client, endpoint, requestBytes)
			}
		}()
	}
	started := time.Now()
	for index := range outcomes {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	duration := time.Since(started)
	client.CloseIdleConnections()
	report := buildStressReport(outcomes, *concurrency, duration)
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if len(encoded) > 1<<20 {
		return errors.New("snapshot stress report exceeds limit")
	}
	if *out != "" {
		if err := atomicfile.Write(*out, encoded, 1<<20, 0o640); err != nil {
			return err
		}
	}
	if _, err := os.Stdout.Write(encoded); err != nil {
		return err
	}
	if !report.Pass {
		return errors.New("snapshot stress invariants failed")
	}
	return nil
}

func executeStressRequest(client *http.Client, endpoint string, body []byte) stressOutcome {
	started := time.Now()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return stressOutcome{Err: err}
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	latency := uint64(time.Since(started).Microseconds())
	if err != nil {
		return stressOutcome{LatencyMicroseconds: latency, Err: err}
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxStressResponseBytes+1))
	if err != nil || len(data) > maxStressResponseBytes {
		return stressOutcome{LatencyMicroseconds: latency, Err: errors.New("response read or size failure")}
	}
	if response.StatusCode != http.StatusOK {
		return stressOutcome{LatencyMicroseconds: latency, Err: fmt.Errorf("HTTP status %d", response.StatusCode)}
	}
	var decoded queryResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return stressOutcome{LatencyMicroseconds: latency, Err: err}
	}
	if decoded.Status != "resolved" || len(decoded.Rows) == 0 || decoded.Plan.NetworkRequests != 0 || decoded.QueryDigest != snapshotartifact.DigestReference(dataplane.DigestBytes(decoded.CanonicalQueryCBOR)) || decoded.ResultDigest != snapshotartifact.DigestReference(dataplane.DigestBytes(decoded.CanonicalResultCBOR)) {
		return stressOutcome{LatencyMicroseconds: latency, Err: errors.New("response invariants failed")}
	}
	return stressOutcome{LatencyMicroseconds: latency, SnapshotID: decoded.SnapshotID, QueryDigest: decoded.QueryDigest, ResultDigest: decoded.ResultDigest, OriginRequests: decoded.Plan.NetworkRequests}
}

func buildStressReport(outcomes []stressOutcome, concurrency int, duration time.Duration) stressReport {
	report := stressReport{Format: "tw.snapshot-stress-report/0.1", Workload: stressWorkload{Requests: uint64(len(outcomes)), Concurrency: uint64(concurrency)}, Errors: []string{}}
	latencies := make([]uint64, 0, len(outcomes))
	for _, outcome := range outcomes {
		if outcome.Err != nil {
			report.Workload.Failures++
			if len(report.Errors) < 16 {
				report.Errors = append(report.Errors, outcome.Err.Error())
			}
			continue
		}
		if report.SnapshotID == "" {
			report.SnapshotID, report.QueryDigest, report.ResultDigest = outcome.SnapshotID, outcome.QueryDigest, outcome.ResultDigest
		} else if report.SnapshotID != outcome.SnapshotID || report.QueryDigest != outcome.QueryDigest || report.ResultDigest != outcome.ResultDigest {
			report.Workload.Failures++
			if len(report.Errors) < 16 {
				report.Errors = append(report.Errors, "response identity changed during workload")
			}
			continue
		}
		report.Workload.Successes++
		report.Workload.RuntimeOriginNetworkRequests += outcome.OriginRequests
		latencies = append(latencies, outcome.LatencyMicroseconds)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	microseconds := uint64(duration.Microseconds())
	if microseconds == 0 {
		microseconds = 1
	}
	report.Performance.DurationMicroseconds = microseconds
	report.Performance.RequestsPerSecondMillionths = uint64(len(outcomes)) * 1000000 * 1000000 / microseconds
	report.Performance.P50Microseconds = percentile(latencies, 50)
	report.Performance.P95Microseconds = percentile(latencies, 95)
	report.Performance.P99Microseconds = percentile(latencies, 99)
	report.Pass = report.Workload.Failures == 0 && report.Workload.Successes == report.Workload.Requests && report.Workload.RuntimeOriginNetworkRequests == 0 && report.SnapshotID != ""
	return report
}

func percentile(sorted []uint64, percentage int) uint64 {
	if len(sorted) == 0 {
		return 0
	}
	index := (len(sorted)*percentage + 99) / 100
	if index == 0 {
		index = 1
	}
	return sorted[index-1]
}

func loopbackStressTransport(base string) (string, *http.Transport, error) {
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Port() == "" {
		return "", nil, errors.New("stress base must be an exact HTTP literal-loopback origin")
	}
	ip := net.ParseIP(parsed.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return "", nil, errors.New("stress base must use a literal loopback IP")
	}
	pinnedAddress := parsed.Host
	dialer := &net.Dialer{Timeout: 2 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{Proxy: nil, DisableCompression: true, MaxIdleConns: 128, MaxIdleConnsPerHost: 128, MaxConnsPerHost: 128, IdleConnTimeout: 30 * time.Second, DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, pinnedAddress)
	}}
	return parsed.String() + "/api/v1/query", transport, nil
}
