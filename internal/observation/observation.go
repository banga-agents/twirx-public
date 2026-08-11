// Package observation defines the Genesis observation envelope: a compact,
// deterministic record of what bytes an origin returned and when.
package observation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/cas"
	"github.com/typed-web-commons/typed-web/internal/cborlite"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

const (
	FormatVersion         uint64 = 1
	ObserverID                   = "typed-web-go/0.1"
	EnvelopeFields               = 11
	MaxEnvelopeBytes             = 64 << 10
	MaxURLBytes                  = 8192
	MaxMediaTypeBytes            = 256
	MaxRetrievedAtBytes          = 30
	MaxIDBytes                   = 256
	MaxBodyBytes                 = 2 << 20
	MaxJSONViewBytes             = 128 << 10
	MaxBodyReferenceBytes        = 128
)

// Envelope is serialized as a fixed CBOR array, not a map. The array layout is
// specified in schemas/cddl/observation.cddl and is normative for Genesis.
type Envelope struct {
	Version     uint64
	RequestURL  string
	FinalURL    string
	Method      string
	Status      uint64
	MediaType   string
	RetrievedAt string
	BodySHA256  [32]byte
	BodySize    uint64
	PolicyID    string
	ObserverID  string
}

type JSONEnvelope struct {
	Format       string `json:"format"`
	Version      uint64 `json:"version"`
	RequestURL   string `json:"request_url"`
	FinalURL     string `json:"final_url"`
	Method       string `json:"method"`
	Status       uint64 `json:"status"`
	MediaType    string `json:"media_type"`
	RetrievedAt  string `json:"retrieved_at"`
	BodyDigest   string `json:"body_digest"`
	BodySize     uint64 `json:"body_size"`
	PolicyID     string `json:"policy_id"`
	ObserverID   string `json:"observer_id"`
	EnvelopeHash string `json:"envelope_hash"`
}

func FromFetch(result *safefetch.Result, policyID string) (*Envelope, error) {
	if result == nil {
		return nil, errors.New("observation: fetch result is nil")
	}
	sum := sha256.Sum256(result.Body)
	env := &Envelope{
		Version:     FormatVersion,
		RequestURL:  result.RequestURL,
		FinalURL:    result.FinalURL,
		Method:      result.Method,
		Status:      uint64(result.Status),
		MediaType:   result.MediaType,
		RetrievedAt: result.RetrievedAt.UTC().Format(time.RFC3339Nano),
		BodySHA256:  sum,
		BodySize:    uint64(len(result.Body)),
		PolicyID:    policyID,
		ObserverID:  ObserverID,
	}
	if err := env.Validate(); err != nil {
		return nil, err
	}
	return env, nil
}

func (e *Envelope) Validate() error {
	if e.Version != FormatVersion {
		return fmt.Errorf("observation: unsupported version %d", e.Version)
	}
	if e.Method != "GET" {
		return fmt.Errorf("observation: method %q is not allowed in Genesis", e.Method)
	}
	if e.Status < 100 || e.Status > 599 {
		return fmt.Errorf("observation: invalid HTTP status %d", e.Status)
	}
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{name: "request_url", value: e.RequestURL, max: MaxURLBytes},
		{name: "final_url", value: e.FinalURL, max: MaxURLBytes},
		{name: "method", value: e.Method, max: len("GET")},
		{name: "media_type", value: e.MediaType, max: MaxMediaTypeBytes},
		{name: "retrieved_at", value: e.RetrievedAt, max: MaxRetrievedAtBytes},
		{name: "policy_id", value: e.PolicyID, max: MaxIDBytes},
		{name: "observer_id", value: e.ObserverID, max: MaxIDBytes},
	} {
		if err := validateText(field.name, field.value, field.max); err != nil {
			return err
		}
	}
	parsed, err := time.Parse(time.RFC3339Nano, e.RetrievedAt)
	if err != nil || parsed.Location() != time.UTC || parsed.Format(time.RFC3339Nano) != e.RetrievedAt {
		return fmt.Errorf("observation: retrieved_at is not canonical RFC3339Nano UTC")
	}
	if e.BodySize > MaxBodyBytes {
		return fmt.Errorf("observation: body size %d exceeds %d-byte Genesis limit", e.BodySize, MaxBodyBytes)
	}
	return nil
}

func validateText(name, value string, max int) error {
	if value == "" {
		return fmt.Errorf("observation: %s is required", name)
	}
	if len(value) > max {
		return fmt.Errorf("observation: %s exceeds %d bytes", name, max)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("observation: %s is not valid UTF-8", name)
	}
	if strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("observation: %s contains U+0000", name)
	}
	return nil
}

func (e *Envelope) MarshalCBOR() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(EnvelopeFields)
	enc.Uint(e.Version)
	enc.Text(e.RequestURL)
	enc.Text(e.FinalURL)
	enc.Text(e.Method)
	enc.Uint(e.Status)
	enc.Text(e.MediaType)
	enc.Text(e.RetrievedAt)
	enc.Bytestring(e.BodySHA256[:])
	enc.Uint(e.BodySize)
	enc.Text(e.PolicyID)
	enc.Text(e.ObserverID)
	return enc.Bytes(), nil
}

func UnmarshalCBOR(data []byte) (*Envelope, error) {
	if len(data) == 0 || len(data) > MaxEnvelopeBytes {
		return nil, fmt.Errorf("observation: envelope size %d is outside allowed range", len(data))
	}
	dec := cborlite.NewDecoder(data)
	n, err := dec.Array()
	if err != nil {
		return nil, fmt.Errorf("observation: decode array: %w", err)
	}
	if n != EnvelopeFields {
		return nil, fmt.Errorf("observation: expected %d fields, got %d", EnvelopeFields, n)
	}

	e := &Envelope{}
	if e.Version, err = dec.Uint(); err != nil {
		return nil, fmt.Errorf("observation: version: %w", err)
	}
	if e.RequestURL, err = dec.Text(MaxURLBytes); err != nil {
		return nil, fmt.Errorf("observation: request URL: %w", err)
	}
	if e.FinalURL, err = dec.Text(MaxURLBytes); err != nil {
		return nil, fmt.Errorf("observation: final URL: %w", err)
	}
	if e.Method, err = dec.Text(uint64(len("GET"))); err != nil {
		return nil, fmt.Errorf("observation: method: %w", err)
	}
	if e.Status, err = dec.Uint(); err != nil {
		return nil, fmt.Errorf("observation: status: %w", err)
	}
	if e.MediaType, err = dec.Text(MaxMediaTypeBytes); err != nil {
		return nil, fmt.Errorf("observation: media type: %w", err)
	}
	if e.RetrievedAt, err = dec.Text(MaxRetrievedAtBytes); err != nil {
		return nil, fmt.Errorf("observation: retrieval time: %w", err)
	}
	hashBytes, err := dec.Bytestring(32)
	if err != nil {
		return nil, fmt.Errorf("observation: body hash: %w", err)
	}
	if len(hashBytes) != 32 {
		return nil, fmt.Errorf("observation: body hash length %d, want 32", len(hashBytes))
	}
	copy(e.BodySHA256[:], hashBytes)
	if e.BodySize, err = dec.Uint(); err != nil {
		return nil, fmt.Errorf("observation: body size: %w", err)
	}
	if e.PolicyID, err = dec.Text(MaxIDBytes); err != nil {
		return nil, fmt.Errorf("observation: policy ID: %w", err)
	}
	if e.ObserverID, err = dec.Text(MaxIDBytes); err != nil {
		return nil, fmt.Errorf("observation: observer ID: %w", err)
	}
	if dec.Remaining() != 0 {
		return nil, fmt.Errorf("observation: %d trailing bytes", dec.Remaining())
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Envelope) BodyDigest() string {
	return cas.Algorithm + ":" + hex.EncodeToString(e.BodySHA256[:])
}

func EnvelopeDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return cas.Algorithm + ":" + hex.EncodeToString(sum[:])
}

func (e *Envelope) JSON(cborBytes []byte) JSONEnvelope {
	return JSONEnvelope{
		Format:       "tw.observation/0.1",
		Version:      e.Version,
		RequestURL:   e.RequestURL,
		FinalURL:     e.FinalURL,
		Method:       e.Method,
		Status:       e.Status,
		MediaType:    e.MediaType,
		RetrievedAt:  e.RetrievedAt,
		BodyDigest:   e.BodyDigest(),
		BodySize:     e.BodySize,
		PolicyID:     e.PolicyID,
		ObserverID:   e.ObserverID,
		EnvelopeHash: EnvelopeDigest(cborBytes),
	}
}

type BundlePaths struct {
	Directory       string
	CBORPath        string
	JSONPath        string
	BodyReference   string
	BodyStoragePath string
}

func WriteBundle(outDir string, store *cas.Store, result *safefetch.Result, policyID string) (*BundlePaths, error) {
	if store == nil {
		return nil, errors.New("observation: CAS store is nil")
	}
	if result == nil {
		return nil, errors.New("observation: fetch result is nil")
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return nil, fmt.Errorf("observation: create output directory: %w", err)
	}
	env, err := FromFetch(result, policyID)
	if err != nil {
		return nil, err
	}
	bodyDigest, bodyPath, err := store.Put(result.Body)
	if err != nil {
		return nil, err
	}
	if env.BodyDigest() != bodyDigest {
		return nil, errors.New("observation: CAS digest does not match envelope body digest")
	}
	cborBytes, err := env.MarshalCBOR()
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.MarshalIndent(env.JSON(cborBytes), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("observation: marshal JSON view: %w", err)
	}
	jsonBytes = append(jsonBytes, '\n')

	cborPath := filepath.Join(outDir, "observation.cbor")
	jsonPath := filepath.Join(outDir, "observation.json")
	refPath := filepath.Join(outDir, "body.ref")
	if err := atomicfile.Write(cborPath, cborBytes, MaxEnvelopeBytes, 0o640); err != nil {
		return nil, err
	}
	if err := atomicfile.Write(jsonPath, jsonBytes, MaxJSONViewBytes, 0o640); err != nil {
		return nil, err
	}
	if err := atomicfile.Write(refPath, []byte(bodyDigest+"\n"), MaxBodyReferenceBytes, 0o640); err != nil {
		return nil, err
	}
	return &BundlePaths{
		Directory:       outDir,
		CBORPath:        cborPath,
		JSONPath:        jsonPath,
		BodyReference:   bodyDigest,
		BodyStoragePath: bodyPath,
	}, nil
}

func Load(path string) (*Envelope, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("observation: read envelope: %w", err)
	}
	env, err := UnmarshalCBOR(data)
	if err != nil {
		return nil, nil, err
	}
	return env, data, nil
}

func VerifyBody(env *Envelope, store *cas.Store, maxBytes int64) error {
	if env == nil || store == nil {
		return errors.New("observation: envelope and store are required")
	}
	if maxBytes <= 0 || maxBytes > MaxBodyBytes {
		return fmt.Errorf("observation: body limit must be between 1 and %d bytes", MaxBodyBytes)
	}
	body, err := store.Read(env.BodyDigest(), maxBytes)
	if err != nil {
		return err
	}
	if uint64(len(body)) != env.BodySize {
		return fmt.Errorf("observation: body size %d, envelope declares %d", len(body), env.BodySize)
	}
	return nil
}
