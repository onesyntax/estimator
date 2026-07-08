package wbs

import (
	"fmt"
	"strings"
)

// Requirement is a validated requirement ready to hand to a Provider. It carries
// exactly one of a plain-text requirement (Text) or a raw PDF document (PDF) for
// the provider to send to the model directly — the domain does not extract PDF
// text itself.
type Requirement struct {
	Text string
	PDF  []byte
}

// RequirementDocument is a submitted requirement in some input format.
// Requirement returns the validated requirement, ErrEmptyRequirement when the
// document has no content, or ErrUnreadableDocument when it cannot be read.
type RequirementDocument interface {
	Requirement() (Requirement, error)
}

type textDocument struct {
	content string
}

// NewTextDocument builds a plain-text requirement document.
func NewTextDocument(content string) RequirementDocument {
	return textDocument{content: content}
}

func (d textDocument) Requirement() (Requirement, error) {
	trimmed := strings.TrimSpace(d.content)
	if trimmed == "" {
		return Requirement{}, ErrEmptyRequirement
	}
	return Requirement{Text: trimmed}, nil
}

type pdfDocument struct {
	data []byte
}

// NewPDFDocument builds a PDF requirement document from raw bytes.
func NewPDFDocument(data []byte) RequirementDocument {
	return pdfDocument{data: data}
}

func (d pdfDocument) Requirement() (Requirement, error) {
	body, wellFormed := pdfBody(d.data)
	if !wellFormed {
		return Requirement{}, ErrUnreadableDocument
	}
	if body == "" {
		return Requirement{}, ErrEmptyRequirement
	}
	return Requirement{PDF: d.data}, nil
}

// pdfBody validates that data looks like a well-formed PDF (a %PDF- header and a
// %%EOF marker) and returns the trimmed body between the version header line and
// the first %%EOF marker. A well-formed PDF with an empty body (no objects,
// streams, or drawn content) yields "". The domain never parses the PDF's text —
// this is only enough to reject corrupt and content-free documents at the
// boundary; the model reads the actual requirement from the bytes.
func pdfBody(data []byte) (body string, wellFormed bool) {
	s := strings.TrimSpace(string(data))
	if !strings.HasPrefix(s, "%PDF-") || !strings.Contains(s, "%%EOF") {
		return "", false
	}
	// Drop the "%PDF-<version>" header line; a PDF that is a single line with no
	// newline has no body, so afterHeader is empty. Then keep only what precedes
	// the first %%EOF marker (the whole remainder when there is none in the body).
	_, afterHeader, _ := strings.Cut(s, "\n")
	beforeEOF, _, _ := strings.Cut(afterHeader, "%%EOF")
	return strings.TrimSpace(beforeEOF), true
}

// EncodeMinimalPDF produces a minimal well-formed PDF whose page draws the given
// text. Passing an empty string yields a readable PDF that draws no text.
func EncodeMinimalPDF(text string) []byte {
	var content strings.Builder
	content.WriteString("%PDF-1.4\n")
	if strings.TrimSpace(text) != "" {
		escaped := strings.NewReplacer("(", "", ")", "").Replace(text)
		fmt.Fprintf(&content, "BT (%s) Tj ET\n", escaped)
	}
	content.WriteString("%%EOF\n")
	return []byte(content.String())
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-08T23:58:51+05:30","module_hash":"f5f576a2c4f36281505959b4bcd0b3cdde650be133deb3a8623eb117bc991914","functions":[{"id":"func/NewTextDocument","name":"NewTextDocument","line":29,"end_line":31,"hash":"c6639f263cc55cb51997fb2b36a6c335c12daed436befdede9d95f652eb211a3"},{"id":"func/textDocument.Requirement","name":"textDocument.Requirement","line":33,"end_line":39,"hash":"dadd6dc78d1574cc25b1bff0f97e3c778570c43a777c53e4b7e2de92699e8353"},{"id":"func/NewPDFDocument","name":"NewPDFDocument","line":46,"end_line":48,"hash":"1aad6c362e3b93af5964e505ce4e5fc737e420e1afa17788e2f03fd03260f117"},{"id":"func/pdfDocument.Requirement","name":"pdfDocument.Requirement","line":50,"end_line":59,"hash":"a1afa56d644e6ebd334dbf461a5f0f9bac478dd83e71778289826746df1e025b"},{"id":"func/pdfBody","name":"pdfBody","line":67,"end_line":78,"hash":"07a65ea9339037dfaf99b2ad3acacdfe347d4123353855552b6889e98518e5cd"},{"id":"func/EncodeMinimalPDF","name":"EncodeMinimalPDF","line":82,"end_line":91,"hash":"3306eeffa4849eb32fff21859c187a589a1ec5e6fb2bcaaa53956d154e8e6af1"}]}
// mutate4go-manifest-end
