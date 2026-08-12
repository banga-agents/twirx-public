package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/e4capacity"
	"github.com/typed-web-commons/typed-web/internal/universesnapshot"
)

const (
	reportFormat = "tw.e4-controlled-capacity-report/0.1"
	maxReport    = 1 << 20
)

type report struct {
	Format                   string `json:"format"`
	Mode                     string `json:"mode"`
	Operation                string `json:"operation"`
	FrameCount               int    `json:"frame_count"`
	SegmentDigest            string `json:"segment_digest"`
	SegmentBytes             int    `json:"segment_bytes"`
	GenerationNanoseconds    int64  `json:"generation_nanoseconds,omitempty"`
	BuildNanoseconds         int64  `json:"build_nanoseconds,omitempty"`
	VerifiedOpenNanoseconds  int64  `json:"verified_open_nanoseconds,omitempty"`
	Queries                  int    `json:"queries,omitempty"`
	QueryP50Nanoseconds      int64  `json:"query_p50_nanoseconds,omitempty"`
	QueryP95Nanoseconds      int64  `json:"query_p95_nanoseconds,omitempty"`
	QueryP99Nanoseconds      int64  `json:"query_p99_nanoseconds,omitempty"`
	ResidentBytes            uint64 `json:"resident_bytes"`
	ResidentHighWaterBytes   uint64 `json:"resident_high_water_bytes"`
	SwapBytes                uint64 `json:"swap_bytes"`
	NetworkRequests          int    `json:"network_requests"`
	RealSourceRecords        int    `json:"real_source_records"`
	RealSourceDerivedPackets int    `json:"real_source_derived_packets"`
	EvidenceClass            string `json:"evidence_class"`
	Statement                string `json:"statement"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: twirx-e4-capacity <build|open> [flags]")
	}
	switch args[0] {
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "open":
		return runOpen(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func runBuild(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("twirx-e4-capacity build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	frames := fs.Int("frames", 100000, "controlled frame count")
	segmentPath := fs.String("segment", "", "required output segment path")
	reportPath := fs.String("report", "", "optional output report path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *segmentPath == "" {
		return errors.New("build requires --segment and accepts no positional arguments")
	}
	generationStart := time.Now()
	source, err := e4capacity.ControlledFrames(*frames)
	if err != nil {
		return err
	}
	generationElapsed := time.Since(generationStart)
	buildStart := time.Now()
	segment, digest, err := universesnapshot.BuildCompact(source)
	if err != nil {
		return err
	}
	buildElapsed := time.Since(buildStart)
	if err := atomicfile.Write(*segmentPath, segment, universesnapshot.MaxBytes, 0o440); err != nil {
		return err
	}
	rss, high, swap := processMemory()
	return emit(stdout, *reportPath, report{
		Format: reportFormat, Mode: "controlled_test_fixture", Operation: "build", FrameCount: *frames,
		SegmentDigest: digestText(digest), SegmentBytes: len(segment), GenerationNanoseconds: generationElapsed.Nanoseconds(), BuildNanoseconds: buildElapsed.Nanoseconds(),
		ResidentBytes: rss, ResidentHighWaterBytes: high, SwapBytes: swap, EvidenceClass: "test_fixture",
		Statement: "Controlled invented frames measure capacity only; they are not source records, packets, observations, or public corpus evidence.",
	})
}

func runOpen(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("twirx-e4-capacity open", flag.ContinueOnError)
	fs.SetOutput(stderr)
	segmentPath := fs.String("segment", "", "required immutable segment path")
	digestValue := fs.String("digest", "", "required sha256:<hex> segment digest")
	frames := fs.Int("frames", 100000, "expected controlled frame count")
	queries := fs.Int("queries", 1000, "exact queries to measure (1..100000)")
	reportPath := fs.String("report", "", "optional output report path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *segmentPath == "" || *digestValue == "" || *frames < 1 || *frames > universesnapshot.MaxFrames || *queries < 1 || *queries > 100000 {
		return errors.New("open requires bounded --segment, --digest, --frames, and --queries")
	}
	digest, err := parseDigest(*digestValue)
	if err != nil {
		return err
	}
	openStart := time.Now()
	view, err := universesnapshot.OpenCompactFile(*segmentPath, digest)
	if err != nil {
		return err
	}
	defer view.Close()
	openElapsed := time.Since(openStart)
	if view.FrameCount() != uint64(*frames) {
		return fmt.Errorf("frame count mismatch: got %d want %d", view.FrameCount(), *frames)
	}
	durations := make([]int64, *queries)
	for index := 0; index < *queries; index++ {
		countryIndex := (index * 7919) % *frames
		query := universesnapshot.Query{UniverseID: e4capacity.UniverseID, FrameType: e4capacity.FrameType, SlotRole: "world:country", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: e4capacity.Country(countryIndex)}, Limit: 1}
		started := time.Now()
		result, queryErr := view.Query(query)
		durations[index] = time.Since(started).Nanoseconds()
		if queryErr != nil || len(result) != 1 {
			return fmt.Errorf("query %d failed with %d results: %w", index, len(result), queryErr)
		}
	}
	runtime.GC()
	rss, high, swap := processMemory()
	stat, err := os.Stat(*segmentPath)
	if err != nil {
		return err
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	return emit(stdout, *reportPath, report{
		Format: reportFormat, Mode: "controlled_test_fixture", Operation: "verified_open_and_query", FrameCount: *frames,
		SegmentDigest: digestText(digest), SegmentBytes: int(stat.Size()), VerifiedOpenNanoseconds: openElapsed.Nanoseconds(), Queries: *queries,
		QueryP50Nanoseconds: percentile(durations, 50), QueryP95Nanoseconds: percentile(durations, 95), QueryP99Nanoseconds: percentile(durations, 99),
		ResidentBytes: rss, ResidentHighWaterBytes: high, SwapBytes: swap, EvidenceClass: "test_fixture",
		Statement: "The read-only runtime verified one immutable controlled segment and served exact posting-index queries with zero source or network execution.",
	})
}

func emit(stdout io.Writer, path string, value report) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := stdout.Write(encoded); err != nil {
		return err
	}
	if path != "" {
		return atomicfile.Write(path, encoded, maxReport, 0o640)
	}
	return nil
}

func processMemory() (uint64, uint64, uint64) {
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, 0, 0
	}
	defer file.Close()
	var rss, high, swap uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 || fields[2] != "kB" {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSuffix(fields[0], ":") {
		case "VmRSS":
			rss = value * 1024
		case "VmHWM":
			high = value * 1024
		case "VmSwap":
			swap = value * 1024
		}
	}
	return rss, high, swap
}

func percentile(values []int64, percent int) int64 {
	index := (len(values)*percent + 99) / 100
	if index < 1 {
		index = 1
	}
	return values[index-1]
}

func parseDigest(value string) (dataplane.Digest, error) {
	var digest dataplane.Digest
	if !strings.HasPrefix(value, "sha256:") {
		return digest, errors.New("digest must use sha256:<hex>")
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	if err != nil || len(decoded) != len(digest) {
		return digest, errors.New("invalid digest")
	}
	copy(digest[:], decoded)
	return digest, nil
}

func digestText(value dataplane.Digest) string { return "sha256:" + hex.EncodeToString(value[:]) }
