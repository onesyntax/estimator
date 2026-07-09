package main

// registerStage1Build registers the Stage-1 Build cases
// (qa/ui_pipeline_qa_suite.md §2, mirroring features/ui_stage1_build.feature).
// Every QA-UI-* case drives the app's screens as a user and asserts on what the
// screen shows.
func registerStage1Build() {
	// QA-UI-2 — A bad requirement shows an inline error and no WBS.
	//
	// The app keeps one session in server memory, so these run first, on the
	// pristine startup session, to assert the "no WBS" state a user sees before
	// generating anything. (A refused Generate never creates a WBS, so the
	// session stays clean across all three inputs.) Every later case begins by
	// generating a fresh WBS, which replaces the active one and isolates it.
	register("QA-UI-2 empty typed requirement", func(t *T) {
		s := submitRequirement("")
		t.inlineErrorIs(s, "requirement is empty")
		t.hides(s, "WORK BREAKDOWN")
	})
	register("QA-UI-2 pdf with no extractable text", func(t *T) {
		s := submitRequirementPDF(validPDF(""))
		t.inlineErrorIs(s, "requirement is empty")
		t.hides(s, "WORK BREAKDOWN")
	})
	register("QA-UI-2 corrupt pdf", func(t *T) {
		s := submitRequirementPDF(corruptPDF())
		t.inlineErrorIs(s, "document could not be read")
		t.hides(s, "WORK BREAKDOWN")
	})

	// QA-UI-1 — Requirement to WBS on screen (typed and PDF).
	register("QA-UI-1 requirement to WBS (typed)", func(t *T) {
		primeWBS("Login API; Login UI; Session store")
		s := submitRequirement("Build a login system.")
		got := taskDescriptions(s)
		t.eq("task count", len(got), 3)
		if len(got) > 0 {
			t.eq("task 1", got[0], "Login API")
		}
	})
	register("QA-UI-1 requirement to WBS (pdf)", func(t *T) {
		primeWBS("Login API; Login UI; Session store")
		s := submitRequirementPDF(validPDF("Build a login system."))
		got := taskDescriptions(s)
		t.eq("task count", len(got), 3)
		if len(got) > 0 {
			t.eq("task 1", got[0], "Login API")
		}
	})

	// QA-UI-3 — Editing the WBS is reflected on screen (each on a fresh WBS).
	register("QA-UI-3 add a task", func(t *T) {
		generateWBS()
		s := addTask("Password reset")
		t.tasksAre(s, "Login API", "Login UI", "Session store", "Password reset")
	})
	register("QA-UI-3 edit task 1", func(t *T) {
		generateWBS()
		s := editTask(1, "SSO API")
		t.tasksAre(s, "SSO API", "Login UI", "Session store")
	})
	register("QA-UI-3 delete task 1", func(t *T) {
		generateWBS()
		s := deleteTask(1)
		t.tasksAre(s, "Login UI", "Session store")
	})

	// QA-UI-4 — Risks and Estimates are locked until the WBS is approved.
	register("QA-UI-4 gates locked until approval", func(t *T) {
		s := generateWBS()
		if canFlagRisks(s) {
			t.fail("Risks section usable before approval")
		}
		if canGenerateEstimates(s) {
			t.fail("Estimates section usable before approval")
		}
		s = approveWBS()
		if !canFlagRisks(s) {
			t.fail("Risks section still locked after approval")
		}
		if !canGenerateEstimates(s) {
			t.fail("Estimates section still locked after approval")
		}
	})

	// QA-UI-5 — Approving an empty WBS is refused inline.
	register("QA-UI-5 approve empty WBS refused", func(t *T) {
		generateWBS()
		deleteTask(1)
		deleteTask(1)
		s := deleteTask(1)
		t.eq("no tasks left", len(taskDescriptions(s)), 0)
		s = approveWBS()
		t.inlineErrorIs(s, "cannot approve an empty WBS")
		if canFlagRisks(s) {
			t.fail("Risks section unlocked after refused approval")
		}
	})

	// QA-UI-6 — Flagging risks renders notes under their tasks.
	register("QA-UI-6 flag risks renders notes", func(t *T) {
		generateWBS()
		approveWBS()
		primeRisks("task 1: SQL injection; task 2: XSS")
		s := flagRisks()
		t.listShows(riskNotesUnder(s, 1), "SQL injection")
		t.listShows(riskNotesUnder(s, 2), "XSS")
	})

	// QA-UI-7 — Risk notes are editable on screen.
	register("QA-UI-7 add a manual risk note", func(t *T) {
		generateWBS()
		approveWBS()
		primeRisks("task 1: SQL injection")
		flagRisks()
		s := addRiskNote(2, "Manual concern")
		t.listShows(riskNotesUnder(s, 2), "Manual concern")
	})

	// QA-UI-8 — Estimates reveal the internal mechanics (per band).
	registerEstimateBand(
		"QA-UI-8 estimates reveal mechanics (green)",
		"task 1: 2/3/5; task 2: 2/3/5; task 3: 2/3/5",
		2, 3, 5, 10, 1, 9, "green", "fixed-price", "10",
	)
	registerEstimateBand(
		"QA-UI-8 estimates reveal mechanics (yellow)",
		"task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20",
		2, 5, 13, 17, 3, 20, "yellow", "fixed-price-with-buffer", "20",
	)
	registerEstimateBand(
		"QA-UI-8 estimates reveal mechanics (red)",
		"task 1: 1/8/40; task 2: 1/8/40; task 3: 1/8/40",
		1, 8, 40, 37, 11, 31, "red", "time-and-materials", "none",
	)

	// QA-UI-9 — A valid estimate override is applied on screen.
	register("QA-UI-9 valid estimate override", func(t *T) {
		generateWBS()
		approveWBS()
		primeEstimates("task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20")
		generateEstimates()
		s := overrideEstimate(1, 3, 8, 20, "revised after review")
		t.task1EstimateIs(s, 3, 8, 20)
	})

	// QA-UI-10 — An invalid override is refused inline, estimate unchanged.
	registerInvalidOverride(
		"QA-UI-10 override not increasing",
		8, 5, 13, "estimate values must be strictly increasing",
	)
	registerInvalidOverride(
		"QA-UI-10 override off Fibonacci scale",
		4, 5, 13, "estimate value is off the Fibonacci scale",
	)

	// QA-UI-11 — Approving the estimates unlocks Stage 2.
	register("QA-UI-11 approving estimates unlocks Stage 2", func(t *T) {
		generateWBS()
		approveWBS()
		primeEstimates("task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20")
		s := generateEstimates()
		if proposalReachable(s) {
			t.fail("Proposal stage reachable before estimates approved")
		}
		if onProposalScreen(screen("/proposal")) {
			t.fail("Proposal screen opened before estimates approved")
		}
		s = approveEstimates()
		if !proposalReachable(s) {
			t.fail("Proposal stage not reachable after approving estimates")
		}
		if !onProposalScreen(screen("/proposal")) {
			t.fail("Proposal screen not reachable after approving estimates")
		}
	})
}

// registerEstimateBand registers a QA-UI-8 band: generate a WBS, approve it,
// prime the band's estimates, generate them, and assert task 1's O/M/P, the
// metrics panel's project E/SD/RSD, and the pricing panel's flag/contract/basis.
func registerEstimateBand(name, estimates string, o, m, p, expected, sd, rsd int, flag, contract, basis string) {
	register(name, func(t *T) {
		generateWBS()
		approveWBS()
		primeEstimates(estimates)
		s := generateEstimates()
		t.task1EstimateIs(s, o, m, p)
		t.eq("project expected", metricValue(s, "Expected effort"), itoa(expected))
		t.eq("project SD", metricValue(s, "Standard deviation"), itoa(sd))
		t.eq("project RSD", metricValue(s, "Rel. std dev (RSD)"), itoa(rsd))
		t.eq("pricing flag", pricingFlag(s), flag)
		t.eq("contract", pricingValue(s, "Contract type"), contract)
		t.eq("recommended basis", pricingValue(s, "Pricing basis"), basis)
	})
}

// registerInvalidOverride registers a QA-UI-10 case: on task 1 generated as
// 2/5/13, attempt an invalid override and assert the inline message with task 1
// still showing 2/5/13.
func registerInvalidOverride(name string, o, m, p int, message string) {
	register(name, func(t *T) {
		generateWBS()
		approveWBS()
		primeEstimates("task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20")
		generateEstimates()
		s := overrideEstimate(1, o, m, p, "attempted change")
		t.inlineErrorIs(s, message)
		t.task1EstimateIs(s, 2, 5, 13)
	})
}
