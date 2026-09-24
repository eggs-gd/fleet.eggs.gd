# Core MVP Operating Model

This document captures how Core is meant to be used before any fully automatic
agent daemon exists.

## Roles

Core has three practical roles in the MVP:

1. **Core Manager API** — voice/text capture, triage, durable write into
   this tree. Replaces Codex-as-manager for non-coding ingestion. Codex
   chat may still be used as an explicit override during migration.
2. **Worker agents** — Codex/Claude/Cursor/Gemini-style workers that read tasks
   and do work. Workers must not own voice triage or board management.
3. **the operator** — owner, reviewer, final commit/push authority.

There is no assumption that a daemon automatically picks up tasks yet.

## Primary Flow

```text
Phone / Action Button / voice or text
  -> Core Manager API (STT + fast-path / cheap structured LLM)
  -> Core task provider (Markdown default; optional Plane — see
     `_docs/PLANE_TASK_PROVIDER.md`)
  -> daemon or manual wakeup of target worker
  -> worker edits repo files
  -> needs_review
  -> the operator reviews and commits later
```

Core is the shared filesystem state. Chat history is temporary.

## Manager Capture Mode

Use the Core Manager API when:

- input is voice-first;
- the user is walking / mobile / not in a repo;
- the task should be remembered for later;
- the target worker is not obvious;
- the input needs triage into task/note/decision/inbox;
- aliases, project routing, or Core metadata should be updated.

Manager writes:

- `Inbox/items/*.md` for ambiguous raw capture;
- `Work/<project-id>/tasks/*.md` for actionable tasks;
- `Work/<project-id>/notes/*.md` for project-specific non-task knowledge;
- `Work/<project-id>/decisions/*.md` for durable decisions.

## Direct Agent Mode

Use direct agent mode when:

- the user already knows the target worker;
- the task should be done immediately;
- the user is already in Claude/Codex/Cursor/Gemini;
- there is no need for Manager triage before work begins.

If the task should remain visible in Core, the worker should create or update a
Core task file before or during the work.

## Manual Wakeup Mode

For now, workers do not poll Core automatically.

The user manually wakes a worker with a short instruction such as:

```text
знайди собі нову задачу
```

The worker should:

1. Read `CLAUDE.md`.
2. Read `_docs/MANAGER.md`.
3. Read `Fleet/ROUTING.md`.
4. Open `Work/INDEX.md`.
5. Find tasks matching its agent id:
   - `assignee: codex` for Codex;
   - `assignee: claude` for Claude;
   - `assignee: cursor` only when explicitly used later.
6. Prefer `status: todo`. Moving a task from `backlog` to `todo` is the operator's
   signal that the task is ready to attempt.
7. If no direct assignment exists, optionally consider `assignee: unassigned`
   tasks that match the worker profile.
8. Open the task file.
9. Read `## Review Comments` before changing files; returned review tasks may
   already have a previous attempt that needs targeted fixes.
10. Open the linked `Work/<project-id>/PROJECT.md`.
11. Open the linked repository path.
12. If multiple `todo` tasks are available for the worker, pick the lowest
    numeric priority first: `1` before `2`, then `3`, `4`, `5`.
13. If priority is tied, pick by natural `CORE-*` ref order.
14. Set task `status: doing` in the canonical task card. Do not edit
    `Work/INDEX.md`; Core runtime or the shared mutation layer refreshes
    generated views.
15. Do the work.
16. For manual wakeup, set task `status: needs_review` when physical artifacts
    are ready for the operator. For daemon-launched work, return a compact typed
    `ExecutionResult` JSON payload and let Core's Finalizer apply the task
    transition through `TaskService.ReportExecution` after execution is
    terminal — workers must not finalize task status themselves.
17. Do not move your own bot-produced work to `done`; the operator closes reviewed
    tasks manually.

Workers must not pick up `status: backlog` tasks during normal wakeup. Backlog
means the task is written down and structured, but the operator has not decided to put
it into the active queue yet.

If a worker opens a `todo` task and cannot proceed because context is missing,
the worker must move it to `blocked`, write the missing question or context in
the task file, and ask the operator.

A `blocked` task must make the blocker visible without reading chat history.
The worker must add a `## Review Comments` entry that answers:

- what is blocked;
- what exact answer, approval, access, or decision the operator needs to provide;
- what status the operator should move the task to after answering, usually
  `needs_rework` or `todo`.

Core exposes the latest non-the operator blocked review comment as `blocked_reason` in
the backoffice API so the operator can see the worker question directly on the blocked
card. The operator comments are treated as answers or extra context, not as the blocker
source.

## Worker-Specific MVP Behavior

### Codex

Codex can be daemon-launched through the `codex-app-server` backend.

For manual pickup, the user can still open a neighboring Codex task/chat and
say:

```text
знайди собі нову задачу
```

Daemon-launched Codex workers use `codex app-server --stdio`, create or resume
a Codex thread, name it as `<task ref> · <task title>`, and persist
`codex_thread_id`, `codex_turn_id`, host name/id, project/repository, and the
Core launch claim id in runtime state. These sessions are LiveReady for normal
daemon auto-launch (CORE-130). They are visible in Codex Remote on paired
clients, but they may not appear in the ordinary desktop project/sidebar
history. The dashboard shows the Remote identity and copyable thread/turn ids;
until Codex exposes a stable deep link, open Codex Remote, select the connected
host and project, then match the thread by its Core title or thread id.
`launch.mode: allow_core_visible` is a legacy no-op.

### Claude

Claude is useful for phone-visible work because the user has Discovery-style
workflow and remote session access available.

Claude can be manually woken, and later it is the first reasonable target for a
deterministic launcher daemon.

### Cursor

Cursor is daemon-launchable through the `cursor-visible` backend when a task is
explicitly assigned to Cursor. Core creates/stores a Cursor chat id, launches
the normal interactive Cursor Agent CLI without `--print`, captures the session
log, and exposes a CLI resume command in the dashboard. Cursor remains explicit
opt-in rather than a default assignee because its operator path is CLI-visible,
not a Claude-style phone remote-control URL.

## No Auto-Commit Rule

Workers do not commit, push, or open pull requests by default.

Default policy:

```yaml
auto_commit: false
auto_push: false
auto_pr: false
```

Workers may leave uncommitted changes in one or more repositories. The operator reviews
later with normal git tools and decides what to commit.

Only change this rule for a specific task if the user explicitly says so.

## Daemon Direction

The first daemon should be deterministic code, not an LLM.

Agent launch and concurrency rules live in `Fleet/LAUNCH_POLICY.md`.
Canonical workspace/project/repository terms, task schema, and status
transitions live in `TASK_LIFECYCLE.md`.

Daemon v0 may:

- watch Core files;
- validate task frontmatter;
- refresh generated views such as `Work/INDEX.md`;
- rebuild per-agent queues;
- notice launchable `todo` and `needs_rework` tasks;
- start a configured local/remote worker process;
- record launch metadata;
- mark task `doing`, `blocked`, or `needs_review`.

Daemon v0 must not:

- classify raw input;
- invent project routing;
- commit changes;
- push changes;
- open pull requests;
- delete repositories.

The task file is the contract. If a `todo` task still lacks enough information,
the worker should mark it `blocked` and write the missing question into the task.

## Launch Policy

Automatic launch is driven by task lifecycle and assignment. Default launch
frontmatter keeps only adapter-specific overrides and safety flags:

```yaml
launch:
  agent:
  mode:
  auto_commit: false
  auto_push: false
  auto_pr: false
```

If a task needs a specific runtime mode or worker override, Manager may set:

```yaml
launch:
  agent: claude
  mode: default
  auto_commit: false
  auto_push: false
  auto_pr: false
```

For MVP daemon launch, prefer tasks with exactly one repository path. Workspace
tasks spanning multiple repositories should remain manual unless the launcher
grows a deterministic multi-repository strategy.

## Status Meaning

Use the canonical task statuses and transitions from `TASK_LIFECYCLE.md`.

Agents should set `needs_review`, not `done`, when they finish code or file
edits. The operator closes reviewed bot-produced work manually unless they explicitly
asks the worker to close it.

If the operator returns a `needs_review` task to `todo`, the review reason belongs in
`## Review Comments`. The next worker must read those comments before deciding
what to change.

`blocked -> needs_review` is an operator/manual recovery path when the operator has
already verified artifacts (or the blocker was infrastructure/visibility) and
does not want a fake re-execution. Prefer a review comment explaining why.
Automated finalizers must not use this path; they only finalize from `doing`.

Allowed status transitions are defined only in `TASK_LIFECYCLE.md`. UI,
daemon, Manager, and workers should not invent additional transitions.

## Artifact Rule

Chat output is not a task deliverable.

Every task must leave a physical artifact in Core or in the target repository:
code, docs, README, roadmap, decision, project note, generated file, or an
updated task card with concrete findings. This applies to research and planning
tasks as much as implementation tasks.

If the worker only answered in chat, the task remains `doing` or `blocked`.
