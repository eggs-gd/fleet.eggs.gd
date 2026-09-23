# Plane Task Provider

Plane is an optional second `TaskProvider` behind the same application
contract as Markdown (`taskflow.TaskProvider` / `TaskService`). Markdown
remains the default until Plane is explicitly configured and verified.

Architectural boundaries: `_docs/TASK_FLOW_ARCHITECTURE.md`.

## When to use it

Use Plane when Alex wants board state in
[app.plane.so/eggs_gd](https://app.plane.so/eggs_gd/) instead of (or as a
trial alongside migrating away from) `Work/*/tasks/*.md` files.

Do **not** treat Plane as a second parallel board while Markdown is still
the active provider — Core runs exactly one task provider at a time.

## Config

Optional root-level `core.config.yaml`. Absent file = Markdown (unchanged).

```yaml
taskProvider:
  type: markdown
```

or:

```yaml
taskProvider:
  type: plane
  workspace: eggs_gd
  baseUrl: https://api.plane.so
  tokenEnv: PLANE_API_TOKEN
  project: <plane-project-uuid>
  coreProject: core-eggs-gd
  repository: core.eggs.gd
statusMap:
  backlog: Backlog
  todo: Todo
  doing: In Progress
  blocked: Blocked
  needs_review: In Review
  needs_rework: Todo
  done: Done
  archived: Cancelled
```

| Field | Meaning |
|---|---|
| `type` | `markdown` (default) or `plane` |
| `workspace` | Plane workspace slug |
| `baseUrl` | API base; defaults to `https://api.plane.so` |
| `tokenEnv` | **Name** of the env var holding the API token (never the token itself) |
| `project` | Plane project UUID for Core-managed work items |
| `coreProject` | Core project id (`Work/<id>`) mapped to that Plane project |
| `repository` | Optional single launch repository for Plane-backed tasks |
| `statusMap` | Optional Core status → Plane workflow state **name** overrides |

Secrets stay local (Ground Rule 7): export the token, do not commit it.

```bash
export PLANE_API_TOKEN=plane_api_...
make serve
```

Loader: `_backoffice/server/internal/providerconfig`.

## Metadata placement

Deliberate mix — Plane owns durable board fields; Core owns execution
runtime:

| Core concept | Plane / local home |
|---|---|
| `CORE-N` ref | Plane `external_id` + `external_source=core` |
| Title / body | Work item name / description |
| Status lifecycle | Plane workflow state (via `statusMap` or identity names) |
| Priority 1–5 | Plane `urgent/high/medium/low/none` |
| Assignee (`claude`/`codex`/…) | Label `core:assignee:<name>` (not Plane member UUIDs) |
| Task type | Label `core:type:<type>` |
| Comments / review notes | Plane work-item comments (`author: text` in HTML) |
| Project / workspace / repository | Configured `coreProject` + optional `repository` on the provider |
| Sessions, locks, launch evaluation, logs | Local `_registry/` only — never mirrored into Plane |

Rationale: Plane assignees are real workspace members; Core workers are a
closed vocabulary without requiring Plane user accounts per agent. Labels
are visible in the Plane UI and round-trip on the work-item payload.
Execution/runtime state is Core-daemon-local and must not depend on Plane
availability.

## Change observation

```text
polling (default)  → plane.ObserveChanges (ChangeSource) → TaskEvent → execution chain
webhook (optional) → plane.ObserveWebhook → TaskEvent → same channel
```

- **Polling:** Runtime attaches `ProviderSync` when the provider implements
  `taskprovider.ChangeSource` (Plane does; Markdown does not). Poll floor is
  45s for Plane's 60 req/min budget. Fingerprinting and `TaskEvent`
  construction live in `plane.ObservePoll` / `ObserveChanges` — corechain
  does not import Plane types or branch on `Type()=="plane"`.
- **Webhook:** `ObserveWebhook` normalizes common Plane JSON envelopes and
  reloads via `Load`. Core does not host an HTTP webhook route in this
  slice; the adapter is ready for a future route without launcher changes.

## Package layout

`_backoffice/server/internal/taskprovider/plane`:

| File | Owns |
|---|---|
| `client.go` / `entities.go` | REST, auth, pagination, API errors |
| `mapping.go` / `provider.go` | Core Task ⇄ Plane work item; legacy `taskprovider.Provider` |
| `flow.go` / `convert.go` | `Flow()` → `taskflow.TaskProvider` for `TaskService` |
| `observe.go` | Poll / webhook → `TaskEvent` |

## Limitations

- One Core project ↔ one Plane project per daemon config (no N:N yet).
- `List()` fetches comments only for `blocked` tasks (rate-limit tradeoff);
  `Load()` always fetches full comments.
- States/labels cached ~60s per provider instance.
- Workflow states must exist for every Core status Core writes (or be
  covered by `statusMap`); missing target state is a hard error.
- Workspaces (`Work/*/PROJECT.md`) and `_registry` stay file-based under
  both providers; only **tasks** move to Plane.
- `Work/INDEX.md` is not rebuilt under Plane (Markdown-only convenience).
- Live smoke against Alex's Plane workspace still requires a real token and
  matching workflow states — unit tests use a mocked HTTP Plane API.

## Verification checklist

1. Create `core.config.yaml` with `type: plane` and required fields.
2. Ensure Plane project states match Core statuses or set `statusMap`.
3. `export PLANE_API_TOKEN=…` (or configured `tokenEnv`).
4. `make serve` and confirm `/api/state` lists Plane work items.
5. Create/transition/comment via Manager or dashboard; confirm Plane UI.
6. Switch back to Markdown by removing the file or setting `type: markdown`.
