//go:build hardening

package wbs

import (
	"bytes"
	"errors"
	"testing"
)

// Kills the empty-body check `body == "" -> body != ""`: a well-formed PDF whose
// body is empty (no objects or drawn content) must be rejected as empty, while a
// PDF that carries a body must pass its raw bytes through untouched.
func TestHardeningPDFEmptyBodyIsRejectedButContentPassesThrough(t *testing.T) {
	if _, err := NewPDFDocument(EncodeMinimalPDF("")).Requirement(); !errors.Is(err, ErrEmptyRequirement) {
		t.Fatalf("empty-body PDF error = %v, want ErrEmptyRequirement", err)
	}

	pdf := EncodeMinimalPDF("Build a payments system")
	req, err := NewPDFDocument(pdf).Requirement()
	if err != nil {
		t.Fatalf("content PDF Requirement error: %v", err)
	}
	if !bytes.Equal(req.PDF, pdf) {
		t.Fatalf("Requirement.PDF must be the raw PDF bytes, unmodified")
	}
}

// Kills the well-formedness guard `!wellFormed -> wellFormed`: bytes missing the
// %%EOF marker are corrupt and must be rejected as unreadable, not treated as an
// empty (or valid) document.
func TestHardeningPDFMissingEOFIsUnreadable(t *testing.T) {
	raw := []byte("%PDF-1.4\nBT (Charge) Tj ET\n")
	if _, err := NewPDFDocument(raw).Requirement(); !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("missing-%%EOF error = %v, want ErrUnreadableDocument", err)
	}
}
