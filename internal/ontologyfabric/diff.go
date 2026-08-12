package ontologyfabric

import (
	"fmt"
	"sort"
)

type Change struct {
	Class  string `json:"class"`
	Object string `json:"object"`
	Detail string `json:"detail"`
}

type DiffReport struct {
	Format       string   `json:"format"`
	BeforeModule string   `json:"before_module"`
	AfterModule  string   `json:"after_module"`
	Compatible   bool     `json:"compatible"`
	Changes      []Change `json:"changes"`
}

func Diff(before, after ModuleSource) (DiffReport, error) {
	if err := before.Validate(); err != nil {
		return DiffReport{}, fmt.Errorf("before: %w", err)
	}
	if err := after.Validate(); err != nil {
		return DiffReport{}, fmt.Errorf("after: %w", err)
	}
	if before.ModuleID != after.ModuleID {
		return DiffReport{}, fmt.Errorf("ontology diff: module IDs differ")
	}
	report := DiffReport{Format: "tw.ontology-diff/0.1", BeforeModule: before.ModuleID + "@" + before.Version, AfterModule: after.ModuleID + "@" + after.Version, Compatible: true}
	appendSetChanges := func(kind string, old, next []string, removeClass string) {
		oldSet, nextSet := textSet(old), textSet(next)
		for _, value := range next {
			if _, exists := oldSet[value]; !exists {
				report.Changes = append(report.Changes, Change{Class: "ADDITIVE", Object: kind + ":" + value, Detail: "added"})
			}
		}
		for _, value := range old {
			if _, exists := nextSet[value]; !exists {
				report.Changes = append(report.Changes, Change{Class: removeClass, Object: kind + ":" + value, Detail: "removed"})
				report.Compatible = false
			}
		}
	}
	appendSetChanges("import", before.Imports, after.Imports, "MEANING_CHANGING")
	appendSetChanges("mapping", before.MappingClaimDigests, after.MappingClaimDigests, "MEANING_CHANGING")
	appendSetChanges("query", before.QueryTemplateIDs, after.QueryTemplateIDs, "DEPRECATION")
	appendSetChanges("visualization", before.VisualizationIDs, after.VisualizationIDs, "DEPRECATION")
	diffConcepts(&report, before.Concepts, after.Concepts)
	diffRoles(&report, before.Roles, after.Roles)
	diffFrames(&report, before.Frames, after.Frames)
	if before.Status != after.Status {
		class := "MEANING_CHANGING"
		if after.Status == "deprecated" || after.Status == "superseded" {
			class = "DEPRECATION"
		}
		report.Changes = append(report.Changes, Change{Class: class, Object: "module-status", Detail: before.Status + " -> " + after.Status})
		if class != "ADDITIVE" {
			report.Compatible = false
		}
	}
	sort.Slice(report.Changes, func(i, j int) bool {
		if report.Changes[i].Object != report.Changes[j].Object {
			return report.Changes[i].Object < report.Changes[j].Object
		}
		if report.Changes[i].Class != report.Changes[j].Class {
			return report.Changes[i].Class < report.Changes[j].Class
		}
		return report.Changes[i].Detail < report.Changes[j].Detail
	})
	return report, nil
}

func diffConcepts(report *DiffReport, before, after []Concept) {
	old, next := conceptMap(before), conceptMap(after)
	for _, concept := range after {
		prior, exists := old[concept.ID]
		if !exists {
			report.Changes = append(report.Changes, Change{Class: "ADDITIVE", Object: "concept:" + concept.ID, Detail: "added"})
			continue
		}
		if prior.Kind != concept.Kind || !equalLabels(prior.Labels, concept.Labels) || !equalStrings(prior.Broader, concept.Broader) {
			report.Changes = append(report.Changes, Change{Class: "MEANING_CHANGING", Object: "concept:" + concept.ID, Detail: "definition changed"})
			report.Compatible = false
		}
	}
	for _, concept := range before {
		if _, exists := next[concept.ID]; !exists {
			report.Changes = append(report.Changes, Change{Class: "MEANING_CHANGING", Object: "concept:" + concept.ID, Detail: "removed"})
			report.Compatible = false
		}
	}
}

func diffRoles(report *DiffReport, before, after []Role) {
	old, next := roleMap(before), roleMap(after)
	for _, role := range after {
		prior, exists := old[role.ID]
		if !exists {
			report.Changes = append(report.Changes, Change{Class: "ADDITIVE", Object: "role:" + role.ID, Detail: "added"})
			continue
		}
		if prior.ValueType != role.ValueType {
			report.Changes = append(report.Changes, Change{Class: "MEANING_CHANGING", Object: "role:" + role.ID, Detail: "value type changed"})
			report.Compatible = false
		}
		if prior.Required != role.Required {
			class := "BROADENING"
			if role.Required {
				class = "RESTRICTIVE"
			}
			report.Changes = append(report.Changes, Change{Class: class, Object: "role:" + role.ID, Detail: "required changed"})
			report.Compatible = false
		}
		if prior.MaxValues != role.MaxValues {
			class := "BROADENING"
			if role.MaxValues < prior.MaxValues {
				class = "RESTRICTIVE"
			}
			report.Changes = append(report.Changes, Change{Class: class, Object: "role:" + role.ID, Detail: "cardinality changed"})
			report.Compatible = false
		}
	}
	for _, role := range before {
		if _, exists := next[role.ID]; !exists {
			report.Changes = append(report.Changes, Change{Class: "RESTRICTIVE", Object: "role:" + role.ID, Detail: "removed"})
			report.Compatible = false
		}
	}
}

func diffFrames(report *DiffReport, before, after []FrameType) {
	old, next := frameMap(before), frameMap(after)
	for _, frame := range after {
		prior, exists := old[frame.ID]
		if !exists {
			report.Changes = append(report.Changes, Change{Class: "ADDITIVE", Object: "frame:" + frame.ID, Detail: "added"})
			continue
		}
		if !equalStrings(prior.KeyRoles, frame.KeyRoles) {
			report.Changes = append(report.Changes, Change{Class: "IDENTITY_CHANGING", Object: "frame:" + frame.ID, Detail: "key roles changed"})
			report.Compatible = false
		}
		if !equalStrings(prior.Roles, frame.Roles) || !equalStrings(prior.RequiredRoles, frame.RequiredRoles) {
			report.Changes = append(report.Changes, Change{Class: "MEANING_CHANGING", Object: "frame:" + frame.ID, Detail: "slot contract changed"})
			report.Compatible = false
		}
	}
	for _, frame := range before {
		if _, exists := next[frame.ID]; !exists {
			report.Changes = append(report.Changes, Change{Class: "MEANING_CHANGING", Object: "frame:" + frame.ID, Detail: "removed"})
			report.Compatible = false
		}
	}
}

func textSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func conceptMap(values []Concept) map[string]Concept {
	out := make(map[string]Concept, len(values))
	for _, value := range values {
		out[value.ID] = value
	}
	return out
}

func roleMap(values []Role) map[string]Role {
	out := make(map[string]Role, len(values))
	for _, value := range values {
		out[value.ID] = value
	}
	return out
}

func frameMap(values []FrameType) map[string]FrameType {
	out := make(map[string]FrameType, len(values))
	for _, value := range values {
		out[value.ID] = value
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalLabels(a, b []Label) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
