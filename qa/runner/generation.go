package main

import "net/http"

// registerGeneration registers the Generation cases (qa/wbs_qa_suite.md §2,
// mirroring features/wbs_generation.feature).
func registerGeneration() {
	// QA-GEN-1 — Generate a WBS from a valid requirement (text and PDF).
	register("QA-GEN-1 generate valid requirement (text)", func(t *T) {
		prime(t, []string{"Login API", "Login UI", "Session store"})
		r := postJSON("/wbs", map[string]string{"requirement": "Build a login system."})
		t.eq("status", r.status, http.StatusCreated)
		v := r.view()
		t.eq("state", v.State, "unapproved")
		t.eq("task count", len(v.Tasks), 3)
		if len(v.Tasks) > 0 {
			t.eq("tasks[0]", v.Tasks[0].Description, "Login API")
		}
		g := getJSON("/wbs/" + v.ID).view()
		t.eq("GET state", g.State, "unapproved")
		if len(g.ApprovedTasks) != 0 {
			t.fail("approvedTasks not null before approval")
		}
		t.tasksAre(v.ID, "Login API", "Login UI", "Session store")
	})

	register("QA-GEN-1 generate valid requirement (pdf)", func(t *T) {
		prime(t, []string{"Charge API", "Refund API"})
		r := postPDF("/wbs", validPDF("Build a payment flow."))
		t.eq("status", r.status, http.StatusCreated)
		v := r.view()
		t.eq("state", v.State, "unapproved")
		t.eq("task count", len(v.Tasks), 2)
		if len(v.Tasks) > 0 {
			t.eq("tasks[0]", v.Tasks[0].Description, "Charge API")
		}
		g := getJSON("/wbs/" + v.ID).view()
		if len(g.ApprovedTasks) != 0 {
			t.fail("approvedTasks not null before approval")
		}
		t.tasksAre(v.ID, "Charge API", "Refund API")
	})

	// QA-GEN-2 — Reject an empty requirement (text and PDF).
	register("QA-GEN-2 reject empty requirement (text)", func(t *T) {
		r := postJSON("/wbs", map[string]string{"requirement": ""})
		t.errorCase(r, http.StatusBadRequest, "requirement is empty")
		if r.view().ID != "" {
			t.fail("a WBS id was returned for an empty requirement")
		}
	})

	register("QA-GEN-2 reject empty requirement (pdf)", func(t *T) {
		r := postPDF("/wbs", validPDF(""))
		t.errorCase(r, http.StatusBadRequest, "requirement is empty")
		if r.view().ID != "" {
			t.fail("a WBS id was returned for an empty PDF")
		}
	})

	// QA-GEN-3 — Reject a corrupt PDF.
	register("QA-GEN-3 reject corrupt pdf", func(t *T) {
		r := postPDF("/wbs", corruptPDF())
		t.errorCase(r, http.StatusBadRequest, "document could not be read")
		if r.view().ID != "" {
			t.fail("a WBS id was returned for a corrupt PDF")
		}
	})

	// QA-GEN-4 — Report a WBS that does not exist.
	register("QA-GEN-4 report missing wbs", func(t *T) {
		r := getJSON("/wbs/does-not-exist")
		t.errorCase(r, http.StatusNotFound, "WBS not found")
	})
}
