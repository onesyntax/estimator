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
// {"version":1,"tested_at":"2026-07-09T12:56:35+05:30","module_hash":"36bf750ac9dd5d95d0fc7355b842cf8acbd9b23253e0e2f6a390c63b18d494fa","functions":[{"id":"func/Estimate.pert","name":"Estimate.pert","line":29,"end_line":34,"hash":"fe17c2daa673d2faa1f6ee1c011da128c3165fd28e26baa90a8c17a7f718c9d4"},{"id":"func/pertValues.expected","name":"pertValues.expected","line":37,"end_line":37,"hash":"6f724f00931b093ef1daf3f6519b9f4481578643361f7165e39545bb1f8ab6a2"},{"id":"func/pertValues.metrics","name":"pertValues.metrics","line":42,"end_line":48,"hash":"ab63fbc20c1d92ff95c5187d9182ad4037a53b67eb197bc9c18bf52c1c10c1e3"},{"id":"func/Estimate.Metrics","name":"Estimate.Metrics","line":51,"end_line":53,"hash":"95da750ebdd39e2bf7983acfb21d9310d6c83157994f2c200ea9a24317206666"},{"id":"func/WBS.ProjectMetrics","name":"WBS.ProjectMetrics","line":59,"end_line":65,"hash":"aed0668dc8a8714cfe6485cacbac062d8b5c05b58d8d04763d1b86f729dcc73c"},{"id":"func/WBS.projectPert","name":"WBS.projectPert","line":71,"end_line":88,"hash":"6daec0db1a51228b12bd5e11ee78e3f561371d629a1cad271a5c5c696b539d95"},{"id":"func/roundSixthsHalfUp","name":"roundSixthsHalfUp","line":94,"end_line":96,"hash":"88ef8d1af285fddf8a1d2eb0ebc3726bc5b9e25ccbe1bce5829017faff419fd4"},{"id":"func/roundHalfUp","name":"roundHalfUp","line":101,"end_line":103,"hash":"1bb457d34bcd3795eef7674272f97ff3367b715bd581f90fae98e7ed5b2506c0"}]}
// mutate4go-manifest-end
