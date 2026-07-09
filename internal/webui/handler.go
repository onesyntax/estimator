package webui

import (
	"io"
	"net/http"
	"strconv"
	"sync"

	"estimation/internal/wbs"
)

// Handler serves the two-stage estimation UI over HTTP. It holds one active
// session — matching the single in-memory estimate the backend keeps — and
// serializes requests behind a mutex. Actions are ordinary form posts followed
// by a redirect (Post/Redirect/Get); the inline error state lives on the session
// and surfaces on the next render.
type Handler struct {
	mu      sync.Mutex
	session *Session
	qaMode  bool
	mux     *http.ServeMux
}

// NewHandler builds a UI handler backed by a fresh deterministic session. When
// qaMode is true the QA priming surface (/ui/qa/*) is enabled; otherwise it
// responds 404 so it is absent in production.
func NewHandler(qaMode bool) *Handler {
	return NewHandlerWithService(wbs.NewService(), qaMode)
}

// NewHandlerWithService builds a UI handler backed by the given service, letting
// a caller inject a real AI provider.
func NewHandlerWithService(svc *wbs.Service, qaMode bool) *Handler {
	h := &Handler{session: NewSessionWithService(svc), qaMode: qaMode, mux: http.NewServeMux()}
	h.routes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

func (h *Handler) routes() {
	h.mux.HandleFunc("GET /", h.handleBuild)
	h.mux.HandleFunc("GET /proposal", h.handleProposal)
	h.mux.HandleFunc("POST /ui/requirement", h.handleRequirement)
	h.mux.HandleFunc("POST /ui/tasks/add", h.action(func(s *Session, r *http.Request) {
		s.AddTask(r.FormValue("description"))
	}))
	h.mux.HandleFunc("POST /ui/tasks/edit", h.action(func(s *Session, r *http.Request) {
		s.EditTask(formInt(r, "number"), r.FormValue("description"))
	}))
	h.mux.HandleFunc("POST /ui/tasks/delete", h.action(func(s *Session, r *http.Request) {
		s.DeleteTask(formInt(r, "number"))
	}))
	h.mux.HandleFunc("POST /ui/wbs/approve", h.action(func(s *Session, _ *http.Request) {
		s.ApproveWBS()
	}))
	h.mux.HandleFunc("POST /ui/risks/flag", h.action(func(s *Session, _ *http.Request) {
		s.FlagRisks()
	}))
	h.mux.HandleFunc("POST /ui/risks/add", h.action(func(s *Session, r *http.Request) {
		s.AddRiskNote(formInt(r, "number"), r.FormValue("description"))
	}))
	h.mux.HandleFunc("POST /ui/estimates/generate", h.action(func(s *Session, _ *http.Request) {
		s.GenerateEstimates()
	}))
	h.mux.HandleFunc("POST /ui/estimates/override", h.action(func(s *Session, r *http.Request) {
		s.OverrideEstimate(formInt(r, "number"), EstimateInput{
			Optimistic:  formInt(r, "optimistic"),
			MostLikely:  formInt(r, "mostLikely"),
			Pessimistic: formInt(r, "pessimistic"),
			Reasoning:   r.FormValue("reasoning"),
		})
	}))
	h.mux.HandleFunc("POST /ui/estimates/approve", h.action(func(s *Session, _ *http.Request) {
		s.ApproveEstimates()
	}))
	h.mux.HandleFunc("POST /ui/proposal", h.handleProposalRequest)
	h.mux.HandleFunc("POST /ui/qa/wbs", h.qaAction(func(s *Session, r *http.Request) {
		s.PrimeWBS(ParseTaskList(r.FormValue("tasks")))
	}))
	h.mux.HandleFunc("POST /ui/qa/risks", h.qaAction(func(s *Session, r *http.Request) {
		s.PrimeRisks(ParseRiskSeeds(r.FormValue("risks")))
	}))
	h.mux.HandleFunc("POST /ui/qa/estimates", h.qaAction(func(s *Session, r *http.Request) {
		s.PrimeEstimates(ParseEstimateSeeds(r.FormValue("estimates")))
	}))
}

func (h *Handler) handleBuild(w http.ResponseWriter, _ *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.renderBuild(w)
}

func (h *Handler) renderBuild(w io.Writer) {
	_ = RenderBuild(w, h.session.BuildView(), RenderOptions{QAMode: h.qaMode})
}

func (h *Handler) handleProposal(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// The Proposal stage is a client deliverable: it is unreachable until the
	// estimates are approved, so the gate is enforced structurally by refusing
	// the screen before then.
	if !h.session.BuildView().ProposalReachable {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	_ = RenderProposal(w, h.session.ProposalView(), RenderOptions{QAMode: h.qaMode})
}

func (h *Handler) handleRequirement(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.session.GenerateWBS(requirementDocument(r))
	redirectBuild(w, r)
}

func (h *Handler) handleProposalRequest(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.session.RequestProposal(TeamInput{
		Velocity: formInt(r, "velocity"),
		Capacity: formInt(r, "capacity"),
		Rate:     formInt(r, "rate"),
	})
	http.Redirect(w, r, "/proposal", http.StatusSeeOther)
}

// action wraps a build-screen mutation as a POST handler that applies it under
// the lock and redirects back to the Build screen.
func (h *Handler) action(apply func(*Session, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		defer h.mu.Unlock()
		apply(h.session, r)
		redirectBuild(w, r)
	}
}

// qaAction is like action but responds 404 outside mock mode, so the QA priming
// surface is absent in production.
func (h *Handler) qaAction(apply func(*Session, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		defer h.mu.Unlock()
		if !h.qaMode {
			http.NotFound(w, r)
			return
		}
		apply(h.session, r)
		redirectBuild(w, r)
	}
}

func redirectBuild(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// requirementDocument builds a requirement document from an uploaded PDF part
// ("document") when present, otherwise from the "requirement" text field.
func requirementDocument(r *http.Request) wbs.RequirementDocument {
	if file, _, err := r.FormFile("document"); err == nil {
		defer file.Close()
		if data, err := io.ReadAll(file); err == nil && len(data) > 0 {
			return wbs.NewPDFDocument(data)
		}
	}
	return wbs.NewTextDocument(r.FormValue("requirement"))
}

func formInt(r *http.Request, name string) int {
	n, _ := strconv.Atoi(r.FormValue(name))
	return n
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:48+05:30","module_hash":"8399b740e2e641d39b45c54d7128be3bc0112c867c6d9e5ad9b65d1506c8e5c7","functions":[{"id":"func/NewHandler","name":"NewHandler","line":27,"end_line":29,"hash":"a559fde7d53cd69bfcb72107d491e5648317fb547f8e615690b210780d13518a"},{"id":"func/NewHandlerWithService","name":"NewHandlerWithService","line":33,"end_line":37,"hash":"1fb6ea6c188920c0e6cc9ffddecc338e79d976260d98f6b2d994ef0fdcc88f57"},{"id":"func/Handler.ServeHTTP","name":"Handler.ServeHTTP","line":39,"end_line":39,"hash":"2d844bf7f431e4eb278c6097a67d2df86dfe5b566689febcfa10cc333a6cbabb"},{"id":"func/Handler.routes","name":"Handler.routes","line":41,"end_line":87,"hash":"43b990ee9815ad6122a2bfe8d82090d2d025dfede4e02eb38cb43c5389ba9b1b"},{"id":"func/Handler.handleBuild","name":"Handler.handleBuild","line":89,"end_line":93,"hash":"e64924ad982c0bc0e435bf88eacd0a31acdf962434e963617461b508c922d763"},{"id":"func/Handler.renderBuild","name":"Handler.renderBuild","line":95,"end_line":97,"hash":"48b1b5df5b2709a5e71e21293876c0ab909b82058359d85b9318e7db92bb10e4"},{"id":"func/Handler.handleProposal","name":"Handler.handleProposal","line":99,"end_line":110,"hash":"2dc5fecde7b182ec99507e866d0fdb09a917eb1901264762f9477263941b5e60"},{"id":"func/Handler.handleRequirement","name":"Handler.handleRequirement","line":112,"end_line":117,"hash":"e480e9aea79a9f425642491cdf7c2ebac93d4f69699c72f2ffe8050795e7520d"},{"id":"func/Handler.handleProposalRequest","name":"Handler.handleProposalRequest","line":119,"end_line":128,"hash":"ed58d58bdf80715e616a90c5adcc532998586c74f1b8e3fe255949823301148d"},{"id":"func/Handler.action","name":"Handler.action","line":132,"end_line":139,"hash":"e41cd72fa89ae69cc0e4b69ebd931efe2a5fbd542ffeccaa4f30d4ab4607fa20"},{"id":"func/Handler.qaAction","name":"Handler.qaAction","line":143,"end_line":154,"hash":"d6e616f7f0ccd8b1a091a4ec21fc510cfcce5aced9c010695d3620c480cf2235"},{"id":"func/redirectBuild","name":"redirectBuild","line":156,"end_line":158,"hash":"619d05dab7dad82b44feede3a4b2569a96a1062bc0c206c64411bfe9a07a0e34"},{"id":"func/requirementDocument","name":"requirementDocument","line":162,"end_line":170,"hash":"96c23d09a9cfdb3807f0c22092610cfaf0562e24ac7812aec30b2492340eab78"},{"id":"func/formInt","name":"formInt","line":172,"end_line":175,"hash":"a3d1d5254a324cf8debf17acabb15cb2cf2a5e6fd0eeec459c232104454868f2"}]}
// mutate4go-manifest-end
