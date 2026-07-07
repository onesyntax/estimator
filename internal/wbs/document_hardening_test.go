//go:build hardening

package wbs

import "testing"

// Kills extractPDFText `m[1] -> m[0]`: the extracted text must be the captured
// group (the drawn string), not the whole regex match including the "(...) Tj"
// delimiters. Uses an exact match rather than a substring check.
func TestHardeningExtractPDFTextReturnsExactDrawnText(t *testing.T) {
	pdf := EncodeMinimalPDF("Build a payments system")
	got, err := NewPDFDocument(pdf).Read()
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if got != "Build a payments system" {
		t.Fatalf("Read = %q, want exactly the drawn text", got)
	}
}

// Kills the text-join separator logic: multiple Tj operators must be joined with
// a single space in draw order. Uses a raw multi-operator PDF and asserts the
// exact joined result.
func TestHardeningExtractPDFTextJoinsMultipleOperatorsWithSpace(t *testing.T) {
	raw := []byte("%PDF-1.4\nBT (Charge) Tj (Refund) Tj (Report) Tj ET\n%%EOF\n")
	got, err := NewPDFDocument(raw).Read()
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if got != "Charge Refund Report" {
		t.Fatalf("Read = %q, want %q", got, "Charge Refund Report")
	}
}
