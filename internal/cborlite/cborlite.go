// Package cborlite implements the deliberately tiny deterministic-CBOR subset
// used by the Genesis observation envelope. It is not a general CBOR library.
package cborlite

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf8"
)

var (
	ErrUnexpectedEOF   = errors.New("cborlite: unexpected end of input")
	ErrUnsupportedType = errors.New("cborlite: unsupported CBOR type")
	ErrNonCanonical    = errors.New("cborlite: non-canonical integer encoding")
	ErrInvalidUTF8     = errors.New("cborlite: invalid UTF-8 text string")
)

// Encoder appends deterministic CBOR values to an internal buffer.
type Encoder struct {
	buf []byte
}

func (e *Encoder) Bytes() []byte {
	out := make([]byte, len(e.buf))
	copy(out, e.buf)
	return out
}

func (e *Encoder) Array(n uint64) { e.appendHead(4, n) }
func (e *Encoder) Uint(v uint64)  { e.appendHead(0, v) }
func (e *Encoder) Nil()           { e.buf = append(e.buf, 0xf6) }

func (e *Encoder) Text(v string) {
	e.appendHead(3, uint64(len(v)))
	e.buf = append(e.buf, v...)
}

func (e *Encoder) Bytestring(v []byte) {
	e.appendHead(2, uint64(len(v)))
	e.buf = append(e.buf, v...)
}

func (e *Encoder) appendHead(major byte, value uint64) {
	switch {
	case value < 24:
		e.buf = append(e.buf, major<<5|byte(value))
	case value <= 0xff:
		e.buf = append(e.buf, major<<5|24, byte(value))
	case value <= 0xffff:
		e.buf = append(e.buf, major<<5|25)
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(value))
		e.buf = append(e.buf, b[:]...)
	case value <= 0xffffffff:
		e.buf = append(e.buf, major<<5|26)
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(value))
		e.buf = append(e.buf, b[:]...)
	default:
		e.buf = append(e.buf, major<<5|27)
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], value)
		e.buf = append(e.buf, b[:]...)
	}
}

// Decoder parses the same constrained deterministic-CBOR subset.
type Decoder struct {
	data []byte
	off  int
}

func NewDecoder(data []byte) *Decoder { return &Decoder{data: data} }
func (d *Decoder) Remaining() int     { return len(d.data) - d.off }

func (d *Decoder) Array() (uint64, error) {
	major, value, err := d.head()
	if err != nil {
		return 0, err
	}
	if major != 4 {
		return 0, fmt.Errorf("%w: expected array, got major type %d", ErrUnsupportedType, major)
	}
	return value, nil
}

func (d *Decoder) Uint() (uint64, error) {
	major, value, err := d.head()
	if err != nil {
		return 0, err
	}
	if major != 0 {
		return 0, fmt.Errorf("%w: expected unsigned integer, got major type %d", ErrUnsupportedType, major)
	}
	return value, nil
}

// TryNil consumes a canonical CBOR null and reports true. It leaves the
// decoder unchanged for every other value so a caller can decode an explicitly
// optional field without accepting an unrelated simple value.
func (d *Decoder) TryNil() (bool, error) {
	if d.off >= len(d.data) {
		return false, ErrUnexpectedEOF
	}
	if d.data[d.off] == 0xf6 {
		d.off++
		return true, nil
	}
	return false, nil
}

func (d *Decoder) Text(max uint64) (string, error) {
	major, n, err := d.head()
	if err != nil {
		return "", err
	}
	if major != 3 {
		return "", fmt.Errorf("%w: expected text string, got major type %d", ErrUnsupportedType, major)
	}
	if n > max {
		return "", fmt.Errorf("cborlite: text length %d exceeds limit %d", n, max)
	}
	b, err := d.take(n)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", ErrInvalidUTF8
	}
	return string(b), nil
}

func (d *Decoder) Bytestring(max uint64) ([]byte, error) {
	major, n, err := d.head()
	if err != nil {
		return nil, err
	}
	if major != 2 {
		return nil, fmt.Errorf("%w: expected byte string, got major type %d", ErrUnsupportedType, major)
	}
	if n > max {
		return nil, fmt.Errorf("cborlite: byte-string length %d exceeds limit %d", n, max)
	}
	b, err := d.take(n)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

func (d *Decoder) head() (major byte, value uint64, err error) {
	if d.off >= len(d.data) {
		return 0, 0, ErrUnexpectedEOF
	}
	initial := d.data[d.off]
	d.off++
	major = initial >> 5
	ai := initial & 0x1f
	switch {
	case ai < 24:
		return major, uint64(ai), nil
	case ai == 24:
		b, err := d.take(1)
		if err != nil {
			return 0, 0, err
		}
		value = uint64(b[0])
		if value < 24 {
			return 0, 0, ErrNonCanonical
		}
		return major, value, nil
	case ai == 25:
		b, err := d.take(2)
		if err != nil {
			return 0, 0, err
		}
		value = uint64(binary.BigEndian.Uint16(b))
		if value <= 0xff {
			return 0, 0, ErrNonCanonical
		}
		return major, value, nil
	case ai == 26:
		b, err := d.take(4)
		if err != nil {
			return 0, 0, err
		}
		value = uint64(binary.BigEndian.Uint32(b))
		if value <= 0xffff {
			return 0, 0, ErrNonCanonical
		}
		return major, value, nil
	case ai == 27:
		b, err := d.take(8)
		if err != nil {
			return 0, 0, err
		}
		value = binary.BigEndian.Uint64(b)
		if value <= 0xffffffff {
			return 0, 0, ErrNonCanonical
		}
		return major, value, nil
	default:
		return 0, 0, fmt.Errorf("%w: indefinite or reserved additional information %d", ErrUnsupportedType, ai)
	}
}

func (d *Decoder) take(n uint64) ([]byte, error) {
	if n > uint64(len(d.data)-d.off) {
		return nil, ErrUnexpectedEOF
	}
	start := d.off
	d.off += int(n)
	return d.data[start:d.off], nil
}
