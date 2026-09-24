# Docs (App)

Architecture notes, decisions, and plans for the Core tool itself
(`server` + `view`). Operator-facing docs about
Inbox/Work/Fleet/Archive conventions live in the configured Data root's
`_docs/`, not in this repository.

Use this directory for explanations that are too detailed for the root
README but still important for future humans and agents.

## Engineering / Architecture

- `CORE_MANAGER_API.md` — Core Manager API for voice/text ingestion; replaces
  Codex-as-manager for non-coding capture. Workers stay execution-only.
- `MANAGER_MCP.md` — the Manager API mounted as MCP tools at `/mcp` on
  `core serve` itself (`manager_schema`/`manager_vocabulary`/
  `manager_command`) so an agent acting as Manager doesn't hand-parse task
  Markdown/registry files.
- `DOMAIN_MODEL.md` — canonical workspace/project/repository terms, task
  schema, and task state transitions.
- `TASK_FLOW_ARCHITECTURE.md` — TaskService / TaskProvider contracts for the
  task-flow refactor (`CORE-100`…`CORE-107`).
- `TASK_FLOW_CONTOURS.md` — independent task-flow contours; no single
  FsWalker→agent chain (`CORE-108`…`CORE-114`).
- `DAEMON_PLAN.md` — parked plan for the deterministic Core daemon/launcher.
- `AGENT_LAUNCHER.md` — Contour 2 launch/session runtime details, including
  per-provider (Claude/Codex/Cursor/Gemini) launch/resume notes.
- `AGENT_SESSION_REUSE.md` — the 1-1-1 session-reuse model and per-backend
  resume verification status.
- `AGENT_TOOL_EVIDENCE.md` — required MCP/tool evidence on sessions
  (CORE-120).
- `TECHNOLOGY.md` — runtime, backoffice, bootstrap script, and entrypoint
  technology decisions.
- `MCP_PROJECT_CAPABILITIES_PLAN.md` — draft plan for project-scoped MCP
  capabilities (global registry, per-project enable/disable); includes the
  Phase 0 infrastructure inventory findings reconciling the plan with the
  actual codebase.
- `PROJECT_TEMPLATES_PLAN.md` — draft plan for user-defined project
  templates (registry, Compare/Apply via the executor); includes Phase 0
  findings answering the plan's own pre-implementation questions.

## Findings / Incident Notes

- `CORE-140_TOOL_EVIDENCE_FINDINGS.md`
- `CORE-142_CLAUDE_REMOTE_CONTROL_FINDINGS.md`
- `LIVE_AGENT_SMOKE_RESULT.md`

## Plane Task Provider

- `PLANE_TASK_PROVIDER.md` — current, shipped one-project-per-daemon Plane
  adapter behavior.
- `PLANE_MULTI_PROJECT_MAPPING.md` — design note (not yet implemented) for
  N:N Plane-project ↔ Core-project mapping.
