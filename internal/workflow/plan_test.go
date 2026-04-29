package workflow

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestFoundationUpPlanHasStableStepIDsAndChecklist(t *testing.T) {
	plan := FoundationUpPlan()
	var ids []string
	for _, step := range plan.Steps {
		ids = append(ids, step.ID)
		if step.Status != StepPending {
			t.Fatalf("step %s status = %q, want pending", step.ID, step.Status)
		}
	}
	want := []string{"resolve_repo", "verify_auth", "check_safety", "preview_state", "save_state"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("step IDs = %#v, want %#v", ids, want)
	}
	checklist := strings.Join(plan.Checklist(), "\n")
	for _, id := range want {
		if !strings.Contains(checklist, id) {
			t.Fatalf("checklist missing %s:\n%s", id, checklist)
		}
	}
}

func TestApplyRunsPendingStepsAndRecordsStatuses(t *testing.T) {
	plan := FoundationUpPlan()
	var ran []string
	applied, err := Apply(context.Background(), plan, StepRunnerFunc(func(_ context.Context, step Step) (StepStatus, error) {
		ran = append(ran, step.ID)
		return StepDone, nil
	}))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(ran) != len(plan.Steps) {
		t.Fatalf("ran %d steps, want %d", len(ran), len(plan.Steps))
	}
	for _, step := range applied.Steps {
		if step.Status != StepDone {
			t.Fatalf("step %s status = %q, want done", step.ID, step.Status)
		}
	}
}
