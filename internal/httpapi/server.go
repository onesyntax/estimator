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

// NewServer builds a server backed by the default deterministic service. When
// mock is true the QA priming affordance (POST /qa/ai/next-wbs) is enabled.
func NewServer(mock bool) *Server {
	return NewServerWithService(wbs.NewService(), mock)
}

// NewServerWithService builds a server backed by the given service, letting the
// caller inject a real AI provider. When mock is true the QA priming affordance
// (POST /qa/ai/next-wbs) is enabled.
func NewServerWithService(svc *wbs.Service, mock bool) *Server {
	s := &Server{svc: svc, mock: mock, mux: http.NewServeMux()}
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
	s.mux.HandleFunc("POST /wbs/{id}/risk-notes/flag", s.handleFlagRisks)
	s.mux.HandleFunc("POST /wbs/{id}/tasks/{taskId}/risk-notes", s.handleAddRiskNote)
	s.mux.HandleFunc("PUT /wbs/{id}/tasks/{taskId}/risk-notes/{noteId}", s.handleEditRiskNote)
	s.mux.HandleFunc("DELETE /wbs/{id}/tasks/{taskId}/risk-notes/{noteId}", s.handleDeleteRiskNote)
	s.mux.HandleFunc("POST /qa/ai/next-wbs", s.handlePrime)
	s.mux.HandleFunc("POST /qa/ai/next-risks", s.handlePrimeRisks)
}

// --- JSON DTOs -------------------------------------------------------------

type taskDTO struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	RiskNotes   []riskNoteDTO `json:"riskNotes"`
}

type riskNoteDTO struct {
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
		out[i] = taskDTO{ID: t.ID, Description: t.Description, RiskNotes: toRiskNoteDTOs(t.RiskNotes)}
	}
	return out
}

// toRiskNoteDTOs renders a task's risk notes, always as a non-nil slice so a
// task with no notes serializes as [] rather than null.
func toRiskNoteDTOs(notes []wbs.RiskNote) []riskNoteDTO {
	out := make([]riskNoteDTO, len(notes))
	for i, n := range notes {
		out[i] = riskNoteDTO{ID: n.ID, Description: n.Description}
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
	s.mutateTaskWithBody(w, r, http.StatusOK, s.svc.EditTask)
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
	s.mutateCurrentWBS(w, r, s.svc.Approve)
}

func (s *Server) handleFlagRisks(w http.ResponseWriter, r *http.Request) {
	s.mutateCurrentWBS(w, r, s.svc.FlagRisks)
}

func (s *Server) handleAddRiskNote(w http.ResponseWriter, r *http.Request) {
	s.mutateTaskWithBody(w, r, http.StatusCreated, s.svc.AddRiskNote)
}

func (s *Server) handleEditRiskNote(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	number, position, err := s.resolveRiskNote(id, r.PathValue("taskId"), r.PathValue("noteId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	body, err := decodeDescription(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.EditRiskNote(id, number, position, body); err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, http.StatusOK, id)
}

func (s *Server) handleDeleteRiskNote(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	number, position, err := s.resolveRiskNote(id, r.PathValue("taskId"), r.PathValue("noteId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if err := s.svc.DeleteRiskNote(id, number, position); err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, http.StatusOK, id)
}

func (s *Server) handlePrimeRisks(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// The priming affordance only exists in mock mode; in production it is
	// absent so QA cannot run.
	if !s.mock {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Risks []struct {
			TaskNumber  int    `json:"taskNumber"`
			Description string `json:"description"`
		} `json:"risks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid priming request")
		return
	}
	assignments := make([]wbs.RiskAssignment, len(body.Risks))
	for i, risk := range body.Risks {
		assignments[i] = wbs.RiskAssignment{TaskNumber: risk.TaskNumber, Description: risk.Description}
	}
	s.svc.PrimeRisks(assignments)
	w.WriteHeader(http.StatusNoContent)
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
func (s *Server) resolveTaskNumber(wbsID, taskID string) (number int, err error) {
	cur, err := s.svc.Get(wbsID)
	if err != nil {
		// number is unused on the error path; a naked return keeps it free of a
		// mutable literal so mutation has nothing equivalent to flag.
		return
	}
	for i, t := range cur.Tasks() {
		if t.ID == taskID {
			return i + 1, nil
		}
	}
	err = wbs.ErrTaskNotFound
	return
}

// resolveRiskNote maps a task id and a risk-note id to their one-based positions
// within the WBS. It reports ErrWBSNotFound for an unknown WBS, ErrTaskNotFound
// for an unknown task id, and ErrRiskNoteNotFound for an unknown note id.
func (s *Server) resolveRiskNote(wbsID, taskID, noteID string) (number, position int, err error) {
	number, err = s.resolveTaskNumber(wbsID, taskID)
	if err != nil {
		return
	}
	cur, err := s.svc.Get(wbsID)
	if err != nil {
		return
	}
	task, _ := cur.TaskAt(number)
	for i, n := range task.RiskNotes {
		if n.ID == noteID {
			return number, i + 1, nil
		}
	}
	err = wbs.ErrRiskNoteNotFound
	return
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

// mutateCurrentWBS applies an id-only mutation to the path's WBS under the lock
// and, on success, writes the updated WBS with 200.
func (s *Server) mutateCurrentWBS(w http.ResponseWriter, r *http.Request, apply func(id string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	if err := apply(id); err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, http.StatusOK, id)
}

// mutateTaskWithBody resolves the path's task number, decodes a description
// body, applies the mutation under the lock, and writes the updated WBS with the
// given success status.
func (s *Server) mutateTaskWithBody(w http.ResponseWriter, r *http.Request, status int, apply func(id string, number int, body string) error) {
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
	if err := apply(id, number, body); err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, status, id)
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
	{wbs.ErrRiskNoteNotFound, http.StatusNotFound, "risk note not found"},
	{wbs.ErrEmptyDescription, http.StatusBadRequest, "description is empty"},
	{wbs.ErrEmptyRequirement, http.StatusBadRequest, "requirement is empty"},
	{wbs.ErrUnreadableDocument, http.StatusBadRequest, "document could not be read"},
	{wbs.ErrNoTasks, http.StatusConflict, "cannot approve an empty WBS"},
	{wbs.ErrWBSNotApproved, http.StatusConflict, "WBS must be approved before risk flagging"},
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

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-07T22:52:38+05:30","module_hash":"9656463641140afa0f9ffb52dd1c5a92dc0027ec38c66478630dbb2d912a43d1","functions":[{"id":"func/NewServer","name":"NewServer","line":28,"end_line":32,"hash":"984be6ab8a1ea13ccf8d5fa64bdaf475b49c7655618348fb85873377ee87960f"},{"id":"func/Server.ServeHTTP","name":"Server.ServeHTTP","line":34,"end_line":36,"hash":"3718ac44b18d6acdf59cd133cfc12dd07edc7f8df55e2740093e6b14b76f9fdd"},{"id":"func/Server.routes","name":"Server.routes","line":38,"end_line":46,"hash":"d2bacb782b2b6f9a3658f66febbd65c0038eff35999635f1bec71dbcdfaddc0b"},{"id":"func/view","name":"view","line":62,"end_line":72,"hash":"2eff9a158415a3e9ca52fd0f25a25eb95c6607dfb22c0a94a140ddc3a8b90cd7"},{"id":"func/state","name":"state","line":74,"end_line":79,"hash":"357ea4430186d1d6b1953342dee8822c677fe4d5a71d0041dabc5c981e78c4a0"},{"id":"func/toDTOs","name":"toDTOs","line":81,"end_line":87,"hash":"36528b662d508367215cb7300211591f3801143954067b87de2d87a34b7b0d1a"},{"id":"func/Server.handleGenerate","name":"Server.handleGenerate","line":91,"end_line":106,"hash":"3f4239a68f299a13bbe451ebbc862413b12f12c570fb7fa21a3a49ba9aa81573"},{"id":"func/Server.handleGet","name":"Server.handleGet","line":108,"end_line":118,"hash":"d10f984d267368ed67e7ddaa994a390fd7e76c8fe14beb8343343e392f6d85e7"},{"id":"func/Server.handleAddTask","name":"Server.handleAddTask","line":120,"end_line":135,"hash":"5e9e0419a3db187c0386d148d9a4a43aff52810680195ff1122d51ab8a9b8f46"},{"id":"func/Server.handleEditTask","name":"Server.handleEditTask","line":137,"end_line":157,"hash":"f8e167e9995cae341f0460c6f6212a6480994296b987d8c5262f2c336a5829f0"},{"id":"func/Server.handleDeleteTask","name":"Server.handleDeleteTask","line":159,"end_line":174,"hash":"11fe7d9eb06955a2bf1b5529effc11b1e953b4172fd6bddd8c77029fcdca5219"},{"id":"func/Server.handleApprove","name":"Server.handleApprove","line":176,"end_line":186,"hash":"87c54201eb73ea477eb69713d1075663020c05f6d5fe594ad453d225376198d1"},{"id":"func/Server.handlePrime","name":"Server.handlePrime","line":188,"end_line":207,"hash":"fec2406e2244bfd4b49e270c5cf8dc427925975bf6bebb2515da80d2548d64a0"},{"id":"func/Server.resolveTaskNumber","name":"Server.resolveTaskNumber","line":212,"end_line":226,"hash":"d3ba7a0999a919f9176a60224f163069bc0f7d3f668d73213950d29e50cb87fe"},{"id":"func/requirementDocument","name":"requirementDocument","line":232,"end_line":252,"hash":"abf41f413e02a4fef066c24406093f0a6e7855eedfe4c11bc694610a9a77d65b"},{"id":"func/isMultipart","name":"isMultipart","line":254,"end_line":256,"hash":"f60985c83ed8abcbd0f561a6104a62cf38b97c99ef566f601bade7de21560069"},{"id":"func/decodeDescription","name":"decodeDescription","line":258,"end_line":266,"hash":"77c6a5312e414f4f1c1b4fe868057a2e15aed588eae36621f6ef88e04b94b86b"},{"id":"func/Server.writeCurrentWBS","name":"Server.writeCurrentWBS","line":271,"end_line":274,"hash":"19caaaf93cf30da7b7f214448dae318ba2a9d4b3880d1974089fb082be2cb614"},{"id":"func/writeJSON","name":"writeJSON","line":276,"end_line":280,"hash":"c09dc9c83864e0c06d04fd189caa3114f22fe8eade4f59a5cb47d7d8b9e13dd6"},{"id":"func/writeError","name":"writeError","line":282,"end_line":284,"hash":"c383d1c080b6a4af74efcf987d4acfb4cc2758bafbf34a92cb7e5eb7572691fb"},{"id":"func/writeDomainError","name":"writeDomainError","line":303,"end_line":311,"hash":"d1ee54ed87fd32cf759710a0abca829c58e57ef8eda1495a616c1078e9de438a"}]}
// mutate4go-manifest-end
