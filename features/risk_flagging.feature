Feature: Risk Flagging

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document

  Scenario Outline: Risk Flagging 1
    Given the WBS has been approved
    And the AI is primed to flag the risks task 1: SQL injection; task 1: Rate limiting; task 2: XSS
    When the Tech Lead flags risks on the WBS
    Then the risk note count on task number <task_number> is <note_count>

    Examples:
      | task_number | note_count |
      | 1           | 2          |
      | 2           | 1          |
      | 3           | 0          |

  Scenario Outline: Risk Flagging 2
    Given the WBS has been approved
    And the AI is primed to flag the risks task 1: SQL injection; task 1: Rate limiting; task 2: XSS
    When the Tech Lead flags risks on the WBS
    Then risk note <note_number> on task number <task_number> is <note_description>

    Examples:
      | task_number | note_number | note_description |
      | 1           | 1           | SQL injection    |
      | 1           | 2           | Rate limiting    |
      | 2           | 1           | XSS              |

  Scenario: Risk Flagging 3
    Given the AI is primed to flag the risks task 1: SQL injection
    When the Tech Lead flags risks on the WBS
    Then risk flagging is rejected because the WBS is unapproved
    And the risk note count on task number 1 is 0

  Scenario Outline: Risk Flagging 4
    Given the WBS has been approved
    And the AI is primed to flag the risks task 1: SQL injection; task 2: XSS
    And the Tech Lead has flagged risks on the WBS
    And the Tech Lead adds a risk note to task number 3 with the description Manual concern
    And the AI is primed to flag the risks task 1: CSRF
    When the Tech Lead flags risks on the WBS
    Then the risk note count on task number <task_number> is <note_count>

    Examples:
      | task_number | note_count |
      | 1           | 1          |
      | 2           | 0          |
      | 3           | 0          |

  Scenario: Risk Flagging 5
    When a Tech Lead flags risks on a WBS that does not exist
    Then the WBS is reported as not found
