Feature: Risk Note Editing

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document
    And the WBS has been approved
    And the AI is primed to flag the risks task 1: SQL injection; task 2: XSS
    And the Tech Lead has flagged risks on the WBS

  Scenario Outline: Risk Note Editing 1
    When the Tech Lead adds a risk note to task number <task_number> with the description <note_description>
    Then the risk note is added
    And the risk note count on task number <task_number> is <note_count>
    And risk note <note_position> on task number <task_number> is <note_description>

    Examples:
      | task_number | note_description    | note_count | note_position |
      | 3           | Missing rate limits | 1          | 1             |
      | 1           | Token leakage       | 2          | 2             |

  Scenario Outline: Risk Note Editing 2
    When the Tech Lead edits risk note 1 on task number <task_number> to the description <new_description>
    Then the risk note is updated
    And risk note 1 on task number <task_number> is <new_description>
    And the risk note count on task number <task_number> is 1

    Examples:
      | task_number | new_description         |
      | 1           | SQL injection via login |
      | 2           | Stored XSS on profile   |

  Scenario Outline: Risk Note Editing 3
    When the Tech Lead deletes risk note 1 on task number <task_number>
    Then the risk note is deleted
    And the risk note count on task number <task_number> is 0

    Examples:
      | task_number |
      | 1           |
      | 2           |

  Scenario Outline: Risk Note Editing 4
    When the Tech Lead <operation> with an empty description
    Then the change is rejected because the description is empty
    And the risk note count on task number 1 is 1

    Examples:
      | operation                          |
      | adds a risk note to task number 1  |
      | edits risk note 1 on task number 1 |

  Scenario Outline: Risk Note Editing 5
    When the Tech Lead <operation> risk note 9 on task number 1
    Then the risk note is reported as not found
    And the risk note count on task number 1 is 1

    Examples:
      | operation |
      | edits     |
      | deletes   |
