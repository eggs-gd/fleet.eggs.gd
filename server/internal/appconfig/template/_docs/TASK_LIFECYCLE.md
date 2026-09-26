# Task Lifecycle

Canonical shape and lifecycle rules for `Work/<project-id>/tasks/*.md`
cards. This is the source of truth for anyone (human, Manager, or worker
agent) reading or writing task files in this tree.

## Project Resolution Precedence

When resolving a phrase to a project/repository:

1. Explicit user instruction.
2. Existing `Work/<project-id>/PROJECT.md` id, title, and aliases.
3. Repository aliases recorded in the workspace card.
4. Exact repository name/path in `_registry/repositories.json`.
5. Workspace group relationship in `_registry/project-groups.json`.
6. High-confidence fuzzy match.
7. Ask the operator, or park in `backlog`/Inbox.

When a new alias is learned with high confidence, persist it in the
workspace card instead of keeping it only in chat.

## Assignee And Launch Agent

`assignee` is the canonical worker:

```yaml
assignee: codex
```

Humans and AI agents are both workers. Valid assignees are:

```text
unassigned
claude, codex, cursor, gemini        (AI agents)
<name>                               (a person with a Fleet/<name>.md file)
```

The template ships `Fleet/owner.md`, so `owner` is a valid assignee. Rename
that file to your own name if you prefer. Nothing in the app hardcodes a
person's name. `owner` is only the author label the app puts on comments made
on the person's behalf.

`launch.agent` is deprecated for normal tasks. Leave it empty unless there
is a documented real use case where the launch process must start a
different worker than the canonical assignee.

Effective launch worker:

1. If `launch.agent` is non-empty, use it only as an explicit override and
   make the reason visible in the task card.
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
`schema_version` are treated as schema version `0` until migrated —
existing version `0` cards remain valid input, migration should happen
when a card is naturally edited, and a missing `schema_version` is legacy
metadata, not a launch blocker by itself.

Required frontmatter for schema version `1`:

```yaml
schema_version: 1
id: work-<yyyy-mm-dd>-<slug>
ref: <TAG>-<number>
title: Human-readable task title
type: feature | bug | research | review | maintenance | decision
status: backlog | needs_rework | todo | doing | blocked | needs_review | done | archived
priority: 1 | 2 | 3 | 4 | 5
project: <project-id>
repositories:
  - <relative/repository/path>
depends_on:
  - <TAG>-<number>   # optional; hard launch gate (see below)
assignee: <person> | codex | claude | cursor | gemini | unassigned
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
legacy `type: todo` values normalize to `feature`.

Runtime/API may expose derived fields that are not frontmatter, including
`workspace_id` and `project_id`. Do not hand-edit derived fields into task
cards.

## Refs And Project Tags

Every project has a short **tag** in its `Work/<project-id>/PROJECT.md`
frontmatter, and every task ref is `<TAG>-<number>`:

```yaml
tag: "FLET"
```

- The scanner gives a project without a tag one: four letters made from its id
  (first letters, initials, consonants), or a seeded sequence of letters when
  those are taken. The same project always gets the same tag.
- Change it by editing `tag:` in the card: 2 to 6 capital letters or digits,
  starting with a letter, and not `INBOX`. Tags are unique across projects. A
  duplicate or malformed tag is reported as a problem and left as written.
- Numbers count per tag in `_registry/counters.json` and are never reused. A
  task keeps its ref if the project's tag changes later. Only new tasks use the
  new tag.
- Refs created before tags existed use the legacy `CORE-` prefix and stay valid.
- `INBOX-<number>` is the global Inbox ref space.

### Task dependencies (`depends_on`)

Optional frontmatter list of prerequisite tasks, treated as a hard gate,
not advisory Markdown prose:

```yaml
depends_on:
  - FLET-144
```

Accepted dependency tokens:

- human refs such as `FLET-144` (case-insensitive);
- canonical task `id` values;
- opaque task locators / relative paths when useful internally.

A dependency is satisfied only when the prerequisite task status is
`done`. `needs_review` is **not** enough — the prerequisite must be
accepted before dependents may auto-launch. Missing or self-referential
dependencies stay unresolved, and keep the dependent task in its pickup
status (`todo` / `needs_rework`) rather than flipping it to
`status: blocked`. See `Fleet/LAUNCH_POLICY.md`.

Operators can set or clear `depends_on` from a dashboard (task modal and
create form) — that writes the same frontmatter field, so dependency gates
are not limited to agent-authored Markdown prose.

## Task State Machine

Canonical statuses:

| Status | Meaning |
|---|---|
| `backlog` | Structured task, not prioritized for worker pickup. |
| `needs_rework` | Failed review; unfinished work returned to the worker queue before normal todo. |
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
- Moving `needs_review -> needs_rework` means the operator rejected the
  produced artifacts and the same task needs fixes. This is not a new task.
- `needs_rework` is picked before normal `todo` for the same assignee.
- Manually woken workers may move their claimed task to `doing`, `blocked`,
  or `needs_review`.
- Daemon-launched workers report a result payload and leave the final
  lifecycle transition to whatever orchestrates them after execution
  reaches a terminal state.
- Bot-produced work must stop at `needs_review`; the operator moves it to
  `done`.
- Returning `needs_review -> todo` must add a `## Review Comments` entry.
- `todo -> doing` is the durable task-level lock before real agent launch.
- `doing -> needs_rework` is allowed for manual/orphan resolution when a
  restarted runtime cannot continue the previous execution and the
  operator wants the same worker to retry with review context.
- `doing -> done` is allowed only for operator resolution of orphaned or
  externally completed work. Bot-produced work still stops at
  `needs_review`.
- `blocked -> needs_review` is an operator/manual recovery path only: when
  the operator has verified that physical artifacts already exist (or the
  blocker was infrastructure/visibility rather than missing work) and
  wants review without a fake re-execution. Prefer a `## Review Comments`
  reason. Automated finalizers must not use this transition to hide failed
  executions — they only finalize from `doing`.
- `blocked -> needs_rework` / `blocked -> todo` are the normal retry paths
  after fixing the blocker.
- `done -> archived` is retention cleanup, not task completion.
- `archived -> backlog` is the only reopen path from archive.

## Generated Views

Canonical task cards live under `Work/<project-id>/tasks/`.

Generated views such as `Work/INDEX.md` are projections. Managers and
agents must not hand-maintain generated indexes — normal task edits update
canonical task cards, and whatever tool reads this tree refreshes derived
views. `make rebuild-index` (or equivalent) is for startup, recovery, and
explicit maintenance.

## Priority And Pickup Order

- `1` is the highest priority, `5` the lowest.
- New tasks default to `priority: 5`. A missing or invalid priority is
  treated as `5`.
- `needs_rework` is picked before `todo` for the same assignee, even when
  the `todo` task has a numerically higher priority.
- Within the same status and assignee queue, pickup order is priority
  ascending, then natural ref order (the number within a tag).
- Workers do not pick up `backlog` tasks. A worker opens a `todo` task; if
  the context is not enough, it sets `blocked`, writes the missing question
  into the task card (not only into chat), and asks the operator.
- Done work stays under `Work/<project-id>/tasks/` with `status: done` or
  `archived`. `Archive/` is for standalone historical material that is not
  an Inbox or Work item.

## Definition Of Done

A task is not complete if the only result is a chat response.

Every task must produce or update at least one physical artifact in this
tree or the target repository before it can move to `needs_review`/`done`.
Valid artifacts:

- code changes in the target repository;
- tests, fixtures, migrations, or config changes;
- updates to README, docs, roadmap, specs, or architecture notes;
- a new or updated decision record or project note;
- a refined task card with concrete findings and next steps;
- generated output saved to a file.

For research, investigation, architecture, planning, or documentation tasks
the artifact may be Markdown. The worker writes the conclusion somewhere
durable: `Work/<project-id>/notes/`, `Work/<project-id>/decisions/`, project
docs, or the task card itself.

When a task reaches `needs_review`, `## Deliverable` lists the changed files
or created artifacts. With no artifact the task stays `doing` or `blocked`.

AI workers do not move their own implementation tasks to `done`. The
operator is the default closer.

How a worker reaches `needs_review` or `blocked` depends on how it started:

- A manually woken worker has direct filesystem access to its target
  repository and may set `status: needs_review` on its own claimed task once
  real artifacts exist (`OPERATING_MODEL.md`, Manual Wakeup Mode).
- A daemon-launched worker never edits its own or any canonical task card.
  It reports the outcome through a compact result payload
  (`../Fleet/LAUNCH_POLICY.md`), and the runtime applies the transition. A
  task-shaped file in the target repository is a debug artifact, never
  canonical state.

When a task returns from `needs_review` to `needs_rework`, a `## Review
Comments` entry says what must change. Workers treat review comments as
active instructions.

A task assigned to a person is closed by that person's own confirmation and
needs no artifact. The artifact rule governs work done by AI workers.
