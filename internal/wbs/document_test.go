package wbs

import (
	"bytes"
	"errors"
	"testing"
)

func TestTextDocumentReturnsContent(t *testing.T) {
	got, err := NewTextDocument("Build a login system").Requirement()
	if err != nil {
		t.Fatalf("Requirement returned error: %v", err)
	}
	if got.Text != "Build a login system" {
		t.Fatalf("Requirement.Text = %q, want the document content", got.Text)
	}
	if got.PDF != nil {
		t.Fatalf("text document must not carry PDF bytes, got %d bytes", len(got.PDF))
	}
}

func TestTextDocumentRejectsEmptyContent(t *testing.T) {
	_, err := NewTextDocument("   ").Requirement()
	if !errors.Is(err, ErrEmptyRequirement) {
		t.Fatalf("Requirement empty error = %v, want ErrEmptyRequirement", err)
	}
}

func TestPDFDocumentPassesRawBytesThrough(t *testing.T) {
	pdf := EncodeMinimalPDF("Build a payments system")
	got, err := NewPDFDocument(pdf).Requirement()
	if err != nil {
		t.Fatalf("Requirement returned error: %v", err)
	}
	if got.Text != "" {
		t.Fatalf("PDF document must not carry text, got %q", got.Text)
	}
	if !bytes.Equal(got.PDF, pdf) {
		t.Fatalf("Requirement.PDF must be the raw PDF bytes, unmodified")
	}
}

func TestPDFDocumentRejectsEmptyContent(t *testing.T) {
	pdf := EncodeMinimalPDF("")
	_, err := NewPDFDocument(pdf).Requirement()
	if !errors.Is(err, ErrEmptyRequirement) {
		t.Fatalf("Requirement empty-pdf error = %v, want ErrEmptyRequirement", err)
	}
}

func TestPDFDocumentRejectsCorruptBytes(t *testing.T) {
	_, err := NewPDFDocument([]byte("this is not a pdf at all")).Requirement()
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("Requirement corrupt error = %v, want ErrUnreadableDocument", err)
	}
}

func TestPDFDocumentRejectsTruncatedFile(t *testing.T) {
	pdf := EncodeMinimalPDF("Build something")
	truncated := pdf[:len(pdf)-len("%%EOF\n")]
	_, err := NewPDFDocument(truncated).Requirement()
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("Requirement truncated error = %v, want ErrUnreadableDocument", err)
	}
}
