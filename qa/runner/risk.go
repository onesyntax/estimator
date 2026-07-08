package main

import "net/http"

// registerRisk registers the Module 2 cases (qa/risk_qa_suite.md, mirroring
// features/risk_flagging.feature and features/risk_note_editing.feature). Risk
// Notes belong to tasks; flagging requires an approved WBS and replaces all
// existing notes; individual notes stay editable with no separate approval gate.

// riskAssignment is one primed risk note: which task (1-based position) receives
// it and its description. It mirrors the /qa/ai/next-risks affordance body.
type riskAssignment struct {
	taskNumber  int
	description string
}

// primeRisks seeds the notes the next flag returns via the QA affordance,
// asserting the documented 204.
func primeRisks(t *T, risks []riskAssignment) {
	payload := make([]map[string]any, len(risks))
	for i, r := range risks {
		payload[i] = map[string]any{"taskNumber": r.taskNumber, "description": r.description}
	}
	r := postJSON("/qa/ai/next-risks", map[string]any{"risks": payload})
	t.eq("prime risks status", r.status, http.StatusNoContent)
}

// approvedWBS is the shared risk setup: the standard three-task WBS, approved so
// it is eligible for flagging.
func approvedWBS(t *T) string {
	id := setupWBS(t)
	r := postJSON("/wbs/"+id+"/approve", nil)
	t.eq("approve status", r.status, http.StatusOK)
	return id
}

// riskNotesOn returns the risk notes on task number n (1-based) in GET order.
func riskNotesOn(t *T, id string, n int) []riskNote {
	v := getJSON("/wbs/" + id).view()
	if n < 1 || n > len(v.Tasks) {
		t.fail("resolve task %d: only %d tasks present", n, len(v.Tasks))
		return nil
	}
	return v.Tasks[n-1].RiskNotes
}

// riskNotesAre asserts the ordered note descriptions on task number n.
func (t *T) riskNotesAre(id string, n int, want ...string) {
	notes := riskNotesOn(t, id, n)
	if len(notes) != len(want) {
		got := make([]string, len(notes))
		for i, nt := range notes {
			got[i] = nt.Description
		}
		t.fail("task %d note count = %d, want %d (%v)", n, len(notes), len(want), got)
		return
	}
	for i := range want {
		if notes[i].Description != want[i] {
			t.fail("task %d note %d = %q, want %q", n, i+1, notes[i].Description, want[i])
		}
	}
}

// noteID resolves "risk note k on task number n" to its id (tasks[n-1].riskNotes[k-1].id).
func noteID(t *T, id string, n, k int) string {
	notes := riskNotesOn(t, id, n)
	if k < 1 || k > len(notes) {
		t.fail("resolve note %d on task %d: only %d notes present", k, n, len(notes))
		return ""
	}
	return notes[k-1].ID
}

// riskNotePath builds the collection or member path for a task's risk notes.
func riskNotePath(id, taskID string) string {
	return "/wbs/" + id + "/tasks/" + taskID + "/risk-notes"
}

func registerRisk() {
	// --- Risk Flagging (risk_flagging.feature) -----------------------------

	// QA-FLAG-1 — Flag an approved WBS produces per-task notes.
	register("QA-FLAG-1 flag approved wbs produces per-task notes", func(t *T) {
		id := approvedWBS(t)
		primeRisks(t, []riskAssignment{
			{1, "SQL injection"}, {1, "Rate limiting"}, {2, "XSS"},
		})
		r := postJSON("/wbs/"+id+"/risk-notes/flag", nil)
		t.eq("flag status", r.status, http.StatusOK)
		t.riskNotesAre(id, 1, "SQL injection", "Rate limiting")
		t.riskNotesAre(id, 2, "XSS")
		t.riskNotesAre(id, 3)
	})

	// QA-FLAG-2 — Reject flagging an unapproved WBS.
	register("QA-FLAG-2 reject flagging unapproved wbs", func(t *T) {
		id := setupWBS(t) // not approved
		primeRisks(t, []riskAssignment{{1, "SQL injection"}})
		r := postJSON("/wbs/"+id+"/risk-notes/flag", nil)
		t.errorCase(r, http.StatusConflict, "WBS must be approved before risk flagging")
		for n := 1; n <= 3; n++ {
			t.riskNotesAre(id, n)
		}
		// The rejected flag did not consume the primed risks — the domain rejects
		// an unapproved WBS before calling the provider — leaving a stranded entry
		// in the mock's shared FIFO prime queue. Drain it through the UI so it
		// cannot leak into the next case: approve this now-discarded WBS and flag
		// once, which consumes the stranded prime. (The Module 1 cases stay
		// isolated the same way: every prime is consumed by a following action.)
		t.eq("drain approve status", postJSON("/wbs/"+id+"/approve", nil).status, http.StatusOK)
		t.eq("drain flag status", postJSON("/wbs/"+id+"/risk-notes/flag", nil).status, http.StatusOK)
	})

	// QA-FLAG-3 — Re-flagging replaces all notes (AI and manual).
	register("QA-FLAG-3 re-flagging replaces all notes", func(t *T) {
		id := approvedWBS(t)
		primeRisks(t, []riskAssignment{{1, "SQL injection"}, {2, "XSS"}})
		f1 := postJSON("/wbs/"+id+"/risk-notes/flag", nil)
		t.eq("first flag status", f1.status, http.StatusOK)

		add := postJSON(riskNotePath(id, taskID(t, id, 3)), map[string]string{"description": "Manual concern"})
		t.eq("add manual note status", add.status, http.StatusCreated)

		primeRisks(t, []riskAssignment{{1, "CSRF"}})
		f2 := postJSON("/wbs/"+id+"/risk-notes/flag", nil)
		t.eq("second flag status", f2.status, http.StatusOK)

		t.riskNotesAre(id, 1, "CSRF")
		t.riskNotesAre(id, 2)
		t.riskNotesAre(id, 3)
	})

	// QA-FLAG-4 — Flag a WBS that does not exist.
	register("QA-FLAG-4 flag missing wbs", func(t *T) {
		r := postJSON("/wbs/does-not-exist/risk-notes/flag", nil)
		t.errorCase(r, http.StatusNotFound, "WBS not found")
	})

	// --- Risk Note Editing (risk_note_editing.feature) ---------------------
	//
	// Shared setup: the three-task WBS, approved, then flagged with task 1 ->
	// ["SQL injection"], task 2 -> ["XSS"], task 3 -> []. flaggedWBS returns its id.
	flaggedWBS := func(t *T) string {
		id := approvedWBS(t)
		primeRisks(t, []riskAssignment{{1, "SQL injection"}, {2, "XSS"}})
		r := postJSON("/wbs/"+id+"/risk-notes/flag", nil)
		t.eq("setup flag status", r.status, http.StatusOK)
		return id
	}

	// QA-NOTE-1 — Add a risk note.
	register("QA-NOTE-1 add a risk note", func(t *T) {
		id := flaggedWBS(t)
		r := postJSON(riskNotePath(id, taskID(t, id, 3)), map[string]string{"description": "Missing rate limits"})
		t.eq("status", r.status, http.StatusCreated)
		t.riskNotesAre(id, 3, "Missing rate limits")
	})

	// QA-NOTE-2 — Edit a risk note.
	register("QA-NOTE-2 edit a risk note", func(t *T) {
		id := flaggedWBS(t)
		r := putJSON(riskNotePath(id, taskID(t, id, 1))+"/"+noteID(t, id, 1, 1),
			map[string]string{"description": "SQL injection via login"})
		t.eq("status", r.status, http.StatusOK)
		t.riskNotesAre(id, 1, "SQL injection via login")
	})

	// QA-NOTE-3 — Delete a risk note.
	register("QA-NOTE-3 delete a risk note", func(t *T) {
		id := flaggedWBS(t)
		r := del(riskNotePath(id, taskID(t, id, 1)) + "/" + noteID(t, id, 1, 1))
		t.eq("status", r.status, http.StatusOK)
		t.riskNotesAre(id, 1)
	})

	// QA-NOTE-4 — Reject an empty description (add and edit).
	register("QA-NOTE-4 reject empty risk note description", func(t *T) {
		id := flaggedWBS(t)
		add := postJSON(riskNotePath(id, taskID(t, id, 1)), map[string]string{"description": ""})
		t.errorCase(add, http.StatusBadRequest, "description is empty")
		edit := putJSON(riskNotePath(id, taskID(t, id, 1))+"/"+noteID(t, id, 1, 1),
			map[string]string{"description": ""})
		t.errorCase(edit, http.StatusBadRequest, "description is empty")
		t.riskNotesAre(id, 1, "SQL injection")
	})

	// QA-NOTE-5 — Reject editing or deleting a note that does not exist.
	register("QA-NOTE-5 reject unknown risk note", func(t *T) {
		id := flaggedWBS(t)
		edit := putJSON(riskNotePath(id, taskID(t, id, 1))+"/nonexistent",
			map[string]string{"description": "X"})
		t.errorCase(edit, http.StatusNotFound, "risk note not found")
		delR := del(riskNotePath(id, taskID(t, id, 1)) + "/nonexistent")
		t.errorCase(delR, http.StatusNotFound, "risk note not found")
		t.riskNotesAre(id, 1, "SQL injection")
	})
}
