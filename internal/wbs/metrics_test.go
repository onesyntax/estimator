package wbs

import "testing"

// est builds a valid estimate; the reasoning is irrelevant to the metrics.
func est(o, m, p int) Estimate {
	return Estimate{Optimistic: o, MostLikely: m, Pessimistic: p, Reasoning: "r"}
}

func TestEstimateMetricsPerTask(t *testing.T) {
	cases := []struct {
		name string
		e    Estimate
		want Metrics
	}{
		// E = 35/6 = 5.83 -> 6, SD = 11/6 = 1.83 -> 2, RSD = 31.43 -> 31.
		{"rounds-up", est(2, 5, 13), Metrics{Expected: 6, StandardDeviation: 2, RelativeStandardDeviation: 31}},
		// SD = 2/6 = 0.33 rounds to 0, yet RSD (from the full-precision SD) is 17.
		{"sd-rounds-to-zero", est(1, 2, 3), Metrics{Expected: 2, StandardDeviation: 0, RelativeStandardDeviation: 17}},
		// E = 55/6 = 9.17 -> 9, SD = 17/6 = 2.83 -> 3, RSD = 30.9 -> 31.
		{"mid-scale", est(3, 8, 20), Metrics{Expected: 9, StandardDeviation: 3, RelativeStandardDeviation: 31}},
		// E = 9/6 = 1.5 rounds half-up to 2, SD = 5/6 = 0.83 -> 1, RSD = 55.56 -> 56.
		{"half-up-and-small", est(0, 1, 5), Metrics{Expected: 2, StandardDeviation: 1, RelativeStandardDeviation: 56}},
		// E = 50/6 = 8.33 -> 8, SD = 8/6 = 1.33 -> 1, RSD = 16.0 -> 16.
		{"override", est(5, 8, 13), Metrics{Expected: 8, StandardDeviation: 1, RelativeStandardDeviation: 16}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.e.Metrics(); got != c.want {
				t.Fatalf("Metrics(%+v) = %+v, want %+v", c.e, got, c.want)
			}
		})
	}
}

func TestProjectMetricsAddsVariances(t *testing.T) {
	w := approvedWBS(t, "a", "b", "c")
	w.SetEstimates([]EstimateAssignment{
		{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "r"},
		{TaskNumber: 2, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "r"},
		{TaskNumber: 3, Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "r"},
	})

	// E = 102/6 = 17; SD = sqrt(11.5) = 3.39 -> 3 (NOT the SD sum 2+0+3=5);
	// RSD = 19.95 -> 20.
	got, ok := w.ProjectMetrics()
	if !ok {
		t.Fatal("ProjectMetrics ok = false, want true for an estimated set")
	}
	if want := (Metrics{Expected: 17, StandardDeviation: 3, RelativeStandardDeviation: 20}); got != want {
		t.Fatalf("ProjectMetrics = %+v, want %+v", got, want)
	}
}

func TestProjectMetricsHalfUpTie(t *testing.T) {
	w := approvedWBS(t, "a", "b", "c")
	w.SetEstimates([]EstimateAssignment{
		{TaskNumber: 1, Optimistic: 5, MostLikely: 8, Pessimistic: 13, Reasoning: "r"},
		{TaskNumber: 2, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "r"},
		{TaskNumber: 3, Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "r"},
	})

	// E = 117/6 = 19.5 exactly, which must round half-up to 20 (not 19).
	got, ok := w.ProjectMetrics()
	if !ok {
		t.Fatal("ProjectMetrics ok = false, want true")
	}
	if want := (Metrics{Expected: 20, StandardDeviation: 3, RelativeStandardDeviation: 16}); got != want {
		t.Fatalf("ProjectMetrics = %+v, want %+v (19.5 must round up)", got, want)
	}
}

func TestProjectMetricsCoversOnlyEstimatedTasks(t *testing.T) {
	w := approvedWBS(t, "a", "b", "c")
	// Only tasks 1 and 2 are estimated; task 3 is left out.
	w.SetEstimates([]EstimateAssignment{
		{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "r"},
		{TaskNumber: 2, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "r"},
	})

	// Rollup over tasks 1-2 only: E = 47/6 = 7.83 -> 8; SD = sqrt(125/36) = 1.86
	// -> 2; RSD = 23.79 -> 24.
	got, ok := w.ProjectMetrics()
	if !ok {
		t.Fatal("ProjectMetrics ok = false, want true")
	}
	if want := (Metrics{Expected: 8, StandardDeviation: 2, RelativeStandardDeviation: 24}); got != want {
		t.Fatalf("ProjectMetrics = %+v, want %+v", got, want)
	}
}

func TestProjectMetricsSingleTaskEqualsThatTask(t *testing.T) {
	w := approvedWBS(t, "a", "b", "c")
	w.SetEstimates([]EstimateAssignment{
		{TaskNumber: 1, Optimistic: 0, MostLikely: 1, Pessimistic: 5, Reasoning: "r"},
	})

	// With one estimated task, sqrt(SD^2) = SD, so the rollup equals task 1.
	got, ok := w.ProjectMetrics()
	if !ok {
		t.Fatal("ProjectMetrics ok = false, want true")
	}
	if want := (Metrics{Expected: 2, StandardDeviation: 1, RelativeStandardDeviation: 56}); got != want {
		t.Fatalf("ProjectMetrics = %+v, want %+v", got, want)
	}
}

func TestProjectMetricsNoneWhenNoEstimates(t *testing.T) {
	w := approvedWBS(t, "a", "b", "c")

	if got, ok := w.ProjectMetrics(); ok {
		t.Fatalf("ProjectMetrics ok = true (%+v), want false when no task is estimated", got)
	}
}
