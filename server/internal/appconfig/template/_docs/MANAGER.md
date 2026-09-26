# Manager Reference

> **Reference, not instructions.** The Manager is a Claude, Codex, Cursor, or
> Gemini session whose folder is this data root. It works through Fleet MCP
> tools (`manager_*`) and the skills in `.agents/skills`. The tools own ids,
> refs, slugs, statuses, priority defaults, Inbox files, project aliases,
> routing, and validation. Do not hand-edit task Markdown or follow the shape
> below by hand.

Chat history is not a source of truth. Durable results are files in this tree.

## What Owns What

| Concern | Where it lives |
|---|---|
| Task refs (`TAG-N`, the tag comes from the project card) and `INBOX-N`, monotonic, never reused | `_registry/counters.json`, allocated by the tools |
| Task ids, filenames, slugs, defaults, dependency cycles | task tools (`manager_command`, `manager_validate`) |
| Status meanings, transitions, priority order, Definition Of Done | `TASK_LIFECYCLE.md` |
| Project resolution and aliases | `manager_resolve_project`, `manager_project` |
| Assignee choice | `manager_route`; `Fleet/ROUTING.md` is the human reference |
| Inbox capture and promotion | `manager_inbox` |
| Judgment: when to ask, how to split, what counts as ready | skills in `.agents/skills` |
| Worker pickup and daemon assumptions | `OPERATING_MODEL.md` |
| Project cards (`Work/<id>/PROJECT.md`) and `_registry/counters.json` | created and repaired by the scanner; the Manager tools never create them |
| `Work/INDEX.md` | generated; never edited by hand |

## Where State Lives

| State | Location |
|---|---|
| Raw capture | `Inbox/items/<yyyy-mm-dd>-<slug>.md`, `status: untriaged`, `ref: INBOX-<n>` |
| Promoted capture | same file, `status: promoted`, `promoted_to: <ref>`, one Activity Log line |
| Task | `Work/<project-id>/tasks/<yyyy-mm-dd>-<slug>.md`, template `_docs/templates/WORK_ITEM.md` |
| Project card | `Work/<project-id>/PROJECT.md`; `aliases` in frontmatter, Activity Log below |
| Project note / decision | `Work/<project-id>/notes/`, `Work/<project-id>/decisions/` |
| Repository paths | `_registry/repositories.json` |
| Cleanup exclusions | `_registry/protection-rules.json` |

A task belongs to exactly one `Work/<project-id>/tasks/` folder. Promotion
never moves the Inbox file.

An Inbox item carries `## Raw Input` and `## Activity Log`. A task card
carries the sections in `WORK_ITEM.md`. Rough wording from voice or text is
always preserved in `## Raw Input`.

## Determinism Rules

1. Use `_registry` and workspace cards for paths. Do not invent local
   repository paths.
2. Use absolute timestamps with timezone.
3. Prefer Markdown with YAML frontmatter for human-editable state.
4. Never delete, move, or archive a repo from a cleanup candidate without an
   explicit user confirmation naming the path.
5. Never mark a draft workspace card reviewed unless the user confirms it.
6. Activity Logs are append-only.
7. If uncertain, park the task in `backlog` with the missing questions in
   `## Handoff` instead of guessing.
