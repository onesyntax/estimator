package wbs

import "fmt"

// RiskNote is a Tech Lead's risk annotation on a task. Its ID is stable so a
// client can address it directly; within a task, notes are also ordered and
// referred to by their one-based position.
type RiskNote struct {
	ID          string
	Description string
}

// RiskAssignment pairs a one-based task number with a risk description. It is
// the unit a RiskProvider returns and the unit ReplaceRiskNotes applies.
type RiskAssignment struct {
	TaskNumber  int
	Description string
}

func (w *WBS) newNoteID() string {
	id := fmt.Sprintf("r%d", w.nextNote)
	w.nextNote++
	return id
}

// ReplaceRiskNotes discards every task's existing risk notes and reapplies the
// given assignments, so re-flagging is destructive: prior AI-generated and
// manually added notes are all replaced. Assignments for task numbers outside
// the current range are ignored. Approval state is left unchanged.
func (w *WBS) ReplaceRiskNotes(assignments []RiskAssignment) {
	for i := range w.tasks {
		w.tasks[i].RiskNotes = nil
	}
	for _, a := range assignments {
		if a.TaskNumber < 1 || a.TaskNumber > len(w.tasks) {
			continue
		}
		w.appendNote(a.TaskNumber, a.Description)
	}
}

// AddRiskNote appends a risk note to the task at the one-based number. It fails
// with ErrTaskNotFound for an unknown task and ErrEmptyDescription for a blank
// description. Adding a note does not change the WBS's approval state.
func (w *WBS) AddRiskNote(taskNumber int, description string) error {
	if !w.taskExists(taskNumber) {
		return ErrTaskNotFound
	}
	description, err := requireDescription(description)
	if err != nil {
		return err
	}
	w.appendNote(taskNumber, description)
	return nil
}

// EditRiskNote replaces the description of the risk note at the one-based
// position on the task at the one-based number, keeping the note's id. It fails
// with ErrTaskNotFound, ErrRiskNoteNotFound, or ErrEmptyDescription.
func (w *WBS) EditRiskNote(taskNumber, notePosition int, description string) error {
	notes, err := w.notesForEdit(taskNumber, notePosition)
	if err != nil {
		return err
	}
	description, err = requireDescription(description)
	if err != nil {
		return err
	}
	notes[notePosition-1].Description = description
	return nil
}

// DeleteRiskNote removes the risk note at the one-based position on the task at
// the one-based number. It fails with ErrTaskNotFound or ErrRiskNoteNotFound.
func (w *WBS) DeleteRiskNote(taskNumber, notePosition int) error {
	if _, err := w.notesForEdit(taskNumber, notePosition); err != nil {
		return err
	}
	notes := w.tasks[taskNumber-1].RiskNotes
	w.tasks[taskNumber-1].RiskNotes = append(notes[:notePosition-1], notes[notePosition:]...)
	return nil
}

func (w *WBS) appendNote(taskNumber int, description string) {
	task := &w.tasks[taskNumber-1]
	task.RiskNotes = append(task.RiskNotes, RiskNote{ID: w.newNoteID(), Description: description})
}

func (w *WBS) taskExists(taskNumber int) bool {
	return taskNumber >= 1 && taskNumber <= len(w.tasks)
}

// notesForEdit resolves the risk-note slice of an existing task, validating the
// task number and the note position. It returns the task's note slice so the
// caller can edit in place.
func (w *WBS) notesForEdit(taskNumber, notePosition int) ([]RiskNote, error) {
	if !w.taskExists(taskNumber) {
		return nil, ErrTaskNotFound
	}
	notes := w.tasks[taskNumber-1].RiskNotes
	if notePosition < 1 || notePosition > len(notes) {
		return nil, ErrRiskNoteNotFound
	}
	return notes, nil
}
