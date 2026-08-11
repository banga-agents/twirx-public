package cborlite

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	var enc Encoder
	enc.Array(4)
	enc.Uint(1)
	enc.Text("hello")
	enc.Bytestring([]byte{1, 2, 3})
	enc.Uint(1000)

	dec := NewDecoder(enc.Bytes())
	if n, err := dec.Array(); err != nil || n != 4 {
		t.Fatalf("array: n=%d err=%v", n, err)
	}
	if v, err := dec.Uint(); err != nil || v != 1 {
		t.Fatalf("uint: v=%d err=%v", v, err)
	}
	if v, err := dec.Text(32); err != nil || v != "hello" {
		t.Fatalf("text: v=%q err=%v", v, err)
	}
	if v, err := dec.Bytestring(32); err != nil || !bytes.Equal(v, []byte{1, 2, 3}) {
		t.Fatalf("bytes: v=%v err=%v", v, err)
	}
	if v, err := dec.Uint(); err != nil || v != 1000 {
		t.Fatalf("uint 1000: v=%d err=%v", v, err)
	}
	if dec.Remaining() != 0 {
		t.Fatalf("remaining=%d", dec.Remaining())
	}
}

func TestRejectsNonCanonicalUint(t *testing.T) {
	dec := NewDecoder([]byte{0x18, 0x01})
	if _, err := dec.Uint(); err == nil {
		t.Fatal("expected non-canonical encoding error")
	}
}

func TestRejectsInvalidUTF8Text(t *testing.T) {
	dec := NewDecoder([]byte{0x61, 0xff})
	if _, err := dec.Text(8); err == nil {
		t.Fatal("expected invalid UTF-8 rejection")
	}
}

func TestNilRoundTripAndNonConsumption(t *testing.T) {
	var enc Encoder
	enc.Array(2)
	enc.Nil()
	enc.Text("value")

	dec := NewDecoder(enc.Bytes())
	if n, err := dec.Array(); err != nil || n != 2 {
		t.Fatalf("array: n=%d err=%v", n, err)
	}
	if ok, err := dec.TryNil(); err != nil || !ok {
		t.Fatalf("nil: ok=%v err=%v", ok, err)
	}
	if ok, err := dec.TryNil(); err != nil || ok {
		t.Fatalf("non-null lookahead: ok=%v err=%v", ok, err)
	}
	if got, err := dec.Text(16); err != nil || got != "value" {
		t.Fatalf("text: got=%q err=%v", got, err)
	}
	if dec.Remaining() != 0 {
		t.Fatalf("remaining = %d", dec.Remaining())
	}
}
