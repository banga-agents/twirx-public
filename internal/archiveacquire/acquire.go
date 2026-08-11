// Package archiveacquire performs one founder-approved Common Crawl
// acquisition plan. Its network authority is fixed to the official Common
// Crawl index and data hosts; origin URLs come only from a sealed work order.
package archiveacquire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/typed-web-commons/typed-web/internal/archiveimport"
	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

const (
	ManifestFormat = "tw.archive-acquisition-manifest/0.1"
	MaxManifest    = 1 << 20
)

type retriever interface {
	Fetch(context.Context, string) (*safefetch.Result, error)
	FetchRange(context.Context, string, string) (*safefetch.Result, error)
}

type Runner struct {
	index retriever
	data  retriever
}

type Artifact struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   uint64 `json:"size"`
}

type Capture struct {
	CollectionID         string `json:"collection_id"`
	Route                string `json:"route"`
	CaptureTimestamp     string `json:"capture_timestamp"`
	SpoolPath            string `json:"spool_path"`
	SpoolManifestDigest  string `json:"spool_manifest_digest"`
	RepresentationDigest string `json:"representation_digest"`
	RepresentationSize   uint64 `json:"representation_size"`
}

type Manifest struct {
	Format              string     `json:"format"`
	WorkOrderID         string     `json:"work_order_id"`
	WorkOrderDigest     string     `json:"work_order_digest"`
	IndexHost           string     `json:"index_host"`
	DataHost            string     `json:"data_host"`
	IndexRequests       uint64     `json:"index_requests"`
	RangeRequests       uint64     `json:"range_requests"`
	NetworkRequestsMade uint64     `json:"network_requests_made"`
	Artifacts           []Artifact `json:"artifacts"`
	Captures            []Capture  `json:"captures"`
}

func NewRunner() (*Runner, error) {
	indexPolicy := safefetch.DefaultPolicy()
	indexPolicy.ID = "tw.archive-acquire.index-v0"
	indexPolicy.MaxRedirects = 0
	indexPolicy.MaxBodyBytes = archiveimport.MaxIndexResponse
	indexPolicy.RequestTimeout = 30 * time.Second
	indexPolicy.AllowedHosts = []string{archiveimport.OfficialIndexHost}
	indexPolicy.UserAgent = "TWIRXArchive/0.1 (+https://twirx.org/bot; contact:security@twirx.org)"
	indexFetcher, err := safefetch.New(indexPolicy)
	if err != nil {
		return nil, err
	}
	dataPolicy := safefetch.DefaultPolicy()
	dataPolicy.ID = "tw.archive-acquire.data-v0"
	dataPolicy.MaxRedirects = 0
	dataPolicy.MaxBodyBytes = archiveimport.MaxCompressed
	dataPolicy.RequestTimeout = 30 * time.Second
	dataPolicy.AllowedHosts = []string{archiveimport.OfficialDataHost}
	dataPolicy.UserAgent = indexPolicy.UserAgent
	dataFetcher, err := safefetch.New(dataPolicy)
	if err != nil {
		return nil, err
	}
	return &Runner{index: indexFetcher, data: dataFetcher}, nil
}

func newRunner(index, data retriever) (*Runner, error) {
	if index == nil || data == nil {
		return nil, errors.New("archiveacquire: both restricted retrievers are required")
	}
	return &Runner{index: index, data: data}, nil
}

// Acquire executes every exact index query in the sealed work order. Raw index
// and range bytes are atomically stored before entering an index or WARC
// parser. acquisition-manifest.json is written last; its absence denotes an
// incomplete acquisition that has no publication authority.
func (r *Runner) Acquire(ctx context.Context, loaded *archiveimport.LoadedWorkOrder, output string) (*Manifest, error) {
	if r == nil || r.index == nil || r.data == nil || loaded == nil || loaded.Order.Validate() != nil || digest(loaded.Bytes) != loaded.Digest {
		return nil, errors.New("archiveacquire: acquisition authority is incomplete")
	}
	root, err := createOutput(output)
	if err != nil {
		return nil, err
	}
	if err := os.Mkdir(filepath.Join(root, "raw"), 0o750); err != nil {
		return nil, err
	}
	if err := os.Mkdir(filepath.Join(root, "captures"), 0o750); err != nil {
		return nil, err
	}
	if err := atomicfile.Write(filepath.Join(root, "work-order.json"), loaded.Bytes, archiveimport.MaxWorkOrder, 0o440); err != nil {
		return nil, err
	}
	manifest := &Manifest{Format: ManifestFormat, WorkOrderID: loaded.Order.ID, WorkOrderDigest: loaded.Digest, IndexHost: archiveimport.OfficialIndexHost, DataHost: archiveimport.OfficialDataHost}
	manifest.Artifacts = append(manifest.Artifacts, artifactFor("work-order.json", loaded.Bytes))
	entry := 0
	captureNumber := 0
	for _, collection := range loaded.Order.CollectionIDs {
		for _, route := range loaded.Order.PermittedRoutes {
			if manifest.IndexRequests >= loaded.Order.IndexRequestBudget {
				return nil, errors.New("archiveacquire: sealed index-request budget exhausted")
			}
			indexURL, err := archiveimport.BuildIndexURL(loaded.Order, collection, route)
			if err != nil {
				return nil, err
			}
			indexResult, err := r.index.Fetch(ctx, indexURL)
			manifest.IndexRequests++
			manifest.NetworkRequestsMade++
			if err != nil {
				return nil, fmt.Errorf("archiveacquire: index request: %w", err)
			}
			if indexResult == nil || indexResult.RequestURL != indexURL || indexResult.FinalURL != indexURL || indexResult.Method != "GET" || indexResult.Status != 200 || len(indexResult.Redirects) != 0 || indexResult.RequestedRange != "" || len(indexResult.Body) == 0 || uint64(len(indexResult.Body)) > loaded.Order.MaxIndexResponseBytes {
				return nil, errors.New("archiveacquire: index response escaped its exact official-host authority")
			}
			indexName := fmt.Sprintf("raw/index-%03d.jsonl", entry)
			indexPath := filepath.Join(root, filepath.FromSlash(indexName))
			if err := atomicfile.Write(indexPath, indexResult.Body, int(loaded.Order.MaxIndexResponseBytes), 0o440); err != nil {
				return nil, err
			}
			manifest.Artifacts = append(manifest.Artifacts, artifactFor(indexName, indexResult.Body))
			captures, err := archiveimport.LoadIndexResponse(indexPath, loaded.Order, collection, route)
			if err != nil {
				return nil, err
			}
			for _, capture := range captures {
				dataURL, err := capture.DataURL()
				if err != nil {
					return nil, err
				}
				rangeHeader, err := capture.RangeHeader()
				if err != nil {
					return nil, err
				}
				rangeResult, err := r.data.FetchRange(ctx, dataURL, rangeHeader)
				manifest.RangeRequests++
				manifest.NetworkRequestsMade++
				if err != nil {
					return nil, fmt.Errorf("archiveacquire: range request: %w", err)
				}
				if rangeResult == nil || rangeResult.RequestURL != dataURL || rangeResult.FinalURL != dataURL || rangeResult.Method != "GET" || rangeResult.Status != 206 || len(rangeResult.Redirects) != 0 || rangeResult.RequestedRange != rangeHeader || rangeResult.ContentRange == "" {
					return nil, errors.New("archiveacquire: range response escaped its exact official-host authority")
				}
				if err := archiveimport.ValidateRangeResponse(capture, rangeResult.Status, rangeResult.ContentRange, rangeResult.Body, loaded.Order.MaxCompressedRecordBytes); err != nil {
					return nil, err
				}
				rangeName := fmt.Sprintf("raw/range-%03d.warc.gz", captureNumber)
				rangePath := filepath.Join(root, filepath.FromSlash(rangeName))
				if err := atomicfile.Write(rangePath, rangeResult.Body, int(loaded.Order.MaxCompressedRecordBytes), 0o440); err != nil {
					return nil, err
				}
				manifest.Artifacts = append(manifest.Artifacts, artifactFor(rangeName, rangeResult.Body))
				spoolName := fmt.Sprintf("captures/capture-%03d", captureNumber)
				spoolPath := filepath.Join(root, filepath.FromSlash(spoolName))
				evidence, err := archiveimport.PublishCaptureFile(spoolPath, loaded, capture, rangeResult.Status, rangeResult.ContentRange, rangePath)
				if err != nil {
					return nil, err
				}
				verified, err := archiveimport.VerifySpool(spoolPath)
				if err != nil || *verified != *evidence {
					return nil, errors.New("archiveacquire: published capture failed immediate reconciliation")
				}
				spoolManifest, err := readRegular(filepath.Join(spoolPath, "manifest.json"), archiveimport.MaxMetadata)
				if err != nil {
					return nil, err
				}
				spoolManifestName := spoolName + "/manifest.json"
				manifest.Artifacts = append(manifest.Artifacts, artifactFor(spoolManifestName, spoolManifest))
				manifest.Captures = append(manifest.Captures, Capture{CollectionID: collection, Route: route, CaptureTimestamp: evidence.CaptureTimestamp, SpoolPath: spoolName, SpoolManifestDigest: digest(spoolManifest), RepresentationDigest: evidence.RepresentationDigest, RepresentationSize: evidence.RepresentationSize})
				captureNumber++
			}
			entry++
		}
	}
	if len(manifest.Captures) == 0 {
		return nil, errors.New("archiveacquire: work order produced no captures")
	}
	sort.Slice(manifest.Artifacts, func(i, j int) bool { return manifest.Artifacts[i].Path < manifest.Artifacts[j].Path })
	if err := manifest.Validate(loaded.Order); err != nil {
		return nil, err
	}
	manifestBytes, err := marshal(manifest)
	if err != nil {
		return nil, err
	}
	if err := atomicfile.Write(filepath.Join(root, "acquisition-manifest.json"), manifestBytes, MaxManifest, 0o440); err != nil {
		return nil, err
	}
	return manifest, nil
}

func Verify(root string) (*Manifest, error) {
	manifestBytes, err := readRegular(filepath.Join(root, "acquisition-manifest.json"), MaxManifest)
	if err != nil {
		return nil, fmt.Errorf("archiveacquire: final manifest unavailable: %w", err)
	}
	var manifest Manifest
	if err := decodeManifest(manifestBytes, &manifest); err != nil {
		return nil, err
	}
	orderBytes, err := readRegular(filepath.Join(root, "work-order.json"), archiveimport.MaxWorkOrder)
	if err != nil || digest(orderBytes) != manifest.WorkOrderDigest {
		return nil, errors.New("archiveacquire: work order does not match the acquisition manifest")
	}
	var order archiveimport.WorkOrder
	if err := jsonbounded.Decode(orderBytes, &order, jsonbounded.Policy{MaxBytes: archiveimport.MaxWorkOrder, MaxDepth: 16, MaxScalarBytes: 16 << 10, MaxContainerEntries: 512, MaxTokens: 10000}, true); err != nil || order.ID != manifest.WorkOrderID || order.Validate() != nil || manifest.Validate(order) != nil {
		return nil, errors.New("archiveacquire: invalid work-order or acquisition binding")
	}
	for _, artifact := range manifest.Artifacts {
		data, err := readRegular(filepath.Join(root, filepath.FromSlash(artifact.Path)), artifactMaximum(artifact.Path, order))
		if err != nil || digest(data) != artifact.Digest || uint64(len(data)) != artifact.Size {
			return nil, fmt.Errorf("archiveacquire: artifact mismatch for %s", artifact.Path)
		}
	}
	for _, capture := range manifest.Captures {
		evidence, err := archiveimport.VerifySpool(filepath.Join(root, filepath.FromSlash(capture.SpoolPath)))
		if err != nil || evidence.CollectionID != capture.CollectionID || evidence.TargetURL != capture.Route || evidence.CaptureTimestamp != capture.CaptureTimestamp || evidence.RepresentationDigest != capture.RepresentationDigest || evidence.RepresentationSize != capture.RepresentationSize {
			return nil, fmt.Errorf("archiveacquire: capture mismatch for %s", capture.SpoolPath)
		}
	}
	return &manifest, nil
}

func (m Manifest) Validate(order archiveimport.WorkOrder) error {
	expectedArtifacts := 1 + int(m.IndexRequests) + 2*int(m.RangeRequests)
	if m.Format != ManifestFormat || m.WorkOrderID != order.ID || !validDigest(m.WorkOrderDigest) || m.IndexHost != archiveimport.OfficialIndexHost || m.DataHost != archiveimport.OfficialDataHost || m.IndexRequests != uint64(len(order.CollectionIDs)*len(order.PermittedRoutes)) || m.IndexRequests == 0 || m.IndexRequests > order.IndexRequestBudget || m.RangeRequests == 0 || m.NetworkRequestsMade != m.IndexRequests+m.RangeRequests || len(m.Captures) == 0 || uint64(len(m.Captures)) != m.RangeRequests || len(m.Artifacts) != expectedArtifacts {
		return errors.New("archiveacquire: invalid acquisition manifest counters or authority")
	}
	previous := ""
	seen := make(map[string]Artifact, len(m.Artifacts))
	for _, artifact := range m.Artifacts {
		if !safeRelative(artifact.Path) || artifact.Path <= previous || !validDigest(artifact.Digest) || artifact.Size == 0 {
			return errors.New("archiveacquire: artifacts must be safe, sorted, unique and digest-bound")
		}
		seen[artifact.Path] = artifact
		previous = artifact.Path
	}
	if _, ok := seen["work-order.json"]; !ok {
		return errors.New("archiveacquire: work-order artifact is missing")
	}
	previous = ""
	for index, capture := range m.Captures {
		expected := fmt.Sprintf("captures/capture-%03d", index)
		parsedTime, timeErr := time.Parse("20060102150405", capture.CaptureTimestamp)
		if capture.SpoolPath != expected || capture.SpoolPath <= previous || !orderContains(order.CollectionIDs, capture.CollectionID) || !orderContains(order.PermittedRoutes, capture.Route) || timeErr != nil || parsedTime.UTC().Format("20060102150405") != capture.CaptureTimestamp || !validDigest(capture.SpoolManifestDigest) || !validDigest(capture.RepresentationDigest) || capture.RepresentationSize == 0 || capture.RepresentationSize > order.MaxRetainedBodyBytes {
			return errors.New("archiveacquire: invalid capture manifest entry")
		}
		artifact, ok := seen[capture.SpoolPath+"/manifest.json"]
		if !ok || artifact.Digest != capture.SpoolManifestDigest {
			return errors.New("archiveacquire: capture manifest artifact is missing")
		}
		previous = capture.SpoolPath
	}
	return nil
}

func decodeManifest(data []byte, target *Manifest) error {
	if err := jsonbounded.Decode(data, target, jsonbounded.Policy{MaxBytes: MaxManifest, MaxDepth: 16, MaxScalarBytes: 16 << 10, MaxContainerEntries: 1 << 15, MaxTokens: 1 << 18}, true); err != nil {
		return fmt.Errorf("archiveacquire: decode manifest: %w", err)
	}
	return nil
}

func marshal(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) == 0 || len(data) > MaxManifest {
		return nil, errors.New("archiveacquire: manifest exceeds its bound")
	}
	return data, nil
}

func createOutput(path string) (string, error) {
	if path == "" {
		return "", errors.New("archiveacquire: output path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(absolute)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("archiveacquire: output parent must be a real directory")
	}
	if _, err := os.Lstat(absolute); err == nil {
		return "", errors.New("archiveacquire: immutable output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Mkdir(absolute, 0o750); err != nil {
		return "", err
	}
	return absolute, nil
}

func readRegular(path string, maximum int64) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("archiveacquire: cannot open artifact")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	stat, statOK := info.Sys().(*syscall.Stat_t)
	if maximum <= 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximum || !statOK || stat.Nlink != 1 {
		return nil, errors.New("archiveacquire: artifact is not a bounded single-link regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(data)) != info.Size() || int64(len(data)) > maximum {
		return nil, errors.New("archiveacquire: artifact changed or exceeded its bound while read")
	}
	return data, nil
}

func artifactFor(path string, data []byte) Artifact {
	return Artifact{Path: path, Digest: digest(data), Size: uint64(len(data))}
}

func artifactMaximum(path string, order archiveimport.WorkOrder) int64 {
	switch {
	case path == "work-order.json":
		return archiveimport.MaxWorkOrder
	case strings.HasPrefix(path, "raw/index-"):
		return int64(order.MaxIndexResponseBytes)
	case strings.HasPrefix(path, "raw/range-"):
		return int64(order.MaxCompressedRecordBytes)
	case strings.HasPrefix(path, "captures/") && strings.HasSuffix(path, "/manifest.json"):
		return archiveimport.MaxMetadata
	default:
		return 0
	}
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

func safeRelative(value string) bool {
	return value != "" && !filepath.IsAbs(value) && !strings.Contains(value, "\\") && filepath.ToSlash(filepath.Clean(value)) == value && value != "." && !strings.HasPrefix(value, "../") && !strings.Contains(value, "/../")
}

func orderContains(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}
