package aiprovider

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"estimation/internal/wbs"
)

func riskToolUseResponse(inputJSON string) string {
	return `{
		"id": "msg_test",
		"type": "message",
		"role": "assistant",
		"model": "claude-opus-4-8",
		"content": [
			{"type": "tool_use", "id": "toolu_1", "name": "submit_risks", "input": ` + inputJSON + `}
		],
		"stop_reason": "tool_use",
		"stop_sequence": null,
		"usage": {"input_tokens": 10, "output_tokens": 20}
	}`
}

func loginTasks() []wbs.Task {
	return []wbs.Task{
		{ID: "t1", Description: "Login API"},
		{ID: "t2", Description: "Login UI"},
		{ID: "t3", Description: "Session store"},
	}
}

func TestFlagRisksReturnsAssignmentsFromToolUse(t *testing.T) {
	ft := &fakeTransport{status: 200, body: riskToolUseResponse(
		`{"risks": [{"taskNumber": 1, "description": "SQL injection"}, {"taskNumber": 1, "description": "Rate limiting"}, {"taskNumber": 2, "description": "XSS"}]}`)}
	p := newTestProvider(t, ft)

	got, err := p.FlagRisks(loginTasks())
	if err != nil {
		t.Fatalf("FlagRisks returned error: %v", err)
	}
	want := []wbs.RiskAssignment{
		{TaskNumber: 1, Description: "SQL injection"},
		{TaskNumber: 1, Description: "Rate limiting"},
		{TaskNumber: 2, Description: "XSS"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FlagRisks = %v, want %v", got, want)
	}
}

func TestFlagRisksSendsToolAndTaskList(t *testing.T) {
	ft := &fakeTransport{status: 200, body: riskToolUseResponse(`{"risks": []}`)}
	p := newTestProvider(t, ft)

	if _, err := p.FlagRisks(loginTasks()); err != nil {
		t.Fatalf("FlagRisks returned error: %v", err)
	}
	if !strings.Contains(ft.lastBody, "submit_risks") {
		t.Fatalf("request did not include the submit_risks tool: %s", ft.lastBody)
	}
	if !strings.Contains(ft.lastBody, "Login API") || !strings.Contains(ft.lastBody, "Session store") {
		t.Fatalf("request did not include the task descriptions: %s", ft.lastBody)
	}
}

func TestFlagRisksReturnsEmptyWhenNoRisks(t *testing.T) {
	ft := &fakeTransport{status: 200, body: riskToolUseResponse(`{"risks": []}`)}
	p := newTestProvider(t, ft)

	got, err := p.FlagRisks(loginTasks())
	if err != nil {
		t.Fatalf("FlagRisks returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("FlagRisks = %v, want no assignments", got)
	}
}

func TestFlagRisksDropsBlankAndInvalidRisks(t *testing.T) {
	ft := &fakeTransport{status: 200, body: riskToolUseResponse(
		`{"risks": [{"taskNumber": 1, "description": "  SQL injection  "}, {"taskNumber": 1, "description": ""}, {"taskNumber": 0, "description": "no task"}]}`)}
	p := newTestProvider(t, ft)

	got, err := p.FlagRisks(loginTasks())
	if err != nil {
		t.Fatalf("FlagRisks returned error: %v", err)
	}
	want := []wbs.RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FlagRisks = %v, want %v", got, want)
	}
}

func TestFlagRisksErrorsWhenNoToolUse(t *testing.T) {
	body := `{
		"id": "msg_test", "type": "message", "role": "assistant", "model": "claude-opus-4-8",
		"content": [{"type": "text", "text": "I cannot help with that."}],
		"stop_reason": "end_turn", "stop_sequence": null,
		"usage": {"input_tokens": 5, "output_tokens": 5}
	}`
	ft := &fakeTransport{status: 200, body: body}
	p := newTestProvider(t, ft)

	if _, err := p.FlagRisks(loginTasks()); !errors.Is(err, ErrNoRiskSubmission) {
		t.Fatalf("FlagRisks no-tool error = %v, want ErrNoRiskSubmission", err)
	}
}

func TestFlagRisksPropagatesAPIError(t *testing.T) {
	ft := &fakeTransport{status: 401, body: `{"type": "error", "error": {"type": "authentication_error", "message": "invalid x-api-key"}}`}
	p := newTestProvider(t, ft)

	_, err := p.FlagRisks(loginTasks())
	if err == nil {
		t.Fatal("FlagRisks should return an error on API failure")
	}
	if !strings.Contains(err.Error(), "anthropic flag risks") {
		t.Fatalf("error = %v, want it wrapped with anthropic flag risks context", err)
	}
}

// Ensure AnthropicProvider satisfies the risk provider port.
var _ wbs.RiskProvider = (*AnthropicProvider)(nil)
