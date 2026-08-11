package atlas

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const (
	PolicySetFormat   = "tw.atlas-policy-set/0.2"
	MaxPolicySetBytes = 8 << 20
	CrawlerToken      = "TWIRXBot"
)

type PolicySet struct {
	Format    string         `json:"format"`
	Version   string         `json:"version"`
	Statement string         `json:"statement"`
	Policies  []OriginPolicy `json:"policies"`
	digest    [32]byte
}

type OriginPolicy struct {
	ID              string            `json:"id"`
	OriginID        string            `json:"origin_id"`
	CanonicalOrigin string            `json:"canonical_origin"`
	ReviewState     PolicyReviewState `json:"review_state"`
	Decision        PolicyDecision    `json:"decision"`
	ReviewedAt      *string           `json:"reviewed_at"`
	Reviewer        *string           `json:"reviewer"`
	Robots          RobotsAssessment  `json:"robots"`
	TermsState      string            `json:"terms_state"`
	TermsReference  string            `json:"terms_reference"`
	Attribution     string            `json:"attribution"`
	Authentication  string            `json:"authentication"`
	RatePolicy      string            `json:"rate_policy"`
	RetentionPolicy string            `json:"retention_policy"`
	RiskState       string            `json:"risk_state"`
	ReviewerNotes   string            `json:"reviewer_notes"`
	EvidenceRefs    []string          `json:"evidence_refs"`
}

type RobotsAssessment struct {
	State          string  `json:"state"`
	URL            string  `json:"url"`
	ObservedAt     *string `json:"observed_at"`
	ArtifactDigest *string `json:"artifact_digest"`
	ProductToken   string  `json:"product_token"`
}

func LoadPolicySet(path string, selection *Selection) (*PolicySet, error) {
	data, err := readBoundedRegular(path, MaxPolicySetBytes)
	if err != nil {
		return nil, fmt.Errorf("atlas: read policy set: %w", err)
	}
	var policies PolicySet
	if err := decodeStrict(data, &policies, jsonPolicy(MaxPolicySetBytes)); err != nil {
		return nil, fmt.Errorf("atlas: decode policy set: %w", err)
	}
	policies.digest = sha256.Sum256(data)
	if err := policies.Validate(selection); err != nil {
		return nil, err
	}
	return &policies, nil
}

// BuildPolicySet deterministically encodes independently reviewed per-origin
// policy artifacts into the canonical policy set. It does not create or alter
// a decision.
func BuildPolicySet(version, statement string, policies []OriginPolicy, selection *Selection) (*PolicySet, []byte, error) {
	set := PolicySet{Format: PolicySetFormat, Version: version, Statement: statement, Policies: append([]OriginPolicy(nil), policies...)}
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("atlas: encode policy set: %w", err)
	}
	data = append(data, '\n')
	set.digest = sha256.Sum256(data)
	if err := set.Validate(selection); err != nil {
		return nil, nil, err
	}
	return &set, data, nil
}

func (p *PolicySet) Validate(selection *Selection) error {
	if p == nil || selection == nil || p.Format != PolicySetFormat || !validText(p.Version, 128) || !validText(p.Statement, 16<<10) {
		return errors.New("atlas: unsupported policy-set metadata")
	}
	if p.digest == ([32]byte{}) {
		return errors.New("atlas: policy set has no exact artifact identity")
	}
	if len(p.Policies) > RequiredCandidates {
		return errors.New("atlas: policy set exceeds selected universe")
	}
	if !sort.SliceIsSorted(p.Policies, func(i, j int) bool { return p.Policies[i].OriginID < p.Policies[j].OriginID }) {
		return errors.New("atlas: policies must be sorted by origin_id")
	}
	seen := make(map[string]struct{}, len(p.Policies))
	for index := range p.Policies {
		policy := &p.Policies[index]
		candidate, err := selection.Find(policy.OriginID)
		if err != nil {
			return fmt.Errorf("atlas: policy %d: %w", index, err)
		}
		if _, exists := seen[policy.OriginID]; exists {
			return fmt.Errorf("atlas: duplicate policy origin %q", policy.OriginID)
		}
		seen[policy.OriginID] = struct{}{}
		if policy.ID != policy.OriginID || policy.CanonicalOrigin != candidate.CanonicalOrigin {
			return fmt.Errorf("atlas: policy %q changes selected identity", policy.ID)
		}
		if err := policy.Validate(); err != nil {
			return fmt.Errorf("atlas: policy %q: %w", policy.ID, err)
		}
	}
	return nil
}

func (p *PolicySet) DigestReference() string {
	return "sha256:" + hex.EncodeToString(p.digest[:])
}

func (p *PolicySet) Find(originID string) (*OriginPolicy, error) {
	if p == nil {
		return nil, errors.New("atlas: policy set is nil")
	}
	index := sort.Search(len(p.Policies), func(index int) bool { return p.Policies[index].OriginID >= originID })
	if index >= len(p.Policies) || p.Policies[index].OriginID != originID {
		return nil, fmt.Errorf("atlas: no policy artifact for origin %q", originID)
	}
	return &p.Policies[index], nil
}

func (p *OriginPolicy) Validate() error {
	if !idPattern.MatchString(p.ID) || p.OriginID != p.ID || !allText(p.TermsReference, p.Attribution, p.Authentication, p.RatePolicy, p.RetentionPolicy, p.ReviewerNotes) {
		return errors.New("invalid required policy text")
	}
	if !validPolicyDecision(p.Decision) {
		return errors.New("invalid policy decision")
	}
	if p.ReviewState == PolicyPending {
		if p.Decision != DecisionUncertain || p.ReviewedAt != nil || p.Reviewer != nil {
			return errors.New("pending policy must be uncertain and cannot claim completed review metadata")
		}
	} else if p.ReviewState == PolicyCompleted {
		if p.ReviewedAt == nil || p.Reviewer == nil || !validText(*p.Reviewer, 4096) {
			return errors.New("completed policy requires reviewer and review time")
		}
		if err := canonicalTime(*p.ReviewedAt); err != nil {
			return fmt.Errorf("reviewed_at: %w", err)
		}
	} else {
		return errors.New("invalid policy review state")
	}
	if p.TermsState != "accepted" && p.TermsState != "denied" && p.TermsState != "not_found" && p.TermsState != "pending" {
		return errors.New("invalid terms state")
	}
	if p.RiskState != "accepted" && p.RiskState != "denied" && p.RiskState != "pending" {
		return errors.New("invalid risk state")
	}
	if len(p.EvidenceRefs) == 0 || !uniqueSorted(p.EvidenceRefs) {
		return errors.New("policy evidence references must be sorted, unique, and non-empty")
	}
	if err := p.Robots.Validate(p.CanonicalOrigin); err != nil {
		return err
	}
	if p.ReviewState == PolicyPending && (p.TermsState != "pending" || p.RiskState != "pending") {
		return errors.New("pending policy cannot claim completed terms or risk review")
	}
	if p.ReviewState == PolicyCompleted && (p.Decision == DecisionPermitLive || p.Decision == DecisionPermitWithConstraints || p.Decision == DecisionProfileOnly) {
		if p.Robots.State != "successful" && p.Robots.State != "unavailable" {
			return errors.New("retrieval-permitting decision requires successful or explicitly unavailable robots assessment")
		}
		if p.TermsState != "accepted" && p.TermsState != "not_found" {
			return errors.New("retrieval-permitting decision requires completed terms review")
		}
		if p.Authentication != "none_required" {
			return errors.New("Genesis retrieval decision cannot require authentication")
		}
		if p.RiskState != "accepted" {
			return errors.New("retrieval-permitting decision requires accepted risk review")
		}
	}
	return nil
}

func (r RobotsAssessment) Validate(canonicalOrigin string) error {
	if r.State != "successful" && r.State != "unavailable" && r.State != "unreachable" && r.State != "redirect_limit" && r.State != "not_observed" {
		return errors.New("invalid robots state")
	}
	if r.ProductToken != CrawlerToken || r.URL != strings.TrimSuffix(canonicalOrigin, "/")+"/robots.txt" {
		return errors.New("robots identity is not bound to the canonical origin and crawler token")
	}
	parsed, err := url.Parse(r.URL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("invalid robots URL")
	}
	if r.State == "not_observed" {
		if r.ObservedAt != nil || r.ArtifactDigest != nil {
			return errors.New("unobserved robots state cannot claim observation evidence")
		}
		return nil
	}
	if r.ObservedAt == nil {
		return errors.New("robots outcome requires observed_at")
	}
	if err := canonicalTime(*r.ObservedAt); err != nil {
		return fmt.Errorf("robots observed_at: %w", err)
	}
	if r.State == "successful" && (r.ArtifactDigest == nil || !validDigestRef(*r.ArtifactDigest)) {
		return errors.New("successful robots assessment requires artifact digest")
	}
	if r.ArtifactDigest != nil && !validDigestRef(*r.ArtifactDigest) {
		return errors.New("invalid robots artifact digest")
	}
	return nil
}
