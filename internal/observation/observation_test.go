package observation

import (
	"bytes"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/cas"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	result := &safefetch.Result{
		RequestURL:  "https://example.org/product/1",
		FinalURL:    "https://example.org/product/1",
		Method:      "GET",
		Status:      200,
		MediaType:   "application/json",
		RetrievedAt: time.Date(2026, 8, 10, 0, 0, 0, 123, time.UTC),
		Body:        []byte(`{"price":"19.99"}`),
	}
	env, err := FromFetch(result, "test-policy")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := env.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnmarshalCBOR(encoded)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := decoded.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, reencoded) {
		t.Fatal("CBOR encoding is not deterministic")
	}
	if decoded.BodyDigest() != cas.Digest(result.Body) {
		t.Fatalf("digest mismatch: %s", decoded.BodyDigest())
	}
}
