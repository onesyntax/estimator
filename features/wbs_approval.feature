Feature: WBS Approval

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document

  Scenario: WBS Approval 1
    When the Tech Lead approves the WBS
    Then the WBS is approved
    And the approved task list contains 3 tasks

  Scenario: WBS Approval 2
    Given the Tech Lead has deleted every task in the WBS
    When the Tech Lead approves the WBS
    Then approval is rejected because the WBS has no tasks
    And the WBS is unapproved

  Scenario Outline: WBS Approval 3
    Given the WBS has been approved
    And the WBS is approved
    When the Tech Lead <edit_action>
    Then the change is accepted
    And the WBS is unapproved

    Examples:
      | edit_action                                    |
      | adds a task with the description Audit log      |
      | edits task number 1 to the description SSO API  |
      | deletes task number 2                           |

  Scenario: WBS Approval 4
    Given the WBS has been approved
    And the Tech Lead adds a task with the description Audit log
    And the WBS is unapproved
    When the Tech Lead approves the WBS
    Then the WBS is approved
    And the approved task list contains 4 tasks
