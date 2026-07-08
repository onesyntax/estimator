package aiprovider

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"

	"estimation/internal/wbs"
)

// fakeTransport returns a canned HTTP response (or error) and records the last
// request body, so the provider is exercised end-to-end without a network call.
type fakeTransport struct {
	status   int
	body     string
	err      error
	lastBody string
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		f.lastBody = string(b)
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil
}

func newTestProvider(t *testing.T, ft *fakeTransport) *AnthropicProvider {
	t.Helper()
	return New("",
		option.WithAPIKey("test-key"),
		option.WithHTTPClient(&http.Client{Transport: ft}),
		option.WithMaxRetries(0),
	)
}

func toolUseResponse(inputJSON string) string {
	return `{
		"id": "msg_test",
		"type": "message",
		"role": "assistant",
		"model": "claude-opus-4-8",
		"content": [
			{"type": "tool_use", "id": "toolu_1", "name": "submit_wbs", "input": ` + inputJSON + `}
		],
		"stop_reason": "tool_use",
		"stop_sequence": null,
		"usage": {"input_tokens": 10, "output_tokens": 20}
	}`
}

func TestGenerateReturnsTasksFromToolUse(t *testing.T) {
	ft := &fakeTransport{status: 200, body: toolUseResponse(`{"tasks": ["Login API", "Login UI", "Session store"]}`)}
	p := newTestProvider(t, ft)

	got, err := p.Generate(wbs.Requirement{Text: "Build a login system."})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	want := []string{"Login API", "Login UI", "Session store"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Generate = %v, want %v", got, want)
	}
}

func TestGenerateSendsToolAndRequirement(t *testing.T) {
	ft := &fakeTransport{status: 200, body: toolUseResponse(`{"tasks": ["A"]}`)}
	p := newTestProvider(t, ft)

	if _, err := p.Generate(wbs.Requirement{Text: "Build a payment flow."}); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !strings.Contains(ft.lastBody, "submit_wbs") {
		t.Fatalf("request did not include the submit_wbs tool: %s", ft.lastBody)
	}
	if !strings.Contains(ft.lastBody, "Build a payment flow.") {
		t.Fatalf("request did not include the requirement text: %s", ft.lastBody)
	}
}

func TestGenerateSendsPDFAsDocumentBlock(t *testing.T) {
	ft := &fakeTransport{status: 200, body: toolUseResponse(`{"tasks": ["A"]}`)}
	p := newTestProvider(t, ft)

	pdf := []byte("%PDF-1.4\nfake requirement body\n%%EOF\n")
	if _, err := p.Generate(wbs.Requirement{PDF: pdf}); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	// The PDF must be sent as a base64 document block, not extracted to text.
	if !strings.Contains(ft.lastBody, `"type":"document"`) {
		t.Fatalf("request did not include a document block: %s", ft.lastBody)
	}
	if !strings.Contains(ft.lastBody, "application/pdf") {
		t.Fatalf("document block did not declare the PDF media type: %s", ft.lastBody)
	}
	b64 := base64.StdEncoding.EncodeToString(pdf)
	if !strings.Contains(ft.lastBody, b64) {
		t.Fatalf("request did not carry the base64-encoded PDF bytes: %s", ft.lastBody)
	}
}

func TestGenerateTrimsAndDropsBlankTasks(t *testing.T) {
	ft := &fakeTransport{status: 200, body: toolUseResponse(`{"tasks": ["  Charge API  ", "", "   ", "Refund API"]}`)}
	p := newTestProvider(t, ft)

	got, err := p.Generate(wbs.Requirement{Text: "req"})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	want := []string{"Charge API", "Refund API"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Generate = %v, want %v", got, want)
	}
}

func TestGenerateErrorsWhenNoTasks(t *testing.T) {
	ft := &fakeTransport{status: 200, body: toolUseResponse(`{"tasks": []}`)}
	p := newTestProvider(t, ft)

	if _, err := p.Generate(wbs.Requirement{Text: "req"}); !errors.Is(err, ErrNoTasks) {
		t.Fatalf("Generate empty tasks error = %v, want ErrNoTasks", err)
	}
}

func TestGenerateErrorsWhenNoToolUse(t *testing.T) {
	body := `{
		"id": "msg_test", "type": "message", "role": "assistant", "model": "claude-opus-4-8",
		"content": [{"type": "text", "text": "I cannot help with that."}],
		"stop_reason": "end_turn", "stop_sequence": null,
		"usage": {"input_tokens": 5, "output_tokens": 5}
	}`
	ft := &fakeTransport{status: 200, body: body}
	p := newTestProvider(t, ft)

	if _, err := p.Generate(wbs.Requirement{Text: "req"}); !errors.Is(err, ErrNoTasks) {
		t.Fatalf("Generate no-tool error = %v, want ErrNoTasks", err)
	}
}

func TestGeneratePropagatesAPIError(t *testing.T) {
	ft := &fakeTransport{status: 401, body: `{"type": "error", "error": {"type": "authentication_error", "message": "invalid x-api-key"}}`}
	p := newTestProvider(t, ft)

	_, err := p.Generate(wbs.Requirement{Text: "req"})
	if err == nil {
		t.Fatal("Generate should return an error on API failure")
	}
	if !strings.Contains(err.Error(), "anthropic generate") {
		t.Fatalf("error = %v, want it wrapped with anthropic generate context", err)
	}
}
