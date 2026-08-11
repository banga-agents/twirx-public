// Package e2format implements the bounded deterministic artifacts used by
// Engineering Gate E2. The format is defined by schemas and conformance
// vectors; this Go package is one implementation, not the normative protocol.
package e2format

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/cborlite"
)

const (
	ResultVersion     = "tw.result/0.2"
	MaxResultBytes    = 1 << 20
	MaxFields         = 128
	MaxTransforms     = 16
	MaxTextBytes      = 16 << 10
	MaxLexicalBytes   = 256 << 10
	ResultArrayLength = 17
	FieldArrayLength  = 12
)

var (
	ErrInvalidResult = errors.New("e2format: invalid result")
	ErrTrailingBytes = errors.New("e2format: trailing bytes")
)

// Field binds a typed semantic value to its source-native statement and the
// declared derivation. Presence is explicit so an empty resolved lexical value
// cannot be confused with unresolved optional content.
type Field struct {
	ID              string
	Status          string
	NativeTerm      string
	NativeLocator   string
	NativePresent   bool
	NativeLexical   string
	SemanticTerm    string
	SemanticType    string
	SemanticPresent bool
	SemanticLexical string
	Transforms      []string
	Mapping         string
}

// Result is the canonical typed-result envelope. Its result digest and the
// enclosing manifest's bundle ID are external, as specified by ADR 003.
type Result struct {
	Version               string
	InvocationID          string
	OriginID              string
	OriginVersion         string
	OperationID           string
	OperationVersion      string
	Effect                string
	Status                string
	ObservedAt            string
	InputDigest           [32]byte
	ObservationDigest     [32]byte
	TransportDigest       [32]byte
	AdapterDigest         [32]byte
	ContractDigest        [32]byte
	SemanticClosureDigest [32]byte
	Fields                []Field
}

func (r Result) Validate() error {
	if r.Version != ResultVersion {
		return fmt.Errorf("%w: version must be %q", ErrInvalidResult, ResultVersion)
	}
	for name, value := range map[string]string{
		"invocation ID": r.InvocationID, "origin ID": r.OriginID,
		"origin version": r.OriginVersion, "operation ID": r.OperationID,
		"operation version": r.OperationVersion, "observed at": r.ObservedAt,
	} {
		if err := validateText(name, value, MaxTextBytes, false); err != nil {
			return err
		}
	}
	if r.Effect != "read" {
		return fmt.Errorf("%w: effect must be read", ErrInvalidResult)
	}
	if r.Status != "resolved" && r.Status != "partial" {
		return fmt.Errorf("%w: status must be resolved or partial", ErrInvalidResult)
	}
	if len(r.Fields) == 0 || len(r.Fields) > MaxFields {
		return fmt.Errorf("%w: field count must be between 1 and %d", ErrInvalidResult, MaxFields)
	}
	seen := make(map[string]struct{}, len(r.Fields))
	partial := false
	for i, field := range r.Fields {
		if err := field.validate(); err != nil {
			return fmt.Errorf("%w: field %d: %v", ErrInvalidResult, i, err)
		}
		if _, exists := seen[field.ID]; exists {
			return fmt.Errorf("%w: duplicate field ID %q", ErrInvalidResult, field.ID)
		}
		seen[field.ID] = struct{}{}
		if field.Status == "unresolved" {
			partial = true
		}
	}
	if partial != (r.Status == "partial") {
		return fmt.Errorf("%w: result status does not match field resolution", ErrInvalidResult)
	}
	return nil
}

func (f Field) validate() error {
	for name, value := range map[string]string{
		"ID": f.ID, "status": f.Status, "native term": f.NativeTerm,
		"native locator": f.NativeLocator, "semantic term": f.SemanticTerm,
		"semantic type": f.SemanticType, "mapping": f.Mapping,
	} {
		if err := validateText("field "+name, value, MaxTextBytes, false); err != nil {
			return err
		}
	}
	if f.Status != "resolved" && f.Status != "unresolved" {
		return errors.New("status must be resolved or unresolved")
	}
	if f.Status == "resolved" {
		if !f.NativePresent || !f.SemanticPresent {
			return errors.New("resolved field must contain native and semantic values")
		}
	} else if f.NativePresent || f.SemanticPresent || f.NativeLexical != "" || f.SemanticLexical != "" {
		return errors.New("unresolved field must omit lexical values")
	}
	if f.NativePresent {
		if err := validateText("native lexical", f.NativeLexical, MaxLexicalBytes, true); err != nil {
			return err
		}
	}
	if f.SemanticPresent {
		if err := validateText("semantic lexical", f.SemanticLexical, MaxLexicalBytes, true); err != nil {
			return err
		}
	}
	if len(f.Transforms) > MaxTransforms {
		return fmt.Errorf("too many transforms: %d", len(f.Transforms))
	}
	for _, transform := range f.Transforms {
		if err := validateText("transform", transform, MaxTextBytes, false); err != nil {
			return err
		}
	}
	return nil
}

func validateText(name, value string, max int, allowEmpty bool) error {
	if (!allowEmpty && value == "") || len(value) > max || !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%w: %s is empty, invalid UTF-8, contains NUL, or exceeds %d bytes", ErrInvalidResult, name, max)
	}
	return nil
}

func MarshalResult(r Result) ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(ResultArrayLength)
	enc.Text(r.Version)
	enc.Text(r.InvocationID)
	enc.Text(r.OriginID)
	enc.Text(r.OriginVersion)
	enc.Text(r.OperationID)
	enc.Text(r.OperationVersion)
	enc.Text(r.Effect)
	enc.Text(r.Status)
	enc.Text(r.ObservedAt)
	encodeDigest(&enc, r.InputDigest)
	encodeDigest(&enc, r.ObservationDigest)
	encodeDigest(&enc, r.TransportDigest)
	encodeDigest(&enc, r.AdapterDigest)
	encodeDigest(&enc, r.ContractDigest)
	encodeDigest(&enc, r.SemanticClosureDigest)
	enc.Array(uint64(len(r.Fields)))
	for _, field := range r.Fields {
		marshalField(&enc, field)
	}
	// Reserved versioned extension array. E2 requires it to be empty.
	enc.Array(0)
	encoded := enc.Bytes()
	if len(encoded) > MaxResultBytes {
		return nil, fmt.Errorf("%w: encoded result exceeds %d bytes", ErrInvalidResult, MaxResultBytes)
	}
	return encoded, nil
}

func marshalField(enc *cborlite.Encoder, field Field) {
	enc.Array(FieldArrayLength)
	enc.Text(field.ID)
	enc.Text(field.Status)
	enc.Text(field.NativeTerm)
	enc.Text(field.NativeLocator)
	enc.Uint(boolUint(field.NativePresent))
	enc.Text(field.NativeLexical)
	enc.Text(field.SemanticTerm)
	enc.Text(field.SemanticType)
	enc.Uint(boolUint(field.SemanticPresent))
	enc.Text(field.SemanticLexical)
	enc.Array(uint64(len(field.Transforms)))
	for _, transform := range field.Transforms {
		enc.Text(transform)
	}
	enc.Text(field.Mapping)
}

func UnmarshalResult(data []byte) (Result, error) {
	var result Result
	if len(data) == 0 || len(data) > MaxResultBytes {
		return result, fmt.Errorf("%w: byte length outside bounds", ErrInvalidResult)
	}
	dec := cborlite.NewDecoder(data)
	n, err := dec.Array()
	if err != nil || n != ResultArrayLength {
		return result, fmt.Errorf("%w: top-level array: %v", ErrInvalidResult, err)
	}
	if result.Version, err = dec.Text(MaxTextBytes); err != nil {
		return result, err
	}
	if result.InvocationID, err = dec.Text(MaxTextBytes); err != nil {
		return result, err
	}
	if result.OriginID, err = dec.Text(MaxTextBytes); err != nil {
		return result, err
	}
	if result.OriginVersion, err = dec.Text(MaxTextBytes); err != nil {
		return result, err
	}
	if result.OperationID, err = dec.Text(MaxTextBytes); err != nil {
		return result, err
	}
	if result.OperationVersion, err = dec.Text(MaxTextBytes); err != nil {
		return result, err
	}
	if result.Effect, err = dec.Text(MaxTextBytes); err != nil {
		return result, err
	}
	if result.Status, err = dec.Text(MaxTextBytes); err != nil {
		return result, err
	}
	if result.ObservedAt, err = dec.Text(MaxTextBytes); err != nil {
		return result, err
	}
	if result.InputDigest, err = decodeDigest(dec); err != nil {
		return result, err
	}
	if result.ObservationDigest, err = decodeDigest(dec); err != nil {
		return result, err
	}
	if result.TransportDigest, err = decodeDigest(dec); err != nil {
		return result, err
	}
	if result.AdapterDigest, err = decodeDigest(dec); err != nil {
		return result, err
	}
	if result.ContractDigest, err = decodeDigest(dec); err != nil {
		return result, err
	}
	if result.SemanticClosureDigest, err = decodeDigest(dec); err != nil {
		return result, err
	}
	fieldCount, err := dec.Array()
	if err != nil || fieldCount == 0 || fieldCount > MaxFields {
		return result, fmt.Errorf("%w: invalid field array", ErrInvalidResult)
	}
	result.Fields = make([]Field, 0, fieldCount)
	for i := uint64(0); i < fieldCount; i++ {
		field, decodeErr := unmarshalField(dec)
		if decodeErr != nil {
			return result, fmt.Errorf("field %d: %w", i, decodeErr)
		}
		result.Fields = append(result.Fields, field)
	}
	extensions, err := dec.Array()
	if err != nil || extensions != 0 {
		return result, fmt.Errorf("%w: extensions must be an empty array", ErrInvalidResult)
	}
	if dec.Remaining() != 0 {
		return result, ErrTrailingBytes
	}
	if err := result.Validate(); err != nil {
		return result, err
	}
	return result, nil
}

func unmarshalField(dec *cborlite.Decoder) (Field, error) {
	var field Field
	n, err := dec.Array()
	if err != nil || n != FieldArrayLength {
		return field, fmt.Errorf("invalid field array: %v", err)
	}
	if field.ID, err = dec.Text(MaxTextBytes); err != nil {
		return field, err
	}
	if field.Status, err = dec.Text(MaxTextBytes); err != nil {
		return field, err
	}
	if field.NativeTerm, err = dec.Text(MaxTextBytes); err != nil {
		return field, err
	}
	if field.NativeLocator, err = dec.Text(MaxTextBytes); err != nil {
		return field, err
	}
	if field.NativePresent, err = decodeBool(dec); err != nil {
		return field, err
	}
	if field.NativeLexical, err = dec.Text(MaxLexicalBytes); err != nil {
		return field, err
	}
	if field.SemanticTerm, err = dec.Text(MaxTextBytes); err != nil {
		return field, err
	}
	if field.SemanticType, err = dec.Text(MaxTextBytes); err != nil {
		return field, err
	}
	if field.SemanticPresent, err = decodeBool(dec); err != nil {
		return field, err
	}
	if field.SemanticLexical, err = dec.Text(MaxLexicalBytes); err != nil {
		return field, err
	}
	count, err := dec.Array()
	if err != nil || count > MaxTransforms {
		return field, errors.New("invalid transforms array")
	}
	field.Transforms = make([]string, 0, count)
	for i := uint64(0); i < count; i++ {
		value, textErr := dec.Text(MaxTextBytes)
		if textErr != nil {
			return field, textErr
		}
		field.Transforms = append(field.Transforms, value)
	}
	if field.Mapping, err = dec.Text(MaxTextBytes); err != nil {
		return field, err
	}
	return field, nil
}

func Digest(data []byte) [32]byte { return sha256.Sum256(data) }

func DigestReference(digest [32]byte) string { return "sha256:" + hex.EncodeToString(digest[:]) }

func ParseDigestReference(value string) ([32]byte, error) {
	var digest [32]byte
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return digest, errors.New("e2format: invalid SHA-256 reference")
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	if err != nil {
		return digest, errors.New("e2format: invalid SHA-256 reference")
	}
	copy(digest[:], decoded)
	return digest, nil
}

func encodeDigest(enc *cborlite.Encoder, digest [32]byte) { enc.Bytestring(digest[:]) }

func decodeDigest(dec *cborlite.Decoder) ([32]byte, error) {
	var digest [32]byte
	value, err := dec.Bytestring(32)
	if err != nil || len(value) != len(digest) {
		return digest, errors.New("e2format: digest must be 32 bytes")
	}
	copy(digest[:], value)
	return digest, nil
}

func boolUint(value bool) uint64 {
	if value {
		return 1
	}
	return 0
}

func decodeBool(dec *cborlite.Decoder) (bool, error) {
	value, err := dec.Uint()
	if err != nil || value > 1 {
		return false, errors.New("e2format: boolean must be 0 or 1")
	}
	return value == 1, nil
}
