// Command runner executes the end-to-end QA suite for Module 1 (WBS Generator +
// Human Approval) against a running estimationd instance, acting as an ordinary
// HTTP client. It is the executable form of qa/wbs_qa_suite.md and must be kept
// aligned with it: every QA-* case below mirrors a case in that document.
//
// This runner speaks HTTP only. It deliberately imports no estimation project
// package, so it can never reach into a private API; the sole interface it
// exercises is the estimationd user interface. scripts/qa.sh launches
// estimationd in mock mode and points this runner at it.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// baseURL is where estimationd is listening; scripts/qa.sh sets QA_BASE_URL.
func baseURL() string {
	if v := os.Getenv("QA_BASE_URL"); v != "" {
		return v
	}
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return "http://127.0.0.1:8080"
}

// --- Response shapes (the documented public contract) ----------------------

type task struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	RiskNotes   []riskNote `json:"riskNotes"`
}

type riskNote struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type wbsView struct {
	ID            string `json:"id"`
	State         string `json:"state"`
	Tasks         []task `json:"tasks"`
	ApprovedTasks []task `json:"approvedTasks"`
}

// response is a decoded HTTP response: status plus the raw body, from which the
// helpers below extract either a WBS view or an error message.
type response struct {
	status int
	body   []byte
}

func (r response) view() wbsView {
	var v wbsView
	_ = json.Unmarshal(r.body, &v)
	return v
}

func (r response) errorMessage() string {
	var e struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(r.body, &e)
	return e.Error
}

// --- HTTP client for the estimationd user interface ------------------------

var client = &http.Client{Timeout: 5 * time.Second}

func do(method, path, contentType string, body []byte) response {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, baseURL()+path, rdr)
	if err != nil {
		fatal("build request %s %s: %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		fatal("request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return response{status: resp.StatusCode, body: data}
}

func getJSON(path string) response { return do(http.MethodGet, path, "", nil) }
func postJSON(path string, payload any) response {
	return do(http.MethodPost, path, "application/json", mustJSON(payload))
}
func putJSON(path string, payload any) response {
	return do(http.MethodPut, path, "application/json", mustJSON(payload))
}
func del(path string) response { return do(http.MethodDelete, path, "", nil) }

// postPDF submits raw PDF bytes as a multipart "document" part, exactly as a
// browser upload would.
func postPDF(path string, pdf []byte) response {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("document", "requirement.pdf")
	if err != nil {
		fatal("create multipart part: %v", err)
	}
	if _, err := part.Write(pdf); err != nil {
		fatal("write multipart part: %v", err)
	}
	if err := mw.Close(); err != nil {
		fatal("close multipart writer: %v", err)
	}
	return do(http.MethodPost, path, mw.FormDataContentType(), buf.Bytes())
}

func mustJSON(payload any) []byte {
	b, err := json.Marshal(payload)
	if err != nil {
		fatal("marshal payload: %v", err)
	}
	return b
}

// --- Minimal PDF construction (a black-box client builds its own PDFs) ------

// validPDF is a well-formed minimal PDF whose page draws the given text; an
// empty text yields a readable PDF that draws no text (an "empty requirement").
func validPDF(text string) []byte {
	if text == "" {
		return []byte("%PDF-1.4\n%%EOF\n")
	}
	return []byte("%PDF-1.4\nBT (" + text + ") Tj ET\n%%EOF\n")
}

// corruptPDF is bytes that are not a well-formed PDF.
func corruptPDF() []byte { return []byte("%PDF-broken") }

// --- Semantic operations on the WBS UI -------------------------------------

func prime(t *T, tasks []string) {
	r := postJSON("/qa/ai/next-wbs", map[string][]string{"tasks": tasks})
	t.eq("prime status", r.status, http.StatusNoContent)
}

// setupWBS primes the standard three tasks, generates a WBS from a valid text
// requirement, and returns its id. This is the shared setup for editing and
// approval cases.
func setupWBS(t *T) string {
	prime(t, []string{"Login API", "Login UI", "Session store"})
	r := postJSON("/wbs", map[string]string{"requirement": "Build a login system."})
	t.eq("setup generate status", r.status, http.StatusCreated)
	return r.view().ID
}

// taskID resolves "task number n" to its stable id via GET, mirroring the
// suite's rule: tasks[n-1].id in GET order.
func taskID(t *T, id string, n int) string {
	v := getJSON("/wbs/" + id).view()
	if n < 1 || n > len(v.Tasks) {
		t.fail("resolve task %d: only %d tasks present", n, len(v.Tasks))
		return ""
	}
	return v.Tasks[n-1].ID
}

// --- Test harness ----------------------------------------------------------

// T accumulates assertion results for one QA case.
type T struct {
	name   string
	failed bool
}

var failures int

func (t *T) fail(format string, args ...any) {
	t.failed = true
	fmt.Printf("  FAIL %s: %s\n", t.name, fmt.Sprintf(format, args...))
}

func (t *T) eq(what string, got, want any) {
	if got != want {
		t.fail("%s = %v, want %v", what, got, want)
	}
}

// errorCase asserts an error response: the documented status and message.
func (t *T) errorCase(r response, status int, message string) {
	t.eq("status", r.status, status)
	if msg := r.errorMessage(); msg != message {
		t.fail("error = %q, want %q", msg, message)
	}
}

// descriptions returns the ordered task descriptions of a WBS view.
func descriptions(v wbsView) []string {
	out := make([]string, len(v.Tasks))
	for i, tk := range v.Tasks {
		out[i] = tk.Description
	}
	return out
}

func (t *T) tasksAre(id string, want ...string) {
	got := descriptions(getJSON("/wbs/" + id).view())
	if len(got) != len(want) {
		t.fail("task count = %d, want %d (%v)", len(got), len(want), got)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.fail("task %d = %q, want %q", i+1, got[i], want[i])
		}
	}
}

var cases []func(*T)

func register(name string, fn func(*T)) {
	cases = append(cases, func(t *T) {
		t.name = name
		fn(t)
		if !t.failed {
			fmt.Printf("  PASS %s\n", name)
		} else {
			failures++
		}
	})
}

func run() {
	waitReady()
	fmt.Printf("QA suite: Module 1 (WBS Generator + Human Approval) @ %s\n", baseURL())
	for _, c := range cases {
		c(&T{})
	}
	fmt.Printf("\nqa: %d passed, %d failed\n", len(cases)-failures, failures)
	if failures > 0 {
		os.Exit(1)
	}
}

// waitReady blocks until estimationd answers, so the suite never races startup.
func waitReady() {
	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, err := client.Get(baseURL() + "/wbs/ready-probe")
		if err == nil {
			resp.Body.Close()
			return
		}
		if time.Now().After(deadline) {
			fatal("estimationd not reachable at %s: %v", baseURL(), err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "qa runner: "+format+"\n", args...)
	os.Exit(2)
}

func main() {
	registerGeneration()
	registerEditing()
	registerApproval()
	registerRisk()
	run()
}
