package wbs

import (
	"errors"
	"testing"
)

func estimatedWBS(t *testing.T, descs ...string) *WBS {
	t.Helper()
	w := approvedWBS(t, descs...)
	w.SetEstimates(estimatesFor(descs))
	return w
}

// estimatesFor builds a valid ascending estimate for each task position.
func estimatesFor(descs []string) []EstimateAssignment {
	base := []Estimate{
		{Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "typical"},
		{Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "trivial"},
		{Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "legacy"},
	}
	out := make([]EstimateAssignment, 0, len(descs))
	for i := range descs {
		e := base[i%len(base)]
		out = append(out, EstimateAssignment{TaskNumber: i + 1, Optimistic: e.Optimistic, MostLikely: e.MostLikely, Pessimistic: e.Pessimistic, Reasoning: e.Reasoning})
	}
	return out
}

func mustEstimate(t *testing.T, w *WBS, taskNumber int) Estimate {
	t.Helper()
	task, ok := w.TaskAt(taskNumber)
	if !ok || task.Estimate == nil {
		t.Fatalf("task number %d has no estimate", taskNumber)
	}
	return *task.Estimate
}

func TestSetEstimatesPopulatesEveryTaskUnapproved(t *testing.T) {
	w := approvedWBS(t, "Login API", "Login UI", "Session store")

	w.SetEstimates([]EstimateAssignment{
		{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "clean"},
		{TaskNumber: 2, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "trivial"},
		{TaskNumber: 3, Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "legacy"},
	})

	if w.EstimatesApproved() {
		t.Fatal("freshly generated estimates must be unapproved")
	}
	got := mustEstimate(t, w, 1)
	if got.Optimistic != 2 || got.MostLikely != 5 || got.Pessimistic != 13 || got.Reasoning != "clean" {
		t.Fatalf("task 1 estimate = %+v, want 2/5/13 clean", got)
	}
}

func TestSetEstimatesReplacesPriorEstimates(t *testing.T) {
	w := estimatedWBS(t, "Login API", "Login UI", "Session store")
	if err := w.OverrideEstimate(1, Estimate{Optimistic: 5, MostLikely: 8, Pessimistic: 13, Reasoning: "manual"}); err != nil {
		t.Fatalf("OverrideEstimate returned error: %v", err)
	}

	w.SetEstimates([]EstimateAssignment{
		{TaskNumber: 1, Optimistic: 5, MostLikely: 13, Pessimistic: 40, Reasoning: "rewrite"},
		{TaskNumber: 2, Optimistic: 3, MostLikely: 5, Pessimistic: 8, Reasoning: "validation"},
		{TaskNumber: 3, Optimistic: 8, MostLikely: 20, Pessimistic: 40, Reasoning: "migration"},
	})

	got := mustEstimate(t, w, 1)
	if got.MostLikely != 13 || got.Reasoning != "rewrite" {
		t.Fatalf("task 1 estimate = %+v, want the regenerated 5/13/40 rewrite (override replaced)", got)
	}
}

func TestTaskAtReturnsCopyOfEstimate(t *testing.T) {
	w := estimatedWBS(t, "Login API")

	task, _ := w.TaskAt(1)
	task.Estimate.MostLikely = 999

	if got := mustEstimate(t, w, 1); got.MostLikely == 999 {
		t.Fatal("internal estimate changed via returned copy")
	}
}

func TestOverrideEstimateUpdatesAndUnapproves(t *testing.T) {
	w := estimatedWBS(t, "Login API")
	if err := w.ApproveEstimates(); err != nil {
		t.Fatalf("ApproveEstimates returned error: %v", err)
	}

	if err := w.OverrideEstimate(1, Estimate{Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "team review"}); err != nil {
		t.Fatalf("OverrideEstimate returned error: %v", err)
	}
	got := mustEstimate(t, w, 1)
	if got.Optimistic != 3 || got.MostLikely != 8 || got.Pessimistic != 20 || got.Reasoning != "team review" {
		t.Fatalf("task 1 estimate = %+v, want 3/8/20 team review", got)
	}
	if w.EstimatesApproved() {
		t.Fatal("an override must revert approved estimates to unapproved")
	}
}

func TestOverrideEstimateRejectsUnknownTask(t *testing.T) {
	w := estimatedWBS(t, "Login API")
	if err := w.OverrideEstimate(9, Estimate{Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "x"}); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("override unknown task error = %v, want ErrTaskNotFound", err)
	}
}

func TestOverrideEstimateRejectsTaskWithNoEstimate(t *testing.T) {
	w := approvedWBS(t, "Login API")
	err := w.OverrideEstimate(1, Estimate{Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "x"})
	if !errors.Is(err, ErrNoEstimate) {
		t.Fatalf("override without estimate error = %v, want ErrNoEstimate", err)
	}
	if task, _ := w.TaskAt(1); task.Estimate != nil {
		t.Fatal("rejected override must not create an estimate")
	}
}

func TestOverrideEstimateRejectsNonIncreasingTriples(t *testing.T) {
	cases := []Estimate{
		{Optimistic: 8, MostLikely: 5, Pessimistic: 13, Reasoning: "r"},
		{Optimistic: 2, MostLikely: 20, Pessimistic: 13, Reasoning: "r"},
		{Optimistic: 5, MostLikely: 5, Pessimistic: 8, Reasoning: "r"},
		{Optimistic: 8, MostLikely: 13, Pessimistic: 13, Reasoning: "r"},
	}
	for _, c := range cases {
		w := estimatedWBS(t, "Login API")
		if err := w.OverrideEstimate(1, c); !errors.Is(err, ErrEstimatesNotIncreasing) {
			t.Fatalf("override %+v error = %v, want ErrEstimatesNotIncreasing", c, err)
		}
		if got := mustEstimate(t, w, 1); got.MostLikely != 5 {
			t.Fatalf("rejected override changed the estimate: %+v", got)
		}
	}
}

func TestOverrideEstimateRejectsOffScaleValues(t *testing.T) {
	cases := []Estimate{
		{Optimistic: 4, MostLikely: 5, Pessimistic: 13, Reasoning: "r"},
		{Optimistic: 21, MostLikely: 40, Pessimistic: 100, Reasoning: "r"},
		{Optimistic: -1, MostLikely: 5, Pessimistic: 13, Reasoning: "r"},
	}
	for _, c := range cases {
		w := estimatedWBS(t, "Login API")
		if err := w.OverrideEstimate(1, c); !errors.Is(err, ErrOffFibonacciScale) {
			t.Fatalf("override %+v error = %v, want ErrOffFibonacciScale", c, err)
		}
	}
}

func TestOverrideEstimateRejectsEmptyReasoning(t *testing.T) {
	w := estimatedWBS(t, "Login API")
	if err := w.OverrideEstimate(1, Estimate{Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "   "}); !errors.Is(err, ErrEmptyReasoning) {
		t.Fatalf("override empty reasoning error = %v, want ErrEmptyReasoning", err)
	}
}

func TestOverrideEstimateAcceptsScaleBoundaries(t *testing.T) {
	w := estimatedWBS(t, "Login API")
	if err := w.OverrideEstimate(1, Estimate{Optimistic: 0, MostLikely: 1, Pessimistic: 2, Reasoning: "low"}); err != nil {
		t.Fatalf("override 0/1/2 error = %v, want accepted", err)
	}
	if err := w.OverrideEstimate(1, Estimate{Optimistic: 8, MostLikely: 40, Pessimistic: 100, Reasoning: "high"}); err != nil {
		t.Fatalf("override 8/40/100 error = %v, want accepted", err)
	}
}

func TestApproveEstimatesRequiresGeneration(t *testing.T) {
	w := approvedWBS(t, "Login API")
	if err := w.ApproveEstimates(); !errors.Is(err, ErrEstimatesNotGenerated) {
		t.Fatalf("approve before generation error = %v, want ErrEstimatesNotGenerated", err)
	}
	if w.EstimatesApproved() {
		t.Fatal("estimates must stay unapproved when approval is rejected")
	}
}

func TestApproveEstimatesApprovesGeneratedSet(t *testing.T) {
	w := estimatedWBS(t, "Login API")
	if err := w.ApproveEstimates(); err != nil {
		t.Fatalf("ApproveEstimates returned error: %v", err)
	}
	if !w.EstimatesApproved() {
		t.Fatal("estimates must be approved after ApproveEstimates")
	}
}
