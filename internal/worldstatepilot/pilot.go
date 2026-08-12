// Package worldstatepilot prepares and admits one bounded E4 World State
// acquisition. It derives every destination from the reviewed E2 operation
// contract and origin catalog. It never accepts a caller-supplied URL and it
// performs no network access.
package worldstatepilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/admission"
	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/egressworker"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/origincatalog"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

const (
	PlanFormat     = "tw.e4-world-state-pilot/0.1"
	PreparedFormat = "tw.e4-world-state-prepared/0.1"
	OperationID    = "development.getIndicator"
	OriginID       = "api-worldbank-org"
	ExecutionAlias = "world-bank-indicators"
	MaxPlanBytes   = 64 << 10
	MaxPilotOrders = 128
	MaxOutputFile  = 1 << 20
)

type Plan struct {
	Format                string   `json:"format"`
	ID                    string   `json:"id"`
	OperationID           string   `json:"operation_id"`
	ExecutionMode         string   `json:"execution_mode"`
	Countries             []string `json:"countries"`
	Indicators            []string `json:"indicators"`
	Years                 []string `json:"years"`
	NotBefore             string   `json:"not_before"`
	ExpiresAt             string   `json:"expires_at"`
	RequestIntervalMillis int      `json:"request_interval_ms"`
	MaximumRequests       int      `json:"maximum_requests"`
	MaximumTotalBytes     int64    `json:"maximum_total_bytes"`
	Retention             string   `json:"retention"`
	SchedulerEnabled      bool     `json:"scheduler_enabled"`
}

type PreparedOrder struct {
	ID         string            `json:"id"`
	Digest     string            `json:"digest"`
	Input      map[string]string `json:"input"`
	RequestURL string            `json:"request_url"`
}

type Prepared struct {
	Format               string          `json:"format"`
	PlanID               string          `json:"plan_id"`
	OperationID          string          `json:"operation_id"`
	OriginID             string          `json:"origin_id"`
	ExecutionAlias       string          `json:"execution_alias"`
	DecisionDigest       string          `json:"decision_digest"`
	PolicyEvidenceDigest string          `json:"policy_evidence_digest"`
	ContractDigest       string          `json:"contract_digest"`
	CatalogDigest        string          `json:"catalog_digest"`
	RequestIntervalMS    int             `json:"request_interval_ms"`
	MaximumTotalBytes    int64           `json:"maximum_total_bytes"`
	SchedulerEnabled     bool            `json:"scheduler_enabled"`
	Orders               []PreparedOrder `json:"orders"`
}

type ExecutionEntry struct {
	OrderID              string `json:"order_id"`
	WorkOrderDigest      string `json:"work_order_digest"`
	ObservationDigest    string `json:"observation_digest"`
	RepresentationDigest string `json:"representation_digest"`
	BodySize             uint64 `json:"body_size"`
	RetrievedAt          string `json:"retrieved_at"`
	HTTPStatus           int    `json:"http_status"`
	MediaType            string `json:"media_type"`
	CompilationEligible  bool   `json:"compilation_eligible"`
	RejectionCode        string `json:"rejection_code,omitempty"`
	NetworkExecuted      bool   `json:"network_executed"`
}

type ExecutionSummary struct {
	Format            string           `json:"format"`
	PlanID            string           `json:"plan_id"`
	OriginID          string           `json:"origin_id"`
	StartedAt         string           `json:"started_at"`
	CompletedAt       string           `json:"completed_at"`
	NetworkRequests   int              `json:"network_requests"`
	ReusedSpools      int              `json:"reused_spools"`
	TransferredBytes  uint64           `json:"transferred_bytes"`
	EligibleResponses int              `json:"eligible_responses"`
	RejectedResponses int              `json:"rejected_responses"`
	SchedulerEnabled  bool             `json:"scheduler_enabled"`
	Entries           []ExecutionEntry `json:"entries"`
}

func LoadPlan(path string) (Plan, []byte, error) {
	var plan Plan
	data, err := readRegular(path, MaxPlanBytes)
	if err != nil {
		return plan, nil, fmt.Errorf("worldstatepilot: read plan: %w", err)
	}
	policy := jsonbounded.Policy{MaxBytes: MaxPlanBytes, MaxDepth: 8, MaxScalarBytes: 16 << 10, MaxContainerEntries: 512, MaxTokens: 4096}
	if err := jsonbounded.Decode(data, &plan, policy, true); err != nil {
		return plan, nil, fmt.Errorf("worldstatepilot: decode plan: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return plan, nil, err
	}
	return plan, data, nil
}

func (plan Plan) Validate() error {
	if plan.Format != PlanFormat || plan.ID != "world-bank-e2-matrix-2026-08" || plan.OperationID != OperationID || plan.ExecutionMode != "manual_once" {
		return errors.New("worldstatepilot: unsupported plan identity or execution mode")
	}
	if plan.SchedulerEnabled {
		return errors.New("worldstatepilot: scheduler must remain disabled")
	}
	if !sortedUnique(plan.Countries) || !sortedUnique(plan.Indicators) || !sortedUnique(plan.Years) {
		return errors.New("worldstatepilot: plan dimensions must be sorted, unique, and non-empty")
	}
	count := len(plan.Countries) * len(plan.Indicators) * len(plan.Years)
	if count < 1 || count > MaxPilotOrders || plan.MaximumRequests != count {
		return errors.New("worldstatepilot: maximum request count must equal the bounded Cartesian product")
	}
	if plan.RequestIntervalMillis < 5000 || plan.RequestIntervalMillis > 60000 {
		return errors.New("worldstatepilot: request interval outside 5..60 seconds")
	}
	if plan.MaximumTotalBytes < int64(count) || plan.MaximumTotalBytes > 64<<20 {
		return errors.New("worldstatepilot: total byte budget outside bounds")
	}
	if plan.Retention != "public_versioned_immutable_evidence" {
		return errors.New("worldstatepilot: unsupported retention policy")
	}
	start, err := canonicalTime(plan.NotBefore)
	if err != nil {
		return fmt.Errorf("worldstatepilot: not_before: %w", err)
	}
	end, err := canonicalTime(plan.ExpiresAt)
	if err != nil || !end.After(start) || end.Sub(start) > 48*time.Hour {
		return errors.New("worldstatepilot: invalid or excessive validity interval")
	}
	return nil
}

// Prepare writes immutable, source-specific work orders. The generic egress
// worker later receives only a sealed order ID; it never receives an input or
// URL from an operator or public caller.
func Prepare(root, planPath, output string) (Prepared, error) {
	var prepared Prepared
	plan, _, err := LoadPlan(planPath)
	if err != nil {
		return prepared, err
	}
	contractPath := filepath.Join(root, "contracts", "e2", "contracts.json")
	catalogPath := filepath.Join(root, "origins", "catalog.json")
	contractBytes, err := readRegular(contractPath, twircontract.MaxContractBytes)
	if err != nil {
		return prepared, err
	}
	catalogBytes, err := readRegular(catalogPath, origincatalog.MaxCatalogBytes)
	if err != nil {
		return prepared, err
	}
	contracts, err := twircontract.Load(contractPath)
	if err != nil {
		return prepared, err
	}
	operation, err := contracts.Find(OperationID)
	if err != nil {
		return prepared, err
	}
	catalog, err := origincatalog.Load(catalogPath)
	if err != nil {
		return prepared, err
	}
	origin, err := catalog.ForOperation(OperationID)
	if err != nil {
		return prepared, err
	}
	if origin.ID != ExecutionAlias || origin.RegistryID != OriginID || operation.OriginID != ExecutionAlias || origin.AllowedHost != "api.worldbank.org" || origin.RequestsPerMinute > 12 {
		return prepared, errors.New("worldstatepilot: E2 origin binding or rate ceiling changed")
	}
	if plan.RequestIntervalMillis < 60000/origin.RequestsPerMinute {
		return prepared, errors.New("worldstatepilot: plan exceeds catalog request rate")
	}
	for _, dimension := range []struct {
		name   string
		values []string
	}{
		{name: "country", values: plan.Countries},
		{name: "indicator", values: plan.Indicators},
		{name: "year", values: plan.Years},
	} {
		field, fieldErr := inputField(operation, dimension.name)
		if fieldErr != nil || !subset(dimension.values, field.Allowed) {
			return prepared, fmt.Errorf("worldstatepilot: plan %s values exceed the reviewed E2 allowlist", dimension.name)
		}
	}

	selection, err := atlas.LoadSelection(filepath.Join(root, "atlas", "genesis-500", "selection.json"))
	if err != nil {
		return prepared, err
	}
	sources, err := admission.Load(filepath.Join(root, "atlas", "admissions"), selection)
	if err != nil {
		return prepared, err
	}
	source, err := sourceByID(sources, OriginID)
	if err != nil {
		return prepared, err
	}
	if source.Decision.ReviewState != admission.ReviewCompleted || source.Decision.PolicyReviewState != atlas.PolicyCompleted || source.Decision.PolicyDecision != atlas.DecisionPermitWithConstraints || source.Policy.Authentication != "none_required" || !contains(source.Record.ExecutionCatalogIDs, ExecutionAlias) {
		return prepared, errors.New("worldstatepilot: exact human policy authority is absent or changed")
	}
	if source.Decision.ApprovalReference != "reports/futo-policy-decision-proposals.md" || !contains(source.Decision.Constraints, "Only the documented E2 country, indicator and year route is permitted.") {
		return prepared, errors.New("worldstatepilot: founder scope does not authorize this route family")
	}

	if _, err := os.Lstat(output); err == nil {
		return prepared, errors.New("worldstatepilot: prepared output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return prepared, err
	}
	if err := os.Mkdir(output, 0o750); err != nil {
		return prepared, err
	}
	ordersDir := filepath.Join(output, "work-orders")
	if err := os.Mkdir(ordersDir, 0o750); err != nil {
		return prepared, err
	}
	prepared = Prepared{
		Format: PreparedFormat, PlanID: plan.ID, OperationID: OperationID, OriginID: OriginID, ExecutionAlias: ExecutionAlias,
		DecisionDigest: source.DecisionDigest, PolicyEvidenceDigest: source.PolicyDigest, ContractDigest: digest(contractBytes), CatalogDigest: digest(catalogBytes),
		RequestIntervalMS: plan.RequestIntervalMillis, MaximumTotalBytes: plan.MaximumTotalBytes, SchedulerEnabled: false,
	}
	for _, country := range plan.Countries {
		for _, indicator := range plan.Indicators {
			for _, year := range plan.Years {
				input := map[string]string{"country": country, "indicator": indicator, "year": year}
				requestURL, requestErr := origin.RequestURL(operation, input)
				if requestErr != nil {
					return Prepared{}, requestErr
				}
				id := orderID(country, indicator, year)
				order := egressworker.WorkOrder{
					Format: egressworker.WorkOrderFormat, ID: id, OriginID: OriginID, Purpose: "observation", AuthorityClass: "reviewed_policy", Method: "GET", URL: requestURL,
					AllowedHosts: []string{"api.worldbank.org"}, PolicyDecision: string(atlas.DecisionPermitWithConstraints), PolicyEvidenceDigest: source.PolicyDigest, DecisionDigest: source.DecisionDigest,
					ApprovalReference: source.Decision.ApprovalReference, NotBefore: plan.NotBefore, ExpiresAt: plan.ExpiresAt, MaxRedirects: 0, MaxBodyBytes: origin.MaxResponseBytes,
					TimeoutMillis: origin.TimeoutSeconds * 1000, ConnectTimeoutMillis: 5000, HeaderTimeoutMillis: 10000, MaxConsecutiveFailures: 2, CircuitCooldownSeconds: 3600,
				}
				if err := order.Validate(); err != nil {
					return Prepared{}, fmt.Errorf("worldstatepilot: generated order %s: %w", id, err)
				}
				encoded, err := marshal(order)
				if err != nil {
					return Prepared{}, err
				}
				if err := atomicfile.Write(filepath.Join(ordersDir, id+".json"), encoded, egressworker.MaxWorkOrder, 0o440); err != nil {
					return Prepared{}, err
				}
				prepared.Orders = append(prepared.Orders, PreparedOrder{ID: id, Digest: digest(encoded), Input: input, RequestURL: requestURL})
			}
		}
	}
	if len(prepared.Orders) != plan.MaximumRequests {
		return Prepared{}, errors.New("worldstatepilot: generated order count disagrees with plan")
	}
	control := egressworker.Control{Format: egressworker.ControlFormat, Enabled: true, EmergencyStop: false, RevokedOrigins: []string{}, RevokedOrders: []string{}}
	controlBytes, err := marshal(control)
	if err != nil {
		return Prepared{}, err
	}
	if err := atomicfile.Write(filepath.Join(output, "manual-control.json"), controlBytes, egressworker.MaxControl, 0o440); err != nil {
		return Prepared{}, err
	}
	preparedBytes, err := marshal(prepared)
	if err != nil {
		return Prepared{}, err
	}
	// Prepared manifest is written last. The scheduler remains disabled; this
	// control artifact permits only explicit one-at-a-time order execution.
	if err := atomicfile.Write(filepath.Join(output, "prepared-manifest.json"), preparedBytes, MaxOutputFile, 0o440); err != nil {
		return Prepared{}, err
	}
	return prepared, nil
}

// ExecutePrepared runs the exact prepared work orders sequentially. A valid
// completed spool is reused without network access, allowing safe recovery
// after interruption. The immutable acquisition summary is published last.
func ExecutePrepared(ctx context.Context, root, planPath, preparedRoot, spoolRoot, stateRoot string, now func() time.Time, wait func(time.Duration)) (ExecutionSummary, error) {
	var summary ExecutionSummary
	if ctx == nil || now == nil || wait == nil {
		return summary, errors.New("worldstatepilot: context, clock, and wait function are required")
	}
	plan, _, err := LoadPlan(planPath)
	if err != nil {
		return summary, err
	}
	preparedBytes, err := readRegular(filepath.Join(preparedRoot, "prepared-manifest.json"), MaxOutputFile)
	if err != nil {
		return summary, err
	}
	policy := jsonbounded.Policy{MaxBytes: MaxOutputFile, MaxDepth: 12, MaxScalarBytes: 32 << 10, MaxContainerEntries: 4096, MaxTokens: 20000}
	var prepared Prepared
	if err := jsonbounded.Decode(preparedBytes, &prepared, policy, true); err != nil {
		return summary, err
	}
	if err := verifyPrepared(root, plan, preparedRoot, prepared); err != nil {
		return summary, err
	}
	if pathExists(filepath.Join(preparedRoot, "acquisition-summary.json")) {
		return summary, errors.New("worldstatepilot: immutable acquisition summary already exists")
	}
	control, err := egressworker.LoadControl(filepath.Join(preparedRoot, "manual-control.json"))
	if err != nil || !control.Enabled || control.EmergencyStop {
		return summary, errors.New("worldstatepilot: exact manual control is not enabled")
	}
	started := now().UTC().Truncate(time.Second)
	summary = ExecutionSummary{Format: "tw.e4-world-state-execution/0.1", PlanID: plan.ID, OriginID: OriginID, StartedAt: started.Format(time.RFC3339), SchedulerEnabled: false}
	for index, expected := range prepared.Orders {
		if err := ctx.Err(); err != nil {
			return ExecutionSummary{}, err
		}
		loaded, err := egressworker.LoadWorkOrder(filepath.Join(preparedRoot, "work-orders"), expected.ID)
		if err != nil {
			return ExecutionSummary{}, err
		}
		spool := filepath.Join(spoolRoot, expected.ID, strings.TrimPrefix(loaded.Digest, "sha256:"))
		verified, verifyErr := egressworker.VerifySpool(spool, loaded.Order.MaxBodyBytes)
		networkExecuted := false
		if verifyErr != nil {
			if !errors.Is(verifyErr, os.ErrNotExist) && pathExists(spool) {
				return ExecutionSummary{}, fmt.Errorf("worldstatepilot: existing spool %s is invalid: %w", expected.ID, verifyErr)
			}
			if index > 0 && summary.NetworkRequests > 0 {
				wait(time.Duration(plan.RequestIntervalMillis) * time.Millisecond)
			}
			verified, err = egressworker.Execute(ctx, loaded, control, spoolRoot, stateRoot, now().UTC())
			if err != nil {
				return ExecutionSummary{}, fmt.Errorf("worldstatepilot: execute %s: %w", expected.ID, err)
			}
			networkExecuted = true
			summary.NetworkRequests++
		} else {
			summary.ReusedSpools++
		}
		if verified.WorkOrderDigest != expected.Digest || verified.RequestURL != expected.RequestURL {
			return ExecutionSummary{}, fmt.Errorf("worldstatepilot: result %s disagrees with sealed order", expected.ID)
		}
		summary.TransferredBytes += verified.BodySize
		if int64(summary.TransferredBytes) > prepared.MaximumTotalBytes {
			return ExecutionSummary{}, errors.New("worldstatepilot: acquisition exceeded total byte budget")
		}
		eligible := verified.HTTPStatus == 200 && verified.MediaType == "application/json"
		rejection := ""
		if eligible {
			summary.EligibleResponses++
		} else {
			summary.RejectedResponses++
			rejection = "response_not_200_json"
		}
		summary.Entries = append(summary.Entries, ExecutionEntry{OrderID: expected.ID, WorkOrderDigest: verified.WorkOrderDigest, ObservationDigest: verified.ObservationDigest, RepresentationDigest: verified.BodyDigest, BodySize: verified.BodySize, RetrievedAt: verified.RetrievedAt, HTTPStatus: verified.HTTPStatus, MediaType: verified.MediaType, CompilationEligible: eligible, RejectionCode: rejection, NetworkExecuted: networkExecuted})
	}
	summary.CompletedAt = now().UTC().Truncate(time.Second).Format(time.RFC3339)
	encoded, err := marshal(summary)
	if err != nil {
		return ExecutionSummary{}, err
	}
	if err := atomicfile.Write(filepath.Join(preparedRoot, "acquisition-summary.json"), encoded, MaxOutputFile, 0o440); err != nil {
		return ExecutionSummary{}, err
	}
	return summary, nil
}

func verifyPrepared(root string, plan Plan, preparedRoot string, prepared Prepared) error {
	if prepared.Format != PreparedFormat || prepared.PlanID != plan.ID || prepared.OperationID != OperationID || prepared.OriginID != OriginID || prepared.ExecutionAlias != ExecutionAlias || prepared.SchedulerEnabled || prepared.RequestIntervalMS != plan.RequestIntervalMillis || prepared.MaximumTotalBytes != plan.MaximumTotalBytes || len(prepared.Orders) != plan.MaximumRequests {
		return errors.New("worldstatepilot: prepared manifest disagrees with plan")
	}
	contractPath := filepath.Join(root, "contracts", "e2", "contracts.json")
	catalogPath := filepath.Join(root, "origins", "catalog.json")
	contractBytes, err := readRegular(contractPath, twircontract.MaxContractBytes)
	if err != nil || digest(contractBytes) != prepared.ContractDigest {
		return errors.New("worldstatepilot: prepared contract identity changed")
	}
	catalogBytes, err := readRegular(catalogPath, origincatalog.MaxCatalogBytes)
	if err != nil || digest(catalogBytes) != prepared.CatalogDigest {
		return errors.New("worldstatepilot: prepared catalog identity changed")
	}
	selection, err := atlas.LoadSelection(filepath.Join(root, "atlas", "genesis-500", "selection.json"))
	if err != nil {
		return err
	}
	sources, err := admission.Load(filepath.Join(root, "atlas", "admissions"), selection)
	if err != nil {
		return err
	}
	source, err := sourceByID(sources, OriginID)
	if err != nil || source.DecisionDigest != prepared.DecisionDigest || source.PolicyDigest != prepared.PolicyEvidenceDigest {
		return errors.New("worldstatepilot: prepared authority identity changed")
	}
	contracts, err := twircontract.Load(contractPath)
	if err != nil {
		return err
	}
	operation, err := contracts.Find(OperationID)
	if err != nil {
		return err
	}
	catalog, err := origincatalog.Load(catalogPath)
	if err != nil {
		return err
	}
	origin, err := catalog.ForOperation(OperationID)
	if err != nil {
		return err
	}
	prior := ""
	for _, expected := range prepared.Orders {
		if expected.ID <= prior || expected.Digest == "" || len(expected.Input) != 3 {
			return errors.New("worldstatepilot: prepared orders are not sorted, unique, and exact")
		}
		prior = expected.ID
		requestURL, err := origin.RequestURL(operation, expected.Input)
		if err != nil || requestURL != expected.RequestURL || expected.ID != orderID(expected.Input["country"], expected.Input["indicator"], expected.Input["year"]) {
			return errors.New("worldstatepilot: prepared order escaped reviewed contract")
		}
		loaded, err := egressworker.LoadWorkOrder(filepath.Join(preparedRoot, "work-orders"), expected.ID)
		if err != nil || loaded.Digest != expected.Digest || loaded.Order.URL != expected.RequestURL || loaded.Order.DecisionDigest != prepared.DecisionDigest || loaded.Order.PolicyEvidenceDigest != prepared.PolicyEvidenceDigest {
			return errors.New("worldstatepilot: prepared work order does not match manifest authority")
		}
	}
	return nil
}

func inputField(operation *twircontract.Operation, name string) (twircontract.Field, error) {
	for _, field := range operation.Input {
		if field.ID == name {
			return field, nil
		}
	}
	return twircontract.Field{}, fmt.Errorf("missing input field %q", name)
}

func sourceByID(sources []admission.Source, id string) (admission.Source, error) {
	for _, source := range sources {
		if source.Record.ID == id {
			return source, nil
		}
	}
	return admission.Source{}, fmt.Errorf("worldstatepilot: admission source %q is absent", id)
}

func orderID(country, indicator, year string) string {
	return "e4-wb-" + strings.ToLower(country) + "-" + strings.ToLower(strings.ReplaceAll(indicator, ".", "-")) + "-" + year
}

func subset(values, allowed []string) bool {
	set := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		set[value] = struct{}{}
	}
	for _, value := range values {
		if _, exists := set[value]; !exists {
			return false
		}
	}
	return true
}

func sortedUnique(values []string) bool {
	if len(values) == 0 {
		return false
	}
	return sort.StringsAreSorted(values) && noDuplicates(values)
}

func noDuplicates(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] == value {
			return false
		}
	}
	return true
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func canonicalTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Location() != time.UTC || parsed.Format(time.RFC3339Nano) != value {
		return time.Time{}, errors.New("time must be canonical UTC RFC3339Nano")
	}
	return parsed, nil
}

func readRegular(path string, maximum int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > int64(maximum) {
		return nil, errors.New("artifact is not a bounded regular file")
	}
	return os.ReadFile(path)
}

func marshal(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
