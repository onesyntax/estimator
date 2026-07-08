# mutation-stamp: sha256=720228f6e1b6d276ab28e539a3f1b1e0668f4fcff04b57327eccd584b140f55a
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-08T10:32:27.278146Z","feature_name":"Estimate Generation","feature_path":"features/estimate_generation.feature","background_hash":"98b98f33ccbcaebdc7684ea295ea82abe118d4344be6c097ca261cfb673c54a0","implementation_hash":"unknown","scenarios":[{"index":0,"name":"Estimate Generation 1","scenario_hash":"12eff9b11eb7fb71919697da0da862a7b8c4285e8f6f63675974d04a2920cc0f","mutation_count":15,"result":{"Total":15,"Killed":15,"Survived":0,"Errors":0},"tested_at":"2026-07-08T10:24:19.478595Z"},{"index":2,"name":"Estimate Generation 3","scenario_hash":"767053f1ad84f851541f0e562ec6e86ab45c3f8b8f6ff28a33e22159c0b0b767","mutation_count":15,"result":{"Total":15,"Killed":15,"Survived":0,"Errors":0},"tested_at":"2026-07-08T10:24:19.478595Z"}]}
# acceptance-mutation-manifest-end

Feature: Estimate Generation

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document

  Scenario Outline: Estimate Generation 1
    Given the WBS has been approved
    And the AI is primed to estimate task 1: 2/5/13 (Clean best, typical likely, legacy worst); task 2: 1/2/3 (Trivial across all points); task 3: 3/8/20 (Legacy widens the range)
    When the Tech Lead generates estimates
    Then the estimates are unapproved
    And task number <task_number> has estimate optimistic <o> most likely <m> pessimistic <p>
    And task number <task_number> has reasoning <reasoning>

    Examples:
      | task_number | o | m | p  | reasoning                                |
      | 1           | 2 | 5 | 13 | Clean best, typical likely, legacy worst |
      | 2           | 1 | 2 | 3  | Trivial across all points                |
      | 3           | 3 | 8 | 20 | Legacy widens the range                  |

  Scenario: Estimate Generation 2
    Given the AI is primed to estimate task 1: 2/5/13
    When the Tech Lead generates estimates
    Then estimation is rejected because the WBS is unapproved
    And task number 1 has no estimate

  Scenario Outline: Estimate Generation 3
    Given the WBS has been approved
    And the AI is primed to estimate task 1: 2/5/13 (Clean best, typical likely, legacy worst); task 2: 1/2/3 (Trivial across all points); task 3: 3/8/20 (Legacy widens the range)
    And the Tech Lead has generated estimates
    And the Tech Lead overrides task number 1 with optimistic 5 most likely 8 pessimistic 13 and reasoning Manual across all points
    And the AI is primed to estimate task 1: 5/13/40 (Rewrite lifts all points); task 2: 3/5/8 (Validation adds spread); task 3: 8/20/40 (Migration keeps wide range)
    When the Tech Lead generates estimates
    Then task number <task_number> has estimate optimistic <o> most likely <m> pessimistic <p>
    And task number <task_number> has reasoning <reasoning>

    Examples:
      | task_number | o | m  | p  | reasoning                  |
      | 1           | 5 | 13 | 40 | Rewrite lifts all points   |
      | 2           | 3 | 5  | 8  | Validation adds spread     |
      | 3           | 8 | 20 | 40 | Migration keeps wide range |

  Scenario: Estimate Generation 4
    When a Tech Lead generates estimates on a WBS that does not exist
    Then the WBS is reported as not found
