package admission

import (
	"errors"
	"fmt"
	"sort"

	"github.com/typed-web-commons/typed-web/internal/atlas"
)

const (
	WorkQueueFormat = "tw.atlas-admission-work-queue/0.1"
	DossierMissing  = "not_prepared"
	DossierPrepared = "prepared"
)

type WorkQueue struct {
	Format          string           `json:"format"`
	Version         string           `json:"version"`
	SelectionDigest string           `json:"selection_digest"`
	Statement       string           `json:"statement"`
	Counts          WorkQueueCounts  `json:"counts"`
	Origins         []OriginWorkItem `json:"origins"`
}

type WorkQueueCounts struct {
	Selected             int            `json:"selected"`
	PreparedDossiers     int            `json:"prepared_dossiers"`
	UnpreparedDossiers   int            `json:"unprepared_dossiers"`
	PendingHumanReview   int            `json:"pending_human_review"`
	CompletedHumanReview int            `json:"completed_human_review"`
	CanonicalAdmissions  int            `json:"canonical_admissions"`
	PolicyReviewState    map[string]int `json:"policy_review_state"`
	PolicyDecision       map[string]int `json:"policy_decision"`
	NextActions          map[string]int `json:"next_actions"`
	DomainFamilies       map[string]int `json:"domain_families"`
}

type OriginWorkItem struct {
	OriginID           string                  `json:"origin_id"`
	CanonicalOrigin    string                  `json:"canonical_origin"`
	DomainFamily       string                  `json:"domain_family"`
	DossierState       string                  `json:"dossier_state"`
	AdmissionReview    string                  `json:"admission_review_state"`
	CanonicalAdmission bool                    `json:"canonical_admission"`
	CatalogState       atlas.CatalogState      `json:"catalog_state"`
	PolicyReviewState  atlas.PolicyReviewState `json:"policy_review_state"`
	PolicyDecision     atlas.PolicyDecision    `json:"policy_decision"`
	NextAction         string                  `json:"next_action"`
	Statement          string                  `json:"statement"`
}

func BuildWorkQueue(selection *atlas.Selection, sources []Source, version string) (WorkQueue, error) {
	if selection == nil || version == "" {
		return WorkQueue{}, errors.New("admission: selection and work-queue version are required")
	}
	if err := selection.Validate(); err != nil {
		return WorkQueue{}, err
	}
	sourceByID := make(map[string]Source, len(sources))
	for _, source := range sources {
		if _, exists := sourceByID[source.Record.ID]; exists {
			return WorkQueue{}, fmt.Errorf("admission: duplicate work-queue source %s", source.Record.ID)
		}
		candidate, err := selection.Find(source.Record.ID)
		if err != nil || candidate.CanonicalOrigin != source.Record.CanonicalOrigin {
			return WorkQueue{}, fmt.Errorf("admission: work-queue source %s changes selected identity", source.Record.ID)
		}
		sourceByID[source.Record.ID] = source
	}
	queue := WorkQueue{
		Format: WorkQueueFormat, Version: version, SelectionDigest: selection.DigestReference(),
		Statement: "Every selected origin has one explicit admission work item. Missing dossiers, pending reviews, and uncertain policy are visible and cannot authorize retrieval or canonical admission.",
		Counts: WorkQueueCounts{
			Selected: len(selection.Candidates), PolicyReviewState: map[string]int{}, PolicyDecision: map[string]int{}, NextActions: map[string]int{}, DomainFamilies: map[string]int{},
		},
		Origins: make([]OriginWorkItem, 0, len(selection.Candidates)),
	}
	for _, candidate := range selection.Candidates {
		item := OriginWorkItem{
			OriginID: candidate.ID, CanonicalOrigin: candidate.CanonicalOrigin, DomainFamily: candidate.DomainFamily,
			DossierState: DossierMissing, AdmissionReview: "not_started", CatalogState: atlas.CatalogCandidate,
			PolicyReviewState: atlas.PolicyPending, PolicyDecision: atlas.DecisionUncertain, NextAction: "prepare_dossier",
			Statement: "Selected candidate only; prepare bounded identity and policy evidence before human review.",
		}
		if source, exists := sourceByID[candidate.ID]; exists {
			item.DossierState = DossierPrepared
			item.AdmissionReview = source.Decision.ReviewState
			item.CanonicalAdmission = source.Decision.ReviewState == ReviewCompleted && source.Decision.AdmitToRegistry
			item.CatalogState = source.Record.Catalog.State
			item.PolicyReviewState = source.Policy.ReviewState
			item.PolicyDecision = source.Policy.Decision
			item.NextAction = nextAction(source)
			item.Statement = "Prepared dossier state is derived from digest-bound per-origin artifacts; only explicit human review can admit it."
			queue.Counts.PreparedDossiers++
			if source.Decision.ReviewState == ReviewPending {
				queue.Counts.PendingHumanReview++
			} else {
				queue.Counts.CompletedHumanReview++
			}
			if item.CanonicalAdmission {
				queue.Counts.CanonicalAdmissions++
			}
		} else {
			queue.Counts.UnpreparedDossiers++
		}
		queue.Counts.PolicyReviewState[string(item.PolicyReviewState)]++
		queue.Counts.PolicyDecision[string(item.PolicyDecision)]++
		queue.Counts.NextActions[item.NextAction]++
		queue.Counts.DomainFamilies[item.DomainFamily]++
		queue.Origins = append(queue.Origins, item)
	}
	sort.Slice(queue.Origins, func(i, j int) bool { return queue.Origins[i].OriginID < queue.Origins[j].OriginID })
	if queue.Counts.PreparedDossiers+queue.Counts.UnpreparedDossiers != atlas.RequiredCandidates || len(queue.Origins) != atlas.RequiredCandidates {
		return WorkQueue{}, errors.New("admission: work queue does not cover the exact Genesis-500 selection")
	}
	return queue, nil
}

func nextAction(source Source) string {
	if source.Decision.ReviewState == ReviewPending {
		return "human_catalog_review"
	}
	if source.Policy.ReviewState == atlas.PolicyPending {
		return "human_policy_review"
	}
	switch source.Policy.Decision {
	case atlas.DecisionPermitLive, atlas.DecisionPermitWithConstraints, atlas.DecisionProfileOnly:
		if source.Record.Technical.Stage == atlas.TechnicalUnprofiled {
			return "prepare_bounded_profile_work_order"
		}
		return "advance_technical_evidence"
	case atlas.DecisionCatalogOnly, atlas.DecisionDeny, atlas.DecisionUncertain:
		return "retain_without_retrieval"
	default:
		return "human_policy_review"
	}
}
