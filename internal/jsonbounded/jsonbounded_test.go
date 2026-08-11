package jsonbounded

import (
	"errors"
	"strings"
	"testing"
)

var testPolicy = Policy{
	MaxBytes:            1024,
	MaxDepth:            3,
	MaxScalarBytes:      8,
	MaxContainerEntries: 3,
	MaxTokens:           32,
}

func TestDecodeRejectsDuplicateKeysAtEveryDepth(t *testing.T) {
	for _, input := range []string{
		`{"a":1,"a":2}`,
		`{"outer":{"a":1,"a":2}}`,
	} {
		var value any
		err := Decode([]byte(input), &value, testPolicy, false)
		var bounded *Error
		if !errors.As(err, &bounded) || bounded.Kind != KindDuplicateKey {
			t.Fatalf("input=%s err=%v", input, err)
		}
	}
}

func TestDecodeRejectsTrailingValue(t *testing.T) {
	var value any
	err := Decode([]byte(`{"a":1} {"b":2}`), &value, testPolicy, false)
	var bounded *Error
	if !errors.As(err, &bounded) || bounded.Kind != KindTrailingData {
		t.Fatalf("err=%v", err)
	}
}

func TestDecodeAppliesResourceBounds(t *testing.T) {
	tests := []struct {
		name  string
		input string
		kind  Kind
	}{
		{name: "depth", input: `[[[[]]]]`, kind: KindDepthExceeded},
		{name: "scalar", input: `"123456789"`, kind: KindScalarTooLarge},
		{name: "container", input: `[1,2,3,4]`, kind: KindContainerTooLarge},
		{name: "bytes", input: strings.Repeat(" ", 1025), kind: KindDocumentTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var value any
			err := Decode([]byte(test.input), &value, testPolicy, false)
			var bounded *Error
			if !errors.As(err, &bounded) || bounded.Kind != test.kind {
				t.Fatalf("err=%v want kind=%s", err, test.kind)
			}
		})
	}
}

func TestDecodePreservesNumberLexicalForm(t *testing.T) {
	var value any
	if err := Decode([]byte(`1.2300`), &value, testPolicy, false); err != nil {
		t.Fatal(err)
	}
	if value.(interface{ String() string }).String() != "1.2300" {
		t.Fatalf("value=%v", value)
	}
}

func TestDecodeRejectsUnpairedSurrogateEscapes(t *testing.T) {
	for _, input := range []string{
		`{"value":"\uD800"}`,
		`{"value":"\uDC00"}`,
		`{"value":"\uD800\u0041"}`,
	} {
		var value any
		err := Decode([]byte(input), &value, testPolicy, false)
		var bounded *Error
		if !errors.As(err, &bounded) || bounded.Kind != KindInvalidSyntax {
			t.Fatalf("input=%s err=%v", input, err)
		}
	}
}

func TestDecodeAcceptsPairedSurrogateEscapes(t *testing.T) {
	var value map[string]string
	if err := Decode([]byte(`{"value":"\uD83D\uDE00"}`), &value, testPolicy, false); err != nil {
		t.Fatal(err)
	}
	if value["value"] != "😀" {
		t.Fatalf("value=%q", value["value"])
	}
}
