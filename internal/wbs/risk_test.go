package wbs

import (
	"errors"
	"testing"
)

func approvedWBS(t *testing.T, descs ...string) *WBS {
	t.Helper()
	w := NewWBS("wbs-1", descs)
	mustApprove(t, w)
	return w
}

func noteDescriptions(w *WBS, taskNumber int) []string {
	task, _ := w.TaskAt(taskNumber)
	out := make([]string, 0, len(task.RiskNotes))
	for _, n := range task.RiskNotes {
		out = append(out, n.Description)
	}
	return out
}

func TestReplaceRiskNotesSpreadsAssignmentsAcrossTasks(t *testing.T) {
	w := approvedWBS(t, "Login API", "Login UI", "Session store")

	w.ReplaceRiskNotes([]RiskAssignment{
		{TaskNumber: 1, Description: "SQL injection"},
		{TaskNumber: 1, Description: "Rate limiting"},
		{TaskNumber: 2, Description: "XSS"},
	})

	if got := noteDescriptions(w, 1); len(got) != 2 || got[0] != "SQL injection" || got[1] != "Rate limiting" {
		t.Fatalf("task 1 notes = %v, want [SQL injection Rate limiting]", got)
	}
	if got := noteDescriptions(w, 2); len(got) != 1 || got[0] != "XSS" {
		t.Fatalf("task 2 notes = %v, want [XSS]", got)
	}
	if got := noteDescriptions(w, 3); len(got) != 0 {
		t.Fatalf("task 3 notes = %v, want none", got)
	}
}

func TestReplaceRiskNotesReplacesPriorNotes(t *testing.T) {
	w := approvedWBS(t, "Login API", "Login UI", "Session store")
	w.ReplaceRiskNotes([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}, {TaskNumber: 2, Description: "XSS"}})
	if err := w.AddRiskNote(3, "Manual concern"); err != nil {
		t.Fatalf("AddRiskNote returned error: %v", err)
	}

	w.ReplaceRiskNotes([]RiskAssignment{{TaskNumber: 1, Description: "CSRF"}})

	if got := noteDescriptions(w, 1); len(got) != 1 || got[0] != "CSRF" {
		t.Fatalf("task 1 notes = %v, want [CSRF]", got)
	}
	if got := noteDescriptions(w, 2); len(got) != 0 {
		t.Fatalf("task 2 notes = %v, want none (replaced)", got)
	}
	if got := noteDescriptions(w, 3); len(got) != 0 {
		t.Fatalf("task 3 notes = %v, want none (manual note replaced)", got)
	}
}

func TestReplaceRiskNotesSkipsOutOfRangeTaskNumbers(t *testing.T) {
	w := approvedWBS(t, "Login API")

	w.ReplaceRiskNotes([]RiskAssignment{
		{TaskNumber: 1, Description: "SQL injection"},
		{TaskNumber: 5, Description: "off the end"},
		{TaskNumber: 0, Description: "before the start"},
	})

	if got := noteDescriptions(w, 1); len(got) != 1 || got[0] != "SQL injection" {
		t.Fatalf("task 1 notes = %v, want only the in-range note", got)
	}
}

func TestReplaceRiskNotesDoesNotChangeApproval(t *testing.T) {
	w := approvedWBS(t, "Login API")
	w.ReplaceRiskNotes([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}})
	if !w.Approved() {
		t.Fatal("flagging risks must not unapprove the WBS")
	}
}

func TestRiskNotesHaveStableDistinctIDs(t *testing.T) {
	w := approvedWBS(t, "Login API", "Login UI")
	w.ReplaceRiskNotes([]RiskAssignment{
		{TaskNumber: 1, Description: "SQL injection"},
		{TaskNumber: 2, Description: "XSS"},
	})

	t1, _ := w.TaskAt(1)
	t2, _ := w.TaskAt(2)
	if t1.RiskNotes[0].ID == "" {
		t.Fatal("risk note ID must be non-empty")
	}
	if t1.RiskNotes[0].ID == t2.RiskNotes[0].ID {
		t.Fatalf("risk notes must have distinct IDs, both = %q", t1.RiskNotes[0].ID)
	}
}

func TestTaskAtReturnsCopyOfRiskNotes(t *testing.T) {
	w := approvedWBS(t, "Login API")
	w.ReplaceRiskNotes([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}})

	task, _ := w.TaskAt(1)
	task.RiskNotes[0].Description = "mutated externally"

	if got := noteDescriptions(w, 1); got[0] != "SQL injection" {
		t.Fatalf("internal note changed via returned copy: %v", got)
	}
}

func TestAddRiskNoteAppendsToTask(t *testing.T) {
	w := approvedWBS(t, "Login API", "Login UI", "Session store")

	if err := w.AddRiskNote(3, "Missing rate limits"); err != nil {
		t.Fatalf("AddRiskNote returned error: %v", err)
	}
	if got := noteDescriptions(w, 3); len(got) != 1 || got[0] != "Missing rate limits" {
		t.Fatalf("task 3 notes = %v, want [Missing rate limits]", got)
	}
}

func TestAddRiskNoteRejectsEmptyDescription(t *testing.T) {
	w := approvedWBS(t, "Login API")

	err := w.AddRiskNote(1, "   ")
	if !errors.Is(err, ErrEmptyDescription) {
		t.Fatalf("AddRiskNote empty error = %v, want ErrEmptyDescription", err)
	}
	if got := noteDescriptions(w, 1); len(got) != 0 {
		t.Fatalf("task 1 notes = %v, want none after rejected add", got)
	}
}

func TestAddRiskNoteRejectsUnknownTask(t *testing.T) {
	w := approvedWBS(t, "Login API")

	if err := w.AddRiskNote(9, "whatever"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("AddRiskNote unknown task error = %v, want ErrTaskNotFound", err)
	}
}

func TestEditRiskNoteUpdatesDescriptionKeepingCount(t *testing.T) {
	w := approvedWBS(t, "Login API")
	w.ReplaceRiskNotes([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}})

	if err := w.EditRiskNote(1, 1, "SQL injection via login"); err != nil {
		t.Fatalf("EditRiskNote returned error: %v", err)
	}
	if got := noteDescriptions(w, 1); len(got) != 1 || got[0] != "SQL injection via login" {
		t.Fatalf("task 1 notes = %v, want [SQL injection via login]", got)
	}
}

func TestEditRiskNoteRejectsEmptyDescription(t *testing.T) {
	w := approvedWBS(t, "Login API")
	w.ReplaceRiskNotes([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}})

	err := w.EditRiskNote(1, 1, "")
	if !errors.Is(err, ErrEmptyDescription) {
		t.Fatalf("EditRiskNote empty error = %v, want ErrEmptyDescription", err)
	}
	if got := noteDescriptions(w, 1); got[0] != "SQL injection" {
		t.Fatalf("note changed after rejected edit: %v", got)
	}
}

func TestEditRiskNoteRejectsUnknownNote(t *testing.T) {
	w := approvedWBS(t, "Login API")
	w.ReplaceRiskNotes([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}})

	if err := w.EditRiskNote(1, 9, "X"); !errors.Is(err, ErrRiskNoteNotFound) {
		t.Fatalf("EditRiskNote unknown note error = %v, want ErrRiskNoteNotFound", err)
	}
}

func TestEditRiskNoteRejectsUnknownTask(t *testing.T) {
	w := approvedWBS(t, "Login API")

	if err := w.EditRiskNote(9, 1, "X"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("EditRiskNote unknown task error = %v, want ErrTaskNotFound", err)
	}
}

func TestDeleteRiskNoteRemovesNote(t *testing.T) {
	w := approvedWBS(t, "Login API")
	w.ReplaceRiskNotes([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}})

	if err := w.DeleteRiskNote(1, 1); err != nil {
		t.Fatalf("DeleteRiskNote returned error: %v", err)
	}
	if got := noteDescriptions(w, 1); len(got) != 0 {
		t.Fatalf("task 1 notes = %v, want none after delete", got)
	}
}

func TestDeleteRiskNoteRejectsUnknownNote(t *testing.T) {
	w := approvedWBS(t, "Login API")
	w.ReplaceRiskNotes([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}})

	if err := w.DeleteRiskNote(1, 9); !errors.Is(err, ErrRiskNoteNotFound) {
		t.Fatalf("DeleteRiskNote unknown note error = %v, want ErrRiskNoteNotFound", err)
	}
	if got := noteDescriptions(w, 1); len(got) != 1 {
		t.Fatalf("task 1 notes = %v, want unchanged after rejected delete", got)
	}
}

func TestDeleteRiskNoteRejectsUnknownTask(t *testing.T) {
	w := approvedWBS(t, "Login API")

	if err := w.DeleteRiskNote(9, 1); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("DeleteRiskNote unknown task error = %v, want ErrTaskNotFound", err)
	}
}
