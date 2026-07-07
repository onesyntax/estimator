Feature: WBS Editing

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document

  Scenario: WBS Editing 1
    When the Tech Lead adds a task with the description Password reset
    Then the task is added
    And the WBS contains 4 tasks
    And task number 4 is Password reset

  Scenario Outline: WBS Editing 2
    When the Tech Lead edits task number <task_number> to the description <new_description>
    Then the task is updated
    And task number <task_number> is <new_description>
    And the WBS contains 3 tasks

    Examples:
      | task_number | new_description     |
      | 1           | OAuth login API     |
      | 3           | Redis session store |

  Scenario Outline: WBS Editing 3
    When the Tech Lead deletes task number <task_number>
    Then the task is deleted
    And the WBS contains 2 tasks
    And no task is <deleted_description>

    Examples:
      | task_number | deleted_description |
      | 2           | Login UI            |
      | 1           | Login API           |

  Scenario Outline: WBS Editing 4
    When the Tech Lead <operation> with an empty description
    Then the change is rejected because the description is empty
    And the WBS contains 3 tasks

    Examples:
      | operation           |
      | adds a task         |
      | edits task number 2 |

  Scenario Outline: WBS Editing 5
    When the Tech Lead <operation> a task that does not exist
    Then the task is reported as not found
    And the WBS contains 3 tasks

    Examples:
      | operation |
      | edits     |
      | deletes   |
