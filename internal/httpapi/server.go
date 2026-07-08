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
	s.mux.HandleFunc("POST /wbs/{id}/estimates/generate", s.handleGenerateEstimates)
	s.mux.HandleFunc("PUT /wbs/{id}/tasks/{taskId}/estimate", s.handleOverrideEstimate)
	s.mux.HandleFunc("POST /wbs/{id}/estimates/approve", s.handleApproveEstimates)
	s.mux.HandleFunc("POST /qa/ai/next-wbs", s.handlePrime)
	s.mux.HandleFunc("POST /qa/ai/next-risks", s.handlePrimeRisks)
	s.mux.HandleFunc("POST /qa/ai/next-estimates", s.handlePrimeEstimates)
}

// --- JSON DTOs -------------------------------------------------------------

type taskDTO struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	RiskNotes   []riskNoteDTO `json:"riskNotes"`
	Estimate    *estimateDTO  `json:"estimate"`
	Metrics     *metricsDTO   `json:"metrics"`
}

// metricsDTO renders the derived PERT metrics as three JSON integers. It serves
// both a task's per-task metrics and the WBS-level project rollup.
type metricsDTO struct {
	Expected                  int `json:"expected"`
	StandardDeviation         int `json:"standardDeviation"`
	RelativeStandardDeviation int `json:"relativeStandardDeviation"`
}

type riskNoteDTO struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type estimateDTO struct {
	Optimistic  int    `json:"optimistic"`
	MostLikely  int    `json:"mostLikely"`
	Pessimistic int    `json:"pessimistic"`
	Reasoning   string `json:"reasoning"`
}

type wbsDTO struct {
	ID             string      `json:"id"`
	State          string      `json:"state"`
	EstimatesState string      `json:"estimatesState"`
	Tasks          []taskDTO   `json:"tasks"`
	ApprovedTasks  []taskDTO   `json:"approvedTasks"`
	ProjectMetrics *metricsDTO `json:"projectMetrics"`
}

func view(w *wbs.WBS) wbsDTO {
	dto := wbsDTO{
		ID:             w.ID(),
		State:          state(w),
		EstimatesState: estimatesState(w),
		Tasks:          toDTOs(w.Tasks()),
	}
	if approved := w.ApprovedTasks(); approved != nil {
		dto.ApprovedTasks = toDTOs(approved)
	}
	if pm, ok := w.ProjectMetrics(); ok {
		dto.ProjectMetrics = metricsFrom(pm)
	}
	return dto
}

func state(w *wbs.WBS) string { return approvalLabel(w.Approved()) }

func estimatesState(w *wbs.WBS) string { return approvalLabel(w.EstimatesApproved()) }

// approvalLabel renders an approval flag as the "approved"/"unapproved" string
// the WBS and estimate states both serialize as.
func approvalLabel(approved bool) string {
	if approved {
		return "approved"
	}
	return "unapproved"
}

func toDTOs(tasks []wbs.Task) []taskDTO {
	out := make([]taskDTO, len(tasks))
	for i, t := range tasks {
		out[i] = taskDTO{ID: t.ID, Description: t.Description, RiskNotes: toRiskNoteDTOs(t.RiskNotes), Estimate: toEstimateDTO(t.Estimate), Metrics: toMetricsDTO(t.Estimate)}
	}
	return out
}

// toEstimateDTO renders a task's estimate, or nil so a task with no estimate
// serializes as null.
func toEstimateDTO(e *wbs.Estimate) *estimateDTO {
	if e == nil {
		return nil
	}
	return &estimateDTO{Optimistic: e.Optimistic, MostLikely: e.MostLikely, Pessimistic: e.Pessimistic, Reasoning: e.Reasoning}
}

// toMetricsDTO renders a task's derived PERT metrics, or nil so a task with no
// estimate serializes its metrics as null.
func toMetricsDTO(e *wbs.Estimate) *metricsDTO {
	if e == nil {
		return nil
	}
	return metricsFrom(e.Metrics())
}

// metricsFrom renders domain metrics as their JSON DTO.
func metricsFrom(m wbs.Metrics) *metricsDTO {
	return &metricsDTO{Expected: m.Expected, StandardDeviation: m.StandardDeviation, RelativeStandardDeviation: m.RelativeStandardDeviation}
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

func (s *Server) handleGenerateEstimates(w http.ResponseWriter, r *http.Request) {
	s.mutateCurrentWBS(w, r, s.svc.GenerateEstimates)
}

func (s *Server) handleOverrideEstimate(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	number, err := s.resolveTaskNumber(id, r.PathValue("taskId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	estimate, err := decodeEstimate(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.OverrideEstimate(id, number, estimate); err != nil {
		writeDomainError(w, err)
		return
	}
	s.writeCurrentWBS(w, http.StatusOK, id)
}

func (s *Server) handleApproveEstimates(w http.ResponseWriter, r *http.Request) {
	s.mutateCurrentWBS(w, r, s.svc.ApproveEstimates)
}

func (s *Server) handlePrimeEstimates(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// The priming affordance only exists in mock mode; in production it is
	// absent so QA cannot run.
	if !s.mock {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Estimates []struct {
			TaskNumber  int    `json:"taskNumber"`
			Optimistic  int    `json:"optimistic"`
			MostLikely  int    `json:"mostLikely"`
			Pessimistic int    `json:"pessimistic"`
			Reasoning   string `json:"reasoning"`
		} `json:"estimates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid priming request")
		return
	}
	assignments := make([]wbs.EstimateAssignment, len(body.Estimates))
	for i, e := range body.Estimates {
		assignments[i] = wbs.EstimateAssignment{
			TaskNumber:  e.TaskNumber,
			Optimistic:  e.Optimistic,
			MostLikely:  e.MostLikely,
			Pessimistic: e.Pessimistic,
			Reasoning:   e.Reasoning,
		}
	}
	s.svc.PrimeEstimates(assignments)
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

// decodeEstimate decodes an override body ({optimistic, mostLikely, pessimistic,
// reasoning}) into a domain estimate. Value and ordering validation is left to
// the domain.
func decodeEstimate(r *http.Request) (wbs.Estimate, error) {
	var body struct {
		Optimistic  int    `json:"optimistic"`
		MostLikely  int    `json:"mostLikely"`
		Pessimistic int    `json:"pessimistic"`
		Reasoning   string `json:"reasoning"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return wbs.Estimate{}, errors.New("invalid request body")
	}
	return wbs.Estimate{
		Optimistic:  body.Optimistic,
		MostLikely:  body.MostLikely,
		Pessimistic: body.Pessimistic,
		Reasoning:   body.Reasoning,
	}, nil
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
	{wbs.ErrWBSNotApprovedForEstimation, http.StatusConflict, "WBS must be approved before estimation"},
	{wbs.ErrNoEstimate, http.StatusConflict, "task has no estimate to override"},
	{wbs.ErrEstimatesNotGenerated, http.StatusConflict, "estimates have not been generated"},
	{wbs.ErrOffFibonacciScale, http.StatusUnprocessableEntity, "estimate value is off the Fibonacci scale"},
	{wbs.ErrEstimatesNotIncreasing, http.StatusUnprocessableEntity, "estimate values must be strictly increasing"},
	{wbs.ErrEmptyReasoning, http.StatusUnprocessableEntity, "reasoning is empty"},
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
// {"version":1,"tested_at":"2026-07-08T15:50:51+05:30","module_hash":"2904c2a354fde4d81b2a37274d56655767c6020f7c8cda2d678d41da5568be9e","functions":[{"id":"func/NewServer","name":"NewServer","line":28,"end_line":30,"hash":"c9d9590eef19ce5738aeb1e336139b634c537b34cfdef40fc45e163e908b90c4"},{"id":"func/NewServerWithService","name":"NewServerWithService","line":35,"end_line":39,"hash":"22ad8c47ee69a9a36f7f01a3a0a2e6410a9b38bd8ead95828af92ba59787d9a9"},{"id":"func/Server.ServeHTTP","name":"Server.ServeHTTP","line":41,"end_line":43,"hash":"3718ac44b18d6acdf59cd133cfc12dd07edc7f8df55e2740093e6b14b76f9fdd"},{"id":"func/Server.routes","name":"Server.routes","line":45,"end_line":62,"hash":"eeaf214fa8cc49c73a2adbce047802b0524382a07cc353b0052b9cc187fbb318"},{"id":"func/view","name":"view","line":93,"end_line":104,"hash":"539becbe30a1e8054c83f288276982f71fec1a9245862c9705854a4fbc1ee4b8"},{"id":"func/state","name":"state","line":106,"end_line":106,"hash":"63aca2a952053e8d1191377107959c94294306fb8cb958ed2f6abc3cd521a1ef"},{"id":"func/estimatesState","name":"estimatesState","line":108,"end_line":108,"hash":"fe8e0205ebffdca15f3ec8cbeae34dfcf5f7a3ac136cdc386e746f4db08a1e9b"},{"id":"func/approvalLabel","name":"approvalLabel","line":112,"end_line":117,"hash":"1cba85c8f754b5d9d81e2c0196d329abff6b53138761c0b7c15bc1b6c56bf9f1"},{"id":"func/toDTOs","name":"toDTOs","line":119,"end_line":125,"hash":"c731f47a3671af2c690f7ced51d57c03fcf7307f63215d495469cd4cf18400cd"},{"id":"func/toEstimateDTO","name":"toEstimateDTO","line":129,"end_line":134,"hash":"308d1e6f0a8ff7d21c2f9195a70cf0b5728cdceef2fb30e71cd6201ca9f1ced8"},{"id":"func/toRiskNoteDTOs","name":"toRiskNoteDTOs","line":138,"end_line":144,"hash":"8fcbe779fa4c28a0bf6f218dc8b1fd8c021e35ca3faecaf5a90755f9e3d2ce6d"},{"id":"func/Server.handleGenerate","name":"Server.handleGenerate","line":148,"end_line":163,"hash":"3f4239a68f299a13bbe451ebbc862413b12f12c570fb7fa21a3a49ba9aa81573"},{"id":"func/Server.handleGet","name":"Server.handleGet","line":165,"end_line":175,"hash":"d10f984d267368ed67e7ddaa994a390fd7e76c8fe14beb8343343e392f6d85e7"},{"id":"func/Server.handleAddTask","name":"Server.handleAddTask","line":177,"end_line":192,"hash":"5e9e0419a3db187c0386d148d9a4a43aff52810680195ff1122d51ab8a9b8f46"},{"id":"func/Server.handleEditTask","name":"Server.handleEditTask","line":194,"end_line":196,"hash":"403a442e325f75492837de2fc7202611e16da5b1d2f8ae906fb94f6fa5385ece"},{"id":"func/Server.handleDeleteTask","name":"Server.handleDeleteTask","line":198,"end_line":213,"hash":"11fe7d9eb06955a2bf1b5529effc11b1e953b4172fd6bddd8c77029fcdca5219"},{"id":"func/Server.handleApprove","name":"Server.handleApprove","line":215,"end_line":217,"hash":"e862e5b17c7ed9448419775285466e8885c9b955e3c2d706e782434fe3e824dc"},{"id":"func/Server.handleFlagRisks","name":"Server.handleFlagRisks","line":219,"end_line":221,"hash":"f14787c5f51a3524d62c9a464aeb3b188fe2c338cffdf927d2653a36551d2b63"},{"id":"func/Server.handleAddRiskNote","name":"Server.handleAddRiskNote","line":223,"end_line":225,"hash":"d928198a0785acb3cc49ff77e424cf3f703d3811e1b60e76869329592b7296ba"},{"id":"func/Server.handleEditRiskNote","name":"Server.handleEditRiskNote","line":227,"end_line":247,"hash":"4bc8cd72da9863c277654fae612866669cc2b7715c068a93e4c5c5901fc31347"},{"id":"func/Server.handleDeleteRiskNote","name":"Server.handleDeleteRiskNote","line":249,"end_line":264,"hash":"1c5144274f1be80f0905b518ac0b4de0917207028c0315cff6f677cae91d0de8"},{"id":"func/Server.handlePrimeRisks","name":"Server.handlePrimeRisks","line":266,"end_line":292,"hash":"91e7e1a683dc1c6f14856c340f1e8d3f16c428fa5002798cadc253fb9fbdc061"},{"id":"func/Server.handlePrime","name":"Server.handlePrime","line":294,"end_line":313,"hash":"fec2406e2244bfd4b49e270c5cf8dc427925975bf6bebb2515da80d2548d64a0"},{"id":"func/Server.handleGenerateEstimates","name":"Server.handleGenerateEstimates","line":315,"end_line":317,"hash":"ae10884f49253bf677a68f3736a829adbd0a09328ba864c5f6743926cdbb02fd"},{"id":"func/Server.handleOverrideEstimate","name":"Server.handleOverrideEstimate","line":319,"end_line":339,"hash":"643675e694f9c2c89ff14af5cc4fdaa5087a1a0d44c17b4759575ea9bb861c2d"},{"id":"func/Server.handleApproveEstimates","name":"Server.handleApproveEstimates","line":341,"end_line":343,"hash":"9b6fefeea6f63cac55e7887d9a2436beca94aa4db6054563055fe86042854430"},{"id":"func/Server.handlePrimeEstimates","name":"Server.handlePrimeEstimates","line":345,"end_line":380,"hash":"2ade7e9d5c829a750604a5fef1df0851674b6f6480ab683c0a3f6bd2d38221f5"},{"id":"func/Server.resolveTaskNumber","name":"Server.resolveTaskNumber","line":385,"end_line":399,"hash":"d3ba7a0999a919f9176a60224f163069bc0f7d3f668d73213950d29e50cb87fe"},{"id":"func/Server.resolveRiskNote","name":"Server.resolveRiskNote","line":404,"end_line":421,"hash":"58679715dadf585c9a5bd87fad28f6d8324d4ed7fce208067c243cfc77191025"},{"id":"func/requirementDocument","name":"requirementDocument","line":427,"end_line":447,"hash":"abf41f413e02a4fef066c24406093f0a6e7855eedfe4c11bc694610a9a77d65b"},{"id":"func/isMultipart","name":"isMultipart","line":449,"end_line":451,"hash":"f60985c83ed8abcbd0f561a6104a62cf38b97c99ef566f601bade7de21560069"},{"id":"func/decodeDescription","name":"decodeDescription","line":453,"end_line":461,"hash":"77c6a5312e414f4f1c1b4fe868057a2e15aed588eae36621f6ef88e04b94b86b"},{"id":"func/decodeEstimate","name":"decodeEstimate","line":466,"end_line":482,"hash":"5a3a273fe2d44c507c08ecf96871701944840160639ea0d97b88f1b34aa4d5ce"},{"id":"func/Server.mutateCurrentWBS","name":"Server.mutateCurrentWBS","line":486,"end_line":496,"hash":"19b4b4781a9f195c27cfb1b41719cca5d6d58178a7235bb2f673ca612c97fa77"},{"id":"func/Server.mutateTaskWithBody","name":"Server.mutateTaskWithBody","line":501,"end_line":521,"hash":"ff094ba6c0714d20dca627ecbaa6e926dc568837312912ff69a1cb2bd43d1ce1"},{"id":"func/Server.writeCurrentWBS","name":"Server.writeCurrentWBS","line":526,"end_line":529,"hash":"19caaaf93cf30da7b7f214448dae318ba2a9d4b3880d1974089fb082be2cb614"},{"id":"func/writeJSON","name":"writeJSON","line":531,"end_line":535,"hash":"c09dc9c83864e0c06d04fd189caa3114f22fe8eade4f59a5cb47d7d8b9e13dd6"},{"id":"func/writeError","name":"writeError","line":537,"end_line":539,"hash":"c383d1c080b6a4af74efcf987d4acfb4cc2758bafbf34a92cb7e5eb7572691fb"},{"id":"func/writeDomainError","name":"writeDomainError","line":566,"end_line":574,"hash":"d1ee54ed87fd32cf759710a0abca829c58e57ef8eda1495a616c1078e9de438a"}]}
// mutate4go-manifest-end
