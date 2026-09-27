# Fleet Launch Policy

Runtime launch and concurrency limits for agents.

It is separate from `Fleet/ROUTING.md` on purpose:

- routing decides who should receive a task;
- launch policy decides whether an agent may actually start.

Canonical task statuses, allowed transitions, and `assignee`/`launch.agent`
semantics live in `../_docs/TASK_LIFECYCLE.md`. This document only defines
launch eligibility and concurrency.

## Current Policy

Do not run more than one agent against the same project or repository at the
same time. This applies to daemon-launched agents, to manually launched agents
when Fleet knows about the work, and to every worker (Claude, Codex, Cursor,
Gemini).

## Concurrency Scopes

Fleet treats these as lock scopes:

| Scope | Meaning | Limit |
|---|---|---:|
| `repository` | One concrete repository path from `_registry` or a project card. | 1 active agent |
| `project` | One `Work/<project-id>/` workspace. | 1 active agent |
| `assignee` | One active runtime session for the same assigned agent, including provider startup and the visible-session handshake. | 1 active session per assignee |
| `assignee_orphan` | One unresolved `doing` task for the same assigned agent after startup found no active runtime session. | 1 active orphan per assignee |
| `task` | One task ref/id. | 1 active agent |

The strictest matching scope wins.

Provider startup states (`starting`, `waiting_for_visible_session`, and
equivalent handshake phases) count as active slot owners. Fleet reserves the
assignee, project, and repository scopes before Claude's remote-control URL is
registered, Cursor's `create-chat` returns, or Codex's `thread/start`
completes. A second task for the same assignee waits while the first is still
starting. When provider startup fails, Fleet releases the slot and requeues
waiting candidates.

Example: if Claude is working on `acme/billing-portal`, Codex does not start
another task against the same repository. Another task in the same workspace
also waits unless the operator explicitly allows parallel work.

## Launch Eligibility

A task is launched automatically only when all are true:

```yaml
status: todo # or needs_rework
assignee: claude # or codex, cursor, gemini
launch:
  agent: # optional explicit override; normally empty
depends_on: [] # empty, or every listed prerequisite is status: done
```

Additionally there must be no active runtime session for the task, for any
repository in `repositories`, for the project workspace, or for the selected
assignee (including a session still starting). There must also be no
unresolved orphaned `doing` task for the selected assignee. The orphan lock is
released by moving the orphaned task out of `doing` to another valid status,
or by reclaiming the same task into a new runtime session.

Any status other than `todo` or `needs_rework` is not launchable. The task
status is the durable task-level lock.

### Hard dependencies (`depends_on`)

A “run after TAG-N” sentence in a task body is advisory. Machine enforcement
uses frontmatter:

```yaml
depends_on:
  - ACME-144
```

- Tokens may be refs, canonical task ids, or locators.
- A prerequisite is satisfied only at `status: done`, not `needs_review`.
- While any dependency is unresolved, Fleet does not claim or start the task.
- The dependent stays `todo` or `needs_rework`. Waiting on a dependency is a
  launch-evaluation skip (`failed_gates` with a `depends_on:` reason), not
  `status: blocked`.
- When a prerequisite becomes `done`, Fleet requeues its dependents.
- `/api/state` shows the `depends_on` gate on `launch_evaluation`, so the
  dashboard can explain why pickup is deferred.
- The operator sets dependencies in the dashboard (task modal and create
  form). Saving writes the frontmatter through `/api/tasks`.

If the selected assignee has no live-ready adapter, Fleet leaves the task in
its pickup status and shows the `agent_live_ready` or `agent_visibility` reason
in runtime state and logs. This is a queue condition, not a task blocker. Fleet
does not move the task to `blocked` and does not reassign it unless the
operator changes `assignee` or `launch.agent`.

## Session Visibility Classes

Fleet distinguishes how an operator can see a daemon-launched session:

| Class | Meaning | Daemon auto-launch |
|---|---|---|
| `app_visible` | Vendor desktop/mobile app session (Claude remote-control URL). | Allowed |
| `cli_visible` | Verified provider CLI identity and resume path (Cursor chat id). | Allowed (explicit opt-in workers) |
| `core_visible` | Fleet API, log and control plane, plus the provider's Remote identity when available (Codex app-server, Codex Remote). Not a deep-link URL. | Allowed when the adapter is live-ready |
| `headless` | Process and log only. | Blocked unless the task card sets `launch.mode: allow_core_visible` |

Adapter mapping:

- Claude `background-remote` → `app_visible`.
- Cursor `cursor-visible` → `cli_visible` (not phone or app visible).
- Codex `codex-app-server` → `core_visible` (Codex Remote plus dashboard controls).
- Gemini `gemini-headless` → `headless` (process and log only, with an
  operator resume command). The visibility gate holds a Gemini task in its
  pickup status unless the task card sets `launch.mode: allow_core_visible`.

Events meaning that Fleet's JSON-RPC control of Codex is ready are
`agent_core_control_ready`. `agent_remote_control_ready` is reserved for
Claude's remote-control URL announcement.

## Pickup Order

When several launchable `needs_rework` or `todo` tasks are available for the
same assignee, Fleet picks deterministically:

1. `needs_rework` before `todo`.
2. Lower numeric `priority` first: `1` before `2`, then `3`, `4`, `5`.
3. Natural ref order inside the same priority.
4. Stable file path order as the final tie-breaker.

Missing priority is treated as `5`.

## Manual Wakeup

Manual wakeup follows the same policy. If the operator opens an agent and asks
it to find itself a task, the worker skips tasks whose project or repository
already has another active agent session. If the operator explicitly says to
ignore the limit for a specific task, record that in the task's `## Handoff`
and `## Activity Log`.

## Lock Model

Fleet uses no separate lock files. Task-level locking is the task status:

```text
todo -> doing
```

Before starting an agent process, Fleet claims the task in the same process:

1. Reload the task.
2. If `status` is not `todo` or `needs_rework`, do not start the agent.
3. Write `status: doing`.
4. Rebuild indexes from source files.
5. Start the agent process only after the claim succeeded.

Repository and project concurrency is enforced from the active runtime
sessions Fleet holds in memory, so it can count active agents without scanning
host processes. Checking for a free slot and reserving it is one atomic step
under one runtime store lock, so two launch candidates cannot both see an empty
session set.

## Conflict Behavior

If a launch conflicts with an existing status, an active runtime session, an
unresolved assignee orphan, or an unresolved `depends_on` prerequisite:

1. Do not start the agent.
2. Keep `status: blocked` for work that needs operator action, missing
   context, approval, access, or after a failed process launch.
3. For an active project or repository session conflict, or an assignee-orphan
   conflict, leave the task status unchanged and expose
   `launch_evaluation.outcome: waiting` with the reason in `/api/state`.
4. For an unresolved `depends_on`, leave the pickup status unchanged and expose
   `launch_evaluation.outcome: not_launchable` with a `depends_on:` failed gate.
5. Record the wait or skip reason in the runtime event log.
6. When the blocking session exits, or a prerequisite reaches `done`, requeue
   the waiting tasks for normal evaluation.

Launch-gate failures are not task statuses. Use `launch_evaluation` and
`failed_gates` for ephemeral decisions, and `status: blocked` only when the
task itself needs human attention.

## Agent Definition Source Of Truth

`Fleet/*.md` is the human routing and capability source of truth. It tells the
Manager which worker is appropriate and why. How the app starts a worker
process is the app's concern and is not recorded here.

## Claude: remote control and human input

Fleet dispatches a Claude session and gets out of the way; the human answers
from the Claude app. Fleet does not try to inject input as the user.

```
claude --bg --remote-control --name "<task ref>" --permission-mode acceptEdits "<prompt>"
```

- `--bg` returns almost immediately and the agent keeps running independently.
- `--remote-control` makes the session visible in the Claude app. The URL
  (`https://claude.ai/code/session_...`) appears in `claude logs <id>` shortly
  after dispatch. Fleet reads it and posts it as a task comment.
- If the URL never appears, Fleet fails the execution as
  `remote_control_unavailable` after the ready timeout instead of treating the
  empty session as idle.
- Fleet supervises completion by polling `claude agents --json` and watching
  the `state` field. Disappearance from the list also counts as ended.
- `claude stop <id>` is used for an explicit stop or cancel, for runtime
  shutdown cleanup, and to end a finished session so the next task can resume
  it. Idle observation shows operator attention instead of stopping a session.
- Fleet never writes to the session's stdin, PTY or GUI. The prompt is a normal
  CLI argument, and follow-up conversation happens between the operator and
  Claude.
- `launch.mode` does not select between headless and visible Claude. Every
  Claude worker is a visible background session.

When no Claude state or transcript activity is seen for the idle threshold,
Fleet marks the execution `operator_attention` and keeps the task in `doing`.
The operator can continue in the app, or cancel or stop the session. A long but
active session is not failed because a fixed time has passed since launch.

## Session Reuse

`/api/state` exposes `session_groups`: every session of a task, live and
historical, with a `role` (`current`, `historical`, `superseded`) and a
`resumable` flag. Relaunching a task links the new session to the most recent
earlier one, so the dashboard shows the chain, not a flat list of duplicates.

Resume by backend:

- **Claude** (`background-remote`, verified): the full session UUID is needed
  (not the short id), the earlier session must be stopped first, and no other
  flags may be repeated on the resume call.
- **Cursor** (`cursor-visible`, verified): the chat id created before the first
  prompt is passed back to `cursor-agent --resume`.
- **Codex** (`codex-app-server`, unverified): a prior thread id is tried first
  and Fleet falls back to a fresh thread on any RPC error.
- **Gemini** (`gemini-headless`, unverified): the conversation id is passed to
  `agy --conversation`.

### One session per agent per project

An agent also keeps one session across different tasks in the same project:
when a task has no same-task session to resume, Fleet resumes the most recent
session that agent ran in that project. The cap is one session per agent per
project.

## Allowed Future Changes

The policy can be relaxed later for read-only research tasks, for separate
repositories inside the same workspace, for agents with proven clean status
writeback, and for tasks that explicitly declare safe parallelism.

More than one concurrent session per agent per project needs a design first:
branch or worktree isolation between concurrent sessions in one repository,
merge and hand-off rules, and who may push what. Land that design before adding
a limit setting. Until then, default to one active agent per project and
repository.
