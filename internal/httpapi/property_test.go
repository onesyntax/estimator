//go:build property

// Property tests for the HTTP adapter's pure projection logic. Kept out of the
// normal unit suite (and coverage/mutation/CRAP) by the `property` build tag;
// run them with scripts/property.sh. They assert that the JSON view is a
// faithful, order-preserving projection of the domain WBS over broad inputs,
// which the example-based handler tests do not cover.
package httpapi

import (
	"math/rand"
	"strconv"
	"testing"

	"estimation/internal/wbs"
)

const propertySeed = 20260707

// TestPropViewProjectsWBSFaithfully checks that view() mirrors the WBS exactly:
// same id, same tasks in the same order, a state that reflects approval, and an
// approved task list present exactly when the WBS is approved.
func TestPropViewProjectsWBSFaithfully(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		n := rng.Intn(6) // 0..5 tasks; 0 exercises the empty, never-approvable WBS
		descs := make([]string, n)
		for i := range descs {
			descs[i] = "task-" + strconv.Itoa(rng.Intn(1000))
		}
		w := wbs.NewWBS("wbs-"+strconv.Itoa(iter), descs)
		approve := n > 0 && rng.Intn(2) == 0
		if approve {
			if err := w.Approve(); err != nil {
				t.Fatalf("iter %d: Approve failed: %v", iter, err)
			}
		}

		dto := view(w)

		if dto.ID != w.ID() {
			t.Fatalf("iter %d: dto.ID = %q, want %q", iter, dto.ID, w.ID())
		}
		wantState := "unapproved"
		if w.Approved() {
			wantState = "approved"
		}
		if dto.State != wantState {
			t.Fatalf("iter %d: dto.State = %q, want %q", iter, dto.State, wantState)
		}

		tasks := w.Tasks()
		if len(dto.Tasks) != len(tasks) {
			t.Fatalf("iter %d: dto has %d tasks, want %d", iter, len(dto.Tasks), len(tasks))
		}
		for i, task := range tasks {
			if dto.Tasks[i].ID != task.ID || dto.Tasks[i].Description != task.Description {
				t.Fatalf("iter %d: dto.Tasks[%d] = %+v, want %+v", iter, i, dto.Tasks[i], task)
			}
		}

		if w.Approved() {
			if len(dto.ApprovedTasks) != len(tasks) {
				t.Fatalf("iter %d: approved dto has %d approved tasks, want %d", iter, len(dto.ApprovedTasks), len(tasks))
			}
			for i, task := range tasks {
				if dto.ApprovedTasks[i].ID != task.ID || dto.ApprovedTasks[i].Description != task.Description {
					t.Fatalf("iter %d: dto.ApprovedTasks[%d] = %+v, want %+v", iter, i, dto.ApprovedTasks[i], task)
				}
			}
		} else if dto.ApprovedTasks != nil {
			t.Fatalf("iter %d: unapproved WBS exposed approvedTasks %v", iter, dto.ApprovedTasks)
		}
	}
}
