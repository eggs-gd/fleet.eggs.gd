# Core Daemon Plan

This document captures the planned daemon direction for Core. It is a parking
lot for daemon thinking, not an implementation commitment.

> **Note (CORE-115):** the independent-contours refactor (`CORE-108`…`CORE-114`)
> replaced the single `FsWalker → … → AgentLauncher` runtime chain this
> document originally planned around. Where sections below still describe
> "the server lifetime chain" as one pipeline, treat
> `_docs/TASK_FLOW_CONTOURS.md` and `_docs/AGENT_LAUNCHER.md` as
> authoritative for current runtime shape.

## Current State

Core currently has a minimal Go daemon/runtime, not a full agent
orchestrator.

It can:

- serve the built backoffice UI from `view/dist`;
- expose `/api/health`;
- expose `/api/state` from the shared in-memory runtime state;
- bootstrap Core state before opening the HTTP listener;
- scan and rescan Core task files through the runtime filesystem walker;
- refresh generated views such as `Work/INDEX.md` after canonical
  task/workspace changes;
- validate launch readiness for changed tasks;
- build agent launch plans;
- launch live agent processes from normal pickup statuses during `core serve`;
- write per-session stdout/stderr logs to `~/.fleet/_registry/sessions/`;
- stop live sessions after a configured timeout;
- update task cards through the API;
- write runtime events to `~/.fleet/_registry/events.ndjson`.

It does not yet:

- watch through native filesystem events instead of polling;
- read structured completion feedback from agents beyond process exit and task
  file changes.

## Principle

The daemon should be deterministic code, not an AI model.

Codex Manager and worker agents interpret messy human intent. The daemon should
only react to already-written Core files with explicit metadata.

The daemon must not guess:

- which project a task belongs to;
- which repository to open;
- which assignee should receive work;
- whether a task is safe to launch.

Those decisions must already exist in Markdown frontmatter, Fleet rules, or the
registry.

Agent launch and concurrency rules live in `Fleet/LAUNCH_POLICY.md`.
Canonical task schema and state transitions live in `_docs/DOMAIN_MODEL.md`.

## Local Implementation Reference

Use Perceptrail as the local implementation reference for daemon flow design.

Relevant code:

- `/Users/operator/Projects/photo/Perceptrail/perceplib/chain/`
- `/Users/operator/Projects/photo/Perceptrail/perceplib/logger/`
- `/Users/operator/Projects/photo/Perceptrail/gontroller/pkg/scan/fswalker.go`
- `/Users/operator/Projects/photo/Perceptrail/gontroller/pkg/scan/entry.go`

Perceptrail already has three patterns Core needs:

- filesystem walking/monitoring that detects known/new/changed files;
- Chain of Responsibility processing where an item moves through explicit
  pipeline stages.
- structured colored service logging on top of zap.

Core should not blindly copy the full Perceptrail flow. The Core chain is
expected to be shorter, but the shape fits:

```text
filesystem event / scan
  -> load Core item
  -> validate metadata
  -> route by status/assignee
  -> optionally claim task
  -> optionally launch worker
  -> wait/read feedback when supported
  -> update state/event log
```

Early Core chains may have only two or three stages. That still counts as a
chain if each stage has one responsibility and can skip, block, or pass the item
forward.

## Non-Goals For MVP

The daemon must not:

- commit automatically;
- push automatically;
- open pull requests automatically;
- delete files or repositories;
- rewrite user-authored task bodies;
- invent missing project aliases;
- route ambiguous work by itself;
- run multiple agents against the same task.

During the MVP, the daemon must also respect the one-active-agent-per-project
and one-active-agent-per-repository limits in `Fleet/LAUNCH_POLICY.md`.

Default policy remains:

```yaml
auto_commit: false
auto_push: false
auto_pr: false
```

## Planned Responsibilities

### 1. Backoffice Host

Serve the local backoffice UI and API.

Status: partially implemented.

Expected behavior:

- `make serve` builds and runs the server;
- server serves `view/dist`;
- server exposes health and state endpoints.
- server performs a synchronous Core bootstrap before listening for HTTP,
  avoiding misleading partial first `/api/state` responses.

### 2. Core State Loader

Read Core files directly instead of serving a pre-exported JSON file.

Status: implemented for the backoffice state surface.

Inputs:

- `Work/*/PROJECT.md`;
- `Work/*/tasks/*.md`;
- `Work/_life/{ideas,notes,reminders,decisions}/*.md`;
- `_registry/*.json`.

Output:

- in-memory state for `/api/state`.

### 3. Validation

Validate Core files and report problems without silently fixing them.

Status: partially implemented for launch readiness.

Examples:

- missing required task frontmatter;
- unknown status;
- unknown assignee;
- unknown project id;
- task project does not match its `Work/<project-id>/` folder;
- `status: todo` without repository or assignee;
- duplicate task ids.

Future command:

```bash
core validate
```

Make target:

```bash
make validate
```

### 4. Index Rebuild

Rebuild deterministic indexes from source files.

Status: implemented for `Work/INDEX.md`.

Targets:

- `Work/INDEX.md`;
- optional per-agent queue files;
- optional per-project task summaries.

Rule:

Source task files are canonical. Index files are generated views.

### 5. Live Refresh and Watcher

Return fresh Core state when files change.

Status:

- `core serve` starts one live runtime chain at server startup.
- `/api/state` reads the shared in-memory state owned by that runtime.
- Bootstrap performs the explicit startup `Work/INDEX.md` rebuild before the
  HTTP server listens.
- The first watcher scan is `initial`: it projects files into memory but does
  not rebuild `Work/INDEX.md` once per file.

Current runtime (superseded description — see note above):

- `core serve` wires independent contours, not one lifetime chain. Markdown
  change detection (`FsWalker -> Projector -> CoreFinalizer`) ends at a
  `TaskEvent`; execution (`ExecutionEntry -> Validator -> AgentLauncher`)
  starts only from that `TaskEvent`. See `_docs/TASK_FLOW_CONTOURS.md` and
  `_docs/AGENT_LAUNCHER.md` for the authoritative topology;
- `Projector` updates the in-memory state consumed by backoffice endpoints;
- watcher input is file events marked as `initial`, `created`, or `modified`,
  not a prebuilt whole-Core state snapshot;
- `created`/`modified` task and workspace files request incremental derived
  state refresh; `initial` observations do not.

Implementation note: Core borrows Perceptrail's `NewFsWalker`/`fsMonitor`
shape, but adapts it to Markdown/Core entities instead of media files. Core does
not need Perceptrail MIME/media grouping behavior.

Watched areas:

- `Work/`;
- `_registry/`;
- `Inbox/`, `Fleet/`, and `_docs/` after their data surfaces become part of
  the loaded daemon state.

Current command:

```bash
core serve --root .
```

Current Make target:

```bash
make serve
```

Dry-run diagnostics are explicit:

```bash
cd server
go run ./cmd/core serve --root ../.. --dry-run
```

Current behavior:

- build shared in-memory state from task file events;
- avoid per-file index rebuilds during the initial watcher scan;
- rebuild `Work/INDEX.md` for explicit recovery commands, API task mutations,
  and created/modified canonical task or workspace files;
- return runtime state from `/api/state`;
- append structured runtime events to `~/.fleet/_registry/events.ndjson`;
- append per-session process logs to `~/.fleet/_registry/sessions/<claim_id>.log` in
  live mode;
- log daemon/runtime messages through the adopted Perceptrail logger package;
- log launch planning as candidate, skipped, not ready, or waiting.

Launch log categories:

- `candidate`: a `todo` or `needs_rework` task passed validation and has an
  agent launch command.
- `skipped`: the task is intentionally not launchable right now, such as
  `backlog`, `doing`, `needs_review`, `done`, or `archived`.
- `not_ready`: the task tried to enter launch planning but failed a launch
  gate, such as missing assignee, missing Fleet profile, missing repository, or
  missing command builder.
- `waiting`: the task is otherwise launchable but an active project/repository
  runtime session owns the concurrency slot.

Routine status skips are written to the event log but are not printed on every
daemon tick. Visible stdout logs use named Perceptrail logger services and are
reserved for candidates, disabled launch items, real blocks, status updates,
bootstrap, chain errors, and finalizer/index activity.
- watch task files and launcher decisions;
- start real agent processes during normal `core serve`;
- plan launches without process start only when `--dry-run` is explicitly
  present.

The runtime does not use generated static JSON. The backoffice should use
`/api/state`.

### 6. Agent Queue Detection

Detect launchable tasks.

A task is launchable only when:

```yaml
status: todo # or needs_rework
priority: 1 # 1 is highest, 5 is lowest; missing means 5
assignee: claude # or codex/cursor later
launch:
```

The daemon should ignore tasks outside pickup statuses. `backlog` is now the
deterministic hold area for tasks that are written down but not ready to run.

When multiple launchable tasks are available for the same assignee, the daemon
must pick deterministically: priority ascending, then natural `CORE-*` ref
order, then stable file path.

Implementation note: queue detection (change detection) and launch decisions
are separate contours, not one lifetime chain — see
`_docs/TASK_FLOW_CONTOURS.md`. Change detection
(`fs_walker -> projector -> finalizer`) publishes `TaskEvent`; execution
(`validator -> agent_launcher`) starts from that event. Validator/launcher
emit launch candidates, routine skips, not-ready gate failures, or waiting
state.

### 7. Task Claiming

Before launching a real process, claim the task by changing its canonical status
from `todo` to `doing`.

This is the MVP task lock:

```text
todo -> doing
```

If the task status is anything other than `todo`, the daemon must not launch an
agent. The status is already the durable lock.

Repository and project limits should be enforced by the daemon's active runtime
sessions once real process launching exists. Those sessions should include:

- task id;
- assignee;
- process id if local;
- started timestamp;
- launcher version;
- command;
- working directory.

Task status prevents duplicate task launches. Runtime sessions prevent too many
agents from working in the same project or repository while the daemon is alive.

### 8. Agent Launcher

Launch local agent processes only from explicit task metadata and Fleet
configuration.

Initial target candidates:

- Claude with remote/session visibility, if command line behavior is stable;
- Codex only if the launched session is useful for the user's phone workflow;
- Cursor later, after repository/cloud behavior is tested.

Launcher inputs:

- task file;
- project card;
- repository path from registry/project card;
- Fleet agent config;
- explicit launch metadata.

Launcher output:

- process started;
- event logged;
- task is claimed as `doing` before process start.

Implementation note: model launching as another short chain. A candidate task
can move through command build, status claim, process start, and feedback
handling stages. If agents later write back status, the feedback reader becomes
another stage rather than special-case logic.

### 9. Event Log

Write append-only events for daemon actions.

Possible path:

```text
~/.fleet/_registry/events.ndjson
```

Decision: `events.ndjson` is local runtime telemetry and is git-ignored. Core
source files and task cards remain the durable source of truth; event logs are
for local diagnosis and orchestration visibility.

If event volume grows because of retries or frequent re-evaluation, add a
rotation/pruning policy before enabling noisy autonomous loops.

Example events:

- state refreshed;
- validation failed;
- task claimed;
- task claim rejected;
- launch dry-run generated;
- agent launch attempted;
- agent launch failed;
- agent process exited;
- index rebuilt.

## Suggested Phases

### Phase 0: Current

- Go server hosts backoffice.
- `/api/state` reads shared runtime memory state.
- Manual wakeup remains the real worker flow.

### Phase 1: Native State API

- Expand runtime projection beyond current workspace, registry, and task files
  when new surfaces need to appear in the backoffice.

### Phase 2: Validation And Indexing

- Add `core validate`.
- Add `core rebuild-index`.
- Add `make validate`.
- Treat indexes as generated views.

### Phase 3: Background Refresh / Watch Mode

- Add background watch/polling loop to `core serve`.
- Refresh state on file changes or polling.
- Rebuild indexes automatically.
- Still no agent launching.

### Phase 4: Launcher Planning

- Detect launchable tasks.
- Print/log what will be launched.
- In dry-run diagnostics, do not start agents.

Status: implemented inside `core serve` as launch planning.

Details: `_docs/AGENT_LAUNCHER.md`.

### Phase 5: Claude Launcher

- Launch only tasks with:
  - `assignee: claude`;
  - `status: todo` or `status: needs_rework`;
  - exactly one repository;
  - a live-ready Claude adapter.
- Add task claiming and event log.
- Do not commit/push/PR.

### Phase 6: Codex/Cursor Launcher Experiments

- Add only after phone visibility and command-line behavior are understood.
- Keep each launcher isolated behind Fleet config.

## Open Questions

- Which exact Claude CLI command creates a phone-visible remote session?
- Can local Codex launched from CLI be useful if it is not visible on phone?
- What is Cursor's reliable local/cloud launch path?
- Should project/repository concurrency remain in memory only, or later gain a
  durable recovery view?
- Should daemon-generated indexes be committed or treated as local generated
  files?
- Should event logs be committed or ignored?

## Review Trigger

Revisit this document after the manual Core workflow has been used for real
tasks from a mobile/voice context.
