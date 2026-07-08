package aiprovider

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"estimation/internal/wbs"
)

func estimateToolUseResponse(inputJSON string) string {
	return `{
		"id": "msg_test",
		"type": "message",
		"role": "assistant",
		"model": "claude-opus-4-8",
		"content": [
			{"type": "tool_use", "id": "toolu_1", "name": "submit_estimates", "input": ` + inputJSON + `}
		],
		"stop_reason": "tool_use",
		"stop_sequence": null,
		"usage": {"input_tokens": 10, "output_tokens": 20}
	}`
}

func TestEstimateReturnsAssignmentsFromToolUse(t *testing.T) {
	ft := &fakeTransport{status: 200, body: estimateToolUseResponse(
		`{"estimates": [{"taskNumber": 1, "optimistic": 2, "mostLikely": 5, "pessimistic": 13, "reasoning": "clean"}, {"taskNumber": 2, "optimistic": 1, "mostLikely": 2, "pessimistic": 3, "reasoning": "trivial"}]}`)}
	p := newTestProvider(t, ft)

	got, err := p.Estimate(loginTasks())
	if err != nil {
		t.Fatalf("Estimate returned error: %v", err)
	}
	want := []wbs.EstimateAssignment{
		{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "clean"},
		{TaskNumber: 2, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "trivial"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Estimate = %v, want %v", got, want)
	}
}

func TestEstimateSendsToolAndTaskList(t *testing.T) {
	ft := &fakeTransport{status: 200, body: estimateToolUseResponse(`{"estimates": []}`)}
	p := newTestProvider(t, ft)

	if _, err := p.Estimate(loginTasks()); err != nil {
		t.Fatalf("Estimate returned error: %v", err)
	}
	if !strings.Contains(ft.lastBody, "submit_estimates") {
		t.Fatalf("request did not include the submit_estimates tool: %s", ft.lastBody)
	}
	if !strings.Contains(ft.lastBody, "Login API") || !strings.Contains(ft.lastBody, "Session store") {
		t.Fatalf("request did not include the task descriptions: %s", ft.lastBody)
	}
}

func TestEstimateDropsInvalidTaskNumbers(t *testing.T) {
	ft := &fakeTransport{status: 200, body: estimateToolUseResponse(
		`{"estimates": [{"taskNumber": 1, "optimistic": 2, "mostLikely": 5, "pessimistic": 13, "reasoning": "keep"}, {"taskNumber": 0, "optimistic": 1, "mostLikely": 2, "pessimistic": 3, "reasoning": "drop"}]}`)}
	p := newTestProvider(t, ft)

	got, err := p.Estimate(loginTasks())
	if err != nil {
		t.Fatalf("Estimate returned error: %v", err)
	}
	want := []wbs.EstimateAssignment{{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "keep"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Estimate = %v, want %v", got, want)
	}
}

func TestEstimateErrorsWhenNoToolUse(t *testing.T) {
	body := `{
		"id": "msg_test", "type": "message", "role": "assistant", "model": "claude-opus-4-8",
		"content": [{"type": "text", "text": "I cannot help with that."}],
		"stop_reason": "end_turn", "stop_sequence": null,
		"usage": {"input_tokens": 5, "output_tokens": 5}
	}`
	ft := &fakeTransport{status: 200, body: body}
	p := newTestProvider(t, ft)

	if _, err := p.Estimate(loginTasks()); !errors.Is(err, ErrNoEstimateSubmission) {
		t.Fatalf("Estimate no-tool error = %v, want ErrNoEstimateSubmission", err)
	}
}

func TestEstimatePropagatesAPIError(t *testing.T) {
	ft := &fakeTransport{status: 401, body: `{"type": "error", "error": {"type": "authentication_error", "message": "invalid x-api-key"}}`}
	p := newTestProvider(t, ft)

	_, err := p.Estimate(loginTasks())
	if err == nil {
		t.Fatal("Estimate should return an error on API failure")
	}
	if !strings.Contains(err.Error(), "anthropic estimate") {
		t.Fatalf("error = %v, want it wrapped with anthropic estimate context", err)
	}
}

// Ensure AnthropicProvider satisfies the estimate provider port.
var _ wbs.EstimateProvider = (*AnthropicProvider)(nil)
