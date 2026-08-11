package e2format

import (
	"errors"
	"testing"
)

func validResult() Result {
	digest := Digest([]byte("fixture"))
	return Result{
		Version: ResultVersion, InvocationID: "invocation-1",
		OriginID: "twirx-project", OriginVersion: "1",
		OperationID: "project.getStatus", OperationVersion: "1",
		Effect: "read", Status: "resolved", ObservedAt: "2026-08-10T00:00:00Z",
		InputDigest: digest, ObservationDigest: digest, TransportDigest: digest,
		AdapterDigest: digest, ContractDigest: digest, SemanticClosureDigest: digest,
		Fields: []Field{{
			ID: "status", Status: "resolved", NativeTerm: "status",
			NativeLocator: "/status", NativePresent: true,
			NativeLexical: "genesis_preview", SemanticTerm: "project:status",
			SemanticType: "string", SemanticPresent: true,
			SemanticLexical: "genesis_preview", Mapping: "identity",
		}},
	}
}

func TestResultRoundTripAndDigest(t *testing.T) {
	want := validResult()
	encoded, err := MarshalResult(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnmarshalResult(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fields[0].NativeLexical != "genesis_preview" || got.OperationID != want.OperationID {
		t.Fatalf("unexpected round trip: %#v", got)
	}
	if DigestReference(Digest(encoded)) == "" {
		t.Fatal("empty digest reference")
	}
}

func TestResultRejectsMalformedAndAmbiguousValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Result)
	}{
		{"wrong version", func(r *Result) { r.Version = "tw.result/9" }},
		{"write effect", func(r *Result) { r.Effect = "write" }},
		{"duplicate field", func(r *Result) { r.Fields = append(r.Fields, r.Fields[0]) }},
		{"unresolved required value", func(r *Result) {
			r.Fields[0].Status = "unresolved"
			r.Fields[0].NativePresent = false
			r.Fields[0].SemanticPresent = false
			r.Fields[0].NativeLexical = ""
			r.Fields[0].SemanticLexical = ""
		}},
		{"NUL", func(r *Result) { r.Fields[0].NativeLexical = "bad\x00value" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value := validResult()
			tc.mutate(&value)
			if _, err := MarshalResult(value); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
	encoded, err := MarshalResult(validResult())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UnmarshalResult(append(encoded, 0)); !errors.Is(err, ErrTrailingBytes) {
		t.Fatalf("got %v", err)
	}
}

func TestOptionalUnresolvedIsExplicit(t *testing.T) {
	value := validResult()
	value.Status = "partial"
	value.Fields[0].Status = "unresolved"
	value.Fields[0].NativePresent = false
	value.Fields[0].SemanticPresent = false
	value.Fields[0].NativeLexical = ""
	value.Fields[0].SemanticLexical = ""
	encoded, err := MarshalResult(value)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnmarshalResult(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Fields[0].NativePresent {
		t.Fatal("unresolved became present")
	}
}

func FuzzUnmarshalResult(f *testing.F) {
	encoded, err := MarshalResult(validResult())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(encoded)
	f.Add([]byte{0x9f, 0xff})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		result, decodeErr := UnmarshalResult(data)
		if decodeErr == nil {
			reencoded, encodeErr := MarshalResult(result)
			if encodeErr != nil {
				t.Fatalf("accepted result cannot be re-encoded: %v", encodeErr)
			}
			if string(reencoded) != string(data) {
				t.Fatal("accepted result was not canonical")
			}
		}
	})
}
