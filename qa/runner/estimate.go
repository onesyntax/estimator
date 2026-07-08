package main

import (
	"fmt"
	"net/http"
)

// registerEstimate registers the Module 3 cases (qa/estimation_qa_suite.md,
// mirroring features/estimate_generation.feature, estimate_override.feature and
// estimate_approval.feature). Each approved task carries a 3-point Estimate
// {optimistic, mostLikely, pessimistic, reasoning}; estimation requires an
// approved WBS, generation replaces every estimate, approval locks the set, and
// any later override (values or reasoning) reverts it to unapproved.

// estimateAssign is one primed estimate: which task (1-based position) it lands
// on and its O/M/P plus the single all-points reasoning. It mirrors one entry of
// the /qa/ai/next-estimates affordance body.
type estimateAssign struct {
	taskNumber                          int
	optimistic, mostLikely, pessimistic int
	reasoning                           string
}

// standardEstimates is the shared prime used across the suite: task 1 -> 2/5/13,
// task 2 -> 1/2/3, task 3 -> 3/8/20, each with its all-points reasoning.
func standardEstimates() []estimateAssign {
	return []estimateAssign{
		{1, 2, 5, 13, "Clean best, typical likely, legacy worst"},
		{2, 1, 2, 3, "Trivial across all points"},
		{3, 3, 8, 20, "Legacy widens the range"},
	}
}

// primeEstimates seeds the estimates the next generation returns via the QA
// affordance, asserting the documented 204.
func primeEstimates(t *T, estimates []estimateAssign) {
	payload := make([]map[string]any, len(estimates))
	for i, e := range estimates {
		payload[i] = map[string]any{
			"taskNumber":  e.taskNumber,
			"optimistic":  e.optimistic,
			"mostLikely":  e.mostLikely,
			"pessimistic": e.pessimistic,
			"reasoning":   e.reasoning,
		}
	}
	r := postJSON("/qa/ai/next-estimates", map[string]any{"estimates": payload})
	t.eq("prime estimates status", r.status, http.StatusNoContent)
}

// estimatePath builds the override path for a task's estimate.
func estimatePath(id, taskID string) string {
	return "/wbs/" + id + "/tasks/" + taskID + "/estimate"
}

// overrideBody builds an override payload from an O/M/P triple and reasoning.
func overrideBody(o, m, p int, reasoning string) map[string]any {
	return map[string]any{"optimistic": o, "mostLikely": m, "pessimistic": p, "reasoning": reasoning}
}

// estimatedWBS is the shared override/approval setup: the standard three-task
// WBS, approved, with estimates generated to task 1 = 2/5/13, task 2 = 1/2/3,
// task 3 = 3/8/20. It returns the WBS id.
func estimatedWBS(t *T) string {
	id := approvedWBS(t)
	primeEstimates(t, standardEstimates())
	r := postJSON("/wbs/"+id+"/estimates/generate", nil)
	t.eq("setup generate estimates status", r.status, http.StatusOK)
	return id
}

// estimateOf returns the estimate on task number n (1-based) in GET order, or nil.
func estimateOf(t *T, id string, n int) *estimateView {
	tk, ok := taskAt(t, id, n)
	if !ok {
		return nil
	}
	return tk.Estimate
}

// estimateIs asserts task number n carries the given O/M/P and reasoning.
func (t *T) estimateIs(id string, n, o, m, p int, reasoning string) {
	e := estimateOf(t, id, n)
	if e == nil {
		t.fail("task %d estimate = null, want %d/%d/%d", n, o, m, p)
		return
	}
	t.eq(fmt.Sprintf("task %d optimistic", n), e.Optimistic, o)
	t.eq(fmt.Sprintf("task %d mostLikely", n), e.MostLikely, m)
	t.eq(fmt.Sprintf("task %d pessimistic", n), e.Pessimistic, p)
	t.eq(fmt.Sprintf("task %d reasoning", n), e.Reasoning, reasoning)
}

// noEstimate asserts task number n has no estimate (JSON null).
func (t *T) noEstimate(id string, n int) {
	if e := estimateOf(t, id, n); e != nil {
		t.fail("task %d estimate = %d/%d/%d, want null", n, e.Optimistic, e.MostLikely, e.Pessimistic)
	}
}

// estimatesStateIs asserts the WBS-level estimatesState label.
func (t *T) estimatesStateIs(id, want string) {
	t.eq("estimatesState", getJSON("/wbs/"+id).view().EstimatesState, want)
}

func registerEstimate() {
	// --- Estimate Generation (estimate_generation.feature) -----------------

	// QA-EST-1 — Generate estimates on an approved WBS.
	register("QA-EST-1 generate estimates on approved wbs", func(t *T) {
		id := approvedWBS(t)
		primeEstimates(t, standardEstimates())
		r := postJSON("/wbs/"+id+"/estimates/generate", nil)
		t.eq("generate status", r.status, http.StatusOK)
		t.estimateIs(id, 1, 2, 5, 13, "Clean best, typical likely, legacy worst")
		t.estimateIs(id, 2, 1, 2, 3, "Trivial across all points")
		t.estimateIs(id, 3, 3, 8, 20, "Legacy widens the range")
		t.estimatesStateIs(id, "unapproved")
	})

	// QA-EST-2 — Reject generation on an unapproved WBS.
	register("QA-EST-2 reject generation on unapproved wbs", func(t *T) {
		id := setupWBS(t) // not approved
		primeEstimates(t, standardEstimates())
		r := postJSON("/wbs/"+id+"/estimates/generate", nil)
		t.errorCase(r, http.StatusConflict, "WBS must be approved before estimation")
		for n := 1; n <= 3; n++ {
			t.noEstimate(id, n)
		}
		// Like QA-FLAG-2: the rejected generation ran before the provider was
		// called, so it did not consume the primed estimates, leaving a stranded
		// entry in the mock's shared FIFO queue. Drain it through the UI — approve
		// this now-discarded WBS and generate once — so it cannot leak into the
		// next case.
		t.eq("drain approve status", postJSON("/wbs/"+id+"/approve", nil).status, http.StatusOK)
		t.eq("drain generate status", postJSON("/wbs/"+id+"/estimates/generate", nil).status, http.StatusOK)
	})

	// QA-EST-3 — Re-generation replaces all estimates (including overrides).
	register("QA-EST-3 re-generation replaces overrides", func(t *T) {
		id := estimatedWBS(t)
		// A valid override (aligns with estimate_generation.feature Scenario 3,
		// which overrides task 1 to 5/8/13; the QA suite's earlier 8/8/8 was a
		// typo — that triple is not strictly increasing and is rejected).
		ovr := putJSON(estimatePath(id, taskID(t, id, 1)), overrideBody(5, 8, 13, "Manual across all points"))
		t.eq("override status", ovr.status, http.StatusOK)
		t.estimateIs(id, 1, 5, 8, 13, "Manual across all points")

		primeEstimates(t, []estimateAssign{
			{1, 5, 13, 40, "Rewrite lifts all points"},
			{2, 3, 5, 8, "Validation adds spread"},
			{3, 8, 20, 40, "Migration keeps wide range"},
		})
		r := postJSON("/wbs/"+id+"/estimates/generate", nil)
		t.eq("regenerate status", r.status, http.StatusOK)
		t.estimateIs(id, 1, 5, 13, 40, "Rewrite lifts all points")
		t.estimateIs(id, 2, 3, 5, 8, "Validation adds spread")
		t.estimateIs(id, 3, 8, 20, 40, "Migration keeps wide range")
	})

	// QA-EST-4 — Generate on a WBS that does not exist.
	register("QA-EST-4 generate on missing wbs", func(t *T) {
		r := postJSON("/wbs/does-not-exist/estimates/generate", nil)
		t.errorCase(r, http.StatusNotFound, "WBS not found")
	})

	// --- Estimate Override (estimate_override.feature) ---------------------

	// QA-OVR-1 — Accept valid overrides, including the 0 lower and 100 upper
	// scale boundaries. Each reverts (keeps) the estimate set unapproved.
	register("QA-OVR-1 accept valid override", func(t *T) {
		id := estimatedWBS(t)
		type ovrCase struct {
			task      int
			o, m, p   int
			reasoning string
		}
		for _, c := range []ovrCase{
			{1, 3, 8, 20, "Team set O M and P together"},
			{2, 0, 1, 2, "All points minimal for trivial work"},
			{3, 8, 40, 100, "Rewrite pushes every point higher"},
		} {
			r := putJSON(estimatePath(id, taskID(t, id, c.task)), overrideBody(c.o, c.m, c.p, c.reasoning))
			t.eq(fmt.Sprintf("override task %d status", c.task), r.status, http.StatusOK)
			t.estimateIs(id, c.task, c.o, c.m, c.p, c.reasoning)
			t.estimatesStateIs(id, "unapproved")
		}
	})

	// QA-OVR-2 — Reject invalid overrides, each isolating a single reason; task 1
	// stays at its generated 2/5/13 after every rejection.
	register("QA-OVR-2 reject invalid override", func(t *T) {
		id := estimatedWBS(t)
		task1 := taskID(t, id, 1)
		notIncreasing := "estimate values must be strictly increasing"
		offScale := "estimate value is off the Fibonacci scale"
		type badCase struct {
			o, m, p int
			reason  string
		}
		for _, c := range []badCase{
			{8, 5, 13, notIncreasing},  // O > M
			{2, 20, 13, notIncreasing}, // M > P
			{5, 5, 8, notIncreasing},   // O = M
			{8, 13, 13, notIncreasing}, // M = P
			{4, 5, 13, offScale},       // 4 not on the deck
			{21, 40, 100, offScale},    // 21 is classic Fibonacci but off the deck
			{-1, 5, 13, offScale},      // negative
		} {
			r := putJSON(estimatePath(id, task1), overrideBody(c.o, c.m, c.p, "Adjusted for risk"))
			t.errorCase(r, http.StatusUnprocessableEntity, c.reason)
			t.estimateIs(id, 1, 2, 5, 13, "Clean best, typical likely, legacy worst")
		}
		// Valid values but an empty reasoning isolates the reasoning rule.
		empty := putJSON(estimatePath(id, task1), overrideBody(3, 8, 20, ""))
		t.errorCase(empty, http.StatusUnprocessableEntity, "reasoning is empty")
		t.estimateIs(id, 1, 2, 5, 13, "Clean best, typical likely, legacy worst")
	})

	// QA-OVR-3 — Reject overriding a task that does not exist.
	register("QA-OVR-3 reject override of missing task", func(t *T) {
		id := estimatedWBS(t)
		r := putJSON("/wbs/"+id+"/tasks/nonexistent/estimate", overrideBody(1, 2, 3, "New estimate for all points"))
		t.errorCase(r, http.StatusNotFound, "task not found")
	})

	// QA-OVR-4 — Reject overriding before estimates are generated.
	register("QA-OVR-4 reject override before generation", func(t *T) {
		id := approvedWBS(t) // estimates not generated
		r := putJSON(estimatePath(id, taskID(t, id, 1)), overrideBody(1, 2, 3, "New estimate for all points"))
		t.errorCase(r, http.StatusConflict, "task has no estimate to override")
		t.noEstimate(id, 1)
	})

	// --- Estimate Approval (estimate_approval.feature) ---------------------

	// QA-APR-1 — Approve generated estimates.
	register("QA-APR-1 approve generated estimates", func(t *T) {
		id := estimatedWBS(t)
		r := postJSON("/wbs/"+id+"/estimates/approve", nil)
		t.eq("approve status", r.status, http.StatusOK)
		t.estimatesStateIs(id, "approved")
	})

	// QA-APR-2 — Reject approval before generation.
	register("QA-APR-2 reject approval before generation", func(t *T) {
		id := approvedWBS(t) // estimates not generated
		r := postJSON("/wbs/"+id+"/estimates/approve", nil)
		t.errorCase(r, http.StatusConflict, "estimates have not been generated")
		t.estimatesStateIs(id, "unapproved")
	})

	// QA-APR-3 — An override after approval reverts to unapproved (values, and a
	// reasoning-only change).
	register("QA-APR-3 override after approval reverts to unapproved", func(t *T) {
		id := estimatedWBS(t)
		t.eq("approve status", postJSON("/wbs/"+id+"/estimates/approve", nil).status, http.StatusOK)
		t.estimatesStateIs(id, "approved")
		ovr := putJSON(estimatePath(id, taskID(t, id, 1)), overrideBody(3, 8, 20, "Team review"))
		t.eq("override status", ovr.status, http.StatusOK)
		t.estimatesStateIs(id, "unapproved")

		// A reasoning-only change (same 1/2/3 values, new reasoning) also reverts.
		id2 := estimatedWBS(t)
		t.eq("approve status 2", postJSON("/wbs/"+id2+"/estimates/approve", nil).status, http.StatusOK)
		t.estimatesStateIs(id2, "approved")
		reOnly := putJSON(estimatePath(id2, taskID(t, id2, 2)), overrideBody(1, 2, 3, "Still trivial"))
		t.eq("reasoning-only override status", reOnly.status, http.StatusOK)
		t.estimatesStateIs(id2, "unapproved")
	})

	// QA-APR-4 — Re-approve after an override.
	register("QA-APR-4 re-approve after override", func(t *T) {
		id := estimatedWBS(t)
		t.eq("approve status", postJSON("/wbs/"+id+"/estimates/approve", nil).status, http.StatusOK)
		ovr := putJSON(estimatePath(id, taskID(t, id, 1)), overrideBody(3, 8, 20, "Team review"))
		t.eq("override status", ovr.status, http.StatusOK)
		t.estimatesStateIs(id, "unapproved")
		r := postJSON("/wbs/"+id+"/estimates/approve", nil)
		t.eq("re-approve status", r.status, http.StatusOK)
		t.estimatesStateIs(id, "approved")
		t.estimateIs(id, 1, 3, 8, 20, "Team review")
	})
}
