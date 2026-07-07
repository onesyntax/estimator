package wbs

import (
	"fmt"
	"strings"
)

// Task is a single unit of work in a WBS. Its ID is stable for the lifetime of
// the task even as tasks around it are added or deleted.
type Task struct {
	ID          string
	Description string
}

// WBS (Work Breakdown Structure) is the aggregate produced from a requirement
// document. Tasks are ordered and referred to by their one-based position
// ("task number N"). Any structural edit unapproves the WBS.
type WBS struct {
	id       string
	tasks    []Task
	approved bool
	nextSeq  int
}

// NewWBS builds an unapproved WBS from the given ordered task descriptions.
func NewWBS(id string, descriptions []string) *WBS {
	w := &WBS{id: id, nextSeq: 1}
	for _, d := range descriptions {
		w.tasks = append(w.tasks, Task{ID: w.newTaskID(), Description: d})
	}
	return w
}

func (w *WBS) newTaskID() string {
	id := fmt.Sprintf("t%d", w.nextSeq)
	w.nextSeq++
	return id
}

// ID returns the WBS identifier.
func (w *WBS) ID() string { return w.id }

// Approved reports whether the WBS is currently approved.
func (w *WBS) Approved() bool { return w.approved }

// TaskCount returns the number of tasks in the WBS.
func (w *WBS) TaskCount() int { return len(w.tasks) }

// Tasks returns a copy of the ordered task list.
func (w *WBS) Tasks() []Task {
	out := make([]Task, len(w.tasks))
	copy(out, w.tasks)
	return out
}

// TaskAt returns the task at the given one-based number, or ok=false when the
// number is out of range.
func (w *WBS) TaskAt(number int) (Task, bool) {
	if number < 1 || number > len(w.tasks) {
		return Task{}, false
	}
	return w.tasks[number-1], true
}

// ApprovedTasks returns the approved task list (the module output) while the
// WBS is approved, and nil otherwise.
func (w *WBS) ApprovedTasks() []Task {
	if !w.approved {
		return nil
	}
	return w.Tasks()
}

// AddTask appends a task with the given description and unapproves the WBS.
func (w *WBS) AddTask(description string) error {
	description, err := requireDescription(description)
	if err != nil {
		return err
	}
	w.tasks = append(w.tasks, Task{ID: w.newTaskID(), Description: description})
	w.approved = false
	return nil
}

// EditTask replaces the description of the task at the one-based number and
// unapproves the WBS.
func (w *WBS) EditTask(number int, description string) error {
	if number < 1 || number > len(w.tasks) {
		return ErrTaskNotFound
	}
	description, err := requireDescription(description)
	if err != nil {
		return err
	}
	w.tasks[number-1].Description = description
	w.approved = false
	return nil
}

// DeleteTask removes the task at the one-based number and unapproves the WBS.
func (w *WBS) DeleteTask(number int) error {
	if number < 1 || number > len(w.tasks) {
		return ErrTaskNotFound
	}
	w.tasks = append(w.tasks[:number-1], w.tasks[number:]...)
	w.approved = false
	return nil
}

// Approve marks the WBS approved. It fails when the WBS has no tasks.
func (w *WBS) Approve() error {
	if len(w.tasks) == 0 {
		return ErrNoTasks
	}
	w.approved = true
	return nil
}

func requireDescription(description string) (string, error) {
	trimmed := strings.TrimSpace(description)
	if trimmed == "" {
		return "", ErrEmptyDescription
	}
	return trimmed, nil
}
