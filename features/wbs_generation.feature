# mutation-stamp: sha256=0ed331bcc235b4b503f35eed91baee068ea2aafe8db329fd9def3c3d9673fce1
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-07T16:45:20.993177Z","feature_name":"WBS Generation","feature_path":"features/wbs_generation.feature","background_hash":"4fd2f172057527235849ab89f08b8590ae19003106a66e89a18da19110e1b98a","implementation_hash":"unknown","scenarios":[{"index":0,"name":"WBS Generation 1","scenario_hash":"4f36c79366047a01c52c164a93baf1f8f88ec8afb15442d533ef0b6a793bc78a","mutation_count":8,"result":{"Total":8,"Killed":8,"Survived":0,"Errors":0},"tested_at":"2026-07-07T16:32:44.979046Z"},{"index":1,"name":"WBS Generation 2","scenario_hash":"b6f24b62efe58608fe1209e25716086c317aa0bc4fe579d2ea483c1b85546d26","mutation_count":2,"result":{"Total":2,"Killed":2,"Survived":0,"Errors":0},"tested_at":"2026-07-07T16:32:44.979046Z"}]}
# acceptance-mutation-manifest-end

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
