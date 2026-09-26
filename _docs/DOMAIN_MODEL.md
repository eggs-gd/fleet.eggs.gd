# Core Domain Model

This is the canonical Core domain contract for workspace/project/repository
ownership, task schema, and task state transitions.

All manager rules, worker rules, daemon validation, API mutations, and UI
status controls should refer here instead of redefining their own lifecycle.

## Mutation Contract

Application-level task writes go through `TaskService` (see
`_docs/TASK_FLOW_ARCHITECTURE.md`). Manager, dashboard, launcher claims, and
finalizer must not talk to Markdown or Plane directly and must not implement
separate read-modify-write logic for task storage.

`TaskService` serializes writes in-process and delegates persistence to the
active `TaskProvider` adapter. While the Markdown provider is active, each
provider mutation still follows the file-safe contract:

- lock the task path;
- reread the current file;
- validate expected version and allowed status transition;
- write through a temporary file and atomic rename;
- update the in-memory projection when attached to runtime;
- refresh derived Work views;
- append mutation and index events;
- unlock.

Direct Markdown edits by the operator or external agents remain allowed. The Markdown
change adapter (`fswalker` → `MarkdownProvider` reload → `TaskEvent`)
reconciles them back into the in-memory projection and generated Work views.
Plane (and future providers) use their own optimistic concurrency inside the
adapter; the process write queue does not replace that.

## Ownership Model

Core separates three concepts:

| Term | Meaning | File source |
|---|---|---|
| Workspace | A durable Core area under `Work/<workspace-id>/` that stores project context, tasks, notes, and decisions. | `Work/<workspace-id>/PROJECT.md` |
| Project | The owner of a task. By default this is the same id as the workspace folder. | task `project` frontmatter |
| Repository | A concrete local git repository that may be touched by a task. | task `repositories` frontmatter and `_registry/repositories.json` |

Current MVP storage is workspace-shaped: `Work/<workspace-id>/tasks/*.md`.
Task ownership is project-first: each task belongs to exactly one project by
default, and the task file lives in that project's workspace folder.

Runtime/API state exposes both levels:

- `workspaces[]`: durable Core containers from `Work/<workspace-id>/PROJECT.md`;
- `projects[]`: project entities derived from workspace cards and task
  ownership.

For `workspace_group` workspaces, Core creates repository-backed child projects
from the workspace `repositories` list. Example:

```text
workspace: eggs-gd-prod
project: eggs-gd-prod/career-wizard
repository: eGGs.gd.prod/career-wizard
```

This is the MVP physical project layer. It does not require moving existing
task files into nested folders. A later migration may add explicit per-project
cards if that becomes useful.

Broad workspace-level tasks are allowed only when the work truly spans the
whole workspace group. Otherwise, resolve the specific project/repository and
record it in `repositories`.

## Task Ownership Rules

Every Work task must have:

```yaml
project: <project-id>
repositories:
  - <relative/repository/path>
```

Rules:

1. `project` is the canonical task owner.
2. The task file should live at `Work/<project-id>/tasks/<date>-<slug>.md`.
3. During the MVP, legacy/group tasks may still live under the owning workspace
   folder while runtime exposes an effective `project_id` derived from the
   repository.
4. `repositories` lists the concrete repositories the worker may inspect or
   edit for the task.
5. A task may reference multiple repositories only when the work genuinely
   crosses repository boundaries.
6. If the project is known but the repository is not yet known, keep the task in
   `backlog` or `blocked` and write the missing question in the task card.
7. If neither project nor repository can be resolved confidently, keep the item
   in Inbox or create a `backlog` task with `assignee: unassigned`.
8. The daemon must not guess missing project or repository values.

## Resolution Precedence

When resolving a user phrase to a Core project/repository:

1. Explicit user instruction.
2. Existing `Work/<project-id>/PROJECT.md` id, title, and aliases.
3. Repository aliases recorded in the workspace card.
4. Exact repository name/path in `_registry/repositories.json`.
5. Workspace group relationship in `_registry/project-groups.json`.
6. High-confidence fuzzy match.
7. Ask the operator or park in `backlog`/Inbox.

When a new alias is learned with high confidence, persist it in the workspace
card instead of keeping it only in chat.

## Assignee And Launch Agent

`assignee` is the canonical worker:

```yaml
assignee: codex
```

Humans and AI agents are both workers. Valid MVP values are:

```text
owner, codex, claude, cursor, gemini, unassigned  # `alex` is a legacy alias of `owner`, still accepted
```

`launch.agent` is deprecated for normal tasks. Leave it empty unless there is a
documented real use case where the launch process must start a different worker
than the canonical assignee.

Effective launch worker:

1. If `launch.agent` is non-empty, use it only as an explicit override and make
   the reason visible in the task card.
2. Otherwise use `assignee`.

Assignment precedence:

1. Explicit user instruction.
2. Project override in `Work/<project-id>/PROJECT.md`.
3. Task category fit from `Fleet/ROUTING.md`.
4. Current active agent context.
5. `unassigned`.

## Task Schema

Task frontmatter schema version:

```yaml
schema_version: 1
```

New tasks must include `schema_version: 1`. Existing tasks without
`schema_version` are treated as schema version `0` until migrated. Runtime code
must preserve old cards and should not rewrite every historical task just to add
the field.

Required frontmatter for schema version `1`:

```yaml
schema_version: 1
id: work-<yyyy-mm-dd>-<slug>
ref: CORE-<number>
title: Human-readable task title
type: feature | bug | research | review | maintenance | decision
status: backlog | needs_rework | todo | doing | blocked | needs_review | done | archived
priority: 1 | 2 | 3 | 4 | 5
project: <project-id>
repositories:
  - <relative/repository/path>
depends_on:
  - CORE-<number>   # optional; hard launch gate (see below)
assignee: owner | codex | claude | cursor | gemini | unassigned
assignment_reason: <short reason>
source: voice | text | url | file | chat | manager
source_inbox: <inbox-id or empty>
created_at: <ISO-8601>
updated_at: <ISO-8601>
launch:
  agent:
  mode:
  auto_commit: false
  auto_push: false
  auto_pr: false
```

`todo` is a lifecycle **status** (ready for pickup), not a `type`. Empty or
legacy `type: todo` values normalize to `feature`; Markdown index regeneration
rewrites those cards on disk.

### Task dependencies (`depends_on`)

Optional frontmatter list of prerequisite tasks. Daemon launch treats this as a
hard gate, not advisory Markdown prose:

```yaml
depends_on:
  - CORE-144
```

Accepted dependency tokens:

- human refs such as `CORE-144` (case-insensitive);
- canonical task `id` values;
- opaque task locators / relative paths when useful internally.

A dependency is satisfied only when the prerequisite task status is `done`.
`needs_review` is **not** enough — the operator must accept the prerequisite before
dependents may auto-launch. Missing or self-referential dependencies stay
unresolved.

Unresolved dependencies keep the dependent task in its pickup status
(`todo` / `needs_rework`). Core records `launch_evaluation.failed_gates` with a
`depends_on:` reason and skips claim/launch; it does **not** flip the task to
operator-facing `status: blocked`. See `Fleet/LAUNCH_POLICY.md`.

Operators can set or clear `depends_on` from the backoffice dashboard (task
modal and create form). That writes the same frontmatter field the daemon
reads — dependency gates are not limited to agent-authored Markdown prose.

Runtime/API may expose derived fields that are not frontmatter, including:

```yaml
workspace_id: <folder-derived workspace id>
project_id: <effective project id used by UI/daemon>
```

Do not hand-edit derived fields into task cards.

Migration expectations:

- New task creation uses schema version `1`.
- Existing version `0` cards remain valid input.
- Migration should happen when a card is naturally edited, or through an
  explicit migration command/task.
- The daemon should report missing `schema_version` as legacy metadata, not as a
  launch blocker by itself.

## Task State Machine

Canonical statuses:

| Status | Meaning |
|---|---|
| `backlog` | Structured task, not prioritized for worker pickup. |
| `needs_rework` | Failed operator review; unfinished work returned to the worker queue before normal todo. |
| `todo` | Ready for active worker pickup, not started. |
| `doing` | A human/agent has started work. |
| `blocked` | Cannot proceed without missing info, approval, access, lock/session conflict, or failed launch. |
| `needs_review` | Worker produced physical artifacts; the operator should review before done/commit. |
| `done` | Reviewed/accepted complete. |
| `archived` | No longer relevant or intentionally kept only for history. |

Allowed transitions:

| From | To |
|---|---|
| `backlog` | `todo`, `archived` |
| `needs_rework` | `doing`, `blocked`, `todo`, `archived` |
| `todo` | `doing`, `blocked`, `backlog`, `archived` |
| `doing` | `needs_review`, `needs_rework`, `blocked`, `todo`, `done` |
| `blocked` | `needs_review`, `needs_rework`, `todo`, `archived` |
| `needs_review` | `needs_rework`, `todo`, `done`, `archived` |
| `done` | `archived` |
| `archived` | `backlog` |

Transition rules:

- Moving `backlog -> todo` is the readiness signal. Do not add a separate
  readiness boolean.
- Moving `needs_review -> needs_rework` means the operator rejected the produced
  artifacts and the same task needs fixes. This is not a new task.
- `needs_rework` is picked before normal `todo` for the same assignee.
- Manually woken workers may move their claimed task to `doing`, `blocked`, or
  `needs_review`.
- Daemon-launched workers must report a result payload and leave final task
  lifecycle transition to the runtime/orchestrator after execution reaches a
  terminal state.
- Bot-produced work must stop at `needs_review`; the operator moves it to `done`.
- Returning `needs_review -> todo` must add a `## Review Comments` entry.
- `todo -> doing` is the durable task-level lock before real agent launch.
- `doing -> needs_rework` is allowed for manual/orphan resolution when a
  restarted runtime cannot continue the previous execution and the operator wants the
  same worker to retry with review context.
- `doing -> done` is allowed only for the operator/operator resolution of orphaned or
  externally completed work. Bot-produced work still stops at `needs_review`.
- `blocked -> needs_review` is an operator/manual recovery path only: when the operator
  has verified that physical artifacts already exist (or the blocker was
  infrastructure/visibility rather than missing work) and wants review without
  a fake re-execution. Prefer a `## Review Comments` reason. Automated
  finalizers must not use this transition to hide failed executions — they
  only finalize from `doing`.
- `blocked -> needs_rework` / `blocked -> todo` are the normal retry paths after
  fixing the blocker.
- `done -> archived` is retention cleanup, not task completion.
- `archived -> backlog` is the only reopen path from archive.

## Generated Views

Canonical task cards live under `Work/<project-id>/tasks/`.

Generated views such as `Work/INDEX.md` are projections. Managers and agents
must not hand-maintain generated indexes. Normal task edits update canonical
task cards and let Core runtime or the shared mutation layer refresh derived
views. The deterministic rebuild command is for startup, recovery, and explicit
maintenance.

## Definition Of Done

A task is not complete if the only result is a chat response.

Every task must produce or update at least one physical artifact in Core or the
target repository: code, docs, tests, README, roadmap, decision, project note,
generated file, or a task-card update with concrete findings.
