package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// postStatus posts body to path on h and returns the status code. The QA
// priming affordances respond 204 only when the server was built in mock mode.
func postStatus(h http.Handler, path, body string) int {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func primeStatus(h http.Handler) int {
	return postStatus(h, "/qa/ai/next-wbs", `{"tasks":["X"]}`)
}

func primeRisksStatus(h http.Handler) int {
	return postStatus(h, "/qa/ai/next-risks", `{"risks":[]}`)
}

func primeEstimatesStatus(h http.Handler) int {
	return postStatus(h, "/qa/ai/next-estimates", `{"estimates":[]}`)
}

func TestBuildServerMockEnablesPrimingAffordance(t *testing.T) {
	h, err := buildServer("mock", "")
	if err != nil {
		t.Fatalf("buildServer(mock): %v", err)
	}
	if code := primeStatus(h); code != http.StatusNoContent {
		t.Fatalf("mock priming status = %d, want 204", code)
	}
	if code := primeRisksStatus(h); code != http.StatusNoContent {
		t.Fatalf("mock risk priming status = %d, want 204", code)
	}
	if code := primeEstimatesStatus(h); code != http.StatusNoContent {
		t.Fatalf("mock estimate priming status = %d, want 204", code)
	}
}

func TestBuildServerNonMockDisablesPrimingAffordance(t *testing.T) {
	// The anthropic arm, the empty default, and any unrecognized provider all
	// build a server without the QA priming affordance.
	for _, provider := range []string{"anthropic", "", "unrecognized"} {
		h, err := buildServer(provider, "")
		if err != nil {
			t.Fatalf("buildServer(%q): %v", provider, err)
		}
		if code := primeStatus(h); code != http.StatusNotFound {
			t.Fatalf("provider %q priming status = %d, want 404 (mock disabled)", provider, code)
		}
	}
}
