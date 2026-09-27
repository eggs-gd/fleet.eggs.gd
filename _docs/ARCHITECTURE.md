# Architecture

Fleet is one Go process (`fleet serve`) that owns a data root (Inbox, Work,
Fleet, Archive), a local HTTP API, the embedded dashboard, an MCP endpoint for
the Manager, and the launcher that hands ready tasks to agents.

```text
dashboard (Svelte) ──HTTP──┐
Manager session ────MCP────┤
                           ▼
                      TaskService ──► TaskProvider (Markdown | Plane)
                           ▲
     execution flow ───────┤◄── task events (file watcher | Plane polling)
     finalizer flow ───────┘
```

Domain terms, the task schema and the status machine are in
[DOMAIN_MODEL](DOMAIN_MODEL.md).

## Task writes

Every task write goes through `taskflow.TaskService`: the dashboard, the
Manager tools, launch claims and the finalizer. Nothing else touches Markdown
or Plane. `TaskService` serializes writes in-process and delegates storage to
the active `TaskProvider`.

Markdown provider mutations lock the task path, reread the file, check the
expected version and the allowed transition, then write through a temporary
file and an atomic rename. Direct edits of task files by a person or an
external agent stay allowed. The file watcher reloads them into the in-memory
projection.

Providers: `internal/taskprovider/markdown` (default) and
`internal/taskprovider/plane` (optional, see [PLANE_TASK_PROVIDER](PLANE_TASK_PROVIDER.md)).
Exactly one provider is active. Workspace cards and the project registry stay
file-based under either.

## Flows

Task management, change detection, execution and finalization are separate
flows. The last three are built on `server/lib/chain` and share typed
channels, not one conveyor.

| Flow | Input | Output |
|---|---|---|
| Task management (`internal/manager`) | a request from the Manager or the dashboard | a `TaskService` command and a confirmation; it never launches, waits or finalizes |
| Change detection | Markdown files (`FsWalker` → `Projector` → `CoreFinalizer`), or Plane polling (`ProviderSync`) | `TaskEvent` |
| Execution | `TaskEvent` | launch plan, or a runtime session and an agent process |
| Finalization | `ExecutionResult` from a finished session | `TaskService.ReportExecution` |

Execution loads the current task through `TaskService` for each event. It
never receives a file path or a Plane payload. Finalization is the only place
that turns a worker outcome into a status change.

`/api/state` reads the shared in-memory projection kept by the runtime. The
dashboard polls it. It does not subscribe to Go channels.

## Launch

Launch gates run in order on each eligible task, and the result is stored in
`Task.LaunchEvaluation` (runtime state, never written into the card):

- `status:pickup` — `todo` or `needs_rework`;
- `assignee` — set;
- `depends_on` — every prerequisite is `done`;
- `fleet_profile` — `Fleet/<agent>.md` exists;
- `repository` — exactly one repository;
- `command` — the adapter produced a launch command;
- `agent_live_ready`, `agent_visibility`, `agent_executable` — the adapter is
  verified, visible to the operator, and its binary was found.

A task that fails a gate stays in its pickup status with the failed gates
recorded. Only a real launch failure moves it to `blocked`. Details, prompts,
the outcome protocol and per-agent notes: [AGENT_LAUNCHER](AGENT_LAUNCHER.md).

## Launch mode

Fleet starts in dry-run: it plans launches and logs them but starts no
process. `--live` (or the saved launch mode in Settings) turns launching on.
Precedence: command-line flag, then the saved mode in `core.local.yaml`, then
dry-run.

## Storage

| What | Where |
|---|---|
| Tasks, project cards, Inbox items, decisions | the data root (`Work/`, `Inbox/`, …) |
| Project registry, ref counters | `_registry/` in the data root |
| Runtime sessions, session logs, audit events, Manager relay log | `~/.fleet/runtime.db` (SQLite, mode 0600) |
| Local launch token | `~/.fleet/launch-token` |
| Saved data-root choice | `~/.fleet/app.json` |
| Per-install settings (agents, launch mode, scan roots) | `core.local.yaml` in the data root |
| Task provider selection | `core.config.yaml` in the data root |

Runtime state lives outside the data root and outside any git repository. A
database from an older layout is imported from its `_registry` files on first
open.

## Failure model

Fleet fails fast where a broken setup makes the system meaningless, and
reports where it can keep running.

- **Startup preflight** checks the data root, permissions, the dashboard and
  the provisioned files, and refuses to start with one aggregated message.
- **Runtime health** (`internal/health`) collects problems found while
  running. `/api/health` returns them in `degraded`, and the dashboard shows
  them. A failed pass is retried.
- **Scanner** (`Scanner.Watch`, every 2 s) repairs data drift: it syncs
  `Work/<id>/PROJECT.md` with the registry, gives a project without a tag one,
  and every 30 ticks checks `counters.json`. It does not rewrite a duplicate
  or invalid tag written by a person; it reports it.
- **Audit**: every `task.*` bus event is written to `events` in `runtime.db`
  before it is published.

## Task refs

A ref is `<TAG>-<number>`. The tag is `tag:` in the project's `PROJECT.md`;
the scanner generates one (four letters from the project id, deterministic
fallbacks on a clash) and a person may change it (2–6 uppercase letters or
digits, unique, not `INBOX`). Counters are per tag in
`_registry/counters.json`. Existing refs stay valid after a tag change.

## Event bus

`internal/eventbus` is an in-process bus with named channels (`task`,
`release`, `worker`, `practice`, `configuration`). Only `task` has a
publisher: `task.needs_attention` and `task.needs_review`. The bus has no disk
queue and no retry. The Manager reads the audit rows through `manager_events`;
nothing is pushed into a Manager session.

## Code map

| Package | Owns |
|---|---|
| `internal/server` | HTTP routes, composition, startup preflight, dashboard and Manager APIs, MCP mount |
| `internal/taskflow` | `TaskService`, `TaskProvider`, status table, execution outcome mapping |
| `internal/taskprovider/{markdown,plane,open}` | storage adapters and change sources |
| `internal/execution`, `internal/execution/providers` | launch gates, prompts, sessions, per-agent adapters and runners |
| `internal/executionapi` | provider-neutral session and capability types, tool-evidence rules |
| `internal/executionfinalizer` | outcome → `ReportExecution` |
| `internal/manager`, `internal/managerskills` | Manager service, intents, skills |
| `internal/projectscan`, `internal/projecttag` | repository discovery, project cards, tags |
| `internal/runtimedb`, `internal/audit`, `internal/health` | runtime storage, audit log, degraded reporting |
| `internal/settings`, `internal/appconfig`, `internal/providerconfig` | local settings, data-root bootstrap and template, provider config |
| `internal/webui` | embedded dashboard |
| `view/` | Svelte dashboard |
