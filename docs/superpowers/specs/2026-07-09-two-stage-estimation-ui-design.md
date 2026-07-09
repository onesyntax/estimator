# Two-Stage Estimation UI — Design

- **Date:** 2026-07-09
- **Author:** Specifier (with Kalpa Perera)
- **Status:** Approved design; specifier deliverables (Gherkin + end-to-end QA suite) to follow.

## Context

`estimation` is a headless Go service (`estimationd`) that exposes the WBS
estimation engine over an HTTP/JSON API. There is no frontend today. A Tech
Lead currently drives the whole workflow — generate a WBS from a requirement,
edit it, approve it, flag risks, generate and override 3-point estimates,
review PERT metrics and the pricing strategy, and produce a client proposal —
entirely through raw API calls.

This design introduces a **user-friendly, modern two-stage UI** that is a single
continuous pipeline over the existing service. It is *not* two separate
products (no distinct "internal tool" and "client tool"); it is one flow that
ends in the single client-facing proposal.

## Goals

- One pipeline, two stages: an internal **Build workspace** and a client-clean
  **Proposal**.
- Expose the full editing surface the API already offers (add/edit/delete
  tasks, add/edit/delete risk notes, override estimates).
- Honor the transparency model: raw estimate mechanics (Optimistic/Most-likely/
  Pessimistic, standard deviation, relative standard deviation, pricing
  flag/contract/basis) are visible only in the Build stage. The Proposal is the
  single shared client-friendly view and hides all of it.
- Respect the domain's real preconditions as UI gates (WBS approval before
  risks/estimates; estimate approval before the proposal).
- Be deterministically testable end-to-end through the UI (not via the domain
  API) by QA.

## Non-goals (v1)

- No saved-project library / multi-project management. A single active estimate
  exists per session, matching the in-memory backend.
- No authentication, roles, or multi-user concurrency beyond what the service
  already serializes.
- No new domain behavior. The UI is an adapter over the existing API surface;
  it introduces no estimation logic of its own.
- This document specifies **behavior**, not visual implementation (framework,
  styling, layout pixels). Those are the coder's choices.

## The pipeline

```
STAGE 1 — Build workspace (one screen, mechanics visible)   STAGE 2 — Proposal (client-clean)
┌───────────────────────────────────────────┐             ┌──────────────────────────┐
│ Requirement  [paste text / upload PDF] [→] │             │ Team inputs:             │
│ ── WBS ───────────────────────────────────  │             │  velocity / capacity /   │
│   tasks: add / edit / delete   [Approve WBS] │     →       │  hourly rate  [Generate] │
│ ── Risks ── (enabled after WBS approved) ──  │             │ ── Proposal ───────────  │
│   [Flag risks]  notes: add / edit / delete   │             │  cost range (low–high)   │
│ ── Estimates ── (enabled after WBS approved) │             │  timeline range (weeks)  │
│   [Generate]  O/M/P + reasoning + override   │  ┌────────┐ │  confidence, contract    │
│                                              │  │ metrics│ │  success probability     │
│              [Approve estimates] ───────────►│  │ pricing│ │  scope + assumptions     │
│                                              │  └────────┘ │  (NO O/M/P, SD, RSD, no  │
└───────────────────────────────────────────┘             │   pricing flag)          │
                                                            └──────────────────────────┘
```

Two stages, one linear pipeline. Stage 2 is reachable only after estimates are
approved. Within Stage 1, sections enable progressively per the domain gates.

## Stage 1 — Build workspace

A single screen. The Requirement input sits at the top; the WBS, Risks, and
Estimates sections appear below it in the same view.

### Requirement (entry)

- Input is either **pasted text** or an **uploaded PDF** (the API accepts both:
  a JSON `{requirement}` body or a multipart `document` part).
- Submitting generates the WBS via the AI provider and populates the WBS
  section below.
- Empty requirement text → inline error "requirement is empty".
- Unreadable PDF → inline error "document could not be read".

### WBS section

- Shows the generated task list. Each task has a stable id and a description.
- Editing: **add** a task, **edit** a task description, **delete** a task.
- Primary action **Approve WBS**. Approving an empty WBS is blocked with an
  inline error "cannot approve an empty WBS".
- Approving the WBS is a commit point: it locks the requirement + task list and
  enables the Risks and Estimates sections.

### Risks section (enabled after WBS approved)

- **Flag risks** invokes the AI to attach risk notes to tasks.
- Editing: **add**, **edit**, and **delete** risk notes on any task.
- Risks are advisory; the user may proceed to estimates without flagging.
- Attempting risk work before the WBS is approved is not offered by the UI; the
  underlying guard is "WBS must be approved before risk flagging".

### Estimates section (enabled after WBS approved)

- **Generate estimates** invokes the AI to produce a 3-point estimate
  (Optimistic / Most-likely / Pessimistic) plus reasoning per task.
- Each task shows its O/M/P, reasoning, and its derived per-task PERT metrics.
- **Override**: the user may replace any task's estimate. Overrides are
  validated by the domain:
  - values must be on the Fibonacci scale → else "estimate value is off the
    Fibonacci scale";
  - values must be strictly increasing (O < M < P) → else "estimate values must
    be strictly increasing";
  - reasoning must be non-empty → else "reasoning is empty".
- A metrics/pricing panel shows the **project rollup**: expected, standard
  deviation, relative standard deviation (RSD), and the derived **pricing
  strategy**: risk band (green/yellow/red), risk level, contract type, and the
  recommended basis. This panel is present as soon as estimates exist (pricing
  is derived live from project metrics, before approval).
- Primary action **Approve estimates**. This is the gate into Stage 2.

### Observable Stage-1 state

The two approval flags — **WBS approved** and **estimates approved** — plus
whether estimates have been generated drive which sections and actions are
enabled. These are the observable states the specs assert against.

## Stage 2 — Proposal (client-clean)

A single screen, reachable only after estimates are approved.

- Inputs: **team velocity** (points/week), **team capacity** (hours/week),
  **hourly rate** ($/hour). All three must be positive → else "team inputs must
  be positive".
- Output — the single shared client-friendly proposal:
  - **Confidence** (high / medium / lower).
  - **Contract** type and whether it **invites renegotiation**.
  - **Cost range** (low–high dollars) and **timeline range** (low–high weeks).
  - **Success probability** at the low and high figures, with plain-language
    reasoning.
  - **Scope** — the task list.
  - **Assumptions & Exclusions** — derived from the risk notes.
- The proposal shows **none** of the raw mechanics: no O/M/P, no standard
  deviation, no RSD, no pricing-band flag. (These remain visible only in the
  Stage-1 Estimates section.)

The proposal's figures follow the existing domain math (pricing basis by risk
band, cost/timeline derivation from team inputs, success probability on the
normal curve). The UI renders what the domain returns; it does not recompute.

## Error handling

Every domain error is surfaced as a **friendly inline message on the relevant
step**, never as a raw HTTP status or JSON error. Mapped messages reuse the
service's documented text:

| Situation | Message |
|---|---|
| Empty requirement | requirement is empty |
| Unreadable PDF | document could not be read |
| Approve empty WBS | cannot approve an empty WBS |
| Override off Fibonacci | estimate value is off the Fibonacci scale |
| Override not increasing | estimate values must be strictly increasing |
| Override empty reasoning | reasoning is empty |
| Non-positive team inputs | team inputs must be positive |

Precondition guards ("WBS must be approved before …", "estimates must be
approved before a proposal") are enforced structurally by the UI (the relevant
action is simply not available until its gate is passed), so the user does not
encounter them as errors under normal flow.

## Assumptions

1. **Single active estimate per session**, held in memory by the service. No
   persistence, no project list, no reopen-across-sessions in v1.
2. **Requirement input lives at the top of the Stage-1 screen** (not a separate
   preceding step).
3. **Determinism for QA** comes from the existing mock provider + priming,
   surfaced as a QA-only UI affordance (see Testing).

## Testing (specifier deliverables)

Per the SwarmForge constitution, the specifier produces **behavior specs, not UI
code**:

1. **Gherkin feature files** for the UI pipeline behavior, separated by behavior
   and technology from the existing API-level features, following the
   Acceptance-Pipeline-Specification format. Mutation-friendly parameters;
   shared setup in `Background`. Working split:
   - a Stage-1 build feature (requirement → WBS → risks → estimates, the two
     approval gates, and the edit/override surface), and
   - a Stage-2 proposal feature (team inputs → the client-clean reveal, with the
     mechanics asserted absent).

   These describe **one pipeline**; the split is by behavior for testability,
   not because there are two products.

2. **End-to-end QA suite(s)** (`qa/ui_*_qa_suite.md`) that operate **at the user
   interface, not via a domain API** — browser-level interactions and on-screen
   assertions. Determinism is provided by the existing **mock provider +
   priming, exposed as a QA-only UI affordance** (a flagged/hidden priming
   surface): QA seeds deterministic WBS / risks / estimates, then drives the
   real Stage-1 and Stage-2 screens and asserts on-screen outputs and observable
   states.

## Handoff note for the coder

The UI is a thin adapter over the existing endpoints:

- `POST /wbs` (text or multipart PDF) → generate.
- `POST /wbs/{id}/tasks`, `PUT/DELETE /wbs/{id}/tasks/{taskId}` → task edits.
- `POST /wbs/{id}/approve` → WBS approval gate.
- `POST /wbs/{id}/risk-notes/flag`, and the risk-note add/edit/delete endpoints.
- `POST /wbs/{id}/estimates/generate`, `PUT …/estimate` override,
  `POST /wbs/{id}/estimates/approve` → estimate approval gate.
- `POST /wbs/{id}/proposal` (team inputs) → the client proposal.
- Mock/priming endpoints (`POST /qa/ai/next-wbs|next-risks|next-estimates`) back
  the QA-only priming affordance and exist only in mock mode.

The coder chooses how the UI is delivered and how QA drives it in the browser;
this design fixes the workflow, states, inputs, outputs, and error behavior.

## Visual design source

The visual design is authored in Pencil at **`es.pen`** (repo root); build the UI
to match it. Its frames:

- `Stage 1 — Build Workspace` — the main build screen (header stage tabs,
  Requirement input, the WBS/estimates table with per-task O/M/P and risk notes,
  and the right-hand Metrics & Pricing panel).
- `Stage 2 — Proposal` — the client-clean proposal screen.
- `Stage 1 — Editing & Gate States` and `Overlays, QA Priming & Error States` —
  reference frames for the locked/available section states, inline errors, and
  the QA priming surface. (Note: these two documentation frames currently
  overlap at the document root — treat them as reference art, not layout.)
- Reusable components: `Button`, `Button Secondary`, `Badge`, `Estimate Row`.

Design tokens live in the `es.pen` variables and should map to the app's style
system: accent `#00BCE9`; semantic `success`/`warning`/`danger` for the
green/yellow/red risk bands; `Space Grotesk` headings, `Inter` body,
`JetBrains Mono` for numeric/monospace figures; `radius-md 10` / `radius-lg 14`.
Export a screen with the Pencil `export_html` tool for exact markup/spacing.
