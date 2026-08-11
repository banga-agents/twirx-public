package observation

import (
	"crypto/sha256"
	"testing"
)

func benchmarkEnvelope() *Envelope {
	body := sha256.Sum256([]byte("benchmark representation"))
	return &Envelope{Version: FormatVersion, RequestURL: "https://example.test/data.json", FinalURL: "https://example.test/data.json", Method: "GET", Status: 200, MediaType: "application/json", RetrievedAt: "2026-08-10T00:00:00Z", BodySHA256: body, BodySize: 24, PolicyID: "tw.fetch.benchmark-v0", ObserverID: ObserverID}
}

func BenchmarkMarshalObservation(b *testing.B) {
	envelope := benchmarkEnvelope()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := envelope.MarshalCBOR(); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkUnmarshalObservation(b *testing.B) {
	encoded, err := benchmarkEnvelope().MarshalCBOR()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(encoded)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := UnmarshalCBOR(encoded); err != nil {
			b.Fatal(err)
		}
	}
}
