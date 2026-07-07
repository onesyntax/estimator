// Package httpapi exposes the estimation service over HTTP. It is a thin
// adapter: every request is translated into a call on the in-process WBS
// service, and domain errors are mapped to the documented status codes and
// messages. A single mutex serializes access so concurrent requests are safe.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"estimation/internal/wbs"
)

// Server is the estimation HTTP server.
type Server struct {
	mu   sync.Mutex
	svc  *wbs.Service
	mock bool
	mux  *http.ServeMux
}

// NewServer builds a server. When mock is true the deterministic QA priming
// affordance (POST /qa/ai/next-wbs) is enabled.
func NewServer(mock bool) *Server {
	s := &Server{svc: wbs.NewService(), mock: mock, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /wbs", s.handleGenerate)
	s.mux.HandleFunc("GET /wbs/{id}", s.handleGet)
	s.mux.HandleFunc("POST /wbs/{id}/tasks", s.handleAddTask)
	s.mux.HandleFunc("PUT /wbs/{id}/tasks/{taskId}", s.handleEditTask)
	s.mux.HandleFunc("DELETE /wbs/{id}/tasks/{taskId}", s.handleDeleteTask)
	s.mux.HandleFunc("POST /wbs/{id}/approve", s.handleApprove)
	s.mux.HandleFunc("POST /qa/ai/next-wbs", s.handlePrime)
}

// --- JSON DTOs -------------------------------------------------------------

type taskDTO struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type wbsDTO struct {
	ID            string    `json:"id"`
	State         string    `json:"state"`
	Tasks         []taskDTO `json:"tasks"`
	ApprovedTasks []taskDTO `json:"approvedTasks"`
}

func view(w *wbs.WBS) wbsDTO {
	dto := wbsDTO{
		ID:    w.ID(),
		State: state(w),
		Tasks: toDTOs(w.Tasks()),
	}
	if approved := w.ApprovedTasks(); approved != nil {
		dto.ApprovedTasks = toDTOs(approved)
	}
	return dto
}

func state(w *wbs.WBS) string {
	if w.Approved() {
		return "approved"
	}
	return "unapproved"
}

func toDTOs(tasks []wbs.Task) []taskDTO {
	out := make([]taskDTO, len(tasks))
	for i, t := range tasks {
		out[i] = taskDTO{ID: t.ID, Description: t.Description}
	}
	return out
}

// --- Handlers --------------------------------------------------------------

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := requirementDocument(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := s.svc.Generate(doc)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, http.StatusCreated, id)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, err := s.svc.Get(r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view(cur))
}

func (s *Server) handleAddTask(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	body, err := decodeDescription(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.AddTask(id, body); err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, http.StatusCreated, id)
}

func (s *Server) handleEditTask(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	number, err := s.resolveTaskNumber(id, r.PathValue("taskId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	body, err := decodeDescription(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.EditTask(id, number, body); err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, http.StatusOK, id)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	number, err := s.resolveTaskNumber(id, r.PathValue("taskId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if err := s.svc.DeleteTask(id, number); err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, http.StatusOK, id)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	if err := s.svc.Approve(id); err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, http.StatusOK, id)
}

func (s *Server) handlePrime(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// The priming affordance only exists in mock mode; in production it is
	// absent so QA cannot run.
	if !s.mock {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Tasks []string `json:"tasks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid priming request")
		return
	}
	s.svc.Prime(body.Tasks)
	w.WriteHeader(http.StatusNoContent)
}

// resolveTaskNumber maps a stable task id to its one-based position within the
// WBS. It reports ErrWBSNotFound for an unknown WBS and ErrTaskNotFound for an
// unknown task id.
func (s *Server) resolveTaskNumber(wbsID, taskID string) (int, error) {
	cur, err := s.svc.Get(wbsID)
	if err != nil {
		return 0, err
	}
	for i, t := range cur.Tasks() {
		if t.ID == taskID {
			return i + 1, nil
		}
	}
	return 0, wbs.ErrTaskNotFound
}

// --- Request/response helpers ---------------------------------------------

// requirementDocument builds a requirement document from either a JSON body
// ({"requirement": "..."}) or a multipart form with a "document" PDF part.
func requirementDocument(r *http.Request) (wbs.RequirementDocument, error) {
	if isMultipart(r) {
		file, _, err := r.FormFile("document")
		if err != nil {
			return nil, wbs.ErrUnreadableDocument
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return nil, wbs.ErrUnreadableDocument
		}
		return wbs.NewPDFDocument(data), nil
	}
	var body struct {
		Requirement string `json:"requirement"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, wbs.ErrEmptyRequirement
	}
	return wbs.NewTextDocument(body.Requirement), nil
}

func isMultipart(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data")
}

func decodeDescription(r *http.Request) (string, error) {
	var body struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", errors.New("invalid request body")
	}
	return body.Description, nil
}

// writeCurrentWBS re-reads the WBS after a successful mutation and writes it as
// the response. The lookup cannot fail here: the caller just operated on this
// id under the held lock.
func (s *Server) writeCurrentWBS(w http.ResponseWriter, status int, id string) {
	cur, _ := s.svc.Get(id)
	writeJSON(w, status, view(cur))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// domainErrorResponses maps each domain error to its documented HTTP status and
// client-facing message. The errors are distinct, so order does not matter.
var domainErrorResponses = []struct {
	err     error
	status  int
	message string
}{
	{wbs.ErrWBSNotFound, http.StatusNotFound, "WBS not found"},
	{wbs.ErrTaskNotFound, http.StatusNotFound, "task not found"},
	{wbs.ErrEmptyDescription, http.StatusBadRequest, "description is empty"},
	{wbs.ErrEmptyRequirement, http.StatusBadRequest, "requirement is empty"},
	{wbs.ErrUnreadableDocument, http.StatusBadRequest, "document could not be read"},
	{wbs.ErrNoTasks, http.StatusConflict, "cannot approve an empty WBS"},
}

// writeDomainError maps a domain error to its documented status code and
// client-facing message, falling back to 500 for an unrecognized error.
func writeDomainError(w http.ResponseWriter, err error) {
	for _, r := range domainErrorResponses {
		if errors.Is(err, r.err) {
			writeError(w, r.status, r.message)
			return
		}
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}
