# End-to-End QA Suite — Module 1: WBS Generator + Human Approval

This suite verifies Module 1 **through the HTTP user interface only**. QA acts as
an ordinary HTTP client. It must not call into project packages, databases, or
any internal API — only the endpoints and QA affordances below. Every
expectation is an observable HTTP response (status code + JSON body) or a state
observable by a follow-up `GET`.

The suite mirrors the acceptance features:
`../features/wbs_generation.feature`, `../features/wbs_editing.feature`,
`../features/wbs_approval.feature`.

---

## 1. Preconditions and UI affordances

### 1.1 Launching the service in QA mode
Start the service so the AI provider is the deterministic mock and the QA
priming affordance is enabled:

```
estimationd --ai-provider=mock --addr=:8080
```

- When `--ai-provider=mock` is set, `POST /qa/ai/next-wbs` is enabled and no
  network LLM call is made.
- When it is not set (production), `POST /qa/ai/next-wbs` responds `404` and QA
  cannot run.

QA treats `http://127.0.0.1:8080` as the base URL below.

### 1.2 QA affordance — prime the next generation
```
POST /qa/ai/next-wbs
Content-Type: application/json

{ "tasks": ["Login API", "Login UI", "Session store"] }
```
- Response `204 No Content`.
- Sets the exact ordered task list the **next** `POST /wbs` will return.
- Priming is one-shot and FIFO: each prime is consumed by the next generation,
  keeping QA cases isolated. QA primes immediately before each generation.

### 1.3 Public contract exercised by QA
| Method | Path | Purpose |
|--------|------|---------|
| POST | `/wbs` | Generate a WBS from a requirement document |
| GET | `/wbs/{id}` | Read a WBS (state, tasks, approved task list) |
| POST | `/wbs/{id}/tasks` | Add a task |
| PUT | `/wbs/{id}/tasks/{taskId}` | Edit a task description |
| DELETE | `/wbs/{id}/tasks/{taskId}` | Delete a task |
| POST | `/wbs/{id}/approve` | Approve the WBS |

Response shapes:
- `POST /wbs` success → `201` `{ "id": "...", "state": "unapproved",
  "tasks": [ { "id": "...", "description": "..." } ] }`
- `GET /wbs/{id}` → `200` `{ "id", "state", "tasks": [...],
  "approvedTasks": [...] | null }`. `approvedTasks` is non-null **only** while
  `state == "approved"`; it is the module output (the Approved Task List).
- Errors → the documented status code with `{ "error": "<message>" }`.

**Resolving "task number N":** QA `GET`s the WBS and uses `tasks[N-1].id` as the
`{taskId}` in edit/delete requests. Ordering is the order returned by `GET`.

---

## 2. QA cases — Generation (`wbs_generation.feature`)

### QA-GEN-1 — Generate a WBS from a valid requirement (text and PDF)
For `input_format` in {text, pdf}:
1. Prime tasks `["Login API", "Login UI", "Session store"]` (text case) or
   `["Charge API", "Refund API"]` (pdf case).
2. Generate:
   - text: `POST /wbs` `Content-Type: application/json`
     `{ "requirement": "Build a login system." }`
   - pdf: `POST /wbs` `multipart/form-data` with part `document` = a valid PDF
     whose text is "Build a payment flow."
3. Expect `201`; body `state == "unapproved"`; `tasks` length == primed length;
   `tasks[0].description` == first primed task.
4. `GET /wbs/{id}` returns the same tasks in order and `approvedTasks == null`.

### QA-GEN-2 — Reject an empty requirement (text and PDF)
For `input_format` in {text, pdf}:
1. text: `POST /wbs` `{ "requirement": "" }`.
   pdf: `POST /wbs` with a valid PDF that contains no extractable text.
2. Expect `400`, body `{ "error": "requirement is empty" }`.
3. No WBS is created (no `id` returned; nothing retrievable).

### QA-GEN-3 — Reject a corrupt PDF
1. `POST /wbs` `multipart/form-data`, part `document` = bytes that are not a
   valid PDF (e.g. `"%PDF-broken"`).
2. Expect `400`, body `{ "error": "document could not be read" }`.
3. No WBS is created.

### QA-GEN-4 — Report a WBS that does not exist
1. `GET /wbs/does-not-exist`.
2. Expect `404`, body `{ "error": "WBS not found" }`.

---

## 3. QA cases — Editing (`wbs_editing.feature`)

**Setup for every editing case:** prime
`["Login API", "Login UI", "Session store"]`, then `POST /wbs` with any valid
text requirement. The resulting WBS is unapproved with tasks 1..3 =
Login API, Login UI, Session store.

### QA-EDIT-1 — Add a task
1. `POST /wbs/{id}/tasks` `{ "description": "Password reset" }`.
2. Expect `201`. `GET` shows 4 tasks; `tasks[3].description == "Password reset"`.

### QA-EDIT-2 — Edit a task
For `(task_number, new_description)` in {(1, "OAuth login API"),
(3, "Redis session store")}:
1. Resolve `taskId` for `task_number`; `PUT /wbs/{id}/tasks/{taskId}`
   `{ "description": new_description }`.
2. Expect `200`. `GET` shows that position now has `new_description`; still 3 tasks.

### QA-EDIT-3 — Delete a task
For `(task_number, deleted_description)` in {(2, "Login UI"), (1, "Login API")}
(run each against a freshly set-up WBS):
1. Resolve `taskId`; `DELETE /wbs/{id}/tasks/{taskId}`.
2. Expect `200`. `GET` shows 2 tasks and no task with `deleted_description`.

### QA-EDIT-4 — Reject an empty description
1. Add empty: `POST /wbs/{id}/tasks` `{ "description": "" }`.
2. Edit empty: resolve task 2's id; `PUT /wbs/{id}/tasks/{taskId}`
   `{ "description": "" }`.
3. Each expects `400`, body `{ "error": "description is empty" }`.
4. `GET` still shows the original 3 tasks unchanged.

### QA-EDIT-5 — Reject editing or deleting a task that does not exist
1. Edit: `PUT /wbs/{id}/tasks/nonexistent` `{ "description": "X" }`.
2. Delete: `DELETE /wbs/{id}/tasks/nonexistent`.
3. Each expects `404`, body `{ "error": "task not found" }`.
4. `GET` still shows the original 3 tasks unchanged.

---

## 4. QA cases — Approval (`wbs_approval.feature`)

**Setup for every approval case:** same 3-task unapproved WBS as Section 3.

### QA-APP-1 — Approve a WBS that has tasks
1. `POST /wbs/{id}/approve`.
2. Expect `200`; body `state == "approved"`.
3. `GET` shows `state == "approved"` and `approvedTasks` with 3 tasks.

### QA-APP-2 — Reject approving an empty WBS
1. Delete all three tasks (delete task 1's id, re-`GET`, repeat until empty).
2. `POST /wbs/{id}/approve`.
3. Expect `409`, body `{ "error": "cannot approve an empty WBS" }`.
4. `GET` shows `state == "unapproved"` and `approvedTasks == null`.

### QA-APP-3 — Editing an approved WBS returns it to unapproved
For each `edit_action` (each against a freshly approved WBS):
- add: `POST /wbs/{id}/tasks` `{ "description": "Audit log" }` → expect `201`.
- edit: `PUT` task 1 `{ "description": "SSO API" }` → expect `200`.
- delete: `DELETE` task 2 → expect `200`.
Steps:
1. Approve the WBS (`POST /wbs/{id}/approve`, confirm `state == "approved"`).
2. Perform the `edit_action`; expect its status above.
3. `GET` shows `state == "unapproved"` and `approvedTasks == null`.

### QA-APP-4 — Re-approve a WBS after an edit
1. Approve the WBS (`state == "approved"`).
2. `POST /wbs/{id}/tasks` `{ "description": "Audit log" }`; `GET` confirms
   `state == "unapproved"`.
3. `POST /wbs/{id}/approve`; expect `200`, `state == "approved"`.
4. `GET` shows `approvedTasks` with 4 tasks.

---

## 5. Notes and assumptions
- INVEST is an AI-prompt instruction, not a validated property; QA does not
  assert INVEST attributes of generated tasks (they are non-deterministic).
- The mock AI provider is the only supported way to make generation
  deterministic at the UI. Against a live LLM, QA-GEN-1 content assertions do
  not hold; only structural checks would.
- "Empty requirement" covers blank text and a PDF with no extractable text.
- Approving an empty WBS is rejected with `409` (assumption; confirmed with the
  specifier).
