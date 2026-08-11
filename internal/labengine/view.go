package labengine

import (
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/e2format"
	"github.com/typed-web-commons/typed-web/internal/proofbundle"
)

type ResultView struct {
	Format           string         `json:"format"`
	ResultID         string         `json:"result_id"`
	ResultDigest     string         `json:"result_digest"`
	BundleID         string         `json:"bundle_id"`
	InvocationID     string         `json:"invocation_id"`
	OriginID         string         `json:"origin_id"`
	OriginVersion    string         `json:"origin_version"`
	OperationID      string         `json:"operation_id"`
	OperationVersion string         `json:"operation_version"`
	Effect           string         `json:"effect"`
	Status           string         `json:"status"`
	ObservedAt       string         `json:"observed_at"`
	Mode             string         `json:"mode,omitempty"`
	Attribution      string         `json:"attribution,omitempty"`
	Bindings         DigestBindings `json:"bindings"`
	Fields           []FieldView    `json:"fields"`
}

type DigestBindings struct {
	Input           string `json:"input"`
	Observation     string `json:"observation"`
	Transport       string `json:"transport"`
	Adapter         string `json:"adapter"`
	Contract        string `json:"contract"`
	SemanticClosure string `json:"semantic_closure"`
}

type FieldView struct {
	ID         string         `json:"id"`
	Status     string         `json:"status"`
	Native     NativeView     `json:"native"`
	Semantic   SemanticView   `json:"semantic"`
	Derivation DerivationView `json:"derivation"`
}
type NativeView struct {
	Term    string  `json:"term"`
	Locator string  `json:"locator"`
	Lexical *string `json:"lexical,omitempty"`
}
type SemanticView struct {
	Term    string  `json:"term"`
	Type    string  `json:"type"`
	Lexical *string `json:"lexical,omitempty"`
}
type DerivationView struct {
	Transforms []string `json:"transforms"`
	Mapping    string   `json:"mapping"`
}

func View(invocation *Invocation) ResultView {
	return resultView(invocation.Result, invocation.Publication, invocation.Mode, invocation.Attribution)
}

func resultView(result e2format.Result, publication proofbundle.Publication, mode, attribution string) ResultView {
	view := ResultView{Format: e2format.ResultVersion, ResultID: publication.ResultID, ResultDigest: publication.ResultDigest, BundleID: publication.BundleID, InvocationID: result.InvocationID, OriginID: result.OriginID, OriginVersion: result.OriginVersion, OperationID: result.OperationID, OperationVersion: result.OperationVersion, Effect: result.Effect, Status: result.Status, ObservedAt: result.ObservedAt, Mode: mode, Attribution: attribution, Bindings: DigestBindings{Input: e2format.DigestReference(result.InputDigest), Observation: e2format.DigestReference(result.ObservationDigest), Transport: e2format.DigestReference(result.TransportDigest), Adapter: e2format.DigestReference(result.AdapterDigest), Contract: e2format.DigestReference(result.ContractDigest), SemanticClosure: e2format.DigestReference(result.SemanticClosureDigest)}, Fields: make([]FieldView, 0, len(result.Fields))}
	for _, field := range result.Fields {
		native := NativeView{Term: field.NativeTerm, Locator: field.NativeLocator}
		semantic := SemanticView{Term: field.SemanticTerm, Type: field.SemanticType}
		if field.NativePresent {
			value := field.NativeLexical
			native.Lexical = &value
		}
		if field.SemanticPresent {
			value := field.SemanticLexical
			semantic.Lexical = &value
		}
		view.Fields = append(view.Fields, FieldView{ID: field.ID, Status: field.Status, Native: native, Semantic: semantic, Derivation: DerivationView{Transforms: append([]string(nil), field.Transforms...), Mapping: field.Mapping}})
	}
	return view
}

func (e *Engine) Load(resultID string) (ResultView, string, error) {
	digest, err := parseResultID(resultID)
	if err != nil {
		return ResultView{}, "", err
	}
	dir := filepath.Join(e.ResultsDir, digest)
	result, publication, err := e.Verify(dir)
	if err != nil {
		return ResultView{}, "", err
	}
	origin, err := e.Catalog.Find(result.OriginID)
	if err != nil {
		return ResultView{}, "", err
	}
	return resultView(*result, publication, "", origin.Attribution), dir, nil
}

func parseResultID(value string) (string, error) {
	value = strings.TrimPrefix(value, "sha256:")
	if len(value) != 64 {
		return "", errors.New("labengine: result ID must be a SHA-256 reference")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return "", errors.New("labengine: result ID must be lowercase hexadecimal")
	}
	if hex.EncodeToString(decoded) != value {
		return "", errors.New("labengine: result ID must be lowercase hexadecimal")
	}
	return value, nil
}
