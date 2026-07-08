# mutation-stamp: sha256=fa655f0abe98541a15cf5041745383887f91302c17810719b2a0f2497f26955e
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-08T16:55:13.811427Z","feature_name":"Estimate Metrics","feature_path":"features/estimate_metrics.feature","background_hash":"3bad5e1405594c3165fd599cac526be2cdeeafb62be6b2eadb71525ac9ea5256","implementation_hash":"unknown","scenarios":[{"index":0,"name":"Estimate Metrics 1","scenario_hash":"cbaf8c3f347f8ace9198ab9c64329daea12500f23476d290161d70e03828980e","mutation_count":12,"result":{"Total":12,"Killed":12,"Survived":0,"Errors":0},"tested_at":"2026-07-08T16:55:11.317692Z"}]}
# acceptance-mutation-manifest-end

Feature: Estimate Metrics

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document
    And the WBS has been approved

  # Per-task PERT metrics are derived live from the current estimate set, so they
  # are present while the set is still unapproved. Each row is an independent
  # oracle for one task: expected = (O+4M+P)/6, standard deviation = (P-O)/6,
  # relative standard deviation = SD/E*100, each rounded half-up to a whole
  # number. The rows cover rounding up (task 1 E 5.83), rounding a small SD down
  # to zero (task 2 SD 0.33 while its RSD stays 17), and a mid-scale task.
  Scenario Outline: Estimate Metrics 1
    Given the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates
    Then the estimates are unapproved
    And task number <task_number> has metrics expected <e> standard deviation <sd> relative standard deviation <rsd>

    Examples:
      | task_number | e | sd | rsd |
      | 1           | 6 | 2  | 31  |
      | 2           | 2 | 0  | 17  |
      | 3           | 9 | 3  | 31  |

  # The project rollup adds variances: expected = sum of the task expected values,
  # standard deviation = sqrt(sum of task SD squared), relative standard deviation
  # = SD/E*100, from full-precision intermediates rounded once. Asserted on an
  # approved set to also pin that approval leaves the derived metrics unchanged.
  Scenario: Estimate Metrics 2
    Given the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates
    And the estimates have been approved
    Then the estimates are approved
    And the project metrics are expected 17 standard deviation 3 relative standard deviation 20

  # A Tech-Lead override recomputes the metrics immediately, before any re-approval.
  # The overridden task 1 (5/8/13) becomes 8/1/16, and the project expected value
  # is 117/6 = 19.5 exactly, which rounds half-up to 20 — pinning the tie-break.
  Scenario: Estimate Metrics 3
    Given the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates
    When the Tech Lead overrides task number 1 with optimistic 5 most likely 8 pessimistic 13 and reasoning Manual across all points
    Then task number 1 has metrics expected 8 standard deviation 1 relative standard deviation 16
    And the project metrics are expected 20 standard deviation 3 relative standard deviation 16

  # With no estimates generated, no metrics exist at either level.
  Scenario: Estimate Metrics 4
    Then task number 1 has no metrics
    And the project has no metrics

  # Partial estimates: priming only tasks 1 and 2 leaves task 3 unestimated, so it
  # carries no metrics, and the project rollup covers only the estimated tasks
  # (expected 47/6 = 7.83 -> 8, SD sqrt(125/36) = 1.86 -> 2, RSD 23.79 -> 24).
  Scenario: Estimate Metrics 5
    Given the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3
    And the Tech Lead has generated estimates
    Then task number 3 has no metrics
    And task number 1 has metrics expected 6 standard deviation 2 relative standard deviation 31
    And the project metrics are expected 8 standard deviation 2 relative standard deviation 24

  # Half-up tie and single-task rollup identity: task 1 = 0/1/5 gives expected
  # 9/6 = 1.5 (rounds up to 2), SD 5/6 = 0.83 -> 1, RSD 55.56 -> 56. With one
  # estimated task the project rollup equals that task (sqrt(SD^2) = SD).
  Scenario: Estimate Metrics 6
    Given the AI is primed to estimate task 1: 0/1/5
    And the Tech Lead has generated estimates
    Then task number 1 has metrics expected 2 standard deviation 1 relative standard deviation 56
    And the project metrics are expected 2 standard deviation 1 relative standard deviation 56
