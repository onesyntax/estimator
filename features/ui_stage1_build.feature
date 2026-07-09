# mutation-stamp: sha256=b647de5c497a389c89a6e5e359adf7a8a247371594684269796cc85664476daa
# acceptance-mutation-manifest-begin
# {"version":1,"tested_at":"2026-07-09T07:29:18.485740Z","feature_name":"UI Stage 1 Build","feature_path":"features/ui_stage1_build.feature","background_hash":"8be93b7bdf18c011078ac3dd0a3a981931a2cefd1a78928951727b70d5a407de","implementation_hash":"unknown","scenarios":[{"index":0,"name":"UI Stage 1 Build 1","scenario_hash":"b0f82e8c46c41c0b7d6d81995794bc1c6e81b54ac45770cad48e3f9606353379","mutation_count":2,"result":{"Total":2,"Killed":2,"Survived":0,"Errors":0},"tested_at":"2026-07-09T07:28:54.629548Z"},{"index":1,"name":"UI Stage 1 Build 2","scenario_hash":"ed04349dcc518a95caaeca99a4a5c8de02c8d1c94f97a76379bcae3a46facc51","mutation_count":6,"result":{"Total":6,"Killed":6,"Survived":0,"Errors":0},"tested_at":"2026-07-09T07:28:54.629548Z"},{"index":2,"name":"UI Stage 1 Build 3","scenario_hash":"5bb4f4bca328d756d3e3f235f68c39ff81962d0729a1d6e5994398215872cb56","mutation_count":12,"result":{"Total":12,"Killed":12,"Survived":0,"Errors":0},"tested_at":"2026-07-09T07:28:54.629548Z"},{"index":5,"name":"UI Stage 1 Build 6","scenario_hash":"0f26efc7c25bfd6209d0da40307a607053baa15d391979bd0ef1051bb9894c84","mutation_count":4,"result":{"Total":4,"Killed":4,"Survived":0,"Errors":0},"tested_at":"2026-07-09T07:28:54.629548Z"},{"index":7,"name":"UI Stage 1 Build 8","scenario_hash":"24bb3e2bbb7895a8ea15708ceba2067a7dabf22b5497b7d75f762df0001e6970","mutation_count":30,"result":{"Total":30,"Killed":30,"Survived":0,"Errors":0},"tested_at":"2026-07-09T07:28:54.629548Z"},{"index":9,"name":"UI Stage 1 Build 10","scenario_hash":"aa09926c96b3b861d2086829f8554f327aee84c0f8a9967072609304cfc4ac77","mutation_count":4,"result":{"Total":4,"Killed":4,"Survived":0,"Errors":0},"tested_at":"2026-07-09T07:28:54.629548Z"}]}
# acceptance-mutation-manifest-end

Feature: UI Stage 1 Build

  # Stage 1 of the two-stage estimation UI is a single Build workspace. The Tech
  # Lead enters a requirement, then works the WBS, Risks, and Estimates sections
  # of the same screen. Because it is the internal stage, it shows the raw
  # mechanics (per-task optimistic/most-likely/pessimistic, the project PERT
  # rollup, and the pricing band). The two domain preconditions appear as UI
  # gates: the Risks and Estimates sections are locked until the WBS is approved,
  # and Stage 2 (Proposal) is unreachable until the estimates are approved. These
  # scenarios verify the UI's own behavior — navigation gates, on-screen editing,
  # inline error surfacing, and the transparency boundary — not the domain
  # arithmetic, which the API-level features already pin.

  Background:
    Given the estimation app is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store

  # Entering a requirement (typed or as an uploaded PDF) generates the WBS and
  # renders its tasks in the Build workspace.
  Scenario Outline: UI Stage 1 Build 1
    When a Tech Lead <requirement_action> on the build screen
    Then the build screen shows a WBS with 3 tasks
    And the build screen shows task number 1 as Login API

    Examples:
      | requirement_action                            |
      | enters a valid text requirement and generates |
      | uploads a valid PDF requirement and generates |

  # A requirement the service refuses is surfaced as an inline message on the
  # build screen, and no WBS section appears. The action is a literal column so
  # the message and the "no WBS" state are the oracles.
  Scenario Outline: UI Stage 1 Build 2
    When a Tech Lead <requirement_action> on the build screen
    Then the build screen shows the error <message>
    And the build screen shows no WBS

    Examples:
      | requirement_action                              | message                  |
      | enters an empty text requirement and generates  | requirement is empty     |
      | uploads an empty PDF requirement and generates  | requirement is empty     |
      | uploads a corrupt PDF requirement and generates | document could not be read |

  # The WBS section is editable in place; each edit is reflected on screen. The
  # action is a literal column; the task count and the description of the task
  # each edit changes (the appended task, the edited task, the new first task
  # after a delete) are the oracles.
  Scenario Outline: UI Stage 1 Build 3
    Given a Tech Lead has generated a WBS on the build screen
    When the Tech Lead <edit_action> on the build screen
    Then the build screen shows a WBS with <count> tasks
    And the build screen shows task number <changed_number> as <changed_task>

    Examples:
      | edit_action                                     | count | changed_number | changed_task   |
      | adds a task with the description Password reset  | 4     | 4              | Password reset |
      | edits task number 1 to the description SSO API   | 3     | 1              | SSO API        |
      | deletes task number 1                            | 2     | 1              | Login UI       |

  # The domain gate "risks and estimates require an approved WBS" is realised as a
  # UI lock: both sections are inert until Approve WBS, and available after it.
  Scenario: UI Stage 1 Build 4
    Given a Tech Lead has generated a WBS on the build screen
    Then the risks section is locked
    And the estimates section is locked
    When the Tech Lead approves the WBS on the build screen
    Then the risks section is available
    And the estimates section is available

  # Approving an empty WBS is refused inline and leaves the gate closed.
  Scenario: UI Stage 1 Build 5
    Given a Tech Lead has generated a WBS on the build screen
    When the Tech Lead deletes every task on the build screen
    And the Tech Lead approves the WBS on the build screen
    Then the build screen shows the error cannot approve an empty WBS
    And the risks section is locked

  # Flagging risks renders the AI's notes under their tasks.
  Scenario Outline: UI Stage 1 Build 6
    Given a Tech Lead has generated and approved a WBS on the build screen
    And the AI is primed to flag the risks task 1: SQL injection; task 2: XSS
    When the Tech Lead flags risks on the build screen
    Then task number <task_number> shows the risk note <note>

    Examples:
      | task_number | note          |
      | 1           | SQL injection |
      | 2           | XSS           |

  # Risk notes are editable in place; a manually added note is reflected on screen.
  Scenario: UI Stage 1 Build 7
    Given a Tech Lead has generated and approved a WBS on the build screen
    And the AI is primed to flag the risks task 1: SQL injection
    And the Tech Lead has flagged risks on the build screen
    When the Tech Lead adds a risk note to task number 2 with the description Manual concern on the build screen
    Then task number 2 shows the risk note Manual concern

  # Generating estimates reveals the raw mechanics of the internal stage: each
  # task's 3-point estimate, the project PERT rollup, and the derived pricing
  # band. Each row is one risk band, so the on-screen mechanics are the oracle.
  Scenario Outline: UI Stage 1 Build 8
    Given a Tech Lead has generated and approved a WBS on the build screen
    And the AI is primed to estimate <estimates>
    When the Tech Lead generates estimates on the build screen
    Then the build screen shows task number 1 with estimate optimistic <o> most likely <m> pessimistic <p>
    And the metrics panel shows project expected <e> standard deviation <sd> relative standard deviation <rsd>
    And the pricing panel shows flag <flag> contract <contract> basis <basis>

    Examples:
      | estimates                                      | o | m | p  | e  | sd | rsd | flag   | contract                | basis |
      | task 1: 2/3/5; task 2: 2/3/5; task 3: 2/3/5    | 2 | 3 | 5  | 10 | 1  | 9   | green  | fixed-price             | 10    |
      | task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20  | 2 | 5 | 13 | 17 | 3  | 20  | yellow | fixed-price-with-buffer | 20    |
      | task 1: 1/8/40; task 2: 1/8/40; task 3: 1/8/40 | 1 | 8 | 40 | 37 | 11 | 31  | red    | time-and-materials      | none  |

  # A valid override is applied and shown on screen.
  Scenario: UI Stage 1 Build 9
    Given a Tech Lead has generated and approved a WBS on the build screen
    And the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates on the build screen
    When the Tech Lead overrides task number 1 with optimistic 3 most likely 8 pessimistic 20 and reasoning Team set them on the build screen
    Then the build screen shows task number 1 with estimate optimistic 3 most likely 8 pessimistic 20

  # An invalid override is refused inline and leaves the generated 2/5/13 estimate
  # in place. The action is a literal column; the message and the unchanged
  # estimate are the oracles.
  Scenario Outline: UI Stage 1 Build 10
    Given a Tech Lead has generated and approved a WBS on the build screen
    And the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates on the build screen
    When the Tech Lead <override_action> on the build screen
    Then the build screen shows the error <message>
    And the build screen shows task number 1 with estimate optimistic 2 most likely 5 pessimistic 13

    Examples:
      | override_action                                                                                | message                                     |
      | overrides task number 1 with optimistic 8 most likely 5 pessimistic 13 and reasoning Adjusted   | estimate values must be strictly increasing |
      | overrides task number 1 with optimistic 4 most likely 5 pessimistic 13 and reasoning Adjusted   | estimate value is off the Fibonacci scale   |

  # Approving the estimates is the gate into Stage 2: the Proposal is unreachable
  # before it and reachable after it.
  Scenario: UI Stage 1 Build 11
    Given a Tech Lead has generated and approved a WBS on the build screen
    And the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates on the build screen
    Then the proposal stage is not reachable
    When the Tech Lead approves the estimates on the build screen
    Then the proposal stage is reachable
