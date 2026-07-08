Feature: Estimate Override

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document
    And the WBS has been approved
    And the AI is primed to estimate task 1: 2/5/13 (Clean best, typical likely, legacy worst); task 2: 1/2/3 (Trivial across all points); task 3: 3/8/20 (Legacy widens the range)

  Scenario Outline: Estimate Override 1
    Given the Tech Lead has generated estimates
    When the Tech Lead overrides task number <task_number> with optimistic <o> most likely <m> pessimistic <p> and reasoning <reasoning>
    Then the override is accepted
    And task number <task_number> has estimate optimistic <o> most likely <m> pessimistic <p>
    And task number <task_number> has reasoning <reasoning>

    Examples:
      | task_number | o | m  | p   | reasoning                           |
      | 1           | 3 | 8  | 20  | Team set O M and P together         |
      | 2           | 0 | 1  | 2   | All points minimal for trivial work |
      | 3           | 8 | 40 | 100 | Rewrite pushes every point higher   |

  Scenario Outline: Estimate Override 2
    Given the Tech Lead has generated estimates
    When the Tech Lead overrides task number 1 with optimistic <o> most likely <m> pessimistic <p> and reasoning <reasoning>
    Then the override is rejected because <reason>
    And task number 1 has estimate optimistic 2 most likely 5 pessimistic 13

    Examples:
      | o  | m  | p   | reasoning         | reason                                 |
      | 8  | 5  | 13  | Adjusted for risk | the values are not strictly increasing |
      | 2  | 20 | 13  | Adjusted for risk | the values are not strictly increasing |
      | 5  | 5  | 8   | Adjusted for risk | the values are not strictly increasing |
      | 8  | 13 | 13  | Adjusted for risk | the values are not strictly increasing |
      | 4  | 5  | 13  | Adjusted for risk | the value is off the Fibonacci scale   |
      | 21 | 40 | 100 | Adjusted for risk | the value is off the Fibonacci scale   |
      | -1 | 5  | 13  | Adjusted for risk | the value is off the Fibonacci scale   |
      | 3  | 8  | 20  |                   | the reasoning is empty                 |

  Scenario: Estimate Override 3
    Given the Tech Lead has generated estimates
    When the Tech Lead overrides a task that does not exist with optimistic 1 most likely 2 pessimistic 3 and reasoning New estimate for all points
    Then the task is reported as not found

  Scenario: Estimate Override 4
    When the Tech Lead overrides task number 1 with optimistic 1 most likely 2 pessimistic 3 and reasoning New estimate for all points
    Then the override is rejected because the task has no estimate
    And task number 1 has no estimate
