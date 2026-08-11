// Package safefetch retrieves public HTTP resources through an explicit,
// deny-by-default network policy. It is intentionally small for Genesis and
// is not yet a substitute for production egress isolation.
package safefetch

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultPolicyID = "tw.fetch.public-v0"

var deniedPrefixes = mustPrefixes(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"64:ff9b:1::/48",
	"100::/64",
	"2001:2::/48",
	"2001:db8::/32",
	"2001:10::/28",
	"3fff::/20",
)

var byteRangePattern = regexp.MustCompile(`^bytes=[0-9]{1,20}-[0-9]{1,20}$`)

func mustPrefixes(values ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		out = append(out, netip.MustParsePrefix(value))
	}
	return out
}

// Policy controls outbound retrieval. Loopback and non-standard ports are
// disabled by default and may only be enabled explicitly for local fixtures.
type Policy struct {
	ID                    string
	AllowLoopback         bool
	AllowNonStandardPorts bool
	MaxRedirects          int
	MaxBodyBytes          int64
	RequestTimeout        time.Duration
	ConnectTimeout        time.Duration
	ResponseHeaderTimeout time.Duration
	UserAgent             string
	// AllowedHosts, when non-empty, constrains the initial URL and every
	// redirect to an explicitly admitted hostname.
	AllowedHosts []string
}

func DefaultPolicy() Policy {
	return Policy{
		ID:                    DefaultPolicyID,
		AllowLoopback:         false,
		AllowNonStandardPorts: false,
		MaxRedirects:          5,
		MaxBodyBytes:          2 << 20,
		RequestTimeout:        20 * time.Second,
		ConnectTimeout:        5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		UserAgent:             "TypedWebObserver/0.1 (+https://typedweb.org)",
	}
}

type Result struct {
	RequestURL     string
	FinalURL       string
	Method         string
	Status         int
	MediaType      string
	RetrievedAt    time.Time
	Body           []byte
	Redirects      []Redirect
	Headers        []Header
	RequestedRange string
	ContentRange   string
}

// Redirect and Header are immutable transport evidence inputs. Header names
// are lower-case and restricted to representation-relevant fields; cookies,
// authorization, and arbitrary response metadata are never recorded.
type Redirect struct {
	FromURL string
	Status  int
	ToURL   string
}

type Header struct {
	Name  string
	Value string
}

var recordedResponseHeaders = []string{
	"Content-Encoding", "Content-Language", "Content-Location", "Content-Type",
}

type Fetcher struct {
	policy       Policy
	resolver     netResolver
	allowedHosts []string
}

type netResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type contextDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

func New(policy Policy) (*Fetcher, error) {
	if policy.ID == "" {
		return nil, errors.New("safefetch: policy ID is required")
	}
	if policy.MaxRedirects < 0 || policy.MaxRedirects > 20 {
		return nil, errors.New("safefetch: redirect limit must be between 0 and 20")
	}
	if policy.MaxBodyBytes <= 0 {
		return nil, errors.New("safefetch: body limit must be positive")
	}
	if policy.RequestTimeout <= 0 || policy.ConnectTimeout <= 0 || policy.ResponseHeaderTimeout <= 0 {
		return nil, errors.New("safefetch: timeouts must be positive")
	}
	if len(policy.AllowedHosts) > 32 {
		return nil, errors.New("safefetch: too many allowed hosts")
	}
	allowedHosts := make([]string, 0, len(policy.AllowedHosts))
	for _, host := range policy.AllowedHosts {
		if host == "" || len(host) > 253 || host != strings.ToLower(host) || strings.ContainsAny(host, "/@[]:") {
			return nil, errors.New("safefetch: invalid allowed host")
		}
		allowedHosts = append(allowedHosts, host)
	}
	return &Fetcher{policy: policy, resolver: net.DefaultResolver, allowedHosts: allowedHosts}, nil
}

func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*Result, error) {
	return f.fetch(ctx, rawURL, "")
}

// FetchRange retrieves one exact HTTP byte range through the same DNS,
// address, redirect, TLS and response bounds as Fetch. It exists for sealed
// content-addressed archive work orders; callers remain responsible for
// checking the exact 206 and Content-Range semantics against their authority.
func (f *Fetcher) FetchRange(ctx context.Context, rawURL, requestedRange string) (*Result, error) {
	if !byteRangePattern.MatchString(requestedRange) {
		return nil, errors.New("safefetch: byte range must use exact bytes=START-END form")
	}
	parts := strings.Split(strings.TrimPrefix(requestedRange, "bytes="), "-")
	start, startErr := strconv.ParseUint(parts[0], 10, 64)
	end, endErr := strconv.ParseUint(parts[1], 10, 64)
	if startErr != nil || endErr != nil || end < start || end-start >= uint64(f.policy.MaxBodyBytes) {
		return nil, errors.New("safefetch: byte range is malformed or exceeds the response bound")
	}
	return f.fetch(ctx, rawURL, requestedRange)
}

func (f *Fetcher) fetch(ctx context.Context, rawURL, requestedRange string) (*Result, error) {
	parsed, err := f.validateURL(rawURL)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: f.policy.ConnectTimeout, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           f.safeDialContext(dialer),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: f.policy.ResponseHeaderTimeout,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
	defer transport.CloseIdleConnections()

	redirects := make([]Redirect, 0, f.policy.MaxRedirects)
	client := &http.Client{
		Transport: transport,
		Timeout:   f.policy.RequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > f.policy.MaxRedirects {
				return fmt.Errorf("safefetch: redirect limit %d exceeded", f.policy.MaxRedirects)
			}
			if _, err := f.validateURL(req.URL.String()); err != nil {
				return fmt.Errorf("safefetch: redirect rejected: %w", err)
			}
			if req.Response == nil || req.Response.Request == nil {
				return errors.New("safefetch: redirect response metadata is unavailable")
			}
			redirects = append(redirects, Redirect{
				FromURL: req.Response.Request.URL.String(), Status: req.Response.StatusCode,
				ToURL: req.URL.String(),
			})
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("safefetch: construct request: %w", err)
	}
	if requestedRange == "" {
		req.Header.Set("Accept", "application/json, application/ld+json, text/html;q=0.9, */*;q=0.1")
	} else {
		req.Header.Set("Accept", "application/octet-stream")
		req.Header.Set("Accept-Encoding", "identity")
		req.Header.Set("Range", requestedRange)
	}
	req.Header.Set("User-Agent", f.policy.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("safefetch: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, f.policy.MaxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("safefetch: read response: %w", err)
	}
	if int64(len(body)) > f.policy.MaxBodyBytes {
		return nil, fmt.Errorf("safefetch: response exceeds %d-byte decompressed limit", f.policy.MaxBodyBytes)
	}

	mediaType := "application/octet-stream"
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		parsedType, _, parseErr := mime.ParseMediaType(contentType)
		if parseErr != nil {
			return nil, fmt.Errorf("safefetch: invalid content type %q: %w", contentType, parseErr)
		}
		mediaType = strings.ToLower(parsedType)
	}

	return &Result{
		RequestURL:     parsed.String(),
		FinalURL:       resp.Request.URL.String(),
		Method:         http.MethodGet,
		Status:         resp.StatusCode,
		MediaType:      mediaType,
		RetrievedAt:    time.Now().UTC(),
		Body:           body,
		Redirects:      redirects,
		Headers:        selectedHeaders(resp.Header),
		RequestedRange: requestedRange,
		ContentRange:   resp.Header.Get("Content-Range"),
	}, nil
}

func selectedHeaders(headers http.Header) []Header {
	selected := make([]Header, 0, len(recordedResponseHeaders))
	for _, name := range recordedResponseHeaders {
		values := headers.Values(name)
		for _, value := range values {
			selected = append(selected, Header{Name: strings.ToLower(name), Value: value})
		}
	}
	sort.Slice(selected, func(i, j int) bool {
		if selected[i].Name == selected[j].Name {
			return selected[i].Value < selected[j].Value
		}
		return selected[i].Name < selected[j].Name
	})
	return selected
}

func (f *Fetcher) validateURL(raw string) (*url.URL, error) {
	if len(raw) == 0 || len(raw) > 4096 {
		return nil, errors.New("safefetch: URL is empty or too long")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("safefetch: parse URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("safefetch: scheme %q is not allowed", u.Scheme)
	}
	if u.User != nil {
		return nil, errors.New("safefetch: embedded credentials are not allowed")
	}
	if u.Hostname() == "" {
		return nil, errors.New("safefetch: hostname is required")
	}
	if len(f.allowedHosts) > 0 && !hostAllowed(f.allowedHosts, u.Hostname()) {
		return nil, fmt.Errorf("safefetch: hostname %q is not admitted", u.Hostname())
	}
	if u.Fragment != "" {
		u.Fragment = ""
	}
	if !f.policy.AllowNonStandardPorts {
		port := u.Port()
		if (u.Scheme == "http" && port != "" && port != "80") || (u.Scheme == "https" && port != "" && port != "443") {
			return nil, fmt.Errorf("safefetch: non-standard port %q is not allowed", port)
		}
	}
	return u, nil
}

func hostAllowed(allowed []string, candidate string) bool {
	for _, host := range allowed {
		if strings.EqualFold(host, candidate) {
			return true
		}
	}
	return false
}

func (f *Fetcher) safeDialContext(dialer contextDialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("safefetch: split dial address: %w", err)
		}
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return nil, fmt.Errorf("safefetch: invalid port: %w", err)
		}
		ips, err := f.resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("safefetch: resolve %q: %w", host, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("safefetch: no addresses for %q", host)
		}
		validated := make([]netip.Addr, 0, len(ips))
		var rejected []string
		for _, ip := range ips {
			addr := ip.Unmap()
			if err := f.validateIP(addr); err != nil {
				rejected = append(rejected, addr.String())
				continue
			}
			validated = append(validated, addr)
		}
		if len(rejected) > 0 {
			return nil, fmt.Errorf("safefetch: DNS answer contains denied addresses=%s", strings.Join(rejected, ","))
		}
		for _, addr := range validated {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
			if err == nil {
				return conn, nil
			}
		}
		return nil, fmt.Errorf("safefetch: no resolved address for %q was reachable", host)
	}
}

func (f *Fetcher) validateIP(addr netip.Addr) error {
	if !addr.IsValid() {
		return errors.New("invalid address")
	}
	addr = addr.Unmap()
	if addr.IsLoopback() {
		if f.policy.AllowLoopback {
			return nil
		}
		return errors.New("loopback address denied")
	}
	if addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return errors.New("non-public address denied")
	}
	for _, prefix := range deniedPrefixes {
		if prefix.Contains(addr) {
			return fmt.Errorf("address in denied range %s", prefix)
		}
	}
	if !addr.IsGlobalUnicast() {
		return errors.New("address is not global unicast")
	}
	return nil
}
