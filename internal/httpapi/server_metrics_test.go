package httpapi

import (
	"testing"
)

// metricsOf reads a task's metrics object, or nil when the task carries none.
func metricsOf(t *testing.T, body map[string]any, taskNumber int) map[string]any {
	t.Helper()
	return taskFieldOf(t, body, taskNumber, "metrics")
}

// projectMetricsOf reads the WBS-level projectMetrics object, or nil.
func projectMetricsOf(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	return bodyFieldOf(t, body, "projectMetrics")
}

func assertMetrics(t *testing.T, got map[string]any, e, sd, rsd float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("metrics = null, want %v/%v/%v", e, sd, rsd)
	}
	if got["expected"] != e || got["standardDeviation"] != sd || got["relativeStandardDeviation"] != rsd {
		t.Fatalf("metrics = %v, want expected %v standardDeviation %v relativeStandardDeviation %v", got, e, sd, rsd)
	}
}

func TestMetricsAppearLiveOnUnapprovedSet(t *testing.T) {
	c := newClient(t, true)
	id := c.estimatedWBS() // 2/5/13, 1/2/3, 3/8/20, not approved

	_, body := c.get(id)
	if body["estimatesState"] != "unapproved" {
		t.Fatalf("estimatesState = %v, want unapproved", body["estimatesState"])
	}
	assertMetrics(t, metricsOf(t, body, 1), 6, 2, 31)
	assertMetrics(t, metricsOf(t, body, 2), 2, 0, 17)
	assertMetrics(t, metricsOf(t, body, 3), 9, 3, 31)
	assertMetrics(t, projectMetricsOf(t, body), 17, 3, 20)
}

func TestMetricsRecomputeAfterOverride(t *testing.T) {
	c := newClient(t, true)
	id := c.estimatedWBS()
	_, body := c.get(id)
	t1 := taskID(t, body, 1)

	c.overrideEstimate(id, t1, map[string]any{"optimistic": 5, "mostLikely": 8, "pessimistic": 13, "reasoning": "Manual across all points"})

	_, got := c.get(id)
	assertMetrics(t, metricsOf(t, got, 1), 8, 1, 16)
	// Project expected is 117/6 = 19.5, which rounds half-up to 20.
	assertMetrics(t, projectMetricsOf(t, got), 20, 3, 16)
}

func TestMetricsUnchangedByApproval(t *testing.T) {
	c := newClient(t, true)
	id := c.estimatedWBS()
	c.approveEstimates(id)

	_, got := c.get(id)
	if got["estimatesState"] != "approved" {
		t.Fatalf("estimatesState = %v, want approved", got["estimatesState"])
	}
	assertMetrics(t, metricsOf(t, got, 1), 6, 2, 31)
	assertMetrics(t, projectMetricsOf(t, got), 17, 3, 20)
}

func TestMetricsNullWithoutEstimates(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.approve(id)

	_, got := c.get(id)
	for i := 1; i <= 3; i++ {
		if m := metricsOf(t, got, i); m != nil {
			t.Fatalf("task %d metrics = %v, want null", i, m)
		}
	}
	if pm := projectMetricsOf(t, got); pm != nil {
		t.Fatalf("projectMetrics = %v, want null", pm)
	}
}

func TestMetricsPartialEstimates(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.approve(id)
	c.primeEstimates([]map[string]any{
		{"taskNumber": 1, "optimistic": 2, "mostLikely": 5, "pessimistic": 13, "reasoning": "clean"},
		{"taskNumber": 2, "optimistic": 1, "mostLikely": 2, "pessimistic": 3, "reasoning": "trivial"},
	})
	c.generateEstimates(id)

	_, got := c.get(id)
	if m := metricsOf(t, got, 3); m != nil {
		t.Fatalf("task 3 metrics = %v, want null (unestimated)", m)
	}
	assertMetrics(t, metricsOf(t, got, 1), 6, 2, 31)
	assertMetrics(t, projectMetricsOf(t, got), 8, 2, 24)
}
