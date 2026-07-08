package main

// registerPricing registers the Module 5 cases (qa/pricing_qa_suite.md, mirroring
// features/pricing_strategy.feature). Module 5 adds a derived `pricingStrategy`
// block to the WBS view, computed deterministically (no AI) from the whole-number
// project RSD Module 4 emits. It appears live as soon as estimates exist — whether
// the set is unapproved or approved — with no new endpoint and no new stored
// state; it is recomputed on every GET /wbs/{id}. This is the internal Tech-Lead
// interface, so it exposes the flag, level, contract, and recommendedBasis; the
// client never sees it. QA drives estimates exactly as in Module 3 and only reads
// the new block from GET /wbs/{id}.

// pricingStrategyView is the derived pricing strategy as GET /wbs/{id} renders it;
// nil (JSON null) when there are no project metrics (no estimates). RecommendedBasis
// is a points figure, or nil (JSON null) for the red band where a fixed price is
// refused.
type pricingStrategyView struct {
	Flag             string `json:"flag"`
	RiskLevel        string `json:"riskLevel"`
	Contract         string `json:"contract"`
	RecommendedBasis *int   `json:"recommendedBasis"`
}

// uniformEstimates primes the same O/M/P triple on all three tasks with a shared
// reasoning — the shape both single-band prime below uses.
func uniformEstimates(o, m, p int, reasoning string) []estimateAssign {
	return []estimateAssign{
		{1, o, m, p, reasoning},
		{2, o, m, p, reasoning},
		{3, o, m, p, reasoning},
	}
}

// greenEstimates is the shared low-risk prime: 2/3/5 on all three tasks, landing
// at project RSD 9 (green band). Shared with the proposal suite.
func greenEstimates() []estimateAssign {
	return uniformEstimates(2, 3, 5, "Tight spread, low risk")
}

// redEstimates is the shared high-risk prime: 1/8/40 on all three tasks, landing
// at project RSD 31 (red band). Shared with the proposal suite.
func redEstimates() []estimateAssign {
	return uniformEstimates(1, 8, 40, "Very wide range, high risk")
}

// pricingStrategyOf returns the WBS-level pricingStrategy block, or nil.
func pricingStrategyOf(id string) *pricingStrategyView {
	return getJSON("/wbs/" + id).view().PricingStrategy
}

// pricingIs asserts the pricingStrategy block matches the expected flag, level,
// contract and recommendedBasis (basis < 0 means "expect null / no fixed price").
func (t *T) pricingIs(id, flag, level, contract string, basis int) {
	p := pricingStrategyOf(id)
	if p == nil {
		t.fail("pricingStrategy = null, want %s/%s/%s", flag, level, contract)
		return
	}
	t.eq("pricing flag", p.Flag, flag)
	t.eq("pricing riskLevel", p.RiskLevel, level)
	t.eq("pricing contract", p.Contract, contract)
	if basis < 0 {
		if p.RecommendedBasis != nil {
			t.fail("recommendedBasis = %d, want null", *p.RecommendedBasis)
		}
		return
	}
	if p.RecommendedBasis == nil {
		t.fail("recommendedBasis = null, want %d", basis)
		return
	}
	t.eq("recommendedBasis", *p.RecommendedBasis, basis)
}

// noPricingStrategy asserts the WBS has no pricingStrategy block (JSON null).
func (t *T) noPricingStrategy(id string) {
	if p := pricingStrategyOf(id); p != nil {
		t.fail("pricingStrategy = %s/%s/%s, want null", p.Flag, p.RiskLevel, p.Contract)
	}
}

func registerPricing() {
	// --- Pricing Strategy (pricing_strategy.feature) -----------------------

	// QA-PRC-1 — The three risk bands map to the right strategy, live on a
	// generated-but-unapproved set. Each row primes its estimates, generates
	// (without approving), then reads the derived pricingStrategy.
	register("QA-PRC-1 risk bands map to strategy (live, unapproved)", func(t *T) {
		type bandCase struct {
			estimates []estimateAssign
			flag      string
			level     string
			contract  string
			basis     int // < 0 means expect null (red)
		}
		for _, c := range []bandCase{
			{greenEstimates(), "green", "low", "fixed-price", 10},
			{standardEstimates(), "yellow", "medium", "fixed-price-with-buffer", 20},
			{redEstimates(), "red", "high", "time-and-materials", -1},
		} {
			id := generatedEstimates(t, c.estimates) // approved WBS, estimates generated (not approved)
			t.estimatesStateIs(id, "unapproved")
			t.pricingIs(id, c.flag, c.level, c.contract, c.basis)
		}
	})

	// QA-PRC-2 — No estimates ⇒ no pricing strategy (and no project metrics).
	register("QA-PRC-2 no estimates means no pricing strategy", func(t *T) {
		id := approvedWBS(t) // estimates not generated
		t.noPricingStrategy(id)
		t.noProjectMetrics(id)
	})
}
