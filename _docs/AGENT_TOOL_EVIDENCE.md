# Agent Tool Usage Evidence (CORE-120)

Core captures compact evidence that required agent tools were actually invoked
during a runtime session. This closes the visibility loop after sessions became
dashboard-visible: Alex can see whether an agent followed `AGENTS.md` tool
policy, instead of trusting transcript claims or a worker `tests[]` self-report.

## What is stored

Each `RuntimeSession` may include `tool_usage`:

```json
{
  "profiles": ["go", "svelte"],
  "required": ["go_diagnostics", "svelte-autofixer"],
  "used": [
    {
      "name": "go_diagnostics",
      "called_at": "2026-08-04T12:00:00Z",
      "source": "provider_rpc",
      "status": "completed",
      "summary": "ok: 0 issues"
    }
  ],
  "missing": ["svelte-autofixer"],
  "warning": "Missing required tool evidence: svelte-autofixer. ...",
  "updated_at": "2026-08-04T12:05:00Z"
}
```

Constraints:

- Evidence is compact (capped call list + short summaries).
- Full transcripts stay in `_registry/sessions/<claim_id>.log` only.
- Worker JSON `outcome` / `tests` prose is **not** accepted as sole evidence.

## How required tools are selected

Profiles are chosen deterministically from task type + path/content signals
(`executionapi.SelectRequirementProfiles`):

| Profile | When selected | Required tools |
|---|---|---|
| `go` | `.go`, `_backoffice/server`, `go test` / `go.mod` signals; Core-tree coding tasks default here when unclear | `go_diagnostics` |
| `svelte` | `.svelte`, `_backoffice/view`, SvelteKit signals | `svelte-autofixer` |
| `architecture` | `research` / review-ish tasks, or refactor/architecture/boundary wording | `find_patterns` |
| `mcp_docs` | Svelte work or explicit MCP/docs-tool wording | `list-sections` |

Aliases (soft satisfaction):

- `list-sections` may be satisfied by `get-documentation`
- `find_patterns` may be satisfied by `search_patterns` / `get_pattern_details`

Future tools such as `find_similar_code` are recognized when observed, and can
be added to a profile without changing the persistence shape.

Non-Core repositories (for example `eggs-gd-prod` / career-wizard) do **not**
inherit the Go default just because the task type is `feature`/`bug`. They only
get `go_diagnostics` when Go path signals are present.

## How evidence is captured

Observation sources (provider streams / logs):

1. **Codex app-server** — `item/started` / `item/completed` JSON-RPC params
   with MCP/tool item types.
2. **Claude background** — transcript text scanned for `tool_use` /
   `CallMcpTool`-style invocations.
3. **Cursor CLI stdout log** — `_registry/sessions/<claim_id>.log` scanned for
   the same structured patterns. Important: `cursor-agent` usually prints only
   the final assistant text + worker JSON to stdout, **not** per-tool events.
4. **Cursor agent transcript (CORE-140)** — when `cursor_chat_id` is known, Core
   also reads
   `~/.cursor/projects/<slug>/agent-transcripts/<chat_id>/<chat_id>.jsonl`
   and extracts `tool_use` / `CallMcpTool` entries. This is the real Cursor
   observation channel.

On launch, Core seeds `required` / `missing` from the task context. During the
session and again at finalization, observed calls merge into `used` and
`missing` is recomputed.

## CORE-140 findings: why plaques showed `req X missing X`

Two separate issues stacked:

1. **Dashboard presentation (fixed)** — Runtime Sessions / Execution History
   rendered every required tool as `req {tool}` **and** every unmet tool as
   `missing {tool}`. When a tool was required and still missing (the common
   Cursor case), the plaque literally read `req go_diagnostics missing
   go_diagnostics`. That was not duplicate storage; it was duplicate chips.
   The UI now shows one chip per required tool (`missing X` or `req X`).

2. **Cursor observation gap (fixed)** — For daemon-launched Cursor sessions,
   scanning only the stdout session log almost never finds MCP evidence, so
   `missing` stayed equal to `required` even when the agent used tools.
   Transcript JSONL scanning closes that gap when Cursor wrote tool_use events.

3. **Real MCP availability (still open, see CORE-32)** — Daemon Cursor sessions
   often have an empty MCP server catalog (`GetMcpTools` → no servers). In that
   case `go_diagnostics` / `svelte-autofixer` cannot be invoked at all, so
   missing evidence is correct. Wiring MCP profiles into launch remains
   CORE-32; this task does not claim MCP is healthy for Cursor.

4. **Over-broad Go default (narrowed)** — Coding tasks outside the Core tree
   previously defaulted to requiring `go_diagnostics` with no Go signals
   (career-wizard Python bugs showed the same warning). Default Go now applies
   only when the task/repo context looks like Core / `_backoffice`.

## Dashboard / review surfacing

- Runtime Sessions and Execution History show required / used chips (one status
  chip per required tool; no duplicate missing list).
- Missing required tools highlight as a warning on the session card.
- Task modal shows `execution.tool_warning` when the annotated session is
  missing required tools.
- Finalizer soft gate: when publishing a worker `completed` result, Core appends
  the missing-tool warning into the Finalizer comment summary. This does **not**
  block `needs_review` yet — hard mechanical gates belong to CORE-117.

## Code map

| Area | Path |
|---|---|
| Model + extract + requirements | `_backoffice/server/internal/executionapi/tool_*.go` |
| Cursor transcript discovery | `_backoffice/server/internal/executionapi/cursor_transcript.go` |
| Session wiring | `_backoffice/server/internal/execution/tool_usage.go` |
| Provider observation | `_backoffice/server/internal/execution/providers/runtime.go` |
| Finalizer soft warning | `_backoffice/server/internal/execution/execution_result.go`, `host_finalize.go` |
| Dashboard | `_backoffice/view/src/RuntimeSessions.svelte`, `ExecutionHistory.svelte`, `TaskModal.svelte` |
