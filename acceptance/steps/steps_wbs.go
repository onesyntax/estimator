package steps

import (
	"errors"
	"fmt"

	"estimation/acceptance/runtime"
	"estimation/internal/wbs"
)

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
