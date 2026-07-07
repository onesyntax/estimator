Feature: WBS Generation

  Background:
    Given the estimation service is running with a deterministic AI provider

  Scenario Outline: WBS Generation 1
    Given the AI is primed to generate the tasks <primed_tasks>
    When a Tech Lead submits a valid <input_format> requirement document
    Then a new WBS is created
    And the WBS is unapproved
    And the WBS contains <task_count> tasks
    And task number 1 is <first_task>

    Examples:
      | input_format | primed_tasks                       | task_count | first_task |
      | text         | Login API; Login UI; Session store | 3          | Login API  |
      | pdf          | Charge API; Refund API             | 2          | Charge API |

  Scenario Outline: WBS Generation 2
    When a Tech Lead submits an empty <input_format> requirement document
    Then the submission is rejected because the requirement is empty
    And no WBS is created

    Examples:
      | input_format |
      | text         |
      | pdf          |

  Scenario: WBS Generation 3
    When a Tech Lead submits a corrupt PDF requirement document
    Then the submission is rejected because the document cannot be read
    And no WBS is created

  Scenario: WBS Generation 4
    When a Tech Lead requests a WBS that does not exist
    Then the WBS is reported as not found
