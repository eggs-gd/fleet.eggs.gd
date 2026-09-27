# Fleet Routing Policy

This document helps the Manager choose an assignee for Work items.

Routing is advisory, not absolute. The user's explicit instruction wins.

The Manager is a session of an agent the operator already uses (Claude, Codex,
Gemini, or Cursor opened on this folder), working through the Fleet MCP. Every
agent can also be a **worker** executor for the tasks the Manager creates, but a
worker must not spend credits on routine triage. Cursor as Manager is only this
folder opened in Cursor: Fleet does not start that session, and it is not
reachable from a phone.

See `_docs/OPERATING_MODEL.md` for manual wakeup mode, daemon direction, and
no-auto-commit policy. See `Fleet/LAUNCH_POLICY.md` before actually launching
or picking up agent work.

Canonical task schema, project/repository ownership, and assignee precedence
live in `_docs/TASK_LIFECYCLE.md`. Routing should not redefine those rules.

## Current Fleet

Active mobile-accessible workers:

- `claude` (`app_visible` remote-control sessions)

Explicit opt-in workers:

- `cursor` (`cli_visible` Cursor Agent CLI sessions; not phone/app_visible)

Fleet-control + Codex Remote workers:

- `codex` (`fleet_visible` app-server JSON-RPC; LiveReady for normal daemon
  auto-launch). Operator path is Codex Remote on paired clients
  plus Fleet dashboard continue/interrupt/cancel. Not a Claude-style phone
  remote-control URL, and threads may not appear in ordinary ChatGPT desktop
  sidebar history.

Terminal-resume workers:

- `gemini` (`cli_visible`; Google's Antigravity CLI, binary `agy`. Not yet in
  the Agent Fit Matrix below by design, see
  `Fleet/gemini.md`, which deliberately leaves "Best At"/"Use For" undecided
  until it has a track record on real tasks.)

Cursor has a daemon-visible CLI launch path, but it is CLI-visible rather than
phone/app-visible. Do not make it the default assignee; assign Cursor when the operator
explicitly asks or when a task/project documents why Cursor is the right worker.

Prefer Claude when the operator must open the session from a phone Claude remote-control
URL. Prefer Codex for local coding/repo work that Fleet can launch and control
through app-server + Codex Remote. Do not use Codex as the default Manager for
non-coding voice/task ingestion — use the Fleet Manager API.

## Assignment Precedence

Use this order:

1. Explicit user instruction.
2. Project override.
3. Task category fit.
4. Current active agent context.
5. `unassigned`.

Examples:

- If the user says "give this to Claude", assign `claude`.
- If `Work/acme-web/PROJECT.md` says Billing Portal is currently Claude-led,
  assign `claude` unless the user says otherwise.
- If no project override exists and the task is repo/code implementation,
  prefer `codex`.
- If confidence is low, keep the task in `backlog`, assign `unassigned`, and
  ask the operator to confirm the project/assignee before moving it to `todo`.

## Task Categories

Use one of these task types in Work item frontmatter:

| Type | Meaning |
|---|---|
| `feature` | New user-facing or system capability. |
| `bug` | Broken behavior, regression, failure, or incorrect output. |
| `research` | Investigation, comparison, reading, discovery, spike. |
| `review` | Code/doc/design review, critique, risk finding. |
| `maintenance` | Cleanup, dependency, _registry, docs, housekeeping. |
| `decision` | Record or prepare a durable project/process decision. |

`todo` is a task **status** (ready for pickup), not a type. Legacy
`type: todo` cards are rewritten to `type: feature` during Work index
regeneration.

## Agent Fit Matrix

`gemini` is intentionally absent from this table — it is a registered
worker (see "Headless workers" above) but has no assigned preference/backup
role yet. Do not add it to a row here without an explicit decision; this is
not an oversight to silently fix.

| Category | Preferred | Good backup | Notes |
|---|---|---|---|
| `feature` | `claude` | `cursor`, `codex` | Prefer Claude for phone/app_visible remote-control. Use Cursor when the operator explicitly wants the CLI-visible executor. Codex is a normal daemon coding worker via app-server + Codex Remote. Board management uses Fleet Manager API, not Codex chat. |
| `bug` | `claude` | `cursor`, `codex` | Same visibility rule as `feature`. |
| `research` | `claude` | `codex` | Prefer Claude for broad reasoning, synthesis, specs, and product thinking. |
| `review` | `claude` | `codex` | Prefer Claude for prose/design/product review; Codex for code-risk review in Manager chat. |
| `maintenance` | `claude` | `cursor`, `codex` | Prefer Claude for phone-visible repo edits; Codex for local coding daemon launches. |
| `decision` | `claude` | `codex`, `owner` | Claude drafts decisions; the operator confirms durable calls. |

## Project Overrides

Project overrides live in `Work/<project-id>/PROJECT.md`.

Use frontmatter like:

```yaml
assignment:
  default_assignee: claude
  reason: "Claude is actively working on this project right now."
```

The Manager should honor the override unless the user explicitly assigns a
different worker.

Known current override candidates:

- Billing Portal (`acme/billing-portal`) may be Claude-led while Claude is
  actively working on it.

## Agent-Specific Notes

### Claude

Claude has Discovery-style project access/workflow available to the user and can
work with project context as-is. Prefer Claude when:

- the project is already actively being handled by Claude;
- the task is product/spec/research/review/decision-heavy;
- continuity matters more than local terminal execution.

### Codex

Prefer Codex when:

- the task needs deterministic local filesystem edits;
- repo scanning, _registry updates, or generated workspace files are involved;
- local tests/commands should be run;
- the user asks this Manager chat to do the work immediately.

Codex is daemon-launchable through `codex app-server --stdio`. Operator
continuation uses Codex Remote (paired clients) and/or Fleet dashboard controls,
not a Claude-style remote-control URL.

### Cursor

Cursor is available through the Fleet `cursor-visible` daemon backend. Use it
when:

- the operator explicitly requests Cursor;
- the task is isolated UI/editor implementation work;
- CLI-visible inspection/resume is acceptable.

Cursor is not a phone-visible Claude remote-control session. Fleet stores the
Cursor chat id and dashboard resume command, but operator continuation happens
through `cursor-agent --resume <chat-id>` / Cursor, not through dashboard input
controls.

## Routing Notes

- Do not overfit early. It is acceptable to assign `unassigned`.
- Do not route `status: backlog` tasks into active worker pickup. Backlog is the
  owner-controlled pre-priority queue.
- For active `todo` pickup, choose by priority: `1` is highest, `5` is lowest,
  then natural ref order.
- Do not start or pick up work that violates `Fleet/LAUNCH_POLICY.md`.
- Record why a non-obvious assignee was chosen in the task's
  `assignment_reason`.
- If task category and project override conflict, project override wins.
- If the task asks the active Manager chat to implement immediately, assign
  `codex` and proceed unless the user says otherwise.
- Do not use `cursor` as a default assignee; keep it explicit opt-in unless a
  project override documents why Cursor should lead that project.
