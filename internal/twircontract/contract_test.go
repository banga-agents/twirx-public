package twircontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func loadTestSet(t *testing.T) *Set {
	t.Helper()
	set, err := Load(filepath.Join("..", "..", "contracts", "e2", "contracts.json"))
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestLoadGenerateAndCanonicalize(t *testing.T) {
	set := loadTestSet(t)
	if len(set.Operations) < 5 {
		t.Fatalf("operation count %d", len(set.Operations))
	}
	op, err := set.Find("development.getIndicator")
	if err != nil {
		t.Fatal(err)
	}
	input := map[string]string{"country": "CHL", "indicator": "SP.POP.TOTL", "year": "2024"}
	first, err := CanonicalInput(op, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CanonicalInput(op, input)
	if err != nil || string(first) != string(second) {
		t.Fatal("input is not deterministic")
	}
	decoded, err := UnmarshalInput(op, first)
	if err != nil || decoded["country"] != "CHL" {
		t.Fatalf("decode input: %#v %v", decoded, err)
	}
	contract, err := set.MarshalOperation(op)
	if err != nil || len(contract) == 0 {
		t.Fatalf("contract: %v", err)
	}
	closure, err := MarshalSemanticClosure(op.SemanticClosure)
	if err != nil || len(closure) == 0 {
		t.Fatalf("closure: %v", err)
	}
	adapter, err := MarshalAdapterDescriptor(op)
	if err != nil || len(adapter) == 0 {
		t.Fatalf("adapter: %v", err)
	}
	schema, err := JSONSchema(op)
	if err != nil || len(schema) == 0 {
		t.Fatalf("schema: %v", err)
	}
}

func TestInputFailsClosed(t *testing.T) {
	set := loadTestSet(t)
	op, err := set.Find("development.getIndicator")
	if err != nil {
		t.Fatal(err)
	}
	tests := []map[string]string{
		{"country": "../../etc/passwd", "indicator": "SP.POP.TOTL", "year": "2024"},
		{"country": "CHL", "indicator": "SP.POP.TOTL", "year": "2099"},
		{"country": "CHL", "indicator": "SP.POP.TOTL", "year": "2024", "url": "http://127.0.0.1"},
	}
	for _, input := range tests {
		if err := ValidateInput(op, input); err == nil {
			t.Fatalf("accepted %#v", input)
		}
	}
}

func TestNormalizeTypedJSONInput(t *testing.T) {
	set := loadTestSet(t)
	op, err := set.Find("development.getIndicator")
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := NormalizeInput(op, map[string]any{"country": "CHL", "indicator": "SP.POP.TOTL", "year": json.Number("2024")})
	if err != nil || normalized["year"] != "2024" {
		t.Fatalf("normalize: %#v %v", normalized, err)
	}
	if _, err := NormalizeInput(op, map[string]any{"country": "CHL", "indicator": "SP.POP.TOTL", "year": "2024"}); err == nil {
		t.Fatal("string integer accepted")
	}
	if _, err := NormalizeInput(op, map[string]any{"country": "CHL", "indicator": "SP.POP.TOTL", "year": json.Number("2024.0")}); err == nil {
		t.Fatal("fractional integer accepted")
	}
}

func TestLoadRejectsDuplicateAndUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	for _, value := range []string{
		`{"format":"tw.contract-set/0.1","format":"x"}`,
		`{"format":"tw.contract-set/0.1","core":"twir-core/0.1","module_id":"m","module_version":"1","operations":[],"extra":true}`,
	} {
		if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}
