// Package egressworker executes sealed, read-only public-origin work orders.
// It accepts no caller-supplied destination and has no registry, database,
// deployment, secret, parser, adapter, or canon write capability.
package egressworker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/cas"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/observation"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

const (
	WorkOrderFormat = "tw.egress-work-order/0.1"
	ControlFormat   = "tw.egress-control/0.1"
	ResultFormat    = "tw.egress-result/0.1"
	ManifestFormat  = "tw.egress-manifest/0.1"
	CircuitFormat   = "tw.egress-circuit/0.1"
	WorkerPolicyID  = "tw.egress.public-origin-v0"
	MaxWorkOrder    = 64 << 10
	MaxControl      = 64 << 10
	MaxResult       = 128 << 10
	MaxBody         = 2 << 20
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,127}$`)

type WorkOrder struct {
	Format                 string   `json:"format"`
	ID                     string   `json:"id"`
	OriginID               string   `json:"origin_id"`
	Purpose                string   `json:"purpose"`
	AuthorityClass         string   `json:"authority_class"`
	Method                 string   `json:"method"`
	URL                    string   `json:"url"`
	AllowedHosts           []string `json:"allowed_hosts"`
	PolicyDecision         string   `json:"policy_decision"`
	PolicyEvidenceDigest   string   `json:"policy_evidence_digest"`
	DecisionDigest         string   `json:"decision_digest"`
	ApprovalReference      string   `json:"approval_reference"`
	NotBefore              string   `json:"not_before"`
	ExpiresAt              string   `json:"expires_at"`
	MaxRedirects           int      `json:"max_redirects"`
	MaxBodyBytes           int64    `json:"max_body_bytes"`
	TimeoutMillis          int      `json:"timeout_ms"`
	ConnectTimeoutMillis   int      `json:"connect_timeout_ms"`
	HeaderTimeoutMillis    int      `json:"header_timeout_ms"`
	MaxConsecutiveFailures int      `json:"max_consecutive_failures"`
	CircuitCooldownSeconds int      `json:"circuit_cooldown_seconds"`
}

type LoadedWorkOrder struct {
	Order  WorkOrder
	Digest string
	bytes  []byte
}

type Control struct {
	Format         string   `json:"format"`
	Enabled        bool     `json:"enabled"`
	EmergencyStop  bool     `json:"emergency_stop"`
	RevokedOrigins []string `json:"revoked_origins"`
	RevokedOrders  []string `json:"revoked_orders"`
}

type Circuit struct {
	Format              string  `json:"format"`
	OriginID            string  `json:"origin_id"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	OpenUntil           *string `json:"open_until"`
}

type Result struct {
	Format            string `json:"format"`
	WorkOrderID       string `json:"work_order_id"`
	WorkOrderDigest   string `json:"work_order_digest"`
	OriginID          string `json:"origin_id"`
	Purpose           string `json:"purpose"`
	RequestURL        string `json:"request_url"`
	FinalURL          string `json:"final_url"`
	RetrievedAt       string `json:"retrieved_at"`
	HTTPStatus        int    `json:"http_status"`
	MediaType         string `json:"media_type"`
	RedirectCount     int    `json:"redirect_count"`
	ObservationDigest string `json:"observation_digest"`
	BodyDigest        string `json:"body_digest"`
	BodySize          uint64 `json:"body_size"`
	DurationMillis    int64  `json:"duration_ms"`
}

type Manifest struct {
	Format          string     `json:"format"`
	WorkOrderID     string     `json:"work_order_id"`
	WorkOrderDigest string     `json:"work_order_digest"`
	Artifacts       []Artifact `json:"artifacts"`
}

type Artifact struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type retriever interface {
	Fetch(context.Context, string) (*safefetch.Result, error)
}

func LoadWorkOrder(root, id string) (*LoadedWorkOrder, error) {
	if !idPattern.MatchString(id) {
		return nil, errors.New("egress worker: invalid work-order ID")
	}
	path := filepath.Join(root, id+".json")
	data, err := readRegular(path, MaxWorkOrder)
	if err != nil {
		return nil, fmt.Errorf("egress worker: read work order: %w", err)
	}
	var order WorkOrder
	if err := decode(data, &order, MaxWorkOrder); err != nil {
		return nil, fmt.Errorf("egress worker: decode work order: %w", err)
	}
	if order.ID != id {
		return nil, errors.New("egress worker: work-order filename and identity disagree")
	}
	if err := order.Validate(); err != nil {
		return nil, err
	}
	return &LoadedWorkOrder{Order: order, Digest: digest(data), bytes: append([]byte(nil), data...)}, nil
}

func LoadControl(path string) (*Control, error) {
	data, err := readRegular(path, MaxControl)
	if err != nil {
		return nil, fmt.Errorf("egress worker: read control: %w", err)
	}
	var control Control
	if err := decode(data, &control, MaxControl); err != nil {
		return nil, fmt.Errorf("egress worker: decode control: %w", err)
	}
	if err := control.Validate(); err != nil {
		return nil, err
	}
	return &control, nil
}

func (c Control) Validate() error {
	if c.Format != ControlFormat || len(c.RevokedOrigins) > 500 || len(c.RevokedOrders) > 500 || !sortedUnique(c.RevokedOrigins) || !sortedUnique(c.RevokedOrders) || !validIDs(c.RevokedOrigins) || !validIDs(c.RevokedOrders) {
		return errors.New("egress worker: invalid control artifact")
	}
	return nil
}

func (w WorkOrder) Validate() error {
	if w.Format != WorkOrderFormat || !idPattern.MatchString(w.ID) || !idPattern.MatchString(w.OriginID) || w.Method != "GET" {
		return errors.New("egress worker: invalid work-order identity or method")
	}
	if w.Purpose != "robots" && w.Purpose != "profile" && w.Purpose != "observation" {
		return errors.New("egress worker: invalid work-order purpose")
	}
	switch w.AuthorityClass {
	case "policy_evidence_collection":
		if w.Purpose != "robots" || w.PolicyDecision != "uncertain" {
			return errors.New("egress worker: evidence-collection authority is limited to robots under pending policy")
		}
	case "reviewed_policy":
		if w.PolicyDecision != "permit_live" && w.PolicyDecision != "permit_with_constraints" && w.PolicyDecision != "profile_only" {
			return errors.New("egress worker: reviewed policy does not permit retrieval")
		}
	default:
		return errors.New("egress worker: invalid work-order authority class")
	}
	for _, value := range []string{w.PolicyEvidenceDigest, w.DecisionDigest} {
		if !validDigest(value) {
			return errors.New("egress worker: invalid authority digest")
		}
	}
	if w.ApprovalReference == "" || len(w.ApprovalReference) > 4096 || !sortedUnique(w.AllowedHosts) || len(w.AllowedHosts) == 0 || len(w.AllowedHosts) > 8 {
		return errors.New("egress worker: missing approval or invalid host allowlist")
	}
	parsed, err := url.Parse(w.URL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Opaque != "" || parsed.Fragment != "" || parsed.Hostname() == "" || parsed.RawPath != "" || parsed.String() != w.URL {
		return errors.New("egress worker: URL must be credential-free HTTPS")
	}
	if parsed.Port() != "" && parsed.Port() != "443" || ambiguousNumericHost(parsed.Hostname()) {
		return errors.New("egress worker: non-standard port or numeric host is forbidden")
	}
	hostAllowed := false
	for _, host := range w.AllowedHosts {
		if host != strings.ToLower(host) || !validHostname(host) || ambiguousNumericHost(host) {
			return errors.New("egress worker: allowed hosts must be canonical lower-case names")
		}
		if strings.EqualFold(host, parsed.Hostname()) {
			hostAllowed = true
		}
	}
	if !hostAllowed {
		return errors.New("egress worker: URL host is not admitted")
	}
	if w.Purpose == "robots" && parsed.Path != "/robots.txt" {
		return errors.New("egress worker: robots work order must target /robots.txt")
	}
	if w.AuthorityClass == "reviewed_policy" && w.PolicyDecision == "profile_only" && w.Purpose == "observation" {
		return errors.New("egress worker: profile-only policy cannot authorize an observation")
	}
	if w.MaxRedirects < 0 || w.MaxRedirects > 3 || w.MaxBodyBytes <= 0 || w.MaxBodyBytes > MaxBody || w.TimeoutMillis < 1000 || w.TimeoutMillis > 30000 || w.ConnectTimeoutMillis < 500 || w.ConnectTimeoutMillis > 10000 || w.HeaderTimeoutMillis < 500 || w.HeaderTimeoutMillis > 15000 {
		return errors.New("egress worker: request, redirect, byte, or duration limit outside bounds")
	}
	if w.MaxConsecutiveFailures < 1 || w.MaxConsecutiveFailures > 10 || w.CircuitCooldownSeconds < 60 || w.CircuitCooldownSeconds > 86400 {
		return errors.New("egress worker: invalid circuit-breaker limits")
	}
	notBefore, err := canonicalTime(w.NotBefore)
	if err != nil {
		return fmt.Errorf("egress worker: not_before: %w", err)
	}
	expires, err := canonicalTime(w.ExpiresAt)
	if err != nil || !expires.After(notBefore) || expires.Sub(notBefore) > 31*24*time.Hour {
		return errors.New("egress worker: invalid work-order validity interval")
	}
	return nil
}

func Execute(ctx context.Context, loaded *LoadedWorkOrder, control *Control, spoolRoot, stateRoot string, now time.Time) (*Result, error) {
	return execute(ctx, loaded, control, spoolRoot, stateRoot, now, nil)
}

func execute(ctx context.Context, loaded *LoadedWorkOrder, control *Control, spoolRoot, stateRoot string, now time.Time, injected retriever) (*Result, error) {
	if loaded == nil || control == nil || now.Location() != time.UTC {
		return nil, errors.New("egress worker: loaded order, control, and UTC time are required")
	}
	if err := loaded.Order.Validate(); err != nil || digest(loaded.bytes) != loaded.Digest {
		return nil, errors.New("egress worker: invalid or substituted loaded work order")
	}
	if err := control.Validate(); err != nil {
		return nil, err
	}
	if !control.Enabled || control.EmergencyStop {
		return nil, errors.New("egress worker: emergency kill switch or global disable is active")
	}
	if contains(control.RevokedOrigins, loaded.Order.OriginID) || contains(control.RevokedOrders, loaded.Order.ID) {
		return nil, errors.New("egress worker: origin or work order is revoked")
	}
	notBefore, _ := canonicalTime(loaded.Order.NotBefore)
	expires, _ := canonicalTime(loaded.Order.ExpiresAt)
	if now.Before(notBefore) || !now.Before(expires) {
		return nil, errors.New("egress worker: work order is outside its validity interval")
	}
	lease, err := acquireLease(stateRoot)
	if err != nil {
		return nil, err
	}
	defer lease.Close()
	if err := checkCircuit(stateRoot, loaded.Order, now); err != nil {
		return nil, err
	}
	outputRoot, err := createSpool(spoolRoot, loaded.Order.ID, strings.TrimPrefix(loaded.Digest, "sha256:"))
	if err != nil {
		return nil, err
	}
	if err := atomicfile.Write(filepath.Join(outputRoot, "work-order.json"), loaded.bytes, MaxWorkOrder, 0o440); err != nil {
		return nil, err
	}
	started := time.Now()
	if injected == nil {
		policy := safefetch.DefaultPolicy()
		policy.ID = WorkerPolicyID
		policy.AllowedHosts = append([]string(nil), loaded.Order.AllowedHosts...)
		policy.MaxRedirects = loaded.Order.MaxRedirects
		policy.MaxBodyBytes = loaded.Order.MaxBodyBytes
		policy.RequestTimeout = time.Duration(loaded.Order.TimeoutMillis) * time.Millisecond
		policy.ConnectTimeout = time.Duration(loaded.Order.ConnectTimeoutMillis) * time.Millisecond
		policy.ResponseHeaderTimeout = time.Duration(loaded.Order.HeaderTimeoutMillis) * time.Millisecond
		policy.UserAgent = "TWIRXBot/0.2 (+https://twirx.org/bot; contact:security@twirx.org)"
		fetcher, err := safefetch.New(policy)
		if err != nil {
			return nil, err
		}
		injected = fetcher
	}
	return executeFetch(ctx, loaded, injected, outputRoot, stateRoot, now, started)
}

func executeFetch(ctx context.Context, loaded *LoadedWorkOrder, fetcher retriever, outputRoot, stateRoot string, now, started time.Time) (*Result, error) {
	fetched, err := fetcher.Fetch(ctx, loaded.Order.URL)
	if err != nil {
		if stateErr := recordFailure(stateRoot, loaded.Order, now); stateErr != nil {
			return nil, fmt.Errorf("egress worker: retrieval failed and circuit state could not be recorded: %w", stateErr)
		}
		return nil, fmt.Errorf("egress worker: retrieve admitted order: %w", err)
	}
	store := cas.New(filepath.Join(outputRoot, "cas"))
	paths, err := observation.WriteBundle(filepath.Join(outputRoot, "evidence"), store, fetched, WorkerPolicyID)
	if err != nil {
		if stateErr := recordFailure(stateRoot, loaded.Order, now); stateErr != nil {
			return nil, fmt.Errorf("egress worker: evidence publication failed and circuit state could not be recorded: %w", stateErr)
		}
		return nil, fmt.Errorf("egress worker: publish evidence: %w", err)
	}
	bodyPath, err := filepath.Rel(outputRoot, paths.BodyStoragePath)
	if err != nil || strings.HasPrefix(bodyPath, "..") || filepath.IsAbs(bodyPath) {
		return nil, errors.New("egress worker: CAS body escaped immutable spool")
	}
	bodyPath = filepath.ToSlash(bodyPath)
	envelope, envelopeBytes, err := observation.Load(paths.CBORPath)
	if err != nil {
		return nil, err
	}
	result := Result{Format: ResultFormat, WorkOrderID: loaded.Order.ID, WorkOrderDigest: loaded.Digest, OriginID: loaded.Order.OriginID, Purpose: loaded.Order.Purpose, RequestURL: envelope.RequestURL, FinalURL: envelope.FinalURL, RetrievedAt: envelope.RetrievedAt, HTTPStatus: int(envelope.Status), MediaType: envelope.MediaType, RedirectCount: len(fetched.Redirects), ObservationDigest: observation.EnvelopeDigest(envelopeBytes), BodyDigest: envelope.BodyDigest(), BodySize: envelope.BodySize, DurationMillis: time.Since(started).Milliseconds()}
	resultBytes, err := marshal(result)
	if err != nil {
		return nil, err
	}
	if err := atomicfile.Write(filepath.Join(outputRoot, "result.json"), resultBytes, MaxResult, 0o440); err != nil {
		return nil, err
	}
	observationJSON, err := readRegular(paths.JSONPath, observation.MaxJSONViewBytes)
	if err != nil {
		return nil, err
	}
	bodyReference := []byte(paths.BodyReference + "\n")
	manifest := Manifest{Format: ManifestFormat, WorkOrderID: loaded.Order.ID, WorkOrderDigest: loaded.Digest, Artifacts: []Artifact{
		{Path: bodyPath, Digest: result.BodyDigest},
		{Path: "evidence/body.ref", Digest: digest(bodyReference)},
		{Path: "evidence/observation.cbor", Digest: result.ObservationDigest},
		{Path: "evidence/observation.json", Digest: digest(observationJSON)},
		{Path: "result.json", Digest: digest(resultBytes)},
		{Path: "work-order.json", Digest: loaded.Digest},
	}}
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}
	manifestBytes, err := marshal(manifest)
	if err != nil {
		return nil, err
	}
	// Manifest-last publication defines completion. No parser or adapter runs
	// before the evidence spool is complete.
	if err := atomicfile.Write(filepath.Join(outputRoot, "manifest.json"), manifestBytes, MaxResult, 0o440); err != nil {
		return nil, err
	}
	_ = resetCircuit(stateRoot, loaded.Order)
	return &result, nil
}

// VerifySpool admits only a complete manifest-last evidence spool and rehashes
// every constituent artifact. The parser consumes nothing before this check.
func VerifySpool(root string, maxBodyBytes int64) (*Result, error) {
	if maxBodyBytes <= 0 || maxBodyBytes > MaxBody {
		return nil, errors.New("egress worker: invalid verification body limit")
	}
	manifestBytes, err := readRegular(filepath.Join(root, "manifest.json"), MaxResult)
	if err != nil {
		return nil, fmt.Errorf("egress worker: final manifest unavailable: %w", err)
	}
	var manifest Manifest
	if err := decode(manifestBytes, &manifest, MaxResult); err != nil {
		return nil, fmt.Errorf("egress worker: decode final manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}
	for _, artifact := range manifest.Artifacts {
		data, err := readSpoolArtifact(root, artifact.Path, maxBodyBytes)
		if err != nil {
			return nil, err
		}
		if digest(data) != artifact.Digest {
			return nil, fmt.Errorf("egress worker: artifact digest mismatch for %s", artifact.Path)
		}
	}
	workOrderBytes, err := readRegular(filepath.Join(root, "work-order.json"), MaxWorkOrder)
	if err != nil || digest(workOrderBytes) != manifest.WorkOrderDigest {
		return nil, errors.New("egress worker: work order does not match manifest authority")
	}
	var order WorkOrder
	if err := decode(workOrderBytes, &order, MaxWorkOrder); err != nil || order.ID != manifest.WorkOrderID || order.Validate() != nil {
		return nil, errors.New("egress worker: invalid spooled work order")
	}
	resultBytes, err := readRegular(filepath.Join(root, "result.json"), MaxResult)
	if err != nil {
		return nil, err
	}
	var result Result
	if err := decode(resultBytes, &result, MaxResult); err != nil || result.Format != ResultFormat || result.WorkOrderID != order.ID || result.WorkOrderDigest != manifest.WorkOrderDigest || result.OriginID != order.OriginID || result.Purpose != order.Purpose || result.RequestURL != order.URL || result.HTTPStatus < 100 || result.HTTPStatus > 599 || result.MediaType == "" || len(result.MediaType) > 255 || result.RedirectCount < 0 || result.RedirectCount > order.MaxRedirects || result.DurationMillis < 0 || !validDigest(result.ObservationDigest) || !validDigest(result.BodyDigest) || result.BodySize > uint64(maxBodyBytes) {
		return nil, errors.New("egress worker: invalid spooled result")
	}
	envelope, envelopeBytes, err := observation.Load(filepath.Join(root, "evidence", "observation.cbor"))
	if err != nil || observation.EnvelopeDigest(envelopeBytes) != result.ObservationDigest || envelope.BodyDigest() != result.BodyDigest || envelope.BodySize != result.BodySize || envelope.RequestURL != result.RequestURL || envelope.FinalURL != result.FinalURL || envelope.PolicyID != WorkerPolicyID {
		return nil, errors.New("egress worker: result and observation disagree")
	}
	bodyReference, err := readRegular(filepath.Join(root, "evidence", "body.ref"), observation.MaxBodyReferenceBytes)
	if err != nil || string(bodyReference) != result.BodyDigest+"\n" {
		return nil, errors.New("egress worker: body reference disagrees with result")
	}
	store := cas.New(filepath.Join(root, "cas"))
	if err := observation.VerifyBody(envelope, store, maxBodyBytes); err != nil {
		return nil, err
	}
	return &result, nil
}

type lease struct {
	file *os.File
}

func acquireLease(root string) (*lease, error) {
	if err := requireDirectory(root); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(root, "worker.lock"), os.O_CREATE|os.O_RDWR, 0o640)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, errors.New("egress worker: global concurrency lease is already held")
	}
	return &lease{file: file}, nil
}

func (l *lease) Close() {
	if l == nil || l.file == nil {
		return
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	_ = l.file.Close()
}

func validateManifest(manifest Manifest) error {
	if manifest.Format != ManifestFormat || !idPattern.MatchString(manifest.WorkOrderID) || !validDigest(manifest.WorkOrderDigest) || len(manifest.Artifacts) != 6 {
		return errors.New("egress worker: invalid final manifest metadata")
	}
	previous := ""
	required := map[string]bool{"evidence/body.ref": false, "evidence/observation.cbor": false, "evidence/observation.json": false, "result.json": false, "work-order.json": false}
	body := false
	for _, artifact := range manifest.Artifacts {
		if !safeRelativePath(artifact.Path) || !validDigest(artifact.Digest) || artifact.Path <= previous || artifact.Path == "manifest.json" {
			return errors.New("egress worker: manifest artifacts must be sorted, unique, safe, and content-addressed")
		}
		previous = artifact.Path
		if _, exists := required[artifact.Path]; exists {
			required[artifact.Path] = true
		} else if strings.HasPrefix(artifact.Path, "cas/sha256/") {
			body = true
		} else {
			return fmt.Errorf("egress worker: unexpected manifest artifact %s", artifact.Path)
		}
	}
	for _, present := range required {
		if !present {
			return errors.New("egress worker: manifest omits required artifact")
		}
	}
	if !body {
		return errors.New("egress worker: manifest omits representation body")
	}
	return nil
}

func readSpoolArtifact(root, relative string, maxBodyBytes int64) ([]byte, error) {
	maximum := int64(MaxResult)
	if strings.HasPrefix(relative, "cas/sha256/") {
		maximum = maxBodyBytes
	} else if relative == "evidence/observation.cbor" {
		maximum = observation.MaxEnvelopeBytes
	}
	return readRegular(filepath.Join(root, filepath.FromSlash(relative)), maximum)
}

func safeRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) || filepath.Clean(filepath.FromSlash(value)) != filepath.FromSlash(value) || strings.Contains(value, "\\") {
		return false
	}
	return value != ".." && !strings.HasPrefix(value, "../")
}

func checkCircuit(root string, order WorkOrder, now time.Time) error {
	state, err := loadCircuit(root, order.OriginID)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if state.OpenUntil != nil {
		until, err := canonicalTime(*state.OpenUntil)
		if err != nil || now.Before(until) {
			return errors.New("egress worker: circuit breaker is open")
		}
	}
	return nil
}

func recordFailure(root string, order WorkOrder, now time.Time) error {
	state, err := loadCircuit(root, order.OriginID)
	if errors.Is(err, os.ErrNotExist) {
		state = &Circuit{Format: CircuitFormat, OriginID: order.OriginID}
	} else if err != nil {
		return err
	}
	state.ConsecutiveFailures++
	if state.ConsecutiveFailures >= order.MaxConsecutiveFailures {
		until := now.Add(time.Duration(order.CircuitCooldownSeconds) * time.Second).Format(time.RFC3339Nano)
		state.OpenUntil = &until
	}
	return writeCircuit(root, *state)
}

func resetCircuit(root string, order WorkOrder) error {
	return writeCircuit(root, Circuit{Format: CircuitFormat, OriginID: order.OriginID})
}

func loadCircuit(root, originID string) (*Circuit, error) {
	data, err := readRegular(filepath.Join(root, originID+".json"), MaxControl)
	if err != nil {
		return nil, err
	}
	var state Circuit
	if err := decode(data, &state, MaxControl); err != nil || state.Format != CircuitFormat || state.OriginID != originID || state.ConsecutiveFailures < 0 || state.ConsecutiveFailures > 100000 {
		return nil, errors.New("egress worker: invalid circuit state")
	}
	return &state, nil
}

func writeCircuit(root string, state Circuit) error {
	if err := requireDirectory(root); err != nil {
		return err
	}
	data, err := marshal(state)
	if err != nil {
		return err
	}
	return atomicfile.Write(filepath.Join(root, state.OriginID+".json"), data, MaxControl, 0o640)
}

func createSpool(root, orderID, digestID string) (string, error) {
	if err := requireDirectory(root); err != nil {
		return "", err
	}
	orderRoot := filepath.Join(root, orderID)
	if err := os.Mkdir(orderRoot, 0o750); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	if err := requireDirectory(orderRoot); err != nil {
		return "", err
	}
	path := filepath.Join(orderRoot, digestID)
	if _, err := os.Lstat(path); err == nil {
		return "", errors.New("egress worker: immutable spool target already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Mkdir(path, 0o750); err != nil {
		return "", err
	}
	return path, nil
}

func requireDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("egress worker: required root is not a real directory")
	}
	return nil
}

func marshal(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decode(data []byte, value any, maximum int) error {
	return jsonbounded.Decode(data, value, jsonbounded.Policy{MaxBytes: maximum, MaxDepth: 16, MaxScalarBytes: 16 << 10, MaxContainerEntries: 512, MaxTokens: 10000}, true)
}

func readRegular(path string, maximum int64) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("egress worker: cannot open artifact")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximum {
		return nil, errors.New("artifact is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum || int64(len(data)) != info.Size() {
		return nil, errors.New("artifact changed size while being read")
	}
	return data, nil
}

func canonicalTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Location() != time.UTC || parsed.Format(time.RFC3339Nano) != value {
		return time.Time{}, errors.New("timestamp is not canonical UTC")
	}
	return parsed, nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func sortedUnique(values []string) bool {
	if !sort.StringsAreSorted(values) {
		return false
	}
	for index, value := range values {
		if value == "" || index > 0 && value == values[index-1] {
			return false
		}
	}
	return true
}

func ambiguousNumericHost(host string) bool {
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}
	labels := strings.Split(host, ".")
	allNumeric := true
	for _, label := range labels {
		if label == "" {
			return true
		}
		lower := strings.ToLower(label)
		if strings.HasPrefix(lower, "0x") && len(lower) > 2 {
			if _, err := hex.DecodeString(strings.TrimPrefix(lower, "0x")); err == nil {
				return true
			}
		}
		for _, char := range label {
			if char < '0' || char > '9' {
				allNumeric = false
				break
			}
		}
	}
	return allNumeric
}

func validHostname(host string) bool {
	if host == "" || len(host) > 253 || strings.HasSuffix(host, ".") {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if char != '-' && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
				return false
			}
		}
	}
	return true
}

func contains(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func validIDs(values []string) bool {
	for _, value := range values {
		if !idPattern.MatchString(value) {
			return false
		}
	}
	return true
}
