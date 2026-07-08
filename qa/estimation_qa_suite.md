# End-to-End QA Suite — Module 3: AI 3-Point Estimation + Approval

Verifies Module 3 **through the HTTP user interface only**, like the Module 1
and 2 suites. QA acts as an ordinary HTTP client and must not call into project
packages. Every expectation is an observable HTTP response or a state observable
by a follow-up `GET /wbs/{id}`.

Mirrors `../features/estimate_generation.feature`,
`../features/estimate_override.feature`,
`../features/estimate_approval.feature`.

Each approved task carries a 3-point Estimate `{ optimistic, mostLikely,
pessimistic, reasoning }`, where the single `reasoning` is one note that
justifies **all three points together** (why the optimistic, most-likely, and
pessimistic values were chosen). Rules for every estimate (AI-suggested and
Tech-Lead override):
- each value must be on the **planning-poker Fibonacci scale**
  `{ 0, 1, 2, 3, 5, 8, 13, 20, 40, 100 }` — note classic Fibonacci numbers not on
  this deck (e.g. `21`, `34`) are off-scale, as are negatives;
- the triple must be **strictly increasing**: `optimistic < mostLikely <
  pessimistic` (equal values are rejected, guaranteeing SD > 0 downstream);
- `reasoning` is a **required non-empty** justification covering all three points.
Values are unitless in this slice (hours-vs-days is presentational and deferred).
The reasoning is bundled with the O/M/P override (there is no standalone
reasoning-edit action); changing it — even with the numbers unchanged — reverts an
approved set to unapproved.

Estimation requires the WBS to be **approved** (risk notes are optional).
Generating replaces all estimates. Approval locks the estimate set; any later
override returns it to unapproved. Re-generation replaces all estimates,
including Tech-Lead overrides.

---

## 1. Preconditions and UI affordances

Start the service in QA mode (`estimationd --ai-provider=mock`), as in Modules
1 and 2.

### 1.1 QA affordance — prime the next estimation
```
POST /qa/ai/next-estimates
Content-Type: application/json

{ "estimates": [
  { "taskNumber": 1, "optimistic": 2, "mostLikely": 5, "pessimistic": 13,
    "reasoning": "Clean best, typical likely, legacy worst" },
  { "taskNumber": 2, "optimistic": 1, "mostLikely": 2, "pessimistic": 3,
    "reasoning": "Trivial across all points" },
  { "taskNumber": 3, "optimistic": 3, "mostLikely": 8, "pessimistic": 20,
    "reasoning": "Legacy widens the range" }
] }
```
- Response `204 No Content`.
- `taskNumber` is the 1-based task position. Sets the O/M/P **and** the single
  reasoning the **next** generation returns; one-shot/FIFO. Enabled only under
  `--ai-provider=mock`. A prime that omits the reasoning leaves the mock to
  supply a non-empty placeholder.

### 1.2 Public contract exercised by QA
| Method | Path | Purpose |
|--------|------|---------|
| POST | `/wbs/{id}/estimates/generate` | Generate O/M/P for every task (replaces all) |
| GET | `/wbs/{id}` | Read the WBS; each task carries `estimate`; WBS carries `estimatesState` |
| PUT | `/wbs/{id}/tasks/{taskId}/estimate` | Override a task's O/M/P |
| POST | `/wbs/{id}/estimates/approve` | Approve the estimate set |

Response shapes:
- `GET /wbs/{id}` → each task `{ "id", "description", "estimate": {
  "optimistic", "mostLikely", "pessimistic", "reasoning" } | null }`, and the
  WBS carries `"estimatesState": "unapproved" | "approved"`.
- `POST …/estimates/generate` success → `200`, every task's `estimate`
  populated per the prime (values + reasoning); `estimatesState` = `"unapproved"`.
- Generate on an **unapproved** WBS → `409`,
  `{ "error": "WBS must be approved before estimation" }`; no estimates set.
- Generate on an unknown WBS → `404`, `{ "error": "WBS not found" }`.
- `PUT …/estimate` body is `{ optimistic, mostLikely, pessimistic, reasoning }`.
- Override success → `200`; sets `estimatesState` back to `"unapproved"`.
- Override with an off-scale value → `422`,
  `{ "error": "estimate value is off the Fibonacci scale" }`.
- Override that is not strictly increasing → `422`,
  `{ "error": "estimate values must be strictly increasing" }`.
- Override with an empty reasoning → `422`, `{ "error": "reasoning is empty" }`.
- Override of a task that has no estimate yet (before generation) → `409`,
  `{ "error": "task has no estimate to override" }`.
- Override on an unknown task/WBS → `404`, `{ "error": "task not found" }`.
- Approve success → `200`; `estimatesState` = `"approved"`.
- Approve before any generation → `409`,
  `{ "error": "estimates have not been generated" }`.

**Resolving "task number N":** `GET` the WBS and use `tasks[N-1].id`.

---

## 2. QA cases — Estimate Generation (`estimate_generation.feature`)

Shared setup: prime tasks `["Login API","Login UI","Session store"]`, generate a
WBS (unapproved, tasks 1..3).

### QA-EST-1 — Generate estimates on an approved WBS
1. Approve the WBS.
2. Prime estimates: task 1 → 2/5/13 "Clean best, typical likely, legacy worst",
   task 2 → 1/2/3 "Trivial across all points", task 3 → 3/8/20 "Legacy widens
   the range".
3. `POST /wbs/{id}/estimates/generate`; expect `200`.
4. `GET` shows each task's `estimate` (values **and** the single all-points
   `reasoning`) matching the prime and `estimatesState == "unapproved"`.

### QA-EST-2 — Reject generation on an unapproved WBS
1. Do **not** approve. Prime any estimates.
2. `POST …/estimates/generate`; expect `409`,
   `{ "error": "WBS must be approved before estimation" }`.
3. `GET` shows every task's `estimate == null`.

### QA-EST-3 — Re-generation replaces all estimates (including overrides)
1. Approve. Prime 2/5/13, 1/2/3, 3/8/20; generate.
2. Override task 1 to 8/8/8 (`PUT …/tasks/{task1Id}/estimate`).
3. Prime 5/13/40, 3/5/8, 8/20/40; generate again.
4. `GET` shows task 1 = 5/13/40, task 2 = 3/5/8, task 3 = 8/20/40 — the override
   is gone.

### QA-EST-4 — Generate on a WBS that does not exist
1. `POST /wbs/does-not-exist/estimates/generate`.
2. Expect `404`, `{ "error": "WBS not found" }`.

---

## 3. QA cases — Estimate Override (`estimate_override.feature`)

Shared setup: the 3-task WBS above, approved, estimates generated to
task 1 = 2/5/13, task 2 = 1/2/3, task 3 = 3/8/20.

### QA-OVR-1 — Accept a valid override
For `(task, O/M/P)` in {(1, 3/8/20), (2, 0/1/2), (3, 8/40/100)} — covering the
`0` lower and `100` upper scale boundaries — each with a non-empty all-points
reasoning:
1. `PUT …/tasks/{taskId}/estimate`
   `{ optimistic, mostLikely, pessimistic, reasoning }`.
2. Expect `200`. `GET` shows that task's estimate updated (values **and**
   reasoning); `estimatesState` becomes `"unapproved"` if it had been approved.

### QA-OVR-2 — Reject an invalid override (isolated single reason per case)
Against task 1 (leaves it 2/5/13 on rejection):
- `8/5/13` → `422` not strictly increasing (O > M).
- `2/20/13` → `422` not strictly increasing (M > P).
- `5/5/8` → `422` not strictly increasing (O = M equality rejected).
- `8/13/13` → `422` not strictly increasing (M = P equality rejected).
- `4/5/13` → `422` off the Fibonacci scale (4 not on the deck).
- `21/40/100` → `422` off the Fibonacci scale (21 is classic Fibonacci but not
  on the planning-poker deck — proves the scale is exactly the deck).
- `-1/5/13` → `422` off the Fibonacci scale (negative).
- `3/8/20` valid values but an **empty** reasoning → `422` reasoning is empty
  (isolates the reasoning rule).
Each case: send the `PUT` (with a valid non-empty reasoning unless it is the
empty-reasoning case), expect the `422` and message, then `GET` and confirm task 1
is still `2/5/13`.

### QA-OVR-3 — Reject overriding a task that does not exist
1. `PUT /wbs/{id}/tasks/nonexistent/estimate` `{ 1, 2, 3 }`.
2. Expect `404`, `{ "error": "task not found" }`.

### QA-OVR-4 — Reject overriding before estimates are generated
1. Fresh approved WBS, estimates **not** generated.
2. `PUT …/tasks/{task1Id}/estimate` `{ 1, 2, 3 }`.
3. Expect `409`, `{ "error": "task has no estimate to override" }`; `GET` shows
   `tasks[0].estimate == null`.

---

## 4. QA cases — Estimate Approval (`estimate_approval.feature`)

Shared setup: the 3-task WBS above, approved, estimates primed (2/5/13, 1/2/3,
3/8/20) but **not** yet generated unless a case generates them.

### QA-APR-1 — Approve generated estimates
1. Generate estimates. `POST …/estimates/approve`; expect `200`.
2. `GET` shows `estimatesState == "approved"`.

### QA-APR-2 — Reject approval before generation
1. Without generating, `POST …/estimates/approve`.
2. Expect `409`, `{ "error": "estimates have not been generated" }`.
3. `GET` shows `estimatesState == "unapproved"`.

### QA-APR-3 — An override after approval reverts to unapproved
1. Generate, then approve (`estimatesState == "approved"`).
2. Override a task with a valid triple + reasoning (e.g. task 1 → 3/8/20
   "Team review"); expect `200`.
3. `GET` shows `estimatesState == "unapproved"`.
4. Also confirm a **reasoning-only** change reverts: on a freshly approved set,
   override task 2 with its existing values `1/2/3` but a new reasoning
   ("Still trivial"); expect `200` and `estimatesState == "unapproved"`.

### QA-APR-4 — Re-approve after an override
1. Generate, approve, override task 1 → 3/8/20 (`estimatesState == "unapproved"`).
2. `POST …/estimates/approve`; expect `200`, `estimatesState == "approved"`.
3. `GET` shows task 1 = 3/8/20.

---

## 5. Notes and assumptions
- Valid estimate values are exactly `{0,1,2,3,5,8,13,20,40,100}`; negatives and
  any other number (4, 9, 10, 21, 34, …) are off-scale and rejected.
- Ordering is strict: `O < M < P`; any equality (`O = M` or `M = P`) is rejected.
- The estimate carries a single reasoning that justifies all three points; it is
  required non-empty. An override supplies (and may change) it alongside O/M/P.
- The override rejection reasons are checked so each QA-OVR-2 case triggers a
  single reason; behavior when a triple is both off-scale and non-increasing is
  not asserted here.
- Estimation requires an approved WBS; risk notes (Module 2) are optional and
  play no gating role.
- The mock provider is the only deterministic path at the UI; against a live LLM
  the suggested values are non-deterministic and only structural checks hold.
- Deleting or editing a task (Module 1) after estimates exist is a cross-module
  interaction assumed but not exercised in this suite.
