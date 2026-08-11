package e2format

import "testing"

func BenchmarkMarshalResult(b *testing.B) {
	result := validResult()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := MarshalResult(result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalResult(b *testing.B) {
	encoded, err := MarshalResult(validResult())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(encoded)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := UnmarshalResult(encoded); err != nil {
			b.Fatal(err)
		}
	}
}
