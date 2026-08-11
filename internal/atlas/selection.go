// Package atlas validates the offline E3 Atlas control-plane artifacts.
// Selection and registry data never authorize network access.
package atlas

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	SelectionFormat    = "tw.atlas-selection/0.2"
	MaxSelectionBytes  = 512 << 10
	RequiredCandidates = 500
)

var (
	idPattern   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	hostPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)
)

var RequiredFamilyQuotas = map[string]int{
	"government_law_public_data":                100,
	"science_research_scholarly_infrastructure": 80,
	"standards_technical_docs_open_source":      60,
	"economics_markets_public_disclosure":       60,
	"journalism_public_interest":                50,
	"health_public_health":                      50,
	"climate_environment_earth_systems":         50,
	"education_reference_culture":               30,
	"humanitarian_civic_global_development":     20,
}

type Selection struct {
	Format          string         `json:"format"`
	Version         string         `json:"version"`
	Status          string         `json:"status"`
	CanonicalScheme string         `json:"canonical_scheme"`
	Statement       string         `json:"statement"`
	FamilyQuotas    map[string]int `json:"family_quotas"`
	Candidates      []Candidate    `json:"candidates"`
	digest          [32]byte
}

type Candidate struct {
	ID               string           `json:"id"`
	CanonicalOrigin  string           `json:"canonical_origin"`
	CanonicalHost    string           `json:"canonical_host"`
	DomainFamily     string           `json:"domain_family"`
	Catalog          CandidateCatalog `json:"catalog"`
	PublisherHint    *string          `json:"publisher_hint"`
	JurisdictionHint *string          `json:"jurisdiction_hint"`
	LanguageHints    []string         `json:"language_hints"`
}

type CandidateCatalog struct {
	State CatalogState `json:"state"`
}

func LoadSelection(path string) (*Selection, error) {
	data, err := readBoundedRegular(path, MaxSelectionBytes)
	if err != nil {
		return nil, fmt.Errorf("atlas: read selection: %w", err)
	}
	var selection Selection
	policy := jsonbounded.Policy{MaxBytes: MaxSelectionBytes, MaxDepth: 12, MaxScalarBytes: 16 << 10, MaxContainerEntries: 1024, MaxTokens: 50000}
	if err := jsonbounded.Decode(data, &selection, policy, true); err != nil {
		return nil, fmt.Errorf("atlas: decode selection: %w", err)
	}
	selection.digest = sha256.Sum256(data)
	if err := selection.Validate(); err != nil {
		return nil, err
	}
	return &selection, nil
}

func (s *Selection) Validate() error {
	if s.Format != SelectionFormat || s.Version == "" || s.Status != "candidate_selection" || s.CanonicalScheme != "https" {
		return errors.New("atlas: unsupported selection metadata")
	}
	if !validText(s.Statement, 16<<10) {
		return errors.New("atlas: invalid selection statement")
	}
	if len(s.Candidates) != RequiredCandidates {
		return fmt.Errorf("atlas: selection must contain exactly %d candidates", RequiredCandidates)
	}
	if !sameQuotas(s.FamilyQuotas, RequiredFamilyQuotas) {
		return errors.New("atlas: family quotas do not match the accepted E3 universe")
	}
	ids := make(map[string]struct{}, len(s.Candidates))
	origins := make(map[string]struct{}, len(s.Candidates))
	counts := make(map[string]int, len(RequiredFamilyQuotas))
	for i := range s.Candidates {
		candidate := &s.Candidates[i]
		if err := candidate.validate(); err != nil {
			return fmt.Errorf("atlas: candidate %d: %w", i, err)
		}
		if _, exists := ids[candidate.ID]; exists {
			return fmt.Errorf("atlas: duplicate candidate ID %q", candidate.ID)
		}
		if _, exists := origins[candidate.CanonicalOrigin]; exists {
			return fmt.Errorf("atlas: duplicate canonical origin %q", candidate.CanonicalOrigin)
		}
		ids[candidate.ID] = struct{}{}
		origins[candidate.CanonicalOrigin] = struct{}{}
		counts[candidate.DomainFamily]++
	}
	if !sameQuotas(counts, RequiredFamilyQuotas) {
		return errors.New("atlas: candidate counts do not satisfy family quotas")
	}
	return nil
}

func (s *Selection) DigestReference() string {
	return "sha256:" + hex.EncodeToString(s.digest[:])
}

func (s *Selection) Find(id string) (*Candidate, error) {
	for i := range s.Candidates {
		if s.Candidates[i].ID == id {
			return &s.Candidates[i], nil
		}
	}
	return nil, fmt.Errorf("atlas: unknown candidate %q", id)
}

func (c *Candidate) validate() error {
	if !idPattern.MatchString(c.ID) || len(c.ID) > 255 {
		return errors.New("invalid candidate ID")
	}
	if c.Catalog.State != CatalogCandidate {
		return errors.New("selection entries must remain catalog candidates")
	}
	if c.PublisherHint != nil || c.JurisdictionHint != nil || c.LanguageHints == nil || len(c.LanguageHints) != 0 {
		return errors.New("candidate identity hints must be explicit null or empty values")
	}
	if _, exists := RequiredFamilyQuotas[c.DomainFamily]; !exists {
		return fmt.Errorf("unknown domain family %q", c.DomainFamily)
	}
	if c.CanonicalHost != strings.ToLower(c.CanonicalHost) || len(c.CanonicalHost) > 253 || !hostPattern.MatchString(c.CanonicalHost) || !strings.Contains(c.CanonicalHost, ".") {
		return errors.New("invalid canonical host")
	}
	parsed, err := url.Parse(c.CanonicalOrigin)
	if err != nil || parsed.Scheme != "https" || parsed.Host != c.CanonicalHost || parsed.Hostname() != c.CanonicalHost || parsed.Port() != "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("canonical origin must be an exact credential-free HTTPS origin")
	}
	if c.ID != slugHost(c.CanonicalHost) {
		return errors.New("candidate ID must be derived from canonical host")
	}
	return nil
}

func slugHost(host string) string {
	// A canonical host migration between an apex and its conventional www
	// publisher host must not manufacture a new Atlas identity. The selection's
	// duplicate-ID check still prevents both forms from entering the universe as
	// distinct origins.
	host = strings.TrimPrefix(host, "www.")
	return strings.Trim(strings.Map(func(char rune) rune {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			return char
		}
		return '-'
	}, host), "-")
}

func sameQuotas(left, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, expected := range right {
		if left[key] != expected {
			return false
		}
	}
	return true
}

func validText(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && !strings.ContainsRune(value, '\x00')
}

func readBoundedRegular(path string, maximum int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > int64(maximum) {
		return nil, errors.New("artifact is not a bounded regular file")
	}
	return os.ReadFile(path)
}
