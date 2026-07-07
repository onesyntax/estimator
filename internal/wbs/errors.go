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
)
