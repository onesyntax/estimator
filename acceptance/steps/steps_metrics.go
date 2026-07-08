package steps

import (
	"fmt"

	"estimation/acceptance/runtime"
	"estimation/internal/wbs"
)

// registerMetricsAssertions registers the steps that assert on the derived
// per-task and project PERT metrics.
func registerMetricsAssertions(reg *runtime.Registry) {
	reg.Step(`^task number (\d+) has metrics expected (-?\d+) standard deviation (-?\d+) relative standard deviation (-?\d+)$`,
		func(wd any, a []string) error {
			e, err := estimateOf(w(wd), a[0])
			if err != nil {
				return err
			}
			if e == nil {
				return fmt.Errorf("task number %s has no metrics", a[0])
			}
			return expectMetrics(e.Metrics(), a[1], a[2], a[3])
		})

	reg.Step(`^task number (\d+) has no metrics$`,
		func(wd any, a []string) error {
			e, err := estimateOf(w(wd), a[0])
			if err != nil {
				return err
			}
			if e != nil {
				return fmt.Errorf("task number %s has metrics %+v, want none", a[0], e.Metrics())
			}
			return nil
		})

	reg.Step(`^the project metrics are expected (-?\d+) standard deviation (-?\d+) relative standard deviation (-?\d+)$`,
		func(wd any, a []string) error {
			cur, err := currentWBS(w(wd))
			if err != nil {
				return err
			}
			m, ok := cur.ProjectMetrics()
			if !ok {
				return fmt.Errorf("project has no metrics, want %s/%s/%s", a[0], a[1], a[2])
			}
			return expectMetrics(m, a[0], a[1], a[2])
		})

	reg.Step(`^the project has no metrics$`,
		func(wd any, _ []string) error {
			cur, err := currentWBS(w(wd))
			if err != nil {
				return err
			}
			if m, ok := cur.ProjectMetrics(); ok {
				return fmt.Errorf("project has metrics %+v, want none", m)
			}
			return nil
		})
}

// expectMetrics compares derived PERT metrics against the three whole numbers a
// metrics assertion captures.
func expectMetrics(m wbs.Metrics, eStr, sdStr, rsdStr string) error {
	e, sd, rsd, err := threeInts(eStr, sdStr, rsdStr)
	if err != nil {
		return err
	}
	if m.Expected != e || m.StandardDeviation != sd || m.RelativeStandardDeviation != rsd {
		return fmt.Errorf("metrics = %d/%d/%d, want %d/%d/%d", m.Expected, m.StandardDeviation, m.RelativeStandardDeviation, e, sd, rsd)
	}
	return nil
}
