Feature: UI Stage 2 Proposal

  # Stage 2 of the pipeline is the client-clean Proposal screen. It is the single
  # shared view: it shows the confidence, contract, cost range, timeline range,
  # success probabilities, scope, and assumptions, and deliberately shows none of
  # the raw mechanics (optimistic/most-likely/pessimistic, standard deviation,
  # relative standard deviation, or the pricing band) that Stage 1 exposes. These
  # scenarios verify what the Proposal screen shows and hides, its inline error
  # surfacing, and that it is unreachable until the estimates are approved; the
  # translation arithmetic itself is pinned by the API-level proposal feature.

  Background:
    Given the estimation app is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated and approved a WBS on the build screen

  # The reveal by risk band: the same team inputs produce a green/yellow/red
  # proposal, and the screen shows the client figures for each. Each row is one
  # band; the on-screen labels, ranges, and probabilities are the oracles.
  Scenario Outline: UI Stage 2 Proposal 1
    Given the AI is primed to estimate <estimates>
    And the Tech Lead has generated and approved estimates on the build screen
    When the Tech Lead requests a proposal with velocity 3 capacity 30 rate 120
    Then the proposal screen shows confidence <confidence>
    And the proposal screen shows contract <contract>
    And the proposal screen shows a cost range from <cost_low> to <cost_high>
    And the proposal screen shows a timeline range from <weeks_low> to <weeks_high> weeks
    And the proposal screen shows success probability <p_low> at the low figure and <p_high> at the high figure

    Examples:
      | estimates                                      | confidence | contract                | cost_low | cost_high | weeks_low | weeks_high | p_low | p_high |
      | task 1: 2/3/5; task 2: 2/3/5; task 3: 2/3/5    | high       | fixed-price             | 11400    | 13478     | 4         | 4          | 50    | 98     |
      | task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20  | medium     | fixed-price-with-buffer | 24469    | 28539     | 7         | 8          | 84    | 98     |
      | task 1: 1/8/40; task 2: 1/8/40; task 3: 1/8/40 | lower      | time-and-materials      | 57310    | 70820     | 16        | 20         | 84    | 98     |

  # The Proposal screen shows the detailed scope and the Assumptions & Exclusions
  # (from the risk notes) and shows none of the internal estimate mechanics.
  Scenario: UI Stage 2 Proposal 2
    Given the AI is primed to flag the risks task 1: SQL injection; task 2: XSS
    And the Tech Lead has flagged risks on the build screen
    And the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated and approved estimates on the build screen
    When the Tech Lead requests a proposal with velocity 3 capacity 30 rate 120
    Then the proposal screen scope includes the task Login API
    And the proposal screen assumptions include SQL injection
    And the proposal screen assumptions include XSS
    And the proposal screen shows no estimate mechanics

  # Non-positive team inputs are refused inline and no proposal is shown. Which
  # input is non-positive varies per row; the request is a literal column so the
  # error and the "no proposal" state are the oracles.
  Scenario Outline: UI Stage 2 Proposal 3
    Given the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated and approved estimates on the build screen
    When the Tech Lead <request>
    Then the proposal screen shows the error team inputs must be positive
    And the proposal screen shows no proposal

    Examples:
      | request                                                   |
      | requests a proposal with velocity 0 capacity 30 rate 120  |
      | requests a proposal with velocity 3 capacity 0 rate 120   |
      | requests a proposal with velocity 3 capacity 30 rate 0    |
      | requests a proposal with velocity -3 capacity 30 rate 120 |

  # The Proposal stage is a client deliverable, so it stays unreachable while the
  # estimates are only generated and not yet approved.
  Scenario: UI Stage 2 Proposal 4
    Given the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates on the build screen
    Then the proposal stage is not reachable
