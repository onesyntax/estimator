package wbs

// PricingStrategy is the internal Tech-Lead pricing decision derived live from
// the project metrics. RecommendedBasis is the whole-number points figure a
// quote should be built on, or nil for a red project where a fixed price is
// refused.
type PricingStrategy struct {
	Flag             string // "green" | "yellow" | "red"
	RiskLevel        string // "low" | "medium" | "high"
	Contract         string // "fixed-price" | "fixed-price-with-buffer" | "time-and-materials"
	RecommendedBasis *int
}

// band collects every value the risk band fixes, so Module 5 (pricing) and
// Module 6 (proposal) share one source of truth keyed off the project RSD.
// lowSDMultiplier is how many standard deviations above the expected value the
// range's low bound (and the pricing basis) sits: 0 for green (the mean itself),
// 1 for yellow and red (the risk buffer). hasFixedBasis is false only for red,
// where a fixed price is refused.
type band struct {
	flag                 string
	riskLevel            string
	contract             string
	confidence           string
	invitesRenegotiation bool
	lowSDMultiplier      int
	hasFixedBasis        bool
}

// bandForRSD maps the whole-number project relative standard deviation to its
// risk band. The yellow band is inclusive at both edges: RSD 9 is green, 10 and
// 20 are yellow, 21 is red.
func bandForRSD(rsd int) band {
	switch {
	case rsd < 10:
		return band{flag: "green", riskLevel: "low", contract: "fixed-price", confidence: "high", invitesRenegotiation: false, lowSDMultiplier: 0, hasFixedBasis: true}
	case rsd <= 20:
		return band{flag: "yellow", riskLevel: "medium", contract: "fixed-price-with-buffer", confidence: "medium", invitesRenegotiation: false, lowSDMultiplier: 1, hasFixedBasis: true}
	default:
		return band{flag: "red", riskLevel: "high", contract: "time-and-materials", confidence: "lower", invitesRenegotiation: true, lowSDMultiplier: 1, hasFixedBasis: false}
	}
}

// PricingStrategy derives the pricing decision from the current estimate set.
// ok is false when no task has an estimate, so there are no project metrics to
// band. The strategy is computed live and does not require approval.
func (w *WBS) PricingStrategy() (PricingStrategy, bool) {
	v, ok := w.projectPert()
	if !ok {
		return PricingStrategy{}, false
	}
	b := bandForRSD(v.metrics().RelativeStandardDeviation)
	ps := PricingStrategy{Flag: b.flag, RiskLevel: b.riskLevel, Contract: b.contract}
	if b.hasFixedBasis {
		basis := roundHalfUp(v.expected() + float64(b.lowSDMultiplier)*v.stdDev)
		ps.RecommendedBasis = &basis
	}
	return ps, true
}
