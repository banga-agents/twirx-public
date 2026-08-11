package adapter

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/cas"
	"github.com/typed-web-commons/typed-web/internal/observation"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

func TestExecutePreservesNativeAndSemanticViews(t *testing.T) {
	root := t.TempDir()
	store := cas.New(filepath.Join(root, "cas"))
	body := []byte(`{"product_id":"sku-001","name":"Field Notebook","offer":{"current_price":"19.99","currency":"usd"}}`)
	digest, _, err := store.Put(body)
	if err != nil {
		t.Fatal(err)
	}
	bodyHash, err := cas.ParseDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	env := &observation.Envelope{
		Version:     1,
		RequestURL:  "http://127.0.0.1:18080/product/sku-001",
		FinalURL:    "http://127.0.0.1:18080/product/sku-001",
		Method:      "GET",
		Status:      200,
		MediaType:   "application/json",
		RetrievedAt: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		BodySHA256:  bodyHash,
		BodySize:    uint64(len(body)),
		PolicyID:    safefetch.DefaultPolicyID,
		ObserverID:  observation.ObserverID,
	}
	envelopeBytes, err := env.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "adapter.json")
	manifest := `{
  "format":"tw.adapter/0.1",
  "id":"origin:test/product-offer",
  "version":"0.1.0",
  "description":"test",
  "origin":{"scheme":"http","host":"127.0.0.1","port":"*","path_prefix":"/product/"},
  "operation":{"id":"commerce:getOffer","effect":"read","idempotent":true},
  "resource_type":"commerce:Offer",
  "semantic_modules":[{"id":"tw:kernel","version":"0.1.0"},{"id":"tw:commerce","version":"0.1.0"}],
  "fields":[
    {"id":"price_amount","native_term":"origin:test/current_price","semantic_term":"commerce:OfferPrice.amount","json_pointer":"/offer/current_price","value_type":"decimal","required":true,"transforms":["trim","decimal_string"],"mapping_relation":"equivalent_in_context"},
    {"id":"price_currency","native_term":"origin:test/currency","semantic_term":"commerce:OfferPrice.currency","json_pointer":"/offer/currency","value_type":"currency_code","required":true,"transforms":["trim","uppercase"],"mapping_relation":"equivalent_in_context"}
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Execute(env, envelopeBytes, store, loaded, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fields) != 2 {
		t.Fatalf("fields=%d", len(result.Fields))
	}
	if result.Fields[0].Native.LexicalValue == nil || *result.Fields[0].Native.LexicalValue != "19.99" ||
		result.Fields[0].Semantic.Value.Lexical == nil || *result.Fields[0].Semantic.Value.Lexical != "19.99" {
		t.Fatalf("unexpected amount field: %+v", result.Fields[0])
	}
	if result.Fields[1].Native.LexicalValue == nil || *result.Fields[1].Native.LexicalValue != "usd" ||
		result.Fields[1].Semantic.Value.Lexical == nil || *result.Fields[1].Semantic.Value.Lexical != "USD" {
		t.Fatalf("native meaning was not preserved: %+v", result.Fields[1])
	}
	if result.Fields[0].Provenance.BodyDigest != digest {
		t.Fatalf("provenance body digest=%s want=%s", result.Fields[0].Provenance.BodyDigest, digest)
	}
}

func BenchmarkResolveJSONPointer(b *testing.B) {
	doc := map[string]any{"offer": map[string]any{"current_price": "19.99"}}
	for i := 0; i < b.N; i++ {
		value, found, err := resolveJSONPointer(doc, "/offer/current_price")
		if err != nil || !found || value != "19.99" {
			b.Fatalf("value=%v found=%v err=%v", value, found, err)
		}
	}
}

func TestDigestSanity(t *testing.T) {
	// Guard against accidental test fixtures with impossible all-zero hashes.
	sum := sha256.Sum256([]byte("typed-web"))
	if sum == [32]byte{} {
		t.Fatal("unexpected zero digest")
	}
}
