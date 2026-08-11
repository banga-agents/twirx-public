// Package origincatalog loads the reviewed, non-user-extensible E2 origin
// catalog and constructs destinations only from admitted operation inputs.
package origincatalog

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

const (
	Format          = "tw.origin-catalog/0.1"
	MaxCatalogBytes = 256 << 10
	MaxOrigins      = 64
)

type Catalog struct {
	Format  string   `json:"format"`
	Version string   `json:"version"`
	Origins []Origin `json:"origins"`
}

type Origin struct {
	ID                string            `json:"id"`
	RegistryID        string            `json:"registry_id"`
	Version           string            `json:"version"`
	Title             string            `json:"title"`
	Publisher         string            `json:"publisher"`
	SourceClass       string            `json:"source_class"`
	AdmissionStatus   string            `json:"admission_status"`
	AccessMode        string            `json:"access_mode"`
	EndpointTemplate  string            `json:"endpoint_template"`
	AllowedHost       string            `json:"allowed_host"`
	ReplayFixture     string            `json:"replay_fixture"`
	ReplayObservedAt  string            `json:"replay_observed_at"`
	ReplayInput       map[string]string `json:"replay_input"`
	Operations        []string          `json:"operations"`
	Attribution       string            `json:"attribution"`
	PolicyReference   string            `json:"policy_reference"`
	TermsReference    string            `json:"terms_reference"`
	MaxResponseBytes  int64             `json:"max_response_bytes"`
	TimeoutSeconds    int               `json:"timeout_seconds"`
	RequestsPerMinute int               `json:"requests_per_minute"`
	FreshEnabled      bool              `json:"fresh_enabled"`
}

func Load(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("origincatalog: read: %w", err)
	}
	var catalog Catalog
	policy := jsonbounded.Policy{MaxBytes: MaxCatalogBytes, MaxDepth: 10, MaxScalarBytes: 16 << 10, MaxContainerEntries: 512, MaxTokens: 10000}
	if err := jsonbounded.Decode(data, &catalog, policy, true); err != nil {
		return nil, fmt.Errorf("origincatalog: decode: %w", err)
	}
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	return &catalog, nil
}

func (c *Catalog) Validate() error {
	if c.Format != Format || c.Version == "" {
		return errors.New("origincatalog: unsupported format or missing version")
	}
	if len(c.Origins) < 3 || len(c.Origins) > MaxOrigins {
		return errors.New("origincatalog: E2 requires 3 to 64 origins")
	}
	seenOrigins := make(map[string]struct{}, len(c.Origins))
	seenOperations := make(map[string]struct{})
	for i := range c.Origins {
		origin := &c.Origins[i]
		if _, exists := seenOrigins[origin.ID]; exists {
			return fmt.Errorf("origincatalog: duplicate origin %q", origin.ID)
		}
		seenOrigins[origin.ID] = struct{}{}
		for name, value := range map[string]string{"ID": origin.ID, "registry ID": origin.RegistryID, "version": origin.Version, "title": origin.Title, "publisher": origin.Publisher, "source class": origin.SourceClass, "endpoint": origin.EndpointTemplate, "allowed host": origin.AllowedHost, "fixture": origin.ReplayFixture, "replay observed at": origin.ReplayObservedAt, "attribution": origin.Attribution, "policy reference": origin.PolicyReference, "terms reference": origin.TermsReference} {
			if value == "" || len(value) > 16384 || strings.ContainsRune(value, '\x00') {
				return fmt.Errorf("origincatalog: origin %d invalid %s", i, name)
			}
		}
		if origin.AdmissionStatus != "reviewed_e2" {
			return fmt.Errorf("origincatalog: origin %q is not reviewed for E2", origin.ID)
		}
		if origin.AccessMode != "https" && origin.AccessMode != "controlled_fixture" {
			return fmt.Errorf("origincatalog: origin %q invalid access mode", origin.ID)
		}
		if origin.MaxResponseBytes <= 0 || origin.MaxResponseBytes > 2<<20 || origin.TimeoutSeconds < 1 || origin.TimeoutSeconds > 30 || origin.RequestsPerMinute < 1 || origin.RequestsPerMinute > 120 {
			return fmt.Errorf("origincatalog: origin %q invalid limits", origin.ID)
		}
		if len(origin.Operations) == 0 || len(origin.Operations) > 32 || !sort.StringsAreSorted(origin.Operations) {
			return fmt.Errorf("origincatalog: origin %q operations must be non-empty, bounded, and sorted", origin.ID)
		}
		if !safeRelative(origin.ReplayFixture) {
			return fmt.Errorf("origincatalog: origin %q unsafe fixture path", origin.ID)
		}
		parsedTime, timeErr := time.Parse(time.RFC3339Nano, origin.ReplayObservedAt)
		if timeErr != nil || parsedTime.Location() != time.UTC || parsedTime.Format(time.RFC3339Nano) != origin.ReplayObservedAt {
			return fmt.Errorf("origincatalog: origin %q replay time is not canonical UTC", origin.ID)
		}
		parsed, err := url.Parse(origin.EndpointTemplate)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() != origin.AllowedHost || parsed.User != nil {
			return fmt.Errorf("origincatalog: origin %q endpoint is outside admitted HTTPS host", origin.ID)
		}
		for _, operation := range origin.Operations {
			if _, exists := seenOperations[operation]; exists {
				return fmt.Errorf("origincatalog: operation %q has multiple origin owners", operation)
			}
			seenOperations[operation] = struct{}{}
		}
	}
	return nil
}

// ValidateRegistry binds every E2 execution entry to the single canonical
// origin identity registry. It does not replace E2's narrower route and input
// admission rules, and it grants no Atlas scheduler authority.
func (c *Catalog) ValidateRegistry(registry *atlas.Registry) error {
	if registry == nil {
		return errors.New("origincatalog: canonical registry is required")
	}
	for index := range c.Origins {
		origin := &c.Origins[index]
		record, err := registry.Find(origin.RegistryID)
		if err != nil {
			return fmt.Errorf("origincatalog: origin %q: %w", origin.ID, err)
		}
		byExecutionID, err := registry.FindExecutionCatalogID(origin.ID)
		if err != nil || byExecutionID.ID != record.ID {
			return fmt.Errorf("origincatalog: origin %q has inconsistent canonical alias binding", origin.ID)
		}
		parsed, err := url.Parse(record.CanonicalOrigin)
		if err != nil || parsed.Hostname() != origin.AllowedHost || record.Publisher.Name != origin.Publisher {
			return fmt.Errorf("origincatalog: origin %q disagrees with canonical identity", origin.ID)
		}
		if origin.SourceClass == "controlled_fixture" && record.Scope != atlas.ScopeTestFixture {
			return fmt.Errorf("origincatalog: controlled fixture %q is not classified as test_fixture", origin.ID)
		}
		if origin.SourceClass != "controlled_fixture" && record.Scope != atlas.ScopeGenesisPublic {
			return fmt.Errorf("origincatalog: public origin %q is not classified as genesis_public", origin.ID)
		}
	}
	return nil
}

func (c *Catalog) Find(id string) (*Origin, error) {
	for i := range c.Origins {
		if c.Origins[i].ID == id {
			return &c.Origins[i], nil
		}
	}
	return nil, fmt.Errorf("origincatalog: unknown origin %q", id)
}

func (c *Catalog) ForOperation(operationID string) (*Origin, error) {
	for i := range c.Origins {
		for _, candidate := range c.Origins[i].Operations {
			if candidate == operationID {
				return &c.Origins[i], nil
			}
		}
	}
	return nil, fmt.Errorf("origincatalog: operation %q is not admitted", operationID)
}

// RequestURL substitutes only contract-validated, allowlisted values and then
// revalidates the final HTTPS host. No caller-supplied URL is accepted.
func (o *Origin) RequestURL(op *twircontract.Operation, input map[string]string) (string, error) {
	if !o.FreshEnabled {
		return "", errors.New("origincatalog: fresh mode is disabled for origin")
	}
	if op.OriginID != o.ID || !contains(o.Operations, op.ID) {
		return "", errors.New("origincatalog: operation is not owned by origin")
	}
	if err := twircontract.ValidateInput(op, input); err != nil {
		return "", err
	}
	raw := o.EndpointTemplate
	for _, field := range op.Input {
		raw = strings.ReplaceAll(raw, "{"+field.ID+"}", url.PathEscape(input[field.ID]))
	}
	if strings.ContainsAny(raw, "{}") {
		return "", errors.New("origincatalog: unresolved endpoint placeholder")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != o.AllowedHost || parsed.User != nil {
		return "", errors.New("origincatalog: constructed endpoint left admitted host")
	}
	return parsed.String(), nil
}

func safeRelative(path string) bool {
	clean := filepath.Clean(path)
	return clean == path && clean != "." && !filepath.IsAbs(clean) && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
