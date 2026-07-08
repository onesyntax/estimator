# End-to-End QA Suite — Module 6: Client Proposal & Metrics Translation

Verifies Module 6 **through the HTTP user interface only**, like the Module 1–5
suites. QA acts as an ordinary HTTP client and must not call into project
packages. Every expectation is an observable HTTP response.

Mirrors `../features/proposal.feature`.

Module 6 generates a **client-friendly proposal** from the current estimates plus
three team inputs. It **translates points into time and money** deterministically
(no AI) and is the one view shared with both the Tech Lead and the client, so it
shows a **confidence label**, a **cost range**, a **timeline range**, a **success
probability** at each end of the range with a plain-language **reasoning**, the
detailed **scope**, and **Assumptions & Exclusions** — and never the raw O/M/P,
standard deviation, RSD, or the risk-band flag.

Translation (inputs: `teamVelocity` points/week, `teamCapacity` hours/week,
`hourlyRate` $/hour):
- `hoursPerPoint = teamCapacity / teamVelocity`
- for a points figure `P`: `weeks = P / teamVelocity`, `cost = P × hoursPerPoint × hourlyRate`
- the range runs from the pricing basis to `E_total + 2·SD_total`:
  - low points = `E_total` (green) or `E_total + SD_total` (yellow and red)
  - high points = `E_total + 2·SD_total`
- **cost** is rounded **half-up to whole dollars**; **weeks** are rounded **up** to
  whole weeks; both use full-precision `E_total`/`SD_total` intermediates.

**Success probability** — the chance the project completes at or under a figure,
from a normal curve `N(mean=E_total, SD_total)`: `P(within mean + k·SD) = Φ(k)`.
Because the low end is `mean + k·SD` (k = 0 green, 1 yellow/red) and the high end
is `mean + 2·SD`, the percentages are the deterministic constants **50 / 84 / 98**
(whole-number, half-up): `atLow` is 50 (green) or 84 (yellow/red), `atHigh` is
always 98. The `reasoning` is a **deterministic templated** plain-language string
that weaves in the two percentages and the task count (no AI, no raw stats).

Confidence label from the risk band: green → `high`, yellow → `medium`,
red → `lower`. Red denies a fixed price: `contract == "time-and-materials"` and
`invitesRenegotiation == true`, with the range shown as indicative.

A proposal is a **client deliverable**, so it requires the **estimates to be
approved** (Module 3). The three inputs must be **positive**.

---

## 1. Preconditions and UI affordances

Start the service in QA mode (`estimationd --ai-provider=mock`), as in Modules
1–5.

### 1.1 Public contract exercised by QA
| Method | Path | Purpose |
|--------|------|---------|
| POST | `/wbs/{id}/proposal` | Generate the client proposal from the approved estimates + team inputs |

`POST /wbs/{id}/proposal` request body:
```json
{ "teamVelocity": 3, "teamCapacity": 30, "hourlyRate": 120 }
```
Success → `200` with:
```jsonc
{
  "confidence": "high" | "medium" | "lower",
  "contract": "fixed-price" | "fixed-price-with-buffer" | "time-and-materials",
  "invitesRenegotiation": <bool>,
  "cost": { "low": <int>, "high": <int> },
  "timelineWeeks": { "low": <int>, "high": <int> },
  "successProbability": { "atLow": <int %>, "atHigh": <int %>, "reasoning": "<string>" },
  "scope": [ { "id": "...", "description": "..." }, ... ],
  "assumptionsAndExclusions": [ "<risk note text>", ... ]
}
```
Failures:
- estimates not approved → `409`, `{ "error": "estimates must be approved before a proposal" }`
- a non-positive input → `422`, `{ "error": "team inputs must be positive" }`
- unknown WBS → `404`, `{ "error": "WBS not found" }`

Shared setup: prime tasks `["Login API","Login UI","Session store"]`, generate a
WBS (tasks 1..3), and **approve the WBS**. Estimates are primed/generated/approved
per case.

---

## 2. QA cases — Client Proposal (`proposal.feature`)

### QA-PROP-1 — Medium-risk proposal: ranges, probabilities, reasoning
1. Prime estimates 2/5/13, 1/2/3, 3/8/20; generate; **approve the estimates**.
2. `POST /wbs/{id}/proposal` `{ teamVelocity: 3, teamCapacity: 30, hourlyRate: 120 }`;
   expect `200` and:
   - `confidence == "medium"`, `contract == "fixed-price-with-buffer"`,
     `invitesRenegotiation == false`
   - `cost == { low: 24469, high: 28539 }`
   - `timelineWeeks == { low: 7, high: 8 }`
   - `successProbability.atLow == 84`, `successProbability.atHigh == 98`
   - `successProbability.reasoning` is non-empty and contains `"84%"` and `"3 tasks"`.

### QA-PROP-2 — Low-risk (green) proposal: 50% at the mean
1. Prime estimates 2/3/5, 2/3/5, 2/3/5; generate; **approve the estimates**.
2. `POST /wbs/{id}/proposal` `{ 3, 30, 120 }`; expect `200` and:
   - `confidence == "high"`, `contract == "fixed-price"`
   - `cost == { low: 11400, high: 13478 }`
   - `successProbability == { atLow: 50, atHigh: 98, reasoning: <mentions 50%> }`
     (the low bound is the mean, so 50%).

### QA-PROP-3 — High-risk proposal recommends T&M and invites renegotiation
1. Prime estimates 1/8/40, 1/8/40, 1/8/40; generate; **approve the estimates**.
2. `POST /wbs/{id}/proposal` `{ 3, 30, 120 }`; expect `200` and:
   - `confidence == "lower"`, `contract == "time-and-materials"`,
     `invitesRenegotiation == true`
   - `cost == { low: 57310, high: 70820 }` (indicative expected+1SD .. expected+2SD)
   - `timelineWeeks == { low: 16, high: 20 }`
   - `successProbability.atLow == 84`, `successProbability.atHigh == 98`

### QA-PROP-4 — Scope and Assumptions & Exclusions, without raw internals
1. Flag risks: prime `task 1: SQL injection; task 2: XSS`;
   `POST /wbs/{id}/risk-notes/flag`.
2. Prime estimates 2/5/13, 1/2/3, 3/8/20; generate; **approve the estimates**.
3. `POST /wbs/{id}/proposal` `{ 3, 30, 120 }`; expect `200` and:
   - `scope` contains a task with `description == "Login API"` (the detailed WBS).
   - `assumptionsAndExclusions` contains `"SQL injection"` and `"XSS"`.
   - the response contains **no** `optimistic`/`mostLikely`/`pessimistic`,
     `standardDeviation`, `relativeStandardDeviation`, or `flag` field anywhere.

### QA-PROP-5 — Reject a proposal before estimates are approved
1. Prime estimates 2/5/13, 1/2/3, 3/8/20; generate; do **not** approve them.
2. `POST /wbs/{id}/proposal` `{ 3, 30, 120 }`; expect `409`,
   `{ "error": "estimates must be approved before a proposal" }`.

### QA-PROP-6 — Reject non-positive team inputs
Approved estimates (2/5/13, 1/2/3, 3/8/20). For each input below, `POST` the
proposal and expect `422`, `{ "error": "team inputs must be positive" }`:
- `{ teamVelocity: 0, teamCapacity: 30, hourlyRate: 120 }`
- `{ teamVelocity: 3, teamCapacity: 0, hourlyRate: 120 }`
- `{ teamVelocity: 3, teamCapacity: 30, hourlyRate: 0 }`
- `{ teamVelocity: -3, teamCapacity: 30, hourlyRate: 120 }`

---

## 3. Notes and assumptions
- The proposal is the single client-facing artifact; the raw estimate mechanics
  (O/M/P, SD, RSD) and the risk-band flag are read separately by the Tech Lead
  from `GET /wbs/{id}` (Modules 4–5) and are deliberately absent from the proposal.
- Cost is whole dollars (half-up); timeline weeks are rounded **up** for safety;
  both derive from full-precision `E_total`/`SD_total`.
- The range's lower bound is the pricing basis (expected for green, expected+1SD
  for yellow/red); the upper bound is always expected+2SD. Red still shows the
  band, framed as indicative alongside the T&M recommendation.
- Success probabilities are the deterministic normal-curve constants 50 / 84 / 98
  (`atLow` 50 green or 84 yellow/red; `atHigh` always 98); `reasoning` is a
  deterministic templated sentence, so QA asserts its content, not exact prose.
- Assumptions & Exclusions are a deterministic aggregation of the Module 2 risk
  notes; with no risk notes the list is empty.
- The mock provider is the only deterministic path at the UI; against a live LLM
  the numbers are non-deterministic and only the formula relationships hold.
- A proposal requires approved **estimates**, which in turn require an approved
  WBS; team inputs must be positive because velocity divides.
