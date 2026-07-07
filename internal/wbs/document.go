package wbs

import (
	"fmt"
	"regexp"
	"strings"
)

// RequirementDocument is a submitted requirement in some input format. Read
// returns the requirement text, ErrEmptyRequirement when the document has no
// content, or ErrUnreadableDocument when it cannot be read.
type RequirementDocument interface {
	Read() (string, error)
}

type textDocument struct {
	content string
}

// NewTextDocument builds a plain-text requirement document.
func NewTextDocument(content string) RequirementDocument {
	return textDocument{content: content}
}

func (d textDocument) Read() (string, error) {
	trimmed := strings.TrimSpace(d.content)
	if trimmed == "" {
		return "", ErrEmptyRequirement
	}
	return trimmed, nil
}

type pdfDocument struct {
	data []byte
}

// NewPDFDocument builds a PDF requirement document from raw bytes.
func NewPDFDocument(data []byte) RequirementDocument {
	return pdfDocument{data: data}
}

func (d pdfDocument) Read() (string, error) {
	return extractPDFText(d.data)
}

var pdfTextOp = regexp.MustCompile(`\(([^)]*)\)\s*Tj`)

// extractPDFText reads the text drawn by a minimal PDF. A document that does not
// look like a well-formed PDF is unreadable; a well-formed PDF that draws no
// text is empty.
func extractPDFText(data []byte) (string, error) {
	s := strings.TrimSpace(string(data))
	if !strings.HasPrefix(s, "%PDF-") || !strings.Contains(s, "%%EOF") {
		return "", ErrUnreadableDocument
	}
	var b strings.Builder
	for _, m := range pdfTextOp.FindAllStringSubmatch(s, -1) {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(m[1])
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", ErrEmptyRequirement
	}
	return text, nil
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
