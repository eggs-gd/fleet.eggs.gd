# Work

Project and workspace operating area.

Each project/workspace lives in:

```text
Work/<project-id>/
```

Expected layout:

```text
Work/<project-id>/
  PROJECT.md
  tasks/
  notes/
  decisions/
  inbox/
```

`PROJECT.md` describes the workspace and its repositories. `tasks/` contains
executable work items. `notes/` contains project-specific knowledge that is not
yet a task. `decisions/` contains durable decisions and rationale. `inbox/`
contains project-specific raw material after the project is already known.

Global raw capture still starts in `Inbox/items/` when the project is unknown or
the input is ambiguous.

See `_docs/MANAGER.md` for the exact task format, slug rules, alias capture
rules, index rules, and handoff rules. Use `_docs/templates/WORK_ITEM.md` for new
tasks.
