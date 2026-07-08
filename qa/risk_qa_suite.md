# End-to-End QA Suite — Module 2: AI Risk Flagger

Verifies Module 2 **through the HTTP user interface only**, exactly like the
Module 1 suite (`wbs_qa_suite.md`). QA acts as an ordinary HTTP client and must
not call into project packages — only the endpoints and QA affordances below.
Every expectation is an observable HTTP response or a state observable by a
follow-up `GET /wbs/{id}`.

Mirrors the acceptance features `../features/risk_flagging.feature` and
`../features/risk_note_editing.feature`.

Risk Notes belong to tasks; a Risk Note is `{ id, description }` (description
only). Risk flagging requires the WBS to be **approved**. Re-flagging **replaces
all** existing notes (AI-generated and manually added). Notes stay editable
(add/edit/delete) with no separate approval gate.

---

## 1. Preconditions and UI affordances

Start the service in QA mode as in Module 1 (`estimationd --ai-provider=mock`),
which enables the QA priming affordances and makes generation deterministic.

### 1.1 QA affordance — prime the next risk flagging
```
POST /qa/ai/next-risks
Content-Type: application/json

{ "risks": [
  { "taskNumber": 1, "description": "SQL injection" },
  { "taskNumber": 1, "description": "Rate limiting" },
  { "taskNumber": 2, "description": "XSS" }
] }
```
- Response `204 No Content`.
- `taskNumber` is the 1-based position of the task in the current WBS order.
- Sets the exact notes the **next** flag returns; a task with no primed entries
  receives zero notes. One-shot/FIFO, consumed by the next flag.
- Enabled only under `--ai-provider=mock`; otherwise `404`.

### 1.2 Public contract exercised by QA
| Method | Path | Purpose |
|--------|------|---------|
| POST | `/wbs/{id}/risk-notes/flag` | Generate risk notes for every task (replaces all) |
| GET | `/wbs/{id}` | Read the WBS; each task carries `riskNotes` |
| POST | `/wbs/{id}/tasks/{taskId}/risk-notes` | Add a risk note to a task |
| PUT | `/wbs/{id}/tasks/{taskId}/risk-notes/{noteId}` | Edit a risk note description |
| DELETE | `/wbs/{id}/tasks/{taskId}/risk-notes/{noteId}` | Delete a risk note |

Response shapes:
- `GET /wbs/{id}` → each task is `{ "id", "description",
  "riskNotes": [ { "id", "description" } ] }`.
- `POST …/flag` success → `200`, WBS with each task's `riskNotes` populated per
  the prime (unlisted tasks have `[]`).
- `POST …/flag` on an **unapproved** WBS → `409`,
  `{ "error": "WBS must be approved before risk flagging" }`, no notes created.
- `POST …/flag` on an unknown WBS → `404`, `{ "error": "WBS not found" }`.
- Add → `201`; empty description → `400`
  `{ "error": "description is empty" }`; unknown task/WBS → `404`.
- Edit → `200`; empty → `400`; unknown note/task/WBS → `404`
  `{ "error": "risk note not found" }`.
- Delete → `200` (or `204`); unknown → `404` `{ "error": "risk note not found" }`.

**Resolving "risk note K on task number N":** `GET` the WBS, then use
`tasks[N-1].riskNotes[K-1].id` as `{noteId}` and `tasks[N-1].id` as `{taskId}`.

---

## 2. QA cases — Risk Flagging (`risk_flagging.feature`)

Shared setup: prime tasks `["Login API","Login UI","Session store"]`, generate a
WBS (unapproved, tasks 1..3).

### QA-FLAG-1 — Flag an approved WBS produces per-task notes
1. Approve the WBS (`POST /wbs/{id}/approve`).
2. Prime risks: task 1 → ["SQL injection","Rate limiting"], task 2 → ["XSS"].
3. `POST /wbs/{id}/risk-notes/flag`; expect `200`.
4. `GET` shows task 1 with 2 notes (`SQL injection`, `Rate limiting` in order),
   task 2 with 1 note (`XSS`), task 3 with 0 notes.

### QA-FLAG-2 — Reject flagging an unapproved WBS
1. Do **not** approve. Prime any risks.
2. `POST /wbs/{id}/risk-notes/flag`; expect `409`,
   `{ "error": "WBS must be approved before risk flagging" }`.
3. `GET` shows every task with 0 notes.

### QA-FLAG-3 — Re-flagging replaces all notes (AI and manual)
1. Approve. Prime task 1 → ["SQL injection"], task 2 → ["XSS"]; flag.
2. Add a manual note to task 3 (`POST …/tasks/{task3Id}/risk-notes`
   `{ "description": "Manual concern" }`).
3. Prime task 1 → ["CSRF"]; flag again.
4. `GET` shows task 1 with 1 note (`CSRF`), task 2 with 0 notes, task 3 with 0
   notes — the prior AI notes and the manual note are all gone.

### QA-FLAG-4 — Flag a WBS that does not exist
1. `POST /wbs/does-not-exist/risk-notes/flag`.
2. Expect `404`, `{ "error": "WBS not found" }`.

---

## 3. QA cases — Risk Note Editing (`risk_note_editing.feature`)

Shared setup: the 3-task WBS above, approved, then flagged with task 1 →
["SQL injection"], task 2 → ["XSS"], task 3 → []. So task 1 has 1 note, task 2
has 1 note, task 3 has 0.

### QA-NOTE-1 — Add a risk note
1. `POST …/tasks/{task3Id}/risk-notes` `{ "description": "Missing rate limits" }`.
2. Expect `201`. `GET` shows task 3 with 1 note `Missing rate limits`.

### QA-NOTE-2 — Edit a risk note
1. Resolve note 1 on task 1; `PUT …/risk-notes/{noteId}`
   `{ "description": "SQL injection via login" }`.
2. Expect `200`. `GET` shows task 1's note now `SQL injection via login`;
   still 1 note.

### QA-NOTE-3 — Delete a risk note
1. Resolve note 1 on task 1; `DELETE …/risk-notes/{noteId}`.
2. Expect `200`. `GET` shows task 1 with 0 notes.

### QA-NOTE-4 — Reject an empty description
1. Add empty: `POST …/tasks/{task1Id}/risk-notes` `{ "description": "" }`.
2. Edit empty: `PUT` note 1 on task 1 `{ "description": "" }`.
3. Each expects `400`, `{ "error": "description is empty" }`.
4. `GET` shows task 1 still with its original 1 note.

### QA-NOTE-5 — Reject editing or deleting a note that does not exist
1. Edit: `PUT …/tasks/{task1Id}/risk-notes/nonexistent` `{ "description": "X" }`.
2. Delete: `DELETE …/tasks/{task1Id}/risk-notes/nonexistent`.
3. Each expects `404`, `{ "error": "risk note not found" }`.
4. `GET` shows task 1 still with its original 1 note.

---

## 4. Notes and assumptions
- Risk Notes are description-only; no category or severity in this slice.
- Flagging requires an approved WBS (`409` otherwise). Editing individual notes
  does not itself require the approved state; it operates on the current notes.
- Re-flagging is destructive by design (replace all); manual notes are not
  preserved across a re-flag.
- The mock provider is the only deterministic path at the UI. Against a live
  LLM, note contents are non-deterministic; only structural checks would hold.
- Deleting a task (Module 1) removes that task's risk notes; this coupling is
  assumed and not separately exercised here.
