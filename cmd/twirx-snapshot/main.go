// Command twirx-snapshot builds, verifies, inspects, and serves immutable
// Semantic Snapshots. It never retrieves an origin or executes an action.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/snapshotartifact"
	"github.com/typed-web-commons/typed-web/internal/snapshotbuild"
	"github.com/typed-web-commons/typed-web/internal/snapshotruntime"
)

const maxRequestBytes = 64 << 10

type queryRequest struct {
	Select                 []string `json:"select"`
	SubjectConcept         string   `json:"subject_concept"`
	SubjectIDs             []string `json:"subject_ids"`
	OriginIDs              []string `json:"origin_ids"`
	MinimumDistinctOrigins uint64   `json:"minimum_distinct_origins"`
	AuthorityClasses       []string `json:"authority_classes"`
	Lanes                  []string `json:"lanes"`
	MappingStatuses        []string `json:"mapping_statuses"`
	TimeMode               string   `json:"time_mode"`
	TimeFrom               string   `json:"time_from"`
	TimeUntil              string   `json:"time_until"`
	MaximumAgeSeconds      *uint64  `json:"maximum_age_seconds"`
	StaleBehavior          string   `json:"stale_behavior"`
	Preference             string   `json:"preference"`
	MaximumResults         uint64   `json:"maximum_results"`
	MaximumProofBytes      uint64   `json:"maximum_proof_bytes"`
	ProofLevel             string   `json:"proof_level"`
}

type queryResponse struct {
	SnapshotID          string                  `json:"snapshot_id"`
	Status              string                  `json:"status"`
	QueryDigest         string                  `json:"query_digest"`
	PlanDigest          string                  `json:"plan_digest"`
	ResultDigest        string                  `json:"result_digest"`
	CanonicalQueryCBOR  []byte                  `json:"canonical_query_cbor"`
	CanonicalResultCBOR []byte                  `json:"canonical_result_cbor"`
	Rows                []queryRow              `json:"rows"`
	ProofArtifacts      []proofReference        `json:"proof_artifacts"`
	EconomicEvent       dataplane.EconomicEvent `json:"economic_event"`
	Plan                snapshotruntime.Plan    `json:"plan"`
}

type queryRow struct {
	SubjectID         string                `json:"subject_id"`
	PredicateID       string                `json:"predicate_id"`
	Status            string                `json:"status"`
	NativeTerm        string                `json:"native_term"`
	NativeLocator     string                `json:"native_locator"`
	NativeLexical     string                `json:"native_lexical"`
	SemanticTerm      string                `json:"semantic_term"`
	Typed             *dataplane.TypedValue `json:"typed,omitempty"`
	OriginID          string                `json:"origin_id"`
	PacketDigest      string                `json:"packet_digest"`
	ObservationDigest string                `json:"observation_digest"`
	Lane              string                `json:"lane"`
	ObservedAt        string                `json:"observed_at"`
}

type proofReference struct {
	Digest string `json:"digest"`
	Size   uint64 `json:"size"`
}

type conceptPage struct {
	Offset int      `json:"offset"`
	Limit  int      `json:"limit"`
	Total  int      `json:"total"`
	Items  []string `json:"items"`
}

type deltaPage struct {
	Offset int                                `json:"offset"`
	Limit  int                                `json:"limit"`
	Total  int                                `json:"total"`
	Items  []snapshotruntime.DeltaDescription `json:"items"`
}

type originPage struct {
	Offset int                                 `json:"offset"`
	Limit  int                                 `json:"limit"`
	Total  int                                 `json:"total"`
	Items  []snapshotruntime.OriginDescription `json:"items"`
}

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("usage: twirx-snapshot build|verify|describe|query|trace|serve|stress"))
	}
	var err error
	switch os.Args[1] {
	case "build":
		err = runBuild(os.Args[2:])
	case "verify":
		err = runVerify(os.Args[2:])
	case "describe":
		err = runDescribe(os.Args[2:])
	case "query":
		err = runQuery(os.Args[2:])
	case "trace":
		err = runTrace(os.Args[2:])
	case "serve":
		err = runServe(os.Args[2:])
	case "stress":
		err = runStress(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fail(err)
	}
}

func runBuild(arguments []string) error {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root")
	output := flags.String("out", "", "new snapshot directory")
	revision := flags.String("source-revision", "", "exact source commit")
	createdAt := flags.String("created-at", "", "canonical UTC timestamp")
	scaleFixturePackets := flags.Uint64("scale-fixture-packets", 0, "controlled non-public scale corpus (0..100000)")
	archiveAcquisition := flags.String("archive-acquisition", "", "one committed immutable archive acquisition ID")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	archiveIDs := []string{}
	if *archiveAcquisition != "" {
		archiveIDs = append(archiveIDs, *archiveAcquisition)
	}
	result, err := snapshotbuild.Build(context.Background(), snapshotbuild.Options{Root: *root, Output: *output, SourceRevision: *revision, CreatedAt: *createdAt, ScaleFixturePackets: *scaleFixturePackets, ArchiveAcquisitionIDs: archiveIDs})
	if err != nil {
		return err
	}
	return printJSON(struct {
		SnapshotID string                        `json:"snapshot_id"`
		Output     string                        `json:"output"`
		Actual     snapshotartifact.ActualCounts `json:"actual"`
	}{snapshotartifact.DigestReference(result.SnapshotID), *output, result.Report.Actual})
}

func runVerify(arguments []string) error {
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	directory := flags.String("snapshot", "", "snapshot directory")
	id := flags.String("id", "", "expected sha256 snapshot ID")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	expected, err := optionalDigest(*id)
	if err != nil {
		return err
	}
	runtime, err := snapshotruntime.Open(*directory, snapshotruntime.Options{ExpectedID: expected})
	if err != nil {
		return err
	}
	return printJSON(runtime.Describe())
}

func runDescribe(arguments []string) error {
	runtime, err := openFromFlags("describe", arguments, false)
	if err != nil {
		return err
	}
	return printJSON(runtime.Describe())
}

func runQuery(arguments []string) error {
	flags := flag.NewFlagSet("query", flag.ContinueOnError)
	directory := flags.String("snapshot", "", "snapshot directory")
	id := flags.String("id", "", "expected sha256 snapshot ID")
	file := flags.String("file", "", "bounded JSON query request")
	includeFixtures := flags.Bool("include-fixtures", false, "include controlled fixtures")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	runtime, err := openRuntime(*directory, *id, *includeFixtures)
	if err != nil {
		return err
	}
	request, err := readQueryFile(*file)
	if err != nil {
		return err
	}
	response, err := executeQuery(runtime, request)
	if err != nil {
		return err
	}
	return printJSON(response)
}

func runTrace(arguments []string) error {
	flags := flag.NewFlagSet("trace", flag.ContinueOnError)
	directory := flags.String("snapshot", "", "snapshot directory")
	id := flags.String("id", "", "expected sha256 snapshot ID")
	digest := flags.String("packet", "", "packet digest")
	includeFixtures := flags.Bool("include-fixtures", false, "include controlled fixtures")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	runtime, err := openRuntime(*directory, *id, *includeFixtures)
	if err != nil {
		return err
	}
	trace, err := runtime.Trace(*digest)
	if err != nil {
		return err
	}
	return printJSON(trace)
}

func runServe(arguments []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	directory := flags.String("snapshot", "", "snapshot directory")
	id := flags.String("id", "", "expected sha256 snapshot ID")
	listen := flags.String("listen", "127.0.0.1:8090", "literal loopback listen address")
	includeFixtures := flags.Bool("include-fixtures", false, "local loopback conformance only; never enable at the public edge")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if err := validateLoopback(*listen); err != nil {
		return err
	}
	runtime, err := openRuntime(*directory, *id, *includeFixtures)
	if err != nil {
		return err
	}
	handler := newHandler(runtime)
	server := &http.Server{Addr: *listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	fmt.Fprintf(os.Stderr, "TWIRX immutable snapshot runtime listening on %s\n", *listen)
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func newHandler(runtime *snapshotruntime.Runtime) http.Handler {
	semaphore := make(chan struct{}, 8)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/status", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, runtime.Describe()) })
	mux.HandleFunc("GET /api/v1/origins", func(w http.ResponseWriter, request *http.Request) {
		offset, limit, err := pagination(request)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_pagination"})
			return
		}
		items, total := runtime.OriginPage(offset, limit)
		writeJSON(w, http.StatusOK, originPage{Offset: offset, Limit: limit, Total: total, Items: items})
	})
	mux.HandleFunc("GET /api/v1/origins/{origin_id}", func(w http.ResponseWriter, request *http.Request) {
		description, found := runtime.DescribeOrigin(request.PathValue("origin_id"))
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "origin_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, description)
	})
	mux.HandleFunc("GET /api/v1/concepts", func(w http.ResponseWriter, request *http.Request) {
		offset, limit, err := pagination(request)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_pagination"})
			return
		}
		items, total := runtime.ConceptPage(offset, limit)
		writeJSON(w, http.StatusOK, conceptPage{Offset: offset, Limit: limit, Total: total, Items: items})
	})
	mux.HandleFunc("GET /api/v1/views", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, runtime.DescribeViews()) })
	mux.HandleFunc("GET /api/v1/deltas", func(w http.ResponseWriter, request *http.Request) {
		offset, limit, err := pagination(request)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_pagination"})
			return
		}
		items, total := runtime.DeltaPage(offset, limit)
		writeJSON(w, http.StatusOK, deltaPage{Offset: offset, Limit: limit, Total: total, Items: items})
	})
	mux.HandleFunc("GET /api/v1/deltas/{digest}", func(w http.ResponseWriter, request *http.Request) {
		data, err := runtime.DeltaCBOR("sha256:" + request.PathValue("digest"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "delta_not_found"})
			return
		}
		writeImmutable(w, "application/cbor", "semantic-delta.cbor", data)
	})
	mux.HandleFunc("GET /api/v1/snapshot/manifest.cbor", func(w http.ResponseWriter, _ *http.Request) {
		data, err := runtime.ManifestCBOR()
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "snapshot_unavailable"})
			return
		}
		writeImmutable(w, "application/cbor", "manifest.cbor", data)
	})
	mux.HandleFunc("GET /api/v1/packets/{digest}", func(w http.ResponseWriter, request *http.Request) {
		data, err := runtime.PacketCBOR("sha256:" + request.PathValue("digest"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "packet_not_found"})
			return
		}
		writeImmutable(w, "application/cbor", "semantic-packet.cbor", data)
	})
	mux.HandleFunc("GET /api/v1/proof/{digest}/{artifact}", func(w http.ResponseWriter, request *http.Request) {
		name := request.PathValue("artifact")
		data, mediaType, err := runtime.ProofArtifact("sha256:"+request.PathValue("digest"), name)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "proof_artifact_not_found"})
			return
		}
		writeImmutable(w, mediaType, name, data)
	})
	mux.HandleFunc("GET /api/v1/trace/{digest}", func(w http.ResponseWriter, request *http.Request) {
		digest := request.PathValue("digest")
		trace, err := runtime.Trace("sha256:" + digest)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "packet_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, trace)
	})
	mux.HandleFunc("POST /api/v1/query", func(w http.ResponseWriter, request *http.Request) {
		mediaType, _, mediaErr := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if mediaErr != nil || strings.ToLower(mediaType) != "application/json" {
			writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "application_json_required"})
			return
		}
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
		default:
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "concurrency_limit"})
			return
		}
		data, err := io.ReadAll(http.MaxBytesReader(w, request.Body, maxRequestBytes))
		if err != nil {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request_too_large"})
			return
		}
		query, err := decodeQueryRequest(data)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		response, err := executeQueryContext(request.Context(), runtime, query)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, response)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, request)
	})
}

func executeQuery(runtime *snapshotruntime.Runtime, request queryRequest) (queryResponse, error) {
	return executeQueryContext(context.Background(), runtime, request)
}

func executeQueryContext(ctx context.Context, runtime *snapshotruntime.Runtime, request queryRequest) (queryResponse, error) {
	query, err := request.canonical()
	if err != nil {
		return queryResponse{}, err
	}
	queryBytes, err := dataplane.MarshalQuery(query)
	if err != nil {
		return queryResponse{}, err
	}
	execution, err := runtime.QueryContext(ctx, query)
	if err != nil {
		return queryResponse{}, err
	}
	encoded, err := dataplane.MarshalQueryResult(execution.Result)
	if err != nil {
		return queryResponse{}, err
	}
	description := runtime.Describe()
	response := queryResponse{SnapshotID: description.SnapshotID, Status: execution.Result.Status, QueryDigest: snapshotartifact.DigestReference(execution.Result.QueryDigest), PlanDigest: snapshotartifact.DigestReference(execution.Result.PlanDigest), ResultDigest: snapshotartifact.DigestReference(dataplane.DigestBytes(encoded)), CanonicalQueryCBOR: queryBytes, CanonicalResultCBOR: encoded, EconomicEvent: execution.EconomicEvent, Plan: execution.Plan}
	for _, row := range execution.Result.Rows {
		response.Rows = append(response.Rows, queryRow{SubjectID: row.SubjectID, PredicateID: row.PredicateID, Status: row.Status, NativeTerm: row.NativeTerm, NativeLocator: row.NativeLocator, NativeLexical: row.NativeLexical, SemanticTerm: row.SemanticTerm.Value, Typed: row.Typed, OriginID: row.OriginID, PacketDigest: snapshotartifact.DigestReference(row.PacketDigest), ObservationDigest: snapshotartifact.DigestReference(row.ObservationDigest), Lane: row.Lane, ObservedAt: row.ObservedAt})
	}
	for _, artifact := range execution.Result.ProofArtifacts {
		response.ProofArtifacts = append(response.ProofArtifacts, proofReference{Digest: snapshotartifact.DigestReference(artifact.Digest), Size: artifact.Size})
	}
	return response, nil
}

func (request queryRequest) canonical() (dataplane.Query, error) {
	setDefaults(&request)
	request.Select = sortedUnique(request.Select)
	request.SubjectIDs = sortedUnique(request.SubjectIDs)
	request.OriginIDs = sortedUnique(request.OriginIDs)
	request.AuthorityClasses = sortedUnique(request.AuthorityClasses)
	request.Lanes = sortedUnique(request.Lanes)
	request.MappingStatuses = sortedUnique(request.MappingStatuses)
	query := dataplane.Query{Version: dataplane.QueryVersion, Select: request.Select, Subject: dataplane.QuerySubject{Concept: optionalText(request.SubjectConcept), IDs: request.SubjectIDs}, Time: dataplane.QueryTime{Mode: request.TimeMode, From: optionalText(request.TimeFrom), Until: optionalText(request.TimeUntil)}, Ontology: dataplane.QueryOntology{AllowedEdgeStatuses: []string{"reviewed"}}, Sources: dataplane.QuerySources{AllowedOriginIDs: request.OriginIDs, MinimumDistinctOrigins: request.MinimumDistinctOrigins, AllowedAuthorityClasses: request.AuthorityClasses}, Trust: dataplane.QueryTrust{AllowedLanes: request.Lanes, AllowedMappingStatuses: request.MappingStatuses}, Freshness: dataplane.QueryFreshness{MaximumAgeSeconds: request.MaximumAgeSeconds, StaleBehavior: request.StaleBehavior}, Conflicts: "preserve_sources", Execution: dataplane.QueryExecution{AllowMaterializedState: true, DeadlineMilliseconds: 1000}, Proof: dataplane.QueryProof{Level: request.ProofLevel, IncludePlan: true, IncludeNative: true}, Preference: request.Preference, Limits: dataplane.QueryLimits{MaximumResults: request.MaximumResults, MaximumPackets: request.MaximumResults, MaximumProofBytes: request.MaximumProofBytes}}
	return query, query.Validate()
}

func setDefaults(request *queryRequest) {
	if request.MinimumDistinctOrigins == 0 {
		request.MinimumDistinctOrigins = 1
	}
	if len(request.Lanes) == 0 {
		request.Lanes = []string{"attested_semantic"}
	}
	if len(request.MappingStatuses) == 0 {
		request.MappingStatuses = []string{"reviewed"}
	}
	if request.TimeMode == "" {
		request.TimeMode = "current"
	}
	if request.StaleBehavior == "" {
		request.StaleBehavior = "return_explicit_stale"
	}
	if request.Preference == "" {
		request.Preference = "highest_proof"
	}
	if request.MaximumResults == 0 {
		request.MaximumResults = 100
	}
	if request.MaximumProofBytes == 0 {
		request.MaximumProofBytes = 4 << 20
	}
	if request.ProofLevel == "" {
		request.ProofLevel = "packet"
	}
}

func decodeQueryRequest(data []byte) (queryRequest, error) {
	var request queryRequest
	policy := jsonbounded.Policy{MaxBytes: maxRequestBytes, MaxDepth: 8, MaxScalarBytes: 16 << 10, MaxContainerEntries: 1024, MaxTokens: 10000}
	if err := jsonbounded.Decode(data, &request, policy, true); err != nil {
		return request, err
	}
	return request, nil
}

func readQueryFile(path string) (queryRequest, error) {
	if path == "" {
		return queryRequest{}, errors.New("query file is required")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxRequestBytes {
		return queryRequest{}, errors.New("query file must be a bounded regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return queryRequest{}, err
	}
	return decodeQueryRequest(data)
}

func openFromFlags(name string, arguments []string, includeFixtures bool) (*snapshotruntime.Runtime, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	directory := flags.String("snapshot", "", "snapshot directory")
	id := flags.String("id", "", "expected sha256 snapshot ID")
	if err := flags.Parse(arguments); err != nil {
		return nil, err
	}
	return openRuntime(*directory, *id, includeFixtures)
}

func openRuntime(directory, id string, includeFixtures bool) (*snapshotruntime.Runtime, error) {
	if directory == "" {
		return nil, errors.New("snapshot directory is required")
	}
	expected, err := optionalDigest(id)
	if err != nil {
		return nil, err
	}
	return snapshotruntime.Open(directory, snapshotruntime.Options{ExpectedID: expected, IncludeFixtures: includeFixtures})
}

func optionalDigest(reference string) (dataplane.Digest, error) {
	if reference == "" {
		return dataplane.Digest{}, nil
	}
	return snapshotartifact.ParseDigest(reference)
}

func validateLoopback(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port == "" {
		return errors.New("listen address must be literal loopback IP and port")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("public binding is forbidden; use a literal loopback address behind the edge proxy")
	}
	return nil
}

func optionalText(value string) dataplane.OptionalText {
	return dataplane.OptionalText{Present: value != "", Value: value}
}

func pagination(request *http.Request) (int, int, error) {
	values := request.URL.Query()
	for key, entries := range values {
		if (key != "offset" && key != "limit") || len(entries) != 1 {
			return 0, 0, errors.New("invalid pagination")
		}
	}
	offset, limit := 0, 100
	var err error
	if raw := values.Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 || offset > 10000000 {
			return 0, 0, errors.New("invalid offset")
		}
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 500 {
			return 0, 0, errors.New("invalid limit")
		}
	}
	return offset, limit, nil
}

func sortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	write := 0
	for _, value := range result {
		if write == 0 || result[write-1] != value {
			result[write] = value
			write++
		}
	}
	return result[:write]
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeImmutable(w http.ResponseWriter, mediaType, name string, data []byte) {
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
