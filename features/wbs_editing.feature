# mutation-stamp: sha256=b94510dc23c5a1b17dafd33a9d065b403b6e03132d9ea338331b7d6e6eb5d7a9
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-07T16:45:20.936993Z","feature_name":"WBS Editing","feature_path":"features/wbs_editing.feature","background_hash":"98b98f33ccbcaebdc7684ea295ea82abe118d4344be6c097ca261cfb673c54a0","implementation_hash":"unknown","scenarios":[{"index":1,"name":"WBS Editing 2","scenario_hash":"c0e052e92c78afcdb0d3b38c7993d888c5366f7de8322cccba7d606fad094f54","mutation_count":6,"result":{"Total":6,"Killed":6,"Survived":0,"Errors":0},"tested_at":"2026-07-07T16:41:35.879951Z"},{"index":2,"name":"WBS Editing 3","scenario_hash":"e42b73e8737538cebd532021f618986d2b2071382a3c1b112b94fc19da7c05ad","mutation_count":6,"result":{"Total":6,"Killed":6,"Survived":0,"Errors":0},"tested_at":"2026-07-07T16:41:35.879951Z"},{"index":3,"name":"WBS Editing 4","scenario_hash":"1c6e56356ffe92b96fbb54eec75b28b2caab2e6badb519cc3a5b67801100bad6","mutation_count":2,"result":{"Total":2,"Killed":2,"Survived":0,"Errors":0},"tested_at":"2026-07-07T16:32:44.933804Z"},{"index":4,"name":"WBS Editing 5","scenario_hash":"7b7eb5e68e1a08aa78a14f5169adbd09d7fd71e01e72decdf74ec2bd032ab0f5","mutation_count":2,"result":{"Total":2,"Killed":2,"Survived":0,"Errors":0},"tested_at":"2026-07-07T16:32:44.933804Z"}]}
# acceptance-mutation-manifest-end

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
    And task number <task_number> is <expected_description>
    And the WBS contains 3 tasks

    Examples:
      | task_number | new_description     | expected_description |
      | 1           | OAuth login API     | OAuth login API      |
      | 3           | Redis session store | Redis session store  |

  Scenario Outline: WBS Editing 3
    When the Tech Lead deletes task number <task_number>
    Then the task is deleted
    And the WBS contains 2 tasks
    And task number 1 is <remaining_first>
    And task number 2 is <remaining_second>

    Examples:
      | task_number | remaining_first | remaining_second |
      | 2           | Login API       | Session store    |
      | 1           | Login UI        | Session store    |

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
