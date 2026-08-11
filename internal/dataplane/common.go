// Package dataplane implements the Genesis deterministic-CBOR profile for the
// TWIRX Semantic Data Plane. The protocol is defined by the repository's CDDL
// and prose specifications; this package is one non-normative implementation.
package dataplane

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/cborlite"
)

const (
	MaxDocumentBytes = 4 << 20
	MaxIdentifier    = 512
	MaxLocator       = 16 << 10
	MaxLexical       = 256 << 10
	MaxShortText     = 255
)

var (
	ErrInvalid       = errors.New("dataplane: invalid document")
	ErrTrailingBytes = errors.New("dataplane: trailing bytes")
	integerPattern   = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)
	decimalPattern   = regexp.MustCompile(`^-?(0|[1-9][0-9]*)\.[0-9]+$`)
)

type Digest [32]byte

type OptionalDigest struct {
	Present bool
	Value   Digest
}

type OptionalText struct {
	Present bool
	Value   string
}

type TypedValue struct {
	Type     string
	Lexical  string
	Unit     OptionalText
	Currency OptionalText
}

func DigestBytes(data []byte) Digest { return sha256.Sum256(data) }

func boundedEncoding(enc *cborlite.Encoder) ([]byte, error) {
	encoded := enc.Bytes()
	if len(encoded) == 0 || len(encoded) > MaxDocumentBytes {
		return nil, fmt.Errorf("%w: encoded document exceeds %d bytes", ErrInvalid, MaxDocumentBytes)
	}
	return encoded, nil
}

func validateText(name, value string, max int, empty bool) error {
	if (!empty && value == "") || len(value) > max || !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%w: %s is empty, invalid UTF-8, contains NUL, or exceeds %d bytes", ErrInvalid, name, max)
	}
	return nil
}

func validateIdentifier(name, value string) error {
	return validateText(name, value, MaxIdentifier, false)
}

func validateEnum(name, value string, allowed ...string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("%w: unsupported %s %q", ErrInvalid, name, value)
}

func validateUTCSecond(name, value string) error {
	if len(value) != 20 {
		return fmt.Errorf("%w: %s must be canonical UTC seconds", ErrInvalid, name)
	}
	parsed, err := time.Parse("2006-01-02T15:04:05Z", value)
	if err != nil || parsed.UTC().Format("2006-01-02T15:04:05Z") != value {
		return fmt.Errorf("%w: %s must be canonical UTC seconds", ErrInvalid, name)
	}
	return nil
}

func validateOptionalUTCSecond(name string, value OptionalText) error {
	if !value.Present {
		if value.Value != "" {
			return fmt.Errorf("%w: absent %s carries a value", ErrInvalid, name)
		}
		return nil
	}
	return validateUTCSecond(name, value.Value)
}

func validateOptionalIdentifier(name string, value OptionalText) error {
	if !value.Present {
		if value.Value != "" {
			return fmt.Errorf("%w: absent %s carries a value", ErrInvalid, name)
		}
		return nil
	}
	return validateIdentifier(name, value.Value)
}

func validateOptionalDigest(name string, value OptionalDigest) error {
	if !value.Present && value.Value != (Digest{}) {
		return fmt.Errorf("%w: absent %s carries a digest", ErrInvalid, name)
	}
	return nil
}

func validateSortedUniqueText(name string, values []string, maxCount int) error {
	if len(values) > maxCount {
		return fmt.Errorf("%w: %s count exceeds %d", ErrInvalid, name, maxCount)
	}
	for i, value := range values {
		if err := validateIdentifier(name, value); err != nil {
			return err
		}
		if i > 0 && strings.Compare(values[i-1], value) >= 0 {
			return fmt.Errorf("%w: %s must be strictly sorted and unique", ErrInvalid, name)
		}
	}
	return nil
}

func validateSortedUniqueDigests(name string, values []Digest, maxCount int, minimum int) error {
	if len(values) < minimum || len(values) > maxCount {
		return fmt.Errorf("%w: %s count outside %d..%d", ErrInvalid, name, minimum, maxCount)
	}
	for i := 1; i < len(values); i++ {
		if bytes.Compare(values[i-1][:], values[i][:]) >= 0 {
			return fmt.Errorf("%w: %s must be strictly sorted and unique", ErrInvalid, name)
		}
	}
	return nil
}

func validateCurrency(value OptionalText) error {
	if !value.Present {
		if value.Value != "" {
			return fmt.Errorf("%w: absent currency carries a value", ErrInvalid)
		}
		return nil
	}
	if len(value.Value) != 3 {
		return fmt.Errorf("%w: currency must be three uppercase ASCII letters", ErrInvalid)
	}
	for _, r := range value.Value {
		if r < 'A' || r > 'Z' {
			return fmt.Errorf("%w: currency must be three uppercase ASCII letters", ErrInvalid)
		}
	}
	return nil
}

func (v TypedValue) Validate() error {
	if err := validateEnum("typed value type", v.Type, "boolean", "integer", "decimal", "text", "date", "datetime", "duration", "uri", "identifier"); err != nil {
		return err
	}
	if err := validateText("typed lexical", v.Lexical, MaxLexical, false); err != nil {
		return err
	}
	if err := validateOptionalIdentifier("unit", v.Unit); err != nil {
		return err
	}
	if err := validateCurrency(v.Currency); err != nil {
		return err
	}
	switch v.Type {
	case "boolean":
		if v.Lexical != "true" && v.Lexical != "false" {
			return fmt.Errorf("%w: invalid boolean lexical form", ErrInvalid)
		}
	case "integer":
		if !integerPattern.MatchString(v.Lexical) {
			return fmt.Errorf("%w: invalid canonical integer lexical form", ErrInvalid)
		}
	case "decimal":
		if !decimalPattern.MatchString(v.Lexical) {
			return fmt.Errorf("%w: invalid canonical decimal lexical form", ErrInvalid)
		}
	case "date":
		parsed, err := time.Parse("2006-01-02", v.Lexical)
		if err != nil || parsed.Format("2006-01-02") != v.Lexical {
			return fmt.Errorf("%w: invalid canonical date lexical form", ErrInvalid)
		}
	case "datetime":
		if err := validateUTCSecond("datetime lexical", v.Lexical); err != nil {
			return err
		}
	case "duration":
		if !validDuration(v.Lexical) {
			return fmt.Errorf("%w: invalid bounded ISO-8601 duration lexical form", ErrInvalid)
		}
	case "uri":
		if !validAbsoluteURI(v.Lexical) {
			return fmt.Errorf("%w: invalid absolute URI lexical form", ErrInvalid)
		}
	case "identifier":
		if err := validateIdentifier("identifier lexical", v.Lexical); err != nil {
			return err
		}
	}
	return nil
}

func validAbsoluteURI(value string) bool {
	separator := strings.IndexByte(value, ':')
	if separator < 1 || value[0] < 'A' || (value[0] > 'Z' && value[0] < 'a') || value[0] > 'z' {
		return false
	}
	for i, r := range value {
		if r <= 0x20 || r == 0x7f {
			return false
		}
		if i > 0 && i < separator && !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '-' || r == '.') {
			return false
		}
	}
	return separator+1 < len(value)
}

func validDuration(value string) bool {
	if len(value) < 2 || value[0] != 'P' {
		return false
	}
	i := 1
	seen := false
	readDigits := func() bool {
		start := i
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		return i > start
	}
	dayStart := i
	if readDigits() {
		if i < len(value) && value[i] == 'D' {
			i++
			seen = true
		} else {
			i = dayStart
		}
	}
	if i < len(value) && value[i] == 'T' {
		i++
		timeSeen := false
		for _, unit := range []byte{'H', 'M', 'S'} {
			start := i
			if !readDigits() {
				i = start
				continue
			}
			if unit == 'S' && i < len(value) && value[i] == '.' {
				i++
				if !readDigits() {
					return false
				}
			}
			if i < len(value) && value[i] == unit {
				i++
				timeSeen = true
				seen = true
				continue
			}
			i = start
		}
		if !timeSeen {
			return false
		}
	}
	return seen && i == len(value)
}

func encodeDigest(enc *cborlite.Encoder, value Digest) { enc.Bytestring(value[:]) }

func decodeDigest(dec *cborlite.Decoder) (Digest, error) {
	var value Digest
	raw, err := dec.Bytestring(32)
	if err != nil || len(raw) != len(value) {
		return value, fmt.Errorf("%w: digest must be 32 bytes", ErrInvalid)
	}
	copy(value[:], raw)
	return value, nil
}

func encodeOptionalDigest(enc *cborlite.Encoder, value OptionalDigest) {
	if !value.Present {
		enc.Nil()
		return
	}
	encodeDigest(enc, value.Value)
}

func decodeOptionalDigest(dec *cborlite.Decoder) (OptionalDigest, error) {
	nilValue, err := dec.TryNil()
	if err != nil {
		return OptionalDigest{}, err
	}
	if nilValue {
		return OptionalDigest{}, nil
	}
	value, err := decodeDigest(dec)
	return OptionalDigest{Present: err == nil, Value: value}, err
}

func encodeOptionalText(enc *cborlite.Encoder, value OptionalText) {
	if !value.Present {
		enc.Nil()
		return
	}
	enc.Text(value.Value)
}

func decodeOptionalText(dec *cborlite.Decoder, max uint64) (OptionalText, error) {
	nilValue, err := dec.TryNil()
	if err != nil {
		return OptionalText{}, err
	}
	if nilValue {
		return OptionalText{}, nil
	}
	value, err := dec.Text(max)
	return OptionalText{Present: err == nil, Value: value}, err
}

func encodeTypedValue(enc *cborlite.Encoder, value TypedValue) {
	enc.Array(4)
	enc.Text(value.Type)
	enc.Text(value.Lexical)
	encodeOptionalText(enc, value.Unit)
	encodeOptionalText(enc, value.Currency)
}

func decodeTypedValue(dec *cborlite.Decoder) (TypedValue, error) {
	var value TypedValue
	n, err := dec.Array()
	if err != nil || n != 4 {
		return value, fmt.Errorf("%w: typed value array", ErrInvalid)
	}
	if value.Type, err = dec.Text(MaxIdentifier); err != nil {
		return value, err
	}
	if value.Lexical, err = dec.Text(MaxLexical); err != nil {
		return value, err
	}
	if value.Unit, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return value, err
	}
	if value.Currency, err = decodeOptionalText(dec, 3); err != nil {
		return value, err
	}
	return value, value.Validate()
}

func encodeTextSet(enc *cborlite.Encoder, values []string) {
	enc.Array(uint64(len(values)))
	for _, value := range values {
		enc.Text(value)
	}
}

func decodeTextSet(dec *cborlite.Decoder, maxCount int, minCount int) ([]string, error) {
	n, err := dec.Array()
	if err != nil || n < uint64(minCount) || n > uint64(maxCount) {
		return nil, fmt.Errorf("%w: text-set count outside bounds", ErrInvalid)
	}
	values := make([]string, 0, n)
	for i := uint64(0); i < n; i++ {
		value, textErr := dec.Text(MaxIdentifier)
		if textErr != nil {
			return nil, textErr
		}
		values = append(values, value)
	}
	if err := validateSortedUniqueText("text set", values, maxCount); err != nil {
		return nil, err
	}
	return values, nil
}

func encodeDigestSet(enc *cborlite.Encoder, values []Digest) {
	enc.Array(uint64(len(values)))
	for _, value := range values {
		encodeDigest(enc, value)
	}
}

func decodeDigestSet(dec *cborlite.Decoder, maxCount int, minCount int) ([]Digest, error) {
	n, err := dec.Array()
	if err != nil || n < uint64(minCount) || n > uint64(maxCount) {
		return nil, fmt.Errorf("%w: digest-set count outside bounds", ErrInvalid)
	}
	values := make([]Digest, 0, n)
	for i := uint64(0); i < n; i++ {
		value, digestErr := decodeDigest(dec)
		if digestErr != nil {
			return nil, digestErr
		}
		values = append(values, value)
	}
	if err := validateSortedUniqueDigests("digest set", values, maxCount, minCount); err != nil {
		return nil, err
	}
	return values, nil
}

func expectEmptyExtensions(dec *cborlite.Decoder) error {
	n, err := dec.Array()
	if err != nil || n != 0 {
		return fmt.Errorf("%w: extensions must be empty", ErrInvalid)
	}
	return nil
}

func finish(dec *cborlite.Decoder) error {
	if dec.Remaining() != 0 {
		return ErrTrailingBytes
	}
	return nil
}

func checkedDocument(data []byte, maximum int) (*cborlite.Decoder, error) {
	if len(data) == 0 || len(data) > maximum {
		return nil, fmt.Errorf("%w: byte length outside 1..%d", ErrInvalid, maximum)
	}
	return cborlite.NewDecoder(data), nil
}

func sortedStrings(values []string) bool {
	return sort.SliceIsSorted(values, func(i, j int) bool { return values[i] < values[j] })
}

func parseCanonicalUintText(value string) (uint64, error) {
	if value == "" || (len(value) > 1 && value[0] == '0') {
		return 0, ErrInvalid
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, ErrInvalid
	}
	return parsed, nil
}
