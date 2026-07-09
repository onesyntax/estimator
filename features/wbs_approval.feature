# mutation-stamp: sha256=35698d3b5a34c71f98313923e31bbde20de7466ffc7829981cef8c67bb746b20
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-09T07:29:18.618280Z","feature_name":"WBS Approval","feature_path":"features/wbs_approval.feature","background_hash":"98b98f33ccbcaebdc7684ea295ea82abe118d4344be6c097ca261cfb673c54a0","implementation_hash":"unknown","scenarios":[{"index":2,"name":"WBS Approval 3","scenario_hash":"56dec7cab996b58acc3bdfba00ad0a74737d01ca96c37e11ec21a24311af8cb2","mutation_count":12,"result":{"Total":12,"Killed":12,"Survived":0,"Errors":0},"tested_at":"2026-07-07T16:41:35.754085Z"}]}
# acceptance-mutation-manifest-end

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
    And the WBS contains <task_count> tasks
    And task number <check_number> is <check_description>

    Examples:
      | edit_action                                    | task_count | check_number | check_description |
      | adds a task with the description Audit log      | 4          | 4            | Audit log         |
      | edits task number 1 to the description SSO API  | 3          | 1            | SSO API           |
      | deletes task number 2                           | 2          | 2            | Session store     |

  Scenario: WBS Approval 4
    Given the WBS has been approved
    And the Tech Lead adds a task with the description Audit log
    And the WBS is unapproved
    When the Tech Lead approves the WBS
    Then the WBS is approved
    And the approved task list contains 4 tasks
