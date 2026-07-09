# mutation-stamp: sha256=bc3f8a0dacd9e6d01f771af45532f47abd755afe81f4d0cbfbce7c4fb2cf8464
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-09T07:29:18.416944Z","feature_name":"Risk Note Editing","feature_path":"features/risk_note_editing.feature","background_hash":"5d57115545632521057d6f5fdc4e16b1a1f601afbceabc6061217d7c3508913f","implementation_hash":"unknown","scenarios":[{"index":0,"name":"Risk Note Editing 1","scenario_hash":"780c8acb12fac6e8ec148d3f0d165836e312cd38adaed4fb323249bd36b22dc9","mutation_count":8,"result":{"Total":8,"Killed":8,"Survived":0,"Errors":0},"tested_at":"2026-07-08T06:48:37.885513Z"},{"index":1,"name":"Risk Note Editing 2","scenario_hash":"3402252f834c194ab7c41dda6013c0a51274d02fbe64db9cecf0bb086f530ad0","mutation_count":4,"result":{"Total":4,"Killed":4,"Survived":0,"Errors":0},"tested_at":"2026-07-08T06:48:37.885513Z"},{"index":2,"name":"Risk Note Editing 3","scenario_hash":"3fe1dc0a4cd6a755907d33cb7af6ed86dcdd3b8876dd5bf7549eb3215be94bb2","mutation_count":2,"result":{"Total":2,"Killed":2,"Survived":0,"Errors":0},"tested_at":"2026-07-08T06:48:37.885513Z"},{"index":3,"name":"Risk Note Editing 4","scenario_hash":"29fad81a4ccd7612762c99fe07a0191452e3d5596a58afac2e4b3d5268851002","mutation_count":2,"result":{"Total":2,"Killed":2,"Survived":0,"Errors":0},"tested_at":"2026-07-08T06:48:37.885513Z"},{"index":4,"name":"Risk Note Editing 5","scenario_hash":"ded5afc60bae27e51bdb539fec21a2cc2b333a1c43c24e1dbffd5cb3da5f2d3a","mutation_count":2,"result":{"Total":2,"Killed":2,"Survived":0,"Errors":0},"tested_at":"2026-07-08T06:48:37.885513Z"}]}
# acceptance-mutation-manifest-end

Feature: Risk Note Editing

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document
    And the WBS has been approved
    And the AI is primed to flag the risks task 1: SQL injection; task 2: XSS
    And the Tech Lead has flagged risks on the WBS

  Scenario Outline: Risk Note Editing 1
    When the Tech Lead adds a risk note to task number <task_number> with the description Missing rate limits
    Then the risk note is added
    And the risk note count on task number <task_number> is <note_count>
    And risk note <note_position> on task number <task_number> is <note_description>

    Examples:
      | task_number | note_count | note_position | note_description    |
      | 3           | 1          | 1             | Missing rate limits |
      | 1           | 2          | 2             | Missing rate limits |

  Scenario Outline: Risk Note Editing 2
    When the Tech Lead edits risk note 1 on task number <task_number> to the description Reviewed by the Tech Lead
    Then the risk note is updated
    And risk note 1 on task number <task_number> is <new_description>
    And the risk note count on task number <task_number> is 1

    Examples:
      | task_number | new_description           |
      | 1           | Reviewed by the Tech Lead |
      | 2           | Reviewed by the Tech Lead |

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
