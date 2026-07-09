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

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:56+05:30","module_hash":"6d76ad421cb6e2d45c0be246d7130b8c36de90cb6829b3dbd92d1d86450e7acf","functions":[{"id":"func/message","name":"message","line":37,"end_line":44,"hash":"ac55023db4803cd74a6ffb6c6453601e493ea45c25c133d1646cad2d49c7d292"},{"id":"func/basisText","name":"basisText","line":48,"end_line":53,"hash":"0c90d736f1db296a0937899a5bf7e1fe654f84eaa086fa0f37bdf8185615afff"}]}
// mutate4go-manifest-end
