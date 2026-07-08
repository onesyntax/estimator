# mutation-stamp: sha256=9d51e035a24aa297e1f99b1fe8b57eb62644b35f9b6b0e2e89368518c3687665
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-08T16:55:13.870595Z","feature_name":"Estimate Override","feature_path":"features/estimate_override.feature","background_hash":"d54f32dd0fadb9e3f44248dc04b848c7efba8b5790849353d30d9bcd640ccb42","implementation_hash":"unknown","scenarios":[{"index":0,"name":"Estimate Override 1","scenario_hash":"0839de9af5fa38a103642c5682f53183c5638891a6aa9118a14d8bdc35c3c13e","mutation_count":18,"result":{"Total":18,"Killed":18,"Survived":0,"Errors":0},"tested_at":"2026-07-08T10:32:02.207603Z"},{"index":1,"name":"Estimate Override 2","scenario_hash":"7ae858b8b4b5f168389b1b0f417870e9bbce56abbf67242de2b64a03843999ae","mutation_count":14,"result":{"Total":14,"Killed":14,"Survived":0,"Errors":0},"tested_at":"2026-07-08T10:32:02.207603Z"},{"index":2,"name":"Estimate Override 3","scenario_hash":"92656b521d51f74df1dbf9ec7a59c90e7de793849fc580b4ecaaebfeaa637c1d","mutation_count":1,"result":{"Total":1,"Killed":1,"Survived":0,"Errors":0},"tested_at":"2026-07-08T10:32:02.207603Z"}]}
# acceptance-mutation-manifest-end

Feature: Estimate Override

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document
    And the WBS has been approved
    And the AI is primed to estimate task 1: 2/5/13 (Clean best, typical likely, legacy worst); task 2: 1/2/3 (Trivial across all points); task 3: 3/8/20 (Legacy widens the range)

  # The override action is a single literal column so the stimulus stays fixed
  # while the assertion reads independent oracle columns: any mutation of the
  # action breaks the step match, drives a value off-scale, or diverges the
  # stored estimate from the oracle, and any mutation of an oracle column checks
  # the wrong task or the wrong value — every case is detected.
  Scenario Outline: Estimate Override 1
    Given the Tech Lead has generated estimates
    When the Tech Lead <override_action>
    Then the override is accepted
    And task number <task_number> has estimate optimistic <o> most likely <m> pessimistic <p>
    And task number <task_number> has reasoning <reasoning>

    Examples:
      | override_action                                                                                                        | task_number | o | m  | p   | reasoning                           |
      | overrides task number 1 with optimistic 3 most likely 8 pessimistic 20 and reasoning Team set O M and P together        | 1           | 3 | 8  | 20  | Team set O M and P together         |
      | overrides task number 2 with optimistic 0 most likely 1 pessimistic 2 and reasoning All points minimal for trivial work | 2           | 0 | 1  | 2   | All points minimal for trivial work |
      | overrides task number 3 with optimistic 8 most likely 40 pessimistic 100 and reasoning Rewrite pushes every point higher | 3           | 8 | 40 | 100 | Rewrite pushes every point higher   |

  # Invalid overrides: the action is a literal, and the oracle pins both the
  # rejection reason and that task 1 keeps its generated 2/5/13 estimate. A
  # mutation that makes the action valid is caught when task 1 changes; one that
  # changes the rejection class is caught by the reason; one that breaks the step
  # text is caught as an unmatched step.
  Scenario Outline: Estimate Override 2
    Given the Tech Lead has generated estimates
    When the Tech Lead <override_action>
    Then the override is rejected because <reason>
    And task number 1 has estimate optimistic 2 most likely 5 pessimistic 13

    Examples:
      | override_action                                                                                       | reason                                 |
      | overrides task number 1 with optimistic 8 most likely 5 pessimistic 13 and reasoning Adjusted for risk | the values are not strictly increasing |
      | overrides task number 1 with optimistic 2 most likely 20 pessimistic 13 and reasoning Adjusted for risk | the values are not strictly increasing |
      | overrides task number 1 with optimistic 5 most likely 5 pessimistic 8 and reasoning Adjusted for risk  | the values are not strictly increasing |
      | overrides task number 1 with optimistic 8 most likely 13 pessimistic 13 and reasoning Adjusted for risk | the values are not strictly increasing |
      | overrides task number 1 with optimistic 4 most likely 5 pessimistic 13 and reasoning Adjusted for risk | the value is off the Fibonacci scale   |
      | overrides task number 1 with optimistic 21 most likely 40 pessimistic 100 and reasoning Adjusted for risk | the value is off the Fibonacci scale |
      | overrides task number 1 with optimistic -1 most likely 5 pessimistic 13 and reasoning Adjusted for risk | the value is off the Fibonacci scale   |

  # Empty-reasoning rejection kept as its own outline: the reasoning is the only
  # mutable cell, and dithering the empty string inserts a character, which makes
  # the otherwise-valid 3/8/20 override succeed and fails the rejection oracle.
  Scenario Outline: Estimate Override 3
    Given the Tech Lead has generated estimates
    When the Tech Lead overrides task number 1 with optimistic 3 most likely 8 pessimistic 20 and reasoning <reasoning>
    Then the override is rejected because the reasoning is empty
    And task number 1 has estimate optimistic 2 most likely 5 pessimistic 13

    Examples:
      | reasoning |
      |           |

  Scenario: Estimate Override 4
    Given the Tech Lead has generated estimates
    When the Tech Lead overrides a task that does not exist with optimistic 1 most likely 2 pessimistic 3 and reasoning New estimate for all points
    Then the task is reported as not found

  Scenario: Estimate Override 5
    When the Tech Lead overrides task number 1 with optimistic 1 most likely 2 pessimistic 3 and reasoning New estimate for all points
    Then the override is rejected because the task has no estimate
    And task number 1 has no estimate
