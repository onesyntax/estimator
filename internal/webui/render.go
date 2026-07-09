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
	return renderString(func(w io.Writer) error {
		return RenderBuild(w, s.BuildView(), RenderOptions{})
	})
}

// ProposalHTML renders the current Proposal screen to a string.
func (s *Session) ProposalHTML() (string, error) {
	return renderString(func(w io.Writer) error {
		return RenderProposal(w, s.ProposalView(), RenderOptions{})
	})
}

// renderString collects a render into a string, returning an empty string when
// the render fails.
func renderString(render func(io.Writer) error) (string, error) {
	var b strings.Builder
	if err := render(&b); err != nil {
		return "", err
	}
	return b.String(), nil
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:55+05:30","module_hash":"71551113fb61c6f0b188dbb8733652f86444e4a05627c33fdf0bbac78fc8dc25","functions":[{"id":"func/RenderBuild","name":"RenderBuild","line":35,"end_line":37,"hash":"fd7422681eba3e4d920b0f4a3438e9390d1fe974002b703a8dbaac9302760785"},{"id":"func/RenderProposal","name":"RenderProposal","line":40,"end_line":42,"hash":"4309179063e03a9e5970ffdb7650d593562da5d8eb876f1aba1936b0f35e5f45"},{"id":"func/Session.BuildHTML","name":"Session.BuildHTML","line":46,"end_line":50,"hash":"9db5dc2ec2931280a3131af32072db92df02ae0b6d60959ef899f7473b222d20"},{"id":"func/Session.ProposalHTML","name":"Session.ProposalHTML","line":53,"end_line":57,"hash":"283a1cf087ea3ef8db54b78d50d3cc7505a1480008e606799f03ee2a9ac683ec"},{"id":"func/renderString","name":"renderString","line":61,"end_line":67,"hash":"bce843f20df1ee83cccba4b8c925a7a9dbf0dcc5b941a23216798c63c8cbaa27"}]}
// mutate4go-manifest-end
