package webui

import (
	"errors"
	"strconv"

	"estimation/internal/wbs"
)

// uiMessages maps each domain error to the friendly inline message the UI shows
// on the relevant step. The text matches the service's documented client-facing
// messages so the UI, HTTP API, and specs agree.
var uiMessages = []struct {
	err     error
	message string
}{
	{wbs.ErrEmptyRequirement, "requirement is empty"},
	{wbs.ErrUnreadableDocument, "document could not be read"},
	{wbs.ErrNoTasks, "cannot approve an empty WBS"},
	{wbs.ErrEmptyDescription, "description is empty"},
	{wbs.ErrTaskNotFound, "task not found"},
	{wbs.ErrRiskNoteNotFound, "risk note not found"},
	{wbs.ErrWBSNotFound, "WBS not found"},
	{wbs.ErrNoEstimate, "task has no estimate to override"},
	{wbs.ErrEstimatesNotGenerated, "estimates have not been generated"},
	{wbs.ErrOffFibonacciScale, "estimate value is off the Fibonacci scale"},
	{wbs.ErrEstimatesNotIncreasing, "estimate values must be strictly increasing"},
	{wbs.ErrEmptyReasoning, "reasoning is empty"},
	{wbs.ErrNonPositiveTeamInputs, "team inputs must be positive"},
	{wbs.ErrWBSNotApproved, "WBS must be approved before risk flagging"},
	{wbs.ErrWBSNotApprovedForEstimation, "WBS must be approved before estimation"},
	{wbs.ErrEstimatesNotApprovedForProposal, "estimates must be approved before a proposal"},
}

// message maps a domain error to its friendly inline message, falling back to a
// generic message for an unrecognized error.
func message(err error) string {
	for _, m := range uiMessages {
		if errors.Is(err, m.err) {
			return m.message
		}
	}
	return "something went wrong"
}

// basisText renders a recommended pricing basis, showing "none" when the band
// refuses a fixed basis (a nil basis, i.e. red).
func basisText(basis *int) string {
	if basis == nil {
		return "none"
	}
	return strconv.Itoa(*basis)
}
