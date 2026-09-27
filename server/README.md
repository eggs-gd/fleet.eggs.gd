# Fleet Go Runtime

This directory contains the Go runtime for the Fleet dashboard and local
automation.

## Current Entry Point

Run Fleet through the top-level Makefile:

```bash
make serve
```

The `serve` command is the single runtime entry point. It starts in dry-run mode
(it plans launches and starts no agents) unless you pass `--live` or set live mode
in Settings. It:

- serves the dashboard built into the binary (`make build` embeds it; `--backoffice-dir`
  serves a directory instead, for dashboard development);
- exposes `/api/health` (`degraded` lists failing parts, empty when healthy);
- exposes runtime Core state at `/api/state`;
- persists task edits through `PATCH /api/tasks`;
- creates new Work tasks through `POST /api/tasks`;
- exposes the Core Manager API under `/api/manager/*` (text, audio, command,
  vocabulary, schema) — see `_docs/CORE_MANAGER_API.md`;
- starts the independent Core contours (Markdown/Plane change detection,
  execution from TaskEvent, finalization from ExecutionResult); see
  `_docs/TASK_FLOW_CONTOURS.md`.

There is intentionally no separate `daemon` command right now.

## Current Packages

- `cmd/fleet`: CLI entry point.
- `internal/server`: HTTP server, static backoffice host, Manager API routes,
  composition root (`Compose`), and dashboard REST (`DashboardSurface`:
  TaskService writes + board/`execution.Status` poll; CORE-114).
- `internal/manager`: Contour 1 (task management) — Human → Manager →
  `taskflow.TaskService` only. Holds `TaskManagement` (Create/Get/List/Patch);
  no Claim, ReportExecution, FsWalker, providers, or agent launch
  (`_docs/TASK_FLOW_CONTOURS.md` §3, CORE-110).
- `internal/taskflow`: application contracts (`TaskService`,
  `TaskEvent{Before,After,Source}`, `ExecutionResult`) and the serialized
  write queue.
- `internal/taskprovider`: board projection, provider factory (`open`),
  migration-era `Provider`/`ChangeSource`; adapters in `markdown/` and `plane/`.
- `internal/execution`: Contour 2 — launches, sessions, orphans, Status/Control
  (black-box API for dashboard/composition).
- `internal/executionfinalizer`: Contour 3 — `ExecutionResult` → task updates.
- `internal/tasklifecycle`: domain status rules, comments, worker-result
  helpers (no storage I/O).
- `lib/chain`: local copy of Perceptrail `perceplib/chain`.
- `lib/logger`: local copy of Perceptrail `perceplib/logger` with the
  Gontroller decorator import path adjusted for the Core module.

## Planned Runtime Areas

These are planned responsibilities, not packages yet. Create directories only
when code exists.

- Backoffice integration: API endpoints and static asset serving.
- Work: load, validate, and update `Work/<project-id>/PROJECT.md` and task
  cards.
- Registry: scan repositories, detect workspace groups, and maintain
  `_registry` state.
- Fleet: load worker profiles, apply routing policy, and build per-agent
  queues.
- Launcher next steps: apply locks, start configured worker commands, and
  record launch results.
- Validation and indexing: validate Core files and rebuild generated indexes.

## Runtime Rules

- Source markdown/json files are canonical.
- Generated JSON and index files are projections.
- The runtime must not guess missing project, repository, assignee, or launch
  metadata.
- Daemon/runtime logs should use `lib/logger` named services instead of ad hoc
  `log.Printf` calls.
- Agent launching follows `Fleet/LAUNCH_POLICY.md`: `todo` and `needs_rework`
  are pickup statuses; `backlog` is the deterministic hold area.
- No automatic commit, push, or pull request creation in MVP.

## Task Edit API

`PATCH /api/tasks` accepts a JSON body with:

- `path`: opaque task locator (Markdown relative/absolute path, or Plane
  `plane://…` locator when that provider is active);
- `status`: optional task status;
- `assignee`: optional assignee;
- `body`: optional body text;
- `comment` / `comment_author`: optional review comment.

Writes go through `TaskService` (`Runtime.PatchTask` /
`DashboardSurface.PatchTask`). The in-memory projection used by
`/api/state` refreshes after a successful mutation. Dashboard handlers
never receive `TaskEvent` / `ExecutionResult` channels.

## Task Create API

`POST /api/tasks` accepts a JSON body with:

- `title` (required);
- `request` (required description);
- `project` (required project id under `Work/<project>/`);
- `repository` (optional relative repository path);
- `status` (optional; defaults to `backlog`);
- `type` (optional; defaults to `feature`; legacy `todo` normalizes to `feature`);
- `assignee` (optional; defaults to `unassigned`);
- `priority` (optional; defaults to `5`);
- `assignment_reason` (optional).

The endpoint allocates the next monotonic `CORE-*` ref from
`_registry/counters.json`, persists through `TaskService` (Markdown file or
Plane work item depending on config), refreshes derived views when the
Markdown provider is active, and does not launch agents or create commits/PRs.

## Core Manager API

See `_docs/CORE_MANAGER_API.md` for the full design. HTTP surface:

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/manager/text` | Classify/act on plain text |
| `POST` | `/api/manager/audio` | STT then classify/act (`multipart` field `audio`) |
| `POST` | `/api/manager/command` | Execute a structured intent |
| `GET` | `/api/manager/vocabulary` | Dynamic STT vocabulary (projects+summaries, technologies) |
| `GET` | `/api/manager/schema` | Manager LLM JSON Schema |

Deterministic board commands (show/move/comment/assign/priority/cancel) run
without an LLM. Free-form task creation falls through to a structured-output
classifier once configured. Contour 1 ends after `TaskService` accepts the
command and returns an operator confirmation — it does not schedule or launch
agents. Writes and queries go through `TaskManagement` / `TaskService`.
Dashboard `PATCH|POST /api/tasks` uses the same `TaskService` path
(`Runtime.CreateTask` / `Runtime.PatchTask`). Launcher claims and execution
finalization also go through `TaskService` (`Claim` / `ReportExecution`) on
separate contours so agents never mutate task status directly.

## Task provider selection

Markdown task cards under `Work/*/tasks/*.md` are the default. To use Plane
instead, add root `core.config.yaml` and set the API token env var — see
`_docs/PLANE_TASK_PROVIDER.md`. Manager, dashboard, launcher, and finalizer
always talk to `TaskService`; they do not know which provider is active.
