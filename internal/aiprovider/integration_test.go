package aiprovider

import (
	"os"
	"testing"
)

// TestGenerateAgainstRealAnthropic exercises the provider against the live
// Anthropic API. It builds the provider with no request options so the SDK
// resolves real credentials (ANTHROPIC_API_KEY, then an authenticated profile)
// and makes a genuine network call. It skips by default so the offline unit and
// property suites — and CI — stay deterministic; set ANTHROPIC_API_KEY to run it.
//
//	ANTHROPIC_API_KEY=sk-ant-... go test ./internal/aiprovider -run RealAnthropic -v
func TestGenerateAgainstRealAnthropic(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("set ANTHROPIC_API_KEY to run the live Anthropic integration test")
	}

	p := New("") // no fake transport → real client and network call

	tasks, err := p.Generate("Build a login system with email/password authentication and session management.")
	if err != nil {
		t.Fatalf("Generate against real Anthropic: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatal("expected at least one task from the model")
	}
	for i, task := range tasks {
		if task == "" {
			t.Fatalf("task %d is empty; parseTasks should have dropped blanks", i)
		}
	}
	t.Logf("real Anthropic returned %d tasks: %v", len(tasks), tasks)
}
