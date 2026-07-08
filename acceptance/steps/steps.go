// Package steps contains the project-specific acceptance step handlers that
// connect Gherkin step text to the estimation service. Repeated step shapes
// that vary only by example values share a single regex-capture handler. The
// handlers are split by feature across steps_wbs.go, steps_risk.go, and
// steps_estimate.go; this file holds the registry wiring and the helpers those
// files share.
package steps

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"estimation/acceptance/runtime"
	"estimation/internal/wbs"
)

// world is the per-scenario state shared by background and scenario steps.
type world struct {
	svc     *wbs.Service
	wbsID   string
	lastErr error
}

func newWorld() any { return &world{svc: wbs.NewService()} }

// NewRegistry builds a runtime registry with a fresh world per execution and
// every estimation step handler registered.
func NewRegistry() *runtime.Registry {
	reg := runtime.NewRegistry(newWorld)
	register(reg)
	return reg
}

func register(reg *runtime.Registry) {
	registerActions(reg)
	registerAssertions(reg)
	registerRiskActions(reg)
	registerRiskAssertions(reg)
	registerEstimateActions(reg)
	registerEstimateAssertions(reg)
	registerMetricsAssertions(reg)
}

const unknownTaskNumber = 9999

func w(wd any) *world { return wd.(*world) }

func generate(s *world, doc wbs.RequirementDocument) error {
	id, err := s.svc.Generate(doc)
	s.lastErr = err
	if err == nil {
		s.wbsID = id
	}
	return nil
}

func validDocument(format string) wbs.RequirementDocument {
	return documentOf(format, "a valid requirement")
}

func emptyDocument(format string) wbs.RequirementDocument {
	return documentOf(format, "")
}

// documentOf builds a requirement document of the requested format ("text" or
// "pdf") carrying the given content.
func documentOf(format, content string) wbs.RequirementDocument {
	if format == "pdf" {
		return wbs.NewPDFDocument(wbs.EncodeMinimalPDF(content))
	}
	return wbs.NewTextDocument(content)
}

func currentWBS(s *world) (*wbs.WBS, error) {
	if s.wbsID == "" {
		return nil, errors.New("no current WBS")
	}
	return s.svc.Get(s.wbsID)
}

// currentTask returns the current WBS task at the one-based number parsed from a
// step capture. Risk-note and estimate assertions both look tasks up this way.
func currentTask(s *world, taskStr string) (wbs.Task, error) {
	cur, err := currentWBS(s)
	if err != nil {
		return wbs.Task{}, err
	}
	n, err := taskNumber(taskStr)
	if err != nil {
		return wbs.Task{}, err
	}
	task, ok := cur.TaskAt(n)
	if !ok {
		return wbs.Task{}, fmt.Errorf("task number %d does not exist", n)
	}
	return task, nil
}

func expectApproved(s *world, want bool) error {
	cur, err := currentWBS(s)
	if err != nil {
		return err
	}
	if cur.Approved() != want {
		return fmt.Errorf("approved = %v, want %v", cur.Approved(), want)
	}
	return nil
}

func expectCount(label string, got int, want string) error {
	n, err := strconv.Atoi(want)
	if err != nil {
		return fmt.Errorf("invalid expected %s %q: %w", label, want, err)
	}
	if got != n {
		return fmt.Errorf("%s = %d, want %d", label, got, n)
	}
	return nil
}

func expectNoError(wd any, _ []string) error {
	if err := w(wd).lastErr; err != nil {
		return fmt.Errorf("expected success, got error: %w", err)
	}
	return nil
}

func expectErr(target error) runtime.HandlerFunc {
	return func(wd any, _ []string) error {
		if err := w(wd).lastErr; !errors.Is(err, target) {
			return fmt.Errorf("error = %v, want %v", err, target)
		}
		return nil
	}
}

func taskNumber(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid task number %q: %w", s, err)
	}
	return n, nil
}

func splitTasks(s string) []string {
	parts := strings.Split(s, ";")
	tasks := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tasks = append(tasks, t)
		}
	}
	return tasks
}
