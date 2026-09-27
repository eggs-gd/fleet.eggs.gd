# Domain model

Workspaces, projects, repositories, the task schema and the status machine.
The Manager tools, the dashboard, the launcher and the finalizer all follow
this file instead of defining their own lifecycle. The write path is described
in [ARCHITECTURE](ARCHITECTURE.md).

## Ownership

| Term | Meaning | Source |
|---|---|---|
| Workspace | A durable area `Work/<workspace-id>/` with project context, tasks, notes and decisions. | `Work/<workspace-id>/PROJECT.md` |
| Project | The owner of a task. By default the workspace id. | task `project` |
| Repository | A local git repository a task may touch. | task `repositories`, `_registry/repositories.json` |

Tasks live in the project's workspace folder: `Work/<project-id>/tasks/*.md`.
The API exposes `workspaces[]` and `projects[]`. For a `workspace_group`
workspace, Fleet derives one child project per repository in the workspace
card, for example workspace `acme`, project `acme/web`.

Every task has one owning project and a list of repositories the worker may
read or edit:

1. `project` is the canonical owner.
2. `repositories` lists concrete repositories. More than one only when the work
   really crosses repositories. Automatic launch needs exactly one.
3. If the project is known but the repository is not, keep the task in
   `backlog` or `blocked` and write the question in the card.
4. If neither can be resolved, keep the item in the Inbox or create a
   `backlog` task with `assignee: unassigned`.
5. Fleet never guesses a missing project or repository.

### Resolving a phrase to a project

1. Explicit instruction.
2. `Work/<id>/PROJECT.md` id, title and `aliases`.
3. Repository aliases in the card.
4. Exact repository name or path in `_registry/repositories.json`.
5. Workspace group relationship in `_registry/project-groups.json`.
6. A high-confidence fuzzy match.
7. Ask, or park in `backlog` or the Inbox.

A newly learned alias is stored in the card (`manager_project`), not left in
chat.

## Assignee

`assignee` is the worker: `unassigned`, an agent (`claude`, `codex`, `cursor`,
`gemini`), or a person who has `Fleet/<name>.md` (the names `readme`,
`routing` and `launch_policy` are reserved). Comments written by the person
who owns the data root carry the author `owner`.

Precedence when routing: explicit instruction, `default_assignee` in
`PROJECT.md`, Claude, a free backup worker. `manager_route` implements it;
`Fleet/ROUTING.md` is the human-readable version.

`launch.agent` overrides the assignee for launching only. Leave it empty
unless the launching worker must differ from the assignee, and say why in the
card.

## Task schema

```yaml
schema_version: 1
id: work-<yyyy-mm-dd>-<slug>
ref: <TAG>-<number>
title: Human-readable title
type: feature | bug | research | review | maintenance | decision
status: backlog | needs_rework | todo | doing | blocked | needs_review | done | archived
priority: 1 | 2 | 3 | 4 | 5
project: <project-id>
repositories:
  - <relative/repository/path>
depends_on:
  - <TAG>-<number>      # optional
assignee: <worker>
assignment_reason: <short reason>
source: voice | text | url | file | chat | manager
source_inbox: <inbox id or empty>
created_at: <ISO-8601>
updated_at: <ISO-8601>
launch:
  agent:
  mode:
  auto_commit: false
  auto_push: false
  auto_pr: false
```

`todo` is a status, not a type. A card without `schema_version` reads as
version 0 and stays valid. `workspace_id` and `project_id` are derived by the
runtime and are not written into cards.

### `depends_on`

A hard launch gate. Tokens are refs (case-insensitive), task ids, or relative
task paths. A prerequisite counts only when it is `done`; `needs_review` is not
enough. A missing or self-referential dependency stays unresolved. An
unresolved dependency keeps the task in its pickup status with a failed
`depends_on` gate. It does not turn the task `blocked`. The dashboard and
`manager_update` write the same field.

## Status machine

| Status | Meaning |
|---|---|
| `backlog` | Structured, not prioritized for pickup. |
| `needs_rework` | Review failed; returned to the worker queue ahead of `todo`. |
| `todo` | Ready for pickup. |
| `doing` | A worker has it. |
| `blocked` | Cannot proceed: missing information, approval or access, or a failed launch. |
| `needs_review` | A worker produced artifacts; a person reviews before `done`. |
| `done` | Reviewed and accepted. |
| `archived` | No longer relevant; kept for history. |

Allowed transitions (`taskflow.AllowedStatusTransition`):

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

Rules:

- `backlog → todo` is the readiness signal. There is no separate flag.
- `needs_review → needs_rework` means the artifacts were rejected and the same
  task needs fixes. It is not a new task. Returning to `todo` or
  `needs_rework` adds a `## Review Comments` entry.
- `todo → doing` is the durable lock taken before a real launch.
- A daemon-launched worker reports an outcome. Only the finalizer changes the
  status, and AI work stops at `needs_review`; a person moves it to `done`.
- `doing → done` and `blocked → needs_review` are recovery paths for a person
  (orphaned or externally completed work, or a blocker that was
  infrastructure). The automated finalizer finalizes only from `doing`.
- `done → archived` is retention. `archived → backlog` is the only reopen.

## Generated views

`Work/INDEX.md` and `_registry/*` are projections. Nobody edits them by hand;
Fleet refreshes them after task and workspace changes. `fleet rebuild-index`
rebuilds the index for recovery.

## Definition of done

A chat reply is not a result. An AI worker's task needs a physical artifact in
the data root or the target repository (code, docs, tests, a decision, a note,
or a card update with concrete findings) before it can be `needs_review`. A
task assigned to a person is closed by that person and needs no artifact.
