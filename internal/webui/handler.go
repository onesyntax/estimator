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
