package wbs

import "strings"

// Estimate is a task's 3-point estimate. The three values are unitless points
// on the planning-poker Fibonacci scale and must be strictly increasing; the
// single Reasoning justifies all three points together and must be non-empty.
type Estimate struct {
	Optimistic  int
	MostLikely  int
	Pessimistic int
	Reasoning   string
}

// EstimateAssignment pairs a one-based task number with a 3-point estimate. It
// is the unit an EstimateProvider returns and the unit SetEstimates applies.
type EstimateAssignment struct {
	TaskNumber  int
	Optimistic  int
	MostLikely  int
	Pessimistic int
	Reasoning   string
}

// onFibonacciScale reports whether v is a value on the planning-poker deck: the
// only values an estimate may take. Classic Fibonacci numbers off the deck (21,
// 34, …) and negatives are rejected. The deck is inlined so each value is an
// executable, mutation-covered literal rather than an uncounted package var.
func onFibonacciScale(v int) bool {
	for _, s := range []int{0, 1, 2, 3, 5, 8, 13, 20, 40, 100} {
		if v == s {
			return true
		}
	}
	return false
}

// validate reports the first rule an estimate breaks: an off-scale value, a
// non-strictly-increasing triple, or a blank reasoning.
func (e Estimate) validate() error {
	for _, v := range []int{e.Optimistic, e.MostLikely, e.Pessimistic} {
		if !onFibonacciScale(v) {
			return ErrOffFibonacciScale
		}
	}
	if !(e.Optimistic < e.MostLikely && e.MostLikely < e.Pessimistic) {
		return ErrEstimatesNotIncreasing
	}
	if strings.TrimSpace(e.Reasoning) == "" {
		return ErrEmptyReasoning
	}
	return nil
}

// EstimatesApproved reports whether the current estimate set is approved.
func (w *WBS) EstimatesApproved() bool { return w.estimatesApproved }

// SetEstimates replaces every task's estimate with the given assignments (so
// re-generation discards prior AI suggestions and Tech-Lead overrides alike),
// marks estimates generated, and leaves them unapproved. Assignments for task
// numbers outside the current range are ignored.
func (w *WBS) SetEstimates(assignments []EstimateAssignment) {
	for i := range w.tasks {
		w.tasks[i].Estimate = nil
	}
	for _, a := range assignments {
		if a.TaskNumber < 1 || a.TaskNumber > len(w.tasks) {
			continue
		}
		w.tasks[a.TaskNumber-1].Estimate = &Estimate{
			Optimistic:  a.Optimistic,
			MostLikely:  a.MostLikely,
			Pessimistic: a.Pessimistic,
			Reasoning:   a.Reasoning,
		}
	}
	w.estimatesGenerated = true
	w.estimatesApproved = false
}

// OverrideEstimate replaces the estimate of the task at the one-based number
// with a validated estimate and reverts the set to unapproved. It fails with
// ErrTaskNotFound, ErrNoEstimate (the task has no estimate to override yet), or
// a validation error (ErrOffFibonacciScale, ErrEstimatesNotIncreasing,
// ErrEmptyReasoning).
func (w *WBS) OverrideEstimate(taskNumber int, estimate Estimate) error {
	if !w.taskExists(taskNumber) {
		return ErrTaskNotFound
	}
	if w.tasks[taskNumber-1].Estimate == nil {
		return ErrNoEstimate
	}
	estimate.Reasoning = strings.TrimSpace(estimate.Reasoning)
	if err := estimate.validate(); err != nil {
		return err
	}
	stored := estimate
	w.tasks[taskNumber-1].Estimate = &stored
	w.estimatesApproved = false
	return nil
}

// ApproveEstimates approves the current estimate set. It fails with
// ErrEstimatesNotGenerated when estimates have not been generated yet.
func (w *WBS) ApproveEstimates() error {
	if !w.estimatesGenerated {
		return ErrEstimatesNotGenerated
	}
	w.estimatesApproved = true
	return nil
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-08T23:58:54+05:30","module_hash":"9de1ef7e3433d94c7cd5319b02dff2a28bc57c1217debb5ba87bfeddcc8ee15b","functions":[{"id":"func/onFibonacciScale","name":"onFibonacciScale","line":29,"end_line":36,"hash":"4b68246da030781eef939a033fb3af56678e7128f7c2ca2096128a3f723d3ba9"},{"id":"func/Estimate.validate","name":"Estimate.validate","line":40,"end_line":53,"hash":"ee177b369f40046ad3c4b09bb954a6433e2489cbba5ce0be7d4d61a4fac695c5"},{"id":"func/WBS.EstimatesApproved","name":"WBS.EstimatesApproved","line":56,"end_line":56,"hash":"27ee6e4ddc96bc4641988cdafa88aa21e4e72acc00530d3fb2815a617ba29b20"},{"id":"func/WBS.SetEstimates","name":"WBS.SetEstimates","line":62,"end_line":79,"hash":"23be4479c7d343d47e6e240c2173cb9deadc46fcf53ee2bc42d746272cc7b517"},{"id":"func/WBS.OverrideEstimate","name":"WBS.OverrideEstimate","line":86,"end_line":101,"hash":"709c42390edcd786a7810690a66c6af873780b7cafb1df9061346bdd33dc6987"},{"id":"func/WBS.ApproveEstimates","name":"WBS.ApproveEstimates","line":105,"end_line":111,"hash":"12076491fba8d2e3c73515962c5eb7328cde0ab650a7ee1a4763b28cc6c435f3"}]}
// mutate4go-manifest-end
