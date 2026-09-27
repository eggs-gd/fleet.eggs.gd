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
```

`PROJECT.md` is the project card: its id, title, ref tag, aliases, default
assignee and repositories. `tasks/` contains executable work items.

Fleet manages work, not knowledge. What a project is, how it is built and why
lives in the project's own repository. A decision that one task needs is
written in that task's `## Context`. Nothing else is kept per project.

Raw capture starts in `Inbox/items/`.

See `_docs/TASK_LIFECYCLE.md` for the task schema and lifecycle,
`_docs/templates/WORK_ITEM.md` for the task template, and `_docs/MANAGER.md`
for what the Manager tools own. Slugs, refs, and aliases are handled by those
tools.
