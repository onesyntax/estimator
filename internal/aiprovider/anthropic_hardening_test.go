//go:build hardening

// Hardening tests for the Anthropic provider, behind the `hardening` build tag
// so they stay out of the unit suite, coverage, CRAP, and DRY runs.
package aiprovider

import (
	"errors"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"

	"estimation/internal/wbs"
)

// Kills New `selected == "" -> selected != ""`: an empty model id must select
// the default, and a supplied id must be used verbatim.
func TestHardeningNewSelectsModel(t *testing.T) {
	if got := New("").model; got != defaultModel {
		t.Fatalf("New(\"\").model = %q, want default %q", got, defaultModel)
	}
	custom := anthropic.Model("claude-sonnet-5")
	if got := New(string(custom)).model; got != custom {
		t.Fatalf("New(%q).model = %q, want it used verbatim", custom, got)
	}
}

// Kills requirementBlocks `len(req.PDF) > 0 -> > 1`: any non-empty PDF, even a
// single byte, must render as a base64 document block (followed by the
// instruction text block), while a text requirement renders as one text block.
func TestHardeningRequirementBlocksRoutesByPDF(t *testing.T) {
	pdf := requirementBlocks(wbs.Requirement{PDF: []byte{'%'}})
	if len(pdf) != 2 || pdf[0].OfDocument == nil {
		t.Fatalf("one-byte PDF requirement must lead with a document block, got %d blocks (OfDocument=%v)", len(pdf), pdf[0].OfDocument != nil)
	}

	text := requirementBlocks(wbs.Requirement{Text: "req"})
	if len(text) != 1 || text[0].OfText == nil {
		t.Fatalf("text requirement must render as a single text block, got %d blocks (OfText=%v)", len(text), text[0].OfText != nil)
	}
}

// Kills tasksFromMessage `ok && use.Name == toolName -> ok || ...`: a tool_use
// block whose name is not the submit tool must be ignored (yielding ErrNoTasks),
// not parsed as the WBS submission.
func TestHardeningTasksFromMessageIgnoresOtherTools(t *testing.T) {
	body := `{
		"id": "msg_test", "type": "message", "role": "assistant", "model": "claude-opus-4-8",
		"content": [{"type": "tool_use", "id": "toolu_1", "name": "some_other_tool", "input": {"tasks": ["X"]}}],
		"stop_reason": "tool_use", "stop_sequence": null,
		"usage": {"input_tokens": 5, "output_tokens": 5}
	}`
	p := newTestProvider(t, &fakeTransport{status: 200, body: body})

	if _, err := p.Generate(wbs.Requirement{Text: "req"}); !errors.Is(err, ErrNoTasks) {
		t.Fatalf("Generate with a non-submit_wbs tool = %v, want ErrNoTasks", err)
	}
}
