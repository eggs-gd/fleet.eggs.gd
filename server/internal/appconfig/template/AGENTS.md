# Agent Instructions — Data

This file is the canonical agent instruction file for this repository —
one operator's own Inbox/Work/Fleet/Archive content and generated
`_registry` state. It is plain data: no source code, no build tooling, and
no assumption about what reads or writes it, where that tool is installed,
or how. The one thing that runs alongside it is MCP servers configured for
operator use (see `.mcp.json`, if present) — those must stay visible and
editable here, because the operator, not any app's source tree, owns them.

This is a personal AI operating system bootstrap. Treat this directory as an
orchestration workspace first: the early value is deterministic structure,
project `_registry` state, and clear handoff rules between humans and
agents.

## Current Scope

The MVP is intentionally small:

- keep raw inputs in `Inbox/`;
- keep project/workspace definitions and executable work in `Work/`;
- keep worker/agent definitions in `Fleet/`;
- keep closed or ignored material in `Archive/`;
- keep generated machine state in `_registry/`;
- keep operating notes for this data root in `_docs/`.

Do not build a UI, task manager, voice interface, or autonomous manager
here — this tree holds data, not code, and none of that belongs in it
regardless of what the `_registry` looks like.

## Ground Rules

1. Determinism first. If a value can be read from the filesystem, git, README,
   or explicit metadata, read it instead of guessing.
2. Registry state is the source of truth for project discovery. Agents should
   use it to decide which repository to open or modify.
3. Repository is not the same as Project. A Project may own many repositories,
   and related repositories should be suggested for grouping before being merged
   into one logical project.
4. Suggestions are not decisions. Duplicate detection and grouping can produce
   candidates, but should not silently rewrite project identity.
5. Incremental sync should mark missing, moved, renamed, and newly discovered
   repositories. Do not delete historical _registry entries automatically.
6. Keep generated data recognizable. Generated _registry files should include
   timestamps and source information where useful.
7. Keep personal data local. Do not push secrets, private notes, raw voice
   dumps, tokens, or machine-specific transient files.
8. Keep architecture light until the _registry proves the next need.
9. The Manager may be a Claude, Codex, Cursor, or Gemini session whose
   folder is this data root. The conversation stays in that provider's app.
   Turn user input into Inbox, Work, Project, Fleet, or Archive changes with
   the Manager skills in `.agents/skills` and the Fleet MCP tools. Do not
   hand-parse or hand-edit task Markdown. `_docs/MANAGER.md` is the human
   reference for the Markdown shape, not the Manager's instructions.
10. Chat output is not a completed task. Every task an AI worker does must
    produce or update a physical artifact in Data or the target repository
    before it can be marked `needs_review` or `done`. A task assigned to a
    person is closed by that person and needs no artifact.
11. Before launching or picking up agent work, check `Fleet/LAUNCH_POLICY.md`.
    During the MVP, do not run more than one agent against the same project or
    repository.

## Directory Contracts

| Path | Contract |
|---|---|
| `Inbox/` | Raw input only. Anything here may be unstructured and untrusted. |
| `Work/` | Project/workspace folders with `PROJECT.md` and `tasks/`. |
| `Fleet/` | Worker definitions and capabilities. Fleet does not own tasks. |
| `Archive/` | Closed or intentionally ignored material. Keep provenance when useful. |
| `_registry/` | Generated or synced discovery state. Prefer machine-readable formats. |
| `_docs/` | Operating notes for this data root. |

## Manager Mode

When this session is the Manager:

- You are the Manager. Work only through Fleet MCP tools.
- Do not edit board files yourself.
- Skills live in `.agents/skills`.
- Before or after each request, call `manager_events` for anything new
  (starting from the last id you saw this session, or 0 on your first call).
  Fleet pushes nothing into this session on its own; this is the only way you
  hear about a task needing attention or review, or a project appearing,
  losing a repository, or dropping out of the registry. Mention what is
  urgent before answering the request; mention the rest after.

## Bootstrap Direction

The scanner walks the configured scan root while pruning noisy
dependency/build folders such as `node_modules`, `.venv`, `venv`, `target`,
`Library`, `Temp`, `bin`, and `obj`.

For each git repository, collect at least:

- canonical local path;
- git remote URL;
- current branch;
- default README path and first useful summary;
- detected stack markers;
- parent/child repository relationship;
- first seen and last seen timestamps.

Use normalized remote URL as the preferred stable repository identity. If a
repository has no remote, use a deterministic path-based fallback identity.

## Verification (bootstrap / documentation changes)

For documentation-only changes, inspect `git diff` and make sure the structure
still matches `README.md`.

For scripts or bootstrap code, run the most focused local command available and
include the exact command in the final report.
