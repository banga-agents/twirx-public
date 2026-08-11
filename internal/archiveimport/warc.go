package archiveimport

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha1" // Common Crawl provider metadata uses WARC SHA-1 payload digests; TWIRX identity remains SHA-256.
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MaxWARCHeaderBytes    = 64 << 10
	MaxHTTPHeaderBytes    = 64 << 10
	MaxDecompressionRatio = 100
)

type ParsedRecord struct {
	TargetURL       string
	WARCDate        string
	HTTPStatus      uint64
	MediaType       string
	ContentEncoding string
	ContentLanguage string
	Body            []byte
	BodyDigest      string
	ProviderDigest  string
}

func ParseCompressedRecord(compressed []byte, capture Capture, order WorkOrder) (ParsedRecord, error) {
	var result ParsedRecord
	if err := order.Validate(); err != nil || !order.permitsCollection(capture.CollectionID) || !order.permitsRoute(capture.RequestedURL) || capture.Length > order.MaxCompressedRecordBytes || uint64(len(compressed)) != capture.Length {
		return result, errors.New("archiveimport: compressed record is outside the sealed work order")
	}
	source := bytes.NewReader(compressed)
	reader, err := gzip.NewReader(source)
	if err != nil {
		return result, fmt.Errorf("archiveimport: open WARC gzip member: %w", err)
	}
	reader.Multistream(false)
	decompressed, readErr := io.ReadAll(io.LimitReader(reader, int64(order.MaxDecompressedRecordBytes)+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		return result, errors.New("archiveimport: malformed WARC gzip member")
	}
	if uint64(len(decompressed)) > order.MaxDecompressedRecordBytes || len(decompressed) > MaxDecompressed {
		return result, errors.New("archiveimport: decompressed WARC record exceeds its bound")
	}
	if len(decompressed) > len(compressed)*MaxDecompressionRatio {
		return result, errors.New("archiveimport: WARC decompression ratio exceeds its bound")
	}
	if source.Len() != 0 {
		return result, errors.New("archiveimport: trailing or concatenated gzip data")
	}
	return parseWARCRecord(decompressed, capture, order)
}

func parseWARCRecord(data []byte, capture Capture, order WorkOrder) (ParsedRecord, error) {
	var result ParsedRecord
	headerBytes, remainder, err := splitCRLFHeader(data, MaxWARCHeaderBytes)
	if err != nil {
		return result, fmt.Errorf("archiveimport: WARC header: %w", err)
	}
	lines := bytes.Split(headerBytes, []byte("\r\n"))
	if len(lines) < 2 || string(lines[0]) != "WARC/1.0" && string(lines[0]) != "WARC/1.1" {
		return result, errors.New("archiveimport: unsupported WARC version")
	}
	headers, err := parseHeaders(lines[1:])
	if err != nil {
		return result, fmt.Errorf("archiveimport: WARC headers: %w", err)
	}
	required := []string{"warc-type", "warc-target-uri", "warc-date", "content-type", "content-length", "warc-payload-digest"}
	for _, name := range required {
		if headers[name] == "" {
			return result, fmt.Errorf("archiveimport: WARC omits %s", name)
		}
	}
	if headers["warc-type"] != "response" || headers["warc-target-uri"] != capture.CapturedURL || headers["content-type"] != "application/http; msgtype=response" {
		return result, errors.New("archiveimport: WARC identity or record type mismatch")
	}
	warcTime, err := time.Parse(time.RFC3339, headers["warc-date"])
	if err != nil || warcTime.UTC().Format("20060102150405") != capture.Timestamp {
		return result, errors.New("archiveimport: WARC date does not match the capture timestamp")
	}
	contentLength, err := strconv.ParseUint(headers["content-length"], 10, 64)
	if err != nil || contentLength > order.MaxDecompressedRecordBytes || uint64(len(remainder)) != contentLength+4 || !bytes.HasSuffix(remainder, []byte("\r\n\r\n")) {
		return result, errors.New("archiveimport: WARC content length or terminator mismatch")
	}
	block := remainder[:contentLength]
	if len(block) == 0 || len(block) > MaxDecompressed {
		return result, errors.New("archiveimport: empty or oversized WARC HTTP block")
	}
	if !providerDigestEqual(headers["warc-payload-digest"], capture.ProviderDigest) {
		return result, errors.New("archiveimport: WARC and index provider digests disagree")
	}
	return parseHTTPBlock(block, capture, order, headers["warc-payload-digest"])
}

func parseHTTPBlock(block []byte, capture Capture, order WorkOrder, providerDigest string) (ParsedRecord, error) {
	var result ParsedRecord
	headerEnd := bytes.Index(block, []byte("\r\n\r\n"))
	if headerEnd < 0 || headerEnd+4 > MaxHTTPHeaderBytes || containsBareLF(block[:headerEnd+4]) {
		return result, errors.New("archiveimport: malformed or oversized archived HTTP headers")
	}
	headerLines := bytes.Split(block[:headerEnd], []byte("\r\n"))
	if len(headerLines) < 1 || !bytes.HasPrefix(headerLines[0], []byte("HTTP/")) || bytes.ContainsAny(headerLines[0], "\x00\r\n") {
		return result, errors.New("archiveimport: malformed archived HTTP status line")
	}
	if err := validateHTTPHeaderLines(headerLines[1:]); err != nil {
		return result, fmt.Errorf("archiveimport: archived HTTP headers: %w", err)
	}
	source := bytes.NewReader(block)
	buffered := bufio.NewReader(source)
	request, _ := http.NewRequest(http.MethodGet, capture.CapturedURL, nil)
	response, err := http.ReadResponse(buffered, request)
	if err != nil {
		return result, fmt.Errorf("archiveimport: parse archived HTTP response: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != int(capture.Status) || response.StatusCode != http.StatusOK {
		return result, errors.New("archiveimport: archived HTTP status mismatch")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(order.MaxRetainedBodyBytes)+1))
	if err != nil || len(body) == 0 || uint64(len(body)) > order.MaxRetainedBodyBytes || len(body) > MaxRetainedBody {
		return result, errors.New("archiveimport: retained representation exceeds its bound")
	}
	if buffered.Buffered() != 0 || source.Len() != 0 {
		return result, errors.New("archiveimport: trailing archived HTTP data")
	}
	mediaType := "application/octet-stream"
	if value := response.Header.Get("Content-Type"); value != "" {
		parsed, _, err := mime.ParseMediaType(value)
		if err != nil || len(parsed) > 255 {
			return result, errors.New("archiveimport: invalid archived content type")
		}
		mediaType = strings.ToLower(parsed)
	}
	if capture.MediaType != "unk" && capture.MediaType != mediaType {
		return result, errors.New("archiveimport: index and archived media types disagree")
	}
	computedProvider := sha1.Sum(body)
	encodedProvider := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(computedProvider[:])
	if !providerDigestEqual(encodedProvider, providerDigest) {
		return result, errors.New("archiveimport: archived payload does not match the provider digest")
	}
	for _, name := range []string{"Content-Encoding", "Content-Language"} {
		if len(response.Header.Values(name)) > 1 || len(response.Header.Get(name)) > 1024 || strings.ContainsAny(response.Header.Get(name), "\r\n\x00") {
			return result, errors.New("archiveimport: invalid selected archived response header")
		}
	}
	return ParsedRecord{TargetURL: capture.CapturedURL, WARCDate: headersTime(capture.Timestamp), HTTPStatus: uint64(response.StatusCode), MediaType: mediaType, ContentEncoding: response.Header.Get("Content-Encoding"), ContentLanguage: response.Header.Get("Content-Language"), Body: body, BodyDigest: digest(body), ProviderDigest: normalizeProviderDigest(providerDigest)}, nil
}

func splitCRLFHeader(data []byte, maximum int) ([]byte, []byte, error) {
	index := bytes.Index(data, []byte("\r\n\r\n"))
	if index < 0 || index+4 > maximum || containsBareLF(data[:index+4]) {
		return nil, nil, errors.New("missing canonical CRLF header terminator")
	}
	return data[:index], data[index+4:], nil
}

func containsBareLF(data []byte) bool {
	for index, value := range data {
		if value == '\n' && (index == 0 || data[index-1] != '\r') {
			return true
		}
	}
	return false
}

func parseHeaders(lines [][]byte) (map[string]string, error) {
	result := make(map[string]string, len(lines))
	repeatable := map[string]bool{"warc-concurrent-to": true, "warc-protocol": true}
	for _, line := range lines {
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' || bytes.ContainsAny(line, "\x00\r\n") {
			return nil, errors.New("invalid or folded header")
		}
		index := bytes.IndexByte(line, ':')
		if index <= 0 {
			return nil, errors.New("header lacks a name/value delimiter")
		}
		name := strings.ToLower(string(line[:index]))
		value := strings.TrimSpace(string(line[index+1:]))
		if name == "" || value == "" || len(name) > 128 || len(value) > 8192 {
			return nil, errors.New("empty or oversized header")
		}
		for _, character := range name {
			if character != '-' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
				return nil, errors.New("invalid header name")
			}
		}
		if result[name] != "" {
			if !repeatable[name] {
				return nil, errors.New("duplicate non-repeatable header")
			}
			continue
		}
		result[name] = value
	}
	return result, nil
}

func validateHTTPHeaderLines(lines [][]byte) error {
	critical := map[string]bool{
		"content-encoding":  true,
		"content-language":  true,
		"content-length":    true,
		"content-type":      true,
		"transfer-encoding": true,
	}
	seen := make(map[string]uint64)
	for _, line := range lines {
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' || bytes.ContainsAny(line, "\x00\r\n") {
			return errors.New("invalid or folded header")
		}
		index := bytes.IndexByte(line, ':')
		if index <= 0 {
			return errors.New("header lacks a name/value delimiter")
		}
		name := strings.ToLower(string(line[:index]))
		if len(name) > 128 || len(line[index+1:]) > 8192 {
			return errors.New("oversized header")
		}
		for _, character := range name {
			if character != '-' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
				return errors.New("invalid header name")
			}
		}
		seen[name]++
		if critical[name] && seen[name] > 1 {
			return errors.New("duplicate security-relevant header")
		}
	}
	return nil
}

func normalizeProviderDigest(value string) string {
	return strings.TrimPrefix(strings.ToUpper(value), "SHA1:")
}

func providerDigestEqual(left, right string) bool {
	return normalizeProviderDigest(left) == normalizeProviderDigest(right)
}

func headersTime(timestamp string) string {
	parsed, _ := time.Parse("20060102150405", timestamp)
	return parsed.UTC().Format(time.RFC3339)
}
