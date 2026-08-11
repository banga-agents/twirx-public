// Package archiveimport validates sealed Common Crawl work orders and turns
// bounded historical captures into immutable evidence. It has no public URL
// input, scheduler, live-origin access, or canonical-admission authority.
package archiveimport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	WorkOrderFormat  = "tw.common-crawl-work-order/0.2"
	MaxWorkOrder     = 64 << 10
	MaxPeriods       = 2
	MaxRoutes        = 6
	MaxCaptures      = 3
	MaxIndexRequests = 4
	MaxIndexResponse = 256 << 10
	MaxCompressed    = 2 << 20
	MaxDecompressed  = 8 << 20
	MaxRetainedBody  = 5 << 20
)

var (
	idPattern         = regexp.MustCompile(`^[a-z][a-z0-9-]{0,127}$`)
	collectionPattern = regexp.MustCompile(`^CC-MAIN-[0-9]{4}-[0-9]{2}$`)
)

type WorkOrder struct {
	Format                     string             `json:"format"`
	ID                         string             `json:"id"`
	OriginID                   string             `json:"origin_id"`
	CanonicalOrigin            string             `json:"canonical_origin"`
	PermittedRoutes            []string           `json:"permitted_routes"`
	CollectionIDs              []string           `json:"collection_ids"`
	SelectedCaptures           []CaptureSelection `json:"selected_captures"`
	CapturesPerCollection      uint64             `json:"captures_per_collection"`
	IndexRequestBudget         uint64             `json:"index_request_budget"`
	MaxIndexResponseBytes      uint64             `json:"max_index_response_bytes"`
	MaxCompressedRecordBytes   uint64             `json:"max_compressed_record_bytes"`
	MaxDecompressedRecordBytes uint64             `json:"max_decompressed_record_bytes"`
	MaxRetainedBodyBytes       uint64             `json:"max_retained_body_bytes"`
	PolicyReviewState          string             `json:"policy_review_state"`
	PolicyDecision             string             `json:"policy_decision"`
	PolicyEvidenceDigest       string             `json:"policy_evidence_digest"`
	DecisionDigest             string             `json:"decision_digest"`
	ApprovalReference          string             `json:"approval_reference"`
	ApprovedBy                 string             `json:"approved_by"`
	ApprovedAt                 string             `json:"approved_at"`
	EvidenceClass              string             `json:"evidence_class"`
	Freshness                  string             `json:"freshness"`
	CurrentPublisherStatement  bool               `json:"current_publisher_statement"`
	ObservedBy                 string             `json:"observed_by"`
}

// CaptureSelection is the complete human-approved Common Crawl record
// identity. Index responses can confirm this identity, but cannot expand it.
type CaptureSelection struct {
	CollectionID   string `json:"collection_id"`
	Route          string `json:"route"`
	Timestamp      string `json:"timestamp"`
	ProviderDigest string `json:"provider_digest"`
	Filename       string `json:"filename"`
	Offset         uint64 `json:"offset"`
	Length         uint64 `json:"length"`
}

type LoadedWorkOrder struct {
	Order  WorkOrder
	Digest string
	Bytes  []byte
}

func LoadWorkOrder(root, id string) (*LoadedWorkOrder, error) {
	if !idPattern.MatchString(id) {
		return nil, errors.New("archiveimport: invalid work-order ID")
	}
	data, err := readRegular(filepath.Join(root, id+".json"), MaxWorkOrder)
	if err != nil {
		return nil, fmt.Errorf("archiveimport: read work order: %w", err)
	}
	var order WorkOrder
	if err := decodeStrict(data, &order, MaxWorkOrder); err != nil {
		return nil, fmt.Errorf("archiveimport: decode work order: %w", err)
	}
	if order.ID != id {
		return nil, errors.New("archiveimport: work-order filename and identity disagree")
	}
	if err := order.Validate(); err != nil {
		return nil, err
	}
	return &LoadedWorkOrder{Order: order, Digest: digest(data), Bytes: append([]byte(nil), data...)}, nil
}

func (w WorkOrder) Validate() error {
	if w.Format != WorkOrderFormat || !idPattern.MatchString(w.ID) || !idPattern.MatchString(w.OriginID) {
		return errors.New("archiveimport: invalid work-order identity")
	}
	origin, err := parseCanonicalOrigin(w.CanonicalOrigin)
	if err != nil {
		return err
	}
	if len(w.PermittedRoutes) == 0 || len(w.PermittedRoutes) > MaxRoutes || !sortedUnique(w.PermittedRoutes) {
		return errors.New("archiveimport: permitted routes must be sorted, unique, and bounded")
	}
	for _, raw := range w.PermittedRoutes {
		route, err := url.Parse(raw)
		if err != nil || route.Scheme != "https" || route.User != nil || route.Opaque != "" || route.Fragment != "" || route.RawPath != "" || route.Host != origin.Host || route.String() != raw {
			return errors.New("archiveimport: permitted route is not an exact canonical HTTPS URL on the reviewed origin")
		}
	}
	if len(w.CollectionIDs) == 0 || len(w.CollectionIDs) > MaxPeriods || !sortedUnique(w.CollectionIDs) {
		return errors.New("archiveimport: collection IDs must be sorted, unique, and bounded")
	}
	for _, collection := range w.CollectionIDs {
		if !collectionPattern.MatchString(collection) {
			return errors.New("archiveimport: invalid Common Crawl collection ID")
		}
	}
	if w.CapturesPerCollection != 1 || w.IndexRequestBudget == 0 || w.IndexRequestBudget > MaxIndexRequests || uint64(len(w.CollectionIDs))*uint64(len(w.PermittedRoutes)) > w.IndexRequestBudget {
		return errors.New("archiveimport: capture or index-request budget is invalid")
	}
	expectedSelections := len(w.CollectionIDs) * len(w.PermittedRoutes)
	if len(w.SelectedCaptures) != expectedSelections || len(w.SelectedCaptures) > MaxPeriods*MaxRoutes*MaxCaptures {
		return errors.New("archiveimport: every collection and route requires one exact selected capture")
	}
	previousSelection := ""
	seenSelections := make(map[string]struct{}, len(w.SelectedCaptures))
	for _, selection := range w.SelectedCaptures {
		key := selection.CollectionID + "\x00" + selection.Route
		if key <= previousSelection || !w.permitsCollection(selection.CollectionID) || !w.permitsRoute(selection.Route) || selection.validate() != nil {
			return errors.New("archiveimport: selected captures must be exact, sorted, unique, and inside the reviewed scope")
		}
		if _, duplicate := seenSelections[key]; duplicate {
			return errors.New("archiveimport: duplicate selected capture")
		}
		seenSelections[key] = struct{}{}
		previousSelection = key
	}
	if w.MaxIndexResponseBytes == 0 || w.MaxIndexResponseBytes > MaxIndexResponse || w.MaxCompressedRecordBytes == 0 || w.MaxCompressedRecordBytes > MaxCompressed || w.MaxDecompressedRecordBytes == 0 || w.MaxDecompressedRecordBytes > MaxDecompressed || w.MaxRetainedBodyBytes == 0 || w.MaxRetainedBodyBytes > MaxRetainedBody || w.MaxRetainedBodyBytes > w.MaxDecompressedRecordBytes {
		return errors.New("archiveimport: byte budget is outside the Genesis bounds")
	}
	if w.PolicyReviewState != "completed" {
		return errors.New("archiveimport: policy review is not completed")
	}
	if w.PolicyDecision != "permit_live" && w.PolicyDecision != "permit_with_constraints" && w.PolicyDecision != "profile_only" {
		return errors.New("archiveimport: policy decision does not authorize bounded archive profiling")
	}
	if !validDigest(w.PolicyEvidenceDigest) || !validDigest(w.DecisionDigest) || !validApprovalReference(w.ApprovalReference) || !validPlainText(w.ApprovedBy, 256) {
		return errors.New("archiveimport: human policy authority is incomplete")
	}
	approvedAt, err := time.Parse("2006-01-02T15:04:05Z", w.ApprovedAt)
	if err != nil || approvedAt.UTC().Format("2006-01-02T15:04:05Z") != w.ApprovedAt {
		return errors.New("archiveimport: approved_at must be canonical UTC seconds")
	}
	if w.EvidenceClass != "archive_observation" || w.Freshness != "historical" || w.CurrentPublisherStatement || w.ObservedBy != "common_crawl" {
		return errors.New("archiveimport: archive evidence classification is invalid")
	}
	return nil
}

func parseCanonicalOrigin(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Opaque != "" || parsed.Fragment != "" || parsed.RawQuery != "" || parsed.RawPath != "" || parsed.Hostname() == "" || parsed.Port() != "" || parsed.Path != "" || parsed.String() != raw || strings.ToLower(parsed.Hostname()) != parsed.Hostname() {
		return nil, errors.New("archiveimport: canonical origin must be a lower-case credential-free HTTPS origin without a path or port")
	}
	return parsed, nil
}

func (w WorkOrder) permitsCollection(value string) bool { return contains(w.CollectionIDs, value) }

func (w WorkOrder) permitsRoute(value string) bool { return contains(w.PermittedRoutes, value) }

func (w WorkOrder) selectedCapture(collectionID, route string) (CaptureSelection, bool) {
	key := collectionID + "\x00" + route
	index := sort.Search(len(w.SelectedCaptures), func(index int) bool {
		candidate := w.SelectedCaptures[index]
		return candidate.CollectionID+"\x00"+candidate.Route >= key
	})
	if index >= len(w.SelectedCaptures) {
		return CaptureSelection{}, false
	}
	candidate := w.SelectedCaptures[index]
	return candidate, candidate.CollectionID == collectionID && candidate.Route == route
}

func (s CaptureSelection) validate() error {
	parsedTime, timeErr := time.Parse("20060102150405", s.Timestamp)
	if !collectionPattern.MatchString(s.CollectionID) || s.Route == "" || timeErr != nil || parsedTime.UTC().Format("20060102150405") != s.Timestamp || !providerDigestPattern.MatchString(s.ProviderDigest) || !validWARCPath(s.Filename, s.CollectionID) || s.Length == 0 || s.Length > MaxCompressed || s.Offset > ^uint64(0)-(s.Length-1) {
		return errors.New("archiveimport: invalid selected capture identity")
	}
	return nil
}

func decodeStrict(data []byte, target any, maximum int) error {
	return jsonbounded.Decode(data, target, jsonbounded.Policy{MaxBytes: maximum, MaxDepth: 16, MaxScalarBytes: 16 << 10, MaxContainerEntries: 512, MaxTokens: 10000}, true)
}

func marshal(value any, maximum int) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) == 0 || len(data) > maximum {
		return nil, errors.New("archiveimport: encoded artifact exceeds bound")
	}
	return data, nil
}

func readRegular(path string, maximum int64) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("archiveimport: cannot open artifact")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximum {
		return nil, errors.New("archiveimport: artifact is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(data)) != info.Size() || int64(len(data)) > maximum {
		return nil, errors.New("archiveimport: artifact changed or exceeded its bound while read")
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

func sortedUnique(values []string) bool {
	if !sort.StringsAreSorted(values) {
		return false
	}
	for index, value := range values {
		if value == "" || strings.ContainsRune(value, '\x00') || index > 0 && value == values[index-1] {
			return false
		}
	}
	return true
}

func contains(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func validPlainText(value string, maximum int) bool {
	if value == "" || len(value) > maximum || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func validApprovalReference(value string) bool {
	return validPlainText(value, 4096) && !filepath.IsAbs(value) && !strings.Contains(value, "\\") && value != "." && value != ".." && filepath.Clean(value) == value && !strings.HasPrefix(value, "../") && !strings.Contains(value, "/../")
}
