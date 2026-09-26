# Personal Space

Core has a first-class personal workspace at `Work/_life/`.

This workspace is for life management material that is not naturally a software
repository or external project.

## Why It Lives In Core

Keep Life inside Core for now.

Reasons:

- Core is already the operating system for capture, triage, refs, tasks, and agents.
- Obsidian can point at the whole Core repository if needed. Obsidian and a separate LifeOS folder below are generic examples, not a particular machine.
- Splitting Life into a separate repository too early would create duplicate
  routing, refs, and promotion rules.
- A separate repository can still be created later if privacy, volume, or tooling
  makes it useful.

## Folder Model

```text
Work/_life/
  PROJECT.md
  inbox/
  tasks/
  notes/
  ideas/
  reminders/
  decisions/
  areas/
```

## Ref Model

Use three human-facing ref spaces:

| Ref | Meaning |
|---|---|
| `INBOX-*` | Raw global captures in `Inbox/items/`. |
| `CORE-*` | Actionable Work tasks across all workspaces, including Life. |
| `LIFE-*` | Personal notes, ideas, reminders, and decisions. |

When the user says "задача 10", resolve it as `CORE-10`.

When the user says "інбокс 10", resolve it as `INBOX-10`.

When the user says "лайф 10" or "life 10", resolve it as `LIFE-10`.

## Promotion Examples

Raw personal thought:

```text
Inbox/items/2026-07-31-think-about-moving.md
ref: INBOX-1
status: untriaged
```

Promoted personal idea:

```text
Work/_life/ideas/2026-07-31-moving-neighborhood.md
ref: LIFE-1
source_inbox: INBOX-1
```

Actionable follow-up task:

```text
Work/_life/tasks/2026-07-31-research-neighborhoods.md
ref: CORE-10
source_life: LIFE-1
status: backlog
```

## Obsidian

The first Obsidian experiment should point at the whole Core repository.

If that becomes noisy, try opening only `Work/_life/` as a vault before creating a
separate Life repository.

## Split-Out Rule

Do not create `/Users/operator/Projects/LifeOS` yet.

Revisit this if:

- Life content becomes too private for the Core repo;
- Obsidian needs a cleaner vault root;
- personal notes become much larger than project/task metadata;
- a dedicated personal knowledge workflow appears.
