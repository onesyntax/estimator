package steps

import (
	"fmt"
	"strings"

	"estimation/acceptance/runtime"
	"estimation/internal/wbs"
)

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
			task, err := currentTask(w(wd), a[0])
			if err != nil {
				return err
			}
			return expectCount("risk note count", len(task.RiskNotes), a[1])
		})

	reg.Step(`^risk note (\d+) on task number (\d+) is (.+)$`,
		func(wd any, a []string) error {
			task, err := currentTask(w(wd), a[1])
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
