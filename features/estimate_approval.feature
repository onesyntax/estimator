# mutation-stamp: sha256=25d5c02d934680d71c1fd7c5a377eea4bfd8aa1af414f92f3a97eccc031c146b
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-09T07:29:17.998102Z","feature_name":"Estimate Approval","feature_path":"features/estimate_approval.feature","background_hash":"d54f32dd0fadb9e3f44248dc04b848c7efba8b5790849353d30d9bcd640ccb42","implementation_hash":"unknown","scenarios":[{"index":2,"name":"Estimate Approval 3","scenario_hash":"25566106ad38e5aaef5f89d051dccb3b2e2824e6bda29bda8b0e3c733bff7ca0","mutation_count":2,"result":{"Total":2,"Killed":2,"Survived":0,"Errors":0},"tested_at":"2026-07-08T10:24:19.522157Z"}]}
# acceptance-mutation-manifest-end

Feature: Estimate Approval

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document
    And the WBS has been approved
    And the AI is primed to estimate task 1: 2/5/13 (Clean best, typical likely, legacy worst); task 2: 1/2/3 (Trivial across all points); task 3: 3/8/20 (Legacy widens the range)

  Scenario: Estimate Approval 1
    Given the Tech Lead has generated estimates
    When the Tech Lead approves the estimates
    Then the estimates are approved

  Scenario: Estimate Approval 2
    When the Tech Lead approves the estimates
    Then approval is rejected because estimates have not been generated
    And the estimates are unapproved

  Scenario Outline: Estimate Approval 3
    Given the Tech Lead has generated estimates
    And the estimates have been approved
    And the estimates are approved
    When the Tech Lead <override_action>
    Then the override is accepted
    And the estimates are unapproved

    Examples:
      | override_action                                                                                          |
      | overrides task number 1 with optimistic 3 most likely 8 pessimistic 20 and reasoning Team reviewed all points |
      | overrides task number 2 with optimistic 1 most likely 2 pessimistic 3 and reasoning Rechecked all points still trivial |

  Scenario: Estimate Approval 4
    Given the Tech Lead has generated estimates
    And the estimates have been approved
    And the Tech Lead overrides task number 1 with optimistic 3 most likely 8 pessimistic 20 and reasoning Team reviewed all points
    And the estimates are unapproved
    When the Tech Lead approves the estimates
    Then the estimates are approved
    And task number 1 has estimate optimistic 3 most likely 8 pessimistic 20
    And task number 1 has reasoning Team reviewed all points
