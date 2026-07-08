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

// fibonacciScale is the planning-poker deck: the only values an estimate may
// take. Classic Fibonacci numbers off the deck (21, 34, …) and negatives are
// rejected.
var fibonacciScale = map[int]bool{0: true, 1: true, 2: true, 3: true, 5: true, 8: true, 13: true, 20: true, 40: true, 100: true}

// validate reports the first rule an estimate breaks: an off-scale value, a
// non-strictly-increasing triple, or a blank reasoning.
func (e Estimate) validate() error {
	for _, v := range []int{e.Optimistic, e.MostLikely, e.Pessimistic} {
		if !fibonacciScale[v] {
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
