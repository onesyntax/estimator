package webui

import (
	"embed"
	"html/template"
	"io"
	"strings"
)

//go:embed templates/*.html
var templatesFS embed.FS

// templates holds the parsed Build and Proposal screen templates.
var templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

// RenderOptions carries per-render presentation flags. QAMode reveals the
// QA-only priming panel, which exists only in mock mode.
type RenderOptions struct {
	QAMode bool
}

// buildData and proposalData are the template payloads: the screen's view model
// plus the render options.
type buildData struct {
	View   BuildView
	QAMode bool
}

type proposalData struct {
	View   ProposalView
	QAMode bool
}

// RenderBuild writes the Stage-1 Build workspace HTML for the view.
func RenderBuild(w io.Writer, v BuildView, opts RenderOptions) error {
	return templates.ExecuteTemplate(w, "build", buildData{View: v, QAMode: opts.QAMode})
}

// RenderProposal writes the Stage-2 Proposal screen HTML for the view.
func RenderProposal(w io.Writer, v ProposalView, opts RenderOptions) error {
	return templates.ExecuteTemplate(w, "proposal", proposalData{View: v, QAMode: opts.QAMode})
}

// BuildHTML renders the current Build screen to a string. It is the on-screen
// state the acceptance specs and QA both read.
func (s *Session) BuildHTML() (string, error) {
	var b strings.Builder
	if err := RenderBuild(&b, s.BuildView(), RenderOptions{}); err != nil {
		return "", err
	}
	return b.String(), nil
}

// ProposalHTML renders the current Proposal screen to a string.
func (s *Session) ProposalHTML() (string, error) {
	var b strings.Builder
	if err := RenderProposal(&b, s.ProposalView(), RenderOptions{}); err != nil {
		return "", err
	}
	return b.String(), nil
}
