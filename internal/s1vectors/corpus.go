// Package s1vectors constructs the reviewed E3.3 S1 cross-implementation
// conformance corpus. It is generation/test support, not runtime authority.
package s1vectors

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

type Vector struct {
	Name   string
	Kind   string
	Valid  bool
	Reason string
	Data   []byte
}

func digest(value byte) dataplane.Digest {
	var d dataplane.Digest
	for i := range d {
		d[i] = value
	}
	return d
}

func Corpus() ([]Vector, error) {
	query := dataplane.Query{Version: dataplane.QueryVersion, Select: []string{"statistics:populationTotal"}, Subject: dataplane.QuerySubject{Concept: dataplane.OptionalText{Present: true, Value: "geo:Country"}, IDs: []string{"geo:CL"}}, Time: dataplane.QueryTime{Mode: "current"}, Ontology: dataplane.QueryOntology{MaximumDepth: 1, MaximumPathCostMillionths: 1000000, AllowedEdgeStatuses: []string{"reviewed"}}, Sources: dataplane.QuerySources{AllowedOriginIDs: []string{}, MinimumDistinctOrigins: 1, AllowedAuthorityClasses: []string{}}, Trust: dataplane.QueryTrust{AllowedLanes: []string{"observed_native"}, AllowedMappingStatuses: []string{"none"}}, Freshness: dataplane.QueryFreshness{StaleBehavior: "return_explicit_stale"}, Economics: dataplane.QueryEconomics{AllowedFundingClasses: []string{}}, Conflicts: "preserve_sources", Execution: dataplane.QueryExecution{AllowMaterializedState: true, DeadlineMilliseconds: 5000}, Proof: dataplane.QueryProof{Level: "bundle", IncludePlan: true, IncludeNative: true}, Preference: "highest_proof", Limits: dataplane.QueryLimits{MaximumResults: 10, MaximumPackets: 100, MaximumProofBytes: 65536}}
	queryBytes, err := dataplane.MarshalQuery(query)
	if err != nil {
		return nil, err
	}
	packet := dataplane.Packet{Version: dataplane.PacketVersion, Kind: "measurement", Subject: dataplane.PacketSubject{Native: "world-bank:country/CHL", CanonicalCandidates: []string{"geo:CL"}}, Predicate: dataplane.PacketPredicate{Native: "SP.POP.TOTL"}, Object: dataplane.PacketObject{NativeStatus: "resolved", NativeLexical: "19629590", MediaType: dataplane.OptionalText{Present: true, Value: "application/json"}, Language: dataplane.OptionalText{Present: true, Value: "en"}, Typed: &dataplane.TypedValue{Type: "integer", Lexical: "19629590", Unit: dataplane.OptionalText{Present: true, Value: "unit:person"}}}, Context: dataplane.PacketContext{Dimensions: []dataplane.ContextDimension{}, Jurisdiction: dataplane.OptionalText{Present: true, Value: "geo:CL"}}, Time: dataplane.PacketTime{ObservedAt: "2026-08-11T00:00:00Z"}, Source: dataplane.PacketSource{OriginID: "world-bank-indicators", RepresentationDigest: digest(1), Locator: "/1/0/value"}, Derivation: dataplane.PacketDerivation{ObservationDigest: digest(2), AdapterDigest: digest(3), ExtractionPlanDigest: digest(4), TransformationIDs: []string{"parse:integer"}, MappingIDs: []string{}, CompilerContractDigest: digest(5), CompilerVersion: "twirx-compiler@0.1"}, Epistemic: dataplane.PacketEpistemic{Lane: "observed_native", ExtractionStatus: "deterministic", MappingStatus: "none", AuthorityClass: "official_public_api", FreshnessStatus: "stale"}, Lifecycle: dataplane.PacketLifecycle{State: "current"}, Retention: "public_versioned", Disclosure: "public"}
	batch := dataplane.BatchManifest{Version: dataplane.BatchVersion, OriginID: "world-bank-indicators", CompilerContractDigest: digest(1), PolicyDecisionDigest: digest(2), StartedAt: "2026-08-11T00:00:00Z", CompletedAt: "2026-08-11T00:00:01Z", Observations: []dataplane.Digest{digest(3)}, Packets: []dataplane.DigestSizeEntry{{Digest: digest(4), Size: 400}}, Deltas: []dataplane.DigestSizeEntry{}, RejectionReportDigest: digest(5), MetricsDigest: digest(6), Artifacts: []dataplane.NamedArtifact{{Name: "a.json", Digest: digest(7), Size: 1}, {Name: "b.json", Digest: digest(8), Size: 1}}}
	delta := dataplane.Delta{Version: dataplane.DeltaVersion, Class: "origin", Kind: "added", SemanticKeyDigest: digest(1), AfterPacketDigest: dataplane.OptionalDigest{Present: true, Value: digest(2)}, AfterSourceEvidenceDigest: dataplane.OptionalDigest{Present: true, Value: digest(3)}, OriginID: "world-bank-indicators", OccurredAt: "2026-08-11T00:00:01Z", BatchID: digest(4), CanonVersion: "tw:canon@0.1", ReasonCode: "first_observation"}
	subscription := dataplane.Subscription{Version: dataplane.SubscriptionVersion, QueryDigest: dataplane.DigestBytes(queryBytes), DeltaClasses: []string{"origin"}, DeltaKinds: []string{"added"}, Delivery: "sse", MaximumEventsPerMinute: 60, ProofLevel: "bundle"}
	result := dataplane.QueryResult{Version: dataplane.QueryResultVersion, QueryDigest: dataplane.DigestBytes(queryBytes), PlanDigest: digest(2), Preference: "highest_proof", SnapshotSequence: 1, Status: "resolved", Rows: []dataplane.QueryResultRow{{SubjectID: "world-bank:country/CHL", PredicateID: "SP.POP.TOTL", Status: "resolved", NativeTerm: "SP.POP.TOTL", NativeLocator: "/1/0/value", NativeLexical: "19629590", Typed: &dataplane.TypedValue{Type: "integer", Lexical: "19629590", Unit: dataplane.OptionalText{Present: true, Value: "unit:person"}}, OriginID: "world-bank-indicators", PacketDigest: digest(3), ObservationDigest: digest(4), Lane: "observed_native", ObservedAt: "2026-08-11T00:00:00Z"}}, Conflicts: []dataplane.ConflictGroup{}, ProofArtifacts: []dataplane.DigestSizeEntry{{Digest: digest(5), Size: 400}}, EconomicEventDigest: digest(6), GeneratedAt: "2026-08-11T00:00:02Z"}
	materialization := dataplane.MaterializationManifest{Version: dataplane.MaterializationVersion, MaterializationID: "latest-country-indicators@0.1", DefinitionDigest: digest(1), CanonVersion: "tw:canon@0.1", ThroughSequence: 1, PacketDigests: []dataplane.Digest{digest(2), digest(3)}, ResultArtifactDigest: digest(4), RowCount: 2, BuiltAt: "2026-08-11T00:00:02Z"}
	economic := dataplane.EconomicEvent{Version: dataplane.EconomicEventVersion, EventID: "economic:event/1", OccurredAt: "2026-08-11T00:00:02Z", OriginID: dataplane.OptionalText{Present: true, Value: "world-bank-indicators"}, WorkType: "semantic_query", QueryDigest: dataplane.OptionalDigest{Present: true, Value: dataplane.DigestBytes(queryBytes)}, Resources: dataplane.EconomicResources{Requests: 1, TransferredBytes: 4096, CPUMilliseconds: 2, PeakMemoryBytes: 1048576, ProofBytesReturned: 400}, FundingClass: "public_commons", Cost: dataplane.EconomicMoney{Currency: "USD", Amount: "0.001", Class: "estimated_infrastructure"}, Revenue: dataplane.EconomicMoney{Currency: "USD", Amount: "0", Class: "measured_revenue"}, MeasurementMethod: "tw:economics-method@0.1"}
	snapshot := dataplane.SnapshotManifest{Version: dataplane.SnapshotVersion, Channel: "genesis", CreatedAt: "2026-08-11T00:00:03Z", SourceRevision: "593c4e8e948f0f1c42ccc337898bb53728f28666", CompilerContractDigest: digest(7), CompilerVersion: "twirx-snapshot@0.1", AtlasSelectionDigest: digest(8), CanonModuleSetDigest: digest(9), EvidenceClasses: []string{"archive_observation"}, Artifacts: []dataplane.SnapshotArtifact{{Path: "artifacts/build.json", Digest: digest(1), Size: 1, MediaType: "application/json", Role: "build_report"}, {Path: "artifacts/canon.cbor", Digest: digest(2), Size: 1, MediaType: "application/cbor", Role: "concepts"}, {Path: "artifacts/mappings.cbor", Digest: digest(3), Size: 1, MediaType: "application/cbor", Role: "mappings"}, {Path: "artifacts/origins.json", Digest: digest(4), Size: 1, MediaType: "application/json", Role: "origin_catalog"}, {Path: "artifacts/packets.cbor", Digest: digest(5), Size: 1, MediaType: "application/cbor", Role: "packet_batch"}, {Path: "artifacts/proof.cbor", Digest: digest(6), Size: 1, MediaType: "application/cbor", Role: "proof_index"}}, Views: []dataplane.SnapshotView{}, Counts: dataplane.SnapshotCounts{Origins: 1, Concepts: 1, Mappings: 1, Packets: 1}, HighestPacketSequence: 1, TotalArtifactBytes: 6, BuildReportDigest: digest(1)}

	types := []struct {
		name, kind string
		marshal    func() ([]byte, error)
	}{{"packet-observed-native", dataplane.KindPacket, func() ([]byte, error) { return dataplane.MarshalPacket(packet) }}, {"batch-manifest", dataplane.KindBatch, func() ([]byte, error) { return dataplane.MarshalBatchManifest(batch) }}, {"delta-origin-added", dataplane.KindDelta, func() ([]byte, error) { return dataplane.MarshalDelta(delta) }}, {"semantic-query", dataplane.KindQuery, func() ([]byte, error) { return dataplane.MarshalQuery(query) }}, {"subscription", dataplane.KindSubscription, func() ([]byte, error) { return dataplane.MarshalSubscription(subscription) }}, {"query-result", dataplane.KindQueryResult, func() ([]byte, error) { return dataplane.MarshalQueryResult(result) }}, {"materialization", dataplane.KindMaterialization, func() ([]byte, error) { return dataplane.MarshalMaterializationManifest(materialization) }}, {"economic-event", dataplane.KindEconomicEvent, func() ([]byte, error) { return dataplane.MarshalEconomicEvent(economic) }}, {"semantic-snapshot", dataplane.KindSnapshot, func() ([]byte, error) { return dataplane.MarshalSnapshotManifest(snapshot) }}}
	var vectors []Vector
	validByKind := make(map[string][]byte)
	for _, item := range types {
		data, marshalErr := item.marshal()
		if marshalErr != nil {
			return nil, fmt.Errorf("%s: %w", item.name, marshalErr)
		}
		validByKind[item.kind] = data
		vectors = append(vectors, Vector{Name: item.name, Kind: item.kind, Valid: true, Reason: "canonical valid 0.1 document", Data: data})
		vectors = append(vectors, Vector{Name: item.name + "-trailing", Kind: item.kind, Reason: "trailing byte", Data: append(append([]byte(nil), data...), 0)})
		vectors = append(vectors, Vector{Name: item.name + "-truncated", Kind: item.kind, Reason: "truncated document", Data: append([]byte(nil), data[:len(data)-1]...)})
		wrong := append([]byte(nil), data...)
		versionOffset := bytes.Index(wrong, []byte("tw."))
		if versionOffset < 0 {
			return nil, fmt.Errorf("%s: version not found", item.name)
		}
		wrong[versionOffset] = 'x'
		vectors = append(vectors, Vector{Name: item.name + "-wrong-version", Kind: item.kind, Reason: "unsupported version", Data: wrong})
	}
	typedCases := []struct {
		name, native, value, invalid string
	}{
		{"packet-typed-date", "source date", "2024-02-29", "2023-02-29"},
		{"packet-typed-duration", "source duration", "P1DT2H30M5.5S", "Q1DT2H30M5.5S"},
		{"packet-typed-uri", "source URI", "https://example.org/resource", "1ttps://example.org/resource"},
	}
	for _, typedCase := range typedCases {
		typedPacket := packet
		typedPacket.Object.NativeLexical = typedCase.native
		typeName := "date"
		if strings.Contains(typedCase.name, "duration") {
			typeName = "duration"
		} else if strings.Contains(typedCase.name, "uri") {
			typeName = "uri"
		}
		typedPacket.Object.Typed = &dataplane.TypedValue{Type: typeName, Lexical: typedCase.value}
		data, marshalErr := dataplane.MarshalPacket(typedPacket)
		if marshalErr != nil {
			return nil, marshalErr
		}
		vectors = append(vectors, Vector{Name: typedCase.name, Kind: dataplane.KindPacket, Valid: true, Reason: "valid typed lexical boundary", Data: data})
		vectors = append(vectors, mutated(typedCase.name+"-invalid", dataplane.KindPacket, data, typedCase.value, typedCase.invalid, "invalid typed lexical form"))
	}
	provisional := packet
	provisional.Predicate.Semantic = dataplane.OptionalText{Present: true, Value: "statistics:populationTotal"}
	provisional.Derivation.MappingIDs = []string{"mapping:world-bank-statistics@0.1"}
	provisional.Derivation.SemanticClosureDigest = dataplane.OptionalDigest{Present: true, Value: digest(12)}
	confidence := uint64(800000)
	provisional.Epistemic.Lane = "provisional_semantic"
	provisional.Epistemic.MappingStatus = "candidate"
	provisional.Epistemic.ConfidenceMillionths = &confidence
	provisionalBytes, marshalErr := dataplane.MarshalPacket(provisional)
	if marshalErr != nil {
		return nil, marshalErr
	}
	vectors = append(vectors, Vector{Name: "packet-provisional-semantic", Kind: dataplane.KindPacket, Valid: true, Reason: "candidate mapping remains provisional", Data: provisionalBytes})
	attested := provisional
	attested.Epistemic.Lane = "attested_semantic"
	attested.Epistemic.MappingStatus = "reviewed"
	attested.Epistemic.ConfidenceMillionths = nil
	attestedBytes, marshalErr := dataplane.MarshalPacket(attested)
	if marshalErr != nil {
		return nil, marshalErr
	}
	vectors = append(vectors, Vector{Name: "packet-attested-semantic", Kind: dataplane.KindPacket, Valid: true, Reason: "reviewed mapping with closure evidence", Data: attestedBytes})
	semanticDelta := dataplane.Delta{Version: dataplane.DeltaVersion, Class: "semantic", Kind: "mapped", SemanticKeyDigest: digest(1), BeforePacketDigest: dataplane.OptionalDigest{Present: true, Value: digest(8)}, AfterPacketDigest: dataplane.OptionalDigest{Present: true, Value: digest(9)}, BeforeSourceEvidenceDigest: dataplane.OptionalDigest{Present: true, Value: digest(10)}, AfterSourceEvidenceDigest: dataplane.OptionalDigest{Present: true, Value: digest(10)}, OriginID: "world-bank-indicators", OccurredAt: "2026-08-11T00:00:02Z", BatchID: digest(11), CanonVersion: "tw:canon@0.1", ReasonCode: "mapping_admitted"}
	semanticDeltaBytes, marshalErr := dataplane.MarshalDelta(semanticDelta)
	if marshalErr != nil {
		return nil, marshalErr
	}
	vectors = append(vectors, Vector{Name: "delta-semantic-mapped", Kind: dataplane.KindDelta, Valid: true, Reason: "semantic reinterpretation preserves source evidence", Data: semanticDeltaBytes})
	vectors = append(vectors, mutatedBytes("delta-semantic-changed-evidence", dataplane.KindDelta, semanticDeltaBytes, bytes.Repeat([]byte{10}, 32), bytes.Repeat([]byte{13}, 32), "semantic delta cannot change source evidence"))
	canonDelta := semanticDelta
	canonDelta.Class = "canon"
	canonDelta.Kind = "closure_changed"
	canonDeltaBytes, marshalErr := dataplane.MarshalDelta(canonDelta)
	if marshalErr != nil {
		return nil, marshalErr
	}
	vectors = append(vectors, Vector{Name: "delta-canon-closure", Kind: dataplane.KindDelta, Valid: true, Reason: "canon reinterpretation preserves source evidence", Data: canonDeltaBytes})
	vectors = append(vectors,
		mutated("packet-invalid-mapping", dataplane.KindPacket, validByKind[dataplane.KindPacket], "none", "nope", "invalid observed-native mapping status"),
		mutated("batch-unsorted-artifacts", dataplane.KindBatch, validByKind[dataplane.KindBatch], "a.json", "z.json", "artifact order is not strict"),
		mutated("query-invalid-proof", dataplane.KindQuery, validByKind[dataplane.KindQuery], "bundle", "broken", "unknown proof level"),
		mutated("subscription-invalid-delivery", dataplane.KindSubscription, validByKind[dataplane.KindSubscription], "sse", "xxx", "unknown delivery mode"),
		mutated("result-invalid-lane", dataplane.KindQueryResult, validByKind[dataplane.KindQueryResult], "observed_native", "observed_nativx", "unknown trust lane"),
		mutatedBytes("materialization-duplicate-packet", dataplane.KindMaterialization, validByKind[dataplane.KindMaterialization], bytes.Repeat([]byte{3}, 32), bytes.Repeat([]byte{2}, 32), "duplicate packet digest"),
		mutated("economic-invalid-currency", dataplane.KindEconomicEvent, validByKind[dataplane.KindEconomicEvent], "USD", "UsD", "currency is not uppercase ASCII"),
		mutated("snapshot-unsafe-path", dataplane.KindSnapshot, validByKind[dataplane.KindSnapshot], "artifacts/", "../aaaaaaa", "path traversal segment"),
	)
	indefinite := append([]byte(nil), validByKind[dataplane.KindPacket]...)
	indefinite[0] = 0x9f
	vectors = append(vectors, Vector{Name: "packet-indefinite-array", Kind: dataplane.KindPacket, Reason: "indefinite array", Data: indefinite})
	sort.Slice(vectors, func(i, j int) bool { return vectors[i].Name < vectors[j].Name })
	for i := 1; i < len(vectors); i++ {
		if vectors[i-1].Name == vectors[i].Name {
			return nil, fmt.Errorf("duplicate vector %q", vectors[i].Name)
		}
	}
	return vectors, nil
}

func mutated(name, kind string, data []byte, old, new, reason string) Vector {
	return mutatedBytes(name, kind, data, []byte(old), []byte(new), reason)
}
func mutatedBytes(name, kind string, data, old, new []byte, reason string) Vector {
	if len(old) != len(new) {
		panic("conformance mutation length mismatch")
	}
	out := append([]byte(nil), data...)
	index := bytes.Index(out, old)
	if index < 0 {
		panic("conformance mutation target absent: " + name)
	}
	copy(out[index:index+len(new)], new)
	return Vector{Name: name, Kind: kind, Reason: reason, Data: out}
}
