//go:build hardening

// Hardening tests: added by the hardender to kill mutation survivors that the
// normal unit suite leaves alive. They live behind the `hardening` build tag so
// they stay out of the unit suite, coverage, CRAP, and DRY runs, and are
// exercised only by the language mutation tool via `-tags hardening`.
package wbs

import "testing"

// Kills SetEstimates `a.TaskNumber < 1 -> a.TaskNumber < 0`: task number 0 is out
// of range and must be skipped, not routed into the negative index tasks[-1].
// Under the mutant, `0 < 0` is false so the zero assignment is applied and
// tasks[-1] panics.
func TestHardeningSetEstimatesSkipsZeroTaskNumber(t *testing.T) {
	w := approvedWBS(t, "Login API", "Login UI")

	w.SetEstimates([]EstimateAssignment{
		{TaskNumber: 0, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "ignored"},
		{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "kept"},
	})

	if got := mustEstimate(t, w, 1); got.Reasoning != "kept" {
		t.Fatalf("task 1 estimate = %+v, want the in-range assignment applied", got)
	}
}

// Kills SetEstimates `< 1 || > len -> < 1 && > len`: an out-of-range high task
// number must be skipped. The `||` guard skips it; the `&&` mutant (never true
// for a single number) applies it and tasks[TaskNumber-1] panics past the end.
func TestHardeningSetEstimatesSkipsTaskNumberPastEnd(t *testing.T) {
	w := approvedWBS(t, "Login API", "Login UI")

	w.SetEstimates([]EstimateAssignment{
		{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "kept"},
		{TaskNumber: 99, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "ignored"},
	})

	if got := mustEstimate(t, w, 1); got.Reasoning != "kept" {
		t.Fatalf("task 1 estimate = %+v, want the in-range assignment applied", got)
	}
	if task, _ := w.TaskAt(2); task.Estimate != nil {
		t.Fatalf("task 2 estimate = %+v, want nil; the past-end assignment must not land anywhere", task.Estimate)
	}
}
