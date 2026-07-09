# Estimation Engine

An AI-assisted project **estimation engine**. A Tech Lead describes a project in
plain language (or uploads a PDF requirement); the app uses Claude to break it
into a Work Breakdown Structure (WBS), flag per-task risks, and produce a
3-point (optimistic / most-likely / pessimistic) estimate for each task. It then
rolls those up with **PERT** statistics into a project expected value, a risk
band, and a recommended pricing/contract strategy — and finally into a
client-facing proposal.

The whole thing is a single self-contained Go HTTP server (`estimationd`) that
serves a two-stage web UI. There is no database; all state lives in memory and
resets when the server restarts.

---

## What it does — the two-stage workflow

**Stage 1 — Build workspace** (`/`): the internal Tech-Lead view that shows the
raw mechanics.

1. **Enter a requirement** — type it, or upload a PDF. Claude generates the WBS.
2. **Approve the WBS** — this gate unlocks the Risks and Estimates sections.
3. **Flag risks** — Claude reviews the approved WBS and attaches risk notes to
   tasks. You can also add/edit notes by hand.
4. **Generate estimates** — Claude produces an optimistic/most-likely/pessimistic
   triple plus a short **reasoning** for each task. You can override any estimate.
5. **Review the rollup** — a Metrics & Pricing panel shows the project PERT
   expected value, standard deviation, relative standard deviation (RSD), and the
   derived pricing band (see below). This panel is internal-only.
6. **Approve the estimates** — this gate unlocks Stage 2.

**Stage 2 — Proposal** (`/proposal`): the client-friendly view — risk, cost, and
time in one presentation, without the raw statistics.

### How the risk band works

The project **RSD** (relative standard deviation = project SD ÷ expected value,
as a percentage) drives everything downstream:

| Project RSD | Flag      | Contract strategy         |
|-------------|-----------|---------------------------|
| `< 10`      | 🟢 green  | fixed-price               |
| `10 – 20`   | 🟡 yellow | fixed-price-with-buffer   |
| `> 20`      | 🔴 red    | time-and-materials        |

Because PERT variances **add** across tasks, the project RSD shrinks roughly as
`per-task RSD ÷ √N` — so larger projects (more tasks) trend greener even when
individual tasks are uncertain. This is the standard independent-risk assumption,
and it means red bands only appear for projects with few, very wide-range tasks.

---

## Prerequisites

- **Go 1.26+** (`go version` — see `go.mod` for the exact version).
- An **Anthropic API key** — only if you want real Claude output. Get one from
  <https://console.anthropic.com>. You can run the whole UI without a key using
  the deterministic **mock** provider (see below).

---

## Quick start

Clone, then build the server:

```bash
git clone <repo-url> estimation
cd estimation
go build -o build/estimationd ./cmd/estimationd
```

### Run with real Claude (Anthropic provider)

The key is read from the `ANTHROPIC_API_KEY` environment variable (or an
authenticated Anthropic CLI profile). Never commit or paste your key into source.

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
./build/estimationd --ai-provider=anthropic --addr=:8080
```

Then open **<http://localhost:8080>** and work through the Stage-1 steps above.
The default model is Claude Opus 4.8; override it with `--model=<model-id>`.

### Run without a key (deterministic mock provider)

Mock mode never calls the network. It also enables a QA priming affordance so you
can seed the exact WBS, risks, and estimates you want — ideal for demos, UI work,
and tests.

```bash
./build/estimationd --ai-provider=mock --addr=:8080
```

### Flags

| Flag            | Values                          | Meaning                                             |
|-----------------|---------------------------------|-----------------------------------------------------|
| `--ai-provider` | `anthropic`, `mock`, *(empty)*  | Generation backend. Empty = no AI (blank estimates).|
| `--addr`        | e.g. `:8080`                    | Address to listen on.                               |
| `--model`       | a Claude model id               | Anthropic provider only; empty selects the default. |

---

## Project layout

```
cmd/estimationd/        The HTTP server entry point (main.go)
internal/
  aiprovider/           Anthropic (Claude) provider: WBS, risk, and estimate generation
  wbs/                  Core domain: WBS, 3-point estimates, PERT metrics, pricing, proposal
  httpapi/              JSON API (/wbs, /qa/ai/*)
  webui/                Two-stage HTML UI (handlers, view models, templates/)
  archtest/             Architecture guard tests
features/               Gherkin acceptance specs (the source of truth for behavior)
acceptance/             Generated acceptance tests + step definitions
qa/                     Human-readable QA suites and their executable runners
scripts/                Build, test, and verification scripts
```

The behavioral contract lives in `features/*.feature`. When in doubt about what
the app *should* do, read the relevant `.feature` file first.

---

## Testing & verification

All caches are kept inside the worktree; `scripts/goenv.sh` sets that up and the
scripts source it automatically.

```bash
scripts/test.sh         # unit tests
scripts/property.sh     # property-based tests (behind the `property` build tag)
scripts/acceptance.sh   # parse features → generate → run acceptance tests
scripts/verify.sh       # all of the above, in order

scripts/qa-ui.sh        # end-to-end UI QA suite, driven through the browser UI in mock mode
```

Run the full local suite before pushing:

```bash
scripts/verify.sh
```

---

## Notes

- **No persistence.** Restarting the server clears all estimates. The mock
  provider's priming affordance is the way to reproduce a specific state.
- **API-key hygiene.** Pass the key via the environment, not on the command line
  where it can land in shell history or process listings.
- **Mock vs. anthropic parity.** Both providers drive the exact same UI and
  domain logic; only the source of the generated WBS/risks/estimates differs.
