# End-to-End QA Suite — Two-Stage Estimation UI

This suite verifies the two-stage estimation UI **through the user interface
only** — i.e. through the app's screens in a browser. QA acts as an ordinary
user: it navigates the app, types into fields, clicks controls, and asserts on
**what the screen shows**. It must **not** call the project HTTP API, project
packages, or any internal interface. The only non-user affordance QA uses is the
QA-only priming surface described in §1.2, which exists solely to make the AI
deterministic and is present only in mock mode.

The suite mirrors the acceptance features
`../features/ui_stage1_build.feature` and
`../features/ui_stage2_proposal.feature`. It is one pipeline: Stage 1 (Build) in
§2 flows into Stage 2 (Proposal) in §3.

---

## 1. Preconditions and UI affordances

### 1.1 Launching the app in QA mode
Start the service so the AI provider is the deterministic mock and the QA
priming surface is enabled, then open the app in a browser:

```
estimationd --ai-provider=mock --addr=:8080
```

- QA opens `http://127.0.0.1:8080/` and interacts with the rendered app.
- When `--ai-provider=mock` is set, the QA priming surface (§1.2) is available.
- When it is not set (production), the priming surface is **absent** and QA
  cannot make the AI deterministic — so this suite runs only in mock mode.

### 1.2 QA affordance — priming the AI through the UI
In mock mode the app exposes a **QA-only priming surface** (a panel/screen that
is present only in mock mode and never in production). Through it QA seeds, as a
user action, what the AI will return **next**:

- **Next WBS** — an ordered list of task descriptions the next *Generate*
  returns (e.g. `Login API; Login UI; Session store`).
- **Next risks** — task-number → risk description pairs the next *Flag risks*
  returns (e.g. `task 1: SQL injection; task 2: XSS`).
- **Next estimates** — task-number → `optimistic/most-likely/pessimistic` triples
  the next *Generate estimates* returns (e.g. `task 1: 2/5/13; task 2: 1/2/3;
  task 3: 3/8/20`).

Priming is **one-shot and FIFO**: each seed is consumed by the next matching
action, keeping cases isolated. QA primes immediately before the action it
feeds. All other interactions in this suite are ordinary app controls.

### 1.3 Reading the screen
- **"Task number N"** is the Nth task shown, top to bottom, in the WBS/estimates
  list. QA edits/overrides the task by acting on that row.
- Every expectation below is text or state **visible on the screen** (a label, a
  value, a section being enabled/disabled, an inline error, or the presence or
  absence of a screen). QA never inspects a network response body.
- **Inline errors** appear on the relevant section of the current screen; QA
  asserts the message text and that the screen did not advance.

### 1.4 Shared Stage-1 setup
Unless a case says otherwise, "**generate a WBS**" means: prime Next WBS
`Login API; Login UI; Session store`, type a non-empty requirement in the
Requirement field, and activate *Generate*. The Build screen then shows a
3-task WBS (Login API, Login UI, Session store), unapproved.

---

## 2. Stage 1 — Build (`ui_stage1_build.feature`)

### QA-UI-1 — Requirement to WBS on screen (typed and PDF)
For `input` in {typed text, uploaded PDF}:
1. Prime Next WBS `Login API; Login UI; Session store`.
2. Provide the requirement (type text, or upload a valid PDF whose text is a
   requirement) and activate *Generate*.
3. Expect the Build screen to show a WBS with **3 tasks** and **task 1 =
   "Login API"**.

### QA-UI-2 — A bad requirement shows an inline error and no WBS
For each `(input, message)`:
- empty typed text → `requirement is empty`
- uploaded PDF with no extractable text → `requirement is empty`
- uploaded corrupt PDF (bytes that are not a valid PDF) → `document could not be read`
Steps: provide the `input`, activate *Generate*, expect the inline `message` on
the Requirement section, and **no WBS section shown**.

### QA-UI-3 — Editing the WBS is reflected on screen
Generate a WBS, then for each `(edit, count, first_task)` (each on a freshly
generated WBS):
- add a task "Password reset" → 4 tasks, task 1 = "Login API"
- edit task 1 to "SSO API" → 3 tasks, task 1 = "SSO API"
- delete task 1 → 2 tasks, task 1 = "Login UI"
Perform the edit with the on-screen controls and assert the WBS list shows
`count` tasks and task 1 = `first_task`.

### QA-UI-4 — Risks and Estimates are locked until the WBS is approved
1. Generate a WBS.
2. Expect the **Risks** and **Estimates** sections to be **locked/disabled**
   (their actions cannot be used).
3. Activate *Approve WBS*.
4. Expect both sections to become **available**.

### QA-UI-5 — Approving an empty WBS is refused inline
1. Generate a WBS, then delete all three tasks using the on-screen delete
   controls.
2. Activate *Approve WBS*.
3. Expect the inline error `cannot approve an empty WBS`, and the Risks section
   still **locked**.

### QA-UI-6 — Flagging risks renders notes under their tasks
1. Generate a WBS and *Approve WBS*.
2. Prime Next risks `task 1: SQL injection; task 2: XSS`; activate *Flag risks*.
3. Expect **task 1** to show the note "SQL injection" and **task 2** to show
   "XSS".

### QA-UI-7 — Risk notes are editable on screen
1. Generate a WBS and *Approve WBS*; prime `task 1: SQL injection`; *Flag risks*.
2. Add a risk note "Manual concern" to **task 2** via the on-screen control.
3. Expect task 2 to show the note "Manual concern".

### QA-UI-8 — Estimates reveal the internal mechanics (per band)
For each band (estimates → task-1 O/M/P; project E/SD/RSD; flag, contract, basis):
- `2/3/5` ×3 → task 1 **2/3/5**; project **E 10, SD 1, RSD 9**; **green**,
  `fixed-price`, basis **10**
- `2/5/13; 1/2/3; 3/8/20` → task 1 **2/5/13**; project **E 17, SD 3, RSD 20**;
  **yellow**, `fixed-price-with-buffer`, basis **20**
- `1/8/40` ×3 → task 1 **1/8/40**; project **E 37, SD 11, RSD 31**; **red**,
  `time-and-materials`, basis **none**
Steps: generate a WBS and *Approve WBS*; prime Next estimates; activate
*Generate estimates*. Expect the screen to show task 1's O/M/P, the metrics
panel's project expected/SD/RSD, and the pricing panel's flag/contract/basis as
above. (This is the transparency boundary: these mechanics are visible in
Stage 1.)

### QA-UI-9 — A valid estimate override is applied on screen
1. Generate a WBS, *Approve WBS*, prime `2/5/13; 1/2/3; 3/8/20`, *Generate
   estimates*.
2. Override task 1 to optimistic 3 / most likely 8 / pessimistic 20 with a
   non-empty reasoning, via the on-screen override control.
3. Expect task 1 to show the estimate **3 / 8 / 20**.

### QA-UI-10 — An invalid override is refused inline, estimate unchanged
Setup as QA-UI-9 (task 1 generated as 2/5/13). For each `(override, message)`:
- optimistic 8 / most likely 5 / pessimistic 13 → `estimate values must be strictly increasing`
- optimistic 4 / most likely 5 / pessimistic 13 → `estimate value is off the Fibonacci scale`
Attempt the override; expect the inline `message` and task 1 still showing
**2 / 5 / 13**.

### QA-UI-11 — Approving the estimates unlocks Stage 2
1. Generate a WBS, *Approve WBS*, prime `2/5/13; 1/2/3; 3/8/20`, *Generate
   estimates*.
2. Expect the **Proposal** stage to be **not reachable** (its control is
   disabled / the screen cannot be opened).
3. Activate *Approve estimates*.
4. Expect the **Proposal** stage to be **reachable**.

---

## 3. Stage 2 — Proposal (`ui_stage2_proposal.feature`)

**Setup for every proposal case:** generate a WBS and *Approve WBS* (§1.4),
then prime/generate the case's estimates and *Approve estimates*, and open the
Proposal stage. Team inputs are entered on the Proposal screen as
velocity / capacity / rate.

### QA-UI-12 — The proposal reveal, per risk band
For each band, prime the estimates, approve them, open the Proposal screen,
enter velocity **3** / capacity **30** / rate **120**, and activate *Generate*:

| estimates | confidence | contract | cost range | timeline | success (low/high) |
|---|---|---|---|---|---|
| `2/3/5` ×3 | high | fixed-price | 11400 – 13478 | 4 – 4 weeks | 50% / 98% |
| `2/5/13; 1/2/3; 3/8/20` | medium | fixed-price-with-buffer | 24469 – 28539 | 7 – 8 weeks | 84% / 98% |
| `1/8/40` ×3 | lower | time-and-materials | 57310 – 70820 | 16 – 20 weeks | 84% / 98% |

Expect the Proposal screen to show the confidence label, contract, cost range,
timeline range in weeks, and the two success probabilities as above.

### QA-UI-13 — The proposal shows scope and assumptions, and hides mechanics
1. Prime risks `task 1: SQL injection; task 2: XSS`; *Flag risks*.
2. Prime `2/5/13; 1/2/3; 3/8/20`; *Generate estimates*; *Approve estimates*.
3. Open the Proposal screen, enter 3 / 30 / 120, *Generate*.
4. Expect the **Scope** to include the task "Login API", the **Assumptions &
   Exclusions** to include "SQL injection" and "XSS", and the screen to show
   **no** optimistic/most-likely/pessimistic values, **no** standard deviation,
   **no** relative standard deviation, and **no** pricing band/flag anywhere.

### QA-UI-14 — Non-positive team inputs are refused inline
Reach an approved-estimates Proposal screen (estimates `2/5/13; 1/2/3; 3/8/20`).
For each input set, enter it and activate *Generate*; expect the inline error
`team inputs must be positive` and **no proposal shown**:
- velocity 0 / capacity 30 / rate 120
- velocity 3 / capacity 0 / rate 120
- velocity 3 / capacity 30 / rate 0
- velocity -3 / capacity 30 / rate 120

### QA-UI-15 — The Proposal stage is unreachable before estimates are approved
1. Generate a WBS, *Approve WBS*, prime `2/5/13; 1/2/3; 3/8/20`, *Generate
   estimates* — but do **not** approve them.
2. Expect the **Proposal** stage to be **not reachable**.

---

## 4. Notes and assumptions
- **Single active estimate per session.** The app holds one WBS in memory; there
  is no saved-project library. Each case starts from a freshly generated WBS.
- **Determinism.** The mock provider + the §1.2 priming surface are the only way
  to make the AI deterministic at the UI. Against a live LLM the task text,
  estimates, and derived figures are non-deterministic, and only structural
  checks (a WBS appears, a proposal appears, the gates hold, mechanics are
  hidden in Stage 2) would hold.
- **Transparency boundary.** Stage 1 shows the raw mechanics (O/M/P, project
  PERT rollup, pricing band); the Stage 2 Proposal is the single client-clean
  view and shows none of them. QA-UI-8 asserts they are present in Stage 1 and
  QA-UI-13 asserts they are absent in Stage 2.
- **Gates.** Risks/Estimates are locked until *Approve WBS*; the Proposal stage
  is unreachable until *Approve estimates*. Under normal navigation QA never
  reaches the underlying "must be approved" server errors, because the UI
  withholds the action until its gate is passed.
- **Numbers.** All figures are the deterministic values the API-level features
  pin (`estimate_metrics`, `pricing_strategy`, `proposal`); this suite asserts
  the UI *renders* them, not the arithmetic that produces them.
