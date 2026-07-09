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
	// ErrWBSNotApprovedForEstimation is returned when estimate generation is
	// attempted on a WBS that has not been approved.
	ErrWBSNotApprovedForEstimation = errors.New("WBS is not approved for estimation")
	// ErrNoEstimate is returned when a task without an estimate is overridden.
	ErrNoEstimate = errors.New("task has no estimate")
	// ErrEstimatesNotGenerated is returned when estimates are approved before
	// they have been generated.
	ErrEstimatesNotGenerated = errors.New("estimates have not been generated")
	// ErrOffFibonacciScale is returned when an estimate value is not on the
	// planning-poker Fibonacci scale.
	ErrOffFibonacciScale = errors.New("estimate value is off the Fibonacci scale")
	// ErrEstimatesNotIncreasing is returned when an estimate triple is not
	// strictly increasing.
	ErrEstimatesNotIncreasing = errors.New("estimate values are not strictly increasing")
	// ErrEmptyReasoning is returned when an estimate reasoning is blank.
	ErrEmptyReasoning = errors.New("reasoning is empty")
	// ErrEstimatesNotApprovedForProposal is returned when a client proposal is
	// requested before the estimate set has been approved.
	ErrEstimatesNotApprovedForProposal = errors.New("estimates are not approved for a proposal")
	// ErrNonPositiveTeamInputs is returned when a proposal's team inputs
	// (velocity, capacity, or rate) are not all positive.
	ErrNonPositiveTeamInputs = errors.New("team inputs must be positive")
)

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:32+05:30","module_hash":"3ab20dd3cbbe1e556af6f557073c76c6fa08ec6edaa66753d4786ddd6913e4c8","functions":null}
// mutate4go-manifest-end
