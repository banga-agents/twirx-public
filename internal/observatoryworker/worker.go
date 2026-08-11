// Package observatoryworker implements the process-separated local-fixture
// retrieval proof for E3. It is not a public-origin crawler or a production
// egress sandbox.
package observatoryworker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/cas"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/observation"
	"github.com/typed-web-commons/typed-web/internal/robotstxt"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

const (
	JobFormat       = "tw.observatory-job/0.1"
	ResultFormat    = "tw.observatory-result/0.1"
	LocalMode       = "local_fixture"
	ArtifactRobots  = "robots"
	NetworkScope    = "literal_127.0.0.1_only"
	WorkerPolicyID  = "tw.observatory.local-fixture-v0"
	ProductToken    = "TWIRXBot"
	maxJobBytes     = 16 << 10
	maxResultBytes  = 64 << 10
	workerUserAgent = "TWIRXBot/0.1 (+https://twirx.org/bot)"
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,127}$`)

// Job is the complete authority presented to the local-fixture worker. E3's
// proof accepts no user-selected public destination.
type Job struct {
	Format       string `json:"format"`
	Mode         string `json:"mode"`
	OriginID     string `json:"origin_id"`
	ArtifactKind string `json:"artifact_kind"`
	URL          string `json:"url"`
	ProductToken string `json:"product_token"`
	TargetPath   string `json:"target_path"`
	MaxBodyBytes int64  `json:"max_body_bytes"`
}

// LoadedJob binds a validated job to the digest of its exact input bytes.
type LoadedJob struct {
	Job    Job
	Digest string
	bytes  []byte
}

// Result binds the successful robots evaluation to immutable observation and
// body evidence. The result is published only after those artifacts exist.
type Result struct {
	Format            string `json:"format"`
	Mode              string `json:"mode"`
	NetworkScope      string `json:"network_scope"`
	WorkerPolicyID    string `json:"worker_policy_id"`
	JobDigest         string `json:"job_digest"`
	OriginID          string `json:"origin_id"`
	ArtifactKind      string `json:"artifact_kind"`
	RequestURL        string `json:"request_url"`
	FinalURL          string `json:"final_url"`
	RetrievedAt       string `json:"retrieved_at"`
	HTTPStatus        int    `json:"http_status"`
	MediaType         string `json:"media_type"`
	ObservationDigest string `json:"observation_digest"`
	BodyDigest        string `json:"body_digest"`
	BodySize          uint64 `json:"body_size"`
	RobotsState       string `json:"robots_state"`
	ProductToken      string `json:"product_token"`
	TargetPath        string `json:"target_path"`
	Allowed           bool   `json:"allowed"`
	Matched           bool   `json:"matched"`
	MatchedPattern    string `json:"matched_pattern"`
	Specificity       int    `json:"specificity"`
	ParseErrors       int    `json:"parse_errors"`
}

// Verification is emitted by offline replay. NetworkAccess is a fixed
// declaration of verifier capability, not an observation about the host.
type Verification struct {
	Status            string `json:"status"`
	NetworkAccess     string `json:"network_access"`
	ObservationDigest string `json:"observation_digest"`
	BodyDigest        string `json:"body_digest"`
	RobotsState       string `json:"robots_state"`
	Allowed           bool   `json:"allowed"`
}

// LoadJob rejects symlinks, non-regular inputs, malformed JSON, duplicate
// keys, trailing data, unknown fields, and jobs outside the fixture profile.
func LoadJob(path string) (*LoadedJob, error) {
	data, err := readRegular(path, maxJobBytes)
	if err != nil {
		return nil, fmt.Errorf("observatory worker: read job: %w", err)
	}
	return decodeJob(data)
}

func decodeJob(data []byte) (*LoadedJob, error) {
	var job Job
	if err := jsonbounded.Decode(data, &job, jsonPolicy(maxJobBytes), true); err != nil {
		return nil, fmt.Errorf("observatory worker: decode job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return nil, err
	}
	return &LoadedJob{Job: job, Digest: cas.Digest(data), bytes: append([]byte(nil), data...)}, nil
}

// Validate constrains E3's executable proof to one literal IPv4 loopback
// destination and the robots artifact class.
func (j Job) Validate() error {
	if j.Format != JobFormat || j.Mode != LocalMode || j.ArtifactKind != ArtifactRobots {
		return errors.New("observatory worker: unsupported job format, mode, or artifact kind")
	}
	if !identifierPattern.MatchString(j.OriginID) {
		return errors.New("observatory worker: invalid origin_id")
	}
	if j.ProductToken != ProductToken {
		return errors.New("observatory worker: product token is not TWIRXBot")
	}
	if j.MaxBodyBytes <= 0 || j.MaxBodyBytes > int64(robotstxt.MaxBytes) {
		return fmt.Errorf("observatory worker: max_body_bytes must be between 1 and %d", robotstxt.MaxBytes)
	}
	if j.TargetPath == "" || len(j.TargetPath) > robotstxt.MaxTargetBytes || !strings.HasPrefix(j.TargetPath, "/") {
		return errors.New("observatory worker: target_path is empty, too long, or not origin-relative")
	}
	parsed, err := url.Parse(j.URL)
	if err != nil {
		return fmt.Errorf("observatory worker: parse URL: %w", err)
	}
	if parsed.Scheme != "http" || parsed.User != nil || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.Path != "/robots.txt" || parsed.EscapedPath() != "/robots.txt" {
		return errors.New("observatory worker: URL must be plain HTTP /robots.txt without credentials, query, or fragment")
	}
	address, err := netip.ParseAddr(parsed.Hostname())
	if err != nil || address.Unmap() != netip.MustParseAddr("127.0.0.1") {
		return errors.New("observatory worker: URL host must be the literal address 127.0.0.1")
	}
	port, err := strconv.ParseUint(parsed.Port(), 10, 16)
	if err != nil || port == 0 {
		return errors.New("observatory worker: URL must contain an explicit valid port")
	}
	return nil
}

// Execute retrieves one validated fixture artifact. The observation and CAS
// body are committed before the untrusted robots bytes are parsed.
func Execute(ctx context.Context, loaded *LoadedJob, outputRoot string) (*Result, error) {
	if loaded == nil {
		return nil, errors.New("observatory worker: loaded job is required")
	}
	if err := loaded.Job.Validate(); err != nil {
		return nil, err
	}
	if _, err := cas.ParseDigest(loaded.Digest); err != nil {
		return nil, fmt.Errorf("observatory worker: invalid job digest: %w", err)
	}
	if len(loaded.bytes) == 0 || cas.Digest(loaded.bytes) != loaded.Digest {
		return nil, errors.New("observatory worker: loaded job bytes do not match job digest")
	}
	if err := createEmptyOutputRoot(outputRoot); err != nil {
		return nil, err
	}
	if err := atomicfile.Write(filepath.Join(outputRoot, "job.json"), loaded.bytes, maxJobBytes, 0o640); err != nil {
		return nil, fmt.Errorf("observatory worker: publish job authority: %w", err)
	}

	policy := safefetch.DefaultPolicy()
	policy.ID = WorkerPolicyID
	policy.AllowLoopback = true
	policy.AllowNonStandardPorts = true
	policy.MaxRedirects = 0
	policy.MaxBodyBytes = loaded.Job.MaxBodyBytes
	policy.UserAgent = workerUserAgent
	policy.AllowedHosts = []string{"127.0.0.1"}
	fetcher, err := safefetch.New(policy)
	if err != nil {
		return nil, fmt.Errorf("observatory worker: construct fetcher: %w", err)
	}
	fetched, err := fetcher.Fetch(ctx, loaded.Job.URL)
	if err != nil {
		return nil, fmt.Errorf("observatory worker: retrieve fixture: %w", err)
	}

	store := cas.New(filepath.Join(outputRoot, "cas"))
	paths, err := observation.WriteBundle(filepath.Join(outputRoot, "evidence"), store, fetched, WorkerPolicyID)
	if err != nil {
		return nil, fmt.Errorf("observatory worker: publish evidence: %w", err)
	}
	state := robotstxt.ClassifyFetch(fetched.Status, len(fetched.Redirects), false)
	if state != robotstxt.FetchSuccessful {
		return nil, fmt.Errorf("observatory worker: robots retrieval state %q is not successful; evidence retained at %s", state, paths.Directory)
	}

	// Parsing begins only after WriteBundle has durably published both the CAS
	// body and observation envelope.
	document, err := robotstxt.Parse(fetched.Body)
	if err != nil {
		return nil, fmt.Errorf("observatory worker: parse robots after evidence publication: %w", err)
	}
	decision, err := document.Evaluate(loaded.Job.ProductToken, loaded.Job.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("observatory worker: evaluate robots: %w", err)
	}
	envelope, cborBytes, err := observation.Load(paths.CBORPath)
	if err != nil {
		return nil, fmt.Errorf("observatory worker: reload published observation: %w", err)
	}
	result := &Result{
		Format: ResultFormat, Mode: LocalMode, NetworkScope: NetworkScope,
		WorkerPolicyID: WorkerPolicyID, JobDigest: loaded.Digest, OriginID: loaded.Job.OriginID,
		ArtifactKind: ArtifactRobots, RequestURL: envelope.RequestURL, FinalURL: envelope.FinalURL,
		RetrievedAt: envelope.RetrievedAt, HTTPStatus: int(envelope.Status), MediaType: envelope.MediaType,
		ObservationDigest: observation.EnvelopeDigest(cborBytes), BodyDigest: envelope.BodyDigest(), BodySize: envelope.BodySize,
		RobotsState: string(state), ProductToken: loaded.Job.ProductToken, TargetPath: loaded.Job.TargetPath,
		Allowed: decision.Allowed, Matched: decision.Matched, MatchedPattern: decision.Pattern,
		Specificity: decision.Specificity, ParseErrors: document.ParseErrors,
	}
	if err := result.Validate(); err != nil {
		return nil, err
	}
	data, err := marshalJSON(result)
	if err != nil {
		return nil, err
	}
	if err := atomicfile.Write(filepath.Join(outputRoot, "result.json"), data, maxResultBytes, 0o640); err != nil {
		return nil, fmt.Errorf("observatory worker: publish result: %w", err)
	}
	return result, nil
}

// Verify replays the robots evaluation using only committed evidence. It
// creates no network client and performs no retrieval.
func Verify(outputRoot string) (*Verification, error) {
	loaded, err := LoadJob(filepath.Join(outputRoot, "job.json"))
	if err != nil {
		return nil, fmt.Errorf("observatory worker: load preserved job: %w", err)
	}
	resultBytes, err := readRegular(filepath.Join(outputRoot, "result.json"), maxResultBytes)
	if err != nil {
		return nil, fmt.Errorf("observatory worker: read result: %w", err)
	}
	var result Result
	if err := jsonbounded.Decode(resultBytes, &result, jsonPolicy(maxResultBytes), true); err != nil {
		return nil, fmt.Errorf("observatory worker: decode result: %w", err)
	}
	if err := result.Validate(); err != nil {
		return nil, err
	}
	if result.JobDigest != loaded.Digest || result.OriginID != loaded.Job.OriginID || result.ArtifactKind != loaded.Job.ArtifactKind || result.RequestURL != loaded.Job.URL || result.ProductToken != loaded.Job.ProductToken || result.TargetPath != loaded.Job.TargetPath || result.BodySize > uint64(loaded.Job.MaxBodyBytes) {
		return nil, errors.New("observatory worker: result disagrees with preserved job authority")
	}

	observationPath := filepath.Join(outputRoot, "evidence", "observation.cbor")
	cborBytes, err := readRegular(observationPath, observation.MaxEnvelopeBytes)
	if err != nil {
		return nil, fmt.Errorf("observatory worker: read observation: %w", err)
	}
	envelope, err := observation.UnmarshalCBOR(cborBytes)
	if err != nil {
		return nil, err
	}
	if observation.EnvelopeDigest(cborBytes) != result.ObservationDigest || envelope.BodyDigest() != result.BodyDigest || envelope.RequestURL != result.RequestURL || envelope.FinalURL != result.FinalURL || envelope.RetrievedAt != result.RetrievedAt || int(envelope.Status) != result.HTTPStatus || envelope.MediaType != result.MediaType || envelope.BodySize != result.BodySize || envelope.PolicyID != result.WorkerPolicyID {
		return nil, errors.New("observatory worker: result disagrees with observation evidence")
	}
	store := cas.New(filepath.Join(outputRoot, "cas"))
	bodyPath, err := store.Path(result.BodyDigest)
	if err != nil {
		return nil, err
	}
	if _, err := readRegular(bodyPath, int64(robotstxt.MaxBytes)); err != nil {
		return nil, fmt.Errorf("observatory worker: inspect CAS body: %w", err)
	}
	if err := observation.VerifyBody(envelope, store, int64(robotstxt.MaxBytes)); err != nil {
		return nil, err
	}
	body, err := store.Read(result.BodyDigest, int64(robotstxt.MaxBytes))
	if err != nil {
		return nil, err
	}
	document, err := robotstxt.Parse(body)
	if err != nil {
		return nil, err
	}
	decision, err := document.Evaluate(result.ProductToken, result.TargetPath)
	if err != nil {
		return nil, err
	}
	if result.RobotsState != string(robotstxt.FetchSuccessful) || decision.Allowed != result.Allowed || decision.Matched != result.Matched || decision.Pattern != result.MatchedPattern || decision.Specificity != result.Specificity || document.ParseErrors != result.ParseErrors {
		return nil, errors.New("observatory worker: recorded robots decision disagrees with offline replay")
	}
	return &Verification{Status: "verified", NetworkAccess: "disabled", ObservationDigest: result.ObservationDigest, BodyDigest: result.BodyDigest, RobotsState: result.RobotsState, Allowed: result.Allowed}, nil
}

func (r Result) Validate() error {
	if r.Format != ResultFormat || r.Mode != LocalMode || r.NetworkScope != NetworkScope || r.WorkerPolicyID != WorkerPolicyID || r.ArtifactKind != ArtifactRobots || r.ProductToken != ProductToken || !identifierPattern.MatchString(r.OriginID) {
		return errors.New("observatory worker: invalid result identity")
	}
	for _, digest := range []string{r.JobDigest, r.ObservationDigest, r.BodyDigest} {
		if _, err := cas.ParseDigest(digest); err != nil {
			return fmt.Errorf("observatory worker: invalid result digest: %w", err)
		}
	}
	if r.RequestURL != r.FinalURL {
		return errors.New("observatory worker: local fixture result cannot contain a redirect")
	}
	job := Job{Format: JobFormat, Mode: r.Mode, OriginID: r.OriginID, ArtifactKind: r.ArtifactKind, URL: r.RequestURL, ProductToken: r.ProductToken, TargetPath: r.TargetPath, MaxBodyBytes: int64(robotstxt.MaxBytes)}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("observatory worker: result URL or target: %w", err)
	}
	parsedTime, err := time.Parse(time.RFC3339Nano, r.RetrievedAt)
	if err != nil || parsedTime.Location() != time.UTC || parsedTime.Format(time.RFC3339Nano) != r.RetrievedAt {
		return errors.New("observatory worker: retrieved_at is not canonical UTC")
	}
	if r.HTTPStatus < 200 || r.HTTPStatus > 299 || r.RobotsState != string(robotstxt.FetchSuccessful) || r.MediaType == "" || r.BodySize > uint64(robotstxt.MaxBytes) || r.Specificity < 0 || r.ParseErrors < 0 {
		return errors.New("observatory worker: result values are outside the successful fixture profile")
	}
	if !r.Matched && (r.MatchedPattern != "" || r.Specificity != 0) {
		return errors.New("observatory worker: unmatched decision carries match metadata")
	}
	return nil
}

func MarshalJSON(value any) ([]byte, error) { return marshalJSON(value) }

func marshalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("observatory worker: encode JSON: %w", err)
	}
	return append(data, '\n'), nil
}

func jsonPolicy(maximum int) jsonbounded.Policy {
	return jsonbounded.Policy{MaxBytes: maximum, MaxDepth: 12, MaxScalarBytes: 16 << 10, MaxContainerEntries: 128, MaxTokens: 1024}
}

func readRegular(path string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("artifact is not a regular file")
	}
	if info.Size() < 0 || info.Size() > maximum {
		return nil, fmt.Errorf("artifact size %d exceeds %d bytes", info.Size(), maximum)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum {
		return nil, fmt.Errorf("artifact exceeds %d bytes", maximum)
	}
	return data, nil
}

func createEmptyOutputRoot(path string) error {
	if path == "" {
		return errors.New("observatory worker: output root is required")
	}
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(path, 0o750); err != nil {
			return fmt.Errorf("observatory worker: create output root: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("observatory worker: inspect output root: %w", err)
	case !info.IsDir():
		return errors.New("observatory worker: output root must be a directory, not a file or symlink")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("observatory worker: read output root: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("observatory worker: output root must be empty")
	}
	return nil
}
