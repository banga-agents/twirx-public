package snapshotruntime_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/snapshotartifact"
	"github.com/typed-web-commons/typed-web/internal/snapshotbuild"
	"github.com/typed-web-commons/typed-web/internal/snapshotruntime"
)

func TestRuntimeQueriesAndTracesWithoutFixtures(t *testing.T) {
	runtime := buildRuntime(t, false)
	origins, totalOrigins := runtime.OriginPage(0, 500)
	if totalOrigins != 500 || len(origins) != 500 || origins[0].ID != "data-gov" {
		t.Fatalf("unexpected Atlas page: total=%d first=%+v", totalOrigins, origins[0])
	}
	packetBearing := 0
	for _, origin := range origins {
		if origin.PacketCount > 0 {
			packetBearing++
			if origin.PacketState != "public_packets_available" {
				t.Fatalf("packet-bearing origin has false state: %+v", origin)
			}
		}
	}
	if packetBearing != 2 {
		t.Fatalf("got %d packet-bearing origins, want 2", packetBearing)
	}
	worldBank, found := runtime.DescribeOrigin("api-worldbank-org")
	if !found || worldBank.PacketCount == 0 || worldBank.CatalogState != "candidate" {
		t.Fatalf("unexpected World Bank Atlas description: %+v found=%v", worldBank, found)
	}
	if _, found := runtime.DescribeOrigin("not-admitted"); found {
		t.Fatal("unknown origin entered the immutable Atlas")
	}
	query := populationQuery()
	execution, err := runtime.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	if execution.Result.Status != "resolved" || len(execution.Result.Rows) != 4 || execution.Plan.NetworkRequests != 0 || execution.Plan.ExcludedFixtures != 5 {
		t.Fatalf("unexpected execution: %+v", execution)
	}
	for _, row := range execution.Result.Rows {
		if row.OriginID != "api-worldbank-org" {
			t.Fatalf("unexpected origin %q", row.OriginID)
		}
	}
	reference := "sha256:" + digestHex(execution.Result.Rows[0].PacketDigest)
	trace, err := runtime.Trace(reference)
	if err != nil {
		t.Fatal(err)
	}
	if trace.PacketDigest != reference || trace.EvidenceClass != "recorded_offline_replay" || len(trace.Proof.Artifacts) != 10 {
		t.Fatalf("unexpected trace: %+v", trace)
	}
	narrow := query
	narrow.Select = []string{"development:IndicatorObservation.year"}
	second, err := runtime.Query(narrow)
	if err != nil {
		t.Fatal(err)
	}
	if second.Result.PlanDigest == execution.Result.PlanDigest {
		t.Fatal("plan digest does not bind the canonical query")
	}
}

func TestRuntimeFixtureBoundaryAndLiveRefreshRejection(t *testing.T) {
	withoutFixtures := buildRuntime(t, false)
	query := fixtureQuery()
	execution, err := withoutFixtures.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	if execution.Result.Status != "unresolved" || len(execution.Result.Rows) != 0 {
		t.Fatalf("fixture leaked through public runtime: %+v", execution.Result)
	}
	withFixtures := buildRuntime(t, true)
	execution, err = withFixtures.Query(query)
	if err != nil || execution.Result.Status != "resolved" || len(execution.Result.Rows) != 1 {
		t.Fatalf("explicit fixture query failed: %+v %v", execution.Result, err)
	}
	query.Execution.AllowLiveRefresh = true
	query.Execution.MaximumLiveOrigins = 1
	if _, err := withFixtures.Query(query); !errors.Is(err, snapshotruntime.ErrUnsupportedQuery) {
		t.Fatalf("expected live-refresh rejection, got %v", err)
	}
}

func TestRuntimeScaleFixtureIsSegmentedAndExcludedByDefault(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "snapshot")
	result, err := snapshotbuild.Build(context.Background(), snapshotbuild.Options{Root: filepath.Clean(filepath.Join("..", "..")), Output: directory, SourceRevision: "test-revision", CreatedAt: "2026-08-11T00:00:00Z", ScaleFixturePackets: 4100})
	if err != nil {
		t.Fatal(err)
	}
	if result.Report.Actual.PublicPackets != 13 || result.Report.Actual.FixturePackets != 4105 || result.Report.Actual.ArchiveProfiles != 0 || result.Report.FixtureCountedPublic {
		t.Fatalf("controlled scale corpus changed public evidence counts: %+v", result.Report)
	}
	for _, path := range []string{"packets/segment-000002.json", "proof/index-000002.json"} {
		if _, err := os.Stat(filepath.Join(directory, filepath.FromSlash(path))); err != nil {
			t.Fatalf("missing segmented artifact %s: %v", path, err)
		}
	}
	query := scaleFixtureQuery("field_04099")
	publicRuntime, err := snapshotruntime.Open(directory, snapshotruntime.Options{ExpectedID: result.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	publicResult, err := publicRuntime.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	if publicResult.Result.Status != "unresolved" || len(publicResult.Result.Rows) != 0 || publicResult.Plan.ExcludedFixtures != 4105 {
		t.Fatalf("controlled fixture leaked through the public default: %+v", publicResult)
	}
	fixtureRuntime, err := snapshotruntime.Open(directory, snapshotruntime.Options{ExpectedID: result.SnapshotID, IncludeFixtures: true})
	if err != nil {
		t.Fatal(err)
	}
	fixtureResult, err := fixtureRuntime.Query(query)
	if err != nil || fixtureResult.Result.Status != "resolved" || len(fixtureResult.Result.Rows) != 1 {
		t.Fatalf("explicit controlled scale query failed: %+v %v", fixtureResult, err)
	}
	row := fixtureResult.Result.Rows[0]
	if row.NativeTerm != "field_04099" || row.NativeLexical != "value_04099" || row.Lane != "observed_native" {
		t.Fatalf("unexpected controlled scale result: %+v", row)
	}
	trace, err := fixtureRuntime.Trace(snapshotartifact.DigestReference(row.PacketDigest))
	if err != nil || trace.Proof.ProofType != "controlled_scale_fixture" || trace.Proof.EvidenceClass != "test_fixture" {
		t.Fatalf("unexpected controlled scale trace: %+v %v", trace, err)
	}
}

func TestRuntimeCompilesHistoricalArchivePacketsAndOriginDelta(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	requirePrivateArchiveEvidence(t, repositoryRoot)
	directory := filepath.Join(t.TempDir(), "snapshot")
	result, err := snapshotbuild.Build(context.Background(), snapshotbuild.Options{
		Root:                  repositoryRoot,
		Output:                directory,
		SourceRevision:        "test-revision",
		CreatedAt:             "2026-08-11T12:15:50Z",
		ArchiveAcquisitionIDs: []string{"rfc-editor-futo-history"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Report.Actual.ArchiveProfiles != 1 || result.Report.Actual.PublicOrigins != 3 || result.Report.Actual.PublicPackets != 15 || result.Report.Actual.Deltas != 1 || result.Report.NetworkRequests != 0 || result.Report.CurrentClaimsMade {
		t.Fatalf("unexpected archive build report: %+v", result.Report)
	}
	runtime, err := snapshotruntime.Open(directory, snapshotruntime.Options{ExpectedID: result.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	query := archiveTitleQuery()
	execution, err := runtime.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	if execution.Result.Status != "resolved" || len(execution.Result.Rows) != 2 || execution.Plan.NetworkRequests != 0 {
		t.Fatalf("unexpected archive query: %+v", execution)
	}
	want := []string{" &raquo; RFC Editor", "RFC Editor"}
	for index, row := range execution.Result.Rows {
		if row.NativeLexical != want[index] || row.OriginID != "rfc-editor-org" || row.Lane != "observed_native" || row.SemanticTerm.Present || row.Typed != nil {
			t.Fatalf("archive native statement %d changed: %+v", index, row)
		}
		trace, traceErr := runtime.Trace(snapshotartifact.DigestReference(row.PacketDigest))
		if traceErr != nil || trace.EvidenceClass != "archive_observation" || trace.Proof.ProofType != "archive_capture" {
			t.Fatalf("archive trace %d: %+v %v", index, trace, traceErr)
		}
	}
	deltas, total := runtime.DeltaPage(0, 10)
	if total != 1 || len(deltas) != 1 || deltas[0].Class != "origin" || deltas[0].Kind != "modified" || deltas[0].OriginID != "rfc-editor-org" || deltas[0].ReasonCode != "source_native_title_changed" {
		t.Fatalf("unexpected archive delta: %+v total=%d", deltas, total)
	}
	encoded, err := runtime.DeltaCBOR(deltas[0].Digest)
	deltaDigest, parseErr := snapshotartifact.ParseDigest(deltas[0].Digest)
	if err != nil || parseErr != nil || dataplane.DigestBytes(encoded) != deltaDigest {
		t.Fatalf("delta bytes do not match description: %v", err)
	}
}

func requirePrivateArchiveEvidence(t *testing.T, repositoryRoot string) {
	t.Helper()
	path := filepath.Join(repositoryRoot, "atlas", "archive-acquisitions", "rfc-editor-futo-history", "captures", "capture-000", "representation.body")
	if _, err := os.Stat(path); err == nil {
		return
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect private archive evidence: %v", err)
	}
	if os.Getenv("TWIRX_REQUIRE_PRIVATE_ARCHIVE_EVIDENCE") == "1" {
		t.Fatalf("required private archive evidence is absent: %s", path)
	}
	t.Skip("private third-party archive bytes are excluded from the public source profile")
}

func TestRuntimeRejectsArtifactTampering(t *testing.T) {
	directory, id := buildSnapshot(t)
	path := filepath.Join(directory, "packets", "segment-000001.json")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("\n")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotruntime.Open(directory, snapshotruntime.Options{ExpectedID: id}); err == nil {
		t.Fatal("expected tampered artifact rejection")
	}
}

func TestRuntimeRejectsSelfConsistentButFalseProofIndex(t *testing.T) {
	directory, _ := buildSnapshot(t)
	indexPath := filepath.Join(directory, "proof", "index.json")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	var index snapshotartifact.ProofIndex
	if err := snapshotartifact.Decode(indexBytes, &index); err != nil {
		t.Fatal(err)
	}
	index.Entries[0].FieldID = strings.Repeat("x", len(index.Entries[0].FieldID))
	indexBytes, err = snapshotartifact.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, indexBytes, 0o640); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(directory, "manifest.cbor")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := dataplane.UnmarshalSnapshotManifest(manifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	for artifactIndex := range manifest.Artifacts {
		if manifest.Artifacts[artifactIndex].Path == "proof/index.json" {
			oldSize := manifest.Artifacts[artifactIndex].Size
			manifest.Artifacts[artifactIndex].Digest = dataplane.DigestBytes(indexBytes)
			manifest.Artifacts[artifactIndex].Size = uint64(len(indexBytes))
			manifest.TotalArtifactBytes = manifest.TotalArtifactBytes - oldSize + uint64(len(indexBytes))
		}
	}
	manifestBytes, err = dataplane.MarshalSnapshotManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, _, err := dataplane.VerifySnapshotDirectory(directory, dataplane.Digest{}); err != nil {
		t.Fatalf("cryptographically self-consistent fixture should reach semantic reconciliation: %v", err)
	}
	if _, err := snapshotruntime.Open(directory, snapshotruntime.Options{}); err == nil {
		t.Fatal("expected false proof-index relationship rejection")
	}
}

func BenchmarkMaterializedQuery(b *testing.B) {
	directory, id := buildSnapshot(b)
	runtime, err := snapshotruntime.Open(directory, snapshotruntime.Options{ExpectedID: id})
	if err != nil {
		b.Fatal(err)
	}
	query := populationQuery()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := runtime.Query(query); err != nil {
			b.Fatal(err)
		}
	}
}

func buildRuntime(t testing.TB, includeFixtures bool) *snapshotruntime.Runtime {
	t.Helper()
	directory, id := buildSnapshot(t)
	runtime, err := snapshotruntime.Open(directory, snapshotruntime.Options{ExpectedID: id, IncludeFixtures: includeFixtures})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func buildSnapshot(t testing.TB) (string, dataplane.Digest) {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "snapshot")
	result, err := snapshotbuild.Build(context.Background(), snapshotbuild.Options{Root: filepath.Clean(filepath.Join("..", "..")), Output: directory, SourceRevision: "test-revision", CreatedAt: "2026-08-11T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	return directory, result.SnapshotID
}

func populationQuery() dataplane.Query {
	return dataplane.Query{Version: dataplane.QueryVersion, Select: []string{"development:Indicator.identifier", "development:IndicatorObservation.countryCode", "development:IndicatorObservation.value", "development:IndicatorObservation.year"}, Subject: dataplane.QuerySubject{Concept: dataplane.OptionalText{Present: true, Value: "development:IndicatorObservation"}}, Time: dataplane.QueryTime{Mode: "current"}, Ontology: dataplane.QueryOntology{AllowedEdgeStatuses: []string{"reviewed"}}, Sources: dataplane.QuerySources{AllowedOriginIDs: []string{"api-worldbank-org"}, MinimumDistinctOrigins: 1}, Trust: dataplane.QueryTrust{AllowedLanes: []string{"attested_semantic"}, AllowedMappingStatuses: []string{"reviewed"}}, Freshness: dataplane.QueryFreshness{StaleBehavior: "return_explicit_stale"}, Conflicts: "preserve_sources", Execution: dataplane.QueryExecution{AllowMaterializedState: true, DeadlineMilliseconds: 1000}, Proof: dataplane.QueryProof{Level: "packet", IncludePlan: true, IncludeNative: true}, Preference: "highest_proof", Limits: dataplane.QueryLimits{MaximumResults: 16, MaximumPackets: 16, MaximumProofBytes: 1 << 20}}
}

func archiveTitleQuery() dataplane.Query {
	return dataplane.Query{Version: dataplane.QueryVersion, Select: []string{"html:title"}, Subject: dataplane.QuerySubject{IDs: []string{"rfc-editor:homepage"}}, Time: dataplane.QueryTime{Mode: "history"}, Ontology: dataplane.QueryOntology{AllowedEdgeStatuses: []string{"reviewed"}}, Sources: dataplane.QuerySources{AllowedOriginIDs: []string{"rfc-editor-org"}, MinimumDistinctOrigins: 1, AllowedAuthorityClasses: []string{"common_crawl_archive_observation"}}, Trust: dataplane.QueryTrust{AllowedLanes: []string{"observed_native"}, AllowedMappingStatuses: []string{"none"}}, Freshness: dataplane.QueryFreshness{StaleBehavior: "return_explicit_stale"}, Conflicts: "preserve_sources", Execution: dataplane.QueryExecution{AllowMaterializedState: true, DeadlineMilliseconds: 1000}, Proof: dataplane.QueryProof{Level: "packet", IncludePlan: true, IncludeNative: true}, Preference: "highest_proof", Limits: dataplane.QueryLimits{MaximumResults: 10, MaximumPackets: 10, MaximumProofBytes: 4 << 20}}
}

func fixtureQuery() dataplane.Query {
	query := populationQuery()
	query.Select = []string{"commerce:Product.identifier"}
	query.Subject = dataplane.QuerySubject{Concept: dataplane.OptionalText{Present: true, Value: "commerce:Offer"}}
	query.Sources.AllowedOriginIDs = []string{"controlled-origin-lab-fixture"}
	return query
}

func scaleFixtureQuery(fieldID string) dataplane.Query {
	return dataplane.Query{Version: dataplane.QueryVersion, Select: []string{fieldID}, Subject: dataplane.QuerySubject{IDs: []string{"fixture:scale-corpus/document/1"}}, Time: dataplane.QueryTime{Mode: "current"}, Ontology: dataplane.QueryOntology{AllowedEdgeStatuses: []string{"reviewed"}}, Sources: dataplane.QuerySources{AllowedOriginIDs: []string{"controlled-scale-corpus-fixture"}, MinimumDistinctOrigins: 1, AllowedAuthorityClasses: []string{"controlled_test_fixture"}}, Trust: dataplane.QueryTrust{AllowedLanes: []string{"observed_native"}, AllowedMappingStatuses: []string{"none"}}, Freshness: dataplane.QueryFreshness{StaleBehavior: "return_explicit_stale"}, Conflicts: "preserve_sources", Execution: dataplane.QueryExecution{AllowMaterializedState: true, DeadlineMilliseconds: 1000}, Proof: dataplane.QueryProof{Level: "packet", IncludePlan: true, IncludeNative: true}, Preference: "highest_proof", Limits: dataplane.QueryLimits{MaximumResults: 1, MaximumPackets: 1, MaximumProofBytes: 1 << 20}}
}

func digestHex(digest dataplane.Digest) string {
	const digits = "0123456789abcdef"
	result := make([]byte, len(digest)*2)
	for index, value := range digest {
		result[index*2] = digits[value>>4]
		result[index*2+1] = digits[value&15]
	}
	return string(result)
}
