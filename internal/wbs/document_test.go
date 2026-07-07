package wbs

import (
	"errors"
	"strings"
	"testing"
)

func TestTextDocumentReturnsContent(t *testing.T) {
	got, err := NewTextDocument("Build a login system").Read()
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if got != "Build a login system" {
		t.Fatalf("Read = %q, want the document content", got)
	}
}

func TestTextDocumentRejectsEmptyContent(t *testing.T) {
	_, err := NewTextDocument("   ").Read()
	if !errors.Is(err, ErrEmptyRequirement) {
		t.Fatalf("Read empty error = %v, want ErrEmptyRequirement", err)
	}
}

func TestPDFDocumentReturnsExtractedText(t *testing.T) {
	pdf := EncodeMinimalPDF("Build a payments system")
	got, err := NewPDFDocument(pdf).Read()
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if !strings.Contains(got, "Build a payments system") {
		t.Fatalf("Read = %q, want extracted PDF text", got)
	}
}

func TestPDFDocumentRejectsEmptyContent(t *testing.T) {
	pdf := EncodeMinimalPDF("")
	_, err := NewPDFDocument(pdf).Read()
	if !errors.Is(err, ErrEmptyRequirement) {
		t.Fatalf("Read empty-pdf error = %v, want ErrEmptyRequirement", err)
	}
}

func TestPDFDocumentRejectsCorruptBytes(t *testing.T) {
	_, err := NewPDFDocument([]byte("this is not a pdf at all")).Read()
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("Read corrupt error = %v, want ErrUnreadableDocument", err)
	}
}

func TestPDFDocumentRejectsTruncatedFile(t *testing.T) {
	pdf := EncodeMinimalPDF("Build something")
	truncated := pdf[:len(pdf)-len("%%EOF\n")]
	_, err := NewPDFDocument(truncated).Read()
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("Read truncated error = %v, want ErrUnreadableDocument", err)
	}
}
