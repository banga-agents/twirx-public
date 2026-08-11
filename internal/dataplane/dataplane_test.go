package dataplane

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func testDigest(value byte) Digest {
	var digest Digest
	for i := range digest {
		digest[i] = value
	}
	return digest
}

func validPacket() Packet {
	return Packet{
		Version:   PacketVersion,
		Kind:      "measurement",
		Subject:   PacketSubject{Native: "world-bank:country/CHL", CanonicalCandidates: []string{"geo:CL"}},
		Predicate: PacketPredicate{Native: "SP.POP.TOTL"},
		Object: PacketObject{
			NativeStatus: "resolved", NativeLexical: "19629590",
			MediaType: OptionalText{Present: true, Value: "application/json"},
			Language:  OptionalText{Present: true, Value: "en"},
			Typed:     &TypedValue{Type: "integer", Lexical: "19629590", Unit: OptionalText{Present: true, Value: "unit:person"}},
		},
		Context:    PacketContext{Dimensions: []ContextDimension{{Key: "statistics:year", Value: TypedValue{Type: "integer", Lexical: "2024"}}}, Jurisdiction: OptionalText{Present: true, Value: "geo:CL"}, Language: OptionalText{Present: true, Value: "en"}},
		Time:       PacketTime{ObservedAt: "2026-08-11T00:00:00Z", ValidFrom: OptionalText{Present: true, Value: "2024-01-01T00:00:00Z"}},
		Source:     PacketSource{OriginID: "world-bank-indicators", RepresentationDigest: testDigest(1), Locator: "/1/0/value", NativeSchemaRef: OptionalText{Present: true, Value: "world-bank:indicator-record@1"}},
		Derivation: PacketDerivation{ObservationDigest: testDigest(2), TransportDigest: OptionalDigest{Present: true, Value: testDigest(3)}, AdapterDigest: testDigest(4), ExtractionPlanDigest: testDigest(5), TransformationIDs: []string{"parse:integer"}, MappingIDs: []string{}, CompilerContractDigest: testDigest(6), CompilerVersion: "twirx-compiler@0.1"},
		Epistemic:  PacketEpistemic{Lane: "observed_native", ExtractionStatus: "deterministic", MappingStatus: "none", AuthorityClass: "official_public_api", FreshnessStatus: "stale"},
		Lifecycle:  PacketLifecycle{State: "current"}, Retention: "public_versioned", Disclosure: "public",
	}
}

func validBatch() BatchManifest {
	return BatchManifest{Version: BatchVersion, OriginID: "world-bank-indicators", CompilerContractDigest: testDigest(1), PolicyDecisionDigest: testDigest(2), StartedAt: "2026-08-11T00:00:00Z", CompletedAt: "2026-08-11T00:00:01Z", Observations: []Digest{testDigest(3)}, Packets: []DigestSizeEntry{{Digest: testDigest(4), Size: 400}}, Deltas: []DigestSizeEntry{{Digest: testDigest(5), Size: 200}}, RejectionReportDigest: testDigest(6), MetricsDigest: testDigest(7), Artifacts: []NamedArtifact{{Name: "metrics.cbor", Digest: testDigest(8), Size: 100}, {Name: "rejections.cbor", Digest: testDigest(9), Size: 100}}}
}

func validDelta() Delta {
	return Delta{Version: DeltaVersion, Class: "origin", Kind: "added", SemanticKeyDigest: testDigest(1), AfterPacketDigest: OptionalDigest{Present: true, Value: testDigest(2)}, AfterSourceEvidenceDigest: OptionalDigest{Present: true, Value: testDigest(3)}, OriginID: "world-bank-indicators", OccurredAt: "2026-08-11T00:00:01Z", BatchID: testDigest(4), CanonVersion: "tw:canon@0.1", ReasonCode: "first_observation"}
}

func validQuery() Query {
	return Query{Version: QueryVersion, Select: []string{"statistics:lifeExpectancy", "statistics:populationTotal"}, Subject: QuerySubject{Concept: OptionalText{Present: true, Value: "geo:Country"}, IDs: []string{"geo:CL", "geo:ID", "geo:VN"}}, Dimensions: []QueryDimension{{Key: "statistics:year", Relation: "eq", Values: []TypedValue{{Type: "integer", Lexical: "2024"}}}}, Time: QueryTime{Mode: "current"}, Ontology: QueryOntology{MaximumDepth: 2, MaximumPathCostMillionths: 2000000, AllowedEdgeStatuses: []string{"candidate", "reviewed"}}, Sources: QuerySources{AllowedOriginIDs: []string{}, MinimumDistinctOrigins: 1, AllowedAuthorityClasses: []string{"official_public_api"}}, Trust: QueryTrust{AllowedLanes: []string{"attested_semantic", "observed_native", "provisional_semantic"}, AllowedMappingStatuses: []string{"candidate", "none", "reviewed"}}, Freshness: QueryFreshness{StaleBehavior: "return_explicit_stale"}, Economics: QueryEconomics{AllowedFundingClasses: []string{}}, Conflicts: "preserve_sources", Execution: QueryExecution{AllowMaterializedState: true, DeadlineMilliseconds: 5000}, Proof: QueryProof{Level: "bundle", IncludePlan: true, IncludeNative: true}, Preference: "highest_proof", Limits: QueryLimits{MaximumResults: 100, MaximumPackets: 1000, MaximumProofBytes: 1048576}}
}

func validSubscription(queryDigest Digest) Subscription {
	return Subscription{Version: SubscriptionVersion, QueryDigest: queryDigest, DeltaClasses: []string{"canon", "origin", "semantic"}, DeltaKinds: []string{"added", "modified"}, Delivery: "sse", MaximumEventsPerMinute: 60, ProofLevel: "bundle", ExpiresAt: OptionalText{Present: true, Value: "2026-09-11T00:00:00Z"}}
}

func validResult(queryDigest Digest) QueryResult {
	return QueryResult{Version: QueryResultVersion, QueryDigest: queryDigest, PlanDigest: testDigest(2), Preference: "highest_proof", SnapshotSequence: 7, Status: "resolved", Rows: []QueryResultRow{{SubjectID: "world-bank:country/CHL", PredicateID: "SP.POP.TOTL", Status: "resolved", NativeTerm: "SP.POP.TOTL", NativeLocator: "/1/0/value", NativeLexical: "19629590", Typed: &TypedValue{Type: "integer", Lexical: "19629590", Unit: OptionalText{Present: true, Value: "unit:person"}}, OriginID: "world-bank-indicators", PacketDigest: testDigest(3), ObservationDigest: testDigest(4), Lane: "observed_native", ObservedAt: "2026-08-11T00:00:00Z"}}, Conflicts: []ConflictGroup{}, ProofArtifacts: []DigestSizeEntry{{Digest: testDigest(5), Size: 400}}, EconomicEventDigest: testDigest(6), GeneratedAt: "2026-08-11T00:00:02Z"}
}

func validMaterialization() MaterializationManifest {
	return MaterializationManifest{Version: MaterializationVersion, MaterializationID: "latest-country-indicators@0.1", DefinitionDigest: testDigest(1), CanonVersion: "tw:canon@0.1", ThroughSequence: 7, PacketDigests: []Digest{testDigest(2), testDigest(3)}, ResultArtifactDigest: testDigest(4), RowCount: 2, BuiltAt: "2026-08-11T00:00:02Z"}
}

func validEconomicEvent() EconomicEvent {
	return EconomicEvent{Version: EconomicEventVersion, EventID: "economic:event/1", OccurredAt: "2026-08-11T00:00:02Z", OriginID: OptionalText{Present: true, Value: "world-bank-indicators"}, WorkType: "semantic_query", QueryDigest: OptionalDigest{Present: true, Value: testDigest(1)}, Resources: EconomicResources{Requests: 1, TransferredBytes: 4096, CPUMilliseconds: 2, PeakMemoryBytes: 1048576, ProofBytesReturned: 400}, FundingClass: "public_commons", Cost: EconomicMoney{Currency: "USD", Amount: "0.001", Class: "estimated_infrastructure"}, Revenue: EconomicMoney{Currency: "USD", Amount: "0", Class: "measured_revenue"}, MeasurementMethod: "tw:economics-method@0.1"}
}

func validSnapshot() SnapshotManifest {
	artifacts := []SnapshotArtifact{
		{Path: "artifacts/build-report.json", Digest: testDigest(1), Size: 1, MediaType: "application/json", Role: "build_report"},
		{Path: "artifacts/canon.cbor", Digest: testDigest(2), Size: 1, MediaType: "application/cbor", Role: "concepts"},
		{Path: "artifacts/mappings.cbor", Digest: testDigest(3), Size: 1, MediaType: "application/cbor", Role: "mappings"},
		{Path: "artifacts/origins.json", Digest: testDigest(4), Size: 1, MediaType: "application/json", Role: "origin_catalog"},
		{Path: "artifacts/packets.cbor", Digest: testDigest(5), Size: 1, MediaType: "application/cbor", Role: "packet_batch"},
		{Path: "artifacts/proof.cbor", Digest: testDigest(6), Size: 1, MediaType: "application/cbor", Role: "proof_index"},
	}
	return SnapshotManifest{Version: SnapshotVersion, Channel: "genesis", CreatedAt: "2026-08-11T00:00:03Z", SourceRevision: "593c4e8e948f0f1c42ccc337898bb53728f28666", CompilerContractDigest: testDigest(7), CompilerVersion: "twirx-snapshot@0.1", AtlasSelectionDigest: testDigest(8), CanonModuleSetDigest: testDigest(9), EvidenceClasses: []string{"archive_observation", "local_fixture"}, Artifacts: artifacts, Views: []SnapshotView{}, Counts: SnapshotCounts{Origins: 1, Concepts: 1, Mappings: 1, Packets: 1}, HighestPacketSequence: 1, TotalArtifactBytes: 6, BuildReportDigest: testDigest(1)}
}

type roundTripCase struct {
	kind      string
	value     any
	marshal   func() ([]byte, error)
	unmarshal func([]byte) (any, error)
}

func validDocuments() []roundTripCase {
	packet := validPacket()
	batch := validBatch()
	delta := validDelta()
	query := validQuery()
	queryBytes, err := MarshalQuery(query)
	if err != nil {
		panic(err)
	}
	subscription := validSubscription(DigestBytes(queryBytes))
	result := validResult(DigestBytes(queryBytes))
	materialization := validMaterialization()
	economic := validEconomicEvent()
	snapshot := validSnapshot()
	return []roundTripCase{
		{KindPacket, packet, func() ([]byte, error) { return MarshalPacket(packet) }, func(b []byte) (any, error) { return UnmarshalPacket(b) }},
		{KindBatch, batch, func() ([]byte, error) { return MarshalBatchManifest(batch) }, func(b []byte) (any, error) { return UnmarshalBatchManifest(b) }},
		{KindDelta, delta, func() ([]byte, error) { return MarshalDelta(delta) }, func(b []byte) (any, error) { return UnmarshalDelta(b) }},
		{KindQuery, query, func() ([]byte, error) { return MarshalQuery(query) }, func(b []byte) (any, error) { return UnmarshalQuery(b) }},
		{KindSubscription, subscription, func() ([]byte, error) { return MarshalSubscription(subscription) }, func(b []byte) (any, error) { return UnmarshalSubscription(b) }},
		{KindQueryResult, result, func() ([]byte, error) { return MarshalQueryResult(result) }, func(b []byte) (any, error) { return UnmarshalQueryResult(b) }},
		{KindMaterialization, materialization, func() ([]byte, error) { return MarshalMaterializationManifest(materialization) }, func(b []byte) (any, error) { return UnmarshalMaterializationManifest(b) }},
		{KindEconomicEvent, economic, func() ([]byte, error) { return MarshalEconomicEvent(economic) }, func(b []byte) (any, error) { return UnmarshalEconomicEvent(b) }},
		{KindSnapshot, snapshot, func() ([]byte, error) { return MarshalSnapshotManifest(snapshot) }, func(b []byte) (any, error) { return UnmarshalSnapshotManifest(b) }},
	}
}

func TestRoundTripEveryDocument(t *testing.T) {
	for _, tc := range validDocuments() {
		t.Run(tc.kind, func(t *testing.T) {
			encoded, err := tc.marshal()
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := tc.unmarshal(encoded)
			if err != nil {
				t.Fatal(err)
			}
			_ = decoded
			if err := ValidateDocument(tc.kind, encoded); err != nil {
				t.Fatal(err)
			}
			withTrailing := append(append([]byte(nil), encoded...), 0)
			if err := ValidateDocument(tc.kind, withTrailing); err == nil {
				t.Fatal("accepted trailing byte")
			}
		})
	}
}

func TestSemanticInvariantsFailClosed(t *testing.T) {
	packet := validPacket()
	packet.Predicate.Semantic = OptionalText{Present: true, Value: "statistics:populationTotal"}
	if _, err := MarshalPacket(packet); err == nil {
		t.Fatal("observed-native semantic promotion accepted")
	}
	delta := validDelta()
	delta.Class = "semantic"
	delta.Kind = "mapped"
	delta.BeforePacketDigest = OptionalDigest{Present: true, Value: testDigest(9)}
	delta.BeforeSourceEvidenceDigest = OptionalDigest{Present: true, Value: testDigest(8)}
	if _, err := MarshalDelta(delta); err == nil {
		t.Fatal("semantic delta with changed source evidence accepted")
	}
	query := validQuery()
	query.Execution.AllowMaterializedState = false
	if _, err := MarshalQuery(query); err == nil {
		t.Fatal("query without execution mode accepted")
	}
	snapshot := validSnapshot()
	snapshot.Artifacts[0].Path = "../escape"
	if _, err := MarshalSnapshotManifest(snapshot); err == nil {
		t.Fatal("unsafe snapshot path accepted")
	}
}

func TestMarshalRejectsDocumentAboveCanonicalLimit(t *testing.T) {
	packet := validPacket()
	large := string(bytes.Repeat([]byte{'a'}, MaxLexical))
	packet.Context.Dimensions = make([]ContextDimension, 32)
	for index := range packet.Context.Dimensions {
		packet.Context.Dimensions[index] = ContextDimension{Key: fmt.Sprintf("dimension:%02d", index), Value: TypedValue{Type: "text", Lexical: large}}
	}
	if err := packet.Validate(); err != nil {
		t.Fatalf("test packet should pass field-level bounds: %v", err)
	}
	if _, err := MarshalPacket(packet); err == nil {
		t.Fatal("accepted canonical document above MaxDocumentBytes")
	}
}

func TestTrustLaneAndDeltaTransitions(t *testing.T) {
	provisional := validPacket()
	provisional.Predicate.Semantic = OptionalText{Present: true, Value: "statistics:populationTotal"}
	provisional.Derivation.MappingIDs = []string{"mapping:world-bank-statistics@0.1"}
	provisional.Derivation.SemanticClosureDigest = OptionalDigest{Present: true, Value: testDigest(9)}
	confidence := uint64(800000)
	provisional.Epistemic.Lane = "provisional_semantic"
	provisional.Epistemic.MappingStatus = "candidate"
	provisional.Epistemic.ConfidenceMillionths = &confidence
	if _, err := MarshalPacket(provisional); err != nil {
		t.Fatalf("valid provisional packet: %v", err)
	}

	attested := provisional
	attested.Epistemic.Lane = "attested_semantic"
	attested.Epistemic.MappingStatus = "reviewed"
	attested.Epistemic.ConfidenceMillionths = nil
	if _, err := MarshalPacket(attested); err != nil {
		t.Fatalf("valid attested packet: %v", err)
	}
	attested.Epistemic.ConfidenceMillionths = &confidence
	if _, err := MarshalPacket(attested); err == nil {
		t.Fatal("attested packet accepted candidate confidence")
	}

	semantic := Delta{Version: DeltaVersion, Class: "semantic", Kind: "mapped", SemanticKeyDigest: testDigest(1), BeforePacketDigest: OptionalDigest{Present: true, Value: testDigest(2)}, AfterPacketDigest: OptionalDigest{Present: true, Value: testDigest(3)}, BeforeSourceEvidenceDigest: OptionalDigest{Present: true, Value: testDigest(4)}, AfterSourceEvidenceDigest: OptionalDigest{Present: true, Value: testDigest(4)}, OriginID: "world-bank-indicators", OccurredAt: "2026-08-11T00:00:01Z", BatchID: testDigest(5), CanonVersion: "tw:canon@0.1", ReasonCode: "mapping_admitted"}
	if _, err := MarshalDelta(semantic); err != nil {
		t.Fatalf("valid semantic delta: %v", err)
	}
	canon := semantic
	canon.Class = "canon"
	canon.Kind = "closure_changed"
	if _, err := MarshalDelta(canon); err != nil {
		t.Fatalf("valid canon delta: %v", err)
	}
	semantic.AfterSourceEvidenceDigest.Value = testDigest(6)
	if _, err := MarshalDelta(semantic); err == nil {
		t.Fatal("semantic delta accepted changed source evidence")
	}
}

func TestVerifySnapshotDirectory(t *testing.T) {
	dir := t.TempDir()
	manifest := validSnapshot()
	for i := range manifest.Artifacts {
		artifact := &manifest.Artifacts[i]
		body := []byte{byte(i + 1)}
		artifact.Digest = DigestBytes(body)
		artifact.Size = uint64(len(body))
		target := filepath.Join(dir, filepath.FromSlash(artifact.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, artifact := range manifest.Artifacts {
		if artifact.Role == "build_report" {
			manifest.BuildReportDigest = artifact.Digest
		}
	}
	encoded, err := MarshalSnapshotManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.cbor"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	expected := DigestBytes(encoded)
	gotManifest, gotID, err := VerifySnapshotDirectory(dir, expected)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := MarshalSnapshotManifest(gotManifest)
	if err != nil {
		t.Fatal(err)
	}
	if gotID != expected || !bytes.Equal(reencoded, encoded) {
		t.Fatal("verified snapshot mismatch")
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(manifest.Artifacts[0].Path)), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := VerifySnapshotDirectory(dir, expected); err == nil {
		t.Fatal("tampered snapshot accepted")
	}
}

func TestVerifySnapshotDirectoryRequiresEveryArtifact(t *testing.T) {
	dir := t.TempDir()
	manifest := validSnapshot()
	for i := range manifest.Artifacts {
		artifact := &manifest.Artifacts[i]
		body := []byte{byte(i + 1)}
		artifact.Digest = DigestBytes(body)
		artifact.Size = 1
		if artifact.Role == "build_report" {
			manifest.BuildReportDigest = artifact.Digest
		}
		target := filepath.Join(dir, filepath.FromSlash(artifact.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if i != len(manifest.Artifacts)-1 {
			if err := os.WriteFile(target, body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	encoded, err := MarshalSnapshotManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.cbor"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := VerifySnapshotDirectory(dir, DigestBytes(encoded)); err == nil {
		t.Fatal("manifest-last verifier accepted a missing constituent")
	}
}

func TestTypedLexicalProfile(t *testing.T) {
	valid := []TypedValue{{Type: "date", Lexical: "2024-02-29"}, {Type: "datetime", Lexical: "2026-08-11T00:00:00Z"}, {Type: "duration", Lexical: "P1DT2H30M5.5S"}, {Type: "uri", Lexical: "https://example.org/resource"}}
	for _, value := range valid {
		if err := value.Validate(); err != nil {
			t.Errorf("valid %s rejected: %v", value.Type, err)
		}
	}
	invalid := []TypedValue{{Type: "date", Lexical: "2023-02-29"}, {Type: "datetime", Lexical: "2026-08-11T00:00:00+01:00"}, {Type: "duration", Lexical: "P1DT"}, {Type: "uri", Lexical: "1ttps://example.org"}}
	for _, value := range invalid {
		if err := value.Validate(); err == nil {
			t.Errorf("invalid %s accepted: %q", value.Type, value.Lexical)
		}
	}
}

func TestSnapshotRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	manifest := validSnapshot()
	for i := range manifest.Artifacts {
		artifact := &manifest.Artifacts[i]
		body := []byte{byte(i + 1)}
		artifact.Digest = DigestBytes(body)
		artifact.Size = 1
		target := filepath.Join(dir, filepath.FromSlash(artifact.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, artifact := range manifest.Artifacts {
		if artifact.Role == "build_report" {
			manifest.BuildReportDigest = artifact.Digest
		}
	}
	encoded, err := MarshalSnapshotManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.cbor"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, filepath.FromSlash(manifest.Artifacts[0].Path))
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("canon.cbor", target); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := VerifySnapshotDirectory(dir, DigestBytes(encoded)); err == nil {
		t.Fatal("symlink artifact accepted")
	}
}

func TestSnapshotRejectsHardLinkedArtifact(t *testing.T) {
	dir := t.TempDir()
	manifest := validSnapshot()
	for i := range manifest.Artifacts {
		artifact := &manifest.Artifacts[i]
		body := []byte{byte(i + 1)}
		artifact.Digest = DigestBytes(body)
		artifact.Size = 1
		target := filepath.Join(dir, filepath.FromSlash(artifact.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, body, 0o644); err != nil {
			t.Fatal(err)
		}
		if artifact.Role == "build_report" {
			manifest.BuildReportDigest = artifact.Digest
		}
	}
	encoded, err := MarshalSnapshotManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.cbor"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, filepath.FromSlash(manifest.Artifacts[0].Path))
	if err := os.Link(target, filepath.Join(dir, "unexpected-hardlink")); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	if _, _, err := VerifySnapshotDirectory(dir, DigestBytes(encoded)); err == nil {
		t.Fatal("hard-linked artifact accepted")
	}
}

func FuzzUnmarshalDataPlane(f *testing.F) {
	for i, tc := range validDocuments() {
		encoded, err := tc.marshal()
		if err == nil {
			f.Add(byte(i), encoded)
		}
	}
	f.Fuzz(func(t *testing.T, selector byte, data []byte) {
		kind := DocumentKinds[int(selector)%len(DocumentKinds)]
		_ = ValidateDocument(kind, data)
	})
}

func BenchmarkPacketRoundTrip(b *testing.B) {
	packet := validPacket()
	encoded, err := MarshalPacket(packet)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(encoded)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoded, err := UnmarshalPacket(encoded)
		if err != nil || decoded.Source.OriginID == "" {
			b.Fatal(err)
		}
	}
}
func BenchmarkQueryRoundTrip(b *testing.B) {
	query := validQuery()
	encoded, err := MarshalQuery(query)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(encoded)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoded, err := UnmarshalQuery(encoded)
		if err != nil || len(decoded.Select) == 0 {
			b.Fatal(err)
		}
	}
}

func TestDocumentKindsAreSorted(t *testing.T) {
	if !sort.StringsAreSorted(DocumentKinds) {
		t.Fatal("document kinds must remain sorted")
	}
}
func TestCanonicalBytesStable(t *testing.T) {
	for _, tc := range validDocuments() {
		a, err := tc.marshal()
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := tc.unmarshal(a)
		if err != nil {
			t.Fatal(err)
		}
		_ = decoded
		b, err := tc.marshal()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("%s encoding changed within process", tc.kind)
		}
	}
}
