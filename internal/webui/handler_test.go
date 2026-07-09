package webui

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"estimation/internal/wbs"
)

func postForm(t *testing.T, h http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// postDocument posts path as a multipart form carrying a single "document" file
// part, mirroring a browser PDF upload.
func postDocument(t *testing.T, h http.Handler, path string, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("document", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write document part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGetRootServesBuildScreen(t *testing.T) {
	h := NewHandler(true)
	rec := get(t, h, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Requirement") {
		t.Error("build screen should show the Requirement section")
	}
}

func TestQAPrimeThenGenerateShowsTasks(t *testing.T) {
	h := NewHandler(true)
	postForm(t, h, "/ui/qa/wbs", url.Values{"tasks": {"Login API; Login UI; Session store"}})
	postForm(t, h, "/ui/requirement", url.Values{"requirement": {"build a login system"}})

	body := get(t, h, "/").Body.String()
	if !strings.Contains(body, "Login API") {
		t.Errorf("build screen should show the generated task Login API:\n%s", body[:min(400, len(body))])
	}
}

func TestUploadedDocumentGeneratesWBS(t *testing.T) {
	h := NewHandler(true)
	postForm(t, h, "/ui/qa/wbs", url.Values{"tasks": {"Login API; Login UI; Session store"}})
	postDocument(t, h, "/ui/requirement", "requirement.pdf", wbs.EncodeMinimalPDF("build a login system"))

	body := get(t, h, "/").Body.String()
	if !strings.Contains(body, "Login API") {
		t.Errorf("uploaded PDF should generate the WBS:\n%s", body[:min(400, len(body))])
	}
}

func TestProposalGateRedirectsWhenUnreachable(t *testing.T) {
	h := NewHandler(true)
	rec := get(t, h, "/proposal")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 redirect", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want /", loc)
	}
}

func TestQAEndpointsHiddenWithoutMock(t *testing.T) {
	h := NewHandler(false)
	rec := postForm(t, h, "/ui/qa/wbs", url.Values{"tasks": {"Login API"}})
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when not in mock mode", rec.Code)
	}
}

func TestProposalScreenHidesMechanicsEndToEnd(t *testing.T) {
	h := NewHandler(true)
	postForm(t, h, "/ui/qa/wbs", url.Values{"tasks": {"Login API; Login UI; Session store"}})
	postForm(t, h, "/ui/requirement", url.Values{"requirement": {"build a login system"}})
	postForm(t, h, "/ui/wbs/approve", nil)
	postForm(t, h, "/ui/qa/risks", url.Values{"risks": {"task 1: SQL injection"}})
	postForm(t, h, "/ui/risks/flag", nil)
	postForm(t, h, "/ui/qa/estimates", url.Values{"estimates": {"task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20"}})
	postForm(t, h, "/ui/estimates/generate", nil)
	postForm(t, h, "/ui/estimates/approve", nil)
	postForm(t, h, "/ui/proposal", url.Values{"velocity": {"3"}, "capacity": {"30"}, "rate": {"120"}})

	body := get(t, h, "/proposal").Body.String()
	for _, want := range []string{"Login API", "SQL injection", "fixed-price-with-buffer"} {
		if !strings.Contains(body, want) {
			t.Errorf("proposal screen missing %q", want)
		}
	}
	for _, marker := range mechanicsMarkers {
		if strings.Contains(body, marker) {
			t.Errorf("proposal screen leaks mechanics marker %q", marker)
		}
	}
}
