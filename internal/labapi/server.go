// Package labapi exposes the bounded E2 Live Provenance Lab over HTTP. It has
// no arbitrary-URL route, no write operation, and no public remote MCP route.
package labapi

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/labengine"
	"github.com/typed-web-commons/typed-web/internal/origincatalog"
	"github.com/typed-web-commons/typed-web/internal/proofbundle"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

const securityPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self' mailto:; object-src 'none'"

type Config struct {
	Engine            *labengine.Engine
	StaticDir         string
	MaxRequestBytes   int
	MaxOutputBytes    int
	InvocationTimeout time.Duration
	PerIPPerMinute    int
	PerIPBurst        int
	GlobalConcurrency int
	AuditWriter       io.Writer
}

type Server struct {
	config    Config
	perIP     *limiter
	perOrigin *limiter
	global    chan struct{}
	audit     *log.Logger
	salt      [32]byte
	now       func() time.Time
}

func New(config Config) (*Server, error) {
	if config.Engine == nil {
		return nil, errors.New("labapi: engine is required")
	}
	if config.MaxRequestBytes == 0 {
		config.MaxRequestBytes = 64 << 10
	}
	if config.MaxOutputBytes == 0 {
		config.MaxOutputBytes = 2 << 20
	}
	if config.InvocationTimeout == 0 {
		config.InvocationTimeout = 25 * time.Second
	}
	if config.PerIPPerMinute == 0 {
		config.PerIPPerMinute = 60
	}
	if config.PerIPBurst == 0 {
		config.PerIPBurst = 20
	}
	if config.GlobalConcurrency == 0 {
		config.GlobalConcurrency = 8
	}
	if config.MaxRequestBytes < 1 || config.MaxRequestBytes > 1<<20 || config.MaxOutputBytes < 1 || config.MaxOutputBytes > 8<<20 || config.InvocationTimeout < time.Second || config.InvocationTimeout > time.Minute || config.PerIPPerMinute < 1 || config.PerIPBurst < 1 || config.GlobalConcurrency < 1 || config.GlobalConcurrency > 128 {
		return nil, errors.New("labapi: invalid resource limits")
	}
	writer := config.AuditWriter
	if writer == nil {
		writer = io.Discard
	}
	server := &Server{config: config, perIP: newLimiter(10000), perOrigin: newLimiter(256), global: make(chan struct{}, config.GlobalConcurrency), audit: log.New(writer, "twirx-lab ", 0), now: time.Now}
	if _, err := rand.Read(server.salt[:]); err != nil {
		return nil, fmt.Errorf("labapi: audit salt: %w", err)
	}
	return server, nil
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	started := s.now()
	setSecurityHeaders(writer)
	if strings.HasPrefix(request.URL.Path, "/api/") || request.URL.Path == "/.well-known/twirx" {
		client := s.clientKey(request)
		if !s.perIP.allow(client, float64(s.config.PerIPPerMinute), float64(s.config.PerIPBurst), started) {
			writer.Header().Set("Retry-After", "60")
			s.writeError(writer, http.StatusTooManyRequests, "rate_limited", "Per-client request limit reached.")
			s.auditEvent(request, client, "rate_limited", started)
			return
		}
		s.serveAPI(writer, request, client, started)
		return
	}
	s.serveStatic(writer, request)
}

func (s *Server) serveAPI(writer http.ResponseWriter, request *http.Request, client string, started time.Time) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.writeError(writer, http.StatusInternalServerError, "internal_error", "Request failed safely.")
			s.audit.Printf("client=%s method=%s path=%s outcome=panic duration_ms=%d", s.auditKey(client), request.Method, request.URL.Path, s.now().Sub(started).Milliseconds())
		}
	}()
	path := request.URL.Path
	switch {
	case path == "/.well-known/twirx":
		s.requireMethod(writer, request, http.MethodGet, func() {
			s.writeJSON(writer, http.StatusOK, map[string]any{"format": "tw.well-known/0.1", "release_label": "Genesis Preview", "interface": "catalog-only read operations", "execution_mode": s.executionMode(), "status": "/api/v1/status", "origins": "/api/v1/origins", "invoke": "/api/v1/invoke", "openapi": "/openapi.json", "arbitrary_urls": false})
		})
	case path == "/api/v1/status":
		s.requireMethod(writer, request, http.MethodGet, func() {
			s.writeJSON(writer, http.StatusOK, map[string]any{"format": "tw.lab-status/0.1", "release_label": "Genesis Preview", "engineering_gate": "E2", "gate_status": "implementation_candidate", "read_only": true, "catalog_only": true, "execution_mode": s.executionMode(), "fresh_origin_access": false, "arbitrary_url_input": false, "origins": len(s.config.Engine.Catalog.Origins), "operations": len(s.config.Engine.Contracts.Operations), "result_format": "tw.result/0.2"})
		})
	case path == "/api/v1/origins":
		s.requireMethod(writer, request, http.MethodGet, func() { s.writeJSON(writer, http.StatusOK, map[string]any{"origins": s.originViews()}) })
	case strings.HasPrefix(path, "/api/v1/origins/"):
		s.serveOrigin(writer, request, strings.TrimPrefix(path, "/api/v1/origins/"))
	case path == "/api/v1/invoke":
		s.requireMethod(writer, request, http.MethodPost, func() { s.invoke(writer, request, client, started) })
	case strings.HasPrefix(path, "/api/v1/results/"):
		s.serveResult(writer, request, strings.TrimPrefix(path, "/api/v1/results/"))
	default:
		s.writeError(writer, http.StatusNotFound, "not_found", "No such Lab endpoint.")
	}
}

func (s *Server) requireMethod(writer http.ResponseWriter, request *http.Request, method string, next func()) {
	if request.Method != method {
		writer.Header().Set("Allow", method)
		s.writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Method is not allowed for this endpoint.")
		return
	}
	next()
}

type originView struct {
	ID              string   `json:"id"`
	Version         string   `json:"version"`
	Title           string   `json:"title"`
	Publisher       string   `json:"publisher"`
	SourceClass     string   `json:"source_class"`
	AdmissionStatus string   `json:"admission_status"`
	FreshEnabled    bool     `json:"fresh_enabled"`
	HealthState     string   `json:"health_state"`
	Operations      []string `json:"operations"`
	Attribution     string   `json:"attribution"`
	TermsReference  string   `json:"terms_reference"`
}

func viewOrigin(origin *origincatalog.Origin) originView {
	return originView{ID: origin.ID, Version: origin.Version, Title: origin.Title, Publisher: origin.Publisher, SourceClass: origin.SourceClass, AdmissionStatus: origin.AdmissionStatus, FreshEnabled: origin.FreshEnabled, HealthState: "not_probed", Operations: append([]string(nil), origin.Operations...), Attribution: origin.Attribution, TermsReference: origin.TermsReference}
}
func (s *Server) originViews() []originView {
	views := make([]originView, 0, len(s.config.Engine.Catalog.Origins))
	for i := range s.config.Engine.Catalog.Origins {
		view := viewOrigin(&s.config.Engine.Catalog.Origins[i])
		view.FreshEnabled = false
		views = append(views, view)
	}
	return views
}

func (s *Server) serveOrigin(writer http.ResponseWriter, request *http.Request, rest string) {
	s.requireMethod(writer, request, http.MethodGet, func() {
		parts := strings.Split(rest, "/")
		if len(parts) < 1 || len(parts) > 2 || parts[0] == "" {
			s.writeError(writer, http.StatusNotFound, "not_found", "Origin endpoint not found.")
			return
		}
		origin, err := s.config.Engine.Catalog.Find(parts[0])
		if err != nil {
			s.writeError(writer, http.StatusNotFound, "unknown_origin", "Origin is not in the admitted catalog.")
			return
		}
		if len(parts) == 1 {
			view := viewOrigin(origin)
			view.FreshEnabled = false
			s.writeJSON(writer, http.StatusOK, view)
			return
		}
		switch parts[1] {
		case "operations":
			operations := make([]any, 0, len(origin.Operations))
			for _, id := range origin.Operations {
				op, _ := s.config.Engine.Contracts.Find(id)
				operations = append(operations, operationView(op))
			}
			s.writeJSON(writer, http.StatusOK, map[string]any{"origin_id": origin.ID, "operations": operations})
		case "schema":
			schemas := make(map[string]any, len(origin.Operations))
			for _, id := range origin.Operations {
				op, _ := s.config.Engine.Contracts.Find(id)
				raw, schemaErr := twircontract.JSONSchema(op)
				if schemaErr != nil {
					s.writeError(writer, http.StatusInternalServerError, "binding_error", "Schema generation failed.")
					return
				}
				var schema any
				if json.Unmarshal(raw, &schema) != nil {
					s.writeError(writer, http.StatusInternalServerError, "binding_error", "Schema generation failed.")
					return
				}
				schemas[id] = schema
			}
			s.writeJSON(writer, http.StatusOK, map[string]any{"origin_id": origin.ID, "schemas": schemas})
		default:
			s.writeError(writer, http.StatusNotFound, "not_found", "Origin endpoint not found.")
		}
	})
}

func operationView(op *twircontract.Operation) map[string]any {
	return map[string]any{"id": op.ID, "version": op.Version, "title": op.Title, "description": op.Description, "resource": op.Resource, "effect": op.Effect, "evidence_requirement": op.EvidenceRequirement}
}

type invokePayload struct {
	OriginID    string         `json:"origin_id"`
	OperationID string         `json:"operation_id"`
	Input       map[string]any `json:"input"`
	Mode        string         `json:"mode"`
}

func (s *Server) invoke(writer http.ResponseWriter, request *http.Request, client string, started time.Time) {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		s.writeError(writer, http.StatusUnsupportedMediaType, "content_type", "Content-Type must be application/json.")
		return
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, int64(s.config.MaxRequestBytes)+1))
	if err != nil || len(body) > s.config.MaxRequestBytes {
		s.writeError(writer, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds the Lab limit.")
		return
	}
	var payload invokePayload
	policy := jsonbounded.Policy{MaxBytes: s.config.MaxRequestBytes, MaxDepth: 12, MaxScalarBytes: 64 << 10, MaxContainerEntries: 256, MaxTokens: 4096}
	if err := jsonbounded.Decode(body, &payload, policy, true); err != nil {
		s.writeError(writer, http.StatusBadRequest, "invalid_request", "Request JSON is malformed, ambiguous, or outside bounds.")
		return
	}
	mode := payload.Mode
	if mode == "" {
		mode = labengine.ModeReplay
	}
	if mode == labengine.ModeFresh {
		s.writeError(writer, http.StatusBadRequest, "fresh_mode_disabled", "Fresh-origin execution is disabled on this Lab surface; use admitted replay evidence.")
		return
	}
	if mode != labengine.ModeReplay {
		s.writeError(writer, http.StatusBadRequest, "invalid_mode", "Mode must be replay on this Lab surface.")
		return
	}
	op, err := s.config.Engine.Contracts.Find(payload.OperationID)
	if err != nil {
		s.writeError(writer, http.StatusBadRequest, "unknown_operation", "Operation is not admitted.")
		return
	}
	input, err := twircontract.NormalizeInput(op, payload.Input)
	if err != nil {
		s.writeError(writer, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	origin, err := s.config.Engine.Catalog.Find(payload.OriginID)
	if err != nil || origin.ID != op.OriginID {
		s.writeError(writer, http.StatusBadRequest, "origin_operation_mismatch", "Operation does not belong to the requested origin.")
		return
	}
	if !s.perOrigin.allow(origin.ID, float64(origin.RequestsPerMinute), float64(origin.RequestsPerMinute), started) {
		writer.Header().Set("Retry-After", "60")
		s.writeError(writer, http.StatusTooManyRequests, "origin_rate_limited", "Origin quota reached.")
		return
	}
	select {
	case s.global <- struct{}{}:
		defer func() { <-s.global }()
	default:
		writer.Header().Set("Retry-After", "1")
		s.writeError(writer, http.StatusTooManyRequests, "concurrency_limited", "Lab concurrency limit reached.")
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), s.config.InvocationTimeout)
	defer cancel()
	invocation, err := s.config.Engine.Invoke(ctx, labengine.Request{OriginID: origin.ID, OperationID: op.ID, Input: input, Mode: mode})
	if err != nil {
		status := http.StatusBadGateway
		if strings.Contains(err.Error(), "mode must") || strings.Contains(err.Error(), "replay fixture") {
			status = http.StatusBadRequest
		}
		s.writeError(writer, status, "invocation_failed", boundedMessage(err.Error()))
		s.auditEvent(request, client, "invoke_rejected", started)
		return
	}
	writer.Header().Set("Location", "/api/v1/results/"+invocation.Publication.ResultID)
	s.writeJSON(writer, http.StatusOK, labengine.View(invocation))
	s.auditEvent(request, client, "invoke_ok", started)
}

func (s *Server) executionMode() string {
	return "replay_only"
}

func (s *Server) serveResult(writer http.ResponseWriter, request *http.Request, rest string) {
	s.requireMethod(writer, request, http.MethodGet, func() {
		parts := strings.Split(rest, "/")
		if len(parts) < 1 || len(parts) > 2 || parts[0] == "" {
			s.writeError(writer, http.StatusNotFound, "not_found", "Result endpoint not found.")
			return
		}
		view, dir, err := s.config.Engine.Load(parts[0])
		if err != nil {
			s.writeError(writer, http.StatusNotFound, "unknown_result", "Result is unavailable or failed verification.")
			return
		}
		writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if len(parts) == 1 {
			s.writeJSON(writer, http.StatusOK, view)
			return
		}
		switch parts[1] {
		case "provenance":
			s.writeJSON(writer, http.StatusOK, map[string]any{"result_id": view.ResultID, "bindings": view.Bindings, "fields": view.Fields, "statement": "Represents an observed origin and declared derivation; does not assert objective truth."})
		case "bundle":
			s.writeBundle(writer, dir, view.ResultID)
		default:
			s.writeError(writer, http.StatusNotFound, "not_found", "Result endpoint not found.")
		}
	})
}

func (s *Server) writeBundle(writer http.ResponseWriter, dir, resultID string) {
	manifestBytes, err := os.ReadFile(filepath.Join(dir, proofbundle.ManifestName))
	if err != nil {
		s.writeError(writer, http.StatusNotFound, "bundle_unavailable", "Proof bundle is unavailable.")
		return
	}
	manifest, err := proofbundle.UnmarshalManifest(manifestBytes)
	if err != nil {
		s.writeError(writer, http.StatusNotFound, "bundle_unavailable", "Proof bundle failed verification.")
		return
	}
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	names := append([]proofbundle.Entry(nil), manifest.Entries...)
	names = append(names, proofbundle.Entry{Name: proofbundle.ManifestName, Size: uint64(len(manifestBytes))})
	for _, entry := range names {
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name))
		if readErr != nil || len(data) > proofbundle.MaxArtifactBytes {
			s.writeError(writer, http.StatusNotFound, "bundle_unavailable", "Proof bundle is unavailable.")
			return
		}
		if entry.Name != proofbundle.ManifestName && (uint64(len(data)) != entry.Size || sha256.Sum256(data) != entry.Digest) {
			s.writeError(writer, http.StatusNotFound, "bundle_unavailable", "Proof bundle failed verification.")
			return
		}
		header := &zip.FileHeader{Name: entry.Name, Method: zip.Store}
		header.SetModTime(time.Unix(0, 0).UTC())
		file, createErr := archive.CreateHeader(header)
		if createErr != nil {
			s.writeError(writer, http.StatusInternalServerError, "bundle_error", "Bundle archive failed.")
			return
		}
		if _, writeErr := file.Write(data); writeErr != nil {
			s.writeError(writer, http.StatusInternalServerError, "bundle_error", "Bundle archive failed.")
			return
		}
	}
	if err := archive.Close(); err != nil {
		s.writeError(writer, http.StatusInternalServerError, "bundle_error", "Bundle archive failed.")
		return
	}
	if output.Len() > s.config.MaxOutputBytes*8 {
		s.writeError(writer, http.StatusInternalServerError, "bundle_error", "Bundle archive exceeds response limit.")
		return
	}
	writer.Header().Set("Content-Type", "application/zip")
	writer.Header().Set("Content-Disposition", `attachment; filename="twirx-`+strings.TrimPrefix(resultID, "sha256:")+`.zip"`)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(output.Bytes())
}

func (s *Server) serveStatic(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		s.writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Static resources are read-only.")
		return
	}
	name := ""
	contentType := ""
	switch request.URL.Path {
	case "/", "/index.html":
		name = "index.html"
		contentType = "text/html; charset=utf-8"
	case "/submit-origin":
		name = "submit-origin.html"
		contentType = "text/html; charset=utf-8"
	case "/assets/styles.css":
		name = "styles.css"
		contentType = "text/css; charset=utf-8"
	case "/assets/app.js":
		name = "app.js"
		contentType = "text/javascript; charset=utf-8"
	case "/openapi.json":
		name = filepath.Join("..", "generated", "e2", "openapi.json")
		contentType = "application/json"
	default:
		s.writeError(writer, http.StatusNotFound, "not_found", "Resource not found.")
		return
	}
	var path string
	if strings.HasPrefix(name, "..") {
		path = filepath.Join(s.config.Engine.Root, "generated", "e2", "openapi.json")
	} else {
		path = filepath.Join(s.config.StaticDir, name)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		s.writeError(writer, http.StatusNotFound, "not_found", "Resource not found.")
		return
	}
	writer.Header().Set("Content-Type", contentType)
	if strings.HasPrefix(request.URL.Path, "/assets/") {
		writer.Header().Set("Cache-Control", "public, max-age=3600")
	} else {
		writer.Header().Set("Cache-Control", "no-cache")
	}
	writer.Header().Set("Content-Length", fmt.Sprint(len(data)))
	writer.WriteHeader(http.StatusOK)
	if request.Method == http.MethodGet {
		_, _ = writer.Write(data)
	}
}

func (s *Server) writeJSON(writer http.ResponseWriter, status int, value any) {
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) > s.config.MaxOutputBytes {
		s.writeError(writer, http.StatusInternalServerError, "response_error", "Response encoding failed safely.")
		return
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_, _ = writer.Write(append(encoded, '\n'))
}
func (s *Server) writeError(writer http.ResponseWriter, status int, code, message string) {
	encoded, _ := json.Marshal(map[string]any{"error": map[string]string{"code": code, "message": boundedMessage(message)}})
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_, _ = writer.Write(append(encoded, '\n'))
}
func boundedMessage(value string) string {
	if len(value) > 512 {
		return value[:512]
	}
	return value
}

func setSecurityHeaders(writer http.ResponseWriter) {
	writer.Header().Set("Content-Security-Policy", securityPolicy)
	writer.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	writer.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	writer.Header().Set("Referrer-Policy", "no-referrer")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("X-Frame-Options", "DENY")
}

func (s *Server) clientKey(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return "invalid"
	}
	if address.IsLoopback() {
		forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-For"), ",")[0])
		if candidate, parseErr := netip.ParseAddr(forwarded); parseErr == nil {
			return candidate.String()
		}
	}
	return address.String()
}
func (s *Server) auditKey(client string) string {
	hash := sha256.New()
	_, _ = hash.Write(s.salt[:])
	_, _ = hash.Write([]byte(client))
	return hex.EncodeToString(hash.Sum(nil)[:8])
}
func (s *Server) auditEvent(request *http.Request, client, outcome string, started time.Time) {
	s.audit.Printf("client=%s method=%s path=%s outcome=%s duration_ms=%d", s.auditKey(client), request.Method, request.URL.Path, outcome, s.now().Sub(started).Milliseconds())
}

type bucket struct {
	tokens float64
	last   time.Time
}
type limiter struct {
	mu      sync.Mutex
	buckets map[string]bucket
	maximum int
}

func newLimiter(maximum int) *limiter {
	return &limiter{buckets: make(map[string]bucket), maximum: maximum}
}
func (l *limiter) allow(key string, perMinute, burst float64, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	current, exists := l.buckets[key]
	if !exists {
		if len(l.buckets) >= l.maximum {
			for candidate, value := range l.buckets {
				if now.Sub(value.last) > 10*time.Minute {
					delete(l.buckets, candidate)
				}
			}
			if len(l.buckets) >= l.maximum {
				return false
			}
		}
		current = bucket{tokens: burst, last: now}
	}
	elapsed := now.Sub(current.last).Minutes()
	if elapsed > 0 {
		current.tokens += elapsed * perMinute
		if current.tokens > burst {
			current.tokens = burst
		}
	}
	current.last = now
	if current.tokens < 1 {
		l.buckets[key] = current
		return false
	}
	current.tokens--
	l.buckets[key] = current
	return true
}
