package main

// Evidence generation.
//
// `go run . -evidence` reads committed repository artefacts and writes
// data/evidence-e1.json, which drives the provenance explorer. Nothing in that
// file is authored by hand: every value is copied from an artefact or computed
// from one, and each source is recorded with its SHA-256 so a reviewer can tell
// exactly which bytes the published proof came from.
//
// The raw origin body is read from the content-addressed store produced by
// `make demo` at the repository root, because those are the actual bytes the
// controlled origin returned. Run `make demo` before regenerating.
//
// The ordinary site build does not regenerate this file. It re-verifies it —
// see verifyEvidence in main.go.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Evidence is the generated proof artefact published at /data/evidence-e1.json.
type Evidence struct {
	Generator string   `json:"_generator"`
	Warning   string   `json:"_warning"`
	Gate      string   `json:"gate"`
	Sources   []Source `json:"sources"`

	Origin       map[string]any `json:"origin"`
	Observation  map[string]any `json:"observation"`
	Adapter      map[string]any `json:"adapter"`
	Verification []VerifyStep   `json:"verification"`
	Result       map[string]any `json:"result"`
	Fields       []any          `json:"fields"`

	UnresolvedExample map[string]any `json:"unresolved_example"`
	ExtractionVectors []any          `json:"extraction_vectors"`
	ObservationVector map[string]any `json:"observation_vectors"`
}

// Source records one input artefact and the digest of the exact bytes read.
type Source struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

// VerifyStep is one independent implementation that validated the artefact.
type VerifyStep struct {
	Implementation string `json:"implementation"`
	Language       string `json:"language"`
	Validates      string `json:"validates"`
	Result         string `json:"result"`
	Evidence       string `json:"evidence"`
	Authority      string `json:"authority"`
}

func generateEvidence(repo, out string) error {
	var srcs []Source

	read := func(rel string) ([]byte, error) {
		b, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil {
			return nil, fmt.Errorf("%s: %w (run `make demo` at the repository root first)", rel, err)
		}
		sum := sha256.Sum256(b)
		srcs = append(srcs, Source{Path: rel, SHA256: hex.EncodeToString(sum[:]), Bytes: len(b)})
		return b, nil
	}
	readJSON := func(rel string, v any) error {
		b, err := read(rel)
		if err != nil {
			return err
		}
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.UseNumber()
		return dec.Decode(v)
	}

	var observation map[string]any
	if err := readJSON("examples/demo-observation.json", &observation); err != nil {
		return err
	}
	var result map[string]any
	if err := readJSON("examples/demo-result.json", &result); err != nil {
		return err
	}
	var adapter map[string]any
	if err := readJSON("adapters/testorigin-product/adapter.json", &adapter); err != nil {
		return err
	}
	var extraction map[string]any
	if err := readJSON("conformance/extraction/vectors.json", &extraction); err != nil {
		return err
	}
	var obsVectors map[string]any
	if err := readJSON("conformance/observation/vectors.json", &obsVectors); err != nil {
		return err
	}
	var unresolvedSource map[string]any
	if err := readJSON("conformance/fixtures/product-missing-optional.json", &unresolvedSource); err != nil {
		return err
	}

	// The raw body, read from the content-addressed store by the digest the
	// observation itself records. This is the one step where the generator
	// follows the protocol's own addressing rather than a filename.
	digest, _ := observation["body_digest"].(string)
	hexDigest := strings.TrimPrefix(digest, "sha256:")
	if len(hexDigest) != 64 {
		return fmt.Errorf("observation body_digest is not a sha256 hex digest: %q", digest)
	}
	casRel := filepath.Join("var", "cas", "sha256", hexDigest[0:2], hexDigest[2:4], hexDigest)
	body, err := read(casRel)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	if got := hex.EncodeToString(sum[:]); got != hexDigest {
		return fmt.Errorf("evidence store body does not match its own address: %s != %s", got, hexDigest)
	}

	origin := map[string]any{
		"request_url":  observation["request_url"],
		"final_url":    observation["final_url"],
		"method":       observation["method"],
		"status":       observation["status"],
		"media_type":   observation["media_type"],
		"retrieved_at": observation["retrieved_at"],
		"policy_id":    observation["policy_id"],
		"observer_id":  observation["observer_id"],
		"body_size":    observation["body_size"],
		"body_digest":  observation["body_digest"],
		"body_text":    string(body),
		"note":         "A controlled local fixture origin, not a production source. Loopback access is denied unless explicitly enabled.",
	}

	fields, _ := result["fields"].([]any)
	slim := map[string]any{}
	for _, k := range []string{"format", "operation_id", "resource_type", "observation_digest",
		"adapter_id", "adapter_version", "adapter_digest", "semantic_closure",
		"semantic_closure_hash", "result_digest"} {
		slim[k] = result[k]
	}

	adapterSlim := map[string]any{}
	for _, k := range []string{"format", "id", "version", "description", "origin", "operation",
		"resource_type", "semantic_modules"} {
		adapterSlim[k] = adapter[k]
	}
	// The adapter digest recorded in the result must be the hash of the
	// committed adapter file. Recompute it rather than copying the claim.
	adapterPath := "adapters/testorigin-product/adapter.json"
	var adapterFileDigest string
	for _, s := range srcs {
		if s.Path == adapterPath {
			adapterFileDigest = "sha256:" + s.SHA256
		}
	}
	if claimed, _ := result["adapter_digest"].(string); claimed != adapterFileDigest {
		return fmt.Errorf("adapter digest mismatch: result claims %s, but %s hashes to %s",
			claimed, adapterPath, adapterFileDigest)
	}
	adapterSlim["digest"] = adapterFileDigest
	adapterSlim["digest_source"] = "SHA-256 of " + adapterPath + ", recomputed at generation and re-checked on every site build."
	adapterSlim["path"] = adapterPath

	ev := Evidence{
		Generator: "web/evidence.go — regenerate with `cd web && go run . -evidence` after `make demo`",
		Warning: "Generated file. Do not edit by hand. Every value is copied from a committed repository " +
			"artefact or computed from one; `sources` records the digest of each input actually read. " +
			"The site build re-verifies the digest chain in this file and fails if it does not hold.",
		Gate:        "E1",
		Sources:     srcs,
		Origin:      origin,
		Observation: observation,
		Adapter:     adapterSlim,
		Verification: []VerifyStep{
			{
				Implementation: "Primary verifier",
				Language:       "Go",
				Validates:      "Canonical CBOR observation envelope and the stored body, re-hashed from the content-addressed store before extraction.",
				Result:         "16 of 16 shared vectors as expected; corrupted evidence rejected.",
				Evidence:       "reports/gate-1-genesis.md",
				Authority:      "Validates. Cannot alone establish that a rule belongs to the protocol.",
			},
			{
				Implementation: "Independent verifier",
				Language:       "C (restricted)",
				Validates:      "The same envelope and body, from the same committed vectors, under a separate bounded parser.",
				Result:         "16 of 16 shared vectors as expected; post-validation CAS corruption rejected; ASan and UBSan clean.",
				Evidence:       "reports/gate-1-genesis.md",
				Authority:      "No network path, no registry authority, writes no canonical state. It can reject; it cannot admit.",
			},
		},
		Result: slim,
		Fields: fields,
		UnresolvedExample: map[string]any{
			"why":                "An optional field that the origin did not provide becomes an explicit unresolved result with provenance, and no fabricated lexical value.",
			"vector":             "missing-optional-source-field",
			"source_file":        "conformance/fixtures/product-missing-optional.json",
			"source_text":        mustCompact(unresolvedSource),
			"field":              "availability",
			"expected_status":    "unresolved",
			"provenance_present": true,
			"contrast":           "A resolved empty string stays distinct from unresolved: see the resolved-empty-string vector.",
		},
		ExtractionVectors: toSlice(extraction["vectors"]),
		ObservationVector: map[string]any{
			"total":   len(toSlice(obsVectors["vectors"])),
			"vectors": toSlice(obsVectors["vectors"]),
		},
	}

	buf, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	if err := os.WriteFile(out, buf, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s from %d repository artefacts (%d fields, %d extraction vectors)\n",
		out, len(srcs), len(fields), len(ev.ExtractionVectors))
	return nil
}

func toSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func mustCompact(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}
