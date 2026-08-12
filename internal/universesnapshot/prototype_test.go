package universesnapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

func TestPrototypeLayoutsRestoreAndAgree(t *testing.T) {
	source := syntheticFrames(t, 100)
	nativeBytes, nativeDigest, err := BuildNative(source)
	if err != nil {
		t.Fatal(err)
	}
	repeated, repeatedDigest, err := BuildNative(source)
	if err != nil || !bytes.Equal(nativeBytes, repeated) || nativeDigest != repeatedDigest {
		t.Fatal("native build is not deterministic")
	}
	columnBytes, columnDigest, err := BuildColumnar(source)
	if err != nil {
		t.Fatal(err)
	}
	native, err := OpenNative(nativeBytes, nativeDigest)
	if err != nil {
		t.Fatalf("open native: %v", err)
	}
	columnar, err := OpenColumnar(columnBytes, columnDigest)
	if err != nil {
		t.Fatalf("open columnar: %v", err)
	}
	compactBytes, compactDigest, err := BuildCompact(source)
	if err != nil {
		t.Fatalf("build compact: %v", err)
	}
	compact, err := OpenCompact(compactBytes, compactDigest)
	if err != nil {
		t.Fatalf("open compact: %v", err)
	}
	query := Query{UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", SlotRole: "world:country", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: "geo:SYN000042"}, Limit: 10}
	nativeResult, err := native.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	columnResult, err := columnar.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	if len(nativeResult) != 1 || len(columnResult) != 1 || nativeResult[0] != columnResult[0] {
		t.Fatalf("query disagreement: native=%x columnar=%x", nativeResult, columnResult)
	}
	for _, runtime := range []interface {
		Trace(dataplane.Digest) ([]byte, error)
	}{native, columnar, compact} {
		trace, err := runtime.Trace(nativeResult[0])
		if err != nil || dataplane.DigestBytes(trace) != nativeResult[0] {
			t.Fatalf("trace did not preserve canonical frame: %v", err)
		}
	}

	nativeByKey, err := native.Query(Query{NativeKey: "world-bank:synthetic/000042", Limit: 1})
	if err != nil || len(nativeByKey) != 1 || nativeByKey[0] != nativeResult[0] {
		t.Fatalf("native-key query = %x, %v", nativeByKey, err)
	}
	compactResult, err := compact.Query(query)
	if err != nil || len(compactResult) != 1 || compactResult[0] != nativeResult[0] {
		t.Fatalf("compact query disagreement: %x, %v", compactResult, err)
	}
}

func TestCompactFileMmapAndTamperRejection(t *testing.T) {
	data, digest, err := BuildCompact(syntheticFrames(t, 100))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "frames.twux")
	if err := os.WriteFile(path, data, 0o440); err != nil {
		t.Fatal(err)
	}
	runtime, err := OpenCompactFile(path, digest)
	if err != nil {
		t.Fatal(err)
	}
	if !runtime.mapped || runtime.FrameCount() != 100 {
		t.Fatal("compact file did not open as a mapped runtime")
	}
	visited := make([]dataplane.Digest, 0, runtime.FrameCount())
	if err := runtime.VisitFrames(func(digest dataplane.Digest, body []byte) error {
		if dataplane.DigestBytes(body) != digest {
			t.Fatal("visitor returned a body that does not match its digest")
		}
		visited = append(visited, digest)
		body[0] ^= 1
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(visited) != 100 || !sort.SliceIsSorted(visited, func(i, j int) bool {
		return bytes.Compare(visited[i][:], visited[j][:]) < 0
	}) {
		t.Fatal("visitor did not cover frames in canonical digest order")
	}
	if traced, err := runtime.Trace(visited[0]); err != nil || dataplane.DigestBytes(traced) != visited[0] {
		t.Fatal("visitor was able to mutate the mapped frame")
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.VisitFrames(func(dataplane.Digest, []byte) error { return nil }); err == nil {
		t.Fatal("closed compact runtime accepted a frame walk")
	}
	if _, err := runtime.Query(Query{NativeKey: "world-bank:synthetic/000001", Limit: 1}); err == nil {
		t.Fatal("closed compact runtime accepted a query")
	}
	corrupt := append([]byte(nil), data...)
	corrupt[len(corrupt)-1] ^= 1
	if _, err := OpenCompact(corrupt, digest); err == nil {
		t.Fatal("accepted compact segment with wrong whole-file digest")
	}
	bogusCount := append([]byte(nil), data...)
	binary.BigEndian.PutUint32(bogusCount[16:20], ^uint32(0))
	if _, err := OpenCompact(bogusCount, dataplane.DigestBytes(bogusCount)); err == nil {
		t.Fatal("accepted posting count larger than its segment")
	}
}

func TestPrototypeLayoutsFailClosed(t *testing.T) {
	source := syntheticFrames(t, 4)
	nativeBytes, nativeDigest, err := BuildNative(source)
	if err != nil {
		t.Fatal(err)
	}
	columnBytes, columnDigest, err := BuildColumnar(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenNative(append(append([]byte(nil), nativeBytes...), 'x'), nativeDigest); err == nil {
		t.Fatal("native accepted digest mismatch")
	}
	if _, err := OpenColumnar(append(append([]byte(nil), columnBytes...), 'x'), columnDigest); err == nil {
		t.Fatal("columnar accepted digest mismatch")
	}

	var nativeArtifact NativeArtifact
	if err := json.Unmarshal(nativeBytes, &nativeArtifact); err != nil {
		t.Fatal(err)
	}
	nativeArtifact.Postings = nativeArtifact.Postings[1:]
	tamperedNative, _ := marshal(nativeArtifact)
	if _, err := OpenNative(tamperedNative, dataplane.DigestBytes(tamperedNative)); err == nil {
		t.Fatal("native accepted missing posting")
	}

	var columnArtifact ColumnarArtifact
	if err := json.Unmarshal(columnBytes, &columnArtifact); err != nil {
		t.Fatal(err)
	}
	columnArtifact.NativeKeys[0] = "tampered"
	tamperedColumn, _ := marshal(columnArtifact)
	if _, err := OpenColumnar(tamperedColumn, dataplane.DigestBytes(tamperedColumn)); err == nil {
		t.Fatal("columnar accepted non-reconciling columns")
	}

	native, err := OpenNative(nativeBytes, nativeDigest)
	if err != nil {
		t.Fatal(err)
	}
	invalidQueries := []Query{
		{},
		{UniverseID: "tw:world-state"},
		{UniverseID: "tw:world-state", Limit: 1001},
		{SlotRole: "world:country", Limit: 1},
		{SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: "geo:SYN000001"}, Limit: 1},
	}
	for index, query := range invalidQueries {
		if _, err := native.Query(query); err == nil {
			t.Fatalf("invalid query %d accepted", index)
		}
	}
}

func TestPrototypeRejectsDuplicateFrames(t *testing.T) {
	source := syntheticFrames(t, 1)
	source = append(source, source[0])
	if _, _, err := BuildNative(source); err == nil {
		t.Fatal("native accepted duplicate frame")
	}
	if _, _, err := BuildColumnar(source); err == nil {
		t.Fatal("columnar accepted duplicate frame")
	}
}

func FuzzOpenNative(f *testing.F) {
	source := syntheticFrames(f, 2)
	data, _, err := BuildNative(source)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(data)
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		runtime, err := OpenNative(input, dataplane.DigestBytes(input))
		if err == nil && runtime.FrameCount() == 0 {
			t.Fatal("accepted empty native runtime")
		}
	})
}

func FuzzOpenColumnar(f *testing.F) {
	source := syntheticFrames(f, 2)
	data, _, err := BuildColumnar(source)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(data)
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		runtime, err := OpenColumnar(input, dataplane.DigestBytes(input))
		if err == nil && runtime.FrameCount() == 0 {
			t.Fatal("accepted empty columnar runtime")
		}
	})
}

func FuzzOpenCompact(f *testing.F) {
	data, _, err := BuildCompact(syntheticFrames(f, 2))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(data)
	f.Add([]byte(compactMagic))
	f.Fuzz(func(t *testing.T, input []byte) {
		runtime, err := OpenCompact(input, dataplane.DigestBytes(input))
		if err == nil && runtime.FrameCount() == 0 {
			t.Fatal("accepted empty compact runtime")
		}
	})
}

func BenchmarkNativeExactSlotQuery(b *testing.B) {
	source := syntheticFrames(b, 10000)
	data, digest, err := BuildNative(source)
	if err != nil {
		b.Fatal(err)
	}
	runtime, err := OpenNative(data, digest)
	if err != nil {
		b.Fatal(err)
	}
	query := Query{UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", SlotRole: "world:country", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: "geo:SYN005000"}, Limit: 10}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if result, err := runtime.Query(query); err != nil || len(result) != 1 {
			b.Fatal(err)
		}
	}
}

func BenchmarkColumnarExactSlotQuery(b *testing.B) {
	source := syntheticFrames(b, 10000)
	data, digest, err := BuildColumnar(source)
	if err != nil {
		b.Fatal(err)
	}
	runtime, err := OpenColumnar(data, digest)
	if err != nil {
		b.Fatal(err)
	}
	query := Query{UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", SlotRole: "world:country", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: "geo:SYN005000"}, Limit: 10}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if result, err := runtime.Query(query); err != nil || len(result) != 1 {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildNative10000(b *testing.B) {
	source := syntheticFrames(b, 10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _, err := BuildNative(source)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(data)), "release-B")
	}
}

func BenchmarkBuildColumnar10000(b *testing.B) {
	source := syntheticFrames(b, 10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _, err := BuildColumnar(source)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(data)), "release-B")
	}
}

func BenchmarkOpenNative10000(b *testing.B) {
	data, digest, err := BuildNative(syntheticFrames(b, 10000))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(len(data)), "release-B")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := OpenNative(data, digest); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOpenColumnar10000(b *testing.B) {
	data, digest, err := BuildColumnar(syntheticFrames(b, 10000))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(len(data)), "release-B")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := OpenColumnar(data, digest); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildCompact10000(b *testing.B) {
	source := syntheticFrames(b, 10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _, err := BuildCompact(source)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(data)), "release-B")
	}
}

func BenchmarkOpenCompact10000(b *testing.B) {
	data, digest, err := BuildCompact(syntheticFrames(b, 10000))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(len(data)), "release-B")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := OpenCompact(data, digest); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompactExactSlotQuery(b *testing.B) {
	data, digest, err := BuildCompact(syntheticFrames(b, 10000))
	if err != nil {
		b.Fatal(err)
	}
	runtime, err := OpenCompact(data, digest)
	if err != nil {
		b.Fatal(err)
	}
	query := Query{UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", SlotRole: "world:country", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: "geo:SYN005000"}, Limit: 10}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if result, err := runtime.Query(query); err != nil || len(result) != 1 {
			b.Fatal(err)
		}
	}
}

func syntheticFrames(tb testing.TB, count int) []SourceFrame {
	tb.Helper()
	result := make([]SourceFrame, count)
	for index := 0; index < count; index++ {
		packetDigests := []dataplane.Digest{
			testDigest(fmt.Sprintf("country/%d", index)),
			testDigest(fmt.Sprintf("indicator/%d", index)),
			testDigest(fmt.Sprintf("value/%d", index)),
			testDigest(fmt.Sprintf("period/%d", index)),
		}
		sortDigests(packetDigests)
		byLabel := map[string]dataplane.Digest{
			"country":   testDigest(fmt.Sprintf("country/%d", index)),
			"indicator": testDigest(fmt.Sprintf("indicator/%d", index)),
			"value":     testDigest(fmt.Sprintf("value/%d", index)),
			"period":    testDigest(fmt.Sprintf("period/%d", index)),
		}
		country := fmt.Sprintf("geo:SYN%06d", index)
		frame := dataplane.Frame{
			Version: dataplane.FrameVersion, UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", NativeKey: fmt.Sprintf("world-bank:synthetic/%06d", index), CanonicalCandidates: []string{fmt.Sprintf("world:synthetic/%06d", index)},
			Slots: []dataplane.FrameSlot{
				{RoleID: "world:country", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "identifier", Lexical: country}}, PacketDigests: []dataplane.Digest{byLabel["country"]}, MappingIDs: []string{"mapping:test/country@0.1"}, Conflict: "none"},
				{RoleID: "world:indicator", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "identifier", Lexical: "TEST.INDICATOR"}}, PacketDigests: []dataplane.Digest{byLabel["indicator"]}, MappingIDs: []string{"mapping:test/indicator@0.1"}, Conflict: "none"},
				{RoleID: "world:observedValue", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "decimal", Lexical: fmt.Sprintf("%d.0", index)}}, PacketDigests: []dataplane.Digest{byLabel["value"]}, MappingIDs: []string{"mapping:test/value@0.1"}, Conflict: "none"},
				{RoleID: "world:period", Status: "resolved", Cardinality: "one", Values: []dataplane.TypedValue{{Type: "date", Lexical: "2026-01-01"}}, PacketDigests: []dataplane.Digest{byLabel["period"]}, MappingIDs: []string{"mapping:test/period@0.1"}, Conflict: "none"},
			},
			Time:       dataplane.FrameTime{ComposedAt: "2026-08-12T00:00:00Z"},
			Epistemic:  dataplane.FrameEpistemic{Lane: "provisional_semantic", CompletenessMillionths: 1000000, ConflictStatus: "none"},
			Derivation: dataplane.FrameDerivation{PacketDigests: packetDigests, ModuleSetDigest: testDigest("modules"), MappingIDs: []string{"mapping:test/country@0.1", "mapping:test/indicator@0.1", "mapping:test/period@0.1", "mapping:test/value@0.1"}, CompilerContractDigest: testDigest("compiler"), CompilerVersion: "twirx-test@0.1"},
			Lifecycle:  dataplane.FrameLifecycle{State: "current"},
		}
		encoded, err := dataplane.MarshalFrame(frame)
		if err != nil {
			tb.Fatalf("marshal synthetic frame %d: %v", index, err)
		}
		result[index] = SourceFrame{Digest: dataplane.DigestBytes(encoded), CBOR: encoded, Frame: frame}
	}
	return result
}

func testDigest(label string) dataplane.Digest { return sha256.Sum256([]byte(label)) }

func sortDigests(values []dataplane.Digest) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && bytes.Compare(values[j][:], values[j-1][:]) < 0; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
