# Manager Operating Rules

> **Migration (CORE-92):** Codex chat is no longer the default Manager for
> non-coding voice/task ingestion. Prefer a structured Manager API surface
> (text/audio + deterministic fast path + structured LLM). Codex / Claude /
> Cursor remain **worker** agents for execution. This file remains the
> canonical Markdown shape, refs, and Definition Of Done that any Manager
> implementation must produce.

This document defines the durable Manager rules for this tree. Historically
the active Codex chat acted as the AI Manager. Capture now targets a
structured Manager API surface, which writes the same durable Markdown
state into this repository.

See `_docs/OPERATING_MODEL.md` for the broader MVP flow, manual wakeup mode,
daemon direction, and no-auto-commit policy.

See `TASK_LIFECYCLE.md` for canonical workspace/project/repository terms,
task schema, project resolution precedence, assignee precedence, and task
state transitions.

## Role

The Manager (Core Manager API, or Codex chat only as an explicit override) is
responsible for turning user input into durable Core artifacts:

- Inbox items;
- Work items;
- Work workspace updates;
- Fleet notes;
- Archive entries;
- task context for another agent or human.

The Manager may reason, clarify, and classify, but the durable result must be
written to files. Chat history is not a source of truth.

## Source Of Truth

Use these files in order:

1. `TASK_LIFECYCLE.md` for canonical schema, ownership, and lifecycle rules.
2. `Work/INDEX.md` and `Work/<project-id>/PROJECT.md` for project identity.
3. `_registry/repositories.json` for exact repository paths/remotes/branches.
4. `_registry/project-groups.json` for workspace/group relationships.
5. `_registry/protection-rules.json` for layouts that must not be treated as
   cleanup/deletion candidates.
6. `Work/<project-id>/tasks/*.md` for active work.
7. `Inbox/items/*.md` for raw captured input.
8. `_docs/OPERATING_MODEL.md` for worker pickup and daemon assumptions.

If chat context contradicts repository files, update the files or ask before
acting. Do not let important project knowledge live only in chat.

## Input Handling

Every meaningful user input should end in one of these outcomes:

| Outcome | Use when |
|---|---|
| `Inbox` | The input is raw, ambiguous, partial, emotional, exploratory, or not yet actionable. |
| `Work` | The input clearly asks for something to be done. |
| `Project update` | The input clarifies ownership, grouping, paths, protection rules, or project meaning. |
| `Archive` | The input is explicitly closed, obsolete, rejected, or only kept for history. |
| `No file change` | The input is casual, asks a direct question, or explicitly says not to persist it. |

If the user gives a rough voice instruction that contains a task, create a Work
item directly. Preserve the raw transcript inside the Work item instead of
forcing a separate Inbox item.

## State Flow

Core stores project context and project work together under `Work/<project-id>/`.

Workspaces are context and identity. Task files are the task source of truth.
Inbox items are raw capture source of truth.

The flow is:

```text
User input
  -> Inbox item, if raw/ambiguous
  -> Work item, when actionable
  -> Work/<project-id>/PROJECT.md context
  -> Archive status, when closed/obsolete
```

Physical locations stay stable:

| State | File location | What changes |
|---|---|---|
| Raw capture | `Inbox/items/*.md` | `status: untriaged` |
| Promoted capture | `Inbox/items/*.md` | `status: promoted`, `promoted_to: <work-id>` |
| Actionable task | `Work/<project-id>/tasks/*.md` | `status`, `project`, `repositories`, `assignee` |
| Workspace metadata | `Work/<project-id>/PROJECT.md` | project meaning, repo list, notes |
| Project note | `Work/<project-id>/notes/*.md` | project-specific non-task knowledge |
| Project decision | `Work/<project-id>/decisions/*.md` | durable decisions and rationale |
| Archived item | original file location | `status: archived` plus archive log |

Do not create a second global task copy. A task belongs to exactly one
`Work/<project-id>/tasks/` folder.

## Promotion Rules

Promotion means creating a new Work item from an Inbox item and linking them.
It does not mean moving the Inbox file.

When promoting:

1. Keep the original Inbox file in `Inbox/items/`.
2. Change its frontmatter:
   - `status: promoted`
   - `promoted_to: work-...`
3. Append an `## Activity Log` entry with timestamp and Work item path.
4. Create a Work item in `Work/<project-id>/tasks/`.
5. In the Work item, set `source_inbox: <inbox-id>` when applicable.
6. Copy the original raw input into `## Raw Input`.
7. Resolve `project` and `repositories` using Work workspace metadata/registry.

If input is already clearly actionable, the Manager may skip a separate Inbox
file and create Work directly. In that case:

- `source: voice | text | url | file`
- `source_inbox:` is omitted or empty;
- `## Raw Input` still preserves the original wording.

## Work Lifecycle

Work item status changes in place. The allowed transition matrix is canonical
in `TASK_LIFECYCLE.md`.

Use `backlog` when a task is already normalized into a Work item, but the operator has
not prioritized it for execution yet. Backlog is not raw Inbox. It is a real
task card parked before the active queue.

Workers must not pick up `backlog` tasks during normal manual wakeup. A backlog
task becomes executable when the operator or the Manager changes it to `todo`, or when
the user explicitly asks a worker to take that specific backlog item.

Use `todo` when the task is approved for the active queue and can be picked by
the assigned worker.

Use `needs_rework` when the operator reviewed `needs_review` artifacts and rejected the
result. This is not a new task; it is the same unfinished task returned to the
front of the worker queue. The review comment is the active rework instruction.

Use `blocked` when the next action needs missing information, approval, or an
external state change.

Use `needs_review` when an agent produced physical repository/Core artifacts
that the operator needs to review before the work is accepted or committed.

Use `done` when the requested work is complete.

Use `archived` when the work is no longer relevant, rejected, duplicated, or
intentionally kept only for history.

Do not move done work into `Archive/` during the MVP. Keep it under
`Work/<project-id>/tasks/` with `status: done` or `status: archived`.
`Archive/` is for standalone historical material that is not naturally an Inbox
or Work item.

## Definition Of Done

A task is not complete if the only result is a chat response.

Every task must produce or update at least one physical artifact before it can
move to `needs_review` or `done`.

Valid artifacts include:

- code changes in the target repository;
- tests, fixtures, migrations, or config changes;
- updates to README, docs, roadmap, specs, or architecture notes;
- a new or updated decision record;
- a new or updated project note;
- a refined task card with concrete findings and next steps;
- generated output saved to a file.

For research, investigation, architecture, planning, or documentation tasks, the
artifact may be Markdown. The worker must write the conclusion somewhere durable
such as `Work/<project-id>/notes/`, `Work/<project-id>/decisions/`, project docs,
or the task card itself.

When moving a task to `needs_review`, update `## Deliverable` with the changed
files or created artifacts. If there is no artifact, keep the task in `doing` or
`blocked`.

AI workers must not move their own completed implementation tasks to `done`.
The operator is the default closer for bot-produced work.

How a worker reaches `needs_review` (or `blocked`) depends on how it was
started (CORE-97):

- A manually woken worker has direct filesystem access to its target
  repository and may set `status: needs_review` on its own claimed task
  itself once real artifacts exist, per `_docs/OPERATING_MODEL.md`'s
  Manual Wakeup Mode.
- A daemon-launched worker must not touch its own (or any) canonical task
  card's status/frontmatter at all. It reports outcome exclusively through
  a compact result payload (see `../Fleet/LAUNCH_POLICY.md`); whatever
  runtime launched it reads that payload and applies the actual status
  transition. If a worker writes a task-shaped file into the target
  repository instead, that file is a debug artifact only — it is never
  treated as canonical task state.

When moving a task from `needs_review` to `needs_rework`, add a `## Review
Comments` entry explaining what must change. Workers must treat review comments
as active instructions, not historical noise.

## Work Item Creation

Work items live in:

```text
Work/<project-id>/tasks/<yyyy-mm-dd>-<slug>.md
```

Use the resolved project id as the directory name. This keeps project context,
tasks, notes, and decisions together.

Examples:

```text
Work/eggs-gd-prod/tasks/2026-07-31-career-wizard-vacancy-statuses.md
Work/audiophile/tasks/2026-08-01-review-jivemax-build-notes.md
Work/core-eggs-gd/tasks/2026-08-02-add-work-index-generator.md
```

Use the template:

```text
_docs/templates/WORK_ITEM.md
```

Required frontmatter:

```yaml
schema_version: 1
id: work-<yyyy-mm-dd>-<slug>
ref: CORE-<number>
title: ...
type: feature | bug | research | review | maintenance | decision
status: backlog | needs_rework | todo | doing | blocked | needs_review | done | archived
priority: 1 | 2 | 3 | 4 | 5
project: <project-id>
repositories:
  - <relative repo path>
depends_on: [] # optional hard launch gate; see TASK_LIFECYCLE.md
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

`assignee` is the canonical worker. `launch.agent` should normally be empty and
is only an explicit launch override when the task card documents why launch must
start a different worker than `assignee`.

Optional `depends_on` lists prerequisite task refs (for example `CORE-144`). The
daemon will not claim or launch the task until every listed prerequisite is
`status: done`. Body prose like "run after CORE-N" is not enforced — use
frontmatter, or set the same field from the backoffice dashboard task modal /
create form. Details: `TASK_LIFECYCLE.md` and `../Fleet/LAUNCH_POLICY.md`.

Priority rules:

- `1` is highest priority.
- `5` is lowest priority.
- New Work items default to `priority: 5`.
- Missing or invalid priority is treated as `5` by Core runtime.
- `needs_rework` is picked before `todo` for the same assignee, even when the
  `todo` task has a numerically higher priority.
- Within the same status/assignee queue, pickup order is priority ascending,
  then natural `CORE-*` ref order.

Required sections:

- `## Request` — normalized user request.
- `## Raw Input` — original transcript/text when available.
- `## Project Resolution` — why the Manager picked this project/repository.
- `## Acceptance Criteria` — concrete expected outcomes, even if draft.
- `## Context` — relevant project/card/registry links.
- `## Deliverable` — expected and produced physical artifacts.
- `## Handoff` — what another agent should do next.
- `## Review Comments` — reviewer notes for returned work.
- `## Activity Log` — append-only notes.

## Human-Facing Refs

Every Work item must have a short human-facing `ref` in frontmatter:

```yaml
ref: CORE-10
```

This ref is for voice and chat usage. When the user says "задача 10", "таска
10", or "CORE-10", the Manager should resolve it to `ref: CORE-10` and open the
matching Work item before discussing or changing it.

Rules:

1. `ref` is Core-wide for Work items, not project-local.
2. Refs are monotonically increasing and must not be reused.
3. The next value lives in `_registry/counters.json`.
4. `id` and filename remain stable machine/file identifiers.
5. `ref` is the short human handle shown in indexes and backoffice.
6. If the user gives only a number, resolve it as `CORE-<number>` by default.
7. If the user explicitly says "інбокс 10", resolve it as `INBOX-10`.
8. If the ref is ambiguous or missing, search `Work/INDEX.md` and ask only when
   no deterministic match exists.

Raw Inbox items use a separate ref space:

```yaml
ref: INBOX-10
```

Personal notes, ideas, reminders, and decisions in `Work/_life/` use:

```yaml
ref: LIFE-10
```

When an Inbox item is promoted to Work, keep the original `INBOX-*` ref on the
Inbox file and create a new `CORE-*` ref on the Work item. Link them with
`source_inbox` and `promoted_to`.

## Work ID And Slug Rules

Work item ids and filenames must be deterministic and human-readable.

Use:

```text
id: work-<yyyy-mm-dd>-<slug>
ref: CORE-<number>
file: Work/<project-id>/tasks/<yyyy-mm-dd>-<slug>.md
```

Slug rules:

1. Start from the normalized task title.
2. Lowercase.
3. Transliterate only when obvious; otherwise keep stable ASCII words from the
   project/repository/request.
4. Replace spaces and punctuation with `-`.
5. Remove filler words and profanity unless they are essential to the task.
6. Keep the slug short enough to scan, usually 3-8 words.
7. Include a project/repository hint when the title could collide.
8. If the target file already exists, append `-2`, `-3`, etc.

Example:

```text
Title: Show vacancy statuses in the Career Wizard web UI
Project: eggs-gd-prod
Repository: eGGs.gd.prod/career-wizard
id: work-2026-07-31-career-wizard-vacancy-statuses
ref: CORE-10
file: Work/eggs-gd-prod/tasks/2026-07-31-career-wizard-vacancy-statuses.md
```

## Project Resolution Rules

Canonical resolution precedence lives in `TASK_LIFECYCLE.md`. Manager uses
that precedence and records the concrete outcome in `## Project Resolution`.

When the user mentions a project name:

1. Search workspace ids and titles first.
2. Search repository names and relative paths second.
3. Prefer a reviewed workspace card over a draft card when both exist.
4. Prefer exact matches over fuzzy matches.
5. If a workspace group owns the repo, record both:
   - `project: <workspace-project-id>`;
   - specific `repositories: [...]`.
6. If confidence is low, keep the Work item in `backlog`, set
   `assignee: unassigned`, and ask for confirmation before moving it to `todo`.

Examples:

- "Career Wizard" resolves to `project: eggs-gd-prod` and repository
  `eGGs.gd.prod/career-wizard`.
- "BubblettHell sonar" resolves to `project: bubbletthell-sonar` and repository
  `BubblettHell_Sonar`.
- "Audiophile" resolves to workspace project `audiophile`; nested repositories
  under it are intentional unless the user says otherwise.

## Alias Capture Rules

When the Manager successfully resolves a user phrase to a workspace or
repository, persist useful aliases.

Persist an alias when all are true:

1. The phrase came from the user.
2. The phrase is likely to be reused.
3. The resolution confidence is high, or the user confirms it.
4. The alias is not already recorded.

Write aliases into the relevant `Work/<project-id>/PROJECT.md` frontmatter:

```yaml
aliases:
  - Career Wizard
  - career-wizard
```

If the alias points to a specific repository inside a workspace, add a
repository alias note in the workspace card:

```yaml
repository_aliases:
  Career Wizard: eGGs.gd.prod/career-wizard
```

After adding an alias, append an `## Activity Log` entry to the workspace card
with timestamp and source phrase.

## Assignment Rules

Use `Fleet/ROUTING.md` when choosing an assignee.

Assignment precedence:

1. Explicit user instruction.
2. Project override in `Work/<project-id>/PROJECT.md`.
3. Task category fit from `Fleet/ROUTING.md`.
4. Current active agent context.
5. `unassigned`.

Default guidance:

- `codex` if the user asks this Codex chat to do the work now;
- `claude` if the user names Claude, the project is currently Claude-led, or
  the task is product/spec/research/review/decision-heavy;
- `unassigned` if the task should be recorded but not started;
- `owner` if the next action belongs to the human operator;
- `cursor` only if the user explicitly names Cursor until Cursor repo access and
  commit/status rules are documented.

Humans and AI agents are both workers. Do not special-case humans as outside the
system.

Record non-obvious assignment decisions in the Work item's `## Handoff` or
`## Project Resolution`.

Worker pickup is manual in the MVP. Assignee means "who should be able to find
or receive this task", not "a daemon has already delivered it". See
`_docs/OPERATING_MODEL.md`.

## Pickup Readiness

`backlog` is the space for tasks that are written down but not yet ready for
execution. When the operator moves a task from `backlog` to `todo`, that move is the
deterministic readiness signal.

Do not add a separate readiness boolean. Core has one readiness signal:
`status: todo`.

If a worker opens a `todo` task and the context is not sufficient, the worker
must set `status: blocked`, add a clear question or missing-context note to the
task, and ask the operator. The missing context belongs in the task card, not in chat
only.

## Work Index Rules

`Work/INDEX.md` is the generated global work/workspace index.

Managers and workers must not edit `Work/INDEX.md` directly. When the Manager
creates a task, changes a task status, archives a task, or changes an assignee,
the normal write path updates the canonical task card and lets Core runtime or
the shared mutation layer refresh generated views.

The index should keep at least:

- workspace list;
- active tasks grouped by status;
- blocked tasks;
- recently created tasks;
- tasks by assignee when useful.

The deterministic rebuild command is a repair/tooling path for startup,
recovery, and explicit maintenance. It is not part of normal task editing.

## Inbox Rules

Inbox items live in:

```text
Inbox/items/<yyyy-mm-dd>-<slug>.md
```

Required Inbox frontmatter:

```yaml
id: inbox-<yyyy-mm-dd>-<slug>
ref: INBOX-<number>
status: untriaged | promoted | archived
source: voice | text | url | file | manager
created_at: <ISO-8601>
updated_at: <ISO-8601>
promoted_to:
```

Use `INBOX-*` refs for raw captured material that has not become a Work item
yet. When the user says "інбокс 10", resolve it to `ref: INBOX-10`.

If the user says only "задача 10", resolve it to `CORE-10` by default, because
Work refs are the task refs.

Use Inbox when the input is worth remembering but not yet a task.

Required frontmatter:

```yaml
id: inbox-<yyyy-mm-dd>-<slug>
status: untriaged | promoted | archived
source: voice | text | url | file
created_at: <ISO-8601>
promoted_to: <work-id or empty>
```

Required sections:

- `## Raw Input`
- `## Initial Read`
- `## Possible Destinations`
- `## Activity Log`

Inbox status changes in place. Do not move the file after promotion.

## Project Knowledge Updates

When the user corrects project meaning, grouping, or cleanup rules, persist it.

Use:

- `_registry/protection-rules.json` for cleanup exclusions and intentional
  layouts;
- the relevant `Work/<project-id>/PROJECT.md` workspace card for project meaning;
- `_docs/MANAGER.md` only for general Manager rules.

Do not bury project-specific facts in this document unless they are examples.

## Determinism Rules

1. Use _registry/workspace cards for paths. Do not invent local repository paths.
2. Use absolute timestamps with timezone.
3. Prefer Markdown files with YAML frontmatter for human-editable state.
4. Never delete, move, or archive a repo from a cleanup candidate without an
   explicit user confirmation naming the path.
5. Never mark a draft workspace card reviewed unless the user confirms it.
6. Keep Activity Logs append-only.
7. When converting voice/text to Work, preserve the original rough wording.
8. If uncertain, create a `backlog` task with the missing questions in
   `## Handoff` instead of silently guessing.

## Example

Raw voice input:

```text
мужик, додай задачу в Career Wizard, щоб він навчився показувати в вебі статуси вакансій
```

Expected Work item:

```yaml
title: Show vacancy statuses in the Career Wizard web UI
type: feature
status: backlog
priority: 5
project: eggs-gd-prod
repositories:
  - eGGs.gd.prod/career-wizard
assignee: unassigned
source: voice
```

The Work item should link the `eggs-gd-prod` workspace card and explain that
`career-wizard` was selected from the workspace repository list.

Workspace-level exception:

```text
винеси спільні компоненти в kit у eggs.gd-prod
```

Expected Work item:

```yaml
title: Extract shared components into the eggs.gd kit
type: feature
status: backlog
priority: 5
project: eggs-gd-prod
repositories:
  - eGGs.gd.prod/eGGs.gd.kit
assignee: unassigned
source: voice
```

This stays on the `eggs-gd-prod` workspace project only because the requested
work spans shared workspace structure. If the user named one concrete app,
resolve to that app's repository-backed project instead.
