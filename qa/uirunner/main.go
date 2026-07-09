// Command uirunner executes the end-to-end QA suite for the two-stage
// estimation UI (qa/ui_pipeline_qa_suite.md) against a running estimationd
// instance, acting as an ordinary browser: it navigates the app's screens,
// submits the on-screen forms (the buttons and fields a user clicks and types
// into), and asserts on what the rendered screen shows.
//
// This runner speaks the *user interface* only. It drives the HTML screens at
// GET / and GET /proposal and the on-screen form actions under POST /ui/*
// (including the mock-only QA priming panel under /ui/qa/*, the sanctioned QA
// affordance of §1.2). It never touches the JSON API (/wbs, /qa/ai/*) and never
// inspects a network response body — every expectation is text or state visible
// in the returned HTML. It imports no estimation project package, so it can
// never reach into a private API. scripts/qa-ui.sh launches estimationd in mock
// mode and points this runner at it.
package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// baseURL is where estimationd is listening; scripts/qa-ui.sh sets QA_BASE_URL.
func baseURL() string {
	if v := os.Getenv("QA_BASE_URL"); v != "" {
		return v
	}
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return "http://127.0.0.1:8080"
}

// --- Browser client for the estimationd user interface ---------------------

// client follows the Post/Redirect/Get the UI uses: after a form action the
// server 303-redirects to the screen the user then sees, and the client fetches
// it, so every helper returns the resulting rendered screen HTML.
var client = &http.Client{Timeout: 5 * time.Second}

// screen fetches a screen (GET) and returns its rendered HTML.
func screen(path string) string {
	resp, err := client.Get(baseURL() + path)
	if err != nil {
		fatal("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return string(data)
}

// submit activates an on-screen form control: it posts form fields to the
// action the button targets and returns the screen the user lands on.
func submit(action string, fields url.Values) string {
	resp, err := client.PostForm(baseURL()+action, fields)
	if err != nil {
		fatal("POST %s: %v", action, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return string(data)
}

// submitRequirement submits the Requirement form's typed text (a multipart post,
// matching the enctype the on-screen form declares) and returns the Build
// screen.
func submitRequirement(text string) string {
	return submitMultipart(func(mw *multipart.Writer) {
		_ = mw.WriteField("requirement", text)
	})
}

// submitRequirementPDF submits the Requirement form's PDF upload (the "document"
// file part a user picks) and returns the Build screen.
func submitRequirementPDF(pdf []byte) string {
	return submitMultipart(func(mw *multipart.Writer) {
		part, err := mw.CreateFormFile("document", "requirement.pdf")
		if err != nil {
			fatal("create multipart part: %v", err)
		}
		if _, err := part.Write(pdf); err != nil {
			fatal("write multipart part: %v", err)
		}
	})
}

func submitMultipart(fill func(*multipart.Writer)) string {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fill(mw)
	if err := mw.Close(); err != nil {
		fatal("close multipart writer: %v", err)
	}
	resp, err := client.Post(baseURL()+"/ui/requirement", mw.FormDataContentType(), &buf)
	if err != nil {
		fatal("POST /ui/requirement: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return string(data)
}

// --- Minimal PDF construction (a black-box browser builds its own PDFs) -----

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

// --- On-screen user actions (buttons and fields) ---------------------------

// primeWBS seeds the next Generate through the QA panel's "Prime WBS" field.
func primeWBS(tasks string) { submit("/ui/qa/wbs", url.Values{"tasks": {tasks}}) }

// primeRisks seeds the next Flag risks through the QA panel's "Prime risks" field.
func primeRisks(risks string) { submit("/ui/qa/risks", url.Values{"risks": {risks}}) }

// primeEstimates seeds the next Generate estimates through the QA panel.
func primeEstimates(estimates string) {
	submit("/ui/qa/estimates", url.Values{"estimates": {estimates}})
}

// generateWBS performs the shared Stage-1 setup (§1.4): prime the standard three
// tasks, type a requirement, and activate Generate. Returns the Build screen.
func generateWBS() string {
	primeWBS("Login API; Login UI; Session store")
	return submitRequirement("Build a login system.")
}

func addTask(desc string) string {
	return submit("/ui/tasks/add", url.Values{"description": {desc}})
}
func editTask(number int, desc string) string {
	return submit("/ui/tasks/edit", url.Values{"number": {itoa(number)}, "description": {desc}})
}
func deleteTask(number int) string {
	return submit("/ui/tasks/delete", url.Values{"number": {itoa(number)}})
}
func approveWBS() string { return submit("/ui/wbs/approve", url.Values{}) }
func flagRisks() string  { return submit("/ui/risks/flag", url.Values{}) }
func addRiskNote(number int, desc string) string {
	return submit("/ui/risks/add", url.Values{"number": {itoa(number)}, "description": {desc}})
}
func generateEstimates() string { return submit("/ui/estimates/generate", url.Values{}) }
func approveEstimates() string  { return submit("/ui/estimates/approve", url.Values{}) }
func overrideEstimate(number, o, m, p int, reasoning string) string {
	return submit("/ui/estimates/override", url.Values{
		"number":      {itoa(number)},
		"optimistic":  {itoa(o)},
		"mostLikely":  {itoa(m)},
		"pessimistic": {itoa(p)},
		"reasoning":   {reasoning},
	})
}

// requestProposal opens the Proposal screen, enters the team inputs, and
// activates Generate, returning the resulting Proposal screen.
func requestProposal(velocity, capacity, rate int) string {
	return submit("/ui/proposal", url.Values{
		"velocity": {itoa(velocity)},
		"capacity": {itoa(capacity)},
		"rate":     {itoa(rate)},
	})
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

// --- Reading the screen (what a user sees) ---------------------------------

var (
	reTaskDesc = regexp.MustCompile(`name="description" value="([^"]*)"`)
	reCellO    = regexp.MustCompile(`<div class="cell c-o"><span class="num">(\d+)</span>`)
	reCellM    = regexp.MustCompile(`<div class="cell c-m"><span class="num">(\d+)</span>`)
	reCellP    = regexp.MustCompile(`<div class="cell c-p"><span class="num">(\d+)</span>`)
	reRiskText = regexp.MustCompile(`<span class="rt">([^<]*)</span>`)
	reScopeIt  = regexp.MustCompile(`<span class="st">([^<]*)</span>`)
	reAssumeIt = regexp.MustCompile(`<span class="at">([^<]*)</span>`)
	reTag      = regexp.MustCompile(`<[^>]*>`)
)

// taskDescriptions returns the WBS list's task descriptions, top to bottom, as
// shown in the editable rows.
func taskDescriptions(html string) []string {
	m := reTaskDesc.FindAllStringSubmatch(html, -1)
	out := make([]string, len(m))
	for i, g := range m {
		out[i] = g[1]
	}
	return out
}

// task1Estimate returns task 1's optimistic/most-likely/pessimistic estimate
// cells (the first WBS row's O/M/P columns), and whether all three are shown.
func task1Estimate(html string) (o, m, p int, ok bool) {
	go1 := reCellO.FindStringSubmatch(html)
	gm1 := reCellM.FindStringSubmatch(html)
	gp1 := reCellP.FindStringSubmatch(html)
	if go1 == nil || gm1 == nil || gp1 == nil {
		return 0, 0, 0, false
	}
	return atoi(go1[1]), atoi(gm1[1]), atoi(gp1[1]), true
}

// errorText returns the inline error surfaced on the current screen (the Build
// requirement error or the Proposal banner error), or "" when none is shown.
func errorText(html string) string {
	for _, sel := range []struct{ start, end string }{
		{`<span class="inline-err">`, `</span>`},
		{`<div class="banner-err">`, `</div>`},
	} {
		if inner := between(html, sel.start, sel.end); inner != "" {
			return strings.TrimSpace(reTag.ReplaceAllString(inner, ""))
		}
	}
	return ""
}

// metricValue returns the project PERT rollup value shown in the Metrics &
// Pricing panel for the given label ("Expected effort", "Standard deviation",
// or "Rel. std dev (RSD)").
func metricValue(html, label string) string {
	// The value follows the label's </div>, either directly (SD/RSD) or nested
	// in a <span class="val"> (Expected effort), possibly across lines.
	return firstGroup(html, `(?s)`+regexp.QuoteMeta(label)+`</div>.*?>±?(\d+)`)
}

// pricingValue returns the pricing-strategy value shown for a key-value row
// ("Contract type" or "Pricing basis").
func pricingValue(html, key string) string {
	return firstGroup(html, regexp.QuoteMeta(key)+`</span><span class="v">([^<]*)</span>`)
}

// pricingFlag returns the pricing risk-band flag shown on the pricing gauge.
func pricingFlag(html string) string {
	return firstGroup(html, `<span class="mid [a-z]*">([a-z]+)</span>`)
}

// proposalBadge returns the value in a Proposal doc badge, e.g.
// proposalBadge(html, "Confidence") or proposalBadge(html, "Contract").
func proposalBadge(html, label string) string {
	return firstGroup(html, regexp.QuoteMeta(label)+`: ([^<]*)</span>`)
}

// firstGroup compiles pattern and returns the first capture group of its first
// match in html, trimmed, or "" when it does not match.
func firstGroup(html, pattern string) string {
	if m := regexp.MustCompile(pattern).FindStringSubmatch(html); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// bodyContent returns the screen's visible <body> region, excluding the <head>
// stylesheet — so absence checks test what the user sees, not CSS.
func bodyContent(html string) string { return between(html, "<body>", "</body>") }

// riskNotesUnder returns the risk notes shown under task number (1-based). Rows
// are div-based, so it splits the WBS list on the row marker and reads the note
// text (<span class="rt">…</span>) in that row's chunk.
func riskNotesUnder(html string, number int) []string {
	parts := strings.Split(html, `<div class="row">`)
	if number < 1 || number >= len(parts) {
		return nil
	}
	return allSubmatches(reRiskText, parts[number])
}

// scopeItems / assumptionItems return the Proposal screen's scope-of-work and
// assumptions-&-exclusions entries.
func scopeItems(html string) []string      { return allSubmatches(reScopeIt, html) }
func assumptionItems(html string) []string { return allSubmatches(reAssumeIt, html) }

func allSubmatches(re *regexp.Regexp, s string) []string {
	m := re.FindAllStringSubmatch(s, -1)
	out := make([]string, len(m))
	for i, g := range m {
		out[i] = strings.TrimSpace(g[1])
	}
	return out
}

// canFlagRisks / canGenerateEstimates report whether the Risks/Estimates
// actions are available: their button is present on the screen.
func canFlagRisks(html string) bool { return strings.Contains(html, ">Flag risks<") }
func canGenerateEstimates(html string) bool {
	return strings.Contains(html, ">Generate estimates<")
}

// proposalReachable reports whether the Build screen's nav offers the Proposal
// stage as a reachable link (rather than a locked label).
func proposalReachable(html string) bool {
	return strings.Contains(html, `href="/proposal"`)
}

// onProposalScreen reports whether the given screen is the Proposal screen (its
// Team-parameters form is shown) rather than the Build screen QA is bounced to.
func onProposalScreen(html string) bool { return strings.Contains(html, ">Team parameters<") }

func between(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	i += len(start)
	j := strings.Index(s[i:], end)
	if j < 0 {
		return ""
	}
	return s[i : i+j]
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// --- Test harness ----------------------------------------------------------

// T accumulates assertion results for one QA case.
type T struct {
	name   string
	failed bool
}

var (
	cases    []func(*T)
	failures int
)

func (t *T) fail(format string, args ...any) {
	t.failed = true
	fmt.Printf("  FAIL %s: %s\n", t.name, fmt.Sprintf(format, args...))
}

func (t *T) eq(what string, got, want any) {
	if got != want {
		t.fail("%s = %v, want %v", what, got, want)
	}
}

// shows asserts a substring is visible on the screen.
func (t *T) shows(html, want string) {
	if !strings.Contains(html, want) {
		t.fail("screen does not show %q", want)
	}
}

// hides asserts a substring is not visible on the screen.
func (t *T) hides(html, unwanted string) {
	if strings.Contains(html, unwanted) {
		t.fail("screen still shows %q", unwanted)
	}
}

// listShows asserts a value appears among a set of list items shown on screen
// (e.g. the risk notes under a task, or the proposal's scope items).
func (t *T) listShows(items []string, want string) {
	for _, it := range items {
		if it == want {
			return
		}
	}
	t.fail("shown list %v does not include %q", items, want)
}

// inlineErrorIs asserts the screen shows exactly the given inline error.
func (t *T) inlineErrorIs(html, message string) {
	if got := errorText(html); got != message {
		t.fail("inline error = %q, want %q", got, message)
	}
}

// tasksAre asserts the WBS list shows exactly these task descriptions in order.
func (t *T) tasksAre(html string, want ...string) {
	got := taskDescriptions(html)
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

// task1EstimateIs asserts task 1's O/M/P cells show the given triple.
func (t *T) task1EstimateIs(html string, o, m, p int) {
	go1, gm1, gp1, ok := task1Estimate(html)
	if !ok {
		t.fail("task 1 estimate not shown")
		return
	}
	t.eq("task 1 optimistic", go1, o)
	t.eq("task 1 mostLikely", gm1, m)
	t.eq("task 1 pessimistic", gp1, p)
}

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
	fmt.Printf("UI QA suite: two-stage estimation UI (Build → Proposal) @ %s\n", baseURL())
	for _, c := range cases {
		c(&T{})
	}
	fmt.Printf("\nqa-ui: %d passed, %d failed\n", len(cases)-failures, failures)
	if failures > 0 {
		os.Exit(1)
	}
}

// waitReady blocks until estimationd serves the Build screen, so the suite
// never races startup.
func waitReady() {
	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, err := client.Get(baseURL() + "/")
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
	fmt.Fprintf(os.Stderr, "qa uirunner: "+format+"\n", args...)
	os.Exit(2)
}

func main() {
	registerStage1Build()
	registerStage2Proposal()
	run()
}
