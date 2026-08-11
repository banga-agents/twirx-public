// Package labengine executes catalog-only, read-only E2 operations and emits
// complete manifest-last proof bundles. No caller-supplied destination reaches
// the network path.
package labengine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/e2format"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/observation"
	"github.com/typed-web-commons/typed-web/internal/origincatalog"
	"github.com/typed-web-commons/typed-web/internal/proofbundle"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
	"github.com/typed-web-commons/typed-web/internal/transportevidence"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

const (
	ModeFresh          = "fresh"
	ModeReplay         = "replay"
	MaxTranscriptBytes = 256 << 10
)

var decimalPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type Engine struct {
	Root       string
	ResultsDir string
	Contracts  *twircontract.Set
	Catalog    *origincatalog.Catalog
	publishMu  sync.Mutex
}

type Request struct {
	OriginID    string            `json:"origin_id"`
	OperationID string            `json:"operation_id"`
	Input       map[string]string `json:"input"`
	Mode        string            `json:"mode"`
}

type Invocation struct {
	Publication proofbundle.Publication
	Result      e2format.Result
	Mode        string
	Attribution string
}

func New(root, resultsDir string) (*Engine, error) {
	contracts, err := twircontract.Load(filepath.Join(root, "contracts", "e2", "contracts.json"))
	if err != nil {
		return nil, err
	}
	catalog, err := origincatalog.Load(filepath.Join(root, "origins", "catalog.json"))
	if err != nil {
		return nil, err
	}
	selection, err := atlas.LoadSelection(filepath.Join(root, "atlas", "genesis-500", "selection.json"))
	if err != nil {
		return nil, fmt.Errorf("labengine: load canonical selection: %w", err)
	}
	policies, err := atlas.LoadPolicySet(filepath.Join(root, "atlas", "policies.json"), selection)
	if err != nil {
		return nil, fmt.Errorf("labengine: load canonical policy set: %w", err)
	}
	registry, err := atlas.LoadRegistry(filepath.Join(root, "atlas", "registry.json"), selection, policies)
	if err != nil {
		return nil, fmt.Errorf("labengine: load canonical origin registry: %w", err)
	}
	if err := catalog.ValidateRegistry(registry); err != nil {
		return nil, err
	}
	if resultsDir == "" {
		return nil, errors.New("labengine: results directory is required")
	}
	return &Engine{Root: root, ResultsDir: resultsDir, Contracts: contracts, Catalog: catalog}, nil
}

func (e *Engine) Invoke(ctx context.Context, request Request) (*Invocation, error) {
	if request.Mode == "" {
		request.Mode = ModeFresh
	}
	if request.Mode != ModeFresh && request.Mode != ModeReplay {
		return nil, errors.New("labengine: mode must be fresh or replay")
	}
	op, err := e.Contracts.Find(request.OperationID)
	if err != nil {
		return nil, err
	}
	origin, err := e.Catalog.Find(request.OriginID)
	if err != nil {
		return nil, err
	}
	owner, err := e.Catalog.ForOperation(request.OperationID)
	if err != nil {
		return nil, err
	}
	if owner.ID != origin.ID || op.OriginID != origin.ID {
		return nil, errors.New("labengine: operation does not belong to requested origin")
	}
	inputBytes, err := twircontract.CanonicalInput(op, request.Input)
	if err != nil {
		return nil, err
	}
	fetched, policyID, err := e.acquire(ctx, origin, op, request.Input, request.Mode)
	if err != nil {
		return nil, err
	}
	envelope, err := observation.FromFetch(fetched, policyID)
	if err != nil {
		return nil, err
	}
	observationBytes, err := envelope.MarshalCBOR()
	if err != nil {
		return nil, err
	}
	transport, err := transportevidence.FromFetch(fetched, policyID)
	if err != nil {
		return nil, err
	}
	transportBytes, err := transport.MarshalCBOR()
	if err != nil {
		return nil, err
	}
	contractBytes, err := e.Contracts.MarshalOperation(op)
	if err != nil {
		return nil, err
	}
	adapterBytes, err := twircontract.MarshalAdapterDescriptor(op)
	if err != nil {
		return nil, err
	}
	closureBytes, err := twircontract.MarshalSemanticClosure(op.SemanticClosure)
	if err != nil {
		return nil, err
	}
	fields, status, err := extractFields(fetched.Body, op)
	if err != nil {
		return nil, err
	}
	invocationID := deriveInvocationID(op.ID, inputBytes, observationBytes)
	result := e2format.Result{
		Version: e2format.ResultVersion, InvocationID: invocationID, OriginID: origin.ID, OriginVersion: origin.Version,
		OperationID: op.ID, OperationVersion: op.Version, Effect: "read", Status: status, ObservedAt: envelope.RetrievedAt,
		InputDigest: sha256.Sum256(inputBytes), ObservationDigest: sha256.Sum256(observationBytes), TransportDigest: sha256.Sum256(transportBytes),
		AdapterDigest: sha256.Sum256(adapterBytes), ContractDigest: sha256.Sum256(contractBytes), SemanticClosureDigest: sha256.Sum256(closureBytes), Fields: fields,
	}
	resultBytes, err := e2format.MarshalResult(result)
	if err != nil {
		return nil, err
	}
	resultDigest := sha256.Sum256(resultBytes)
	transcriptBytes, err := marshalTranscript(request, origin, op, result, envelope, e2format.DigestReference(resultDigest))
	if err != nil {
		return nil, err
	}
	artifacts := map[string][]byte{
		"adapter.cbor": adapterBytes, "contract.cbor": contractBytes, "input.cbor": inputBytes, "observation.cbor": observationBytes,
		"representation.body": append([]byte(nil), fetched.Body...), "result.cbor": resultBytes, "semantic-closure.cbor": closureBytes,
		"transcript.json": transcriptBytes, "transport.cbor": transportBytes,
	}
	digestHex := hex.EncodeToString(resultDigest[:])
	if err := os.MkdirAll(e.ResultsDir, 0o750); err != nil {
		return nil, fmt.Errorf("labengine: create results root: %w", err)
	}
	dir := filepath.Join(e.ResultsDir, digestHex)
	e.publishMu.Lock()
	defer e.publishMu.Unlock()
	publication, err := proofbundle.Write(dir, artifacts)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		publication, err = proofbundle.Verify(dir)
		if err != nil {
			return nil, fmt.Errorf("labengine: existing result is invalid: %w", err)
		}
	}
	if _, _, err := e.Verify(dir); err != nil {
		return nil, fmt.Errorf("labengine: post-publication verification: %w", err)
	}
	return &Invocation{Publication: publication, Result: result, Mode: request.Mode, Attribution: origin.Attribution}, nil
}

func (e *Engine) acquire(ctx context.Context, origin *origincatalog.Origin, op *twircontract.Operation, input map[string]string, mode string) (*safefetch.Result, string, error) {
	requestURL, err := origin.RequestURL(op, input)
	if err != nil {
		return nil, "", err
	}
	if mode == ModeReplay || origin.AccessMode == "controlled_fixture" {
		if mode == ModeReplay && !sameInput(input, origin.ReplayInput) {
			return nil, "", errors.New("labengine: no admitted replay fixture for this input")
		}
		path := filepath.Join(e.Root, origin.ReplayFixture)
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, "", fmt.Errorf("labengine: read fixture: %w", readErr)
		}
		if int64(len(body)) > origin.MaxResponseBytes {
			return nil, "", errors.New("labengine: fixture exceeds origin response limit")
		}
		retrievedAt := time.Now().UTC()
		policyID := "tw.fetch.controlled-fixture-v0"
		if mode == ModeReplay {
			retrievedAt, err = time.Parse(time.RFC3339Nano, origin.ReplayObservedAt)
			if err != nil {
				return nil, "", err
			}
			policyID = "tw.fetch.offline-replay-v0"
		}
		return &safefetch.Result{RequestURL: requestURL, FinalURL: requestURL, Method: http.MethodGet, Status: http.StatusOK, MediaType: "application/json", RetrievedAt: retrievedAt, Body: body, Headers: []safefetch.Header{{Name: "content-type", Value: "application/json"}}}, policyID, nil
	}
	policy := safefetch.DefaultPolicy()
	policy.ID = "tw.fetch.catalog-v0"
	policy.MaxBodyBytes = origin.MaxResponseBytes
	policy.RequestTimeout = time.Duration(origin.TimeoutSeconds) * time.Second
	policy.UserAgent = "TWIRXLab/0.1 (+https://twirx.org; contact:rick@twirx.org)"
	policy.AllowedHosts = []string{origin.AllowedHost}
	fetcher, err := safefetch.New(policy)
	if err != nil {
		return nil, "", err
	}
	fetched, err := fetcher.Fetch(ctx, requestURL)
	if err != nil {
		return nil, "", err
	}
	return fetched, policy.ID, nil
}

func extractFields(body []byte, op *twircontract.Operation) ([]e2format.Field, string, error) {
	var document any
	policy := jsonbounded.Policy{MaxBytes: 2 << 20, MaxDepth: 64, MaxScalarBytes: 64 << 10, MaxContainerEntries: 16384, MaxTokens: 65536}
	if err := jsonbounded.Decode(body, &document, policy, false); err != nil {
		return nil, "", fmt.Errorf("labengine: parse observed JSON: %w", err)
	}
	fields := make([]e2format.Field, 0, len(op.Output))
	partial := false
	for _, spec := range op.Output {
		value, found, err := resolvePointer(document, spec.NativeLocator)
		if err != nil {
			return nil, "", fmt.Errorf("labengine: field %q: %w", spec.ID, err)
		}
		if !found || value == nil {
			if spec.Required {
				return nil, "", fmt.Errorf("labengine: required field %q missing at %s", spec.ID, spec.NativeLocator)
			}
			partial = true
			fields = append(fields, e2format.Field{ID: spec.ID, Status: "unresolved", NativeTerm: spec.NativeTerm, NativeLocator: spec.NativeLocator, SemanticTerm: spec.SemanticTerm, SemanticType: spec.Type, Transforms: append([]string(nil), spec.Transforms...), Mapping: spec.Mapping})
			continue
		}
		native, err := lexicalValue(value, spec.Type)
		if err != nil {
			return nil, "", fmt.Errorf("labengine: field %q native value: %w", spec.ID, err)
		}
		semantic, err := applyTransforms(native, spec.Transforms)
		if err != nil {
			return nil, "", fmt.Errorf("labengine: field %q transform: %w", spec.ID, err)
		}
		if err := validateType(semantic, spec.Type); err != nil {
			return nil, "", fmt.Errorf("labengine: field %q: %w", spec.ID, err)
		}
		fields = append(fields, e2format.Field{ID: spec.ID, Status: "resolved", NativeTerm: spec.NativeTerm, NativeLocator: spec.NativeLocator, NativePresent: true, NativeLexical: native, SemanticTerm: spec.SemanticTerm, SemanticType: spec.Type, SemanticPresent: true, SemanticLexical: semantic, Transforms: append([]string(nil), spec.Transforms...), Mapping: spec.Mapping})
	}
	status := "resolved"
	if partial {
		status = "partial"
	}
	return fields, status, nil
}

func resolvePointer(document any, pointer string) (any, bool, error) {
	if pointer == "" {
		return document, true, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, false, errors.New("JSON pointer must begin with slash")
	}
	current := document
	for _, raw := range strings.Split(pointer[1:], "/") {
		token, err := decodeToken(raw)
		if err != nil {
			return nil, false, err
		}
		switch node := current.(type) {
		case map[string]any:
			value, ok := node[token]
			if !ok {
				return nil, false, nil
			}
			current = value
		case []any:
			if token == "" || (len(token) > 1 && token[0] == '0') {
				return nil, false, errors.New("invalid array index")
			}
			index, parseErr := strconv.ParseUint(token, 10, 31)
			if parseErr != nil || int(index) >= len(node) {
				if parseErr == nil {
					return nil, false, nil
				}
				return nil, false, errors.New("invalid array index")
			}
			current = node[index]
		default:
			return nil, false, nil
		}
	}
	return current, true, nil
}

func decodeToken(raw string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] != '~' {
			out.WriteByte(raw[i])
			continue
		}
		if i+1 >= len(raw) {
			return "", errors.New("terminal tilde in JSON pointer")
		}
		i++
		switch raw[i] {
		case '0':
			out.WriteByte('~')
		case '1':
			out.WriteByte('/')
		default:
			return "", errors.New("invalid JSON pointer escape")
		}
	}
	return out.String(), nil
}

func lexicalValue(value any, valueType string) (string, error) {
	if valueType == "list:string" {
		list, ok := value.([]any)
		if !ok {
			return "", errors.New("expected array")
		}
		for _, item := range list {
			if _, ok := item.(string); !ok {
				return "", errors.New("list contains non-string value")
			}
		}
		encoded, err := json.Marshal(list)
		return string(encoded), err
	}
	switch typed := value.(type) {
	case string:
		return typed, nil
	case json.Number:
		return typed.String(), nil
	case bool:
		return strconv.FormatBool(typed), nil
	default:
		return "", fmt.Errorf("object or array cannot be a scalar (%T)", value)
	}
}

func applyTransforms(value string, transforms []string) (string, error) {
	out := value
	for _, transform := range transforms {
		switch transform {
		case "trim":
			out = strings.Trim(out, " \t\r\n")
		case "uppercase":
			out = asciiCase(out, true)
		case "lowercase":
			out = asciiCase(out, false)
		case "decimal_string":
			if !decimalPattern.MatchString(out) {
				return "", errors.New("not a canonical decimal lexical form")
			}
		default:
			return "", fmt.Errorf("unsupported transform %q", transform)
		}
	}
	return out, nil
}

func asciiCase(value string, upper bool) string {
	data := []byte(value)
	for i, char := range data {
		if upper && char >= 'a' && char <= 'z' {
			data[i] = char - ('a' - 'A')
		}
		if !upper && char >= 'A' && char <= 'Z' {
			data[i] = char + ('a' - 'A')
		}
	}
	return string(data)
}

func validateType(value, valueType string) error {
	switch valueType {
	case "string":
		return nil
	case "decimal":
		if !decimalPattern.MatchString(value) {
			return errors.New("not a decimal")
		}
	case "currency_code":
		if !currencyPattern.MatchString(value) {
			return errors.New("not a currency code")
		}
	case "integer":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return errors.New("not an integer")
		}
	case "list:string":
		var list []string
		if err := json.Unmarshal([]byte(value), &list); err != nil {
			return errors.New("not a string list")
		}
	default:
		return errors.New("unsupported type")
	}
	return nil
}

func deriveInvocationID(operationID string, input, observed []byte) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(operationID))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(input)
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(observed)
	return "inv-" + hex.EncodeToString(hash.Sum(nil)[:16])
}

func sameInput(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

type transcript struct {
	Format            string            `json:"format"`
	Mode              string            `json:"mode"`
	OriginID          string            `json:"origin_id"`
	OperationID       string            `json:"operation_id"`
	Input             map[string]string `json:"input"`
	InvocationID      string            `json:"invocation_id"`
	ObservedAt        string            `json:"observed_at"`
	ObservationDigest string            `json:"observation_digest"`
	ResultDigest      string            `json:"result_digest"`
	Statement         string            `json:"statement"`
}

func marshalTranscript(request Request, origin *origincatalog.Origin, op *twircontract.Operation, result e2format.Result, envelope *observation.Envelope, resultDigest string) ([]byte, error) {
	value := transcript{Format: "tw.execution-transcript/0.1", Mode: request.Mode, OriginID: origin.ID, OperationID: op.ID, Input: request.Input, InvocationID: result.InvocationID, ObservedAt: envelope.RetrievedAt, ObservationDigest: e2format.DigestReference(result.ObservationDigest), ResultDigest: resultDigest, Statement: "Records origin representation and declared derivation; does not assert objective truth."}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	encoded = append(encoded, '\n')
	if len(encoded) > MaxTranscriptBytes {
		return nil, errors.New("labengine: transcript exceeds bound")
	}
	return encoded, nil
}

// Verify performs the primary Go verification, including re-extraction from
// the immutable representation and comparison to the admitted contract.
func (e *Engine) Verify(dir string) (*e2format.Result, proofbundle.Publication, error) {
	publication, err := proofbundle.Verify(dir)
	if err != nil {
		return nil, publication, err
	}
	read := func(name string) ([]byte, error) {
		data, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			return nil, readErr
		}
		if len(data) > proofbundle.MaxArtifactBytes {
			return nil, errors.New("labengine: artifact exceeds bound")
		}
		return data, nil
	}
	resultBytes, err := read("result.cbor")
	if err != nil {
		return nil, publication, err
	}
	result, err := e2format.UnmarshalResult(resultBytes)
	if err != nil {
		return nil, publication, err
	}
	op, err := e.Contracts.Find(result.OperationID)
	if err != nil {
		return nil, publication, err
	}
	origin, err := e.Catalog.Find(result.OriginID)
	if err != nil {
		return nil, publication, err
	}
	if op.OriginID != origin.ID || op.Version != result.OperationVersion || origin.Version != result.OriginVersion {
		return nil, publication, errors.New("labengine: result identity is outside admitted contract")
	}
	inputBytes, err := read("input.cbor")
	if err != nil {
		return nil, publication, err
	}
	input, err := twircontract.UnmarshalInput(op, inputBytes)
	if err != nil {
		return nil, publication, err
	}
	if sha256.Sum256(inputBytes) != result.InputDigest {
		return nil, publication, errors.New("labengine: input digest mismatch")
	}
	observationBytes, err := read("observation.cbor")
	if err != nil {
		return nil, publication, err
	}
	envelope, err := observation.UnmarshalCBOR(observationBytes)
	if err != nil {
		return nil, publication, err
	}
	if sha256.Sum256(observationBytes) != result.ObservationDigest || envelope.RetrievedAt != result.ObservedAt {
		return nil, publication, errors.New("labengine: observation binding mismatch")
	}
	body, err := read("representation.body")
	if err != nil {
		return nil, publication, err
	}
	bodyDigest := sha256.Sum256(body)
	if bodyDigest != envelope.BodySHA256 || uint64(len(body)) != envelope.BodySize {
		return nil, publication, errors.New("labengine: representation binding mismatch")
	}
	transportBytes, err := read("transport.cbor")
	if err != nil {
		return nil, publication, err
	}
	transport, err := transportevidence.UnmarshalCBOR(transportBytes)
	if err != nil {
		return nil, publication, err
	}
	if sha256.Sum256(transportBytes) != result.TransportDigest || transport.RequestURL != envelope.RequestURL || transport.FinalURL != envelope.FinalURL || transport.PolicyID != envelope.PolicyID {
		return nil, publication, errors.New("labengine: transport binding mismatch")
	}
	contractBytes, err := e.Contracts.MarshalOperation(op)
	if err != nil {
		return nil, publication, err
	}
	storedContract, err := read("contract.cbor")
	if err != nil || !bytes.Equal(contractBytes, storedContract) || sha256.Sum256(storedContract) != result.ContractDigest {
		return nil, publication, errors.New("labengine: contract binding mismatch")
	}
	adapterBytes, err := twircontract.MarshalAdapterDescriptor(op)
	if err != nil {
		return nil, publication, err
	}
	storedAdapter, err := read("adapter.cbor")
	if err != nil || !bytes.Equal(adapterBytes, storedAdapter) || sha256.Sum256(storedAdapter) != result.AdapterDigest {
		return nil, publication, errors.New("labengine: adapter binding mismatch")
	}
	closureBytes, err := twircontract.MarshalSemanticClosure(op.SemanticClosure)
	if err != nil {
		return nil, publication, err
	}
	storedClosure, err := read("semantic-closure.cbor")
	if err != nil || !bytes.Equal(closureBytes, storedClosure) || sha256.Sum256(storedClosure) != result.SemanticClosureDigest {
		return nil, publication, errors.New("labengine: semantic closure binding mismatch")
	}
	expectedURL, err := origin.RequestURL(op, input)
	if err != nil || expectedURL != envelope.RequestURL {
		return nil, publication, errors.New("labengine: observation request is outside catalog")
	}
	expectedFields, expectedStatus, err := extractFields(body, op)
	if err != nil {
		return nil, publication, err
	}
	expected := result
	expected.Fields = expectedFields
	expected.Status = expectedStatus
	expectedBytes, err := e2format.MarshalResult(expected)
	if err != nil || !bytes.Equal(expectedBytes, resultBytes) {
		return nil, publication, errors.New("labengine: re-extracted result differs")
	}
	return &result, publication, nil
}
