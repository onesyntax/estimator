# End-to-End QA Suite — Module 5: Pricing Strategy Decision Engine

Verifies Module 5 **through the HTTP user interface only**, like the Module 1–4
suites. QA acts as an ordinary HTTP client and must not call into project
packages. Every expectation is an observable HTTP response or a state observable
by a follow-up `GET /wbs/{id}`.

Mirrors `../features/pricing_strategy.feature`.

Module 5 adds a **derived pricing strategy** to the WBS view. It is computed
deterministically (no AI) from the **project RSD** Module 4 emits (the whole-number
`projectMetrics.relativeStandardDeviation`), and appears **live** — as soon as
estimates exist, whether the set is unapproved or approved. There is **no new
endpoint and no new stored state**: it is recomputed on every `GET /wbs/{id}`.

Risk bands (integer RSD):

| RSD | flag | riskLevel | contract | recommendedBasis |
|-----|------|-----------|----------|------------------|
| `< 10` | green | low | fixed-price | `E_total` |
| `10 … 20` (inclusive) | yellow | medium | fixed-price-with-buffer | `E_total + SD_total` |
| `> 20` | red | high | time-and-materials | `null` (no fixed price) |

`recommendedBasis` is the whole-number **points** figure a quote should be built
on — expected for green, expected + 1 SD (the risk buffer) for yellow, and `null`
for red where a fixed price is refused. It is computed from full-precision
`E_total`/`SD_total` and rounded half-up, consistent with Module 4.

**Controlled Transparency:** this is the internal Tech-Lead interface, so it
exposes the flag, level, contract, and basis. The client never sees this block or
the raw RSD; the client-facing confidence label and price range are Module 6.

---

## 1. Preconditions and UI affordances

Start the service in QA mode (`estimationd --ai-provider=mock`), as in Modules
1–4. Module 5 adds **no** endpoints or QA affordances; it reuses the Module 3
estimate affordances and reads the new block from `GET /wbs/{id}`.

### 1.1 Response shape added by Module 5
`GET /wbs/{id}` — additive; all Module 1–4 fields unchanged:
```jsonc
"pricingStrategy": {
  "flag": "green" | "yellow" | "red",
  "riskLevel": "low" | "medium" | "high",
  "contract": "fixed-price" | "fixed-price-with-buffer" | "time-and-materials",
  "recommendedBasis": <int> | null
} | null   // null when there are no project metrics (no estimates)
```

Shared setup for every case: prime tasks `["Login API","Login UI","Session store"]`,
generate a WBS (tasks 1..3), and **approve the WBS**.

---

## 2. QA cases — Pricing Strategy (`pricing_strategy.feature`)

### QA-PRC-1 — The three risk bands map to the right strategy (live, unapproved)
For each estimate set below: prime the three tasks with those O/M/P values,
`POST /wbs/{id}/estimates/generate` (do **not** approve), then `GET /wbs/{id}` and
assert `estimatesState == "unapproved"` and `pricingStrategy` equals the expected
row.

| Estimate set (all 3 tasks) | project RSD | pricingStrategy |
|----------------------------|-------------|-----------------|
| `2/3/5, 2/3/5, 2/3/5` | 9 | `{ green, low, fixed-price, recommendedBasis: 10 }` |
| `2/5/13, 1/2/3, 3/8/20` (standard) | 20 | `{ yellow, medium, fixed-price-with-buffer, recommendedBasis: 20 }` |
| `1/8/40, 1/8/40, 1/8/40` | 31 | `{ red, high, time-and-materials, recommendedBasis: null }` |

The standard set lands at RSD **exactly 20**, confirming 20 is inside the yellow
band (the upper boundary is inclusive).

### QA-PRC-2 — No estimates ⇒ no pricing strategy
1. Fresh approved WBS; do **not** generate estimates.
2. `GET /wbs/{id}`; expect `pricingStrategy == null` (and `projectMetrics == null`).

---

## 3. Notes and assumptions
- The strategy is **derived on read** from the emitted integer project RSD, never
  stored, and reflects whatever estimates the WBS currently holds — so QA can read
  it on a generated-but-unapproved set.
- Boundaries are inclusive at the yellow edges: RSD 9 → green, 10 → yellow,
  20 → yellow, 21 → red.
- `recommendedBasis` is a **points** figure (unitless), not a currency; Module 6
  translates it into time and money.
- The mock provider is the only deterministic path at the UI; against a live LLM
  the estimate values are non-deterministic, so only the band relationships hold.
- Estimation still requires an approved WBS (Module 3); risk notes (Module 2) are
  optional and play no role in the pricing decision.
