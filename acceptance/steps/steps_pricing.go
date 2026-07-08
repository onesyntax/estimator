package steps

import (
	"fmt"

	"estimation/acceptance/runtime"
	"estimation/internal/wbs"
)

// registerPricingAssertions registers the steps that assert on the derived
// pricing strategy.
func registerPricingAssertions(reg *runtime.Registry) {
	reg.Step(`^the pricing strategy is flag (\S+) risk (\S+) contract (\S+) basis (\S+)$`,
		func(wd any, a []string) error {
			ps, ok := currentPricing(w(wd))
			if !ok {
				return fmt.Errorf("WBS has no pricing strategy, want flag %s", a[0])
			}
			return expectPricing(ps, a[0], a[1], a[2], a[3])
		})

	reg.Step(`^the WBS has no pricing strategy$`,
		func(wd any, _ []string) error {
			if ps, ok := currentPricing(w(wd)); ok {
				return fmt.Errorf("WBS has pricing strategy %+v, want none", ps)
			}
			return nil
		})
}

// currentPricing derives the current WBS's pricing strategy.
func currentPricing(s *world) (wbs.PricingStrategy, bool) {
	cur, err := currentWBS(s)
	if err != nil {
		return wbs.PricingStrategy{}, false
	}
	return cur.PricingStrategy()
}

// expectPricing checks a pricing strategy against the captured flag, risk level,
// contract, and basis. The basis capture is a whole number, or "none" for a red
// project with no fixed-price basis.
func expectPricing(ps wbs.PricingStrategy, flag, risk, contract, basis string) error {
	if ps.Flag != flag || ps.RiskLevel != risk || ps.Contract != contract {
		return fmt.Errorf("strategy = %s/%s/%s, want %s/%s/%s", ps.Flag, ps.RiskLevel, ps.Contract, flag, risk, contract)
	}
	if basis == "none" {
		if ps.RecommendedBasis != nil {
			return fmt.Errorf("recommendedBasis = %d, want none", *ps.RecommendedBasis)
		}
		return nil
	}
	want, err := atoi(basis)
	if err != nil {
		return err
	}
	if ps.RecommendedBasis == nil {
		return fmt.Errorf("recommendedBasis = none, want %d", want)
	}
	if *ps.RecommendedBasis != want {
		return fmt.Errorf("recommendedBasis = %d, want %d", *ps.RecommendedBasis, want)
	}
	return nil
}
