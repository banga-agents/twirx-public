// Package opportunitypilot implements the sealed, manual-once Grants.gov
// bulk-extract boundary for E4. It accepts no caller-supplied URL, performs no
// XML parsing before immutable range evidence is complete, and grants no
// scheduler or semantic-admission authority.
package opportunitypilot

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	WorkOrderFormat   = "tw.e4-opportunity-bulk-work-order/0.1"
	ControlFormat     = "tw.e4-opportunity-control/0.1"
	DecisionFormat    = "tw.e4-opportunity-policy-decision/0.1"
	OriginID          = "grants-gov-api"
	SourceURL         = "https://prod-grants-gov-chatbot.s3.amazonaws.com/extracts/GrantsDBExtract20260811v2.zip"
	SourceFilename    = "GrantsDBExtract20260811v2.zip"
	SourceHost        = "prod-grants-gov-chatbot.s3.amazonaws.com"
	ProposalPath      = "reports/e4-opportunity-policy-decision-proposal.md"
	DecisionPath      = "atlas/e4-decisions/grants-gov-20260811/decision.json"
	FounderReviewPath = "atlas/e4-decisions/grants-gov-20260811/founder-review.txt"

	MaxWorkOrder       = 64 << 10
	MaxControl         = 64 << 10
	RangeBytes         = 1 << 20
	MaximumRequests    = 96
	MaximumArchive     = 96 << 20
	MaximumExpandedXML = 512 << 20
	MaximumRecords     = 250000
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,127}$`)

type WorkOrder struct {
	Format                   string `json:"format"`
	ID                       string `json:"id"`
	OriginID                 string `json:"origin_id"`
	SourceURL                string `json:"source_url"`
	SourceFilename           string `json:"source_filename"`
	ExecutionMode            string `json:"execution_mode"`
	PolicyReviewState        string `json:"policy_review_state"`
	PolicyDecision           string `json:"policy_decision"`
	PolicyEvidenceDigest     string `json:"policy_evidence_digest"`
	DecisionDigest           string `json:"decision_digest"`
	ApprovalReference        string `json:"approval_reference"`
	ApprovedBy               string `json:"approved_by"`
	ApprovedAt               string `json:"approved_at"`
	NotBefore                string `json:"not_before"`
	ExpiresAt                string `json:"expires_at"`
	RangeBytes               uint64 `json:"range_bytes"`
	MaximumRequests          uint64 `json:"maximum_requests"`
	MaximumArchiveBytes      uint64 `json:"maximum_archive_bytes"`
	MaximumExpandedXMLBytes  uint64 `json:"maximum_expanded_xml_bytes"`
	MaximumRecords           uint64 `json:"maximum_records"`
	RequestIntervalMillis    uint64 `json:"request_interval_ms"`
	RequestTimeoutMillis     uint64 `json:"request_timeout_ms"`
	Redirects                uint64 `json:"redirects"`
	SchedulerEnabled         bool   `json:"scheduler_enabled"`
	RawEvidencePublic        bool   `json:"raw_evidence_public"`
	ContactProjectionEnabled bool   `json:"contact_projection_enabled"`
}

type LoadedWorkOrder struct {
	Order             WorkOrder
	Bytes             []byte
	Digest            string
	AuthorityVerified bool
}

type Decision struct {
	Format            string `json:"format"`
	OriginID          string `json:"origin_id"`
	ReviewState       string `json:"review_state"`
	Decision          string `json:"decision"`
	ReviewedAt        string `json:"reviewed_at"`
	Reviewer          string `json:"reviewer"`
	ApprovalReference string `json:"approval_reference"`
	ProposalDigest    string `json:"proposal_digest"`
	FounderReviewRef  string `json:"founder_review_ref"`
	SourceURL         string `json:"source_url"`
	Scope             string `json:"scope"`
}

type Control struct {
	Format         string   `json:"format"`
	Enabled        bool     `json:"enabled"`
	EmergencyStop  bool     `json:"emergency_stop"`
	RevokedOrders  []string `json:"revoked_orders"`
	RevokedOrigins []string `json:"revoked_origins"`
}

type ByteRange struct {
	Index uint64
	Start uint64
	End   uint64
}

func (r ByteRange) Header() string { return fmt.Sprintf("bytes=%d-%d", r.Start, r.End) }

func LoadWorkOrder(path string) (*LoadedWorkOrder, error) {
	data, err := readRegular(path, MaxWorkOrder)
	if err != nil {
		return nil, fmt.Errorf("opportunity pilot: read work order: %w", err)
	}
	var order WorkOrder
	if err := decode(data, &order, MaxWorkOrder); err != nil {
		return nil, fmt.Errorf("opportunity pilot: decode work order: %w", err)
	}
	if err := order.Validate(); err != nil {
		return nil, err
	}
	return &LoadedWorkOrder{Order: order, Bytes: append([]byte(nil), data...), Digest: digest(data)}, nil
}

// VerifyAuthority binds the work order to the exact committed proposal,
// steward decision and verbatim approval. It is deliberately separate from
// structural work-order parsing so offline tools can report which authority
// check failed without treating a syntactically valid order as authorized.
func VerifyAuthority(root string, loaded *LoadedWorkOrder) error {
	if loaded == nil || loaded.Order.Validate() != nil || digest(loaded.Bytes) != loaded.Digest {
		return errors.New("opportunity pilot: exact work order is required")
	}
	proposal, err := readRootArtifact(root, ProposalPath, 1<<20)
	if err != nil || digest(proposal) != loaded.Order.PolicyEvidenceDigest {
		return errors.New("opportunity pilot: approved proposal does not reconcile")
	}
	decisionBytes, err := readRootArtifact(root, DecisionPath, MaxWorkOrder)
	var decision Decision
	if err != nil || digest(decisionBytes) != loaded.Order.DecisionDigest || decode(decisionBytes, &decision, MaxWorkOrder) != nil {
		return errors.New("opportunity pilot: human decision does not reconcile")
	}
	if decision.Format != DecisionFormat || decision.OriginID != OriginID || decision.ReviewState != "completed" || decision.Decision != "permit_with_constraints" || decision.ReviewedAt != loaded.Order.ApprovedAt || decision.Reviewer != loaded.Order.ApprovedBy || decision.ApprovalReference != ProposalPath || decision.ProposalDigest != loaded.Order.PolicyEvidenceDigest || decision.FounderReviewRef != FounderReviewPath || decision.SourceURL != SourceURL || decision.Scope == "" || len(decision.Scope) > 4096 {
		return errors.New("opportunity pilot: human decision scope or identity is invalid")
	}
	if _, err := canonicalTime(decision.ReviewedAt); err != nil {
		return errors.New("opportunity pilot: decision review time is invalid")
	}
	review, err := readRootArtifact(root, FounderReviewPath, 4096)
	const exactApproval = "I approve the exact E4 Opportunity policy proposal in reports/e4-opportunity-policy-decision-proposal.md. Use the current UTC time as reviewed_at. No route, file, field, retention class or execution beyond that proposal is authorized\n"
	if err != nil || string(review) != exactApproval {
		return errors.New("opportunity pilot: verbatim Genesis steward approval is unavailable")
	}
	loaded.AuthorityVerified = true
	return nil
}

func readRootArtifact(root, relative string, maximum int64) ([]byte, error) {
	if root == "" || !safeReference(relative) {
		return nil, errors.New("opportunity pilot: repository root and safe artifact path are required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(absolute)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("opportunity pilot: repository root must be a real directory")
	}
	return readRegular(filepath.Join(absolute, filepath.FromSlash(relative)), maximum)
}

func LoadControl(path string) (*Control, error) {
	data, err := readRegular(path, MaxControl)
	if err != nil {
		return nil, fmt.Errorf("opportunity pilot: read control: %w", err)
	}
	var control Control
	if err := decode(data, &control, MaxControl); err != nil {
		return nil, fmt.Errorf("opportunity pilot: decode control: %w", err)
	}
	if err := control.Validate(); err != nil {
		return nil, err
	}
	return &control, nil
}

func (w WorkOrder) Validate() error {
	if w.Format != WorkOrderFormat || !idPattern.MatchString(w.ID) || w.OriginID != OriginID || w.SourceURL != SourceURL || w.SourceFilename != SourceFilename || w.ExecutionMode != "manual_once" {
		return errors.New("opportunity pilot: work-order identity or exact source is invalid")
	}
	parsed, err := url.Parse(w.SourceURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Host != SourceHost || parsed.Path != "/extracts/"+SourceFilename || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.String() != w.SourceURL {
		return errors.New("opportunity pilot: source must be the exact credential-free HTTPS object")
	}
	if w.PolicyReviewState != "completed" || w.PolicyDecision != "permit_with_constraints" || !validDigest(w.PolicyEvidenceDigest) || !validDigest(w.DecisionDigest) || !safeReference(w.ApprovalReference) || w.ApprovedBy == "" || len(w.ApprovedBy) > 256 {
		return errors.New("opportunity pilot: completed human policy authority is required")
	}
	approved, err := canonicalTime(w.ApprovedAt)
	if err != nil || approved.IsZero() {
		return errors.New("opportunity pilot: approved_at must be canonical UTC seconds")
	}
	notBefore, err := canonicalTime(w.NotBefore)
	if err != nil {
		return errors.New("opportunity pilot: invalid not_before")
	}
	expires, err := canonicalTime(w.ExpiresAt)
	if err != nil || !expires.After(notBefore) || expires.Sub(notBefore) > 24*time.Hour || approved.After(expires) {
		return errors.New("opportunity pilot: validity interval is invalid")
	}
	if w.RangeBytes != RangeBytes || w.MaximumRequests != MaximumRequests || w.MaximumArchiveBytes != MaximumArchive || w.MaximumExpandedXMLBytes != MaximumExpandedXML || w.MaximumRecords != MaximumRecords || w.RequestIntervalMillis < 2000 || w.RequestIntervalMillis > 60000 || w.RequestTimeoutMillis != 30000 || w.Redirects != 0 {
		return errors.New("opportunity pilot: source budgets differ from the reviewed proposal")
	}
	if w.SchedulerEnabled || w.RawEvidencePublic || w.ContactProjectionEnabled {
		return errors.New("opportunity pilot: scheduler, public raw evidence, and contact projection are forbidden")
	}
	return nil
}

func (c Control) Validate() error {
	if c.Format != ControlFormat || !sortedUnique(c.RevokedOrders) || !sortedUnique(c.RevokedOrigins) || len(c.RevokedOrders) > 128 || len(c.RevokedOrigins) > 128 {
		return errors.New("opportunity pilot: invalid control artifact")
	}
	for _, values := range [][]string{c.RevokedOrders, c.RevokedOrigins} {
		for _, value := range values {
			if !idPattern.MatchString(value) {
				return errors.New("opportunity pilot: invalid revoked identity")
			}
		}
	}
	return nil
}

func (c Control) Permits(order WorkOrder) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if !c.Enabled || c.EmergencyStop || contains(c.RevokedOrders, order.ID) || contains(c.RevokedOrigins, order.OriginID) {
		return errors.New("opportunity pilot: disabled, emergency-stopped, or revoked")
	}
	return nil
}

func BuildRanges(total uint64) ([]ByteRange, error) {
	if total == 0 || total > MaximumArchive {
		return nil, errors.New("opportunity pilot: archive total is outside the reviewed bound")
	}
	count := (total + RangeBytes - 1) / RangeBytes
	if count == 0 || count > MaximumRequests {
		return nil, errors.New("opportunity pilot: range count exceeds the reviewed request budget")
	}
	ranges := make([]ByteRange, 0, count)
	for index, start := uint64(0), uint64(0); start < total; index, start = index+1, start+RangeBytes {
		end := start + RangeBytes - 1
		if end >= total {
			end = total - 1
		}
		ranges = append(ranges, ByteRange{Index: index, Start: start, End: end})
	}
	return ranges, nil
}

func canonicalTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || parsed.Location() != time.UTC || parsed.Format(time.RFC3339) != value {
		return time.Time{}, errors.New("timestamp is not canonical UTC seconds")
	}
	return parsed, nil
}

func decode(data []byte, target any, maximum int) error {
	return jsonbounded.Decode(data, target, jsonbounded.Policy{MaxBytes: maximum, MaxDepth: 12, MaxScalarBytes: 16 << 10, MaxContainerEntries: 512, MaxTokens: 8192}, true)
}

func readRegular(path string, maximum int64) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("opportunity pilot: cannot open artifact")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximum {
		return nil, errors.New("opportunity pilot: artifact is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(data)) != info.Size() || int64(len(data)) > maximum {
		return nil, errors.New("opportunity pilot: artifact changed or exceeded its bound")
	}
	return data, nil
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

func safeReference(value string) bool {
	return value != "" && len(value) <= 4096 && !filepath.IsAbs(value) && filepath.Clean(value) == value && !strings.HasPrefix(value, "../") && !strings.Contains(value, "/../") && !strings.ContainsAny(value, "\\\x00\r\n")
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

func contains(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}
