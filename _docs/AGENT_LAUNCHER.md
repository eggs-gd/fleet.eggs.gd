# Agent launcher

The execution flow decides whether a task may go to an agent, claims it,
starts the agent, tracks the session, and hands the outcome to the finalizer.
Where it sits in the whole system: [ARCHITECTURE](ARCHITECTURE.md). Statuses
and the task schema: [DOMAIN_MODEL](DOMAIN_MODEL.md).

## Running it

```bash
make serve                  # build, then plan launches (dry-run)
make serve LIVE=1           # also start agents for ready tasks
bin/fleet serve --live      # the same, without make
```

Dry-run is the default. It keeps validation and launch planning active and
logs the plan, but starts no process and changes no status. Live mode can also
be saved in Settings (`core.local.yaml`). `--session-timeout` (make
`SESSION_TIMEOUT`, default `10m`) is an idle attention threshold, not a
lifetime cap.

Before the listener opens, `serve` loads workspaces, tasks and registry counts,
so the first `/api/state` is complete. With the Markdown provider it also
rebuilds `Work/INDEX.md`.

## Pipeline

```text
TaskEvent → ExecutionEntry → Validator → AgentLauncher → launch plan
```

- `ExecutionEntry` loads the current task through `TaskService`.
- `Validator` runs the gates (see [ARCHITECTURE](ARCHITECTURE.md#launch)).
- `AgentLauncher` asks `internal/execution/providers` for the command. It only
  fills `Task.LaunchEvaluation`: selected agent, command, passed and failed
  gates, launchability. It does not start anything.

The runtime then claims the task (`TaskService.Claim`, `todo` or
`needs_rework` → `doing`, atomic), creates a runtime session, starts the
process from the plan, and waits for it in a goroutine. Adapters never own a
process. `execution.Service.WaitIdle` waits for those goroutines.

One active runtime session is allowed per project and per repository. A task
that would collide gets `launch_evaluation.waiting` and is requeued when the
blocking session ends. Automatic launch refuses multi-repository tasks.
Sessions are reused across tasks, see [AGENT_SESSION_REUSE](AGENT_SESSION_REUSE.md).

## Launch prompt

`executionapi.BuildPrompt` (`internal/executionapi/prompt.go`) builds a fresh
prompt from the task, never from chat memory. It gives the task ref, title,
type, project, workspace, repository and card path, and points the worker at
the data root's `_docs/TASK_LIFECYCLE.md`, `_docs/OPERATING_MODEL.md`,
`Fleet/ROUTING.md` and `Fleet/LAUNCH_POLICY.md`. Its rules: work only on this
task, leave physical artifacts in the target repository, do not commit, push or
open a PR, do not edit any task card or generated index, and report the outcome
as JSON. The card body is fenced in `<task-card>` tags and labeled as data. The
worker must stop with `blocked` if it asks for something outside these rules.

## Worker outcome protocol

A worker does not write task state. It ends its final message with a JSON
outcome, and only the finalizer changes the status, through
`TaskService.ReportExecution`.

```json
{
  "outcome": "completed",
  "summary": "Implemented the outcome protocol.",
  "artifacts": ["internal/taskflow/execution.go"],
  "tests": ["go test ./internal/taskflow/..."],
  "question": "",
  "error": ""
}
```

| `outcome` | Task status |
|---|---|
| `completed` | `needs_review` |
| `blocked`, `failed` | `blocked` |
| `needs_rework` | `needs_rework` |
| `needs_input` | pause: the task stays `doing`, the session stays `waiting_input`, and the project's queue waits |
| `cancelled`, `timed_out` (set by the runtime) | `blocked` |
| anything else | `blocked`, with an "invalid result outcome" comment |

`question` carries the question for `needs_input`. `error` carries the detail
for `failed` and `blocked`.

`taskflow.ParseExecutionResult` finds the payload in captured output. It
accepts only a JSON object whose `outcome` is recognized, and the last one
wins. Some providers echo the launch prompt back, and the prompt shows the
payload shape with a placeholder outcome; the allowlist and "last wins" keep
that example from being read as a report.

Where the payload is read:

- **Claude** (`background-remote`): every ~5 s the transcript is refetched with
  `claude logs`, and the task is finalized as soon as a valid payload appears.
  After the outcome (except `needs_input`) the background session is stopped
  so the next task resumes it instead of forking it.
- **Codex** (`app-server`): streamed messages are parsed as they arrive, and
  finalization happens at turn and process end.
- **Cursor** (`cursor-visible`): stdout goes to the session log, which is
  parsed after the process exits.
- **Gemini** (`gemini-headless`): the stream-json `result` event.

The parsed result is stored on the session record as evidence. It is not a
second source of task truth. After a restart, reconciliation re-reads a
finished Claude session's transcript and finalizes from it instead of writing
a generic stale-session comment.

A task card edited to a non-`doing` status while its session is still active is
projected back to `doing` with a guard comment. `waiting_input`, `stalled` and
`operator_attention` execution states keep the task in `doing` and show the
attention in `/api/state`.

Task status and execution status are separate:

```text
task:      backlog | needs_rework | todo | doing | blocked | needs_review | done | archived
execution: queued | claimed | starting | running | waiting_input |
           operator_attention | stalled | succeeded | failed | timed_out | cancelled
```

## Agents

An adapter is launchable only when it is marked `LiveReady`. An unverified
adapter fails the `agent_live_ready` gate instead of hanging.

| Agent | Command | Backend | Operator path |
|---|---|---|---|
| Claude | `claude --bg --remote-control --name <ref> --permission-mode acceptEdits <prompt>` | `background-remote` | app-visible remote-control URL |
| Codex | `codex app-server --stdio` (JSON-RPC) | `codex-app-server` | Codex Remote thread, dashboard controls |
| Cursor | `cursor-agent --resume <chat-id> --trust --workspace <path> <prompt>` | `cursor-visible` | `cursor-agent --resume <chat-id>` |
| Gemini | `agy --print <prompt> --output-format stream-json` | `gemini-headless` | `agy --project <id> --conversation <id>` |

Command templates live in `internal/execution/providers`. `Fleet/*.md` holds
the human-facing role and routing; it is not executable configuration.

### Claude

Runs as a background agent with remote control, so a person can continue it
from the Claude app. Edits are accepted automatically; shell and git actions
are not, so the no-commit, no-push, no-PR rules hold. Fleet does not inject
stdin. The launch mode cannot select a headless print path. If Claude does not
publish the remote-control URL within two minutes, Fleet stops the session and
blocks the task as `remote_control_unavailable`.

### Codex

Fleet owns the JSON-RPC channel: `initialize`, `thread/start`,
`thread/name/set` (`<ref> · <title>`), `turn/start`. It stores the thread and
turn ids and the thread title on the session, streams notifications into the
session log, and emits `agent_core_control_ready`. Dashboard controls
`continue`, `interrupt` and `cancel` call `/api/sessions/control`; `continue`
sends another `turn/start` on the stored thread, and the others call
`turn/interrupt`. Thread start uses `approvalPolicy: "never"` and
`sandbox: "workspace-write"`. The visibility class is `core_visible`: the
thread shows up in Codex Remote on a paired client under the Fleet title. It
has no deep link and may not appear in the ChatGPT desktop history.

Resume uses `thread/resume`; if it fails, Fleet starts a fresh thread and
records `session_resume_fallback`. The full transcript is not copied into
runtime state, only identity, status, the last event, tool evidence and the
log.

Requirements: the standalone Codex CLI on `PATH`, or `FLEET_CODEX_BINARY` set
to its absolute path (the ChatGPT app's bundled binary is not accepted); a
writable authenticated `CODEX_HOME`; outbound access to `api.openai.com`.

### Cursor

Fleet creates a chat (`cursor-agent create-chat`), starts the visible CLI with
`--resume <chat-id>` (never `--print`), stores `cursor_chat_id` and a resume
command, and detects the end from process exit. Cursor has no structured event
stream and no remote-control URL. A failed Cursor resume is a launch failure,
not a silent fresh start.

Requirements: `cursor-agent` on `PATH` or under
`~/.local/share/cursor-agent/versions/*/`, an authenticated user, and a
writable `~/.cursor/projects`.

### Gemini

Fleet runs Google's Antigravity CLI `agy`, a synchronous subprocess that ends
with a `result` event. `agy --new-project` does not print the project id, so
Fleet reads it from `~/.gemini/config/projects/*.json` (matching the working
directory, newest file if several) and stores `gemini_project_id`.

Terminal status: `SUCCESS` → `succeeded`, `WAITING` → `waiting_input`,
`CANCELED` and `INTERRUPTED` → `cancelled`, anything else → `failed`
(`provider_error`). An exit without a `result` event is a `provider_error`.

The visibility class is `headless`. The `agent_visibility` gate holds a Gemini
task in its pickup status unless the card sets `launch.mode:
allow_core_visible`.

Requirements: `agy` on `PATH` or `agents.gemini.executable` in
`core.local.yaml`, and an authenticated user. Global MCP servers (`agy mcp
add`) work. Project-scoped MCP through `.agents/mcp_config.json` is documented
by Antigravity but was not recognized in testing. Headless stream-json turns
have upstream reliability issues (a turn can end with an empty response while
a tool call is still running); the missing-`result` check is a safety net, not
a fix. Session resume via `--conversation` is registered as unverified.

## Launch log

Launch decisions are categorized:

- `launch-plan candidate` — passes all gates and has a command.
- `launch skipped` — not pickup-eligible (`backlog`, `doing`, `needs_review`,
  `done`, `archived`). Written to the audit log, not printed.
- `launch not_ready` — failed a gate.
- `launch waiting` — launchable, but a project or repository session is
  active.

Events written to `runtime.db`: `task_claimed`, `agent_process_starting`,
`agent_process_started`, `agent_process_start_failed` (task → `blocked`),
`agent_process_exited`, `launch_wait_requeued`, `runtime_orphan_detected`
(a `doing` task with no session at startup; reported, not auto-recovered),
`session_superseded`, `session_resume_fallback`, `agent_project_session_reused`.
Dashboard and API edits add `task_patched`. `/api/state` exposes
`session_groups`: every known session of a task, marked `current`,
`historical` or `superseded`, and whether it is `resumable`.

Session logs are stored in `runtime.db` and shown by the dashboard. Event
volume is not rotated yet; add pruning before enabling high-frequency
automatic retries.

## Live verification

Three tests start a real agent and are skipped unless
`FLEET_MANUAL_LIVE_VERIFY=1`. Run them from an authenticated account with
network access:

```bash
cd server
FLEET_MANUAL_LIVE_VERIFY=1 go test ./internal/server/... -run TestRuntimeManualVerifyCodexSession -v -timeout 3m
FLEET_MANUAL_LIVE_VERIFY=1 go test ./internal/server/... -run TestRuntimeManualVerifyCursorSession -v -timeout 3m
FLEET_MANUAL_LIVE_VERIFY=1 go test ./internal/server/... -run TestRuntimeManualVerifyClaudeBackgroundRemoteSession -v -timeout 3m
```

Each creates a temporary task, launches it through the same backend `serve`
uses, and checks that the session log shows the expected protocol and that the
task reaches `needs_review` through finalization, not through a worker edit.
There is no such test for Gemini yet.
