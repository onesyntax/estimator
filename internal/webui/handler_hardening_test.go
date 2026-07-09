//go:build hardening

// Hardening tests for the UI handler, behind the `hardening` build tag so they
// stay out of the unit suite, coverage, CRAP, and DRY runs.
package webui

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"estimation/internal/wbs"
)

// requirementRequest builds a POST request carrying an optional "document" file
// part (present only when withDoc is true) and a "requirement" text field, so a
// hardening test can drive requirementDocument's upload-vs-text decision directly.
func requirementRequest(t *testing.T, withDoc bool, doc []byte, requirement string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if withDoc {
		part, err := mw.CreateFormFile("document", "requirement.pdf")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(doc); err != nil {
			t.Fatalf("write document part: %v", err)
		}
	}
	if err := mw.WriteField("requirement", requirement); err != nil {
		t.Fatalf("write requirement field: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/ui/requirement", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// TestRequirementDocumentPrefersReadablePDFOverText pins the upload path: when a
// non-empty document part is present, requirementDocument must build a PDF
// document from the uploaded bytes and ignore the text field. This kills the
// `err == nil` -> `err != nil` mutation, which would fall through to the (here
// empty) text field and yield ErrEmptyRequirement instead of a readable PDF.
func TestRequirementDocumentPrefersReadablePDFOverText(t *testing.T) {
	doc := requirementDocument(requirementRequest(t, true, wbs.EncodeMinimalPDF("build a login system"), ""))
	req, err := doc.Requirement()
	if err != nil {
		t.Fatalf("readable PDF upload: Requirement() error = %v, want nil", err)
	}
	if len(req.PDF) == 0 || req.Text != "" {
		t.Fatalf("readable PDF upload: requirement = %+v, want PDF bytes and no text", req)
	}
}

// TestRequirementDocumentFallsBackToTextWhenDocumentEmpty pins the empty-part
// fallback: an empty document part must fall through to the text field. This
// kills `&& -> ||` and `len(data) > 0` -> `>= 0`, both of which would take the
// PDF path with zero bytes and produce an unreadable document instead of using
// the text requirement.
func TestRequirementDocumentFallsBackToTextWhenDocumentEmpty(t *testing.T) {
	doc := requirementDocument(requirementRequest(t, true, nil, "build a login system"))
	req, err := doc.Requirement()
	if err != nil {
		t.Fatalf("empty document part: Requirement() error = %v, want nil", err)
	}
	if req.Text != "build a login system" || len(req.PDF) != 0 {
		t.Fatalf("empty document part: requirement = %+v, want the text field and no PDF", req)
	}
}

// TestRequirementDocumentTreatsSingleBytePartAsUpload pins the exact `> 0`
// boundary: a one-byte document part is still an upload and goes to the PDF path
// (which then rejects it as unreadable, since it is not a well-formed PDF). This
// kills `len(data) > 0` -> `> 1`, which would misroute a one-byte part to the
// text field and accept it as a valid text requirement.
func TestRequirementDocumentTreatsSingleBytePartAsUpload(t *testing.T) {
	doc := requirementDocument(requirementRequest(t, true, []byte("x"), "build a login system"))
	if _, err := doc.Requirement(); !errors.Is(err, wbs.ErrUnreadableDocument) {
		t.Fatalf("one-byte document part: Requirement() error = %v, want ErrUnreadableDocument", err)
	}
}
