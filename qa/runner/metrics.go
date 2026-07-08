package main

import (
	"fmt"
	"net/http"
)

// registerMetrics registers the Module 4 cases (qa/metrics_qa_suite.md, mirroring
// features/estimate_metrics.feature). Module 4 adds derived PERT metrics to the
// WBS view: each task with an estimate carries a `metrics` object and the WBS
// carries a `projectMetrics` rollup. Metrics are computed deterministically by
// the backend from the current estimate set and appear live — whether the set is
// unapproved or approved — with no new endpoint and no new stored state. Every
// emitted value is a whole number rounded half-up. QA drives estimates exactly as
// in Module 3 and only reads the new fields from GET /wbs/{id}.

// metricsView is the derived PERT metrics as GET /wbs/{id} renders them, both
// per-task (`metrics`) and for the WBS rollup (`projectMetrics`); nil (JSON null)
// when there is nothing to report — a task with no estimate, or a WBS where no
// task is estimated.
type metricsView struct {
	Expected                  int `json:"expected"`
	StandardDeviation         int `json:"standardDeviation"`
	RelativeStandardDeviation int `json:"relativeStandardDeviation"`
}

// metricsOf returns the metrics on task number n (1-based) in GET order, or nil.
func metricsOf(t *T, id string, n int) *metricsView {
	tk, ok := taskAt(t, id, n)
	if !ok {
		return nil
	}
	return tk.Metrics
}

// projectMetricsOf returns the WBS-level projectMetrics rollup, or nil.
func projectMetricsOf(id string) *metricsView {
	return getJSON("/wbs/" + id).view().ProjectMetrics
}

// metricsAre asserts task number n carries the given expected, standard deviation
// and relative standard deviation.
func (t *T) metricsAre(id string, n, e, sd, rsd int) {
	m := metricsOf(t, id, n)
	if m == nil {
		t.fail("task %d metrics = null, want %d/%d/%d", n, e, sd, rsd)
		return
	}
	t.eq(fmt.Sprintf("task %d expected", n), m.Expected, e)
	t.eq(fmt.Sprintf("task %d standardDeviation", n), m.StandardDeviation, sd)
	t.eq(fmt.Sprintf("task %d relativeStandardDeviation", n), m.RelativeStandardDeviation, rsd)
}

// noMetrics asserts task number n has no metrics (JSON null).
func (t *T) noMetrics(id string, n int) {
	if m := metricsOf(t, id, n); m != nil {
		t.fail("task %d metrics = %d/%d/%d, want null", n, m.Expected, m.StandardDeviation, m.RelativeStandardDeviation)
	}
}

// projectMetricsAre asserts the WBS rollup carries the given expected, standard
// deviation and relative standard deviation.
func (t *T) projectMetricsAre(id string, e, sd, rsd int) {
	m := projectMetricsOf(id)
	if m == nil {
		t.fail("projectMetrics = null, want %d/%d/%d", e, sd, rsd)
		return
	}
	t.eq("project expected", m.Expected, e)
	t.eq("project standardDeviation", m.StandardDeviation, sd)
	t.eq("project relativeStandardDeviation", m.RelativeStandardDeviation, rsd)
}

// noProjectMetrics asserts the WBS has no rollup (JSON null).
func (t *T) noProjectMetrics(id string) {
	if m := projectMetricsOf(id); m != nil {
		t.fail("projectMetrics = %d/%d/%d, want null", m.Expected, m.StandardDeviation, m.RelativeStandardDeviation)
	}
}

// generatedEstimates is the shared Module 4 setup: an approved WBS with the given
// estimates primed and generated (but not approved). It returns the WBS id. The
// generate consumes the primed estimates, so it leaves no stranded entry in the
// mock's FIFO queue.
func generatedEstimates(t *T, estimates []estimateAssign) string {
	id := approvedWBS(t)
	primeEstimates(t, estimates)
	r := postJSON("/wbs/"+id+"/estimates/generate", nil)
	t.eq("setup generate estimates status", r.status, http.StatusOK)
	return id
}

func registerMetrics() {
	// --- Estimate Metrics (estimate_metrics.feature) -----------------------

	// QA-MET-1 — Per-task and project metrics appear live on an unapproved set.
	register("QA-MET-1 metrics appear live on an unapproved set", func(t *T) {
		id := generatedEstimates(t, standardEstimates())
		t.estimatesStateIs(id, "unapproved")
		t.metricsAre(id, 1, 6, 2, 31)
		t.metricsAre(id, 2, 2, 0, 17) // SD rounds 0.33 -> 0 while RSD is 17
		t.metricsAre(id, 3, 9, 3, 31)
		t.projectMetricsAre(id, 17, 3, 20) // E 102/6=17; SD sqrt(11.5)=3.39->3; RSD 19.95->20
	})

	// QA-MET-2 — An override recomputes metrics immediately (half-up tie). From
	// QA-MET-1's generated-but-unapproved set, override task 1 to 5/8/13.
	register("QA-MET-2 override recomputes metrics immediately", func(t *T) {
		id := generatedEstimates(t, standardEstimates())
		ovr := putJSON(estimatePath(id, taskID(t, id, 1)), overrideBody(5, 8, 13, "Manual across all points"))
		t.eq("override status", ovr.status, http.StatusOK)
		t.estimatesStateIs(id, "unapproved")
		t.metricsAre(id, 1, 8, 1, 16)
		t.projectMetricsAre(id, 20, 3, 16) // project expected 117/6=19.5 rounds half-up to 20
	})

	// QA-MET-3 — Metrics are unchanged by approval: the same numbers as QA-MET-1
	// on an approved set.
	register("QA-MET-3 metrics unchanged by approval", func(t *T) {
		id := generatedEstimates(t, standardEstimates())
		t.eq("approve estimates status", postJSON("/wbs/"+id+"/estimates/approve", nil).status, http.StatusOK)
		t.estimatesStateIs(id, "approved")
		t.metricsAre(id, 1, 6, 2, 31)
		t.metricsAre(id, 2, 2, 0, 17)
		t.metricsAre(id, 3, 9, 3, 31)
		t.projectMetricsAre(id, 17, 3, 20)
	})

	// QA-MET-4 — No estimates ⇒ no metrics at either level.
	register("QA-MET-4 no estimates means no metrics", func(t *T) {
		id := approvedWBS(t) // estimates not generated
		for n := 1; n <= 3; n++ {
			t.noMetrics(id, n)
		}
		t.noProjectMetrics(id)
	})

	// QA-MET-5 — Partial estimates: the unestimated task has no metrics; the
	// rollup covers only the estimated tasks.
	register("QA-MET-5 partial estimates roll up the estimated tasks", func(t *T) {
		id := generatedEstimates(t, []estimateAssign{
			{1, 2, 5, 13, "Clean best, typical likely, legacy worst"},
			{2, 1, 2, 3, "Trivial across all points"},
		})
		t.noMetrics(id, 3) // task 3 was left unestimated
		t.metricsAre(id, 1, 6, 2, 31)
		t.projectMetricsAre(id, 8, 2, 24) // E 47/6=7.83->8; SD sqrt(125/36)=1.86->2; RSD 23.79->24
	})

	// QA-MET-6 — Rounding half-up and the single-task rollup identity: one
	// estimated task 0/1/5 makes the rollup equal that task (sqrt(SD^2)=SD).
	register("QA-MET-6 half-up rounding and single-task rollup identity", func(t *T) {
		id := generatedEstimates(t, []estimateAssign{
			{1, 0, 1, 5, "Best case near zero, wide worst case"},
		})
		t.metricsAre(id, 1, 2, 1, 56)     // E 9/6=1.5->2; SD 5/6=0.83->1; RSD 55.56->56
		t.projectMetricsAre(id, 2, 1, 56) // one estimated task: rollup equals that task
	})
}
