// Package adapter executes deterministic, manually admitted Genesis adapters.
// It preserves native source statements while adding explicit semantic views.
package adapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/cas"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/observation"
)

const (
	ManifestFormat            = "tw.adapter/0.1"
	MaxManifestBytes          = 256 << 10
	MaxManifestDepth          = 32
	MaxManifestScalarBytes    = 8192
	MaxManifestContainerItems = 4096
	MaxManifestTokens         = 32768
	MaxObservedJSONDepth      = 64
	MaxObservedScalarBytes    = 64 << 10
	MaxObservedContainerItems = 16384
	MaxObservedJSONTokens     = 65536
	MaxFields                 = 64
	MaxSemanticModules        = 64
	MaxTransformsPerField     = 16
	MaxIDBytes                = 256
	MaxDescriptionBytes       = 4096
	MaxTermBytes              = 1024
	MaxJSONPointerBytes       = 4096
	MaxJSONPointerTokens      = 128
	MaxResultBytes            = 8 << 20
)

var manifestJSONPolicy = jsonbounded.Policy{
	MaxBytes:            MaxManifestBytes,
	MaxDepth:            MaxManifestDepth,
	MaxScalarBytes:      MaxManifestScalarBytes,
	MaxContainerEntries: MaxManifestContainerItems,
	MaxTokens:           MaxManifestTokens,
}

var decimalPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type Manifest struct {
	Format          string           `json:"format"`
	ID              string           `json:"id"`
	Version         string           `json:"version"`
	Description     string           `json:"description"`
	Origin          OriginScope      `json:"origin"`
	Operation       Operation        `json:"operation"`
	ResourceType    string           `json:"resource_type"`
	SemanticModules []SemanticModule `json:"semantic_modules"`
	Fields          []Field          `json:"fields"`
}

type OriginScope struct {
	Scheme     string `json:"scheme"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	PathPrefix string `json:"path_prefix"`
}

type Operation struct {
	ID         string `json:"id"`
	Effect     string `json:"effect"`
	Idempotent bool   `json:"idempotent"`
}

type SemanticModule struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Field struct {
	ID              string   `json:"id"`
	NativeTerm      string   `json:"native_term"`
	SemanticTerm    string   `json:"semantic_term"`
	JSONPointer     string   `json:"json_pointer"`
	ValueType       string   `json:"value_type"`
	Required        bool     `json:"required"`
	Transforms      []string `json:"transforms"`
	MappingRelation string   `json:"mapping_relation"`
}

type LoadedManifest struct {
	Manifest *Manifest
	Raw      []byte
	Digest   string
}

func Load(path string) (*LoadedManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("adapter: read manifest: %w", err)
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, MaxManifestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("adapter: read manifest: %w", err)
	}
	if len(raw) > MaxManifestBytes {
		return nil, fmt.Errorf("adapter: manifest exceeds %d-byte limit", MaxManifestBytes)
	}
	return DecodeManifest(raw)
}

// DecodeManifest validates and decodes an in-memory manifest under the Gate 1
// JSON policy. It is shared by file loading, tests, and fuzz targets.
func DecodeManifest(raw []byte) (*LoadedManifest, error) {
	var manifest Manifest
	if err := jsonbounded.Decode(raw, &manifest, manifestJSONPolicy, true); err != nil {
		return nil, fmt.Errorf("adapter: decode manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	return &LoadedManifest{
		Manifest: &manifest,
		Raw:      raw,
		Digest:   "sha256:" + hex.EncodeToString(sum[:]),
	}, nil
}

func (m *Manifest) Validate() error {
	if m == nil {
		return errors.New("adapter: manifest is nil")
	}
	if m.Format != ManifestFormat {
		return fmt.Errorf("adapter: format %q, want %q", m.Format, ManifestFormat)
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "id", value: m.ID}, {name: "version", value: m.Version},
		{name: "origin.scheme", value: m.Origin.Scheme}, {name: "origin.host", value: m.Origin.Host},
		{name: "operation.id", value: m.Operation.ID}, {name: "operation.effect", value: m.Operation.Effect},
		{name: "resource_type", value: m.ResourceType},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("adapter: %s is required", required.name)
		}
		if err := boundedText(required.name, required.value, MaxIDBytes); err != nil {
			return err
		}
	}
	if err := boundedText("description", m.Description, MaxDescriptionBytes); err != nil {
		return err
	}
	if m.Operation.Effect != "read" {
		return fmt.Errorf("adapter: Genesis permits only read operations, got %q", m.Operation.Effect)
	}
	if !m.Operation.Idempotent {
		return errors.New("adapter: Genesis read operations must be idempotent")
	}
	if m.Origin.Scheme != "http" && m.Origin.Scheme != "https" {
		return fmt.Errorf("adapter: origin scheme %q is invalid", m.Origin.Scheme)
	}
	if m.Origin.Port != "" && m.Origin.Port != "*" {
		port, err := strconv.ParseUint(m.Origin.Port, 10, 16)
		if err != nil || port == 0 || strconv.FormatUint(port, 10) != m.Origin.Port {
			return fmt.Errorf("adapter: origin port %q is invalid", m.Origin.Port)
		}
	}
	if !strings.HasPrefix(m.Origin.PathPrefix, "/") || len(m.Origin.PathPrefix) > MaxJSONPointerBytes {
		return fmt.Errorf("adapter: origin path_prefix must begin with '/' and be at most %d bytes", MaxJSONPointerBytes)
	}
	if err := boundedText("origin.path_prefix", m.Origin.PathPrefix, MaxJSONPointerBytes); err != nil {
		return err
	}
	if len(m.Fields) == 0 {
		return errors.New("adapter: at least one field is required")
	}
	if len(m.Fields) > MaxFields {
		return fmt.Errorf("adapter: field count %d exceeds limit %d", len(m.Fields), MaxFields)
	}
	if len(m.SemanticModules) > MaxSemanticModules {
		return fmt.Errorf("adapter: semantic module count %d exceeds limit %d", len(m.SemanticModules), MaxSemanticModules)
	}
	seen := make(map[string]struct{}, len(m.Fields))
	for i, field := range m.Fields {
		if field.ID == "" || field.NativeTerm == "" || field.SemanticTerm == "" || field.JSONPointer == "" || field.ValueType == "" || field.MappingRelation == "" {
			return fmt.Errorf("adapter: field %d has missing required properties", i)
		}
		if _, exists := seen[field.ID]; exists {
			return fmt.Errorf("adapter: duplicate field ID %q", field.ID)
		}
		seen[field.ID] = struct{}{}
		for _, property := range []struct {
			name  string
			value string
			max   int
		}{
			{name: "id", value: field.ID, max: MaxIDBytes},
			{name: "native_term", value: field.NativeTerm, max: MaxTermBytes},
			{name: "semantic_term", value: field.SemanticTerm, max: MaxTermBytes},
			{name: "mapping_relation", value: field.MappingRelation, max: MaxTermBytes},
		} {
			if err := boundedText("field "+field.ID+" "+property.name, property.value, property.max); err != nil {
				return err
			}
		}
		if err := validateJSONPointer(field.JSONPointer); err != nil {
			return fmt.Errorf("adapter: field %q JSON pointer: %w", field.ID, err)
		}
		switch field.ValueType {
		case "string", "decimal", "currency_code", "integer", "boolean":
		default:
			return fmt.Errorf("adapter: field %q has unsupported value type %q", field.ID, field.ValueType)
		}
		if len(field.Transforms) > MaxTransformsPerField {
			return fmt.Errorf("adapter: field %q transform count exceeds limit %d", field.ID, MaxTransformsPerField)
		}
		for _, transform := range field.Transforms {
			switch transform {
			case "trim", "uppercase", "lowercase", "decimal_string":
			default:
				return fmt.Errorf("adapter: field %q has unsupported transform %q", field.ID, transform)
			}
		}
	}
	for i, module := range m.SemanticModules {
		if module.ID == "" || module.Version == "" {
			return fmt.Errorf("adapter: semantic module %d is incomplete", i)
		}
		if err := boundedText(fmt.Sprintf("semantic module %d id", i), module.ID, MaxIDBytes); err != nil {
			return err
		}
		if err := boundedText(fmt.Sprintf("semantic module %d version", i), module.Version, MaxIDBytes); err != nil {
			return err
		}
		if strings.ContainsAny(module.ID, "@\r\n") || strings.ContainsAny(module.Version, "@\r\n") {
			return fmt.Errorf("adapter: semantic module %d identity contains a reserved closure delimiter", i)
		}
	}
	return nil
}

func boundedText(name, value string, max int) error {
	if len(value) > max {
		return fmt.Errorf("adapter: %s exceeds %d bytes", name, max)
	}
	if !utf8.ValidString(value) || strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("adapter: %s must be valid UTF-8 without U+0000", name)
	}
	return nil
}

func (m *Manifest) SemanticClosure() (string, []string) {
	modules := make([]string, 0, len(m.SemanticModules))
	for _, module := range m.SemanticModules {
		modules = append(modules, module.ID+"@"+module.Version)
	}
	sort.Strings(modules)
	sum := sha256.Sum256([]byte(strings.Join(modules, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:]), modules
}

func (m *Manifest) MatchesOrigin(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("adapter: parse observation URL: %w", err)
	}
	if u.Scheme != m.Origin.Scheme || !strings.EqualFold(u.Hostname(), m.Origin.Host) {
		return fmt.Errorf("adapter: origin %s://%s does not match %s://%s", u.Scheme, u.Hostname(), m.Origin.Scheme, m.Origin.Host)
	}
	if m.Origin.Port != "*" && u.Port() != m.Origin.Port {
		return fmt.Errorf("adapter: port %q does not match required %q", u.Port(), m.Origin.Port)
	}
	if !strings.HasPrefix(u.EscapedPath(), m.Origin.PathPrefix) {
		return fmt.Errorf("adapter: path %q does not match prefix %q", u.EscapedPath(), m.Origin.PathPrefix)
	}
	return nil
}

type ResultCore struct {
	Format              string        `json:"format"`
	OperationID         string        `json:"operation_id"`
	ResourceType        string        `json:"resource_type"`
	ObservationDigest   string        `json:"observation_digest"`
	AdapterID           string        `json:"adapter_id"`
	AdapterVersion      string        `json:"adapter_version"`
	AdapterDigest       string        `json:"adapter_digest"`
	SemanticClosure     []string      `json:"semantic_closure"`
	SemanticClosureHash string        `json:"semantic_closure_hash"`
	Fields              []ResultField `json:"fields"`
}

type Result struct {
	ResultCore
	ResultDigest string `json:"result_digest"`
}

type ResultField struct {
	ID         string          `json:"id"`
	Status     string          `json:"status"`
	Native     NativeStatement `json:"native"`
	Semantic   SemanticView    `json:"semantic"`
	Provenance FieldProvenance `json:"provenance"`
}

type NativeStatement struct {
	Term         string  `json:"term"`
	Locator      string  `json:"locator"`
	LexicalValue *string `json:"lexical_value,omitempty"`
}

type SemanticView struct {
	Term  string     `json:"term"`
	Value TypedValue `json:"value"`
}

type TypedValue struct {
	Type    string  `json:"type"`
	Lexical *string `json:"lexical,omitempty"`
}

type FieldProvenance struct {
	RequestURL       string   `json:"request_url"`
	FinalURL         string   `json:"final_url"`
	RetrievedAt      string   `json:"retrieved_at"`
	BodyDigest       string   `json:"body_digest"`
	ObservationHash  string   `json:"observation_hash"`
	AdapterID        string   `json:"adapter_id"`
	AdapterVersion   string   `json:"adapter_version"`
	AdapterDigest    string   `json:"adapter_digest"`
	ExtractionMethod string   `json:"extraction_method"`
	Locator          string   `json:"locator"`
	TransformChain   []string `json:"transform_chain"`
	MappingRelation  string   `json:"mapping_relation"`
}

func Execute(env *observation.Envelope, envelopeBytes []byte, store *cas.Store, loaded *LoadedManifest, maxBodyBytes int64) (*Result, error) {
	if env == nil || store == nil || loaded == nil || loaded.Manifest == nil {
		return nil, errors.New("adapter: envelope, store, and manifest are required")
	}
	if maxBodyBytes <= 0 || maxBodyBytes > observation.MaxBodyBytes {
		return nil, fmt.Errorf("adapter: body limit must be between 1 and %d bytes", observation.MaxBodyBytes)
	}
	decodedEnvelope, err := observation.UnmarshalCBOR(envelopeBytes)
	if err != nil {
		return nil, fmt.Errorf("adapter: verify envelope bytes: %w", err)
	}
	if *decodedEnvelope != *env {
		return nil, errors.New("adapter: envelope bytes do not match supplied envelope")
	}
	if err := loaded.Manifest.Validate(); err != nil {
		return nil, err
	}
	if err := loaded.Manifest.MatchesOrigin(env.FinalURL); err != nil {
		return nil, err
	}
	if env.Status < 200 || env.Status > 299 {
		return nil, fmt.Errorf("adapter: observation status %d is not successful", env.Status)
	}
	if env.MediaType != "application/json" && env.MediaType != "application/ld+json" {
		return nil, fmt.Errorf("adapter: media type %q is unsupported by the Genesis JSON adapter", env.MediaType)
	}
	body, err := store.Read(env.BodyDigest(), maxBodyBytes)
	if err != nil {
		return nil, err
	}
	if uint64(len(body)) != env.BodySize {
		return nil, fmt.Errorf("adapter: body size mismatch: got %d, envelope says %d", len(body), env.BodySize)
	}

	var document any
	bodyPolicy := jsonbounded.Policy{
		MaxBytes:            int(maxBodyBytes),
		MaxDepth:            MaxObservedJSONDepth,
		MaxScalarBytes:      MaxObservedScalarBytes,
		MaxContainerEntries: MaxObservedContainerItems,
		MaxTokens:           MaxObservedJSONTokens,
	}
	if err := jsonbounded.Decode(body, &document, bodyPolicy, false); err != nil {
		return nil, fmt.Errorf("adapter: parse JSON body: %w", err)
	}

	closureHash, closure := loaded.Manifest.SemanticClosure()
	observationHash := observation.EnvelopeDigest(envelopeBytes)
	fields, err := extractFields(document, env, observationHash, loaded)
	if err != nil {
		return nil, err
	}

	core := ResultCore{
		Format:              "tw.result/0.1",
		OperationID:         loaded.Manifest.Operation.ID,
		ResourceType:        loaded.Manifest.ResourceType,
		ObservationDigest:   observationHash,
		AdapterID:           loaded.Manifest.ID,
		AdapterVersion:      loaded.Manifest.Version,
		AdapterDigest:       loaded.Digest,
		SemanticClosure:     closure,
		SemanticClosureHash: closureHash,
		Fields:              fields,
	}
	canonical, err := json.Marshal(core)
	if err != nil {
		return nil, fmt.Errorf("adapter: marshal result core: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return &Result{ResultCore: core, ResultDigest: "sha256:" + hex.EncodeToString(sum[:])}, nil
}

func extractFields(document any, env *observation.Envelope, observationHash string, loaded *LoadedManifest) ([]ResultField, error) {
	fields := make([]ResultField, 0, len(loaded.Manifest.Fields))
	for _, field := range loaded.Manifest.Fields {
		value, found, err := resolveJSONPointer(document, field.JSONPointer)
		if err != nil {
			return nil, fmt.Errorf("adapter: field %q: %w", field.ID, err)
		}
		if !found {
			if field.Required {
				return nil, fmt.Errorf("adapter: required field %q missing at %s", field.ID, field.JSONPointer)
			}
			fields = append(fields, unresolvedField(env, observationHash, loaded, field))
			continue
		}
		lexical, err := lexicalValue(value)
		if err != nil {
			return nil, fmt.Errorf("adapter: field %q: %w", field.ID, err)
		}
		normalized, err := applyTransforms(lexical, field.Transforms)
		if err != nil {
			return nil, fmt.Errorf("adapter: field %q transform: %w", field.ID, err)
		}
		if err := validateTypedValue(normalized, field.ValueType); err != nil {
			return nil, fmt.Errorf("adapter: field %q typed value: %w", field.ID, err)
		}
		nativeLexical := lexical
		semanticLexical := normalized
		fields = append(fields, ResultField{
			ID:     field.ID,
			Status: "resolved",
			Native: NativeStatement{Term: field.NativeTerm, Locator: field.JSONPointer, LexicalValue: &nativeLexical},
			Semantic: SemanticView{
				Term:  field.SemanticTerm,
				Value: TypedValue{Type: field.ValueType, Lexical: &semanticLexical},
			},
			Provenance: provenance(env, observationHash, loaded, field),
		})
	}

	return fields, nil
}

func unresolvedField(env *observation.Envelope, observationHash string, loaded *LoadedManifest, field Field) ResultField {
	return ResultField{
		ID:     field.ID,
		Status: "unresolved",
		Native: NativeStatement{Term: field.NativeTerm, Locator: field.JSONPointer},
		Semantic: SemanticView{
			Term:  field.SemanticTerm,
			Value: TypedValue{Type: field.ValueType},
		},
		Provenance: provenance(env, observationHash, loaded, field),
	}
}

func provenance(env *observation.Envelope, observationHash string, loaded *LoadedManifest, field Field) FieldProvenance {
	chain := make([]string, len(field.Transforms))
	copy(chain, field.Transforms)
	return FieldProvenance{
		RequestURL:       env.RequestURL,
		FinalURL:         env.FinalURL,
		RetrievedAt:      env.RetrievedAt,
		BodyDigest:       env.BodyDigest(),
		ObservationHash:  observationHash,
		AdapterID:        loaded.Manifest.ID,
		AdapterVersion:   loaded.Manifest.Version,
		AdapterDigest:    loaded.Digest,
		ExtractionMethod: "json_pointer",
		Locator:          field.JSONPointer,
		TransformChain:   chain,
		MappingRelation:  field.MappingRelation,
	}
}

func MarshalResult(result *Result) ([]byte, error) {
	if result == nil {
		return nil, errors.New("adapter: result is nil")
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) > MaxResultBytes {
		return nil, fmt.Errorf("adapter: result size %d exceeds %d-byte limit", len(data), MaxResultBytes)
	}
	return data, nil
}

func resolveJSONPointer(document any, pointer string) (any, bool, error) {
	if err := validateJSONPointer(pointer); err != nil {
		return nil, false, err
	}
	if pointer == "" {
		return document, true, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, false, errors.New("JSON pointer must be empty or begin with '/'")
	}
	current := document
	for _, rawToken := range strings.Split(pointer[1:], "/") {
		token, err := decodePointerToken(rawToken)
		if err != nil {
			return nil, false, err
		}
		switch node := current.(type) {
		case map[string]any:
			value, ok := node[token]
			if !ok {
				return nil, false, nil
			}
			current = value
		case []any:
			if token == "-" {
				return nil, false, errors.New("'-' is not valid for extraction")
			}
			index, err := parseArrayIndex(token)
			if err != nil {
				return nil, false, err
			}
			if index >= len(node) {
				return nil, false, nil
			}
			current = node[index]
		default:
			return nil, false, nil
		}
	}
	return current, true, nil
}

func validateJSONPointer(pointer string) error {
	if len(pointer) > MaxJSONPointerBytes {
		return fmt.Errorf("JSON pointer exceeds %d bytes", MaxJSONPointerBytes)
	}
	if !utf8.ValidString(pointer) || strings.IndexByte(pointer, 0) >= 0 {
		return errors.New("JSON pointer must be valid UTF-8 without U+0000")
	}
	if pointer == "" {
		return nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return errors.New("JSON pointer must be empty or begin with '/'")
	}
	tokens := strings.Split(pointer[1:], "/")
	if len(tokens) > MaxJSONPointerTokens {
		return fmt.Errorf("JSON pointer has more than %d tokens", MaxJSONPointerTokens)
	}
	for _, token := range tokens {
		if _, err := decodePointerToken(token); err != nil {
			return err
		}
	}
	return nil
}

func parseArrayIndex(token string) (int, error) {
	if token == "" || (len(token) > 1 && token[0] == '0') {
		return 0, fmt.Errorf("invalid JSON pointer array index %q", token)
	}
	for i := 0; i < len(token); i++ {
		if token[i] < '0' || token[i] > '9' {
			return 0, fmt.Errorf("invalid JSON pointer array index %q", token)
		}
	}
	index, err := strconv.ParseUint(token, 10, 63)
	if err != nil || uint64(int(index)) != index {
		return 0, fmt.Errorf("JSON pointer array index %q is out of range", token)
	}
	return int(index), nil
}

func decodePointerToken(token string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(token); i++ {
		if token[i] != '~' {
			b.WriteByte(token[i])
			continue
		}
		if i+1 >= len(token) {
			return "", errors.New("invalid terminal '~' in JSON pointer")
		}
		i++
		switch token[i] {
		case '0':
			b.WriteByte('~')
		case '1':
			b.WriteByte('/')
		default:
			return "", fmt.Errorf("invalid JSON pointer escape ~%c", token[i])
		}
	}
	return b.String(), nil
}

func lexicalValue(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case json.Number:
		return v.String(), nil
	case bool:
		return strconv.FormatBool(v), nil
	case nil:
		return "", errors.New("null cannot be converted to a typed scalar")
	default:
		return "", fmt.Errorf("object or array cannot be converted to a typed scalar (%T)", value)
	}
}

func applyTransforms(value string, transforms []string) (string, error) {
	out := value
	for _, transform := range transforms {
		switch transform {
		case "trim":
			out = strings.Trim(out, " \t\r\n")
		case "uppercase":
			out = asciiCase(out, true)
		case "lowercase":
			out = asciiCase(out, false)
		case "decimal_string":
			if !decimalPattern.MatchString(out) {
				return "", fmt.Errorf("%q is not a canonical decimal lexical form", out)
			}
		default:
			return "", fmt.Errorf("unsupported transform %q", transform)
		}
	}
	return out, nil
}

func asciiCase(value string, upper bool) string {
	data := []byte(value)
	for i, char := range data {
		if upper && char >= 'a' && char <= 'z' {
			data[i] = char - ('a' - 'A')
		}
		if !upper && char >= 'A' && char <= 'Z' {
			data[i] = char + ('a' - 'A')
		}
	}
	return string(data)
}

func validateTypedValue(value, valueType string) error {
	switch valueType {
	case "string":
		return nil
	case "decimal":
		if !decimalPattern.MatchString(value) {
			return fmt.Errorf("%q is not a decimal", value)
		}
	case "currency_code":
		if !currencyPattern.MatchString(value) {
			return fmt.Errorf("%q is not a three-letter uppercase currency code", value)
		}
	case "integer":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("%q is not a signed 64-bit integer: %w", value, err)
		}
	case "boolean":
		if value != "true" && value != "false" {
			return fmt.Errorf("%q is not a boolean", value)
		}
	default:
		return fmt.Errorf("unsupported type %q", valueType)
	}
	return nil
}
