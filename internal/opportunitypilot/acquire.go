package opportunitypilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

const (
	AcquisitionFormat    = "tw.e4-opportunity-bulk-acquisition/0.1"
	RangeFormat          = "tw.e4-opportunity-range-evidence/0.1"
	ExecutionClaimFormat = "tw.e4-opportunity-execution-claim/0.1"
	MaxManifest          = 4 << 20
)

type ExecutionClaim struct {
	Format          string `json:"format"`
	WorkOrderID     string `json:"work_order_id"`
	WorkOrderDigest string `json:"work_order_digest"`
	OriginID        string `json:"origin_id"`
	ClaimedAt       string `json:"claimed_at"`
}

type RangeEvidence struct {
	Format         string `json:"format"`
	Index          uint64 `json:"index"`
	SourceURL      string `json:"source_url"`
	RequestedRange string `json:"requested_range"`
	ContentRange   string `json:"content_range"`
	HTTPStatus     int    `json:"http_status"`
	MediaType      string `json:"media_type"`
	RetrievedAt    string `json:"retrieved_at"`
	BodyDigest     string `json:"body_digest"`
	BodySize       uint64 `json:"body_size"`
	TotalSize      uint64 `json:"total_size"`
}

type AcquisitionArtifact struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   uint64 `json:"size"`
}

type AcquisitionManifest struct {
	Format               string                `json:"format"`
	WorkOrderID          string                `json:"work_order_id"`
	WorkOrderDigest      string                `json:"work_order_digest"`
	OriginID             string                `json:"origin_id"`
	SourceURL            string                `json:"source_url"`
	SourceFilename       string                `json:"source_filename"`
	PolicyDecisionDigest string                `json:"policy_decision_digest"`
	ExecutionClaimDigest string                `json:"execution_claim_digest"`
	StartedAt            string                `json:"started_at"`
	CompletedAt          string                `json:"completed_at"`
	NetworkRequests      uint64                `json:"network_requests"`
	TransferredBytes     uint64                `json:"transferred_bytes"`
	ArchiveDigest        string                `json:"archive_digest"`
	ArchiveSize          uint64                `json:"archive_size"`
	SchedulerEnabled     bool                  `json:"scheduler_enabled"`
	RawEvidencePublic    bool                  `json:"raw_evidence_public"`
	Artifacts            []AcquisitionArtifact `json:"artifacts"`
}

type rangeRetriever interface {
	FetchRange(context.Context, string, string) (*safefetch.Result, error)
}

func Acquire(ctx context.Context, loaded *LoadedWorkOrder, control *Control, output, stateRoot string, now func() time.Time, pause func(time.Duration)) (AcquisitionManifest, error) {
	return acquire(ctx, loaded, control, output, stateRoot, now, pause, nil)
}

func acquire(ctx context.Context, loaded *LoadedWorkOrder, control *Control, output, stateRoot string, now func() time.Time, pause func(time.Duration), injected rangeRetriever) (AcquisitionManifest, error) {
	var manifest AcquisitionManifest
	if loaded == nil || !loaded.AuthorityVerified || control == nil || now == nil || pause == nil || loaded.Order.Validate() != nil || digest(loaded.Bytes) != loaded.Digest {
		return manifest, errors.New("opportunity pilot: sealed order, control, UTC clock and pause function are required")
	}
	if err := control.Permits(loaded.Order); err != nil {
		return manifest, err
	}
	started := now().UTC()
	notBefore, _ := canonicalTime(loaded.Order.NotBefore)
	expires, _ := canonicalTime(loaded.Order.ExpiresAt)
	if started.Before(notBefore) || !started.Before(expires) {
		return manifest, errors.New("opportunity pilot: work order is outside its validity interval")
	}
	root, err := createOutput(output)
	if err != nil {
		return manifest, err
	}
	claimBytes, err := claimExecution(stateRoot, loaded, started)
	if err != nil {
		return manifest, err
	}
	if err := atomicfile.Write(filepath.Join(root, "execution-claim.json"), claimBytes, MaxManifest, 0o440); err != nil {
		return manifest, err
	}
	claimDigest := digest(claimBytes)
	if injected == nil {
		policy := safefetch.DefaultPolicy()
		policy.ID = "tw.e4.opportunity-bulk-v0"
		policy.AllowedHosts = []string{SourceHost}
		policy.MaxRedirects = 0
		policy.MaxBodyBytes = int64(loaded.Order.RangeBytes)
		policy.RequestTimeout = time.Duration(loaded.Order.RequestTimeoutMillis) * time.Millisecond
		policy.ConnectTimeout = 5 * time.Second
		policy.ResponseHeaderTimeout = 10 * time.Second
		policy.UserAgent = "TWIRXBot/0.2 (+https://twirx.org/bot; contact:security@twirx.org)"
		fetcher, err := safefetch.New(policy)
		if err != nil {
			return manifest, err
		}
		injected = fetcher
	}
	archiveTemp, err := os.CreateTemp(root, ".source-archive-*")
	if err != nil {
		return manifest, err
	}
	tempName := archiveTemp.Name()
	defer func() { _ = os.Remove(tempName) }()
	archiveHash := sha256.New()
	first := ByteRange{Index: 0, Start: 0, End: RangeBytes - 1}
	firstResult, err := injected.FetchRange(ctx, loaded.Order.SourceURL, first.Header())
	if err != nil {
		_ = archiveTemp.Close()
		return manifest, fmt.Errorf("opportunity pilot: retrieve first sealed range: %w", err)
	}
	total, err := validateRangeResult(firstResult, first, 0)
	if err != nil {
		_ = archiveTemp.Close()
		return manifest, err
	}
	ranges, err := BuildRanges(total)
	if err != nil || len(ranges) == 0 || ranges[0] != first {
		_ = archiveTemp.Close()
		return manifest, errors.New("opportunity pilot: first range does not establish a reviewed archive layout")
	}
	manifest = AcquisitionManifest{Format: AcquisitionFormat, WorkOrderID: loaded.Order.ID, WorkOrderDigest: loaded.Digest, OriginID: OriginID, SourceURL: SourceURL, SourceFilename: SourceFilename, PolicyDecisionDigest: loaded.Order.DecisionDigest, ExecutionClaimDigest: claimDigest, StartedAt: started.Format(time.RFC3339), SchedulerEnabled: false, RawEvidencePublic: false}
	manifest.Artifacts = append(manifest.Artifacts, AcquisitionArtifact{Path: "execution-claim.json", Digest: claimDigest, Size: uint64(len(claimBytes))})
	for index, item := range ranges {
		result := firstResult
		if index > 0 {
			pause(time.Duration(loaded.Order.RequestIntervalMillis) * time.Millisecond)
			result, err = injected.FetchRange(ctx, loaded.Order.SourceURL, item.Header())
			if err != nil {
				_ = archiveTemp.Close()
				return AcquisitionManifest{}, fmt.Errorf("opportunity pilot: retrieve sealed range %d: %w", index, err)
			}
		}
		if _, err := validateRangeResult(result, item, total); err != nil {
			_ = archiveTemp.Close()
			return AcquisitionManifest{}, err
		}
		evidence := RangeEvidence{Format: RangeFormat, Index: item.Index, SourceURL: SourceURL, RequestedRange: item.Header(), ContentRange: result.ContentRange, HTTPStatus: result.Status, MediaType: result.MediaType, RetrievedAt: result.RetrievedAt.UTC().Format(time.RFC3339Nano), BodyDigest: digest(result.Body), BodySize: uint64(len(result.Body)), TotalSize: total}
		evidenceBytes, err := marshalJSON(evidence, MaxManifest)
		if err != nil {
			_ = archiveTemp.Close()
			return AcquisitionManifest{}, err
		}
		base := filepath.ToSlash(filepath.Join("ranges", fmt.Sprintf("%06d", index)))
		bodyPath := filepath.Join(root, filepath.FromSlash(base), "body.part")
		evidencePath := filepath.Join(root, filepath.FromSlash(base), "evidence.json")
		if err := atomicfile.Write(bodyPath, result.Body, int(RangeBytes), 0o440); err != nil {
			_ = archiveTemp.Close()
			return AcquisitionManifest{}, err
		}
		if err := atomicfile.Write(evidencePath, evidenceBytes, MaxManifest, 0o440); err != nil {
			_ = archiveTemp.Close()
			return AcquisitionManifest{}, err
		}
		if err := writeArchiveChunk(archiveTemp, archiveHash, result.Body); err != nil {
			_ = archiveTemp.Close()
			return AcquisitionManifest{}, err
		}
		manifest.Artifacts = append(manifest.Artifacts,
			AcquisitionArtifact{Path: base + "/body.part", Digest: evidence.BodyDigest, Size: evidence.BodySize},
			AcquisitionArtifact{Path: base + "/evidence.json", Digest: digest(evidenceBytes), Size: uint64(len(evidenceBytes))},
		)
		manifest.NetworkRequests++
		manifest.TransferredBytes += uint64(len(result.Body))
	}
	if manifest.TransferredBytes != total {
		_ = archiveTemp.Close()
		return AcquisitionManifest{}, errors.New("opportunity pilot: range bodies do not reconstruct the advertised archive")
	}
	if err := archiveTemp.Sync(); err != nil || archiveTemp.Chmod(0o440) != nil || archiveTemp.Close() != nil {
		return AcquisitionManifest{}, errors.New("opportunity pilot: finalize reconstructed archive")
	}
	archivePath := filepath.Join(root, SourceFilename)
	if err := os.Rename(tempName, archivePath); err != nil {
		return AcquisitionManifest{}, err
	}
	archiveDigest := "sha256:" + hex.EncodeToString(archiveHash.Sum(nil))
	manifest.ArchiveDigest = archiveDigest
	manifest.ArchiveSize = total
	manifest.Artifacts = append(manifest.Artifacts, AcquisitionArtifact{Path: SourceFilename, Digest: archiveDigest, Size: total})
	completed := now().UTC()
	if completed.Before(started) || !completed.Before(expires) {
		return AcquisitionManifest{}, errors.New("opportunity pilot: completion time is invalid")
	}
	manifest.CompletedAt = completed.Format(time.RFC3339)
	sortArtifacts(manifest.Artifacts)
	manifestBytes, err := marshalJSON(manifest, MaxManifest)
	if err != nil {
		return AcquisitionManifest{}, err
	}
	// Manifest-last publication is the only completion signal. No ZIP or XML
	// parser runs anywhere in this acquisition function.
	if err := atomicfile.Write(filepath.Join(root, "acquisition-manifest.json"), manifestBytes, MaxManifest, 0o440); err != nil {
		return AcquisitionManifest{}, err
	}
	return manifest, nil
}

func VerifyAcquisition(root string, loaded *LoadedWorkOrder) (AcquisitionManifest, error) {
	var manifest AcquisitionManifest
	if loaded == nil || !loaded.AuthorityVerified || loaded.Order.Validate() != nil || digest(loaded.Bytes) != loaded.Digest {
		return manifest, errors.New("opportunity pilot: exact work order is required")
	}
	manifestBytes, err := readRegular(filepath.Join(root, "acquisition-manifest.json"), MaxManifest)
	if err != nil || decode(manifestBytes, &manifest, MaxManifest) != nil {
		return AcquisitionManifest{}, errors.New("opportunity pilot: acquisition manifest is unavailable or invalid")
	}
	if manifest.Format != AcquisitionFormat || manifest.WorkOrderID != loaded.Order.ID || manifest.WorkOrderDigest != loaded.Digest || manifest.OriginID != OriginID || manifest.SourceURL != SourceURL || manifest.SourceFilename != SourceFilename || manifest.PolicyDecisionDigest != loaded.Order.DecisionDigest || !validDigest(manifest.ExecutionClaimDigest) || manifest.SchedulerEnabled || manifest.RawEvidencePublic || manifest.NetworkRequests == 0 || manifest.NetworkRequests > MaximumRequests || manifest.ArchiveSize == 0 || manifest.ArchiveSize > MaximumArchive || manifest.TransferredBytes != manifest.ArchiveSize || !validDigest(manifest.ArchiveDigest) || len(manifest.Artifacts) != int(manifest.NetworkRequests*2+2) {
		return AcquisitionManifest{}, errors.New("opportunity pilot: acquisition manifest authority or counts are invalid")
	}
	started, err := canonicalTime(manifest.StartedAt)
	if err != nil {
		return AcquisitionManifest{}, errors.New("opportunity pilot: acquisition start time is invalid")
	}
	completed, err := canonicalTime(manifest.CompletedAt)
	if err != nil || completed.Before(started) {
		return AcquisitionManifest{}, errors.New("opportunity pilot: acquisition completion time is invalid")
	}
	notBefore, _ := canonicalTime(loaded.Order.NotBefore)
	expires, _ := canonicalTime(loaded.Order.ExpiresAt)
	if started.Before(notBefore) || !started.Before(expires) || !completed.Before(expires) {
		return AcquisitionManifest{}, errors.New("opportunity pilot: acquisition occurred outside the work-order validity interval")
	}
	claimBytes, err := readRegular(filepath.Join(root, "execution-claim.json"), MaxManifest)
	var claim ExecutionClaim
	if err != nil || digest(claimBytes) != manifest.ExecutionClaimDigest || decode(claimBytes, &claim, MaxManifest) != nil || claim.Format != ExecutionClaimFormat || claim.WorkOrderID != loaded.Order.ID || claim.WorkOrderDigest != loaded.Digest || claim.OriginID != OriginID || claim.ClaimedAt != manifest.StartedAt {
		return AcquisitionManifest{}, errors.New("opportunity pilot: execution claim does not reconcile")
	}
	if !artifactsSortedUnique(manifest.Artifacts) {
		return AcquisitionManifest{}, errors.New("opportunity pilot: acquisition artifacts are not sorted and unique")
	}
	archive, err := readRegular(filepath.Join(root, SourceFilename), int64(MaximumArchive))
	if err != nil || uint64(len(archive)) != manifest.ArchiveSize || digest(archive) != manifest.ArchiveDigest {
		return AcquisitionManifest{}, errors.New("opportunity pilot: reconstructed archive does not verify")
	}
	ranges, err := BuildRanges(manifest.ArchiveSize)
	if err != nil || uint64(len(ranges)) != manifest.NetworkRequests {
		return AcquisitionManifest{}, errors.New("opportunity pilot: archive range topology is invalid")
	}
	offset := uint64(0)
	expectedArtifacts := make([]AcquisitionArtifact, 0, len(manifest.Artifacts))
	expectedArtifacts = append(expectedArtifacts, AcquisitionArtifact{Path: "execution-claim.json", Digest: manifest.ExecutionClaimDigest, Size: uint64(len(claimBytes))})
	for _, item := range ranges {
		base := filepath.Join(root, "ranges", fmt.Sprintf("%06d", item.Index))
		body, err := readRegular(filepath.Join(base, "body.part"), int64(RangeBytes))
		if err != nil || uint64(len(body)) != item.End-item.Start+1 || string(body) != string(archive[item.Start:item.End+1]) {
			return AcquisitionManifest{}, errors.New("opportunity pilot: range body does not reconcile with archive")
		}
		evidenceBytes, err := readRegular(filepath.Join(base, "evidence.json"), MaxManifest)
		var evidence RangeEvidence
		if err != nil || decode(evidenceBytes, &evidence, MaxManifest) != nil {
			return AcquisitionManifest{}, errors.New("opportunity pilot: range evidence is unavailable or invalid")
		}
		retrieved, retrievedErr := time.Parse(time.RFC3339Nano, evidence.RetrievedAt)
		// The manifest binds canonical UTC seconds while individual range
		// evidence preserves nanoseconds. Compare the range's canonical second
		// to the manifest interval so an honestly retained sub-second timestamp
		// in the completion second does not appear to be later than completion.
		if evidence.Format != RangeFormat || evidence.Index != item.Index || evidence.SourceURL != SourceURL || evidence.RequestedRange != item.Header() || evidence.HTTPStatus != 206 || evidence.MediaType != "application/zip" && evidence.MediaType != "application/octet-stream" || retrievedErr != nil || retrieved.Location() != time.UTC || retrieved.Format(time.RFC3339Nano) != evidence.RetrievedAt || retrieved.Before(started) || retrieved.Truncate(time.Second).After(completed) || evidence.BodyDigest != digest(body) || evidence.BodySize != uint64(len(body)) || evidence.TotalSize != manifest.ArchiveSize {
			return AcquisitionManifest{}, errors.New("opportunity pilot: range evidence does not reconcile")
		}
		if _, parsedTotal, err := parseContentRange(evidence.ContentRange, item); err != nil || parsedTotal != manifest.ArchiveSize {
			return AcquisitionManifest{}, errors.New("opportunity pilot: range Content-Range is invalid")
		}
		offset += uint64(len(body))
		expectedArtifacts = append(expectedArtifacts,
			AcquisitionArtifact{Path: filepath.ToSlash(filepath.Join("ranges", fmt.Sprintf("%06d", item.Index), "body.part")), Digest: digest(body), Size: uint64(len(body))},
			AcquisitionArtifact{Path: filepath.ToSlash(filepath.Join("ranges", fmt.Sprintf("%06d", item.Index), "evidence.json")), Digest: digest(evidenceBytes), Size: uint64(len(evidenceBytes))},
		)
	}
	if offset != manifest.ArchiveSize {
		return AcquisitionManifest{}, errors.New("opportunity pilot: range coverage is incomplete")
	}
	expectedArtifacts = append(expectedArtifacts, AcquisitionArtifact{Path: SourceFilename, Digest: manifest.ArchiveDigest, Size: manifest.ArchiveSize})
	sortArtifacts(expectedArtifacts)
	for index := range expectedArtifacts {
		if manifest.Artifacts[index] != expectedArtifacts[index] {
			return AcquisitionManifest{}, errors.New("opportunity pilot: acquisition artifact index does not reconcile")
		}
	}
	return manifest, nil
}

func claimExecution(stateRoot string, loaded *LoadedWorkOrder, started time.Time) ([]byte, error) {
	if stateRoot == "" {
		return nil, errors.New("opportunity pilot: one-shot execution state is required")
	}
	absolute, err := filepath.Abs(stateRoot)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(absolute)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("opportunity pilot: execution state must be a real existing directory")
	}
	claim := ExecutionClaim{Format: ExecutionClaimFormat, WorkOrderID: loaded.Order.ID, WorkOrderDigest: loaded.Digest, OriginID: OriginID, ClaimedAt: started.Format(time.RFC3339)}
	data, err := marshalJSON(claim, MaxManifest)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(absolute, loaded.Order.ID+".claim.json")
	fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o440)
	if err != nil {
		if errors.Is(err, syscall.EEXIST) {
			return nil, errors.New("opportunity pilot: manual-once work order is already consumed")
		}
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("opportunity pilot: cannot create execution claim")
	}
	if _, err := file.Write(data); err != nil || file.Sync() != nil || file.Close() != nil {
		return nil, errors.New("opportunity pilot: cannot persist one-shot execution claim")
	}
	return data, nil
}

func validateRangeResult(result *safefetch.Result, expected ByteRange, knownTotal uint64) (uint64, error) {
	if result == nil || result.RequestURL != SourceURL || result.FinalURL != SourceURL || result.Method != "GET" || result.Status != 206 || result.RequestedRange != expected.Header() || len(result.Redirects) != 0 || result.RetrievedAt.Location() != time.UTC || result.RetrievedAt.IsZero() || result.MediaType != "application/zip" && result.MediaType != "application/octet-stream" || uint64(len(result.Body)) != expected.End-expected.Start+1 {
		return 0, errors.New("opportunity pilot: range response identity, status, media type, time, or size is invalid")
	}
	_, total, err := parseContentRange(result.ContentRange, expected)
	if err != nil || knownTotal != 0 && total != knownTotal {
		return 0, errors.New("opportunity pilot: Content-Range does not match the sealed request")
	}
	return total, nil
}

func parseContentRange(value string, expected ByteRange) (uint64, uint64, error) {
	prefix := fmt.Sprintf("bytes %d-%d/", expected.Start, expected.End)
	if !strings.HasPrefix(value, prefix) {
		return 0, 0, errors.New("invalid Content-Range prefix")
	}
	totalText := strings.TrimPrefix(value, prefix)
	if totalText == "" || strings.TrimSpace(totalText) != totalText || strings.HasPrefix(totalText, "+") {
		return 0, 0, errors.New("invalid Content-Range total")
	}
	total, err := strconv.ParseUint(totalText, 10, 64)
	if err != nil || total <= expected.End || total > MaximumArchive {
		return 0, 0, errors.New("invalid Content-Range total")
	}
	return expected.End - expected.Start + 1, total, nil
}

func createOutput(path string) (string, error) {
	if path == "" {
		return "", errors.New("opportunity pilot: output is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(absolute)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("opportunity pilot: output parent must be a real directory")
	}
	if _, err := os.Lstat(absolute); err == nil {
		return "", errors.New("opportunity pilot: immutable output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Mkdir(absolute, 0o750); err != nil {
		return "", err
	}
	return absolute, nil
}

func writeArchiveChunk(writer io.Writer, hasher hash.Hash, data []byte) error {
	if _, err := writer.Write(data); err != nil {
		return err
	}
	_, err := hasher.Write(data)
	return err
}

func marshalJSON(value any, maximum int) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) == 0 || len(data) > maximum {
		return nil, errors.New("opportunity pilot: JSON artifact exceeds its bound")
	}
	return data, nil
}

func sortArtifacts(values []AcquisitionArtifact) {
	sort.Slice(values, func(i, j int) bool { return values[i].Path < values[j].Path })
}

func artifactsSortedUnique(values []AcquisitionArtifact) bool {
	for index, value := range values {
		if value.Path == "" || filepath.IsAbs(value.Path) || filepath.Clean(value.Path) != value.Path || strings.Contains(value.Path, "..") || !validDigest(value.Digest) || value.Size == 0 || index > 0 && values[index-1].Path >= value.Path {
			return false
		}
	}
	return true
}
