# Agent Launcher

Core launcher is the deterministic layer that decides whether a task may be
handed to an agent.

`core serve` starts the independent task-flow contours (see
`_docs/TASK_FLOW_CONTOURS.md`) and is the normal local operation path. This
file documents Contour 2 (execution) in detail: launch candidates are claimed
atomically, tracked as runtime sessions, logged to per-session files, and
started only after the deterministic runtime gates pass.

Use `core serve --dry-run` only for diagnostics when launch planning should be
logged without spawning external agent processes.

Task schema, project/repository ownership, assignee semantics, and allowed
status transitions are canonical in `_docs/DOMAIN_MODEL.md`.

## Agent Runtime Package

Agent-specific launch logic lives outside the chain/runtime package:

```text
server/internal/execution
server/internal/execution/providers
```

These packages own:

- the provider-neutral `Agent`/`Plan`/session/capability contracts;
- deterministic launch prompt assembly;
- concrete Claude, Codex, Cursor, and Gemini adapters/runners;
- launch plan construction for a task.

The Core chain remains responsible for routing one validated task through the
runtime pipeline. It does not hardcode Claude/Codex/Cursor/Gemini command
details.

Process lifecycle belongs to `corechain.Runtime`, not to agent adapters:

```text
Agent adapter -> Launch plan -> Runtime session -> OS process
```

Agent adapters never own live processes. Runtime creates the session, starts
the OS process from the launch plan, stores process metadata, waits for exit in
a goroutine, and emits lifecycle events.

## Command

```bash
make serve
```

Backend-only debugging after the frontend is already built:

```bash
cd server
go run ./cmd/core serve --root ../.. --addr 127.0.0.1:8788 --session-timeout 10m
```

Dry-run diagnostics:

```bash
cd server
go run ./cmd/core serve --root ../.. --dry-run
```

Prefer `make serve` for normal use because it rebuilds the Svelte frontend
before starting the Go server.

There is no separate daemon command and no separate live command. Launch
decisions are part of Contour 2 (execution), one of several independent
contours `core serve` wires at startup — see `_docs/TASK_FLOW_CONTOURS.md`.

On startup, `core serve` runs `Runtime.Bootstrap()` before opening the HTTP
listener. Bootstrap loads Workspaces, Tasks, and registry counts. When the
Markdown task provider is active it also rebuilds `Work/INDEX.md`. Plane
skips that derived view. The first `/api/state` response is therefore not an
intentionally partial cold-start snapshot.

## Dry Run

`--dry-run` is a troubleshooting mode. It keeps the chain, validation, and
launch planning active, but `Config.AllowsProcessStart()` is false and no
external worker processes are spawned.

## Chain

> **Independent contours, not one chain.** `core serve` wires three
> independent `chain.Processor` contours (Markdown change detection,
> execution, finalization) that fan into shared `TaskEvent` /
> `ExecutionResult` channels — it does **not** wire one continuous
> `FsWalker → … → AgentLauncher` conveyor. See `_docs/TASK_FLOW_CONTOURS.md`
> for the full topology and the hard rejection criteria this section must
> not reintroduce. The rest of this section describes Contour 2 (execution)
> in isolation, which is the piece this file is scoped to.

The launcher uses the same Chain of Responsibility framework shape as
Perceptrail, one instance per contour.

Core vendors a local copy here:

```text
server/lib/chain
```

Source reference:

```text
/Users/operator/Projects/photo/Perceptrail/perceplib/chain
```

The server launcher runtime is built with:

- `chain.NewChainProcessor`;
- `chain.NewEntryPoint`;
- `chain.NewDecorator`;
- typed channels between stages.

Markdown change detection is its own contour, ending at a `TaskEvent` — it
never reaches `AgentLauncher`:

```text
FsWalker
  -> Projector
  -> CoreFinalizer
  -> taskEvents chan TaskEvent
```

`FsWalker` is that contour's entry point. It monitors Core task markdown files
under `Work/**/tasks/*.md`, detects new or changed files, and feeds file
events into the first channel. Events are tagged as `initial`, `created`, or
`modified`. The initial watcher scan hydrates runtime state but does not ask
the finalizer to rebuild generated indexes once per file.

`Projector` parses one task/workspace/registry file event, updates the shared
in-memory Core state, and passes task changes forward as a `ProjectedChange`.

`CoreFinalizer` (not to be confused with Contour 3's `ExecutionFinalizer`,
which handles post-execution task transitions — see below) rebuilds generated
indexes only when the projected event is a created or modified canonical
task/workspace file, then publishes a normalized `TaskEvent` onto the shared
channel. Explicit API task mutations and `core rebuild-index` still rebuild
derived state through their own direct paths. Initial observations do not
trigger per-file rebuild spam.

`/api/state` reads the shared in-memory state owned by the runtime. It must not
load a separate whole-Core snapshot.

Plane task changes reach the same `TaskEvent` channel through an independent
`ProviderSync` contour (polling), not through `FsWalker`/`Projector`.

Execution is a separate contour that begins only at `TaskEvent`:

```text
taskEvents chan TaskEvent
  -> ExecutionEntry
  -> Validator
  -> AgentLauncher
  -> chan *corechain.Task
```

`ExecutionEntry` is Contour 2's entry point. It loads the current task through
`TaskService` for eligible `TaskEvent` values (Markdown and Plane alike); it
never receives a filesystem path, Markdown document, Plane payload, or Manager
request directly.

`Validator` is one decorator that runs the gates synchronously against the same
`*corechain.Task`.

Validator gates:

- `status:pickup`: task must be `needs_rework` or `todo`.
- `assignee`: task must have an assignee.
- `fleet_profile`: Fleet must contain a profile for the selected agent.
- `repository`: task must point to exactly one repository for automatic
  launch.

`AgentLauncher` is the final planning step. It builds the command for tasks
that passed validation. Runtime decides whether to keep that candidate as a
dry-run log entry or continue into the guarded real process path.

`AgentLauncher` calls `internal/execution/providers` to build the launch plan. The chain step
mutates only `Task.LaunchEvaluation`: selected agent, command, passed gates,
failed gates, and final launchability. It does not start a process.

Each step within a contour keeps the same task value moving through that
contour's chain; contours do not share a chain with each other. The task
card remains the durable source of truth; `LaunchEvaluation` is runtime
scratchpad and public API state for `/api/state`, logging, and dashboard
visibility. It is not written back into frontmatter.

Post-execution finalization (Contour 3) is a third, independent contour: the
runtime publishes a typed `ExecutionResult` after a launched agent
process/session ends, and `ExecutionFinalizer` consumes it from its own
channel to apply `TaskService.ReportExecution` — it is not appended as a
sixth step onto the execution contour above. See "Worker Outcome Protocol"
below and `_docs/TASK_FLOW_CONTOURS.md` §6.

## Launch Prompt

Every daemon-launched agent receives a fresh prompt assembled from the current
task context and Core rules. The prompt must not rely on chat memory.

Required prompt content:

- task ref, title, project, workspace, repository, and task card path;
- the task card body;
- `_docs/MANAGER.md` in the Data root;
- `_docs/OPERATING_MODEL.md` in the Data root;
- `Fleet/ROUTING.md`;
- `Fleet/LAUNCH_POLICY.md`;
- completed bot work normally moves to `needs_review`, not `done`;
- do not manually edit generated/service indexes such as `Work/INDEX.md`;
- Core daemon/finalizer refreshes derived files;
- respect no-auto-commit, no-auto-push, and no-auto-PR defaults;
- do not edit this or any canonical task card's status/frontmatter directly,
  and do not create a task-shaped file in the target repository as a stand-in
  for task state (CORE-97) — see "Worker Outcome Protocol" below.

The prompt builder is `internal/execution.BuildPrompt`
(`server/internal/execution/prompt.go`). Its example result payload
deliberately uses a placeholder outcome value
(`"<completed|failed|needs_input|needs_rework|blocked>"`) rather than a real
one like `"completed"` — see "Worker Outcome Protocol" below for why a
literal, recognized example value in the prompt itself would be unsafe.

## Worker Outcome Protocol (CORE-97 / CORE-105)

Daemon-launched workers do not have write access to `core.eggs.gd`'s task
storage in general (only tasks whose target repository happens to be
`core.eggs.gd` itself are the exception, and even then the rule below still
applies). Workers report a typed outcome; only Core's Finalizer mutates task
status, through `TaskService.ReportExecution` (never by editing task cards
or calling Markdown/Plane directly from the execution chain).

The canonical worker outcome payload, appended to a worker's final message:

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

Fields:

| Field | Meaning |
|---|---|
| `outcome` | One of `completed`, `failed`, `needs_input`, `needs_rework`, `blocked`. Legacy `waiting_input` is still accepted and maps to `needs_input`. Runtime may also produce `cancelled` / `timed_out`. Anything else is treated as invalid and blocks the task with an explanation. |
| `summary` | Short prose summary folded into the finalization comment. |
| `artifacts` | Changed files/artifacts left in the target repository. |
| `tests` | Checks/commands run to validate the change. |
| `question` | Operator-facing question when `outcome` is `needs_input` (or `blocked` needing a decision). |
| `error` | Optional failure detail for `failed` / `blocked`. |

Legacy CORE-97 fields (`blockers`, `review_notes`, `suggested_next_status`)
remain accepted during migration and are folded into comment text; workers
must not treat `suggested_next_status` as a status write.

`TaskService.ReportExecution` / `taskflow.MapExecutionOutcomeToStatus` maps
`outcome` to a task status:

| Outcome | Task status |
|---|---|
| `completed` | `needs_review` |
| `blocked` | `blocked` |
| `needs_rework` | `needs_rework` |
| `failed` | `blocked` |
| `needs_input` (legacy `waiting_input`) | pause: task stays `doing`, session stays `waiting_input`, 1-1-1 slot held |
| `cancelled` | `blocked` |
| `timed_out` | `blocked` |
| anything else | `blocked`, with an "invalid result outcome" comment |

Runtime does not call `ReportExecution` for HITL `needs_input`. If that
API is invoked directly, `MapExecutionOutcomeToStatus` still maps it to
`blocked` as a last-resort close. The pause path above is the product
behavior.

`taskflow.ParseExecutionResult` (and compatibility
`tasklifecycle.ParseWorkerResult`) extracts this payload from free-form
captured output (a session transcript, CLI stdout, a JSON-RPC message's
text). It only accepts a candidate JSON object whose `outcome` is recognized,
and if several candidates are present it returns the last one. Both
properties matter for the same reason: some providers echo their own launch
prompt back into the same output this function scans (see the live runner
implementations in `internal/execution/providers`), and that prompt necessarily shows the
worker the exact JSON shape to use. An outcome allowlist is what keeps that
example from ever being mistaken for a real report, and "last match wins" is
what lets a later real report override it once the worker actually finishes.

How each backend gets from captured output to a task transition differs,
because each backend observes the worker differently:

- **Codex** (`codex-app-server`): parses every streamed
  `item/agentMessage/delta`/`item/completed` message's full (untruncated)
  text as it arrives; `session.Result` is set as soon as a valid payload
  appears. The app-server process is fully owned by Core and exits shortly
  after `turn/completed`, so finalization still effectively happens at
  process end (`finishCodexAppServerSession`), just backed by data collected
  throughout the turn rather than only at the very end.
- **Claude** (`background-remote`): the background session is *not* owned by
  Core — it stays open independently so Alex can keep chatting with it on the
  phone, and its `claude agents --json` state can stay `running` indefinitely.
  Every ~5s poll refetches the transcript via `claude logs`; as soon as
  `ParseWorkerResult` finds a valid payload in it, Core applies the task
  finalization immediately (`Runtime.applyWorkerReportedResult`) without
  tearing the session down. This is what makes the phone-visible/HITL
  constraint documented in `Fleet/LAUNCH_POLICY.md` compatible with prompt
  daemon-owned task transitions: the runtime does not need the session to end
  to know the task is done.
- **Cursor** (`cursor-visible`): the process is a blocking foreground child
  Core owns directly; stdout is captured straight to the session log. Once
  `cmd.Wait()` returns, Core re-reads that log and parses it the same way
  before finalizing — there is no long-lived "stays open after the task is
  done" case to handle here the way there is for Claude.

In all three cases the worker's only action is emitting the JSON payload; it
never edits a task card, and Core never grants it task-repo write access to
do so. Launcher claims and Finalizer transitions both go through
`TaskService` only.

Once parsed, the same `WorkerResult` is persisted on the runtime session
record in `~/.fleet/_registry/sessions/<claim_id>.json` as `result` (CORE-99). That
field is evidence/history for dashboard/session inspection — it is not a
second source of task truth. Task status still comes only from Core's
Finalizer via `TaskService.ReportExecution`. Heartbeat upserts must not wipe an already-captured
`result`. Claude also captures the payload on the terminal
`claude agents --json` transition (not only while the session stays
`running`), and restart reconciliation recovers `result` from the provider
transcript when a still-`doing` session ended offline.

## Runtime Logs

Launcher output uses the adopted Perceptrail logger package at
`server/lib/logger`. The package is copied from
`/Users/operator/Projects/photo/Perceptrail/perceplib/logger`; the only
required adjustment is the Core module import path inside the Gontroller
decorator.

Launcher output uses explicit categories:

- `launch-plan candidate`: the task is `needs_rework` or `todo`, has an
  assignee/Fleet profile/exactly one repository, and has a command plan.
- `launch skipped`: the task is intentionally not launchable right now.
  Examples: `backlog`, `doing`, `needs_review`, `done`, or `archived`.
- `launch not_ready`: the task reached launch planning but failed a launch
  gate, such as missing assignee, missing Fleet profile, empty repositories, or
  missing command builder.
- `launch waiting`: the task is otherwise launchable but a project/repository
  runtime session is already active.

Persistent `status: blocked` is not a launch-log category. It is reserved for
operator-facing lifecycle blocks.

Routine non-pickup statuses (`backlog`, `doing`, `needs_review`, `done`,
`archived`) are normal skips. They are appended to
`~/.fleet/_registry/events.ndjson` as `launch_skipped` for auditability, but they are not
printed to stdout on every daemon scan.

Real-mode launch events:

- `task_claimed`: candidate was atomically moved to `doing`.
- `agent_process_starting`: runtime session was created before process start.
- `agent_process_started`: local process started and process id is known.
- `agent_process_start_failed`: process start failed; task is moved to
  `blocked` with a visible review comment.
- `agent_process_exited`: process exited and the runtime session was removed.
- `launch_wait_requeued`: a blocking runtime session ended and waiting tasks
  were sent back into the validator/launcher chain.
- `runtime_orphan_detected`: startup found a `doing` task without an active
  runtime session.
- `session_superseded`: a new runtime session was linked as the replacement
  for the task's most recent prior session (CORE-77 relaunch reuse), whether
  or not an automatic resume was attempted.
- `session_resume_fallback`: an automatic resume attempt (Codex
  `thread/resume`) failed and Core fell back to starting a fresh session
  instead of failing the launch. Claude and Cursor resume do not have this
  fallback: a failed Claude/Cursor resume surfaces as a launch failure rather
  than silently starting fresh, since forking a copy defeats the point of
  1-1-1 session reuse (`_docs/AGENT_SESSION_REUSE.md`).
- `agent_project_session_reused`: a task was claimed for an agent that has no
  same-task session to resume, but does have a session from a *different*
  task in the same project — Core resumed that session instead of starting
  fresh (1-1-1 model, `_docs/AGENT_SESSION_REUSE.md`; capped at one session
  per agent per project, not a general concurrency feature).

Dashboard/API edits log a `status update` message when status, assignee,
priority, body, or comments change. The matching `task_patched` event records
the previous/current task metadata, `source: api`, `stage: status-update`, and
`index_rebuilt: true`. Runtime messages carry fields such as `task`, `stage`,
`reason`, `agent`, `cwd`, and `dry_run`.

## Supported Dry-Run Commands

Current command builders:

- `claude`: builds `claude --bg --remote-control --name <task ref>
  --permission-mode acceptEdits <task prompt>`. Core sets the process working
  directory through the launcher process, not through Claude CLI flags.
  `launch.mode` cannot select a headless/print Claude path; every
  daemon-launched Claude session must expose a background id, session log, and
  remote-control URL when Claude publishes it. If Claude never publishes the
  app-visible URL within `BackgroundRemoteControlReadyTimeout` (default 2m;
  CORE-142), Core stops the hollow background session and blocks the task as
  `remote_control_unavailable` instead of waiting the full idle-attention
  threshold. See `_docs/CORE-142_CLAUDE_REMOTE_CONTROL_FINDINGS.md`.
- `codex`: builds `codex app-server --stdio` and uses Codex App Server as a
  JSON-RPC control plane. Visibility class is `core_visible` (CORE-130):
  LiveReady for normal daemon auto-launch. Sessions appear in Codex Remote on
  paired clients with Core-set thread titles; they are not Claude-style
  phone/app deep links and may not appear in ordinary ChatGPT desktop sidebar
  history. `launch.mode: allow_core_visible` is a legacy no-op. Core sends the
  JSON-RPC initialize handshake, calls `thread/start`, names the thread with
  `thread/name/set` using `<task ref> · <task title>`, then calls `turn/start`
  with the task prompt, and emits `agent_core_control_ready` (not
  `agent_remote_control_ready`). Runtime sessions store `codex_thread_id`,
  `codex_turn_id`, `codex_thread_title`, host name/id when available,
  `visibility_mode`, `last_event`, `last_message`, process id, lifecycle
  `status`, separate `execution_status`, optional result payload, compact
  `tool_usage` evidence (required/used/missing MCP tools; CORE-120), and
  `~/.fleet/_registry/sessions/<claim_id>.log`, all visible through `/api/state` and the
  dashboard runtime strip. Dashboard controls call `/api/sessions/control` for
  `continue`, `interrupt`, and `cancel`; `continue`/`resume` submit another
  `turn/start` input on the stored thread, while `interrupt`/`cancel` call
  `turn/interrupt` when a turn id is known. The verified local build accepts
  `approvalPolicy: "never"` and `sandbox: "workspace-write"` on
  `thread/start`; it rejected the docs-style camelCase sandbox value during
  smoke. Codex sessions use an idle attention threshold that resets on
  JSON-RPC activity; when the threshold is crossed Core records
  `operator_attention`, keeps the task in `doing`, and leaves
  `continue`/`interrupt`/`cancel` controls available. `codex exec` remains a
  fallback/smoke-only path, not the Core primary launcher contract.
- `cursor`: builds the visible CLI command
  `cursor-agent --trust --workspace <path> <task-prompt>` and uses the
  `cursor-visible` backend. Runtime creates a chat id first with
  `cursor-agent create-chat`, launches the prompt with
  `cursor-agent --resume <chat-id> --trust --workspace <path> <task-prompt>`,
  persists `cursor_chat_id` plus an operator resume command, captures
  stdout/stderr in `~/.fleet/_registry/sessions/<claim_id>.log`, and detects terminal
  state from process exit. `--print` remains historical/diagnostic only and is
  never used for normal daemon-launched Cursor tasks.

`make serve` can start only adapters marked `LiveReady`; unverified adapters
fail the `agent_live_ready` launch gate instead of silently hanging.

Agent definition source of truth:

- `Fleet/*.md` documents human-facing role, routing, and capability policy.
- `server/internal/execution/providers` owns executable launch adapters and
  prompt construction.

Do not duplicate command templates into Fleet markdown unless Core also grows a
parser that makes those files executable configuration.

## Current Limits

- `make serve` is the normal runtime path and may start launchable agent
  processes.
- `core serve --dry-run` plans launches without starting agent processes.
- Each controllable live session has an idle attention threshold. The default
  is `10m`; override with `SESSION_TIMEOUT=2m` or
  `core serve --session-timeout 2m`. This is not a wall-clock lifetime cap.
- Claude `background-remote` also has a separate remote-control ready timeout
  (default `2m`). A background id without a published remote-control URL is
  treated as provider failure (`remote_control_unavailable`), not idle HITL.

## Codex App Server Notes

Investigation source: the current OpenAI Codex app-server documentation in
`openai/codex` documents `codex app-server` as the JSON-RPC interface used by
rich Codex clients. It requires an `initialize` request followed by an
`initialized` notification before other calls. New integrations should use
thread and turn APIs: `thread/start`, `thread/resume`, `turn/start`,
`turn/interrupt`, `thread/read`, and `thread/list`.

Core's contract is:

```text
Task -> Core runtime session -> codex app-server --stdio
     -> thread/start -> persist codex_thread_id
     -> thread/name/set -> persist/display codex_thread_title
     -> turn/start -> persist codex_turn_id
     -> stream notifications to session log and dashboard summary fields
     -> worker emits compact result payload
     -> completed turn + process/session finalization decides task transition
```

Core does not copy the full Codex transcript into runtime state. The dashboard
shows the session identity, status, last event/message, compact tool-usage
evidence (CORE-120), and log path; transcript history should be read from the
Codex thread when a dedicated dashboard reader is added.

Visibility class for daemon Codex app-server sessions is `core_visible`
(CORE-130). Core JSON-RPC thread/turn control, dashboard controls, and Codex
Remote thread identity are the approved operator path. Normal daemon
auto-launch is LiveReady; `launch.mode: allow_core_visible` is a legacy no-op.
Core emits `agent_core_control_ready` (not `agent_remote_control_ready`) with
the thread id/title. This is intentionally not Claude-style `app_visible`
deep-link semantics, and threads may not appear in ordinary ChatGPT desktop
sidebar history — open Codex Remote on a paired client and match by Core title
or thread id.

`/api/state` also exposes `session_groups`: every known session for a task
(live plus persisted history from `~/.fleet/_registry/sessions/*.json`), each annotated
with `role` (`current` / `historical` / `superseded`) and `resumable`. This is
how the dashboard groups relaunches of the same task ref instead of showing a
flat, unlabeled list of duplicates — see "Session Reuse On Relaunch" in
`Fleet/LAUNCH_POLICY.md` (CORE-77). Launch evaluations also expose
`visibility_mode`, and execution state exposes `session_pointer` when an
openable/resume pointer is known.

Task lifecycle is intentionally separate from execution lifecycle:

```text
task:      backlog | todo | doing | blocked | needs_rework | needs_review | done
execution: queued | claimed | starting | running | waiting_input |
           operator_attention | stalled | succeeded | failed | timed_out |
           cancelled
```

Daemon-launched workers must not directly finalize task lifecycle. Their final
answer should include a compact result payload — see "Worker Outcome
Protocol" above for the full field set (`outcome`, `summary`, `artifacts`,
`tests`, `question`, `error`) and the `outcome` vocabulary (`completed` /
`failed` / `needs_input` / `needs_rework` / `blocked`; legacy `waiting_input`
still accepted). Core Finalizer applies transitions only through
`TaskService.ReportExecution`.

The runtime/orchestrator applies task transitions from that payload:

- HITL `needs_input` / legacy `waiting_input` is a pause, not a close.
  The task stays `doing`, the session stays `waiting_input` (still
  `IsActive`), and later todos in the same project/repository/assignee
  wait. Runtime does not call `TaskService.ReportExecution`. The human
  answers in the live session or moves the task off `doing` / releases
  the session to free the 1-1-1 slot;
- for Codex and Cursor, once the execution reaches a terminal state
  (`succeeded` plus the parsed `outcome` decides `needs_review` /
  `blocked` / `needs_rework`; `failed`, `timed_out`, or `cancelled` moves
  `doing -> blocked` with the session log path regardless of any parsed
  result);
- for Claude's `background-remote` backend, as soon as a valid payload is
  observed in the polled transcript — except HITL pause above. Completed
  / failed / blocked / needs_rework still finalize without waiting for
  the process to exit (see "Worker Outcome Protocol" above);
- `waiting_input`, `operator_attention`, or `stalled` execution status
  (the live session itself, not only a worker `outcome`) also leaves the
  task in `doing` and exposes operator attention in `/api/state`.

The same outcome parse also runs during daemon-restart session reconciliation
(`Runtime.reconcilePersistedSessions`), not just the live poll loop: if a
Claude background session already went terminal before the daemon came back
up (fast finish, restart mid-session), Core re-fetches its transcript and
finalizes from any outcome payload found there instead of falling back to a
generic stale-session `blocked` comment. This closes a gap where a worker's
already-reported outcome could otherwise be silently dropped by a restart
that happened to land between the report and the next poll tick (see
`Work/core-eggs-gd/decisions/2026-08-03-worker-outcome-protocol.md`).

If a task card is edited to `needs_review`, `blocked`, or another non-`doing`
state while its runtime session is still active, Core projects it back to
`doing` and records a guard comment. This prevents the card from claiming
success while the execution later fails or times out.

Local setup:

- Install the official standalone Codex CLI so `codex` resolves on the daemon
  PATH (for example via `~/.local/bin/codex`). Core launches only that
  shell-visible binary through `exec.LookPath("codex")`.
- Optional explicit override: set `CORE_CODEX_BINARY` to an absolute path when
  the daemon environment cannot see the PATH install. ChatGPT.app's bundled
  `/Applications/ChatGPT.app/Contents/Resources/codex` is not an accepted
  daemon fallback (CORE-151); missing standalone CLI blocks the task with an
  actionable setup diagnostic instead of silently using the app bundle.
- Launch/session diagnostics expose the resolved binary path in plan
  `LiveNotes` (`Resolved Codex binary: …`), `LaunchEvaluation.command[0]`,
  runtime session `command`, and the session log start line (`binary=…`).
- The daemon user must have a writable authenticated `CODEX_HOME`. This Codex
  worker sandbox could not write `/Users/operator/.codex`, so a temporary
  `CODEX_HOME` was used for protocol smoke only.
- The daemon host must have outbound access to `api.openai.com`; the restricted
  Codex worker sandbox could start threads and turns but the model turn failed
  on DNS/request errors.

Smoke procedure:

1. Start `core serve` with an idle threshold from Alex's normal host console.
2. Move a low-stakes Codex task to `todo` or `needs_rework`.
3. Confirm `/api/state` and the dashboard runtime strip show the Core claim id,
   provider `codex`, process id, host name/id, human `codex_thread_title`,
   `codex_thread_id`, `codex_turn_id`, last event, and session log path.
4. Confirm `~/.fleet/_registry/sessions/<claim_id>.log` contains app-server JSON-RPC
   request/notification lines including `thread/name/set` and a terminal
   `turn/completed`.
5. Confirm Codex Remote shows the thread under the connected host/project with
   the Core title rather than the generic launch prompt.
6. Confirm the worker emits an `outcome: "completed"` result payload and the
   runtime moves the task card to `needs_review` only after the turn is
   completed and the app-server process/session has finalized.

Manual runtime smoke command:

```text
cd server
CORE_MANUAL_LIVE_VERIFY=1 go test ./internal/corechain/... -run TestManualVerifyCodexAppServerRuntimeSession -v -timeout 3m
```

This smoke is intentionally skipped by default. It launches the same
`codex-app-server` backend that `core serve` uses, writes a temporary Core task,
waits for the runtime session to leave the active session set, and verifies:

- the session log contains Core-to-Codex `initialize`, `thread/start`, and
  `thread/name/set`, and `turn/start` JSON-RPC requests;
- the session log contains a Codex `turn/completed` notification;
- the worker result payload contains `outcome: "completed"` and the smoke
  marker `CORE_CODEX_APP_SERVER_SMOKE_OK`;
- the temporary task reaches `needs_review` through runtime finalization, not a
  direct worker lifecycle edit.

Partial smoke performed on 2026-08-01 with `codex-cli 0.146.0-alpha.9.2`:
`initialize`, `thread/start`, `turn/start`, event streaming, and terminal
failure handling worked. The created thread was
`019fbeb1-5ec8-7081-940a-b482f8228f46`; the turn was
`019fbeb1-5ef8-7db0-b764-8965a4f2f795`. The assistant response could not
complete in this worker sandbox because the app-server could not reach
`api.openai.com`.

General runtime limits:

- Agent stdout/stderr are written to `~/.fleet/_registry/sessions/<claim_id>.log`.
- Status claim uses `TaskStore.Claim` before process start.
- Runtime sessions are in-memory and intentionally separate from append-only
  event history.
- Session-conflict waits are in-memory `launch_evaluation.waiting` signals,
  exposed through `/api/state` and requeued when the blocking session exits.
- Startup detects orphaned `doing` tasks and exposes them through `/api/state`;
  it does not auto-recover them.
- Runtime events are appended to `~/.fleet/_registry/events.ndjson` — App's
  own runtime home, outside any Data root and outside any git repo.
- Runtime session logs are written under `~/.fleet/_registry/sessions/`.
- Event volume can grow once real retry loops become busy; add rotation or
  pruning before enabling high-frequency automatic retries.
- No agent-launch status changes in dry-run.
- Cursor live launch uses the `cursor-visible` backend. It is CLI-visible:
  operators inspect/list/resume with `cursor-agent ls` and
  `cursor-agent --resume <chat-id> --workspace <path>`, not with a
  Claude-style remote-control URL.
- Automatic launch refuses multi-repository tasks.
- One active runtime session per project/repository is enforced while the
  runtime is alive.

## Cursor Agent Notes

Verified locally on 2026-08-02:

```text
cursor-agent --help
cursor-agent --version
cursor-agent create-chat
cursor-agent --trust --workspace <path> <prompt>
cursor-agent --resume <chat-id> --trust --workspace <path> <prompt>
cursor-agent ls
cursor-agent resume
```

Observed version after reinstall:

```text
2026.07.23-e383d2b
```

The installed path on Alex's host was:

```text
/Users/operator/.local/bin/cursor-agent
```

The executable is a wrapper for:

```text
~/.local/share/cursor-agent/versions/2026.07.23-e383d2b/cursor-agent
```

Core daemon-launches Cursor through the `cursor-visible` backend. The normal
runtime session behavior applies:

- claim task `todo`/`needs_rework` to `doing` before process start;
- reserve task/project/repository runtime session scopes;
- create or reuse a Cursor chat id before task prompt launch;
- start Cursor with `--resume <chat-id>` and without `--print`;
- persist `cursor_chat_id`, `operator_command`, visibility mode
  `cli_visible`, provider capability flags, and process id;
- write stdout/stderr to `~/.fleet/_registry/sessions/<claim_id>.log`;
- move failed manual-smoke launches to `blocked` with the session log path;
- leave successful task completion/writeback to the Cursor worker prompt.

Provider capability flags for Cursor:

| Capability | Contract |
|---|---|
| can start visible session | yes, normal interactive Cursor CLI |
| can list/query sessions | yes, `cursor-agent ls` |
| can resume session | yes, `cursor-agent --resume <chat-id>` |
| can expose operator path | yes, dashboard resume command |
| can detect terminal state | yes, launched process exit |

Cursor does not currently expose a structured JSON-RPC stream comparable to
Codex app-server, and does not expose a Claude-style phone remote-control URL.
Its operator path is CLI/app-visible through the Cursor chat id.

Required local setup:

- `cursor-agent` must resolve on the daemon `$PATH`, or the versioned fallback
  path under `~/.local/share/cursor-agent/versions/*/cursor-agent` must exist.
- The daemon user must be authenticated with Cursor Agent. `cursor-agent status`
  or `cursor-agent about` should run without keychain errors.
- The daemon user must be able to write `~/.cursor/projects`.
- The daemon user must be able to run `cursor-agent create-chat` and
  `cursor-agent ls` without keychain/local-state errors.

Manual visible smoke:

```text
cd server
CORE_MANUAL_LIVE_VERIFY=1 go test ./internal/corechain/... -run TestManualVerifyCursorAgentProcessSession -v -timeout 3m
```

The smoke exercises the same `cursor-visible` backend that `core serve` uses:
it creates a temporary task, creates/persists a Cursor chat id, starts the
visible CLI process without `--print`, captures the session log, verifies the
Cursor-created smoke artifact, and verifies the temporary task reaches
`needs_review` through runtime finalization.

Historical/diagnostic print command:

```text
cursor-agent --print --output-format text --trust --workspace <path> <prompt>
```

This command is explicitly non-interactive automation mode and is not a
supported daemon launch contract.

Codex sandbox attempt on 2026-08-01 verified help/version and the argv contract,
but could not complete authenticated model work:

- `cursor-agent status`, `about`, `ls`, and authenticated operations exited
  with `SecItemCopyMatching failed -50` in the restricted Codex sandbox.
- `cursor-agent --print ... --workspace /private/tmp/core-cursor-smoke ...`
  exited with `EPERM` while creating `~/.cursor/projects/...`.
- After Alex reinstalled Cursor Agent, the Core manual smoke command started a
  real process through `Runtime.startLaunchCandidate`, captured stdout/stderr in
  `~/.fleet/_registry/sessions/<claim_id>.log`, observed exit code 1, and moved the
  temporary smoke task to `blocked`. The captured session log showed the same
  environment restriction:
  `EPERM: operation not permitted, mkdir '/Users/operator/.cursor/projects/...'`.

Even after environment issues are fixed, `cursor-agent --print` remains
non-visible/headless and must not be promoted to daemon-live-ready.

## Gemini Headless Notes

Added 2026-09-22. Core daemon-launches Gemini through Google's Antigravity
CLI, binary `agy` (the product migrated off the old `gemini` CLI/package;
`agy` is what's on `$PATH`), via the `gemini-headless` backend.

Unlike Codex/Cursor, this is not a daemon or a foreground process Core
attaches to — `agy --print <prompt> --output-format stream-json` is a plain
synchronous subprocess that blocks until the turn's terminal `result` event
and then exits:

```text
agy --new-project --print <prompt> --output-format stream-json   # first launch in a cwd
agy --project <id> --print <prompt> --output-format stream-json  # later launches
agy --project <id> --conversation <id> --print <prompt> --output-format stream-json  # resume
```

`agy` does not print the project id `--new-project` creates, so Core
discovers it afterward from `~/.gemini/config/projects/*.json` (matching
`folderUri` against the task's working directory; picks the newest by file
mtime if more than one project file matches, since `--new-project` is not
idempotent per directory) and stores it as `gemini_project_id` on the
runtime session for reuse.

Provider capability flags for Gemini:

| Capability | Contract |
|---|---|
| can start visible session | no — headless only |
| can list/query sessions | no — no discovered `agy` equivalent of `claude agents --json`/`codex thread/resume` |
| can resume session | yes, unverified — `agy --conversation <id>` as a fresh process invocation (same shape as Claude's `--bg --resume`), not yet confirmed against a real Core restart |
| can expose operator path | yes — `agy --project <id> --conversation <id>` printed for manual continuation |
| can detect terminal state | yes — process exit + stream-json terminal `result.status` |

Terminal status mapping (`result.status` → `execution_status`): `SUCCESS` →
`succeeded`, `WAITING` → `waiting_input` (HITL), `CANCELED`/`INTERRUPTED` →
`cancelled`, anything else (`ERROR`/`INVALID`/`RUNNING`/unrecognized) →
`failed`/`provider_error`.

Required local setup:

- `agy` must resolve on the daemon `$PATH` (or `agents.gemini.executable` in
  `core.local.yaml`).
- The daemon user must already be authenticated with Antigravity — Core does
  not perform auth/login itself.

MCP: servers added via `agy mcp add` are global
(`~/.gemini/config/mcp_config.json`), confirmed by live testing against a
real MCP server (not just static config parsing). Antigravity's own docs
describe an additional workspace-scoped `.agents/mcp_config.json`; three
attempts to get a real MCP server recognized through that file (default
project, explicit `--project <id>`, and via the `--project` flag on `mcp
add` itself) all came back empty in this environment, so treat
project-scoped MCP for Gemini as documented-but-unconfirmed rather than
either working or broken — retest before depending on it.

Known upstream reliability caveat (google-antigravity/antigravity-cli, not a
Core bug, as of CLI 1.2.8): headless stream-json turns have several open
issues where a turn can end `SUCCESS`/`CANCELED` with an empty response
while a `run_command` tool call is still executing and gets killed, or where
delegated-subagent output never reaches the parent stream. Not yet exercised
against a real Core task with tool use — `RunGeminiHeadlessSession`'s
"exited without a terminal `result` event" fallback (`provider_error`) is
the current safety net for this class of failure, not a fix for it.

No manual live-smoke Go test exists yet for this backend (unlike
`TestManualVerifyCodexAppServerRuntimeSession` /
`TestManualVerifyCursorAgentProcessSession`) — this session's verification
was ad hoc CLI probing outside the Go test suite, not a repeatable one.
Adding one is a natural next step before relying on this backend for real
work.

## Next Steps

- Add a dashboard action that reads Codex transcript/diff from
  `codex_thread_id` through app-server instead of copying full transcripts into
  runtime state.
- When changing the Codex launcher contract, rerun the manual app-server
  runtime smoke above from an unrestricted authenticated host account.

## First Live Smoke

`CORE-52` was the first intentional live smoke task. Expected operator checks:

- start Core with `make serve ADDR=127.0.0.1:8788 SESSION_TIMEOUT=2m`;
- confirm startup logs show `Launch mode: live`;
- open the dashboard and watch `CORE-52` claim into `doing`;
- confirm `/api/state` exposes the runtime session and `log_path`;
- inspect `~/.fleet/_registry/sessions/<claim_id>.log`;
- confirm the task either reaches `needs_review` or records a clear timeout or
  start failure.
