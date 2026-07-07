package main

import "net/http"

// registerApproval registers the Approval cases (qa/wbs_qa_suite.md §4,
// mirroring features/wbs_approval.feature). Every case starts from the same
// three-task unapproved WBS.
func registerApproval() {
	// QA-APP-1 — Approve a WBS that has tasks.
	register("QA-APP-1 approve a wbs with tasks", func(t *T) {
		id := setupWBS(t)
		r := postJSON("/wbs/"+id+"/approve", nil)
		t.eq("status", r.status, http.StatusOK)
		t.eq("state", r.view().State, "approved")
		g := getJSON("/wbs/" + id).view()
		t.eq("GET state", g.State, "approved")
		t.eq("approved task count", len(g.ApprovedTasks), 3)
	})

	// QA-APP-2 — Reject approving an empty WBS.
	register("QA-APP-2 reject approving empty wbs", func(t *T) {
		id := setupWBS(t)
		for {
			v := getJSON("/wbs/" + id).view()
			if len(v.Tasks) == 0 {
				break
			}
			r := del("/wbs/" + id + "/tasks/" + v.Tasks[0].ID)
			t.eq("delete status", r.status, http.StatusOK)
		}
		r := postJSON("/wbs/"+id+"/approve", nil)
		t.errorCase(r, http.StatusConflict, "cannot approve an empty WBS")
		g := getJSON("/wbs/" + id).view()
		t.eq("state", g.State, "unapproved")
		if len(g.ApprovedTasks) != 0 {
			t.fail("approvedTasks not null after rejected approval")
		}
	})

	// QA-APP-3 — Editing an approved WBS returns it to unapproved.
	editActions := []struct {
		name string
		do   func(t *T, id string) response
		want int
	}{
		{"add", func(t *T, id string) response {
			return postJSON("/wbs/"+id+"/tasks", map[string]string{"description": "Audit log"})
		}, http.StatusCreated},
		{"edit", func(t *T, id string) response {
			return putJSON("/wbs/"+id+"/tasks/"+taskID(t, id, 1), map[string]string{"description": "SSO API"})
		}, http.StatusOK},
		{"delete", func(t *T, id string) response {
			return del("/wbs/" + id + "/tasks/" + taskID(t, id, 2))
		}, http.StatusOK},
	}
	for _, ea := range editActions {
		ea := ea
		register("QA-APP-3 edit approved wbs unapproves it ("+ea.name+")", func(t *T) {
			id := setupWBS(t)
			ar := postJSON("/wbs/"+id+"/approve", nil)
			t.eq("approve status", ar.status, http.StatusOK)
			t.eq("approved", ar.view().State, "approved")
			r := ea.do(t, id)
			t.eq("edit status", r.status, ea.want)
			g := getJSON("/wbs/" + id).view()
			t.eq("state", g.State, "unapproved")
			if len(g.ApprovedTasks) != 0 {
				t.fail("approvedTasks not null after edit")
			}
		})
	}

	// QA-APP-4 — Re-approve a WBS after an edit.
	register("QA-APP-4 re-approve after edit", func(t *T) {
		id := setupWBS(t)
		a1 := postJSON("/wbs/"+id+"/approve", nil)
		t.eq("first approve status", a1.status, http.StatusOK)
		t.eq("approved", a1.view().State, "approved")

		add := postJSON("/wbs/"+id+"/tasks", map[string]string{"description": "Audit log"})
		t.eq("add status", add.status, http.StatusCreated)
		t.eq("state after edit", getJSON("/wbs/"+id).view().State, "unapproved")

		a2 := postJSON("/wbs/"+id+"/approve", nil)
		t.eq("second approve status", a2.status, http.StatusOK)
		t.eq("re-approved", a2.view().State, "approved")
		t.eq("approved task count", len(getJSON("/wbs/"+id).view().ApprovedTasks), 4)
	})
}
