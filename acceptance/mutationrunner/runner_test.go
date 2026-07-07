package mutationrunner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// baseIR is a small feature whose example-free scenario passes against the real
// project step handlers: the background primes two tasks and generates a WBS,
// and the scenario asserts the resulting count and first task.
const baseIR = `{
  "name": "Runner Test",
  "background": [
    {"keyword": "Given", "text": "the estimation service is running with a deterministic AI provider"},
    {"keyword": "And", "text": "the AI is primed to generate the tasks Alpha; Beta"},
    {"keyword": "And", "text": "a Tech Lead has generated a WBS from a valid requirement document"}
  ],
  "scenarios": [
    {"name": "s", "steps": [
      {"keyword": "Then", "text": "the WBS contains 2 tasks"},
      {"keyword": "And", "text": "task number 1 is Alpha"}
    ], "examples": []}
  ]
}`

func writeIR(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "feature.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write IR: %v", err)
	}
	return path
}

func TestEvaluatePassingFeatureSurvives(t *testing.T) {
	outcome, output, errText := Evaluate(writeIR(t, baseIR))
	if outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q, want %q (err=%q)", outcome, OutcomeSuccess, errText)
	}
	if !strings.Contains(output, "PASS") {
		t.Fatalf("output = %q, want a PASS line", output)
	}
}

func TestEvaluateFailingFeatureIsKilled(t *testing.T) {
	// Wrong expected count: the acceptance run fails, i.e. a killed mutation.
	body := strings.Replace(baseIR, "the WBS contains 2 tasks", "the WBS contains 9 tasks", 1)
	outcome, _, _ := Evaluate(writeIR(t, body))
	if outcome != OutcomeFailure {
		t.Fatalf("outcome = %q, want %q", outcome, OutcomeFailure)
	}
}

func TestEvaluateMissingFileIsInfrastructureError(t *testing.T) {
	outcome, _, errText := Evaluate(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if outcome != OutcomeError {
		t.Fatalf("outcome = %q, want %q", outcome, OutcomeError)
	}
	if errText == "" {
		t.Fatal("expected a diagnostic for a missing IR file")
	}
}

func TestEvaluateMalformedIRIsInfrastructureError(t *testing.T) {
	outcome, _, errText := Evaluate(writeIR(t, "{ not valid json"))
	if outcome != OutcomeError {
		t.Fatalf("outcome = %q, want %q", outcome, OutcomeError)
	}
	if errText == "" {
		t.Fatal("expected a diagnostic for malformed IR")
	}
}

func TestServeProcessesJobsAndReportsPerLine(t *testing.T) {
	good := writeIR(t, baseIR)
	in := strings.NewReader(
		`{"id":"m1","feature_json":"` + good + `"}` + "\n" +
			`not-json` + "\n")
	var out strings.Builder
	if err := Serve(in, &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d response lines, want 2: %q", len(lines), out.String())
	}

	var first response
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("first response not JSON: %v", err)
	}
	if first.ID != "m1" || first.Outcome != OutcomeSuccess {
		t.Fatalf("first response = %+v, want id m1 outcome %q", first, OutcomeSuccess)
	}

	var second response
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("second response not JSON: %v", err)
	}
	if second.Outcome != OutcomeError {
		t.Fatalf("second outcome = %q, want %q for a malformed job line", second.Outcome, OutcomeError)
	}
}
