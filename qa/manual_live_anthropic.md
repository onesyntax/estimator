# Manual QA — Live Anthropic Smoke Test (WBS Generator)

A manual, human-run walkthrough that exercises Module 1 **through the HTTP user
interface only** against the **real Anthropic provider**. It complements the
deterministic suite in [`wbs_qa_suite.md`](./wbs_qa_suite.md): that suite pins
generation with the mock provider and asserts exact task content; this one uses a
live LLM, so it asserts **structure only** (status codes, state transitions,
task counts) — never specific task text, which is non-deterministic
(`wbs_qa_suite.md` §5).

Use this to confirm the Anthropic wiring works end-to-end (credentials → Messages
API → `submit_wbs` tool → domain → HTTP). It is **not** part of the automated
suite and makes **paid API calls** — run it deliberately, not in CI.

> Automated equivalent: `go test ./internal/aiprovider -run RealAnthropic -v`
> (in `internal/aiprovider/integration_test.go`) makes one live call at the
> package layer. It skips unless `ANTHROPIC_API_KEY` is set. This document is the
> UI-level counterpart.

---

## 0. Preconditions

- A valid `ANTHROPIC_API_KEY` (`sk-ant-...`) from console.anthropic.com. Export it
  in your own shell so it never lands in a log or transcript.
- `jq` for readable output (`brew install jq`), optional.

Start the service with the live provider (leave running; use a second terminal
for the requests):

```
export ANTHROPIC_API_KEY=sk-ant-...
go run ./cmd/estimationd --ai-provider=anthropic --addr=127.0.0.1:8080
```

Base URL below is `http://127.0.0.1:8080`.

- Only `POST /wbs` (and its PDF variant) call Claude — each is a paid request,
  ~7–8s. All other steps are local, instant, and free.
- `POST /qa/ai/next-wbs` (the deterministic priming affordance) is **absent**
  under the anthropic provider and returns `404`. It exists only in mock mode.

## Public contract exercised

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/wbs` | Generate a WBS from a requirement (JSON text or multipart PDF) |
| GET | `/wbs/{id}` | Read a WBS (state, tasks, approved task list) |
| POST | `/wbs/{id}/tasks` | Add a task |
| PUT | `/wbs/{id}/tasks/{taskId}` | Edit a task description |
| DELETE | `/wbs/{id}/tasks/{taskId}` | Delete a task |
| POST | `/wbs/{id}/approve` | Approve the WBS |

Response shapes:
- `POST /wbs` success → `201` `{ "id", "state": "unapproved",
  "tasks": [ { "id", "description" } ], "approvedTasks": null }`
- `GET /wbs/{id}` → `200` same shape; `approvedTasks` is non-null **only** while
  `state == "approved"`.
- Errors → the documented status with `{ "error": "<message>" }`.

**Resolving "task N":** `GET` the WBS and use `tasks[N-1].id` as `{taskId}`.

---

## 1. Generate from a text requirement — `POST /wbs`

```
curl -sS -X POST http://127.0.0.1:8080/wbs \
  -H 'Content-Type: application/json' \
  -d '{"requirement":"Build a login system with email/password auth and session management."}' | jq
```

Expect **`201`** and:
- `state == "unapproved"`,
- `tasks` is a non-empty array; each task has a non-empty `id` and `description`,
- `approvedTasks == null`.

Record the returned `id` (e.g. `wbs-1`) and task ids (`t1`, `t2`, …) for the
steps below. **Do not** assert specific task text — content varies per call.

## 2. Re-read — `GET /wbs/{id}`

```
curl -sS http://127.0.0.1:8080/wbs/wbs-1 | jq
```

Expect **`200`** with the same tasks in the same order; `approvedTasks == null`.

## 3. Add a task — `POST /wbs/{id}/tasks`

```
curl -sS -X POST http://127.0.0.1:8080/wbs/wbs-1/tasks \
  -H 'Content-Type: application/json' \
  -d '{"description":"Add rate limiting on the login endpoint."}' | jq
```

Expect **`201`**; task count increases by one; the new task appears last with the
given description.

## 4. Edit a task — `PUT /wbs/{id}/tasks/{taskId}`

```
curl -sS -X PUT http://127.0.0.1:8080/wbs/wbs-1/tasks/t2 \
  -H 'Content-Type: application/json' \
  -d '{"description":"Implement registration with email verification."}' | jq
```

Expect **`200`**; `t2`'s description is replaced; task count unchanged.

## 5. Delete a task — `DELETE /wbs/{id}/tasks/{taskId}`

```
curl -sS -X DELETE http://127.0.0.1:8080/wbs/wbs-1/tasks/t1 -w '\nHTTP %{http_code}\n'
```

Expect **`200`**; `t1` no longer present; remaining task ids unchanged.

## 6. Approve — `POST /wbs/{id}/approve`

```
curl -sS -X POST http://127.0.0.1:8080/wbs/wbs-1/approve | jq '.state, .approvedTasks'
```

Expect **`200`**; `state == "approved"`; `approvedTasks` populated with the
current tasks (the Tech Lead sign-off / module output).

---

## 7. Error cases (structural, no paid call except where noted)

```
# Empty requirement → 400 { "error": "requirement is empty" }   (calls the API path, but rejected before the LLM)
curl -sS -X POST http://127.0.0.1:8080/wbs -H 'Content-Type: application/json' \
  -d '{"requirement":""}' -w '\nHTTP %{http_code}\n'

# Unknown WBS → 404 { "error": "WBS not found" }
curl -sS http://127.0.0.1:8080/wbs/wbs-999 -w '\nHTTP %{http_code}\n'

# Unknown task id → 404 { "error": "task not found" }
curl -sS -X DELETE http://127.0.0.1:8080/wbs/wbs-1/tasks/t999 -w '\nHTTP %{http_code}\n'

# Priming affordance is mock-only → 404 under the anthropic provider
curl -sS -X POST http://127.0.0.1:8080/qa/ai/next-wbs -d '{"tasks":["x"]}' -w '\nHTTP %{http_code}\n'
```

## 8. PDF requirement (alternative to JSON)

`POST /wbs` also accepts a multipart PDF in a `document` field:

```
curl -sS -X POST http://127.0.0.1:8080/wbs -F 'document=@requirement.pdf' | jq
```

Same structural expectations as step 1 (paid call).

---

## 9. Notes

- **Structural assertions only.** With a live LLM, task text and count are
  non-deterministic; assert shape and state transitions, not content. For
  content-pinned regression checks, use the deterministic mock suite
  (`wbs_qa_suite.md`).
- **Cost & latency.** Steps 1 and 8 are the only paid, slow calls; 2–7 are local.
- **Credential hygiene.** Keep `ANTHROPIC_API_KEY` in your shell environment only.
  Never commit it or paste it into logs, tickets, or chat. Rotate any key that
  has been exposed.
