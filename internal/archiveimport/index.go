package archiveimport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	OfficialIndexHost = "index.commoncrawl.org"
	OfficialDataHost  = "data.commoncrawl.org"
	MaxIndexLine      = 64 << 10
)

var providerDigestPattern = regexp.MustCompile(`^(?:sha1:)?[A-Z2-7]{32}$`)

type Capture struct {
	CollectionID   string `json:"collection_id"`
	RequestedURL   string `json:"requested_url"`
	CapturedURL    string `json:"captured_url"`
	Timestamp      string `json:"timestamp"`
	Status         uint64 `json:"status"`
	MediaType      string `json:"media_type"`
	ProviderDigest string `json:"provider_digest"`
	Filename       string `json:"filename"`
	Offset         uint64 `json:"offset"`
	Length         uint64 `json:"length"`
	raw            []byte
}

func BuildIndexURL(order WorkOrder, collectionID, route string) (string, error) {
	if err := order.Validate(); err != nil || !order.permitsCollection(collectionID) || !order.permitsRoute(route) {
		return "", errors.New("archiveimport: index query is absent from the sealed work order")
	}
	selection, ok := order.selectedCapture(collectionID, route)
	if !ok {
		return "", errors.New("archiveimport: index query lacks an exact selected capture")
	}
	query := url.Values{}
	query.Add("filter", "status:200")
	query.Add("filter", "timestamp:"+selection.Timestamp)
	query.Set("matchType", "exact")
	query.Set("output", "json")
	query.Set("pageSize", strconv.FormatUint(order.CapturesPerCollection, 10))
	query.Set("url", route)
	return (&url.URL{Scheme: "https", Host: OfficialIndexHost, Path: "/" + collectionID + "-index", RawQuery: query.Encode()}).String(), nil
}

func (c Capture) DataURL() (string, error) {
	if err := c.validateIdentity(); err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "https", Host: OfficialDataHost, Path: "/" + c.Filename}).String(), nil
}

func (c Capture) RangeHeader() (string, error) {
	if err := c.validateIdentity(); err != nil || c.Offset > ^uint64(0)-(c.Length-1) {
		return "", errors.New("archiveimport: invalid capture byte range")
	}
	return fmt.Sprintf("bytes=%d-%d", c.Offset, c.Offset+c.Length-1), nil
}

func ParseIndexResponse(data []byte, order WorkOrder, collectionID, route string) ([]Capture, error) {
	if err := order.Validate(); err != nil || !order.permitsCollection(collectionID) || !order.permitsRoute(route) {
		return nil, errors.New("archiveimport: index response has no sealed authority")
	}
	if len(data) == 0 || uint64(len(data)) > order.MaxIndexResponseBytes || len(data) > MaxIndexResponse {
		return nil, errors.New("archiveimport: index response exceeds its bound")
	}
	lines := bytes.Split(data, []byte{'\n'})
	captures := make([]Capture, 0, len(lines))
	identities := make(map[string]struct{})
	digests := make(map[string]string)
	selection, selected := order.selectedCapture(collectionID, route)
	if !selected {
		return nil, errors.New("archiveimport: index response lacks an exact selected capture")
	}
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if len(line) > MaxIndexLine {
			return nil, errors.New("archiveimport: oversized index line")
		}
		capture, err := parseIndexLine(line, collectionID, route)
		if err != nil {
			return nil, err
		}
		if capture.Length > order.MaxCompressedRecordBytes {
			return nil, errors.New("archiveimport: capture exceeds the sealed compressed-record budget")
		}
		if !selection.matches(capture) {
			return nil, errors.New("archiveimport: index record does not match the exact selected capture")
		}
		identity := fmt.Sprintf("%s:%d:%d", capture.Filename, capture.Offset, capture.Length)
		if _, exists := identities[identity]; exists {
			return nil, errors.New("archiveimport: duplicate capture")
		}
		if prior, exists := digests[capture.ProviderDigest]; exists && prior != identity {
			return nil, errors.New("archiveimport: provider digest is ambiguous across captures")
		}
		identities[identity] = struct{}{}
		digests[capture.ProviderDigest] = identity
		captures = append(captures, capture)
	}
	if len(captures) == 0 || uint64(len(captures)) > order.CapturesPerCollection {
		return nil, errors.New("archiveimport: index capture count is outside the work-order budget")
	}
	sort.Slice(captures, func(i, j int) bool {
		if captures[i].Timestamp != captures[j].Timestamp {
			return captures[i].Timestamp < captures[j].Timestamp
		}
		if captures[i].Filename != captures[j].Filename {
			return captures[i].Filename < captures[j].Filename
		}
		return captures[i].Offset < captures[j].Offset
	})
	return captures, nil
}

func (s CaptureSelection) matches(c Capture) bool {
	return s.CollectionID == c.CollectionID && s.Route == c.RequestedURL && s.Timestamp == c.Timestamp && s.ProviderDigest == c.ProviderDigest && s.Filename == c.Filename && s.Offset == c.Offset && s.Length == c.Length
}

func parseIndexLine(line []byte, collectionID, route string) (Capture, error) {
	var raw map[string]json.RawMessage
	if err := jsonbounded.Decode(line, &raw, jsonbounded.Policy{MaxBytes: MaxIndexLine, MaxDepth: 8, MaxScalarBytes: 16 << 10, MaxContainerEntries: 64, MaxTokens: 1024}, false); err != nil || len(raw) == 0 || len(raw) > 32 {
		return Capture{}, errors.New("archiveimport: malformed index JSON")
	}
	required := []string{"url", "timestamp", "status", "mime", "digest", "filename", "offset", "length"}
	for _, key := range required {
		if _, ok := raw[key]; !ok {
			return Capture{}, fmt.Errorf("archiveimport: index record omits %s", key)
		}
	}
	stringField := func(name string) (string, error) {
		var value string
		if err := json.Unmarshal(raw[name], &value); err != nil || value == "" || len(value) > 8192 || strings.ContainsRune(value, '\x00') {
			return "", fmt.Errorf("archiveimport: invalid index field %s", name)
		}
		return value, nil
	}
	capturedURL, err := stringField("url")
	if err != nil || capturedURL != route {
		return Capture{}, errors.New("archiveimport: captured URL does not match the exact reviewed route")
	}
	timestamp, err := stringField("timestamp")
	if err != nil {
		return Capture{}, err
	}
	parsedTime, err := time.Parse("20060102150405", timestamp)
	if err != nil || parsedTime.UTC().Format("20060102150405") != timestamp {
		return Capture{}, errors.New("archiveimport: invalid capture timestamp")
	}
	status, err := uintField(raw["status"])
	if err != nil || status != 200 {
		return Capture{}, errors.New("archiveimport: capture status is not 200")
	}
	mediaType, err := stringField("mime")
	if err != nil || !validPlainText(mediaType, 255) {
		return Capture{}, errors.New("archiveimport: invalid capture media type")
	}
	providerDigest, err := stringField("digest")
	if err != nil || !providerDigestPattern.MatchString(providerDigest) {
		return Capture{}, errors.New("archiveimport: invalid provider payload digest")
	}
	filename, err := stringField("filename")
	if err != nil || !validPlainText(filename, 4096) || !validWARCPath(filename, collectionID) {
		return Capture{}, errors.New("archiveimport: invalid or cross-collection WARC filename")
	}
	offset, err := uintField(raw["offset"])
	if err != nil {
		return Capture{}, errors.New("archiveimport: invalid capture offset")
	}
	length, err := uintField(raw["length"])
	if err != nil || length == 0 || length > MaxCompressed || offset > ^uint64(0)-(length-1) {
		return Capture{}, errors.New("archiveimport: invalid capture length")
	}
	capture := Capture{CollectionID: collectionID, RequestedURL: route, CapturedURL: capturedURL, Timestamp: timestamp, Status: status, MediaType: mediaType, ProviderDigest: providerDigest, Filename: filename, Offset: offset, Length: length, raw: append([]byte(nil), line...)}
	return capture, capture.validateIdentity()
}

func uintField(raw json.RawMessage) (uint64, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return 0, err
	}
	switch typed := value.(type) {
	case string:
		return strconv.ParseUint(typed, 10, 64)
	case json.Number:
		if strings.ContainsAny(string(typed), ".eE+-") {
			return 0, errors.New("not an unsigned integer")
		}
		return strconv.ParseUint(string(typed), 10, 64)
	default:
		return 0, errors.New("not an unsigned integer")
	}
}

func validWARCPath(value, collectionID string) bool {
	if value == "" || len(value) > 4096 || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || path.Clean(value) != value || strings.Contains(value, "..") || !strings.HasSuffix(value, ".warc.gz") {
		return false
	}
	return strings.HasPrefix(value, "crawl-data/"+collectionID+"/")
}

func (c Capture) validateIdentity() error {
	if !collectionPattern.MatchString(c.CollectionID) || c.RequestedURL == "" || c.CapturedURL != c.RequestedURL || c.Status != 200 || c.MediaType == "" || !providerDigestPattern.MatchString(c.ProviderDigest) || !validWARCPath(c.Filename, c.CollectionID) || c.Length == 0 || c.Length > MaxCompressed || c.Offset > ^uint64(0)-(c.Length-1) {
		return errors.New("archiveimport: invalid capture identity")
	}
	route, err := url.Parse(c.RequestedURL)
	if err != nil || route.Scheme != "https" || route.User != nil || route.Opaque != "" || route.Fragment != "" || route.RawPath != "" || route.Hostname() == "" || route.Port() != "" || strings.ToLower(route.Hostname()) != route.Hostname() || route.String() != c.RequestedURL {
		return errors.New("archiveimport: capture route is not an exact canonical HTTPS URL")
	}
	parsedTime, err := time.Parse("20060102150405", c.Timestamp)
	if err != nil || parsedTime.UTC().Format("20060102150405") != c.Timestamp {
		return errors.New("archiveimport: invalid capture timestamp")
	}
	return nil
}

func ValidateRangeResponse(capture Capture, status int, contentRange string, body []byte, maximum uint64) error {
	return validateRangeMetadata(capture, status, contentRange, uint64(len(body)), maximum)
}

func validateRangeMetadata(capture Capture, status int, contentRange string, bodyLength, maximum uint64) error {
	if err := capture.validateIdentity(); err != nil || maximum == 0 || maximum > MaxCompressed || capture.Length > maximum {
		return errors.New("archiveimport: range response has invalid authority or budget")
	}
	if status != 206 || bodyLength != capture.Length {
		return errors.New("archiveimport: bounded range request did not return the exact partial body")
	}
	prefix := fmt.Sprintf("bytes %d-%d/", capture.Offset, capture.Offset+capture.Length-1)
	if !strings.HasPrefix(contentRange, prefix) {
		return errors.New("archiveimport: Content-Range does not match the sealed range")
	}
	totalText := strings.TrimPrefix(contentRange, prefix)
	if totalText == "*" || strings.TrimSpace(totalText) != totalText {
		return errors.New("archiveimport: Content-Range total is unavailable or malformed")
	}
	total, err := strconv.ParseUint(totalText, 10, 64)
	if err != nil || total <= capture.Offset+capture.Length-1 {
		return errors.New("archiveimport: Content-Range total is inconsistent")
	}
	return nil
}
