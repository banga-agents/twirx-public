package ontologyfabric

import (
	"bytes"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

const kernelSource = `{
  "format": "tw.ontology-module-source/0.1",
  "module_id": "tw:kernel",
  "version": "0.1.0",
  "status": "draft",
  "imports": [],
  "concepts": [
    {"id":"tw:Entity","kind":"entity","labels":[{"language":"en","value":"Entity"}],"broader":[]}
  ],
  "roles": [
    {"id":"tw:identifier","value_type":"identifier","required":true,"max_values":1}
  ],
  "frames": [
    {"id":"tw:EntityFrame","roles":["tw:identifier"],"required_roles":["tw:identifier"],"key_roles":["tw:identifier"]}
  ],
  "mapping_claim_digests": [],
  "query_template_ids": [],
  "visualization_ids": [],
  "review_decision_digest": null
}`

const universeSource = `{
  "format": "tw.semantic-universe-source/0.1",
  "universe_id": "tw:world-state",
  "version": "0.1.0",
  "title": "World State",
  "module_ids": ["tw:kernel@0.1.0","tw:world-state@0.1.0"],
  "frame_type_ids": ["world:IndicatorObservation"],
  "source_origin_ids": [],
  "mapping_claim_digests": [],
  "materialized_view_ids": [],
  "query_template_ids": ["query:world-state/compare@0.1"],
  "visualization_ids": ["view:world-state/time-series@0.1"],
  "update_policy_id": "policy:world-state/disabled@0.1",
  "evaluation_suite_id": "eval:world-state@0.1",
  "compiled_at": "2026-08-12T00:00:00Z"
}`

func TestCompileModuleDeterministic(t *testing.T) {
	first, err := CompileModule([]byte(kernelSource))
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompileModule([]byte(kernelSource))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.CBOR, second.CBOR) || first.Digest != second.Digest {
		t.Fatal("same source did not compile deterministically")
	}
	if first.Manifest.SourceArtifactDigest != dataplane.DigestBytes([]byte(kernelSource)) {
		t.Fatal("manifest did not bind exact authoring bytes")
	}
	if _, err := dataplane.UnmarshalOntologyModule(first.CBOR); err != nil {
		t.Fatal(err)
	}
}

func TestCompileUniverseDeterministic(t *testing.T) {
	first, err := CompileUniverse([]byte(universeSource))
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompileUniverse([]byte(universeSource))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.CBOR, second.CBOR) || first.Digest != second.Digest {
		t.Fatal("same universe did not compile deterministically")
	}
}

func TestRejectsDuplicateUnknownAndUnsortedSource(t *testing.T) {
	duplicate := strings.Replace(kernelSource, `"module_id": "tw:kernel",`, `"module_id": "tw:kernel", "module_id": "tw:other",`, 1)
	if _, err := ParseModuleSource([]byte(duplicate)); err == nil {
		t.Fatal("accepted duplicate JSON key")
	}
	unknown := strings.Replace(kernelSource, `"status": "draft",`, `"status": "draft", "surprise": true,`, 1)
	if _, err := ParseModuleSource([]byte(unknown)); err == nil {
		t.Fatal("accepted unknown source field")
	}
	unsorted := strings.Replace(kernelSource, `"imports": []`, `"imports": ["z@0.1.0","a@0.1.0"]`, 1)
	if _, err := ParseModuleSource([]byte(unsorted)); err == nil {
		t.Fatal("accepted unsorted imports")
	}
}

func TestValidateModuleSetRejectsMissingAndCycles(t *testing.T) {
	kernel, err := ParseModuleSource([]byte(kernelSource))
	if err != nil {
		t.Fatal(err)
	}
	world := kernel
	world.ModuleID = "tw:world-state"
	world.Imports = []string{"tw:kernel@0.1.0"}
	world.Concepts = []Concept{{ID: "world:Observation", Kind: "measurement", Labels: []Label{{Language: "en", Value: "World observation"}}, Broader: []string{"tw:Entity"}}}
	if err := ValidateModuleSet([]ModuleSource{kernel, world}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateModuleSet([]ModuleSource{world}); err == nil {
		t.Fatal("accepted unresolved import")
	}
	kernel.Imports = []string{"tw:world-state@0.1.0"}
	if err := ValidateModuleSet([]ModuleSource{kernel, world}); err == nil {
		t.Fatal("accepted import cycle")
	}
}

func TestValidateModuleSetRejectsUnavailableAndCyclicBroaderConcepts(t *testing.T) {
	kernel, err := ParseModuleSource([]byte(kernelSource))
	if err != nil {
		t.Fatal(err)
	}
	world := kernel
	world.ModuleID = "tw:world-state"
	world.Imports = []string{"tw:kernel@0.1.0"}
	world.Concepts = []Concept{{ID: "world:Observation", Kind: "measurement", Labels: []Label{{Language: "en", Value: "World observation"}}, Broader: []string{"missing:Concept"}}}
	if err := ValidateModuleSet([]ModuleSource{kernel, world}); err == nil {
		t.Fatal("accepted broader concept outside import closure")
	}
	world.Concepts = []Concept{
		{ID: "world:A", Kind: "entity", Labels: []Label{{Language: "en", Value: "A"}}, Broader: []string{"world:B"}},
		{ID: "world:B", Kind: "entity", Labels: []Label{{Language: "en", Value: "B"}}, Broader: []string{"world:A"}},
	}
	if err := ValidateModuleSet([]ModuleSource{kernel, world}); err == nil {
		t.Fatal("accepted broader-concept cycle")
	}
}

func TestSemanticDiffClassifiesIdentityAndRestriction(t *testing.T) {
	before, err := ParseModuleSource([]byte(kernelSource))
	if err != nil {
		t.Fatal(err)
	}
	after := before
	after.Version = "0.2.0"
	after.Roles = append([]Role(nil), before.Roles...)
	after.Roles[0].MaxValues = 2
	after.Frames = append([]FrameType(nil), before.Frames...)
	after.Frames[0].KeyRoles = []string{"tw:identifier"}
	report, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if report.Compatible || len(report.Changes) != 1 || report.Changes[0].Class != "BROADENING" {
		t.Fatalf("unexpected diff: %#v", report)
	}
	after = before
	after.Version = "0.2.0"
	after.Frames = nil
	report, err = Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if report.Compatible || len(report.Changes) == 0 || report.Changes[0].Class != "MEANING_CHANGING" {
		t.Fatalf("unexpected frame removal diff: %#v", report)
	}
}

func FuzzModuleSource(f *testing.F) {
	f.Add([]byte(kernelSource))
	f.Add([]byte(`{"format":"tw.ontology-module-source/0.1"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseModuleSource(data)
	})
}
