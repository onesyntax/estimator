package steps

import (
	"fmt"
	"strconv"
	"strings"

	"estimation/acceptance/runtime"
	"estimation/internal/wbs"
)

// registerEstimateActions registers the precondition and action steps that
// generate, override, and approve 3-point estimates.
func registerEstimateActions(reg *runtime.Registry) {
	reg.Step(`^the AI is primed to estimate (.+)$`,
		func(wd any, a []string) error {
			assignments, err := splitEstimateAssignments(a[0])
			if err != nil {
				return err
			}
			w(wd).svc.PrimeEstimates(assignments)
			return nil
		})

	reg.Step(`^the Tech Lead generates estimates$`, generateEstimates)
	reg.Step(`^the Tech Lead has generated estimates$`, generateEstimates)

	reg.Step(`^a Tech Lead generates estimates on a WBS that does not exist$`,
		func(wd any, _ []string) error {
			s := w(wd)
			s.lastErr = s.svc.GenerateEstimates("nonexistent-wbs")
			return nil
		})

	reg.Step(`^the Tech Lead overrides task number (\d+) with optimistic (-?\d+) most likely (-?\d+) pessimistic (-?\d+) and reasoning (.*)$`,
		func(wd any, a []string) error {
			n, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			return overrideEstimate(w(wd), n, a[1], a[2], a[3], a[4])
		})

	reg.Step(`^the Tech Lead overrides a task that does not exist with optimistic (-?\d+) most likely (-?\d+) pessimistic (-?\d+) and reasoning (.*)$`,
		func(wd any, a []string) error {
			return overrideEstimate(w(wd), unknownTaskNumber, a[0], a[1], a[2], a[3])
		})

	reg.Step(`^the Tech Lead approves the estimates$`, approveEstimates)
	reg.Step(`^the estimates have been approved$`, approveEstimates)
}

// registerEstimateAssertions registers the steps that assert on estimate
// outcomes.
func registerEstimateAssertions(reg *runtime.Registry) {
	reg.Step(`^the estimates are unapproved$`,
		func(wd any, _ []string) error { return expectEstimatesApproved(w(wd), false) })
	reg.Step(`^the estimates are approved$`,
		func(wd any, _ []string) error { return expectEstimatesApproved(w(wd), true) })

	reg.Step(`^task number (\d+) has estimate optimistic (-?\d+) most likely (-?\d+) pessimistic (-?\d+)$`,
		func(wd any, a []string) error {
			e, err := estimateOf(w(wd), a[0])
			if err != nil {
				return err
			}
			if e == nil {
				return fmt.Errorf("task number %s has no estimate", a[0])
			}
			o, m, p, err := threeInts(a[1], a[2], a[3])
			if err != nil {
				return err
			}
			if e.Optimistic != o || e.MostLikely != m || e.Pessimistic != p {
				return fmt.Errorf("estimate = %d/%d/%d, want %d/%d/%d", e.Optimistic, e.MostLikely, e.Pessimistic, o, m, p)
			}
			return nil
		})

	reg.Step(`^task number (\d+) has reasoning (.+)$`,
		func(wd any, a []string) error {
			e, err := estimateOf(w(wd), a[0])
			if err != nil {
				return err
			}
			if e == nil {
				return fmt.Errorf("task number %s has no estimate", a[0])
			}
			if e.Reasoning != a[1] {
				return fmt.Errorf("reasoning = %q, want %q", e.Reasoning, a[1])
			}
			return nil
		})

	reg.Step(`^task number (\d+) has no estimate$`,
		func(wd any, a []string) error {
			e, err := estimateOf(w(wd), a[0])
			if err != nil {
				return err
			}
			if e != nil {
				return fmt.Errorf("task number %s has estimate %+v, want none", a[0], *e)
			}
			return nil
		})

	reg.Step(`^the override is accepted$`, expectNoError)

	reg.Step(`^the override is rejected because (.+)$`,
		func(wd any, a []string) error {
			target, ok := overrideRejections[a[0]]
			if !ok {
				return fmt.Errorf("unknown override rejection reason %q", a[0])
			}
			return expectErr(target)(wd, nil)
		})

	reg.Step(`^estimation is rejected because the WBS is unapproved$`,
		expectErr(wbs.ErrWBSNotApprovedForEstimation))
	reg.Step(`^approval is rejected because estimates have not been generated$`,
		expectErr(wbs.ErrEstimatesNotGenerated))
}

// overrideRejections maps each override-rejection reason phrase to the domain
// error the step expects.
var overrideRejections = map[string]error{
	"the values are not strictly increasing": wbs.ErrEstimatesNotIncreasing,
	"the value is off the Fibonacci scale":   wbs.ErrOffFibonacciScale,
	"the reasoning is empty":                 wbs.ErrEmptyReasoning,
	"the task has no estimate":               wbs.ErrNoEstimate,
}

func generateEstimates(wd any, _ []string) error {
	s := w(wd)
	s.lastErr = s.svc.GenerateEstimates(s.wbsID)
	return nil
}

func approveEstimates(wd any, _ []string) error {
	s := w(wd)
	s.lastErr = s.svc.ApproveEstimates(s.wbsID)
	return nil
}

func overrideEstimate(s *world, taskNum int, oStr, mStr, pStr, reasoning string) error {
	o, m, p, err := threeInts(oStr, mStr, pStr)
	if err != nil {
		return err
	}
	s.lastErr = s.svc.OverrideEstimate(s.wbsID, taskNum, wbs.Estimate{Optimistic: o, MostLikely: m, Pessimistic: p, Reasoning: reasoning})
	return nil
}

// estimateOf returns the estimate of the current WBS task at the one-based
// number, or nil when the task has none.
func estimateOf(s *world, taskStr string) (*wbs.Estimate, error) {
	task, err := currentTask(s, taskStr)
	if err != nil {
		return nil, err
	}
	return task.Estimate, nil
}

func expectEstimatesApproved(s *world, want bool) error {
	cur, err := currentWBS(s)
	if err != nil {
		return err
	}
	if cur.EstimatesApproved() != want {
		return fmt.Errorf("estimates approved = %v, want %v", cur.EstimatesApproved(), want)
	}
	return nil
}

// splitEstimateAssignments parses "task N: O/M/P (reasoning)" entries separated
// by semicolons. The reasoning parenthetical is optional; when absent a
// non-empty placeholder is used.
func splitEstimateAssignments(s string) ([]wbs.EstimateAssignment, error) {
	parts := strings.Split(s, ";")
	out := make([]wbs.EstimateAssignment, 0, len(parts))
	for _, p := range parts {
		entry := strings.TrimSpace(p)
		if entry == "" {
			continue
		}
		colon := strings.IndexByte(entry, ':')
		if colon < 0 {
			return nil, fmt.Errorf("invalid estimate %q: want \"task N: O/M/P\"", entry)
		}
		n, err := taskNumber(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(entry[:colon]), "task")))
		if err != nil {
			return nil, err
		}
		body := strings.TrimSpace(entry[colon+1:])
		reasoning := "AI estimate"
		if openIdx := strings.IndexByte(body, '('); openIdx >= 0 {
			if closeIdx := strings.LastIndexByte(body, ')'); closeIdx > openIdx {
				reasoning = strings.TrimSpace(body[openIdx+1 : closeIdx])
			}
			body = strings.TrimSpace(body[:openIdx])
		}
		nums := strings.Split(body, "/")
		if len(nums) != 3 {
			return nil, fmt.Errorf("invalid estimate values %q: want O/M/P", body)
		}
		o, m, pp, err := threeInts(strings.TrimSpace(nums[0]), strings.TrimSpace(nums[1]), strings.TrimSpace(nums[2]))
		if err != nil {
			return nil, err
		}
		out = append(out, wbs.EstimateAssignment{TaskNumber: n, Optimistic: o, MostLikely: m, Pessimistic: pp, Reasoning: reasoning})
	}
	return out, nil
}

// threeInts parses three signed integers from their string captures.
func threeInts(a, b, c string) (int, int, int, error) {
	x, err := atoi(a)
	if err != nil {
		return 0, 0, 0, err
	}
	y, err := atoi(b)
	if err != nil {
		return 0, 0, 0, err
	}
	z, err := atoi(c)
	if err != nil {
		return 0, 0, 0, err
	}
	return x, y, z, nil
}

func atoi(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", s, err)
	}
	return n, nil
}
