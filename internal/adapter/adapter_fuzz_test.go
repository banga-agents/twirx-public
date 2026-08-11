package adapter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/observation"
)

func FuzzJSONPointer(f *testing.F) {
	for _, seed := range []string{"", "/offer/current_price", "/a~1b", "/m~0n", "/array/0", "/array/01", "/bad~2escape"} {
		f.Add(seed)
	}
	document := map[string]any{
		"offer": map[string]any{"current_price": "19.99"},
		"a/b":   "slash",
		"m~n":   "tilde",
		"array": []any{"zero", "one"},
	}
	f.Fuzz(func(t *testing.T, pointer string) {
		_, _, _ = resolveJSONPointer(document, pointer)
	})
}

func FuzzDecodeManifest(f *testing.F) {
	for _, path := range []string{
		filepath.Join("..", "..", "adapters", "testorigin-product", "adapter.json"),
		filepath.Join("..", "..", "conformance", "adversarial", "duplicate-manifest.json"),
	} {
		seed, err := os.ReadFile(path)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		loaded, err := DecodeManifest(data)
		if err == nil {
			if loaded == nil || loaded.Manifest == nil {
				t.Fatal("successful manifest decode returned nil")
			}
			if err := loaded.Manifest.Validate(); err != nil {
				t.Fatalf("decoded manifest does not validate: %v", err)
			}
		}
	})
}

func FuzzResultExtraction(f *testing.F) {
	root := filepath.Join("..", "..")
	for _, path := range []string{
		"conformance/fixtures/product.json",
		"conformance/fixtures/product-missing-required.json",
		"conformance/adversarial/prompt-injection.json",
		"conformance/adversarial/duplicate-keys.json",
		"conformance/adversarial/trailing-json.json",
		"conformance/adversarial/unpaired-surrogate.json",
	} {
		seed, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(seed)
	}
	loaded, err := Load(filepath.Join(root, "adapters", "testorigin-product", "adapter.json"))
	if err != nil {
		f.Fatal(err)
	}
	env := &observation.Envelope{
		Version:     observation.FormatVersion,
		RequestURL:  "http://127.0.0.1:18080/product/fuzz",
		FinalURL:    "http://127.0.0.1:18080/product/fuzz",
		Method:      "GET",
		Status:      200,
		MediaType:   "application/json",
		RetrievedAt: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		PolicyID:    "tw.fetch.local-fixture-v0",
		ObserverID:  observation.ObserverID,
	}
	policy := jsonbounded.Policy{
		MaxBytes:            observation.MaxBodyBytes,
		MaxDepth:            MaxObservedJSONDepth,
		MaxScalarBytes:      MaxObservedScalarBytes,
		MaxContainerEntries: MaxObservedContainerItems,
		MaxTokens:           MaxObservedJSONTokens,
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var document any
		if err := jsonbounded.Decode(data, &document, policy, false); err != nil {
			return
		}
		_, _ = extractFields(document, env, "sha256:fuzz-observation", loaded)
	})
}
