package wbs

import (
	"errors"
	"testing"
)

func descriptions(w *WBS) []string {
	out := make([]string, 0, w.TaskCount())
	for _, t := range w.Tasks() {
		out = append(out, t.Description)
	}
	return out
}

func newTestWBS(t *testing.T, descs ...string) *WBS {
	t.Helper()
	return NewWBS("wbs-1", descs)
}

func TestNewWBSIsUnapprovedWithGeneratedTasks(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")

	if w.Approved() {
		t.Fatal("a freshly generated WBS must be unapproved")
	}
	if got := w.TaskCount(); got != 3 {
		t.Fatalf("task count = %d, want 3", got)
	}
	if got := descriptions(w); got[0] != "Login API" {
		t.Fatalf("task 1 = %q, want %q", got[0], "Login API")
	}
	if w.ApprovedTasks() != nil {
		t.Fatal("an unapproved WBS must not expose an approved task list")
	}
}

func TestTasksAreNumberedFromOne(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")

	task, ok := w.TaskAt(1)
	if !ok || task.Description != "Login API" {
		t.Fatalf("task number 1 = %+v ok=%v, want Login API", task, ok)
	}
	task, ok = w.TaskAt(3)
	if !ok || task.Description != "Session store" {
		t.Fatalf("task number 3 = %+v ok=%v, want Session store", task, ok)
	}
	if _, ok := w.TaskAt(0); ok {
		t.Fatal("task number 0 must not exist")
	}
	if _, ok := w.TaskAt(4); ok {
		t.Fatal("task number 4 must not exist")
	}
}

func TestAddTaskAppendsAndUnapproves(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")
	mustApprove(t, w)

	if err := w.AddTask("Password reset"); err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}
	if got := w.TaskCount(); got != 4 {
		t.Fatalf("task count = %d, want 4", got)
	}
	task, ok := w.TaskAt(4)
	if !ok || task.Description != "Password reset" {
		t.Fatalf("task number 4 = %+v ok=%v, want Password reset", task, ok)
	}
	if w.Approved() {
		t.Fatal("adding a task must unapprove the WBS")
	}
}

func TestAddTaskRejectsEmptyDescription(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")

	err := w.AddTask("   ")
	if !errors.Is(err, ErrEmptyDescription) {
		t.Fatalf("AddTask empty error = %v, want ErrEmptyDescription", err)
	}
	if got := w.TaskCount(); got != 3 {
		t.Fatalf("task count = %d, want 3 (unchanged)", got)
	}
}

func TestEditTaskUpdatesAndUnapproves(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")
	mustApprove(t, w)

	if err := w.EditTask(1, "OAuth login API"); err != nil {
		t.Fatalf("EditTask returned error: %v", err)
	}
	task, _ := w.TaskAt(1)
	if task.Description != "OAuth login API" {
		t.Fatalf("task number 1 = %q, want OAuth login API", task.Description)
	}
	if got := w.TaskCount(); got != 3 {
		t.Fatalf("task count = %d, want 3 (unchanged)", got)
	}
	if w.Approved() {
		t.Fatal("editing a task must unapprove the WBS")
	}
}

func TestEditTaskRejectsEmptyDescription(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")

	err := w.EditTask(2, "")
	if !errors.Is(err, ErrEmptyDescription) {
		t.Fatalf("EditTask empty error = %v, want ErrEmptyDescription", err)
	}
	if task, _ := w.TaskAt(2); task.Description != "Login UI" {
		t.Fatalf("task 2 changed to %q, want unchanged Login UI", task.Description)
	}
}

func TestEditTaskRejectsUnknownNumber(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")

	err := w.EditTask(99, "whatever")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("EditTask unknown error = %v, want ErrTaskNotFound", err)
	}
	if got := w.TaskCount(); got != 3 {
		t.Fatalf("task count = %d, want 3 (unchanged)", got)
	}
}

func TestDeleteTaskRemovesAndRenumbers(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")
	mustApprove(t, w)

	if err := w.DeleteTask(2); err != nil {
		t.Fatalf("DeleteTask returned error: %v", err)
	}
	if got := w.TaskCount(); got != 2 {
		t.Fatalf("task count = %d, want 2", got)
	}
	for _, d := range descriptions(w) {
		if d == "Login UI" {
			t.Fatal("deleted task Login UI is still present")
		}
	}
	if task, _ := w.TaskAt(2); task.Description != "Session store" {
		t.Fatalf("after delete, task 2 = %q, want Session store (renumbered)", task.Description)
	}
	if w.Approved() {
		t.Fatal("deleting a task must unapprove the WBS")
	}
}

func TestDeleteTaskRejectsUnknownNumber(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")

	err := w.DeleteTask(99)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("DeleteTask unknown error = %v, want ErrTaskNotFound", err)
	}
	if got := w.TaskCount(); got != 3 {
		t.Fatalf("task count = %d, want 3 (unchanged)", got)
	}
}

func TestApproveExposesApprovedTaskList(t *testing.T) {
	w := newTestWBS(t, "Login API", "Login UI", "Session store")

	if err := w.Approve(); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	if !w.Approved() {
		t.Fatal("WBS must be approved after Approve")
	}
	if got := len(w.ApprovedTasks()); got != 3 {
		t.Fatalf("approved task list length = %d, want 3", got)
	}
}

func TestApproveRejectsEmptyWBS(t *testing.T) {
	w := newTestWBS(t, "Login API")
	if err := w.DeleteTask(1); err != nil {
		t.Fatalf("DeleteTask returned error: %v", err)
	}

	err := w.Approve()
	if !errors.Is(err, ErrNoTasks) {
		t.Fatalf("Approve empty error = %v, want ErrNoTasks", err)
	}
	if w.Approved() {
		t.Fatal("approving an empty WBS must leave it unapproved")
	}
}

func mustApprove(t *testing.T, w *WBS) {
	t.Helper()
	if err := w.Approve(); err != nil {
		t.Fatalf("precondition Approve failed: %v", err)
	}
}
