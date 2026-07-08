package wbs

import "errors"

// Domain-level failures shared by the WBS aggregate and the service.
var (
	// ErrEmptyDescription is returned when a task description is blank.
	ErrEmptyDescription = errors.New("description is empty")
	// ErrTaskNotFound is returned when a task number does not exist.
	ErrTaskNotFound = errors.New("task not found")
	// ErrNoTasks is returned when approving a WBS that has no tasks.
	ErrNoTasks = errors.New("WBS has no tasks")
	// ErrWBSNotFound is returned when a WBS id is unknown.
	ErrWBSNotFound = errors.New("WBS not found")
	// ErrEmptyRequirement is returned when a requirement document has no content.
	ErrEmptyRequirement = errors.New("requirement is empty")
	// ErrUnreadableDocument is returned when a requirement document cannot be read.
	ErrUnreadableDocument = errors.New("document cannot be read")
	// ErrWBSNotApproved is returned when risk flagging is attempted on a WBS that
	// has not been approved.
	ErrWBSNotApproved = errors.New("WBS is not approved")
	// ErrRiskNoteNotFound is returned when a risk note position does not exist.
	ErrRiskNoteNotFound = errors.New("risk note not found")
)

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-07T22:19:10+05:30","module_hash":"1bd0160f60c28486da9504d9b97bfe4dc6b97c0a54c2450f34e13fe46ab03bcd","functions":null}
// mutate4go-manifest-end
