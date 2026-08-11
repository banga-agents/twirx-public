package atlas

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

const FrontierPlanFormat = "tw.atlas-frontier-plan/0.1"

type SchedulerState struct {
	State                string  `json:"state"`
	LastAttempt          *string `json:"last_attempt"`
	CooldownUntil        *string `json:"cooldown_until"`
	ConsecutiveFailures  int     `json:"consecutive_failures"`
	PublicValue          int     `json:"public_value"`
	SemanticNovelty      int     `json:"semantic_novelty"`
	DemonstratedDemand   int     `json:"demonstrated_demand"`
	ChangeProbability    int     `json:"change_probability"`
	HealthNeed           int     `json:"health_need"`
	EstimatedRequestCost int     `json:"estimated_request_cost"`
}

type FrontierPlan struct {
	Format        string             `json:"format"`
	GeneratedAt   string             `json:"generated_at"`
	Mode          string             `json:"mode"`
	NetworkAccess string             `json:"network_access"`
	Decisions     []FrontierDecision `json:"decisions"`
	Jobs          []FrontierJob      `json:"jobs"`
}

type FrontierDecision struct {
	OriginID string  `json:"origin_id"`
	Action   string  `json:"action"`
	Reason   string  `json:"reason"`
	DueAt    *string `json:"due_at"`
}

type FrontierJob struct {
	OriginID          string `json:"origin_id"`
	Priority          int64  `json:"priority"`
	RefreshClass      string `json:"refresh_class"`
	DueAt             string `json:"due_at"`
	RequestBudget     int    `json:"request_budget"`
	StorageBudgetByte int64  `json:"storage_budget_bytes"`
}

func (s SchedulerState) Validate(runtime RuntimeRecord) error {
	if s.State != "disabled" && s.State != "ready" && s.State != "cooldown" {
		return errors.New("scheduler state must be disabled, ready, or cooldown")
	}
	if s.ConsecutiveFailures < 0 || s.ConsecutiveFailures > 1000 {
		return errors.New("scheduler failure count outside bounds")
	}
	for _, field := range []struct {
		name  string
		value int
	}{
		{name: "public_value", value: s.PublicValue},
		{name: "semantic_novelty", value: s.SemanticNovelty},
		{name: "demonstrated_demand", value: s.DemonstratedDemand},
		{name: "change_probability", value: s.ChangeProbability},
		{name: "health_need", value: s.HealthNeed},
		{name: "estimated_request_cost", value: s.EstimatedRequestCost},
	} {
		if field.value < 0 || field.value > 100 {
			return fmt.Errorf("scheduler %s outside bounds", field.name)
		}
	}
	for _, field := range []struct {
		name  string
		value *string
	}{{name: "last_attempt", value: s.LastAttempt}, {name: "cooldown_until", value: s.CooldownUntil}} {
		if field.value != nil {
			if err := canonicalTime(*field.value); err != nil {
				return fmt.Errorf("scheduler %s: %w", field.name, err)
			}
		}
	}
	if s.State == "disabled" {
		if s.CooldownUntil != nil || s.PublicValue != 0 || s.SemanticNovelty != 0 || s.DemonstratedDemand != 0 || s.ChangeProbability != 0 || s.HealthNeed != 0 || s.EstimatedRequestCost != 0 {
			return errors.New("disabled scheduler cannot carry active priority or cooldown state")
		}
		return nil
	}
	if runtime.RequestBudget <= 0 || runtime.StorageBudgetByte <= 0 {
		return errors.New("active scheduler requires positive request and storage budgets")
	}
	if _, exists := refreshIntervals[runtime.RefreshClass]; !exists {
		return errors.New("active scheduler requires hot, warm, cool, or archival refresh class")
	}
	if s.PublicValue == 0 || s.SemanticNovelty == 0 || s.DemonstratedDemand == 0 || s.ChangeProbability == 0 || s.HealthNeed == 0 || s.EstimatedRequestCost == 0 {
		return errors.New("active scheduler requires explicit non-zero priority factors")
	}
	if s.State == "ready" && s.CooldownUntil != nil || s.State == "cooldown" && s.CooldownUntil == nil {
		return errors.New("scheduler cooldown state is inconsistent")
	}
	return nil
}

var refreshIntervals = map[string]time.Duration{
	"hot":      time.Hour,
	"warm":     24 * time.Hour,
	"cool":     7 * 24 * time.Hour,
	"archival": 30 * 24 * time.Hour,
}

func BuildDryRunFrontier(selection *Selection, registry *Registry, policies *PolicySet, at time.Time) (FrontierPlan, error) {
	if selection == nil || registry == nil || policies == nil {
		return FrontierPlan{}, errors.New("atlas: selection, registry, and policies are required")
	}
	if err := policies.Validate(selection); err != nil {
		return FrontierPlan{}, err
	}
	if err := registry.Validate(selection, policies); err != nil {
		return FrontierPlan{}, err
	}
	if at.Location() != time.UTC || at.Format(time.RFC3339Nano) != at.UTC().Format(time.RFC3339Nano) {
		return FrontierPlan{}, errors.New("atlas: frontier time must be UTC")
	}
	plan := FrontierPlan{Format: FrontierPlanFormat, GeneratedAt: at.Format(time.RFC3339Nano), Mode: "dry_run", NetworkAccess: "disabled", Decisions: []FrontierDecision{}, Jobs: []FrontierJob{}}
	publicRecords := make(map[string]*OriginRecord, len(registry.Origins))
	for index := range registry.Origins {
		record := &registry.Origins[index]
		if record.Scope == ScopeGenesisPublic {
			publicRecords[record.ID] = record
		}
	}
	for index := range selection.Candidates {
		candidate := &selection.Candidates[index]
		record, admitted := publicRecords[candidate.ID]
		if !admitted {
			plan.Decisions = append(plan.Decisions, FrontierDecision{OriginID: candidate.ID, Action: "blocked", Reason: "catalog_review_pending"})
			continue
		}
		decision := FrontierDecision{OriginID: record.ID}
		if record.Policy.ReviewState != PolicyCompleted {
			decision.Action, decision.Reason = "blocked", "policy_review_pending"
			plan.Decisions = append(plan.Decisions, decision)
			continue
		}
		if record.Policy.Decision != DecisionPermitLive && record.Policy.Decision != DecisionPermitWithConstraints {
			decision.Action, decision.Reason = "blocked", "policy_not_live_permitted"
			plan.Decisions = append(plan.Decisions, decision)
			continue
		}
		if record.Runtime.Scheduler.State == "disabled" {
			decision.Action, decision.Reason = "blocked", "scheduler_disabled"
			plan.Decisions = append(plan.Decisions, decision)
			continue
		}
		if record.Runtime.Scheduler.CooldownUntil != nil {
			cooldown, _ := time.Parse(time.RFC3339Nano, *record.Runtime.Scheduler.CooldownUntil)
			if cooldown.After(at) {
				value := cooldown.Format(time.RFC3339Nano)
				decision.Action, decision.Reason, decision.DueAt = "defer", "cooldown", &value
				plan.Decisions = append(plan.Decisions, decision)
				continue
			}
		}
		due := at
		if record.Runtime.Scheduler.LastAttempt != nil {
			last, _ := time.Parse(time.RFC3339Nano, *record.Runtime.Scheduler.LastAttempt)
			due = last.Add(refreshIntervals[record.Runtime.RefreshClass])
		}
		dueText := due.Format(time.RFC3339Nano)
		if due.After(at) {
			decision.Action, decision.Reason, decision.DueAt = "defer", "not_due", &dueText
			plan.Decisions = append(plan.Decisions, decision)
			continue
		}
		decision.Action, decision.Reason, decision.DueAt = "schedule", "due", &dueText
		plan.Decisions = append(plan.Decisions, decision)
		scheduler := record.Runtime.Scheduler
		priority := int64(scheduler.PublicValue) * int64(scheduler.SemanticNovelty) * int64(scheduler.DemonstratedDemand) * int64(scheduler.ChangeProbability) * int64(scheduler.HealthNeed) / int64(scheduler.EstimatedRequestCost)
		if priority == 0 {
			priority = 1
		}
		plan.Jobs = append(plan.Jobs, FrontierJob{OriginID: record.ID, Priority: priority, RefreshClass: record.Runtime.RefreshClass, DueAt: dueText, RequestBudget: record.Runtime.RequestBudget, StorageBudgetByte: record.Runtime.StorageBudgetByte})
	}
	sort.Slice(plan.Decisions, func(i, j int) bool { return plan.Decisions[i].OriginID < plan.Decisions[j].OriginID })
	sort.Slice(plan.Jobs, func(i, j int) bool {
		if plan.Jobs[i].Priority != plan.Jobs[j].Priority {
			return plan.Jobs[i].Priority > plan.Jobs[j].Priority
		}
		return plan.Jobs[i].OriginID < plan.Jobs[j].OriginID
	})
	return plan, nil
}
