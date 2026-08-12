// Package e4source validates source-specific E4 policy and capacity dossiers.
// A dossier is descriptive preparation. It is never a network work order.
package e4source

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	Format          = "tw.e4-source-dossier/0.1"
	MaxDossierBytes = 64 << 10
)

type Policy struct {
	ExistingDecisionRef string   `json:"existing_decision_ref"`
	ExistingScope       []string `json:"existing_scope"`
	E4ReviewState       string   `json:"e4_review_state"`
	E4Decision          string   `json:"e4_decision"`
	ReviewedAt          *string  `json:"reviewed_at"`
	Reviewer            *string  `json:"reviewer"`
}

type Budget struct {
	MaximumRequests      uint64 `json:"maximum_requests"`
	MaximumResponseBytes uint64 `json:"maximum_response_bytes"`
	MaximumTotalBytes    uint64 `json:"maximum_total_bytes"`
	MaximumRecords       uint64 `json:"maximum_records"`
	TimeoutSeconds       uint64 `json:"timeout_seconds"`
	Concurrency          uint64 `json:"concurrency"`
}

type Dossier struct {
	Format                string   `json:"format"`
	ID                    string   `json:"id"`
	UniverseID            string   `json:"universe_id"`
	CanonicalOrigin       string   `json:"canonical_origin"`
	AtlasOriginID         *string  `json:"atlas_origin_id"`
	SourceClass           string   `json:"source_class"`
	AccessClass           string   `json:"access_class"`
	NetworkExecutionState string   `json:"network_execution_state"`
	Policy                Policy   `json:"policy"`
	ProposedRoutes        []string `json:"proposed_routes"`
	ProposedBudget        Budget   `json:"proposed_budget"`
	Authentication        string   `json:"authentication"`
	Attribution           string   `json:"attribution"`
	Retention             string   `json:"retention"`
	PersonalDataTreatment string   `json:"personal_data_treatment"`
	LicenseNotes          string   `json:"license_notes"`
	EmergencyDisable      string   `json:"emergency_disable"`
	InitialScope          string   `json:"initial_scope"`
	Documentation         []string `json:"documentation"`
}

func Parse(data []byte) (Dossier, error) {
	var dossier Dossier
	policy := jsonbounded.Policy{MaxBytes: MaxDossierBytes, MaxDepth: 12, MaxScalarBytes: 16 << 10, MaxContainerEntries: 256, MaxTokens: 4096}
	if err := jsonbounded.Decode(data, &dossier, policy, true); err != nil {
		return dossier, fmt.Errorf("e4 source dossier: %w", err)
	}
	if err := dossier.Validate(); err != nil {
		return dossier, err
	}
	return dossier, nil
}

func (d Dossier) Validate() error {
	if d.Format != Format {
		return fmt.Errorf("e4 source dossier: unsupported format %q", d.Format)
	}
	for name, value := range map[string]string{"id": d.ID, "universe_id": d.UniverseID} {
		if err := identifier(name, value); err != nil {
			return err
		}
	}
	origin, err := url.Parse(d.CanonicalOrigin)
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || (origin.Path != "" && origin.Path != "/") {
		return fmt.Errorf("e4 source dossier: canonical_origin must be an HTTPS origin")
	}
	if d.AtlasOriginID != nil {
		if err := identifier("atlas_origin_id", *d.AtlasOriginID); err != nil {
			return err
		}
	}
	if err := oneOf("source_class", d.SourceClass, "public_api", "public_bulk", "public_repository", "public_api_and_bulk"); err != nil {
		return err
	}
	if err := oneOf("access_class", d.AccessClass, "unauthenticated_public", "public_file", "review_required"); err != nil {
		return err
	}
	if d.NetworkExecutionState != "disabled" {
		return fmt.Errorf("e4 source dossier: network execution must remain disabled in E4 preparation")
	}
	if err := oneOf("policy e4_review_state", d.Policy.E4ReviewState, "pending", "completed"); err != nil {
		return err
	}
	if err := oneOf("policy e4_decision", d.Policy.E4Decision, "none", "permit_live", "permit_with_constraints", "profile_only", "catalog_only", "deny", "uncertain"); err != nil {
		return err
	}
	if d.Policy.E4ReviewState == "pending" {
		if d.Policy.E4Decision != "none" || d.Policy.ReviewedAt != nil || d.Policy.Reviewer != nil {
			return fmt.Errorf("e4 source dossier: pending review cannot carry a decision or reviewer")
		}
	} else if d.Policy.E4Decision == "none" || d.Policy.ReviewedAt == nil || d.Policy.Reviewer == nil {
		return fmt.Errorf("e4 source dossier: completed review requires decision, time and reviewer")
	}
	if err := sortedText("existing_scope", d.Policy.ExistingScope, 32, true); err != nil {
		return err
	}
	if err := sortedText("proposed_routes", d.ProposedRoutes, 32, false); err != nil {
		return err
	}
	if err := sortedText("documentation", d.Documentation, 32, false); err != nil {
		return err
	}
	for _, route := range d.ProposedRoutes {
		parsed, parseErr := url.Parse(route)
		if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Host != origin.Host || parsed.User != nil || parsed.Fragment != "" {
			return fmt.Errorf("e4 source dossier: proposed route is not on exact canonical host: %q", route)
		}
	}
	for _, document := range d.Documentation {
		parsed, parseErr := url.Parse(document)
		if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("e4 source dossier: invalid documentation URL %q", document)
		}
	}
	budget := d.ProposedBudget
	if budget.MaximumRequests < 1 || budget.MaximumRequests > 100000 || budget.MaximumResponseBytes < 1024 || budget.MaximumResponseBytes > 1<<30 || budget.MaximumTotalBytes < budget.MaximumResponseBytes || budget.MaximumTotalBytes > 1<<40 || budget.MaximumRecords < 1 || budget.MaximumRecords > 10000000 || budget.TimeoutSeconds < 1 || budget.TimeoutSeconds > 3600 || budget.Concurrency < 1 || budget.Concurrency > 8 {
		return fmt.Errorf("e4 source dossier: proposed budget outside bounded profile")
	}
	for name, value := range map[string]string{
		"authentication": d.Authentication, "attribution": d.Attribution, "retention": d.Retention,
		"personal_data_treatment": d.PersonalDataTreatment, "license_notes": d.LicenseNotes,
		"emergency_disable": d.EmergencyDisable, "initial_scope": d.InitialScope,
	} {
		if value == "" || len(value) > 16<<10 || !utf8.ValidString(value) {
			return fmt.Errorf("e4 source dossier: invalid %s", name)
		}
	}
	return nil
}

func identifier(name, value string) error {
	if value == "" || len(value) > 512 || !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("e4 source dossier: invalid %s", name)
	}
	return nil
}

func oneOf(name, value string, allowed ...string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("e4 source dossier: unsupported %s %q", name, value)
}

func sortedText(name string, values []string, maximum int, allowEmpty bool) error {
	if (!allowEmpty && len(values) == 0) || len(values) > maximum {
		return fmt.Errorf("e4 source dossier: %s count invalid", name)
	}
	for i, value := range values {
		if value == "" || len(value) > 16<<10 || !utf8.ValidString(value) {
			return fmt.Errorf("e4 source dossier: invalid %s entry", name)
		}
		if i > 0 && values[i-1] >= value {
			return fmt.Errorf("e4 source dossier: %s must be sorted and unique", name)
		}
	}
	return nil
}
