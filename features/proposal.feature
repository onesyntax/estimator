Feature: Client Proposal

  Background:
    Given the estimation service is running with a deterministic AI provider
    And the AI is primed to generate the tasks Login API; Login UI; Session store
    And a Tech Lead has generated a WBS from a valid requirement document
    And the WBS has been approved

  # The client proposal translates the project points into time and money using
  # the team inputs (velocity points/week, capacity hours/week, hourly rate):
  # hours per point = capacity/velocity, weeks = points/velocity, cost =
  # points*hoursPerPoint*rate. The range runs from the pricing basis (expected
  # for green, expected + 1 SD for yellow/red) up to expected + 2 SD, with cost
  # rounded to whole dollars and weeks rounded up. Each end also carries a
  # success probability: the chance of completing within that figure on a normal
  # curve N(mean=expected, SD). Because the ends are mean+1SD and mean+2SD, the
  # yellow proposal is 84% likely within 24469 and 98% within 28539 over 7 to 8
  # weeks, and a deterministic reasoning explains those figures in plain language.
  Scenario: Client Proposal 1
    Given the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates
    And the estimates have been approved
    When the Tech Lead requests a proposal with velocity 3 capacity 30 rate 120
    Then the proposal confidence is medium
    And the proposal contract is fixed-price-with-buffer
    And the proposal does not invite renegotiation
    And the proposal cost ranges from 24469 to 28539
    And the proposal timeline ranges from 7 to 8 weeks
    And the proposal success probability is 84% at the low figure and 98% at the high figure
    And the proposal reasoning mentions 84%
    And the proposal reasoning mentions 3 tasks

  # A low-risk (green) project quotes the expected value as the low figure, so its
  # low bound sits at the mean — a 50% success probability — while the high bound
  # stays at expected + 2 SD (98%). Confidence is high and the contract is a plain
  # fixed price.
  Scenario: Client Proposal 2
    Given the AI is primed to estimate task 1: 2/3/5; task 2: 2/3/5; task 3: 2/3/5
    And the Tech Lead has generated estimates
    And the estimates have been approved
    When the Tech Lead requests a proposal with velocity 3 capacity 30 rate 120
    Then the proposal confidence is high
    And the proposal contract is fixed-price
    And the proposal cost ranges from 11400 to 13478
    And the proposal success probability is 50% at the low figure and 98% at the high figure

  # A high-risk (red) project denies a fixed price: the proposal recommends Time &
  # Materials and invites renegotiation, showing the same inputs an indicative
  # expected+1SD..expected+2SD band (57310 to 70820 over 16 to 20 weeks) with the
  # matching 84% and 98% success probabilities.
  Scenario: Client Proposal 3
    Given the AI is primed to estimate task 1: 1/8/40; task 2: 1/8/40; task 3: 1/8/40
    And the Tech Lead has generated estimates
    And the estimates have been approved
    When the Tech Lead requests a proposal with velocity 3 capacity 30 rate 120
    Then the proposal confidence is lower
    And the proposal contract is time-and-materials
    And the proposal invites renegotiation
    And the proposal cost ranges from 57310 to 70820
    And the proposal timeline ranges from 16 to 20 weeks
    And the proposal success probability is 84% at the low figure and 98% at the high figure

  # The client-facing proposal shows the detailed scope (the WBS task list) and
  # translates the Module 2 risk notes into the Assumptions & Exclusions, without
  # exposing any raw estimate, standard deviation, or RSD.
  Scenario: Client Proposal 4
    Given the AI is primed to flag the risks task 1: SQL injection; task 2: XSS
    And the Tech Lead has flagged risks on the WBS
    And the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates
    And the estimates have been approved
    When the Tech Lead requests a proposal with velocity 3 capacity 30 rate 120
    Then the proposal scope includes the task Login API
    And the proposal assumptions include SQL injection
    And the proposal assumptions include XSS

  # A proposal is a client deliverable, so it is refused until the Tech Lead has
  # approved the estimate set.
  Scenario: Client Proposal 5
    Given the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates
    When the Tech Lead requests a proposal with velocity 3 capacity 30 rate 120
    Then the proposal is rejected because estimates are not approved

  # The three team inputs must be positive, because velocity divides.
  Scenario Outline: Client Proposal 6
    Given the AI is primed to estimate task 1: 2/5/13; task 2: 1/2/3; task 3: 3/8/20
    And the Tech Lead has generated estimates
    And the estimates have been approved
    When the Tech Lead requests a proposal with velocity <velocity> capacity <capacity> rate <rate>
    Then the proposal is rejected because the inputs must be positive

    Examples:
      | velocity | capacity | rate |
      | 0        | 30       | 120  |
      | 3        | 0        | 120  |
      | 3        | 30       | 0    |
      | -3       | 30       | 120  |
