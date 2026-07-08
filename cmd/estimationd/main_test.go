package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// primeStatus posts to the QA priming affordance and returns the status code.
// The affordance responds 204 only when the server was built in mock mode.
func primeStatus(h http.Handler) int {
	req := httptest.NewRequest(http.MethodPost, "/qa/ai/next-wbs", strings.NewReader(`{"tasks":["X"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// primeRisksStatus posts to the risk priming affordance and returns the status
// code. Like the WBS affordance, it responds 204 only in mock mode.
func primeRisksStatus(h http.Handler) int {
	req := httptest.NewRequest(http.MethodPost, "/qa/ai/next-risks", strings.NewReader(`{"risks":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
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
