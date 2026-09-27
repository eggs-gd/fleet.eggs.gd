# Fleet Go runtime

The Go side of Fleet: one binary, `fleet`, that serves the dashboard, the HTTP
and MCP APIs, and launches agents. Design and package map:
[`_docs/ARCHITECTURE.md`](../_docs/ARCHITECTURE.md).

## Running

Use the top-level Makefile:

```bash
make serve            # build, then run bin/fleet serve (plans launches only)
make serve LIVE=1     # also start agents for ready tasks
```

`serve` starts in dry-run mode unless you pass `--live` or turn live mode on in
Settings. It:

- serves the dashboard built into the binary (`make build` embeds it;
  `--backoffice-dir` serves a directory instead, for dashboard development);
- exposes `/api/health` (`degraded` lists failing parts, empty when healthy);
- exposes the board state at `/api/state`;
- edits and creates tasks through `PATCH` and `POST /api/tasks`;
- exposes the Manager API under `/api/manager/*` and the Manager MCP tools at
  `/mcp` (see [`_docs/MANAGER.md`](../_docs/MANAGER.md));
- runs the task flows: change detection, execution, finalization.

Other commands: `fleet scan`, `fleet rebuild-index`, `fleet version`.

## Layout

- `cmd/fleet` — CLI entry point.
- `internal/…` — packages, listed in the code map in `_docs/ARCHITECTURE.md`.
- `lib/chain` — the chain-of-responsibility framework the flows use.
- `lib/logger` — named-service logger.

## Rules

- Source Markdown and JSON files are canonical. Generated JSON and index files
  are projections.
- The runtime does not guess a missing project, repository, assignee or launch
  setting.
- Runtime logs go through `lib/logger`, not ad hoc `log.Printf`.
- Agent launching follows `Fleet/LAUNCH_POLICY.md`: `todo` and `needs_rework`
  are pickup statuses, and `backlog` is the hold area.
- Fleet does not commit, push or open a pull request on its own.

## Task edit API

`PATCH /api/tasks` takes a JSON body with:

- `path` — an opaque task locator (a Markdown path, or a `plane://…` locator
  when Plane is active);
- `status`, `assignee`, `body` — optional;
- `comment`, `comment_author` — an optional review comment.

The write goes through `TaskService`, and the in-memory projection behind
`/api/state` refreshes after it succeeds.

## Task create API

`POST /api/tasks` takes a JSON body with:

- `title` (required) and `request` (required description);
- `project` (required, a project id under `Work/`);
- `repository` (optional relative path);
- `status` (default `backlog`), `type` (default `feature`), `assignee`
  (default `unassigned`), `priority` (default `5`), `assignment_reason`
  (all optional).

The endpoint allocates the next `<TAG>-<number>` ref for the project from
`_registry/counters.json` and writes through `TaskService` (a Markdown file or
a Plane work item, depending on config). It does not launch agents or create
commits.

## Task provider

Markdown task cards in `Work/*/tasks/` are the default. To use Plane instead,
add `core.config.yaml` to the data root and export the token variable, see
[`_docs/PLANE_TASK_PROVIDER.md`](../_docs/PLANE_TASK_PROVIDER.md). The
dashboard, the Manager, the launcher and the finalizer all talk to
`TaskService`, not to a provider.
