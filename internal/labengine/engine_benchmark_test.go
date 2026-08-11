package labengine

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkExtractionAndProvenance(b *testing.B) {
	root := repositoryRoot(b)
	contracts, err := New(root, filepath.Join(b.TempDir(), "unused"))
	if err != nil {
		b.Fatal(err)
	}
	op, err := contracts.Contracts.Find("fixture.getOffer")
	if err != nil {
		b.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "origins", "fixtures", "controlled-product.json"))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := extractFields(body, op); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSHA256Representation(b *testing.B) {
	data := make([]byte, 64<<10)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sha256.Sum256(data)
	}
}

func BenchmarkGoBundleVerification(b *testing.B) {
	engine, err := New(repositoryRoot(b), filepath.Join(b.TempDir(), "results"))
	if err != nil {
		b.Fatal(err)
	}
	invocation, err := engine.Invoke(context.Background(), Request{OriginID: "controlled-origin-lab", OperationID: "fixture.getOffer", Mode: ModeReplay, Input: map[string]string{"product_id": "demo-1"}})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := engine.Verify(invocation.Publication.Directory); err != nil {
			b.Fatal(err)
		}
	}
}
