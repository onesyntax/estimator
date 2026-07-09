package wbs

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

func (w *WBS) newNoteID() string { return nextID("r", &w.nextNote) }

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

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:33+05:30","module_hash":"b8f7ae0ec5f23aa3656b10faafcf1da8b518a6ca46454436eaa31bbc8f06a814","functions":[{"id":"func/WBS.newNoteID","name":"WBS.newNoteID","line":18,"end_line":18,"hash":"907cc49ccd17c1aad6fb4443fdaaebf79ee41cbd511e1513705493d149964f86"},{"id":"func/WBS.ReplaceRiskNotes","name":"WBS.ReplaceRiskNotes","line":24,"end_line":34,"hash":"a6ef6022c0a640964cc9075e3f6cb00b3016e45dc8d52e45bce366dc5a845a69"},{"id":"func/WBS.AddRiskNote","name":"WBS.AddRiskNote","line":39,"end_line":49,"hash":"8e0f91f194d1ec55e91998c3225844995a120234c00fcbbd6949f9fb6f2b8314"},{"id":"func/WBS.EditRiskNote","name":"WBS.EditRiskNote","line":54,"end_line":65,"hash":"9d4ca5ae40a1c17ef5a570019a4d582aebfc979bafb88d903a672185d17fe86b"},{"id":"func/WBS.DeleteRiskNote","name":"WBS.DeleteRiskNote","line":69,"end_line":76,"hash":"7226bf54e7366d16b1af4f4596696c0c2c20f41e707cf88ef13d6b5f0ac6f637"},{"id":"func/WBS.appendNote","name":"WBS.appendNote","line":78,"end_line":81,"hash":"867cd82c2b73e7da1fe08152d39df18c582ba474b44c1ca1c4f914b507df3d8f"},{"id":"func/WBS.taskExists","name":"WBS.taskExists","line":83,"end_line":85,"hash":"2f7aaf1a094113d73da9323d52acce5b9fb9989e9b07d0d5a1c8b0a2b66bf043"},{"id":"func/WBS.notesForEdit","name":"WBS.notesForEdit","line":90,"end_line":99,"hash":"a2ec656b7ba01df5063749b626127e45c29f498a910c3e679c1cf4d811bb0fac"}]}
// mutate4go-manifest-end
