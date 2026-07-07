package main

import "net/http"

// registerEditing registers the Editing cases (qa/wbs_qa_suite.md §3, mirroring
// features/wbs_editing.feature). Every case starts from a freshly set-up
// unapproved WBS with tasks 1..3 = Login API, Login UI, Session store.
func registerEditing() {
	// QA-EDIT-1 — Add a task.
	register("QA-EDIT-1 add a task", func(t *T) {
		id := setupWBS(t)
		r := postJSON("/wbs/"+id+"/tasks", map[string]string{"description": "Password reset"})
		t.eq("status", r.status, http.StatusCreated)
		t.tasksAre(id, "Login API", "Login UI", "Session store", "Password reset")
	})

	// QA-EDIT-2 — Edit a task.
	editCases := []struct {
		number int
		newDsc string
	}{{1, "OAuth login API"}, {3, "Redis session store"}}
	for _, ec := range editCases {
		ec := ec
		register("QA-EDIT-2 edit task", func(t *T) {
			id := setupWBS(t)
			r := putJSON("/wbs/"+id+"/tasks/"+taskID(t, id, ec.number),
				map[string]string{"description": ec.newDsc})
			t.eq("status", r.status, http.StatusOK)
			v := getJSON("/wbs/" + id).view()
			t.eq("task count", len(v.Tasks), 3)
			if ec.number-1 < len(v.Tasks) {
				t.eq("edited position", v.Tasks[ec.number-1].Description, ec.newDsc)
			}
		})
	}

	// QA-EDIT-3 — Delete a task (each against a freshly set-up WBS).
	deleteCases := []struct {
		number  int
		deleted string
	}{{2, "Login UI"}, {1, "Login API"}}
	for _, dc := range deleteCases {
		dc := dc
		register("QA-EDIT-3 delete a task", func(t *T) {
			id := setupWBS(t)
			r := del("/wbs/" + id + "/tasks/" + taskID(t, id, dc.number))
			t.eq("status", r.status, http.StatusOK)
			v := getJSON("/wbs/" + id).view()
			t.eq("task count", len(v.Tasks), 2)
			for _, tk := range v.Tasks {
				if tk.Description == dc.deleted {
					t.fail("deleted task %q still present", dc.deleted)
				}
			}
		})
	}

	// QA-EDIT-4 — Reject an empty description (add and edit).
	register("QA-EDIT-4 reject empty description", func(t *T) {
		id := setupWBS(t)
		add := postJSON("/wbs/"+id+"/tasks", map[string]string{"description": ""})
		t.errorCase(add, http.StatusBadRequest, "description is empty")
		edit := putJSON("/wbs/"+id+"/tasks/"+taskID(t, id, 2), map[string]string{"description": ""})
		t.errorCase(edit, http.StatusBadRequest, "description is empty")
		t.tasksAre(id, "Login API", "Login UI", "Session store")
	})

	// QA-EDIT-5 — Reject editing or deleting a task that does not exist.
	register("QA-EDIT-5 reject unknown task", func(t *T) {
		id := setupWBS(t)
		edit := putJSON("/wbs/"+id+"/tasks/nonexistent", map[string]string{"description": "X"})
		t.errorCase(edit, http.StatusNotFound, "task not found")
		delR := del("/wbs/" + id + "/tasks/nonexistent")
		t.errorCase(delR, http.StatusNotFound, "task not found")
		t.tasksAre(id, "Login API", "Login UI", "Session store")
	})
}
