package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/typed-web-commons/typed-web/internal/cas"
	"github.com/typed-web-commons/typed-web/internal/observation"
)

type extractionCorpus struct {
	Format  string             `json:"format"`
	Vectors []extractionVector `json:"vectors"`
}

type extractionVector struct {
	ID            string                   `json:"id"`
	SourceFile    string                   `json:"source_file"`
	AdapterFile   string                   `json:"adapter_file"`
	FinalURL      string                   `json:"final_url"`
	MediaType     string                   `json:"media_type"`
	Expected      string                   `json:"expected"`
	ErrorContains string                   `json:"error_contains"`
	Fields        map[string]expectedField `json:"fields"`
}

type expectedField struct {
	Status     string  `json:"status"`
	Native     *string `json:"native"`
	Semantic   *string `json:"semantic"`
	Provenance bool    `json:"provenance"`
}

func TestPublicExtractionVectors(t *testing.T) {
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "conformance", "extraction", "vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus extractionCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Format != "tw.conformance-extraction/0.1" || len(corpus.Vectors) == 0 {
		t.Fatalf("invalid extraction corpus metadata")
	}

	for _, vector := range corpus.Vectors {
		vector := vector
		t.Run(vector.ID, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(vector.SourceFile)))
			if err != nil {
				t.Fatal(err)
			}
			store := cas.New(filepath.Join(t.TempDir(), "cas"))
			digest, _, err := store.Put(body)
			if err != nil {
				t.Fatal(err)
			}
			bodyHash, err := cas.ParseDigest(digest)
			if err != nil {
				t.Fatal(err)
			}
			env := &observation.Envelope{
				Version:     observation.FormatVersion,
				RequestURL:  vector.FinalURL,
				FinalURL:    vector.FinalURL,
				Method:      "GET",
				Status:      200,
				MediaType:   vector.MediaType,
				RetrievedAt: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
				BodySHA256:  bodyHash,
				BodySize:    uint64(len(body)),
				PolicyID:    "tw.fetch.local-fixture-v0",
				ObserverID:  observation.ObserverID,
			}
			envelopeBytes, err := env.MarshalCBOR()
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := Load(filepath.Join(root, filepath.FromSlash(vector.AdapterFile)))
			if err != nil {
				t.Fatal(err)
			}
			result, executeErr := Execute(env, envelopeBytes, store, loaded, observation.MaxBodyBytes)
			if vector.Expected == "reject" {
				if executeErr == nil || !strings.Contains(executeErr.Error(), vector.ErrorContains) {
					t.Fatalf("err=%v want rejection containing %q", executeErr, vector.ErrorContains)
				}
				return
			}
			if executeErr != nil {
				t.Fatal(executeErr)
			}
			fields := make(map[string]ResultField, len(result.Fields))
			for _, field := range result.Fields {
				fields[field.ID] = field
				assertCompleteProvenance(t, field)
				if field.Status != "resolved" && field.Status != "unresolved" {
					t.Fatalf("field %s has invalid status %q", field.ID, field.Status)
				}
				if field.Status == "unresolved" && (field.Native.LexicalValue != nil || field.Semantic.Value.Lexical != nil) {
					t.Fatalf("unresolved field %s contains a fabricated lexical value", field.ID)
				}
			}
			for id, expected := range vector.Fields {
				field, ok := fields[id]
				if !ok {
					t.Fatalf("missing result field %q", id)
				}
				if field.Status != expected.Status {
					t.Fatalf("field %s status=%q want=%q", id, field.Status, expected.Status)
				}
				if expected.Native != nil && (field.Native.LexicalValue == nil || *field.Native.LexicalValue != *expected.Native) {
					t.Fatalf("field %s native=%v want=%q", id, field.Native.LexicalValue, *expected.Native)
				}
				if expected.Semantic != nil && (field.Semantic.Value.Lexical == nil || *field.Semantic.Value.Lexical != *expected.Semantic) {
					t.Fatalf("field %s semantic=%v want=%q", id, field.Semantic.Value.Lexical, *expected.Semantic)
				}
				if expected.Provenance {
					assertCompleteProvenance(t, field)
				}
			}
			encoded, err := MarshalResult(result)
			if err != nil {
				t.Fatal(err)
			}
			if vector.ID == "resolved-empty-string" && !strings.Contains(string(encoded), `"lexical_value": ""`) {
				t.Fatalf("resolved empty native lexical value was omitted: %s", encoded)
			}
		})
	}
}

func assertCompleteProvenance(t *testing.T, field ResultField) {
	t.Helper()
	provenance := field.Provenance
	for name, value := range map[string]string{
		"request_url": provenance.RequestURL, "final_url": provenance.FinalURL,
		"retrieved_at": provenance.RetrievedAt, "body_digest": provenance.BodyDigest,
		"observation_hash": provenance.ObservationHash, "adapter_id": provenance.AdapterID,
		"adapter_version": provenance.AdapterVersion, "adapter_digest": provenance.AdapterDigest,
		"extraction_method": provenance.ExtractionMethod, "locator": provenance.Locator,
		"mapping_relation": provenance.MappingRelation,
	} {
		if value == "" {
			t.Fatalf("field %s provenance %s is empty", field.ID, name)
		}
	}
	if provenance.Locator != field.Native.Locator {
		t.Fatalf("field %s locator mismatch", field.ID)
	}
	if provenance.TransformChain == nil {
		t.Fatalf("field %s provenance transform chain is nil", field.ID)
	}
}

func TestManifestRejectsDuplicateKeys(t *testing.T) {
	_, err := Load(filepath.Join("..", "..", "conformance", "adversarial", "duplicate-manifest.json"))
	if err == nil || !strings.Contains(err.Error(), "duplicate object key") {
		t.Fatalf("err=%v", err)
	}
}

func TestJSONPointerRejectsNonCanonicalArrayIndexes(t *testing.T) {
	document := []any{"zero", "one"}
	for _, pointer := range []string{"/01", "/+1", "/-1"} {
		if _, _, err := resolveJSONPointer(document, pointer); err == nil {
			t.Fatalf("pointer %q was accepted", pointer)
		}
	}
}
