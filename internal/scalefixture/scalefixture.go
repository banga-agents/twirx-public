// Package scalefixture defines a deterministic controlled representation used
// to exercise Semantic Snapshot scale without inflating public-origin claims.
// It is never a source of publisher or archive evidence.
package scalefixture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	Format          = "tw.controlled-scale-fixture/0.1"
	OriginID        = "controlled-scale-corpus-fixture"
	OperationID     = "fixture.scaleCorpus"
	ObserverID      = "twirx-controlled-scale-fixture/0.1"
	PolicyID        = "tw.fixture.scale-v0"
	RequestURL      = "http://127.0.0.1:18099/scale.json"
	SubjectID       = "fixture:scale-corpus/document/1"
	MaxFields       = 100000
	MaxBodyBytes    = 2 << 20
	MaxProfileBytes = 64 << 10
)

type Profile struct {
	Format     string `json:"format"`
	OriginID   string `json:"origin_id"`
	Operation  string `json:"operation_id"`
	SubjectID  string `json:"subject_id"`
	FieldCount uint64 `json:"field_count"`
	ObservedAt string `json:"observed_at"`
	Generator  string `json:"generator"`
}

type Plan struct {
	Format     string `json:"format"`
	FieldID    string `json:"field_id"`
	Locator    string `json:"locator"`
	Extraction string `json:"extraction"`
}

func NewProfile(fieldCount uint64, observedAt string) (Profile, error) {
	profile := Profile{Format: Format, OriginID: OriginID, Operation: OperationID, SubjectID: SubjectID, FieldCount: fieldCount, ObservedAt: observedAt, Generator: "sorted-json-string-fields-v1"}
	return profile, profile.Validate()
}

func (p Profile) Validate() error {
	if p.Format != Format || p.OriginID != OriginID || p.Operation != OperationID || p.SubjectID != SubjectID || p.Generator != "sorted-json-string-fields-v1" || p.FieldCount == 0 || p.FieldCount > MaxFields {
		return errors.New("scalefixture: invalid profile")
	}
	observedAt, err := time.Parse("2006-01-02T15:04:05Z", p.ObservedAt)
	if err != nil || observedAt.UTC().Format("2006-01-02T15:04:05Z") != p.ObservedAt {
		return errors.New("scalefixture: observed_at must be canonical UTC seconds")
	}
	return nil
}

func MarshalProfile(profile Profile) ([]byte, error) {
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) > MaxProfileBytes {
		return nil, errors.New("scalefixture: profile exceeds bound")
	}
	return data, nil
}

func UnmarshalProfile(data []byte) (Profile, error) {
	var profile Profile
	policy := jsonbounded.Policy{MaxBytes: MaxProfileBytes, MaxDepth: 4, MaxScalarBytes: 1024, MaxContainerEntries: 16, MaxTokens: 64}
	if err := jsonbounded.Decode(data, &profile, policy, true); err != nil {
		return profile, err
	}
	return profile, profile.Validate()
}

func FieldID(index uint64) string { return fmt.Sprintf("field_%05d", index) }

func FieldValue(index uint64) string { return fmt.Sprintf("value_%05d", index) }

func Locator(fieldID string) string { return "/" + fieldID }

func PlanBytes(fieldID string) ([]byte, error) {
	if !validFieldID(fieldID) {
		return nil, errors.New("scalefixture: invalid field ID")
	}
	return json.Marshal(Plan{Format: "tw.controlled-scale-extraction-plan/0.1", FieldID: fieldID, Locator: Locator(fieldID), Extraction: "exact-json-object-string-field"})
}

func GenerateBody(profile Profile) ([]byte, error) {
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	var body bytes.Buffer
	body.Grow(int(profile.FieldCount) * 32)
	body.WriteByte('{')
	for index := uint64(0); index < profile.FieldCount; index++ {
		if index != 0 {
			body.WriteByte(',')
		}
		key, _ := json.Marshal(FieldID(index))
		value, _ := json.Marshal(FieldValue(index))
		body.Write(key)
		body.WriteByte(':')
		body.Write(value)
	}
	body.WriteByte('}')
	if body.Len() > MaxBodyBytes {
		return nil, fmt.Errorf("scalefixture: generated body exceeds %d bytes", MaxBodyBytes)
	}
	return body.Bytes(), nil
}

// ParseBody validates the complete controlled representation and returns
// deterministic field/value pairs. The caller must still bind the body digest
// through an observation envelope.
func ParseBody(data []byte, profile Profile) ([][2]string, error) {
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	var values map[string]string
	policy := jsonbounded.Policy{MaxBytes: MaxBodyBytes, MaxDepth: 2, MaxScalarBytes: 128, MaxContainerEntries: int(profile.FieldCount), MaxTokens: int(profile.FieldCount)*2 + 2}
	if err := jsonbounded.Decode(data, &values, policy, false); err != nil {
		return nil, err
	}
	if uint64(len(values)) != profile.FieldCount {
		return nil, errors.New("scalefixture: field count mismatch")
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([][2]string, 0, len(keys))
	for index, key := range keys {
		expectedKey := FieldID(uint64(index))
		if key != expectedKey || values[key] != FieldValue(uint64(index)) {
			return nil, errors.New("scalefixture: generated field sequence mismatch")
		}
		fields = append(fields, [2]string{key, values[key]})
	}
	return fields, nil
}

func validFieldID(value string) bool {
	if !strings.HasPrefix(value, "field_") || len(value) < len("field_0") {
		return false
	}
	number := strings.TrimPrefix(value, "field_")
	if len(number) < 5 {
		return false
	}
	parsed, err := strconv.ParseUint(number, 10, 64)
	return err == nil && parsed < MaxFields && FieldID(parsed) == value
}
