// Package ontologyfabric compiles bounded authoring records into the
// language-neutral canonical objects defined by the TWIRX Ontology Fabric.
// JSON is a Genesis authoring carriage, not normative protocol authority.
package ontologyfabric

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	ModuleSourceFormat   = "tw.ontology-module-source/0.1"
	UniverseSourceFormat = "tw.semantic-universe-source/0.1"
	MaxSourceBytes       = 4 << 20
)

var sourcePolicy = jsonbounded.Policy{
	MaxBytes:            MaxSourceBytes,
	MaxDepth:            24,
	MaxScalarBytes:      256 << 10,
	MaxContainerEntries: 8192,
	MaxTokens:           200000,
}

type Label struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type Concept struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Labels  []Label  `json:"labels"`
	Broader []string `json:"broader"`
}

type Role struct {
	ID        string `json:"id"`
	ValueType string `json:"value_type"`
	Required  bool   `json:"required"`
	MaxValues uint64 `json:"max_values"`
}

type FrameType struct {
	ID            string   `json:"id"`
	Roles         []string `json:"roles"`
	RequiredRoles []string `json:"required_roles"`
	KeyRoles      []string `json:"key_roles"`
}

type ModuleSource struct {
	Format               string      `json:"format"`
	ModuleID             string      `json:"module_id"`
	Version              string      `json:"version"`
	Status               string      `json:"status"`
	Imports              []string    `json:"imports"`
	Concepts             []Concept   `json:"concepts"`
	Roles                []Role      `json:"roles"`
	Frames               []FrameType `json:"frames"`
	MappingClaimDigests  []string    `json:"mapping_claim_digests"`
	QueryTemplateIDs     []string    `json:"query_template_ids"`
	VisualizationIDs     []string    `json:"visualization_ids"`
	ReviewDecisionDigest *string     `json:"review_decision_digest"`
}

type UniverseSource struct {
	Format              string   `json:"format"`
	UniverseID          string   `json:"universe_id"`
	Version             string   `json:"version"`
	Title               string   `json:"title"`
	ModuleIDs           []string `json:"module_ids"`
	FrameTypeIDs        []string `json:"frame_type_ids"`
	SourceOriginIDs     []string `json:"source_origin_ids"`
	MappingClaimDigests []string `json:"mapping_claim_digests"`
	MaterializedViewIDs []string `json:"materialized_view_ids"`
	QueryTemplateIDs    []string `json:"query_template_ids"`
	VisualizationIDs    []string `json:"visualization_ids"`
	UpdatePolicyID      string   `json:"update_policy_id"`
	EvaluationSuiteID   string   `json:"evaluation_suite_id"`
	CompiledAt          string   `json:"compiled_at"`
}

type CompiledModule struct {
	Manifest dataplane.OntologyModuleManifest
	CBOR     []byte
	Digest   dataplane.Digest
}

type CompiledUniverse struct {
	Universe dataplane.SemanticUniverse
	CBOR     []byte
	Digest   dataplane.Digest
}

func ParseModuleSource(data []byte) (ModuleSource, error) {
	var source ModuleSource
	if err := jsonbounded.Decode(data, &source, sourcePolicy, true); err != nil {
		return source, fmt.Errorf("ontology source: %w", err)
	}
	if err := source.Validate(); err != nil {
		return source, err
	}
	return source, nil
}

func ParseUniverseSource(data []byte) (UniverseSource, error) {
	var source UniverseSource
	if err := jsonbounded.Decode(data, &source, sourcePolicy, true); err != nil {
		return source, fmt.Errorf("universe source: %w", err)
	}
	if err := source.Validate(); err != nil {
		return source, err
	}
	return source, nil
}

func (source ModuleSource) Validate() error {
	if source.Format != ModuleSourceFormat {
		return fmt.Errorf("ontology source: unsupported format %q", source.Format)
	}
	if err := identifier("module_id", source.ModuleID); err != nil {
		return err
	}
	if err := sortedText("imports", source.Imports, 64, 0); err != nil {
		return err
	}
	if len(source.Concepts) < 1 || len(source.Concepts) > 4096 {
		return fmt.Errorf("ontology source: concept count outside 1..4096")
	}
	concepts := make(map[string]struct{}, len(source.Concepts))
	for i, concept := range source.Concepts {
		if err := concept.Validate(); err != nil {
			return fmt.Errorf("concept %d: %w", i, err)
		}
		if i > 0 && source.Concepts[i-1].ID >= concept.ID {
			return fmt.Errorf("ontology source: concepts must be strictly sorted")
		}
		concepts[concept.ID] = struct{}{}
	}
	if len(source.Roles) > 4096 {
		return fmt.Errorf("ontology source: role count exceeds 4096")
	}
	roles := make(map[string]struct{}, len(source.Roles))
	for i, role := range source.Roles {
		if err := role.Validate(); err != nil {
			return fmt.Errorf("role %d: %w", i, err)
		}
		if i > 0 && source.Roles[i-1].ID >= role.ID {
			return fmt.Errorf("ontology source: roles must be strictly sorted")
		}
		roles[role.ID] = struct{}{}
	}
	if len(source.Frames) > 512 {
		return fmt.Errorf("ontology source: frame count exceeds 512")
	}
	for i, frame := range source.Frames {
		if err := frame.Validate(roles); err != nil {
			return fmt.Errorf("frame %d: %w", i, err)
		}
		if i > 0 && source.Frames[i-1].ID >= frame.ID {
			return fmt.Errorf("ontology source: frames must be strictly sorted")
		}
	}
	if err := sortedDigests("mapping_claim_digests", source.MappingClaimDigests, 4096); err != nil {
		return err
	}
	if err := sortedText("query_template_ids", source.QueryTemplateIDs, 256, 0); err != nil {
		return err
	}
	if err := sortedText("visualization_ids", source.VisualizationIDs, 256, 0); err != nil {
		return err
	}
	if source.ReviewDecisionDigest != nil {
		if _, err := parseDigest(*source.ReviewDecisionDigest); err != nil {
			return fmt.Errorf("review_decision_digest: %w", err)
		}
	}
	manifest, err := source.manifest(make([]byte, 1))
	if err != nil {
		return err
	}
	return manifest.Validate()
}

func (concept Concept) Validate() error {
	if err := identifier("concept id", concept.ID); err != nil {
		return err
	}
	if err := identifier("concept kind", concept.Kind); err != nil {
		return err
	}
	if len(concept.Labels) < 1 || len(concept.Labels) > 32 {
		return fmt.Errorf("concept labels outside 1..32")
	}
	for i, label := range concept.Labels {
		if err := language(label.Language); err != nil {
			return err
		}
		if label.Value == "" || len(label.Value) > 255 || !utf8.ValidString(label.Value) {
			return fmt.Errorf("invalid concept label")
		}
		if i > 0 && labelKey(concept.Labels[i-1]) >= labelKey(label) {
			return fmt.Errorf("concept labels must be strictly sorted")
		}
	}
	return sortedText("concept broader", concept.Broader, 64, 0)
}

func labelKey(label Label) string { return label.Language + "\x00" + label.Value }

func (role Role) Validate() error {
	if err := identifier("role id", role.ID); err != nil {
		return err
	}
	allowed := map[string]bool{"boolean": true, "integer": true, "decimal": true, "text": true, "date": true, "datetime": true, "duration": true, "uri": true, "identifier": true, "frame_ref": true}
	if !allowed[role.ValueType] {
		return fmt.Errorf("unsupported role value_type %q", role.ValueType)
	}
	if role.MaxValues < 1 || role.MaxValues > 64 {
		return fmt.Errorf("role max_values outside 1..64")
	}
	return nil
}

func (frame FrameType) Validate(roles map[string]struct{}) error {
	if err := identifier("frame id", frame.ID); err != nil {
		return err
	}
	if err := sortedText("frame roles", frame.Roles, 256, 1); err != nil {
		return err
	}
	if err := sortedText("frame required_roles", frame.RequiredRoles, 256, 0); err != nil {
		return err
	}
	if err := sortedText("frame key_roles", frame.KeyRoles, 32, 1); err != nil {
		return err
	}
	roleSet := make(map[string]struct{}, len(frame.Roles))
	for _, id := range frame.Roles {
		if _, ok := roles[id]; !ok {
			return fmt.Errorf("frame references undeclared role %q", id)
		}
		roleSet[id] = struct{}{}
	}
	for _, group := range [][]string{frame.RequiredRoles, frame.KeyRoles} {
		for _, id := range group {
			if _, ok := roleSet[id]; !ok {
				return fmt.Errorf("frame role subset contains %q outside roles", id)
			}
		}
	}
	return nil
}

func (source UniverseSource) Validate() error {
	if source.Format != UniverseSourceFormat {
		return fmt.Errorf("universe source: unsupported format %q", source.Format)
	}
	for name, value := range map[string]string{
		"universe_id": source.UniverseID, "update_policy_id": source.UpdatePolicyID, "evaluation_suite_id": source.EvaluationSuiteID,
	} {
		if err := identifier(name, value); err != nil {
			return err
		}
	}
	if source.Title == "" || len(source.Title) > 255 || !utf8.ValidString(source.Title) {
		return fmt.Errorf("universe source: invalid title")
	}
	sets := []struct {
		name string
		set  []string
		max  int
		min  int
	}{
		{"module_ids", source.ModuleIDs, 64, 1}, {"frame_type_ids", source.FrameTypeIDs, 512, 1},
		{"source_origin_ids", source.SourceOriginIDs, 1024, 0}, {"materialized_view_ids", source.MaterializedViewIDs, 64, 0},
		{"query_template_ids", source.QueryTemplateIDs, 256, 0}, {"visualization_ids", source.VisualizationIDs, 256, 0},
	}
	for _, item := range sets {
		if err := sortedText(item.name, item.set, item.max, item.min); err != nil {
			return err
		}
	}
	if err := sortedDigests("mapping_claim_digests", source.MappingClaimDigests, 4096); err != nil {
		return err
	}
	moduleSetDigest := digestModuleIDs(source.ModuleIDs)
	universe, err := source.universe(moduleSetDigest)
	if err != nil {
		return err
	}
	return universe.Validate()
}

func CompileModule(data []byte) (CompiledModule, error) {
	source, err := ParseModuleSource(data)
	if err != nil {
		return CompiledModule{}, err
	}
	manifest, err := source.manifest(data)
	if err != nil {
		return CompiledModule{}, err
	}
	encoded, err := dataplane.MarshalOntologyModule(manifest)
	if err != nil {
		return CompiledModule{}, err
	}
	return CompiledModule{Manifest: manifest, CBOR: encoded, Digest: dataplane.DigestBytes(encoded)}, nil
}

func (source ModuleSource) manifest(exactSource []byte) (dataplane.OntologyModuleManifest, error) {
	mappings, err := parseDigests(source.MappingClaimDigests)
	if err != nil {
		return dataplane.OntologyModuleManifest{}, err
	}
	review, err := parseOptionalDigest(source.ReviewDecisionDigest)
	if err != nil {
		return dataplane.OntologyModuleManifest{}, err
	}
	conceptIDs := make([]string, len(source.Concepts))
	for i := range source.Concepts {
		conceptIDs[i] = source.Concepts[i].ID
	}
	roleIDs := make([]string, len(source.Roles))
	for i := range source.Roles {
		roleIDs[i] = source.Roles[i].ID
	}
	frameIDs := make([]string, len(source.Frames))
	for i := range source.Frames {
		frameIDs[i] = source.Frames[i].ID
	}
	return dataplane.OntologyModuleManifest{
		Version: dataplane.ModuleVersion, ModuleID: source.ModuleID, SemanticVersion: source.Version, Status: source.Status,
		Imports: source.Imports, ConceptIDs: conceptIDs, FrameTypeIDs: frameIDs, RoleIDs: roleIDs,
		MappingClaimDigests: mappings, QueryTemplateIDs: source.QueryTemplateIDs, VisualizationIDs: source.VisualizationIDs,
		SourceArtifactDigest: dataplane.DigestBytes(exactSource), ReviewDecisionDigest: review,
	}, nil
}

func CompileUniverse(data []byte) (CompiledUniverse, error) {
	source, err := ParseUniverseSource(data)
	if err != nil {
		return CompiledUniverse{}, err
	}
	universe, err := source.universe(digestModuleIDs(source.ModuleIDs))
	if err != nil {
		return CompiledUniverse{}, err
	}
	encoded, err := dataplane.MarshalSemanticUniverse(universe)
	if err != nil {
		return CompiledUniverse{}, err
	}
	return CompiledUniverse{Universe: universe, CBOR: encoded, Digest: dataplane.DigestBytes(encoded)}, nil
}

func (source UniverseSource) universe(moduleSetDigest dataplane.Digest) (dataplane.SemanticUniverse, error) {
	mappings, err := parseDigests(source.MappingClaimDigests)
	if err != nil {
		return dataplane.SemanticUniverse{}, err
	}
	return dataplane.SemanticUniverse{
		Version: dataplane.UniverseVersion, UniverseID: source.UniverseID, SemanticVersion: source.Version, Title: source.Title,
		ModuleIDs: source.ModuleIDs, FrameTypeIDs: source.FrameTypeIDs, SourceOriginIDs: source.SourceOriginIDs,
		MappingClaimDigests: mappings, MaterializedViewIDs: source.MaterializedViewIDs,
		QueryTemplateIDs: source.QueryTemplateIDs, VisualizationIDs: source.VisualizationIDs,
		UpdatePolicyID: source.UpdatePolicyID, EvaluationSuiteID: source.EvaluationSuiteID,
		ModuleSetDigest: moduleSetDigest, CompiledAt: source.CompiledAt,
	}, nil
}

func ValidateModuleSet(sources []ModuleSource) error {
	if len(sources) == 0 || len(sources) > 64 {
		return fmt.Errorf("ontology set: source count outside 1..64")
	}
	byRef := make(map[string]ModuleSource, len(sources))
	for _, source := range sources {
		if err := source.Validate(); err != nil {
			return err
		}
		ref := source.ModuleID + "@" + source.Version
		if _, exists := byRef[ref]; exists {
			return fmt.Errorf("ontology set: duplicate module %q", ref)
		}
		byRef[ref] = source
	}
	state := make(map[string]uint8, len(byRef))
	var visit func(string, int) error
	visit = func(ref string, depth int) error {
		if depth > 16 {
			return fmt.Errorf("ontology set: import depth exceeds 16")
		}
		if state[ref] == 1 {
			return fmt.Errorf("ontology set: import cycle at %q", ref)
		}
		if state[ref] == 2 {
			return nil
		}
		source, exists := byRef[ref]
		if !exists {
			return fmt.Errorf("ontology set: missing exact import %q", ref)
		}
		state[ref] = 1
		for _, imported := range source.Imports {
			if err := visit(imported, depth+1); err != nil {
				return err
			}
		}
		state[ref] = 2
		return nil
	}
	refs := make([]string, 0, len(byRef))
	for ref := range byRef {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		if err := visit(ref, 1); err != nil {
			return err
		}
	}
	concepts := make(map[string]Concept)
	conceptOwner := make(map[string]string)
	for ref, source := range byRef {
		for _, concept := range source.Concepts {
			if prior, exists := conceptOwner[concept.ID]; exists {
				return fmt.Errorf("ontology set: concept %q is declared by both %q and %q", concept.ID, prior, ref)
			}
			concepts[concept.ID] = concept
			conceptOwner[concept.ID] = ref
		}
	}
	accessible := make(map[string]map[string]struct{}, len(byRef))
	var collect func(string) map[string]struct{}
	collect = func(ref string) map[string]struct{} {
		if result, ok := accessible[ref]; ok {
			return result
		}
		result := make(map[string]struct{})
		accessible[ref] = result
		source := byRef[ref]
		for _, concept := range source.Concepts {
			result[concept.ID] = struct{}{}
		}
		for _, imported := range source.Imports {
			for conceptID := range collect(imported) {
				result[conceptID] = struct{}{}
			}
		}
		return result
	}
	for ref, source := range byRef {
		allowed := collect(ref)
		for _, concept := range source.Concepts {
			for _, broader := range concept.Broader {
				if _, ok := allowed[broader]; !ok {
					return fmt.Errorf("ontology set: concept %q references unavailable broader concept %q", concept.ID, broader)
				}
			}
		}
	}
	conceptState := make(map[string]uint8, len(concepts))
	var visitConcept func(string) error
	visitConcept = func(id string) error {
		if conceptState[id] == 1 {
			return fmt.Errorf("ontology set: broader-concept cycle at %q", id)
		}
		if conceptState[id] == 2 {
			return nil
		}
		conceptState[id] = 1
		for _, broader := range concepts[id].Broader {
			if err := visitConcept(broader); err != nil {
				return err
			}
		}
		conceptState[id] = 2
		return nil
	}
	conceptIDs := make([]string, 0, len(concepts))
	for id := range concepts {
		conceptIDs = append(conceptIDs, id)
	}
	sort.Strings(conceptIDs)
	for _, id := range conceptIDs {
		if err := visitConcept(id); err != nil {
			return err
		}
	}
	return nil
}

func digestModuleIDs(ids []string) dataplane.Digest {
	var joined bytes.Buffer
	for _, id := range ids {
		joined.WriteString(id)
		joined.WriteByte('\n')
	}
	return sha256.Sum256(joined.Bytes())
}

func identifier(name, value string) error {
	if value == "" || len(value) > dataplane.MaxIdentifier || !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("ontology source: invalid %s", name)
	}
	return nil
}

func language(value string) error {
	if len(value) < 2 || len(value) > 63 {
		return fmt.Errorf("ontology source: invalid language %q", value)
	}
	for _, r := range value {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("ontology source: invalid language %q", value)
		}
	}
	return nil
}

func sortedText(name string, values []string, maximum, minimum int) error {
	if len(values) < minimum || len(values) > maximum {
		return fmt.Errorf("ontology source: %s count outside %d..%d", name, minimum, maximum)
	}
	for i, value := range values {
		if err := identifier(name, value); err != nil {
			return err
		}
		if i > 0 && values[i-1] >= value {
			return fmt.Errorf("ontology source: %s must be strictly sorted", name)
		}
	}
	return nil
}

func sortedDigests(name string, values []string, maximum int) error {
	if len(values) > maximum {
		return fmt.Errorf("ontology source: %s exceeds %d", name, maximum)
	}
	for i, value := range values {
		if _, err := parseDigest(value); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if i > 0 && values[i-1] >= value {
			return fmt.Errorf("ontology source: %s must be strictly sorted", name)
		}
	}
	return nil
}

func parseDigest(value string) (dataplane.Digest, error) {
	var digest dataplane.Digest
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return digest, fmt.Errorf("digest must be sha256 plus 64 lowercase hex characters")
	}
	if value != strings.ToLower(value) {
		return digest, fmt.Errorf("digest must use lowercase hexadecimal")
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	if err != nil || len(decoded) != 32 {
		return digest, fmt.Errorf("invalid SHA-256 digest")
	}
	copy(digest[:], decoded)
	return digest, nil
}

func parseDigests(values []string) ([]dataplane.Digest, error) {
	out := make([]dataplane.Digest, len(values))
	for i, value := range values {
		digest, err := parseDigest(value)
		if err != nil {
			return nil, err
		}
		out[i] = digest
	}
	return out, nil
}

func parseOptionalDigest(value *string) (dataplane.OptionalDigest, error) {
	if value == nil {
		return dataplane.OptionalDigest{}, nil
	}
	digest, err := parseDigest(*value)
	if err != nil {
		return dataplane.OptionalDigest{}, err
	}
	return dataplane.OptionalDigest{Present: true, Value: digest}, nil
}

func MarshalReport(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
