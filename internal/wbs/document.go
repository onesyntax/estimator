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
	// Drop the "%PDF-<version>" header line.
	if nl := strings.IndexByte(s, '\n'); nl >= 0 {
		s = s[nl+1:]
	} else {
		s = ""
	}
	// Keep only what precedes the first %%EOF marker.
	if eof := strings.Index(s, "%%EOF"); eof >= 0 {
		s = s[:eof]
	}
	return strings.TrimSpace(s), true
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
// {"version":1,"tested_at":"2026-07-07T22:19:06+05:30","module_hash":"a1aee252ac1022950d0fa10ee07f4d03eb684c6388a9fa91b269be60af1ec401","functions":[{"id":"func/NewTextDocument","name":"NewTextDocument","line":21,"end_line":23,"hash":"c6639f263cc55cb51997fb2b36a6c335c12daed436befdede9d95f652eb211a3"},{"id":"func/textDocument.Read","name":"textDocument.Read","line":25,"end_line":31,"hash":"36dd76d161d8a3970643c58d4a85e0033efc4238955ebb3990e42fd200a10884"},{"id":"func/NewPDFDocument","name":"NewPDFDocument","line":38,"end_line":40,"hash":"1aad6c362e3b93af5964e505ce4e5fc737e420e1afa17788e2f03fd03260f117"},{"id":"func/pdfDocument.Read","name":"pdfDocument.Read","line":42,"end_line":44,"hash":"56452c458dd44e6ad6c7cc8cb88614e72adcc2fe84875c9605bbb4ac755e0e69"},{"id":"func/extractPDFText","name":"extractPDFText","line":51,"end_line":65,"hash":"d73a011908ba83371fe4582a77271d18b7d57d7069817e922a11cbe6a32043e4"},{"id":"func/EncodeMinimalPDF","name":"EncodeMinimalPDF","line":69,"end_line":78,"hash":"3306eeffa4849eb32fff21859c187a589a1ec5e6fb2bcaaa53956d154e8e6af1"}]}
// mutate4go-manifest-end
