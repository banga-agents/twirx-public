// Package twircontract loads the bounded E2 TWIR Core 0.1 operation profile
// and derives transport bindings from it. The JSON source is project input;
// deterministic CBOR and conformance define interchange behavior.
package twircontract

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/cborlite"
	"github.com/typed-web-commons/typed-web/internal/e2format"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	Format           = "tw.contract-set/0.1"
	CoreVersion      = "twir-core/0.1"
	MaxContractBytes = 512 << 10
	MaxOperations    = 64
	MaxFields        = 64
)

type Set struct {
	Format        string      `json:"format"`
	Core          string      `json:"core"`
	ModuleID      string      `json:"module_id"`
	ModuleVersion string      `json:"module_version"`
	Operations    []Operation `json:"operations"`
}

type Operation struct {
	ID                  string       `json:"id"`
	Version             string       `json:"version"`
	OriginID            string       `json:"origin_id"`
	OriginVersion       string       `json:"origin_version"`
	Title               string       `json:"title"`
	Description         string       `json:"description"`
	Resource            string       `json:"resource"`
	Effect              string       `json:"effect"`
	EvidenceRequirement string       `json:"evidence_requirement"`
	NativeReference     string       `json:"native_reference"`
	SemanticReference   string       `json:"semantic_reference"`
	SemanticClosure     []string     `json:"semantic_closure"`
	Input               []Field      `json:"input"`
	Output              []Field      `json:"output"`
	Errors              []TypedError `json:"errors"`
}

type Field struct {
	ID            string   `json:"id"`
	Description   string   `json:"description"`
	Type          string   `json:"type"`
	Required      bool     `json:"required"`
	Allowed       []string `json:"allowed,omitempty"`
	MaxLength     uint64   `json:"max_length,omitempty"`
	NativeTerm    string   `json:"native_term,omitempty"`
	NativeLocator string   `json:"native_locator,omitempty"`
	SemanticTerm  string   `json:"semantic_term,omitempty"`
	Transforms    []string `json:"transforms,omitempty"`
	Mapping       string   `json:"mapping,omitempty"`
}

type TypedError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

func Load(path string) (*Set, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("twircontract: read: %w", err)
	}
	var set Set
	policy := jsonbounded.Policy{MaxBytes: MaxContractBytes, MaxDepth: 12, MaxScalarBytes: 64 << 10, MaxContainerEntries: 512, MaxTokens: 20000}
	if err := jsonbounded.Decode(data, &set, policy, true); err != nil {
		return nil, fmt.Errorf("twircontract: decode: %w", err)
	}
	if err := set.Validate(); err != nil {
		return nil, err
	}
	return &set, nil
}

func (s *Set) Validate() error {
	if s.Format != Format || s.Core != CoreVersion {
		return errors.New("twircontract: unsupported format or core version")
	}
	if err := required("module ID", s.ModuleID); err != nil {
		return err
	}
	if err := required("module version", s.ModuleVersion); err != nil {
		return err
	}
	if len(s.Operations) == 0 || len(s.Operations) > MaxOperations {
		return errors.New("twircontract: invalid operation count")
	}
	seen := make(map[string]struct{}, len(s.Operations))
	for i := range s.Operations {
		op := &s.Operations[i]
		if _, exists := seen[op.ID]; exists {
			return fmt.Errorf("twircontract: duplicate operation %q", op.ID)
		}
		seen[op.ID] = struct{}{}
		if err := op.validate(); err != nil {
			return fmt.Errorf("twircontract: operation %q: %w", op.ID, err)
		}
	}
	return nil
}

func (op *Operation) validate() error {
	for name, value := range map[string]string{
		"ID": op.ID, "version": op.Version, "origin ID": op.OriginID,
		"origin version": op.OriginVersion, "title": op.Title,
		"description": op.Description, "resource": op.Resource,
		"evidence requirement": op.EvidenceRequirement,
		"native reference":     op.NativeReference, "semantic reference": op.SemanticReference,
	} {
		if err := required(name, value); err != nil {
			return err
		}
	}
	if op.Effect != "read" {
		return errors.New("only the read effect is admitted in Genesis")
	}
	if len(op.Output) == 0 || len(op.Output) > MaxFields || len(op.Input) > MaxFields {
		return errors.New("invalid input or output field count")
	}
	if len(op.SemanticClosure) == 0 || len(op.SemanticClosure) > 32 {
		return errors.New("invalid semantic closure")
	}
	if !sort.StringsAreSorted(op.SemanticClosure) {
		return errors.New("semantic closure must be sorted")
	}
	if err := validateFields(op.Input, false); err != nil {
		return fmt.Errorf("input: %w", err)
	}
	if err := validateFields(op.Output, true); err != nil {
		return fmt.Errorf("output: %w", err)
	}
	if len(op.Errors) == 0 || len(op.Errors) > 32 {
		return errors.New("typed errors are required and bounded")
	}
	for _, typed := range op.Errors {
		if err := required("error code", typed.Code); err != nil {
			return err
		}
		if err := required("error description", typed.Description); err != nil {
			return err
		}
	}
	return nil
}

func validateFields(fields []Field, output bool) error {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if _, exists := seen[field.ID]; exists {
			return fmt.Errorf("duplicate field %q", field.ID)
		}
		seen[field.ID] = struct{}{}
		for name, value := range map[string]string{"ID": field.ID, "description": field.Description, "type": field.Type} {
			if err := required(name, value); err != nil {
				return err
			}
		}
		switch field.Type {
		case "string", "integer", "decimal", "currency_code", "list:string":
		default:
			return fmt.Errorf("field %q has unsupported type %q", field.ID, field.Type)
		}
		if field.MaxLength > 65536 {
			return fmt.Errorf("field %q max length exceeds bound", field.ID)
		}
		if len(field.Allowed) > 128 || len(field.Transforms) > e2format.MaxTransforms {
			return fmt.Errorf("field %q list exceeds bound", field.ID)
		}
		if output {
			for name, value := range map[string]string{"native term": field.NativeTerm, "native locator": field.NativeLocator, "semantic term": field.SemanticTerm, "mapping": field.Mapping} {
				if err := required(name, value); err != nil {
					return err
				}
			}
		} else if field.NativeTerm != "" || field.NativeLocator != "" || field.SemanticTerm != "" || field.Mapping != "" || len(field.Transforms) != 0 {
			return fmt.Errorf("input field %q cannot contain extraction metadata", field.ID)
		}
	}
	return nil
}

func required(name, value string) error {
	if value == "" || len(value) > e2format.MaxTextBytes || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%s is empty, contains NUL, or exceeds bounds", name)
	}
	return nil
}

func (s *Set) Find(operationID string) (*Operation, error) {
	for i := range s.Operations {
		if s.Operations[i].ID == operationID {
			return &s.Operations[i], nil
		}
	}
	return nil, fmt.Errorf("twircontract: unknown operation %q", operationID)
}

// MarshalOperation returns the deterministic contract artifact for one
// operation. Empty extraction metadata remains explicit in input fields.
func (s *Set) MarshalOperation(op *Operation) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if found, err := s.Find(op.ID); err != nil || found.Version != op.Version {
		return nil, errors.New("twircontract: operation is not a member of set")
	}
	var enc cborlite.Encoder
	enc.Array(18)
	enc.Text(CoreVersion)
	enc.Text(s.ModuleID)
	enc.Text(s.ModuleVersion)
	enc.Text(op.ID)
	enc.Text(op.Version)
	enc.Text(op.OriginID)
	enc.Text(op.OriginVersion)
	enc.Text(op.Title)
	enc.Text(op.Description)
	enc.Text(op.Resource)
	enc.Text(op.Effect)
	enc.Text(op.EvidenceRequirement)
	enc.Text(op.NativeReference)
	enc.Text(op.SemanticReference)
	encodeTextArray(&enc, op.SemanticClosure)
	encodeFields(&enc, op.Input)
	encodeFields(&enc, op.Output)
	enc.Array(uint64(len(op.Errors)))
	for _, typed := range op.Errors {
		enc.Array(2)
		enc.Text(typed.Code)
		enc.Text(typed.Description)
	}
	return enc.Bytes(), nil
}

func MarshalSemanticClosure(values []string) ([]byte, error) {
	if len(values) == 0 || len(values) > 32 || !sort.StringsAreSorted(values) {
		return nil, errors.New("twircontract: semantic closure must be non-empty, bounded, and sorted")
	}
	var enc cborlite.Encoder
	enc.Array(2)
	enc.Text("tw.semantic-closure/0.1")
	encodeTextArray(&enc, values)
	return enc.Bytes(), nil
}

func MarshalAdapterDescriptor(op *Operation) ([]byte, error) {
	if err := op.validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(5)
	enc.Text("tw.adapter-descriptor/0.2")
	enc.Text(op.OriginID)
	enc.Text(op.ID)
	enc.Text(op.Version)
	encodeFields(&enc, op.Output)
	return enc.Bytes(), nil
}

func encodeFields(enc *cborlite.Encoder, fields []Field) {
	enc.Array(uint64(len(fields)))
	for _, field := range fields {
		enc.Array(11)
		enc.Text(field.ID)
		enc.Text(field.Description)
		enc.Text(field.Type)
		if field.Required {
			enc.Uint(1)
		} else {
			enc.Uint(0)
		}
		encodeTextArray(enc, field.Allowed)
		enc.Uint(field.MaxLength)
		enc.Text(field.NativeTerm)
		enc.Text(field.NativeLocator)
		enc.Text(field.SemanticTerm)
		encodeTextArray(enc, field.Transforms)
		enc.Text(field.Mapping)
	}
}

func encodeTextArray(enc *cborlite.Encoder, values []string) {
	enc.Array(uint64(len(values)))
	for _, value := range values {
		enc.Text(value)
	}
}

func CanonicalInput(op *Operation, input map[string]string) ([]byte, error) {
	if err := ValidateInput(op, input); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(2)
	enc.Text("tw.input/0.1")
	enc.Array(uint64(len(op.Input)))
	for _, field := range op.Input {
		enc.Array(2)
		enc.Text(field.ID)
		enc.Text(input[field.ID])
	}
	return enc.Bytes(), nil
}

func UnmarshalInput(op *Operation, data []byte) (map[string]string, error) {
	if len(data) == 0 || len(data) > MaxContractBytes {
		return nil, errors.New("twircontract: input artifact outside bounds")
	}
	dec := cborlite.NewDecoder(data)
	top, err := dec.Array()
	if err != nil || top != 2 {
		return nil, errors.New("twircontract: invalid input artifact")
	}
	version, err := dec.Text(64)
	if err != nil || version != "tw.input/0.1" {
		return nil, errors.New("twircontract: unsupported input artifact")
	}
	count, err := dec.Array()
	if err != nil || count != uint64(len(op.Input)) {
		return nil, errors.New("twircontract: input field count mismatch")
	}
	input := make(map[string]string, len(op.Input))
	for i, field := range op.Input {
		entry, entryErr := dec.Array()
		if entryErr != nil || entry != 2 {
			return nil, errors.New("twircontract: invalid input field entry")
		}
		fieldID, fieldErr := dec.Text(256)
		if fieldErr != nil || fieldID != field.ID {
			return nil, fmt.Errorf("twircontract: input field %d identity mismatch", i)
		}
		value, valueErr := dec.Text(65536)
		if valueErr != nil {
			return nil, valueErr
		}
		input[field.ID] = value
	}
	if dec.Remaining() != 0 {
		return nil, errors.New("twircontract: trailing input bytes")
	}
	if err := ValidateInput(op, input); err != nil {
		return nil, err
	}
	return input, nil
}

func ValidateInput(op *Operation, input map[string]string) error {
	if len(input) > len(op.Input) {
		return errors.New("twircontract: unexpected input field")
	}
	allowedFields := make(map[string]Field, len(op.Input))
	for _, field := range op.Input {
		allowedFields[field.ID] = field
	}
	for key := range input {
		if _, exists := allowedFields[key]; !exists {
			return fmt.Errorf("twircontract: unexpected input field %q", key)
		}
	}
	for _, field := range op.Input {
		value, exists := input[field.ID]
		if field.Required && (!exists || value == "") {
			return fmt.Errorf("twircontract: required input %q is missing", field.ID)
		}
		if !exists {
			continue
		}
		if field.MaxLength > 0 && uint64(len(value)) > field.MaxLength {
			return fmt.Errorf("twircontract: input %q exceeds length limit", field.ID)
		}
		if len(field.Allowed) > 0 && !contains(field.Allowed, value) {
			return fmt.Errorf("twircontract: input %q is outside the admitted values", field.ID)
		}
	}
	return nil
}

// NormalizeInput converts transport-level JSON values into the canonical
// lexical input representation. It is shared by HTTP and MCP bindings.
func NormalizeInput(op *Operation, values map[string]any) (map[string]string, error) {
	if len(values) > len(op.Input) {
		return nil, errors.New("twircontract: unexpected input field")
	}
	specs := make(map[string]Field, len(op.Input))
	for _, field := range op.Input {
		specs[field.ID] = field
	}
	input := make(map[string]string, len(values))
	for key, value := range values {
		field, exists := specs[key]
		if !exists {
			return nil, fmt.Errorf("twircontract: unexpected input field %q", key)
		}
		switch field.Type {
		case "integer":
			number, ok := value.(json.Number)
			if !ok {
				return nil, fmt.Errorf("twircontract: input %q must be a JSON integer", key)
			}
			if _, err := number.Int64(); err != nil {
				return nil, fmt.Errorf("twircontract: input %q must be a signed 64-bit JSON integer", key)
			}
			input[key] = number.String()
		default:
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("twircontract: input %q must be a JSON string", key)
			}
			input[key] = text
		}
	}
	if err := ValidateInput(op, input); err != nil {
		return nil, err
	}
	return input, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func Digest(data []byte) [32]byte { return sha256.Sum256(data) }

// JSONSchema derives a bounded invocation schema from the canonical contract.
func JSONSchema(op *Operation) ([]byte, error) {
	properties := make(map[string]any, len(op.Input))
	requiredFields := make([]string, 0, len(op.Input))
	for _, field := range op.Input {
		jsonType := "string"
		if field.Type == "integer" {
			jsonType = "integer"
		}
		property := map[string]any{"type": jsonType, "description": field.Description}
		if len(field.Allowed) > 0 {
			if field.Type == "integer" {
				allowed := make([]int64, 0, len(field.Allowed))
				for _, lexical := range field.Allowed {
					value, err := strconv.ParseInt(lexical, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("twircontract: invalid integer allowlist for %s", field.ID)
					}
					allowed = append(allowed, value)
				}
				property["enum"] = allowed
			} else {
				property["enum"] = field.Allowed
			}
		}
		if field.MaxLength > 0 && jsonType == "string" {
			property["maxLength"] = field.MaxLength
		}
		properties[field.ID] = property
		if field.Required {
			requiredFields = append(requiredFields, field.ID)
		}
	}
	schema := map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "$id": "urn:twirx:operation:" + op.ID + ":" + op.Version, "title": op.Title, "description": op.Description, "type": "object", "additionalProperties": false, "properties": properties, "required": requiredFields}
	return json.MarshalIndent(schema, "", "  ")
}
