package admission

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/atlas"
)

func TestWorkQueueCoversExactSelectionWithoutPromotion(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	selection, err := atlas.LoadSelection(filepath.Join(root, "atlas", "genesis-500", "selection.json"))
	if err != nil {
		t.Fatal(err)
	}
	sources, err := Load(filepath.Join(root, "atlas", "admissions"), selection)
	if err != nil {
		t.Fatal(err)
	}
	queue, err := BuildWorkQueue(selection, sources, "test")
	if err != nil {
		t.Fatal(err)
	}
	if queue.Counts.Selected != 500 || len(queue.Origins) != 500 || queue.Counts.PreparedDossiers != 25 || queue.Counts.UnpreparedDossiers != 475 || queue.Counts.PendingHumanReview != 22 || queue.Counts.CompletedHumanReview != 3 || queue.Counts.CanonicalAdmissions != 3 || queue.Counts.PolicyReviewState["pending"] != 497 || queue.Counts.PolicyReviewState["completed"] != 3 || queue.Counts.NextActions["prepare_dossier"] != 475 || queue.Counts.NextActions["human_catalog_review"] != 22 || queue.Counts.NextActions["advance_technical_evidence"] != 2 || queue.Counts.NextActions["prepare_bounded_profile_work_order"] != 1 {
		t.Fatalf("unexpected work queue: %#v", queue.Counts)
	}
	seen := make(map[string]struct{}, len(queue.Origins))
	for _, item := range queue.Origins {
		if _, duplicate := seen[item.OriginID]; duplicate {
			t.Fatalf("duplicate work item %s", item.OriginID)
		}
		seen[item.OriginID] = struct{}{}
		if item.CanonicalAdmission && item.PolicyReviewState != atlas.PolicyCompleted {
			t.Fatalf("canonical admission lacks completed policy for %s: %#v", item.OriginID, item)
		}
		if !item.CanonicalAdmission && (item.PolicyReviewState != atlas.PolicyPending || item.PolicyDecision != atlas.DecisionUncertain) {
			t.Fatalf("unadmitted work item promoted policy for %s: %#v", item.OriginID, item)
		}
	}
}

func TestWorkQueueRejectsMissingInputs(t *testing.T) {
	if _, err := BuildWorkQueue(nil, nil, "test"); err == nil {
		t.Fatal("nil selection accepted")
	}
}
