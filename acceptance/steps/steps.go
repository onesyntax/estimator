// Package steps contains the project-specific acceptance step handlers that
// connect Gherkin step text to the estimation service. Repeated step shapes
// that vary only by example values share a single regex-capture handler.
package steps

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"estimation/acceptance/runtime"
	"estimation/internal/wbs"
)

// world is the per-scenario state shared by background and scenario steps.
type world struct {
	svc     *wbs.Service
	wbsID   string
	lastErr error
}

func newWorld() any { return &world{svc: wbs.NewService()} }

// NewRegistry builds a runtime registry with a fresh world per execution and
// every estimation step handler registered.
func NewRegistry() *runtime.Registry {
	reg := runtime.NewRegistry(newWorld)
	register(reg)
	return reg
}

func register(reg *runtime.Registry) {
	registerActions(reg)
	registerAssertions(reg)
	registerRiskActions(reg)
	registerRiskAssertions(reg)
	registerEstimateActions(reg)
	registerEstimateAssertions(reg)
}

// registerActions registers the precondition and action steps that drive the
// estimation service.
func registerActions(reg *runtime.Registry) {
	reg.Step(`^the estimation service is running with a deterministic AI provider$`,
		func(wd any, _ []string) error {
			w(wd).svc = wbs.NewService()
			return nil
		})

	reg.Step(`^the AI is primed to generate the tasks (.+)$`,
		func(wd any, a []string) error {
			w(wd).svc.Prime(splitTasks(a[0]))
			return nil
		})

	reg.Step(`^a Tech Lead has generated a WBS from a valid requirement document$`,
		func(wd any, _ []string) error {
			return generate(w(wd), wbs.NewTextDocument("a valid requirement"))
		})

	reg.Step(`^the Tech Lead has deleted every task in the WBS$`,
		func(wd any, _ []string) error {
			s := w(wd)
			cur, err := s.svc.Get(s.wbsID)
			if err != nil {
				return err
			}
			for cur.TaskCount() > 0 {
				if err := s.svc.DeleteTask(s.wbsID, 1); err != nil {
					return err
				}
			}
			return nil
		})

	reg.Step(`^the WBS has been approved$`,
		func(wd any, _ []string) error {
			s := w(wd)
			s.lastErr = s.svc.Approve(s.wbsID)
			return nil
		})

	reg.Step(`^a Tech Lead submits a valid (text|pdf) requirement document$`,
		func(wd any, a []string) error {
			return generate(w(wd), validDocument(a[0]))
		})

	reg.Step(`^a Tech Lead submits an empty (text|pdf) requirement document$`,
		func(wd any, a []string) error {
			generate(w(wd), emptyDocument(a[0]))
			return nil
		})

	reg.Step(`^a Tech Lead submits a corrupt PDF requirement document$`,
		func(wd any, _ []string) error {
			generate(w(wd), wbs.NewPDFDocument([]byte("not a valid pdf")))
			return nil
		})

	reg.Step(`^a Tech Lead requests a WBS that does not exist$`,
		func(wd any, _ []string) error {
			s := w(wd)
			_, s.lastErr = s.svc.Get("nonexistent-wbs")
			return nil
		})

	reg.Step(`^the Tech Lead adds a task with the description (.+)$`,
		func(wd any, a []string) error {
			s := w(wd)
			s.lastErr = s.svc.AddTask(s.wbsID, a[0])
			return nil
		})

	reg.Step(`^the Tech Lead adds a task with an empty description$`,
		func(wd any, _ []string) error {
			s := w(wd)
			s.lastErr = s.svc.AddTask(s.wbsID, "")
			return nil
		})

	reg.Step(`^the Tech Lead edits task number (\d+) to the description (.+)$`,
		func(wd any, a []string) error {
			s := w(wd)
			n, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			s.lastErr = s.svc.EditTask(s.wbsID, n, a[1])
			return nil
		})

	reg.Step(`^the Tech Lead edits task number (\d+) with an empty description$`,
		func(wd any, a []string) error {
			s := w(wd)
			n, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			s.lastErr = s.svc.EditTask(s.wbsID, n, "")
			return nil
		})

	reg.Step(`^the Tech Lead deletes task number (\d+)$`,
		func(wd any, a []string) error {
			s := w(wd)
			n, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			s.lastErr = s.svc.DeleteTask(s.wbsID, n)
			return nil
		})

	reg.Step(`^the Tech Lead edits a task that does not exist$`,
		func(wd any, _ []string) error {
			s := w(wd)
			s.lastErr = s.svc.EditTask(s.wbsID, unknownTaskNumber, "irrelevant")
			return nil
		})

	reg.Step(`^the Tech Lead deletes a task that does not exist$`,
		func(wd any, _ []string) error {
			s := w(wd)
			s.lastErr = s.svc.DeleteTask(s.wbsID, unknownTaskNumber)
			return nil
		})

	reg.Step(`^the Tech Lead approves the WBS$`,
		func(wd any, _ []string) error {
			s := w(wd)
			s.lastErr = s.svc.Approve(s.wbsID)
			return nil
		})

}

// registerAssertions registers the steps that assert on estimation outcomes.
func registerAssertions(reg *runtime.Registry) {
	reg.Step(`^a new WBS is created$`,
		func(wd any, _ []string) error {
			s := w(wd)
			if s.lastErr != nil {
				return fmt.Errorf("generation failed: %w", s.lastErr)
			}
			if s.wbsID == "" {
				return errors.New("no WBS was created")
			}
			return nil
		})

	reg.Step(`^no WBS is created$`,
		func(wd any, _ []string) error {
			if got := w(wd).svc.Count(); got != 0 {
				return fmt.Errorf("expected no WBS, but %d exist", got)
			}
			return nil
		})

	reg.Step(`^the WBS is unapproved$`,
		func(wd any, _ []string) error { return expectApproved(w(wd), false) })

	reg.Step(`^the WBS is approved$`,
		func(wd any, _ []string) error { return expectApproved(w(wd), true) })

	reg.Step(`^the WBS contains (\d+) tasks$`,
		func(wd any, a []string) error {
			cur, err := currentWBS(w(wd))
			if err != nil {
				return err
			}
			return expectCount("task count", cur.TaskCount(), a[0])
		})

	reg.Step(`^task number (\d+) is (.+)$`,
		func(wd any, a []string) error {
			cur, err := currentWBS(w(wd))
			if err != nil {
				return err
			}
			n, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			task, ok := cur.TaskAt(n)
			if !ok {
				return fmt.Errorf("task number %d does not exist", n)
			}
			if task.Description != a[1] {
				return fmt.Errorf("task number %d = %q, want %q", n, task.Description, a[1])
			}
			return nil
		})

	reg.Step(`^no task is (.+)$`,
		func(wd any, a []string) error {
			cur, err := currentWBS(w(wd))
			if err != nil {
				return err
			}
			for _, t := range cur.Tasks() {
				if t.Description == a[0] {
					return fmt.Errorf("task %q is still present", a[0])
				}
			}
			return nil
		})

	reg.Step(`^the task is added$`, expectNoError)
	reg.Step(`^the task is updated$`, expectNoError)
	reg.Step(`^the task is deleted$`, expectNoError)
	reg.Step(`^the change is accepted$`, expectNoError)

	reg.Step(`^the submission is rejected because the requirement is empty$`,
		expectErr(wbs.ErrEmptyRequirement))
	reg.Step(`^the submission is rejected because the document cannot be read$`,
		expectErr(wbs.ErrUnreadableDocument))
	reg.Step(`^the WBS is reported as not found$`,
		expectErr(wbs.ErrWBSNotFound))
	reg.Step(`^the change is rejected because the description is empty$`,
		expectErr(wbs.ErrEmptyDescription))
	reg.Step(`^the task is reported as not found$`,
		expectErr(wbs.ErrTaskNotFound))
	reg.Step(`^approval is rejected because the WBS has no tasks$`,
		expectErr(wbs.ErrNoTasks))

	reg.Step(`^the approved task list contains (\d+) tasks$`,
		func(wd any, a []string) error {
			cur, err := currentWBS(w(wd))
			if err != nil {
				return err
			}
			return expectCount("approved task count", len(cur.ApprovedTasks()), a[0])
		})
}

// registerRiskActions registers the precondition and action steps that flag and
// edit risk notes on the WBS.
func registerRiskActions(reg *runtime.Registry) {
	reg.Step(`^the AI is primed to flag the risks (.+)$`,
		func(wd any, a []string) error {
			assignments, err := splitRiskAssignments(a[0])
			if err != nil {
				return err
			}
			w(wd).svc.PrimeRisks(assignments)
			return nil
		})

	reg.Step(`^the Tech Lead flags risks on the WBS$`, flagRisks)
	reg.Step(`^the Tech Lead has flagged risks on the WBS$`, flagRisks)

	reg.Step(`^a Tech Lead flags risks on a WBS that does not exist$`,
		func(wd any, _ []string) error {
			s := w(wd)
			s.lastErr = s.svc.FlagRisks("nonexistent-wbs")
			return nil
		})

	reg.Step(`^the Tech Lead adds a risk note to task number (\d+) with the description (.+)$`,
		func(wd any, a []string) error { return addRiskNote(w(wd), a[0], a[1]) })

	reg.Step(`^the Tech Lead adds a risk note to task number (\d+) with an empty description$`,
		func(wd any, a []string) error { return addRiskNote(w(wd), a[0], "") })

	reg.Step(`^the Tech Lead edits risk note (\d+) on task number (\d+) to the description (.+)$`,
		func(wd any, a []string) error { return editRiskNote(w(wd), a[1], a[0], a[2]) })

	reg.Step(`^the Tech Lead edits risk note (\d+) on task number (\d+) with an empty description$`,
		func(wd any, a []string) error { return editRiskNote(w(wd), a[1], a[0], "") })

	reg.Step(`^the Tech Lead edits risk note (\d+) on task number (\d+)$`,
		func(wd any, a []string) error { return editRiskNote(w(wd), a[1], a[0], "irrelevant") })

	reg.Step(`^the Tech Lead deletes risk note (\d+) on task number (\d+)$`,
		func(wd any, a []string) error {
			s := w(wd)
			taskNum, notePos, err := taskAndNote(a[1], a[0])
			if err != nil {
				return err
			}
			s.lastErr = s.svc.DeleteRiskNote(s.wbsID, taskNum, notePos)
			return nil
		})
}

// registerRiskAssertions registers the steps that assert on risk-note outcomes.
func registerRiskAssertions(reg *runtime.Registry) {
	reg.Step(`^the risk note count on task number (\d+) is (\d+)$`,
		func(wd any, a []string) error {
			task, err := riskTask(w(wd), a[0])
			if err != nil {
				return err
			}
			return expectCount("risk note count", len(task.RiskNotes), a[1])
		})

	reg.Step(`^risk note (\d+) on task number (\d+) is (.+)$`,
		func(wd any, a []string) error {
			task, err := riskTask(w(wd), a[1])
			if err != nil {
				return err
			}
			pos, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			if pos < 1 || pos > len(task.RiskNotes) {
				return fmt.Errorf("risk note %d on task number %s does not exist", pos, a[1])
			}
			if got := task.RiskNotes[pos-1].Description; got != a[2] {
				return fmt.Errorf("risk note %d = %q, want %q", pos, got, a[2])
			}
			return nil
		})

	reg.Step(`^the risk note is added$`, expectNoError)
	reg.Step(`^the risk note is updated$`, expectNoError)
	reg.Step(`^the risk note is deleted$`, expectNoError)

	reg.Step(`^risk flagging is rejected because the WBS is unapproved$`,
		expectErr(wbs.ErrWBSNotApproved))
	reg.Step(`^the risk note is reported as not found$`,
		expectErr(wbs.ErrRiskNoteNotFound))
}

func flagRisks(wd any, _ []string) error {
	s := w(wd)
	s.lastErr = s.svc.FlagRisks(s.wbsID)
	return nil
}

func addRiskNote(s *world, taskStr, description string) error {
	n, err := taskNumber(taskStr)
	if err != nil {
		return err
	}
	s.lastErr = s.svc.AddRiskNote(s.wbsID, n, description)
	return nil
}

func editRiskNote(s *world, taskStr, noteStr, description string) error {
	taskNum, notePos, err := taskAndNote(taskStr, noteStr)
	if err != nil {
		return err
	}
	s.lastErr = s.svc.EditRiskNote(s.wbsID, taskNum, notePos, description)
	return nil
}

// taskAndNote parses a one-based task number and note position from their step
// captures.
func taskAndNote(taskStr, noteStr string) (taskNum, notePos int, err error) {
	if taskNum, err = taskNumber(taskStr); err != nil {
		return 0, 0, err
	}
	notePos, err = taskNumber(noteStr)
	return taskNum, notePos, err
}

// riskTask returns the current WBS task at the one-based number for a risk
// assertion.
func riskTask(s *world, taskStr string) (wbs.Task, error) {
	cur, err := currentWBS(s)
	if err != nil {
		return wbs.Task{}, err
	}
	n, err := taskNumber(taskStr)
	if err != nil {
		return wbs.Task{}, err
	}
	task, ok := cur.TaskAt(n)
	if !ok {
		return wbs.Task{}, fmt.Errorf("task number %d does not exist", n)
	}
	return task, nil
}

// splitRiskAssignments parses "task N: description" entries separated by
// semicolons into risk assignments.
func splitRiskAssignments(s string) ([]wbs.RiskAssignment, error) {
	parts := strings.Split(s, ";")
	out := make([]wbs.RiskAssignment, 0, len(parts))
	for _, p := range parts {
		entry := strings.TrimSpace(p)
		if entry == "" {
			continue
		}
		colon := strings.IndexByte(entry, ':')
		if colon < 0 {
			return nil, fmt.Errorf("invalid risk assignment %q: want \"task N: description\"", entry)
		}
		head := strings.TrimSpace(entry[:colon])
		desc := strings.TrimSpace(entry[colon+1:])
		n, err := taskNumber(strings.TrimSpace(strings.TrimPrefix(head, "task")))
		if err != nil {
			return nil, err
		}
		out = append(out, wbs.RiskAssignment{TaskNumber: n, Description: desc})
	}
	return out, nil
}

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
	task, err := riskTask(s, taskStr)
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
		if open := strings.IndexByte(body, '('); open >= 0 {
			if close := strings.LastIndexByte(body, ')'); close > open {
				reasoning = strings.TrimSpace(body[open+1 : close])
			}
			body = strings.TrimSpace(body[:open])
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

const unknownTaskNumber = 9999

func w(wd any) *world { return wd.(*world) }

func generate(s *world, doc wbs.RequirementDocument) error {
	id, err := s.svc.Generate(doc)
	s.lastErr = err
	if err == nil {
		s.wbsID = id
	}
	return nil
}

func validDocument(format string) wbs.RequirementDocument {
	return documentOf(format, "a valid requirement")
}

func emptyDocument(format string) wbs.RequirementDocument {
	return documentOf(format, "")
}

// documentOf builds a requirement document of the requested format ("text" or
// "pdf") carrying the given content.
func documentOf(format, content string) wbs.RequirementDocument {
	if format == "pdf" {
		return wbs.NewPDFDocument(wbs.EncodeMinimalPDF(content))
	}
	return wbs.NewTextDocument(content)
}

func currentWBS(s *world) (*wbs.WBS, error) {
	if s.wbsID == "" {
		return nil, errors.New("no current WBS")
	}
	return s.svc.Get(s.wbsID)
}

func expectApproved(s *world, want bool) error {
	cur, err := currentWBS(s)
	if err != nil {
		return err
	}
	if cur.Approved() != want {
		return fmt.Errorf("approved = %v, want %v", cur.Approved(), want)
	}
	return nil
}

func expectCount(label string, got int, want string) error {
	n, err := strconv.Atoi(want)
	if err != nil {
		return fmt.Errorf("invalid expected %s %q: %w", label, want, err)
	}
	if got != n {
		return fmt.Errorf("%s = %d, want %d", label, got, n)
	}
	return nil
}

func expectNoError(wd any, _ []string) error {
	if err := w(wd).lastErr; err != nil {
		return fmt.Errorf("expected success, got error: %w", err)
	}
	return nil
}

func expectErr(target error) runtime.HandlerFunc {
	return func(wd any, _ []string) error {
		if err := w(wd).lastErr; !errors.Is(err, target) {
			return fmt.Errorf("error = %v, want %v", err, target)
		}
		return nil
	}
}

func taskNumber(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid task number %q: %w", s, err)
	}
	return n, nil
}

func splitTasks(s string) []string {
	parts := strings.Split(s, ";")
	tasks := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tasks = append(tasks, t)
		}
	}
	return tasks
}
