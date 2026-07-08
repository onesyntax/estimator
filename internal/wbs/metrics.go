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

// expected returns the full-precision expected value (expectedSixths/6).
func (v pertValues) expected() float64 { return float64(v.expectedSixths) / 6 }

// metrics rounds the intermediates into the emitted whole numbers. The relative
// standard deviation is derived from the full-precision SD and expected value,
// so a task whose SD rounds to 0 can still report a non-zero RSD.
func (v pertValues) metrics() Metrics {
	return Metrics{
		Expected:                  roundSixthsHalfUp(v.expectedSixths),
		StandardDeviation:         roundHalfUp(v.stdDev),
		RelativeStandardDeviation: roundHalfUp(v.stdDev / v.expected() * 100),
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
	v, ok := w.projectPert()
	if !ok {
		return Metrics{}, false
	}
	return v.metrics(), true
}

// projectPert accumulates the full-precision project rollup: the expected values
// sum (kept as an exact sixths numerator) and the variances add (SD =
// sqrt(sum of the task SDs squared)). ok is false when no task has an estimate.
// Both ProjectMetrics and the derived pricing strategy build on it.
func (w *WBS) projectPert() (pertValues, bool) {
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
		return pertValues{}, false
	}
	return pertValues{expectedSixths: sumExpectedSixths, stdDev: math.Sqrt(sumVariance)}, true
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

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-08T22:24:35+05:30","module_hash":"3b767a0d3fcccafbcdca5ed7f73ab4a67b69f81dc59988983f39e16fffc77f4c","functions":[{"id":"func/Estimate.pert","name":"Estimate.pert","line":29,"end_line":34,"hash":"fe17c2daa673d2faa1f6ee1c011da128c3165fd28e26baa90a8c17a7f718c9d4"},{"id":"func/pertValues.metrics","name":"pertValues.metrics","line":39,"end_line":46,"hash":"8f02d788454f1a2aa2dd883ecd4b7c2dc13398b5a597e624d1c8be689b802d6c"},{"id":"func/Estimate.Metrics","name":"Estimate.Metrics","line":49,"end_line":51,"hash":"95da750ebdd39e2bf7983acfb21d9310d6c83157994f2c200ea9a24317206666"},{"id":"func/WBS.ProjectMetrics","name":"WBS.ProjectMetrics","line":57,"end_line":74,"hash":"60e29746ea63f3c6b13d079f1b1751b8986971daf68b6724c6dfb961a5676892"},{"id":"func/roundSixthsHalfUp","name":"roundSixthsHalfUp","line":80,"end_line":82,"hash":"88ef8d1af285fddf8a1d2eb0ebc3726bc5b9e25ccbe1bce5829017faff419fd4"},{"id":"func/roundHalfUp","name":"roundHalfUp","line":87,"end_line":89,"hash":"1bb457d34bcd3795eef7674272f97ff3367b715bd581f90fae98e7ed5b2506c0"}]}
// mutate4go-manifest-end
