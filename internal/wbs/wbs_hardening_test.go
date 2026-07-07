//go:build hardening

// Hardening tests: added by the hardender to kill mutation survivors that the
// normal unit suite leaves alive. They live behind the `hardening` build tag so
// they stay out of the unit suite, coverage, CRAP, and DRY runs, and are
// exercised only by the language mutation tool via `-tags hardening`.
package wbs

import (
	"errors"
	"testing"
)

// Kills NewWBS `nextSeq: 1 -> 0`: task IDs must start at t1, not t0, and
// increase by one across the generated tasks.
func TestHardeningNewWBSAssignsSequentialIDsFromOne(t *testing.T) {
	w := NewWBS("wbs-1", []string{"Login API", "Login UI", "Session store"})

	tasks := w.Tasks()
	want := []string{"t1", "t2", "t3"}
	for i, expect := range want {
		if tasks[i].ID != expect {
			t.Fatalf("task %d ID = %q, want %q", i+1, tasks[i].ID, expect)
		}
	}
}

// Kills EditTask `number < 1 -> number < 0`: number 0 is out of range and must
// be rejected, not routed into a negative slice index.
func TestHardeningEditTaskRejectsZeroNumber(t *testing.T) {
	w := NewWBS("wbs-1", []string{"Login API", "Login UI", "Session store"})

	err := w.EditTask(0, "whatever")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("EditTask(0) error = %v, want ErrTaskNotFound", err)
	}
	if got := w.TaskCount(); got != 3 {
		t.Fatalf("task count = %d, want 3 (unchanged)", got)
	}
}

// Kills EditTask `number > len -> number >= len`: the last task number is valid
// and editing it must succeed.
func TestHardeningEditTaskEditsLastTask(t *testing.T) {
	w := NewWBS("wbs-1", []string{"Login API", "Login UI", "Session store"})

	if err := w.EditTask(3, "Session store v2"); err != nil {
		t.Fatalf("EditTask(3) on last task returned error: %v", err)
	}
	if task, _ := w.TaskAt(3); task.Description != "Session store v2" {
		t.Fatalf("last task = %q, want Session store v2", task.Description)
	}
}

// Kills DeleteTask `number < 1 -> number < 0`: number 0 is out of range and must
// be rejected, not routed into a negative slice bound.
func TestHardeningDeleteTaskRejectsZeroNumber(t *testing.T) {
	w := NewWBS("wbs-1", []string{"Login API", "Login UI", "Session store"})

	err := w.DeleteTask(0)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("DeleteTask(0) error = %v, want ErrTaskNotFound", err)
	}
	if got := w.TaskCount(); got != 3 {
		t.Fatalf("task count = %d, want 3 (unchanged)", got)
	}
}
