package httpapi

import (
	"net/http"
	"testing"
)

// primeRisks seeds the next risk flagging via the QA affordance.
func (c *client) primeRisks(risks []map[string]any) {
	c.t.Helper()
	status, _ := c.json(http.MethodPost, "/qa/ai/next-risks", map[string]any{"risks": risks})
	if status != http.StatusNoContent {
		c.t.Fatalf("prime risks status = %d, want 204", status)
	}
}

func (c *client) approve(id string) {
	c.t.Helper()
	status, _ := c.do(http.MethodPost, "/wbs/"+id+"/approve", "", nil)
	if status != http.StatusOK {
		c.t.Fatalf("approve status = %d, want 200", status)
	}
}

func (c *client) flag(id string) (int, map[string]any) {
	c.t.Helper()
	return c.do(http.MethodPost, "/wbs/"+id+"/risk-notes/flag", "", nil)
}

// riskNotesOf returns the riskNotes array of the one-based task from a WBS body.
func riskNotesOf(t *testing.T, body map[string]any, taskNumber int) []map[string]any {
	t.Helper()
	tasks := tasksOf(t, body)
	if taskNumber < 1 || taskNumber > len(tasks) {
		t.Fatalf("task number %d out of range (%d tasks)", taskNumber, len(tasks))
	}
	raw, ok := tasks[taskNumber-1]["riskNotes"].([]any)
	if !ok {
		t.Fatalf("task %d has no riskNotes array: %v", taskNumber, tasks[taskNumber-1])
	}
	out := make([]map[string]any, len(raw))
	for i, r := range raw {
		out[i] = r.(map[string]any)
	}
	return out
}

func noteID(t *testing.T, body map[string]any, taskNumber, notePos int) string {
	t.Helper()
	notes := riskNotesOf(t, body, taskNumber)
	if notePos < 1 || notePos > len(notes) {
		t.Fatalf("note position %d out of range (%d notes on task %d)", notePos, len(notes), taskNumber)
	}
	return notes[notePos-1]["id"].(string)
}

// flaggedWBS sets up the standard approved-and-flagged WBS: task 1 has one note
// (SQL injection), task 2 has one (XSS), task 3 has none.
func (c *client) flaggedWBS() string {
	c.t.Helper()
	id := c.setupWBS()
	c.approve(id)
	c.primeRisks([]map[string]any{
		{"taskNumber": 1, "description": "SQL injection"},
		{"taskNumber": 2, "description": "XSS"},
	})
	if status, _ := c.flag(id); status != http.StatusOK {
		c.t.Fatalf("flag status = %d, want 200", status)
	}
	return id
}

func TestFlagApprovedWBSProducesPerTaskNotes(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.approve(id)
	c.primeRisks([]map[string]any{
		{"taskNumber": 1, "description": "SQL injection"},
		{"taskNumber": 1, "description": "Rate limiting"},
		{"taskNumber": 2, "description": "XSS"},
	})

	status, _ := c.flag(id)
	if status != http.StatusOK {
		t.Fatalf("flag status = %d, want 200", status)
	}
	_, got := c.get(id)
	if n := riskNotesOf(t, got, 1); len(n) != 2 || n[0]["description"] != "SQL injection" || n[1]["description"] != "Rate limiting" {
		t.Fatalf("task 1 notes = %v, want [SQL injection, Rate limiting]", n)
	}
	if n := riskNotesOf(t, got, 2); len(n) != 1 || n[0]["description"] != "XSS" {
		t.Fatalf("task 2 notes = %v, want [XSS]", n)
	}
	if n := riskNotesOf(t, got, 3); len(n) != 0 {
		t.Fatalf("task 3 notes = %v, want []", n)
	}
}

func TestFlagUnapprovedWBSRejected(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.primeRisks([]map[string]any{{"taskNumber": 1, "description": "SQL injection"}})

	status, body := c.flag(id)
	if status != http.StatusConflict || body["error"] != "WBS must be approved before risk flagging" {
		t.Fatalf("flag unapproved = %d %v, want 409 WBS must be approved before risk flagging", status, body)
	}
	_, got := c.get(id)
	if n := riskNotesOf(t, got, 1); len(n) != 0 {
		t.Fatalf("task 1 notes = %v, want [] (none created)", n)
	}
}

func TestReFlagReplacesAllNotes(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.approve(id)
	c.primeRisks([]map[string]any{{"taskNumber": 1, "description": "SQL injection"}, {"taskNumber": 2, "description": "XSS"}})
	c.flag(id)

	_, body := c.get(id)
	t3 := taskID(t, body, 3)
	status, _ := c.json(http.MethodPost, "/wbs/"+id+"/tasks/"+t3+"/risk-notes", map[string]any{"description": "Manual concern"})
	if status != http.StatusCreated {
		t.Fatalf("add manual note status = %d, want 201", status)
	}

	c.primeRisks([]map[string]any{{"taskNumber": 1, "description": "CSRF"}})
	c.flag(id)

	_, got := c.get(id)
	if n := riskNotesOf(t, got, 1); len(n) != 1 || n[0]["description"] != "CSRF" {
		t.Fatalf("task 1 notes = %v, want [CSRF]", n)
	}
	if n := riskNotesOf(t, got, 2); len(n) != 0 {
		t.Fatalf("task 2 notes = %v, want []", n)
	}
	if n := riskNotesOf(t, got, 3); len(n) != 0 {
		t.Fatalf("task 3 notes = %v, want [] (manual note replaced)", n)
	}
}

func TestFlagUnknownWBS(t *testing.T) {
	c := newClient(t, true)
	status, body := c.flag("does-not-exist")
	if status != http.StatusNotFound || body["error"] != "WBS not found" {
		t.Fatalf("flag unknown = %d %v, want 404 WBS not found", status, body)
	}
}

func TestAddRiskNote(t *testing.T) {
	c := newClient(t, true)
	id := c.flaggedWBS()
	_, body := c.get(id)
	t3 := taskID(t, body, 3)

	status, _ := c.json(http.MethodPost, "/wbs/"+id+"/tasks/"+t3+"/risk-notes", map[string]any{"description": "Missing rate limits"})
	if status != http.StatusCreated {
		t.Fatalf("add note status = %d, want 201", status)
	}
	_, got := c.get(id)
	if n := riskNotesOf(t, got, 3); len(n) != 1 || n[0]["description"] != "Missing rate limits" {
		t.Fatalf("task 3 notes = %v, want [Missing rate limits]", n)
	}
}

func TestAddRiskNoteRejectsEmptyDescription(t *testing.T) {
	c := newClient(t, true)
	id := c.flaggedWBS()
	_, body := c.get(id)
	t1 := taskID(t, body, 1)

	status, resp := c.json(http.MethodPost, "/wbs/"+id+"/tasks/"+t1+"/risk-notes", map[string]any{"description": ""})
	if status != http.StatusBadRequest || resp["error"] != "description is empty" {
		t.Fatalf("add empty = %d %v, want 400 description is empty", status, resp)
	}
	_, got := c.get(id)
	if n := riskNotesOf(t, got, 1); len(n) != 1 {
		t.Fatalf("task 1 notes = %v, want unchanged (1 note)", n)
	}
}

func TestAddRiskNoteUnknownTask(t *testing.T) {
	c := newClient(t, true)
	id := c.flaggedWBS()

	status, body := c.json(http.MethodPost, "/wbs/"+id+"/tasks/nonexistent/risk-notes", map[string]any{"description": "X"})
	if status != http.StatusNotFound || body["error"] != "task not found" {
		t.Fatalf("add unknown task = %d %v, want 404 task not found", status, body)
	}
}

func TestEditRiskNote(t *testing.T) {
	c := newClient(t, true)
	id := c.flaggedWBS()
	_, body := c.get(id)
	t1 := taskID(t, body, 1)
	nid := noteID(t, body, 1, 1)

	status, _ := c.json(http.MethodPut, "/wbs/"+id+"/tasks/"+t1+"/risk-notes/"+nid, map[string]any{"description": "SQL injection via login"})
	if status != http.StatusOK {
		t.Fatalf("edit note status = %d, want 200", status)
	}
	_, got := c.get(id)
	if n := riskNotesOf(t, got, 1); len(n) != 1 || n[0]["description"] != "SQL injection via login" {
		t.Fatalf("task 1 notes = %v, want [SQL injection via login]", n)
	}
}

func TestEditRiskNoteRejectsEmptyDescription(t *testing.T) {
	c := newClient(t, true)
	id := c.flaggedWBS()
	_, body := c.get(id)
	t1 := taskID(t, body, 1)
	nid := noteID(t, body, 1, 1)

	status, resp := c.json(http.MethodPut, "/wbs/"+id+"/tasks/"+t1+"/risk-notes/"+nid, map[string]any{"description": ""})
	if status != http.StatusBadRequest || resp["error"] != "description is empty" {
		t.Fatalf("edit empty = %d %v, want 400 description is empty", status, resp)
	}
}

func TestEditRiskNoteUnknownNote(t *testing.T) {
	c := newClient(t, true)
	id := c.flaggedWBS()
	_, body := c.get(id)
	t1 := taskID(t, body, 1)

	status, resp := c.json(http.MethodPut, "/wbs/"+id+"/tasks/"+t1+"/risk-notes/nonexistent", map[string]any{"description": "X"})
	if status != http.StatusNotFound || resp["error"] != "risk note not found" {
		t.Fatalf("edit unknown note = %d %v, want 404 risk note not found", status, resp)
	}
}

func TestDeleteRiskNote(t *testing.T) {
	c := newClient(t, true)
	id := c.flaggedWBS()
	_, body := c.get(id)
	t1 := taskID(t, body, 1)
	nid := noteID(t, body, 1, 1)

	status, _ := c.do(http.MethodDelete, "/wbs/"+id+"/tasks/"+t1+"/risk-notes/"+nid, "", nil)
	if status != http.StatusOK {
		t.Fatalf("delete note status = %d, want 200", status)
	}
	_, got := c.get(id)
	if n := riskNotesOf(t, got, 1); len(n) != 0 {
		t.Fatalf("task 1 notes = %v, want [] after delete", n)
	}
}

func TestDeleteRiskNoteUnknownNote(t *testing.T) {
	c := newClient(t, true)
	id := c.flaggedWBS()
	_, body := c.get(id)
	t1 := taskID(t, body, 1)

	status, resp := c.do(http.MethodDelete, "/wbs/"+id+"/tasks/"+t1+"/risk-notes/nonexistent", "", nil)
	if status != http.StatusNotFound || resp["error"] != "risk note not found" {
		t.Fatalf("delete unknown note = %d %v, want 404 risk note not found", status, resp)
	}
}

func TestRiskPrimingAffordanceDisabledWithoutMock(t *testing.T) {
	c := newClient(t, false)
	status, _ := c.json(http.MethodPost, "/qa/ai/next-risks", map[string]any{"risks": []any{}})
	if status != http.StatusNotFound {
		t.Fatalf("prime risks without mock = %d, want 404", status)
	}
}
