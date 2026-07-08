package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"estimation/internal/wbs"
)

// client wraps a test server with helpers that mirror how QA drives the API.
type client struct {
	t    *testing.T
	srv  *httptest.Server
	http *http.Client
}

func newClient(t *testing.T, mock bool) *client {
	t.Helper()
	srv := httptest.NewServer(NewServer(mock))
	t.Cleanup(srv.Close)
	return &client{t: t, srv: srv, http: srv.Client()}
}

func (c *client) do(method, path string, contentType string, body io.Reader) (int, map[string]any) {
	c.t.Helper()
	req, err := http.NewRequest(method, c.srv.URL+path, body)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var decoded map[string]any
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &decoded); err != nil {
			c.t.Fatalf("decode %s %s body %q: %v", method, path, raw, err)
		}
	}
	return resp.StatusCode, decoded
}

func (c *client) json(method, path string, payload any) (int, map[string]any) {
	c.t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		c.t.Fatalf("marshal: %v", err)
	}
	return c.do(method, path, "application/json", bytes.NewReader(b))
}

func (c *client) prime(tasks ...string) {
	c.t.Helper()
	status, _ := c.json(http.MethodPost, "/qa/ai/next-wbs", map[string]any{"tasks": tasks})
	if status != http.StatusNoContent {
		c.t.Fatalf("prime status = %d, want 204", status)
	}
}

func (c *client) generateText(requirement string) (int, map[string]any) {
	c.t.Helper()
	return c.json(http.MethodPost, "/wbs", map[string]any{"requirement": requirement})
}

func (c *client) generatePDF(pdf []byte) (int, map[string]any) {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("document", "requirement.pdf")
	if err != nil {
		c.t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(pdf); err != nil {
		c.t.Fatalf("write pdf: %v", err)
	}
	mw.Close()
	return c.do(http.MethodPost, "/wbs", mw.FormDataContentType(), &buf)
}

func tasksOf(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, ok := body["tasks"].([]any)
	if !ok {
		t.Fatalf("body has no tasks array: %v", body)
	}
	out := make([]map[string]any, len(raw))
	for i, r := range raw {
		out[i] = r.(map[string]any)
	}
	return out
}

func taskID(t *testing.T, body map[string]any, number int) string {
	t.Helper()
	tasks := tasksOf(t, body)
	if number < 1 || number > len(tasks) {
		t.Fatalf("task number %d out of range (%d tasks)", number, len(tasks))
	}
	return tasks[number-1]["id"].(string)
}

// setupWBS generates the standard 3-task unapproved WBS and returns its id.
func (c *client) setupWBS() string {
	c.t.Helper()
	c.prime("Login API", "Login UI", "Session store")
	status, body := c.generateText("Build a login system.")
	if status != http.StatusCreated {
		c.t.Fatalf("setup generate status = %d, want 201", status)
	}
	return body["id"].(string)
}

func (c *client) get(id string) (int, map[string]any) {
	c.t.Helper()
	return c.do(http.MethodGet, "/wbs/"+id, "", nil)
}

// assertNotFound checks a response reports a missing resource as 404 with the
// given error message (e.g. "WBS not found" or "task not found").
func assertNotFound(t *testing.T, status int, body map[string]any, wantError string) {
	t.Helper()
	if status != http.StatusNotFound || body["error"] != wantError {
		t.Fatalf("want 404 %q, got %d %v", wantError, status, body)
	}
}

// assertUnknownWBS404 drives op against a nonexistent WBS id and asserts the
// response is 404 WBS not found. Shared by the GET and risk-flag endpoints.
func assertUnknownWBS404(t *testing.T, op func(*client, string) (int, map[string]any)) {
	t.Helper()
	c := newClient(t, true)
	status, body := op(c, "does-not-exist")
	assertNotFound(t, status, body, "WBS not found")
}

// assertUnknownTask404 sends a description request via method to path, which must
// target a nonexistent task, and asserts 404 task not found. Shared by the task
// and risk-note endpoints.
func (c *client) assertUnknownTask404(method, path string) {
	c.t.Helper()
	status, body := c.json(method, path, map[string]any{"description": "X"})
	assertNotFound(c.t, status, body, "task not found")
}

func TestGenerateTextWBS(t *testing.T) {
	c := newClient(t, true)
	c.prime("Login API", "Login UI", "Session store")

	status, body := c.generateText("Build a login system.")
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", status)
	}
	if body["state"] != "unapproved" {
		t.Fatalf("state = %v, want unapproved", body["state"])
	}
	tasks := tasksOf(t, body)
	if len(tasks) != 3 || tasks[0]["description"] != "Login API" {
		t.Fatalf("tasks = %v, want 3 with first Login API", tasks)
	}

	id := body["id"].(string)
	status, got := c.get(id)
	if status != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", status)
	}
	if got["approvedTasks"] != nil {
		t.Fatalf("approvedTasks = %v, want null", got["approvedTasks"])
	}
	if len(tasksOf(t, got)) != 3 {
		t.Fatalf("GET returned %d tasks, want 3", len(tasksOf(t, got)))
	}
}

func TestGeneratePDFWBS(t *testing.T) {
	c := newClient(t, true)
	c.prime("Charge API", "Refund API")

	status, body := c.generatePDF(wbs.EncodeMinimalPDF("Build a payment flow."))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", status)
	}
	if len(tasksOf(t, body)) != 2 {
		t.Fatalf("tasks = %d, want 2", len(tasksOf(t, body)))
	}
}

func TestGenerateRejectsEmptyRequirement(t *testing.T) {
	c := newClient(t, true)

	status, body := c.generateText("")
	if status != http.StatusBadRequest || body["error"] != "requirement is empty" {
		t.Fatalf("text empty = %d %v, want 400 requirement is empty", status, body)
	}

	status, body = c.generatePDF(wbs.EncodeMinimalPDF(""))
	if status != http.StatusBadRequest || body["error"] != "requirement is empty" {
		t.Fatalf("pdf empty = %d %v, want 400 requirement is empty", status, body)
	}
}

func TestGenerateRejectsCorruptPDF(t *testing.T) {
	c := newClient(t, true)

	status, body := c.generatePDF([]byte("%PDF-broken"))
	if status != http.StatusBadRequest || body["error"] != "document could not be read" {
		t.Fatalf("corrupt = %d %v, want 400 document could not be read", status, body)
	}
}

func TestGetUnknownWBS(t *testing.T) {
	assertUnknownWBS404(t, (*client).get)
}

func TestAddTask(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()

	status, _ := c.json(http.MethodPost, "/wbs/"+id+"/tasks", map[string]any{"description": "Password reset"})
	if status != http.StatusCreated {
		t.Fatalf("add status = %d, want 201", status)
	}
	_, body := c.get(id)
	tasks := tasksOf(t, body)
	if len(tasks) != 4 || tasks[3]["description"] != "Password reset" {
		t.Fatalf("after add: %v, want 4 with last Password reset", tasks)
	}
}

func TestAddTaskRejectsEmptyDescription(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()

	status, body := c.json(http.MethodPost, "/wbs/"+id+"/tasks", map[string]any{"description": ""})
	if status != http.StatusBadRequest || body["error"] != "description is empty" {
		t.Fatalf("add empty = %d %v, want 400 description is empty", status, body)
	}
	_, got := c.get(id)
	if len(tasksOf(t, got)) != 3 {
		t.Fatalf("task count changed after rejected add")
	}
}

func TestEditTask(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	_, body := c.get(id)
	tid := taskID(t, body, 1)

	status, _ := c.json(http.MethodPut, "/wbs/"+id+"/tasks/"+tid, map[string]any{"description": "OAuth login API"})
	if status != http.StatusOK {
		t.Fatalf("edit status = %d, want 200", status)
	}
	_, got := c.get(id)
	tasks := tasksOf(t, got)
	if len(tasks) != 3 || tasks[0]["description"] != "OAuth login API" {
		t.Fatalf("after edit: %v, want task 1 OAuth login API", tasks)
	}
}

func TestEditTaskRejectsEmptyDescription(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	_, body := c.get(id)
	tid := taskID(t, body, 2)

	status, resp := c.json(http.MethodPut, "/wbs/"+id+"/tasks/"+tid, map[string]any{"description": ""})
	if status != http.StatusBadRequest || resp["error"] != "description is empty" {
		t.Fatalf("edit empty = %d %v, want 400 description is empty", status, resp)
	}
}

// Every task-scoped endpoint resolves the task through the same guard, so all of
// them report an unknown task as 404 task not found.
func TestUnknownTaskEndpointsReturn404(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(*client) string
		method string
		path   func(id string) string
	}{
		{"edit task", (*client).setupWBS, http.MethodPut, func(id string) string { return "/wbs/" + id + "/tasks/nonexistent" }},
		{"add risk note", (*client).flaggedWBS, http.MethodPost, func(id string) string { return "/wbs/" + id + "/tasks/nonexistent/risk-notes" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, true)
			id := tc.setup(c)
			c.assertUnknownTask404(tc.method, tc.path(id))
		})
	}
}

func TestDeleteTask(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	_, body := c.get(id)
	tid := taskID(t, body, 2)

	status, _ := c.do(http.MethodDelete, "/wbs/"+id+"/tasks/"+tid, "", nil)
	if status != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", status)
	}
	_, got := c.get(id)
	tasks := tasksOf(t, got)
	if len(tasks) != 2 {
		t.Fatalf("after delete: %d tasks, want 2", len(tasks))
	}
	for _, task := range tasks {
		if task["description"] == "Login UI" {
			t.Fatal("deleted task Login UI still present")
		}
	}
}

func TestDeleteUnknownTask(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()

	status, body := c.do(http.MethodDelete, "/wbs/"+id+"/tasks/nonexistent", "", nil)
	if status != http.StatusNotFound || body["error"] != "task not found" {
		t.Fatalf("delete unknown = %d %v, want 404 task not found", status, body)
	}
}

func TestApprove(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()

	status, body := c.do(http.MethodPost, "/wbs/"+id+"/approve", "", nil)
	if status != http.StatusOK || body["state"] != "approved" {
		t.Fatalf("approve = %d %v, want 200 approved", status, body)
	}
	_, got := c.get(id)
	if got["state"] != "approved" {
		t.Fatalf("GET state = %v, want approved", got["state"])
	}
	if len(tasksOf(t, mapAt(t, got, "approvedTasks"))) != 3 {
		t.Fatalf("approvedTasks length wrong: %v", got["approvedTasks"])
	}
}

func TestApproveEmptyWBSRejected(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	// delete all three tasks
	for i := 0; i < 3; i++ {
		_, body := c.get(id)
		tid := taskID(t, body, 1)
		c.do(http.MethodDelete, "/wbs/"+id+"/tasks/"+tid, "", nil)
	}

	status, body := c.do(http.MethodPost, "/wbs/"+id+"/approve", "", nil)
	if status != http.StatusConflict || body["error"] != "cannot approve an empty WBS" {
		t.Fatalf("approve empty = %d %v, want 409 cannot approve an empty WBS", status, body)
	}
	_, got := c.get(id)
	if got["state"] != "unapproved" || got["approvedTasks"] != nil {
		t.Fatalf("after rejected approve: state=%v approvedTasks=%v", got["state"], got["approvedTasks"])
	}
}

func TestEditingApprovedWBSReturnsToUnapproved(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.do(http.MethodPost, "/wbs/"+id+"/approve", "", nil)

	status, _ := c.json(http.MethodPost, "/wbs/"+id+"/tasks", map[string]any{"description": "Audit log"})
	if status != http.StatusCreated {
		t.Fatalf("add to approved status = %d, want 201", status)
	}
	_, got := c.get(id)
	if got["state"] != "unapproved" || got["approvedTasks"] != nil {
		t.Fatalf("after edit: state=%v approvedTasks=%v, want unapproved/null", got["state"], got["approvedTasks"])
	}
}

func TestPrimingAffordanceDisabledWithoutMock(t *testing.T) {
	c := newClient(t, false)
	status, _ := c.json(http.MethodPost, "/qa/ai/next-wbs", map[string]any{"tasks": []string{"X"}})
	if status != http.StatusNotFound {
		t.Fatalf("prime without mock = %d, want 404", status)
	}
}

func mapAt(t *testing.T, body map[string]any, key string) map[string]any {
	t.Helper()
	return map[string]any{"tasks": body[key]}
}
