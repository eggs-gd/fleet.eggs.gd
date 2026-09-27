# Agent tool evidence

Fleet records compact evidence that the tools a task requires were actually
called during a session. The operator sees whether an agent followed the
tool policy in `AGENTS.md`, instead of trusting a transcript claim or the
worker's own `tests[]`.

## What is stored

A runtime session may carry `tool_usage`:

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

The list of calls and the summaries are capped. Full transcripts stay in the
session log. A worker's `outcome` or `tests` prose is never accepted as the
only evidence.

## Which tools are required

`executionapi.SelectRequirementProfiles` picks profiles deterministically from
the task type and path or content signals.

| Profile | Selected when | Required tool |
|---|---|---|
| `go` | `.go`, `server/`, `go test`, `go.mod` signals. Coding tasks inside the Fleet source tree default here when signals are weak. | `go_diagnostics` |
| `svelte` | `.svelte`, `view/`, SvelteKit signals | `svelte-autofixer` |
| `architecture` | `research` and review tasks, or refactor, architecture and boundary wording | `find_patterns` |
| `mcp_docs` | Svelte work, or explicit MCP or documentation-tool wording | `list-sections` |

`list-sections` is also satisfied by `get-documentation`, and `find_patterns`
by `search_patterns` or `get_pattern_details`. Tools not in a profile, such as
`find_similar_code`, are recorded when observed.

Repositories outside the Fleet source tree do not inherit the Go default. They
get `go_diagnostics` only when Go path signals are present.

## How evidence is captured

At launch, `required` and `missing` are seeded from the task. During the
session and again at finalization, observed calls are merged into `used` and
`missing` is recomputed.

| Provider | Source |
|---|---|
| Codex | `item/started` and `item/completed` JSON-RPC items with tool types |
| Claude | the polled transcript, scanned for `tool_use` and `CallMcpTool` entries |
| Cursor | the session log, and the agent transcript `~/.cursor/projects/<slug>/agent-transcripts/<chat_id>/<chat_id>.jsonl` when `cursor_chat_id` is known |

Cursor's stdout normally holds only the final answer and the worker JSON, not
per-tool events, so the transcript file is the real channel.

A daemon-launched session may have no MCP servers configured at all. Then a
required tool cannot be called, and "missing" is correct.

## Where it shows

- Session cards and the session detail show one chip per required tool
  (`req X` or `missing X`), and a warning when a required tool is missing.
- The task modal shows the warning of its session.
- When a worker reports `completed`, the finalizer adds the missing-tool
  warning to its comment. This does not block `needs_review`.

## Code map

| Area | Path |
|---|---|
| Model, extraction, requirements | `server/internal/executionapi/tool_*.go` |
| Cursor transcript discovery | `server/internal/executionapi/cursor_transcript.go` |
| Session wiring | `server/internal/execution/tool_usage.go` |
| Provider observation | `server/internal/execution/providers/runtime.go` |
| Finalizer warning | `server/internal/execution/execution_result.go`, `host_finalize.go` |
| Dashboard | `view/src/SessionCard.svelte`, `SessionDetail.svelte`, `SessionModal.svelte`, `TaskModal.svelte`, `lib/taskDisplay.js` |
