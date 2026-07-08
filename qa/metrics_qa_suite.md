# End-to-End QA Suite — Module 4: Deterministic PERT Metrics

Verifies Module 4 **through the HTTP user interface only**, like the Module 1–3
suites. QA acts as an ordinary HTTP client and must not call into project
packages. Every expectation is an observable HTTP response or a state observable
by a follow-up `GET /wbs/{id}`.

Mirrors `../features/estimate_metrics.feature`.

Module 4 adds **derived metrics** to the WBS view. They are computed
deterministically by the backend (no AI) from the **current** estimate set and
appear **live** — as soon as estimates exist, whether the set is unapproved or
approved. There is **no new endpoint and no new stored state**: the metrics are
recomputed on every `GET /wbs/{id}`.

For each task that has an estimate `{ optimistic (O), mostLikely (M),
pessimistic (P) }`:
- `expected` (E) = `(O + 4M + P) / 6`
- `standardDeviation` (SD) = `(P − O) / 6`
- `relativeStandardDeviation` (RSD) = `(SD / E) × 100`

Project rollup over the tasks that have an estimate (variances add):
- `expected` = `Σ Eᵢ`
- `standardDeviation` = `√(Σ SDᵢ²)`
- `relativeStandardDeviation` = `SD_total / E_total × 100`

**Every emitted value is a whole number, rounded half-up.** Intermediates and the
rollup use full precision; only the six emitted numbers are rounded. A small task
can therefore show `standardDeviation` 0 while its `relativeStandardDeviation` is
non-zero (they measure absolute vs relative spread). Because `M ≥ 1` from the
strict-increase rule, `E > 0` always, so RSD never divides by zero.

**Controlled Transparency:** this is the internal Tech-Lead interface, so it
exposes all metrics (per-task and project). Hiding raw O/M/P and RSD from the
*client* is Module 6's concern and is not exercised here. Values are unitless in
this slice (hours-vs-days is presentational and deferred).

---

## 1. Preconditions and UI affordances

Start the service in QA mode (`estimationd --ai-provider=mock`), as in Modules
1–3. Module 4 adds **no** endpoints or QA affordances; it reuses the Module 3
affordance (`POST /qa/ai/next-estimates`) and contract (generate / override /
approve). QA drives estimates exactly as in the Module 3 suite and reads the new
metric fields from `GET /wbs/{id}`.

### 1.1 Public contract exercised by QA
| Method | Path | Purpose |
|--------|------|---------|
| POST | `/qa/ai/next-estimates` | Prime the next generation's O/M/P (mock only) |
| POST | `/wbs/{id}/estimates/generate` | Generate O/M/P for every task (replaces all) |
| PUT | `/wbs/{id}/tasks/{taskId}/estimate` | Override a task's O/M/P |
| POST | `/wbs/{id}/estimates/approve` | Approve the estimate set |
| GET | `/wbs/{id}` | Read the WBS; each task carries `metrics`; WBS carries `projectMetrics` |

### 1.2 Response shape added by Module 4
`GET /wbs/{id}` — additive; all Module 1–3 fields unchanged:
- each task gains
  `"metrics": { "expected": <int>, "standardDeviation": <int>, "relativeStandardDeviation": <int> } | null`
  (`null` when the task has no estimate);
- the WBS gains
  `"projectMetrics": { "expected": <int>, "standardDeviation": <int>, "relativeStandardDeviation": <int> } | null`
  (`null` when no task has an estimate).

All six numbers are JSON integers.

**Resolving "task number N":** `GET` the WBS and use `tasks[N-1].id`.

Shared setup for every case below: prime tasks
`["Login API","Login UI","Session store"]`, generate a WBS (tasks 1..3), and
**approve the WBS** (estimation requires an approved WBS). Estimate priming and
generation are per-case.

---

## 2. QA cases — Estimate Metrics (`estimate_metrics.feature`)

### QA-MET-1 — Per-task and project metrics appear live on an unapproved set
1. Prime estimates: task 1 → 2/5/13, task 2 → 1/2/3, task 3 → 3/8/20.
2. `POST /wbs/{id}/estimates/generate`; expect `200`. Do **not** approve.
3. `GET /wbs/{id}`; expect `estimatesState == "unapproved"` and:
   - `tasks[0].metrics == { expected: 6, standardDeviation: 2, relativeStandardDeviation: 31 }`
   - `tasks[1].metrics == { expected: 2, standardDeviation: 0, relativeStandardDeviation: 17 }`
     (SD rounds 0.33 → 0 while RSD is 17)
   - `tasks[2].metrics == { expected: 9, standardDeviation: 3, relativeStandardDeviation: 31 }`
   - `projectMetrics == { expected: 17, standardDeviation: 3, relativeStandardDeviation: 20 }`
     (E = 102/6 = 17; SD = √11.5 = 3.39 → 3; RSD = 19.95 → 20)

### QA-MET-2 — An override recomputes metrics immediately (half-up tie)
1. From QA-MET-1's generated set (still unapproved), override task 1:
   `PUT …/tasks/{task1Id}/estimate`
   `{ optimistic: 5, mostLikely: 8, pessimistic: 13, reasoning: "Manual across all points" }`;
   expect `200`.
2. `GET /wbs/{id}`; expect `estimatesState == "unapproved"` and:
   - `tasks[0].metrics == { expected: 8, standardDeviation: 1, relativeStandardDeviation: 16 }`
   - `projectMetrics == { expected: 20, standardDeviation: 3, relativeStandardDeviation: 16 }`
     — the project expected value is 117/6 = **19.5**, which rounds half-up to **20**.

### QA-MET-3 — Metrics are unchanged by approval
1. Fresh approved WBS; prime 2/5/13, 1/2/3, 3/8/20; generate.
2. `POST …/estimates/approve`; expect `200`.
3. `GET /wbs/{id}`; expect `estimatesState == "approved"` and the **same** metrics
   as QA-MET-1: `tasks[0].metrics == { 6, 2, 31 }`,
   `projectMetrics == { 17, 3, 20 }`. Approval derives nothing new and drops
   nothing.

### QA-MET-4 — No estimates ⇒ no metrics
1. Fresh approved WBS; do **not** generate estimates.
2. `GET /wbs/{id}`; expect every `tasks[i].metrics == null` and
   `projectMetrics == null`.

### QA-MET-5 — Partial estimates: unestimated task has no metrics; rollup covers the rest
1. Fresh approved WBS; prime **only** task 1 → 2/5/13 and task 2 → 1/2/3 (omit
   task 3); generate.
2. `GET /wbs/{id}`; expect:
   - `tasks[2].metrics == null` (task 3 was left unestimated)
   - `tasks[0].metrics == { expected: 6, standardDeviation: 2, relativeStandardDeviation: 31 }`
   - `projectMetrics == { expected: 8, standardDeviation: 2, relativeStandardDeviation: 24 }`
     (rollup over tasks 1–2: E = 47/6 = 7.83 → 8; SD = √(125/36) = 1.86 → 2;
     RSD = 23.79 → 24)

### QA-MET-6 — Rounding half-up and the single-task rollup identity
1. Fresh approved WBS; prime **only** task 1 → 0/1/5; generate.
2. `GET /wbs/{id}`; expect:
   - `tasks[0].metrics == { expected: 2, standardDeviation: 1, relativeStandardDeviation: 56 }`
     (E = 9/6 = **1.5** → 2; SD = 5/6 = 0.83 → 1; RSD = 55.56 → 56)
   - `projectMetrics == { expected: 2, standardDeviation: 1, relativeStandardDeviation: 56 }`
     — with one estimated task the rollup equals that task (`√(SD²) = SD`).

---

## 3. Notes and assumptions
- Metrics are **derived on read**, never stored, and never require their own
  request; they reflect whatever estimates the WBS currently holds.
- They are present for any task that has an estimate regardless of approval
  state, so QA can read them on a generated-but-unapproved set (QA-MET-1) and an
  approved set (QA-MET-3) alike.
- All six emitted numbers are whole (JSON integers), rounded half-up; the
  `standardDeviation 0 / relativeStandardDeviation 17` pairing on a small task
  (QA-MET-1, task 2) is intended, not a defect.
- The project rollup adds variances (`√Σσ²`), so `projectMetrics.standardDeviation`
  is at most the sum of the task SDs and equals a lone task's SD when only one
  task is estimated (QA-MET-6).
- The mock provider is the only deterministic path at the UI; against a live LLM
  the estimate values are non-deterministic, so only the formula relationships
  (e.g. `O < expected < P`, integer outputs) hold, not the exact numbers.
- Estimation still requires an approved WBS (Module 3); risk notes (Module 2) are
  optional and play no role in the metrics.
