# Fleet Launch Policy

This document defines runtime launch and concurrency limits for agents.

It is separate from `Fleet/ROUTING.md` on purpose:

- routing decides who should receive a task;
- launch policy decides whether an agent may actually start.

These rules are MVP defaults and may change later.

Canonical task statuses, allowed transitions, and `assignee`/`launch.agent`
semantics live in `../_docs/TASK_LIFECYCLE.md`. This document only defines
launch eligibility and concurrency.

## Current Policy

Do not run more than one agent against the same project or repository at the
same time.

This applies to:

- daemon-launched agents;
- manually launched agents when Core knows about the work;
- Codex, Claude, Cursor, and future workers.

## Concurrency Scopes

Core should treat these as lock scopes:

| Scope | Meaning | MVP Limit |
|---|---|---:|
| `repository` | One concrete repository path from `_registry` or a project card. | 1 active agent |
| `project` | One `Work/<project-id>/` workspace. | 1 active agent |
| `assignee` | One active runtime session for the same assigned agent, including provider startup / visible-session handshake. | 1 active session per assignee |
| `assignee_orphan` | One unresolved `doing` task for the same assigned agent after startup found no active runtime session. | 1 active orphan per assignee |
| `task` | One task ref/id. | 1 active agent |

The strictest matching scope wins.

Provider startup states (`starting`, `waiting_for_visible_session`, and equivalent
handshake phases) count as active slot owners. Core reserves assignee, project,
and repository scopes before Claude remote-control URL registration, Cursor
`create-chat`, or Codex thread/start completes. A second task for the same
assignee must wait while the first is still waiting for visible-session
registration. When provider startup fails, Core releases the slot and requeues
waiting candidates.

Example: if Claude is working on `acme/billing-portal`, Codex should not
start another task against the same repository. Another task in the same
workspace should also wait unless the operator explicitly allows parallel work.

## Launch Eligibility

A task can only be launched automatically when all are true:

```yaml
status: todo # or needs_rework
assignee: claude # or codex/cursor later
launch:
  agent: # optional explicit override; normally empty
depends_on: [] # empty, or every listed prerequisite is status: done
```

Additionally, there must be no active runtime session for:

- the task;
- any repository in `repositories`;
- the project workspace;
- the selected assignee/agent (including sessions still in provider startup /
  visible-session handshake).

There must also be no unresolved orphaned `doing` task for the selected
assignee/agent. The orphan lock is released by moving the orphaned task out of
`doing` to another valid lifecycle status, or by reclaiming the same task into
a new active runtime session.

Any status other than `todo` or `needs_rework` is not launchable. For the MVP,
task status is the durable task-level lock.

### Hard dependencies (`depends_on`)

Written “run after CORE-N” prose in a task body is advisory only. Machine
enforcement uses frontmatter:

```yaml
depends_on:
  - CORE-144
```

Rules:

- Tokens may be human refs (`CORE-144`), canonical task ids, or locators.
- A prerequisite is satisfied only at `status: done` (not `needs_review`).
- While any dependency is unresolved, Core must not claim or start the task.
- The dependent may remain `todo` / `needs_rework`. Dependency waiting is a
  launch-evaluation skip (`failed_gates` / `launch_skipped` with a
  `depends_on:` reason), not operator-facing `status: blocked`.
- When a prerequisite becomes `done`, Core requeues dependents for normal
  launch evaluation.
- `/api/state` surfaces the current `depends_on` gate on
  `launch_evaluation` so the dashboard can explain why pickup is deferred.
- Operators set blockers in the dashboard task modal / create form
  (`depends_on` field), not only as agent instructions in the body.
  Saves write durable Markdown frontmatter through `/api/tasks`.

If the selected assignee has no live-ready adapter, Core must leave the task in
its pickup status and expose the `agent_live_ready` or `agent_visibility`
reason in runtime state and logs. This is a queue availability condition, not a
task blocker. Do not move the task to `blocked` and do not reassign it to
another live-ready agent unless the operator explicitly changes the assignee or
`launch.agent`.

## Session Visibility Classes (CORE-70 / CORE-130)

Core distinguishes how an operator can see a daemon-launched session:

| Class | Meaning | Daemon auto-launch |
|---|---|---|
| `app_visible` | Vendor desktop/mobile app session (Claude remote-control URL). | Allowed |
| `cli_visible` | Verified provider CLI identity/resume path (Cursor chat id). | Allowed (explicit opt-in workers) |
| `core_visible` | Core API/log/control plane plus provider Remote identity when available (Codex app-server / Codex Remote). Not a Claude-style deep-link URL. | Allowed when the adapter is LiveReady (CORE-130) |
| `headless` | Process/log only. | Blocked |

Current adapter mapping:

- Claude `background-remote` → `app_visible` (live-ready).
- Cursor `cursor-visible` → `cli_visible` (live-ready; not phone/app_visible).
- Codex `codex-app-server` → `core_visible` (live-ready; Codex Remote + Core dashboard controls).

`launch.mode: allow_core_visible` is a legacy no-op kept for older task cards.
Normal Codex daemon pickup no longer requires it.

Codex events that mean Core JSON-RPC control is ready use
`agent_core_control_ready`, not `agent_remote_control_ready`. The latter is
reserved for Claude's phone/app remote-control URL announcement.

## Pickup Order

When multiple launchable `needs_rework` or `todo` tasks are available for the
same assignee, Core must pick deterministically:

1. `needs_rework` before `todo`.
2. Lower numeric `priority` first: `1` before `2`, then `3`, `4`, `5`.
3. Natural ref order inside the same priority.
4. Stable file path order only as a final tie-breaker.

Missing priority is treated as `5`.

## Manual Wakeup

Manual wakeup still respects the same policy.

If the user opens an agent and says "знайди собі нову задачу", the worker should
skip tasks whose project or repository already has another active agent session.

If the operator explicitly says to ignore the limit for a specific task, record that in
the task's `## Handoff` and `## Activity Log`.

## Lock Model

Core does not create separate lock files for MVP task locking.

Task-level locking is the task status itself:

```text
todo -> doing
```

Before starting a real agent process, the daemon must claim the task in the same
process:

1. Reload the task file.
2. If `status` is not `todo`, do not start the agent.
3. If `status` is `todo`, write `status: doing` before process start.
4. Rebuild indexes from source files.
5. Start the agent process only after the claim succeeds.

Repository and project concurrency should be enforced from active runtime
sessions owned by the daemon. The real process launcher should keep these
sessions in memory, which lets Core count active agents without scanning host
processes.

The check and reservation must be atomic. The daemon must reserve the
project/repository session slot under one runtime store lock before claiming
and starting a task. A separate "check then later insert session" flow is not
safe because concurrent launch candidates can both observe an empty session
set.

## Conflict Behavior

If a launch conflicts with an existing status, active runtime session,
unresolved assignee orphan, or unresolved `depends_on` prerequisite:

1. Do not start the agent.
2. Keep persistent `status: blocked` reserved for work that needs operator
   action, missing context, approval, access, or a failed process launch.
3. For an active project/repository session conflict or assignee-orphan
   conflict, leave the task lifecycle status unchanged and expose
   `launch_evaluation.outcome: waiting` with the conflict reason in
   `/api/state`.
4. For unresolved `depends_on`, leave pickup status unchanged and expose
   `launch_evaluation.outcome: not_launchable` with a `depends_on:` failed
   gate (logged as `launch_skipped`, not `status: blocked`).
5. Record the wait/skip reason in `~/.fleet/_registry/events.ndjson`.
6. When the blocking runtime session exits, or when a prerequisite reaches
   `done`, requeue waiting/dependent tasks for normal validator/launcher
   evaluation instead of relying on file mtime changes.

Runtime launch-gate failures are not task statuses. Use
`launch_evaluation.outcome: not_launchable` and `failed_gates` for ephemeral
daemon decisions. Use `status: blocked` only when the task itself needs human
attention.

## Agent Definition Source Of Truth

`Fleet/*.md` is the human routing and capability source of truth. It tells
the manager which worker is appropriate and why — that is the only
question this tree answers. How the app actually starts a worker process
is that app's own concern, not something recorded here.

## Claude Remote And HITL Constraint

Core must not pretend that Claude CLI remote visibility and daemon-controlled
human-in-the-loop input are the same capability.

Observed from `CORE-53`:

- `claude -p ... --output-format text` is a headless one-shot process. It can
  complete tasks and write artifacts, but it is not a phone-visible interactive
  session.
- `script -q -F /dev/null claude` can create a phone-visible Claude session,
  but bytes written by Core to stdin did not appear as a user message in the
  remote chat.
- macOS Terminal / AppleScript automation launches a separate GUI process that
  Core does not own. Core can no longer observe stdout, know whether Claude is
  waiting, or safely correlate process completion with task completion.

Valid future directions must choose one primary capability:

| Goal | Candidate mechanism | Tradeoff |
|---|---|---|
| Deterministic daemon-controlled HITL | Claude Agent SDK / callback-based gate | Not phone-visible as a normal Claude remote session. |
| Phone-visible Claude session | Claude CLI remote-capable interactive session | The human operates the session; Core should not inject stdin as if it were the user. |
| Both phone visibility and hosted HITL | Managed/hosted agent runtime if available | Not self-hosted by the Core daemon. |

Do not add another argv/stdin/PTY/GUI paste workaround without first recording
which capability is being selected for the task.

### Decision (2026-08-01)

The operator chose the "phone-visible Claude session" row: Core dispatches the
session and gets out of the way; the human answers from the phone. Core does
not try to be the second row (deterministic daemon-controlled input) at the
same time.

Verified working mechanism, `claude --help` / `claude agents --help`:

```
claude --bg --remote-control --name "<task ref>" --permission-mode acceptEdits "<prompt>"
```

- `--bg` returns almost immediately; the agent keeps running independently.
  Core does not track it via a blocking foreground process/`cmd.Wait()`.
- `--remote-control` makes the session phone-visible. The URL
  (`https://claude.ai/code/session_...`) is printed into `claude logs <id>`
  shortly after dispatch; Core scrapes it and posts it as a task review
  comment so the operator can open it from the phone.
- If that URL never appears (a hollow background session), Core fails the
  execution as `remote_control_unavailable` after the remote-control ready
  timeout instead of treating the hollow shell as idle HITL.
- Core supervises completion by polling `claude agents --json`, matching the
  dispatched short id, and watching the `state` field for a terminal value.
  Confirmed working values: `state: "done"` on normal completion. Other
  terminal values are not yet empirically confirmed; treat disappearance from
  the list as ended too.
- `claude stop <id>` is reserved for explicit operator stop/cancel or runtime
  shutdown cleanup. Idle observation should surface operator attention instead
  of silently stopping a process Core no longer owns.
- Core does not inject stdin/PTY/GUI keystrokes into the session at all. The
  prompt is passed as a normal CLI argument, and any follow-up conversation
  happens directly between the operator and Claude on the phone.
- `launch.mode` is not a selector between headless and visible Claude behavior.
  The daemon does not support invisible/headless Claude workers; an empty mode
  and any legacy mode string both plan the same visible background remote-control
  session.

Operational note: when no Claude state or transcript activity is observed for
the configured idle threshold, Core marks the execution `operator_attention`
and keeps the task in `doing`. The operator can open the phone session, continue there,
or explicitly cancel/stop it. A long but active session must not be failed only
because fixed wall time since launch elapsed.

## Session Reuse On Relaunch (CORE-77)

Repeated launches for the same task ref (reopen, `needs_rework`, retried
`todo`) must not silently pile up duplicate provider-visible sessions. Example
observed live: `CORE-13` had two simultaneously "current" background Claude
sessions in `claude agents --json`, with no way to tell which one was the
active continuation.

Core now tracks this explicitly instead of guessing: every relaunch looks
up the most recent persisted session for the same task and links the new
session to it (regardless of whether an automatic resume is attempted), so
the dashboard can always show the chain instead of a flat list of
duplicates.

Per-backend behavior:

- Codex (`codex-app-server`): resume is attempted automatically. A prior
  thread id is seeded onto the new session and resume is tried before
  falling back to a fresh thread on any RPC error. This is safe to attempt
  automatically because a resume failure is an ordinary RPC error observed
  before any operator-visible side effect — unlike Claude, there is no
  live human-visible session at risk if the attempt fails. Not yet
  empirically confirmed end-to-end that a resumed thread reconstructs the
  exact same operator-visible thread across a fresh process; the safe
  fallback is what makes attempting it responsible anyway.
- Claude (`background-remote`): `CanAttempt: true`, `verified` (2026-09-12,
  live test). CLI quirks that make resume easy to get wrong: the full
  session UUID is required, not the short id; the prior session must be
  stopped first; no other flags may be repeated on the resume call). Was
  `CanAttempt: false` before this verification — do not revert that
  without a reason, this was a deliberate caution, not an oversight.
- Cursor (`cursor-visible`): `CanAttempt: true`, `verified`. Core creates a
  Cursor chat id before the first prompt with `cursor-agent create-chat`,
  stores it as `cursor_chat_id`, and launches visible CLI work with
  `cursor-agent --resume <chat-id> --trust --workspace <repo> <prompt>`.
  Relaunch/reopen can safely reuse the same provider-visible identity by
  passing the previous `cursor_chat_id` back to `--resume`; if no prior chat id
  exists, Core creates a fresh chat and links the sessions through the normal
  supersede fields. Cursor visibility is `cli_visible`, not a Claude-style
  phone remote-control URL.
- `/api/state` exposes `session_groups`: every task's sessions (live and
  historical) with a computed `role` (`current` / `historical` /
  `superseded`) and a `resumable` flag reflecting the capability above, so
  the dashboard can group and label them instead of showing a flat list of
  duplicates.

## Persistent Session Per Agent Per Project (1-1-1 model)

Status: **implemented for exactly one session per `(project, agent)` pair.
Not extended to N concurrent sessions — read "Allowed Future Changes" below
before assuming this scales by changing a number.**

Session reuse above is per-*task* (same task ref, relaunched). The runtime
also now reuses a session across *different* tasks in the same project for
the same agent: when a task is claimed and has no same-task session to
resume, it looks for the most recent session this agent ran anywhere else
in the same project and resumes that instead of starting fresh.

This is deliberately capped at one session, never more, per agent per
project. Raising that cap needs git-flow/worktree coordination rules
decided first, not just a config value — do not assume it scales by
changing a number.

## Allowed Future Changes

This policy can be relaxed later for:

- read-only research tasks;
- separate repositories inside the same workspace;
- agents with proven clean status writeback;
- tasks that explicitly declare safe parallelism.

Concurrency above 1 session per agent per project (worker naming like
`worker1`/`worker2`, a configurable limit) is a real future direction, not a
rejected one — but it needs a git-flow/worktree coordination design first
(branch or worktree isolation between concurrent sessions in the same
repository, merge/handoff rules, who is allowed to push what). Write that
design and land it before adding a concurrency-limit setting; do not add the
setting speculatively and figure out coordination afterward.

Until then, default to one active agent per project/repository.
