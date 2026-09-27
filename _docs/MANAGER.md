# Manager

The Manager turns loose human input into Inbox items, tasks, project notes and
status changes. It is not a model inside Fleet. It is a Claude, Codex, Cursor
or Gemini session whose folder is the data root; the conversation stays in that
provider's app. Fleet gives the session tools and skills, and does the
bookkeeping in code.

Workers (Codex, Claude, Cursor, Gemini) execute tasks. They do not triage
input. The person reviews and closes.

## What the Manager uses

- **MCP tools** on `fleet serve` at `/mcp`. They are the only way the Manager
  reads or changes the board. It does not hand-edit task Markdown, `INDEX.md` or
  `_registry/`.
- **Skills** in the data root's `.agents/skills/`. They hold the judgment:
  when to ask, how to split, what counts as ready. See
  [manager-skills](specs/manager-skills.md).

## MCP endpoint

`internal/server/manager_mcp.go` mounts a streamable-HTTP MCP server
(`github.com/modelcontextprotocol/go-sdk`) on the same listener as the
dashboard. Tool handlers call the same `manager.Service` as `/api/manager/*`,
in process. There is nothing to spawn or install: the client only needs the
URL.

Failures come back as typed errors with a suggested fix, and a call that
worked but needs the person's attention carries `warnings`.

| Group | Tools |
|---|---|
| Schema | `manager_schema` (the Intent JSON schema), `manager_vocabulary` (valid project ids, workspaces, assignees, statuses) |
| Command | `manager_command` executes one structured `Intent` |
| Read | `manager_board`, `manager_task`, `manager_workers`, `manager_events`, `manager_project_facts` (read-only, writes nothing) |
| Decide | `manager_resolve_project`, `manager_similar`, `manager_route`, `manager_validate` (dry run, writes nothing) |
| Write | `manager_inbox`, `manager_alias`, `manager_describe`, `manager_answer`, `manager_review`, `manager_update` |

`manager_command` covers creating a task (`description`, `acceptance_criteria`, `context`), changing status, assignee or
priority, adding a comment, cancelling (requires `confirm: true`), and looking
up a task or the board. It is deterministic. The calling agent is already the
LLM, so no classification runs. `manager_events` returns `task.*` and
`project.*` audit rows after a row id; Fleet pushes nothing into the session,
so a skill calls it every turn instead. Each tool's behavior is in
[manager-skills](specs/manager-skills.md).

`manager_inbox` is where a person's own wording is kept: `capture` is the one
durable copy of raw voice or chat input, and `intake` calls it before any new
task. `promote` (or a direct `manager_command` with `source_inbox` set) creates
a task from a capture and links it back; calling it again on the same ref links
another task, since one capture can decompose into several. `list` gives a
status and preview per item, `show` gives one item's full text and every task
it produced. `manager_task` shows a task's own `source_inbox`, so the path from
a captured sentence to the task it became is visible both ways.

## Describing a project

`manager_vocabulary` marks each project's `summary_source`: `generated` (the
scanner's own guess from a README, may say nothing real) or `confirmed` (a
person or the Manager already approved it). `manager_project_facts` reads the
current summary and the project's own README (each member's, for a group);
`manager_describe` writes the paragraph the person approved and flips it to
`confirmed`, and the scanner never overwrites it again. Aliasing is the only
other thing the Manager writes on a project card — what a project is stays in
its repository.

## Watching the bus

Fleet pushes nothing into a Manager session. Instead, the Manager checks:
`manager_events` before or after each turn, starting from the last row id it
saw (0 on a fresh session). `task.needs_attention` and `task.needs_review`
come from execution; `project.discovered`, `project.repos_changed` and
`project.missing` come from the scanner, edge-triggered so a project already
reported missing is not reported again. The bus itself has no disk queue: an
event fires when the scanner or the execution runtime notices the change, not
on a fixed schedule, and a project already gone before Fleet started is not
reported.

The skills are also served as MCP prompts under the same names (`intake`,
`shape-task`, `resolve-project`, `route`, `triage-attention`, `review`,
`briefing`).

### Registration

Before a Manager session is created or adopted, Fleet writes its MCP entry into
the provider's file in the data root: `.mcp.json` (Claude), `.codex/config.toml`
(Codex), `.cursor/mcp.json` (Cursor), `.agents/mcp_config.json` (Gemini). Other
keys and servers already in the file are kept, and a file that is not valid
JSON is refused, not overwritten.

```json
{
  "mcpServers": {
    "manager": {
      "type": "http",
      "url": "http://127.0.0.1:8787/mcp?token=<launch-token>"
    }
  }
}
```

The token is created once in `~/.fleet/launch-token`, so the files are stable
between restarts. `/mcp` requires it, and it is valid only for `127.0.0.1` with
the server's own Host. A query string reaches logs, so do not publish these
files. The address is the `--addr` of `fleet serve` (default
`127.0.0.1:8787`). `fleet serve` must be running.

## HTTP API

Base URL: the `fleet serve` address.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/manager/text` | plain text → fast path → act |
| `POST` | `/api/manager/audio` | `multipart/form-data` field `audio` → speech-to-text → act |
| `POST` | `/api/manager/command` | an already-structured intent |
| `GET` | `/api/manager/schema` | the Intent JSON schema |
| `GET` | `/api/manager/vocabulary` | project, workspace, repository and technology terms for speech-to-text hints |
| `GET` | `/api/manager/binding` | which agent is bound as Manager |
| `GET` | `/api/manager/threads?agent=` | that agent's recent Manager threads |

Board endpoints remain: `POST /api/tasks`, `PATCH /api/tasks`,
`GET /api/state`. The Manager API writes through the same `TaskService` and
never writes Markdown itself.

### Intents

`kind` selects the intent: `task`, `status_change`, `comment`, `board_command`,
`question`, `assignee_change`, `priority_change`, `cancel`. The schema is
served at `/api/manager/schema` and defined in `internal/manager/schema.go`.

### Fast path

For plain text, a deterministic parser runs first. It matches unambiguous
phrases such as `show board`, `show <ref>`, `move <ref> to todo`,
`add comment to <ref>: …`, `assign <ref> to claude`, `set priority of <ref> to
2`, and `cancel <ref> confirm`. `cancel` without `confirm` fails as
`unsafe_command`.

Anything else needs a classifier and, for audio, a transcriber. Neither is
configured: both are stubs that return `llm_not_configured` and
`stt_not_configured`. A Manager session does not need them, because it builds
the Intent itself and calls `manager_command`. The dashboard has no Manager
panel.

### Failure codes

`ambiguous_project`, `ambiguous_repository`, `ambiguous_command`,
`unsafe_command`, `malformed_model_output`, `provider_error`, `not_found`,
`llm_not_configured`, `stt_not_configured`.

### Vocabulary

`/api/manager/vocabulary` is built from live state, not a fixed catalog:
projects (id, title, summary, aliases), workspaces, repositories, the
technologies detected in them (with speech-friendly spellings), and a few
fixed operational terms (agent names, statuses, `MCP`, `Plane`). The `prompt`
field is a readable hint list for a speech-to-text call.

## Data root files

The Markdown shape, `TASK_LIFECYCLE.md`, `OPERATING_MODEL.md` and
`Fleet/ROUTING.md` in the data root are reference for a person. The Manager's
own instructions are the skills and `AGENTS.md`.
