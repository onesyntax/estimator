# mutation-stamp: sha256=67292c4e50093b19781035b0dbf659b3ca0daefb41963de961094d5676dbc10a
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-09T07:29:18.263433Z","feature_name":"Pricing Strategy","feature_path":"features/pricing_strategy.feature","background_hash":"3bad5e1405594c3165fd599cac526be2cdeeafb62be6b2eadb71525ac9ea5256","implementation_hash":"unknown","scenarios":[{"index":0,"name":"Pricing Strategy 1","scenario_hash":"4475146f93e98e483b29f60be0a2034b2732d1cdbe167adcb6d5c2fff80186b4","mutation_count":15,"result":{"Total":15,"Killed":15,"Survived":0,"Errors":0},"tested_at":"2026-07-08T18:29:29.843532Z"}]}
# acceptance-mutation-manifest-end

Feature: Pricing Strategy

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document
    And the WBS has been approved

  # The pricing strategy is derived live from the project metrics (the same
  # integer project RSD Module 4 emits), so it is present while the estimate set
  # is still unapproved. Each row is one risk band: RSD < 10 is green (low risk,
  # fixed price on the expected value), 10 <= RSD <= 20 is yellow (medium risk,
  # fixed price on expected + 1 SD), RSD > 20 is red (high risk, no fixed price,
  # Time & Materials with no basis). The green set 3x2/3/5 gives RSD 9, the yellow
  # set the standard triples give RSD exactly 20 (pinning the inclusive upper
  # boundary), and the red set 3x1/8/40 gives RSD 31.
  Scenario Outline: Pricing Strategy 1
    Given the AI is primed to estimate <estimates>
    And the Tech Lead has generated estimates
    Then the estimates are unapproved
    And the pricing strategy is flag <flag> risk <risk_level> contract <contract> basis <basis>

    Examples:
      | estimates                                     | flag   | risk_level | contract                | basis |
      | task 1: 2/3/5; task 2: 2/3/5; task 3: 2/3/5   | green  | low        | fixed-price             | 10    |
      | task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20 | yellow | medium     | fixed-price-with-buffer | 20    |
      | task 1: 1/8/40; task 2: 1/8/40; task 3: 1/8/40 | red    | high       | time-and-materials      | none  |

  # With no estimates there are no project metrics, so there is no pricing
  # strategy to derive.
  Scenario: Pricing Strategy 2
    Then the WBS has no pricing strategy
