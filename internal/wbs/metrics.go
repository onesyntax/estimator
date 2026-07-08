package wbs

import "math"

// Metrics are the whole-number PERT metrics derived from a single estimate or a
// project rollup: the expected value, the standard deviation of that estimate,
// and the relative standard deviation (SD as a percentage of the expected
// value). Every field is rounded half-up from full-precision intermediates.
type Metrics struct {
	Expected                  int
	StandardDeviation         int
	RelativeStandardDeviation int
}

// pertValues holds the full-precision PERT intermediates before rounding. The
// expected value is carried as an exact sixths numerator (expected =
// expectedSixths/6): estimates are whole numbers so every task expected is an
// exact multiple of 1/6, and summing the numerators keeps the project expected
// exact — so a documented half-up tie such as 117/6 = 19.5 rounds up
// deterministically instead of drifting on floating-point error. The standard
// deviation is a float because summed variances take a square root.
type pertValues struct {
	expectedSixths int
	stdDev         float64
}

// pert computes the full-precision PERT intermediates of a single 3-point
// estimate: expected = (O + 4M + P)/6 and standard deviation = (P - O)/6.
func (e Estimate) pert() pertValues {
	return pertValues{
		expectedSixths: e.Optimistic + 4*e.MostLikely + e.Pessimistic,
		stdDev:         float64(e.Pessimistic-e.Optimistic) / 6,
	}
}

// metrics rounds the intermediates into the emitted whole numbers. The relative
// standard deviation is derived from the full-precision SD and expected value,
// so a task whose SD rounds to 0 can still report a non-zero RSD.
func (v pertValues) metrics() Metrics {
	expected := float64(v.expectedSixths) / 6
	return Metrics{
		Expected:                  roundSixthsHalfUp(v.expectedSixths),
		StandardDeviation:         roundHalfUp(v.stdDev),
		RelativeStandardDeviation: roundHalfUp(v.stdDev / expected * 100),
	}
}

// Metrics returns the per-task PERT metrics for this estimate.
func (e Estimate) Metrics() Metrics {
	return e.pert().metrics()
}

// ProjectMetrics returns the PERT rollup over every task that currently has an
// estimate: expected values sum and variances add, so the project standard
// deviation is sqrt(sum of the task SDs squared). ok is false when no task has
// an estimate, leaving no rollup to report.
func (w *WBS) ProjectMetrics() (Metrics, bool) {
	sumExpectedSixths := 0
	var sumVariance float64
	estimated := false
	for _, t := range w.tasks {
		if t.Estimate == nil {
			continue
		}
		estimated = true
		v := t.Estimate.pert()
		sumExpectedSixths += v.expectedSixths
		sumVariance += v.stdDev * v.stdDev
	}
	if !estimated {
		return Metrics{}, false
	}
	return pertValues{expectedSixths: sumExpectedSixths, stdDev: math.Sqrt(sumVariance)}.metrics(), true
}

// roundSixthsHalfUp rounds n/6 to the nearest whole number, rounding a half up.
// It uses integer arithmetic so exact ties never fall victim to floating-point
// error: floor(n/6 + 1/2) = (2n + 6)/12 for the non-negative n that PERT
// expected values always are.
func roundSixthsHalfUp(n int) int {
	return (2*n + 6) / 12
}

// roundHalfUp rounds a non-negative value to the nearest whole number, rounding
// a half up. Every PERT metric is non-negative because estimates are strictly
// increasing, so SD and expected are positive.
func roundHalfUp(x float64) int {
	return int(math.Floor(x + 0.5))
}
