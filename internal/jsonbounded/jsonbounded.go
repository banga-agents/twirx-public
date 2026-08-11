// Package jsonbounded validates and decodes the bounded JSON profile used by
// Genesis adapters. It rejects duplicate keys and resource-ambiguous inputs.
package jsonbounded

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// Kind identifies a stable class of bounded JSON failure.
type Kind string

const (
	KindInvalidSyntax     Kind = "invalid_syntax"
	KindTrailingData      Kind = "trailing_data"
	KindDuplicateKey      Kind = "duplicate_key"
	KindDocumentTooLarge  Kind = "document_too_large"
	KindDepthExceeded     Kind = "depth_exceeded"
	KindScalarTooLarge    Kind = "scalar_too_large"
	KindContainerTooLarge Kind = "container_too_large"
	KindTokenLimit        Kind = "token_limit"
)

// Error reports a typed parser failure and the decoder offset where available.
type Error struct {
	Kind   Kind
	Offset int64
	Detail string
}

func (e *Error) Error() string {
	if e.Offset > 0 {
		return fmt.Sprintf("jsonbounded: %s at byte %d: %s", e.Kind, e.Offset, e.Detail)
	}
	return fmt.Sprintf("jsonbounded: %s: %s", e.Kind, e.Detail)
}

// Policy bounds one JSON document. Byte and count limits must be positive.
type Policy struct {
	MaxBytes            int
	MaxDepth            int
	MaxScalarBytes      int
	MaxContainerEntries int
	MaxTokens           int
}

// Decode validates data and then decodes it into dst. Numbers are preserved as
// json.Number. Unknown struct fields are rejected when disallowUnknown is true.
func Decode(data []byte, dst any, policy Policy, disallowUnknown bool) error {
	if dst == nil {
		return &Error{Kind: KindInvalidSyntax, Detail: "destination is required"}
	}
	if err := policy.validate(); err != nil {
		return err
	}
	if len(data) > policy.MaxBytes {
		return &Error{Kind: KindDocumentTooLarge, Detail: fmt.Sprintf("size %d exceeds %d bytes", len(data), policy.MaxBytes)}
	}
	if !utf8.Valid(data) {
		return &Error{Kind: KindInvalidSyntax, Detail: "document is not valid UTF-8"}
	}
	if err := validateUnicodeEscapes(data); err != nil {
		return err
	}
	validator := newValidator(data, policy)
	if err := validator.value(0); err != nil {
		return err
	}
	if token, err := validator.token(); err != io.EOF {
		if err == nil {
			return &Error{Kind: KindTrailingData, Offset: validator.decoder.InputOffset(), Detail: fmt.Sprintf("unexpected token %v", token)}
		}
		return &Error{Kind: KindTrailingData, Offset: validator.decoder.InputOffset(), Detail: err.Error()}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if disallowUnknown {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(dst); err != nil {
		return &Error{Kind: KindInvalidSyntax, Offset: decoder.InputOffset(), Detail: err.Error()}
	}
	return nil
}

// validateUnicodeEscapes rejects lone UTF-16 surrogate escapes. encoding/json
// replaces them with U+FFFD, which would otherwise lose the source scalar and
// produce implementation-dependent lexical values.
func validateUnicodeEscapes(data []byte) error {
	inString := false
	for offset := 0; offset < len(data); offset++ {
		char := data[offset]
		if !inString {
			if char == '"' {
				inString = true
			}
			continue
		}
		if char == '"' {
			inString = false
			continue
		}
		if char != '\\' {
			continue
		}
		escapeOffset := offset
		offset++
		if offset >= len(data) || data[offset] != 'u' {
			continue
		}
		codeUnit, ok := parseHexCodeUnit(data, offset+1)
		if !ok {
			continue
		}
		offset += 4
		switch {
		case codeUnit >= 0xdc00 && codeUnit <= 0xdfff:
			return &Error{Kind: KindInvalidSyntax, Offset: int64(escapeOffset + 1), Detail: "unpaired low-surrogate Unicode escape"}
		case codeUnit >= 0xd800 && codeUnit <= 0xdbff:
			if offset+6 >= len(data) || data[offset+1] != '\\' || data[offset+2] != 'u' {
				return &Error{Kind: KindInvalidSyntax, Offset: int64(escapeOffset + 1), Detail: "unpaired high-surrogate Unicode escape"}
			}
			low, lowOK := parseHexCodeUnit(data, offset+3)
			if !lowOK || low < 0xdc00 || low > 0xdfff {
				return &Error{Kind: KindInvalidSyntax, Offset: int64(escapeOffset + 1), Detail: "unpaired high-surrogate Unicode escape"}
			}
			offset += 6
		}
	}
	return nil
}

func parseHexCodeUnit(data []byte, start int) (uint16, bool) {
	if start < 0 || start+4 > len(data) {
		return 0, false
	}
	var value uint16
	for _, char := range data[start : start+4] {
		value <<= 4
		switch {
		case char >= '0' && char <= '9':
			value |= uint16(char - '0')
		case char >= 'a' && char <= 'f':
			value |= uint16(char-'a') + 10
		case char >= 'A' && char <= 'F':
			value |= uint16(char-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}

func (p Policy) validate() error {
	if p.MaxBytes <= 0 || p.MaxDepth <= 0 || p.MaxScalarBytes <= 0 || p.MaxContainerEntries <= 0 || p.MaxTokens <= 0 {
		return &Error{Kind: KindInvalidSyntax, Detail: "all policy limits must be positive"}
	}
	return nil
}

type validator struct {
	decoder *json.Decoder
	policy  Policy
	tokens  int
}

func newValidator(data []byte, policy Policy) *validator {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return &validator{decoder: decoder, policy: policy}
}

func (v *validator) token() (json.Token, error) {
	token, err := v.decoder.Token()
	if err != nil {
		return nil, err
	}
	v.tokens++
	if v.tokens > v.policy.MaxTokens {
		return nil, &Error{Kind: KindTokenLimit, Offset: v.decoder.InputOffset(), Detail: fmt.Sprintf("more than %d tokens", v.policy.MaxTokens)}
	}
	return token, nil
}

func (v *validator) value(depth int) error {
	token, err := v.token()
	if err != nil {
		if bounded, ok := err.(*Error); ok {
			return bounded
		}
		return &Error{Kind: KindInvalidSyntax, Offset: v.decoder.InputOffset(), Detail: err.Error()}
	}
	switch value := token.(type) {
	case json.Delim:
		if depth >= v.policy.MaxDepth {
			return &Error{Kind: KindDepthExceeded, Offset: v.decoder.InputOffset(), Detail: fmt.Sprintf("more than %d nested containers", v.policy.MaxDepth)}
		}
		switch value {
		case '{':
			return v.object(depth + 1)
		case '[':
			return v.array(depth + 1)
		default:
			return &Error{Kind: KindInvalidSyntax, Offset: v.decoder.InputOffset(), Detail: fmt.Sprintf("unexpected delimiter %q", value)}
		}
	case string:
		return v.scalar(len(value))
	case json.Number:
		return v.scalar(len(value.String()))
	case bool, nil:
		return nil
	default:
		return &Error{Kind: KindInvalidSyntax, Offset: v.decoder.InputOffset(), Detail: fmt.Sprintf("unexpected token type %T", token)}
	}
}

func (v *validator) object(depth int) error {
	seen := make(map[string]struct{})
	entries := 0
	for v.decoder.More() {
		keyToken, err := v.token()
		if err != nil {
			return v.wrap(err)
		}
		key, ok := keyToken.(string)
		if !ok {
			return &Error{Kind: KindInvalidSyntax, Offset: v.decoder.InputOffset(), Detail: "object key is not a string"}
		}
		if err := v.scalar(len(key)); err != nil {
			return err
		}
		if _, exists := seen[key]; exists {
			return &Error{Kind: KindDuplicateKey, Offset: v.decoder.InputOffset(), Detail: fmt.Sprintf("duplicate object key %q", key)}
		}
		seen[key] = struct{}{}
		entries++
		if entries > v.policy.MaxContainerEntries {
			return &Error{Kind: KindContainerTooLarge, Offset: v.decoder.InputOffset(), Detail: fmt.Sprintf("object has more than %d entries", v.policy.MaxContainerEntries)}
		}
		if err := v.value(depth); err != nil {
			return err
		}
	}
	return v.close('}')
}

func (v *validator) array(depth int) error {
	entries := 0
	for v.decoder.More() {
		entries++
		if entries > v.policy.MaxContainerEntries {
			return &Error{Kind: KindContainerTooLarge, Offset: v.decoder.InputOffset(), Detail: fmt.Sprintf("array has more than %d entries", v.policy.MaxContainerEntries)}
		}
		if err := v.value(depth); err != nil {
			return err
		}
	}
	return v.close(']')
}

func (v *validator) close(expected json.Delim) error {
	token, err := v.token()
	if err != nil {
		return v.wrap(err)
	}
	if token != expected {
		return &Error{Kind: KindInvalidSyntax, Offset: v.decoder.InputOffset(), Detail: fmt.Sprintf("expected %q, got %v", expected, token)}
	}
	return nil
}

func (v *validator) scalar(length int) error {
	if length > v.policy.MaxScalarBytes {
		return &Error{Kind: KindScalarTooLarge, Offset: v.decoder.InputOffset(), Detail: fmt.Sprintf("decoded scalar size %d exceeds %d bytes", length, v.policy.MaxScalarBytes)}
	}
	return nil
}

func (v *validator) wrap(err error) error {
	if bounded, ok := err.(*Error); ok {
		return bounded
	}
	return &Error{Kind: KindInvalidSyntax, Offset: v.decoder.InputOffset(), Detail: err.Error()}
}
