# mutation-stamp: sha256=2960e29611c7c0e61650c073a0dd9697cf6e836088cdf09ebc5b03842391cfca
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-08T18:33:53.283848Z","feature_name":"Risk Flagging","feature_path":"features/risk_flagging.feature","background_hash":"98b98f33ccbcaebdc7684ea295ea82abe118d4344be6c097ca261cfb673c54a0","implementation_hash":"unknown","scenarios":[{"index":0,"name":"Risk Flagging 1","scenario_hash":"6fa76112caa99ca010785ba9c802510d36f14f299ecf98a1bccbfd50679cdb60","mutation_count":6,"result":{"Total":6,"Killed":6,"Survived":0,"Errors":0},"tested_at":"2026-07-08T06:43:20.449780Z"},{"index":1,"name":"Risk Flagging 2","scenario_hash":"09f57a3ae0a58944f48cb1e6e23929e981f039f1658cf679bd3dfbe2cb577471","mutation_count":9,"result":{"Total":9,"Killed":9,"Survived":0,"Errors":0},"tested_at":"2026-07-08T06:43:20.449780Z"},{"index":3,"name":"Risk Flagging 4","scenario_hash":"a640aeccb7dded6f70a5ab76f14abb1a56b8996669968ed6d2c408a7f2431418","mutation_count":6,"result":{"Total":6,"Killed":6,"Survived":0,"Errors":0},"tested_at":"2026-07-08T06:43:20.449780Z"}]}
# acceptance-mutation-manifest-end

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
