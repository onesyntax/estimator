package main

// registerStage2Proposal registers the Stage-2 Proposal cases
// (qa/ui_pipeline_qa_suite.md §3, mirroring features/ui_stage2_proposal.feature).
// The setup for each case reaches an approved-estimates Proposal screen and then
// enters team inputs, asserting on the client-clean deliverable the screen
// shows.
func registerStage2Proposal() {
	// QA-UI-12 — The proposal reveal, per risk band.
	registerProposalBand(
		"QA-UI-12 proposal reveal (green)",
		"task 1: 2/3/5; task 2: 2/3/5; task 3: 2/3/5",
		"high", "fixed-price", 11400, 13478, 4, 4, 50, 98,
	)
	registerProposalBand(
		"QA-UI-12 proposal reveal (medium)",
		"task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20",
		"medium", "fixed-price-with-buffer", 24469, 28539, 7, 8, 84, 98,
	)
	registerProposalBand(
		"QA-UI-12 proposal reveal (lower)",
		"task 1: 1/8/40; task 2: 1/8/40; task 3: 1/8/40",
		"lower", "time-and-materials", 57310, 70820, 16, 20, 84, 98,
	)

	// QA-UI-13 — The proposal shows scope and assumptions, and hides mechanics.
	register("QA-UI-13 proposal scope/assumptions, mechanics hidden", func(t *T) {
		generateWBS()
		approveWBS()
		primeRisks("task 1: SQL injection; task 2: XSS")
		flagRisks()
		primeEstimates("task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20")
		generateEstimates()
		approveEstimates()
		s := requestProposal(3, 30, 120)

		t.listShows(scopeItems(s), "Login API")
		assumptions := assumptionItems(s)
		t.listShows(assumptions, "SQL injection")
		t.listShows(assumptions, "XSS")

		// The Stage-2 deliverable hides every raw mechanic Stage 1 revealed. The
		// checks read the visible <body>, not the <head> stylesheet, and use
		// Stage-1-only labels that never appear on the client-clean proposal.
		body := bodyContent(s)
		t.hides(body, "Most Likely")        // the per-task O/M/P columns
		t.hides(body, "Standard deviation") // the project PERT rollup
		t.hides(body, "Rel. std dev")
		t.hides(body, "PROJECT ROLLUP")
		t.hides(body, "PRICING STRATEGY") // the pricing band/flag
		t.hides(body, "Pricing basis")
		t.hides(body, "Risk level")
	})

	// QA-UI-14 — Non-positive team inputs are refused inline.
	registerBadTeamInputs("QA-UI-14 velocity 0", 0, 30, 120)
	registerBadTeamInputs("QA-UI-14 capacity 0", 3, 0, 120)
	registerBadTeamInputs("QA-UI-14 rate 0", 3, 30, 0)
	registerBadTeamInputs("QA-UI-14 velocity -3", -3, 30, 120)

	// QA-UI-15 — The Proposal stage is unreachable before estimates are approved.
	register("QA-UI-15 proposal unreachable before estimate approval", func(t *T) {
		generateWBS()
		approveWBS()
		primeEstimates("task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20")
		s := generateEstimates() // deliberately not approved
		if proposalReachable(s) {
			t.fail("Proposal stage reachable before estimates approved")
		}
		if onProposalScreen(screen("/proposal")) {
			t.fail("Proposal screen opened before estimates approved")
		}
	})
}

// approvedEstimatesProposalScreen runs the full Stage-1 pipeline for the given
// estimates and opens the Proposal screen, ready for team inputs.
func approvedEstimatesProposalScreen(estimates string) string {
	generateWBS()
	approveWBS()
	primeEstimates(estimates)
	generateEstimates()
	approveEstimates()
	return screen("/proposal")
}

// registerProposalBand registers a QA-UI-12 band: reach the Proposal screen for
// the band's estimates, enter velocity 3 / capacity 30 / rate 120, generate, and
// assert the confidence, contract, cost range, timeline range, and the two
// success probabilities.
func registerProposalBand(name, estimates, confidence, contract string, costLow, costHigh, weeksLow, weeksHigh, successLow, successHigh int) {
	register(name, func(t *T) {
		approvedEstimatesProposalScreen(estimates)
		s := requestProposal(3, 30, 120)
		t.eq("confidence", proposalBadge(s, "Confidence"), confidence)
		t.eq("contract", proposalBadge(s, "Contract"), contract)
		t.shows(s, costFigure(costLow))
		t.shows(s, costFigure(costHigh))
		t.shows(s, weeksAtFigure(weeksLow))
		t.shows(s, weeksAtFigure(weeksHigh))
		t.shows(s, successPct(successLow))
		t.shows(s, successPct(successHigh))
	})
}

// registerBadTeamInputs registers a QA-UI-14 case: on an approved-estimates
// Proposal screen, enter the non-positive inputs and assert the inline error
// with no proposal shown.
func registerBadTeamInputs(name string, velocity, capacity, rate int) {
	register(name, func(t *T) {
		approvedEstimatesProposalScreen("task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20")
		s := requestProposal(velocity, capacity, rate)
		t.inlineErrorIs(s, "team inputs must be positive")
		t.hides(s, ">Estimate summary<")
	})
}

// costFigure / weeksAtFigure / successPct are the distinctive visible strings the
// Proposal renders for a cost dollar figure, a timeline (in the per-figure
// success target), and a success percentage.
func costFigure(dollars int) string  { return ">$" + itoa(dollars) + "<" }
func weeksAtFigure(weeks int) string { return "· " + itoa(weeks) + " weeks<" }
func successPct(pct int) string      { return ">" + itoa(pct) + "%</div>" }
