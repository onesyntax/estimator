//go:build hardening

// Hardening tests for the Build view model, behind the `hardening` build tag so
// they stay out of the unit suite, coverage, CRAP, and DRY runs.
package webui

import "testing"

// TestTaskRowsAreNumberedFromOne pins the one-based task numbering. The unit
// tests assert task descriptions but never the Number field, so `i + 1` -> `i - 1`
// and `i + 1` -> `i + 0` both survive. Asserting the numbers of a three-task WBS
// kills them.
func TestTaskRowsAreNumberedFromOne(t *testing.T) {
	s := primeAndGenerate(t)

	rows := s.BuildView().Tasks
	if len(rows) != 3 {
		t.Fatalf("task count = %d, want 3", len(rows))
	}
	for i, want := range []int{1, 2, 3} {
		if rows[i].Number != want {
			t.Errorf("Tasks[%d].Number = %d, want %d", i, rows[i].Number, want)
		}
	}
}
