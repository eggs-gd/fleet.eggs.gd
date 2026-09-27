# Plane task provider

Plane is an optional `TaskProvider` behind the same contract as Markdown
(`taskflow.TaskProvider`, see [ARCHITECTURE](ARCHITECTURE.md)). Markdown is the
default. Exactly one provider is active, so Plane is not a second board next to
Markdown.

## Config

An optional `core.config.yaml` in the data root. No file means Markdown.

```yaml
taskProvider:
  type: plane
  workspace: my-workspace
  baseUrl: https://api.plane.so
  tokenEnv: PLANE_API_TOKEN
  coreProject: my-project
  repository: my-repo
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
| `baseUrl` | API base, default `https://api.plane.so` |
| `tokenEnv` | **Name** of the environment variable holding the API token, never the token |
| `coreProject` | The Fleet project (`Work/<id>`) whose tasks live in this Plane project |
| `project` | Optional Plane project UUID. When absent, Fleet finds the Plane project whose identifier or name equals `coreProject` (case-insensitive) |
| `repository` | Optional single launch repository for Plane-backed tasks |
| `statusMap` | Optional Fleet status → Plane workflow state **name**; a status not listed maps to a state with the same name |

Keep the token out of files:

```bash
export PLANE_API_TOKEN=plane_api_...
make serve
```

The loader is `server/internal/providerconfig`.

## Refs

A new Plane-backed task gets a ref from the tag of its `coreProject`
(`<TAG>-<number>`, counters in `_registry/counters.json`). The ref is stored as
the work item's `external_id` with `external_source=core`. If the project has
no tag, task creation fails with an explanation rather than creating a task
without a ref.

## What lives where

Plane owns the durable board fields. Fleet keeps execution state local.

| Fleet concept | Plane or local home |
|---|---|
| Ref | `external_id` + `external_source=core` |
| Title, body | work item name and description |
| Status | workflow state (`statusMap` or same-named states) |
| Priority 1–5 | `urgent`, `high`, `medium`, `low`, `none` |
| Assignee | label `core:assignee:<name>` (Plane members are real accounts; Fleet workers are not) |
| Type | label `core:type:<type>` |
| Comments and review notes | work item comments (`author: text`) |
| Project, workspace, repository | configured `coreProject` and `repository` |
| Sessions, launch evaluation, logs | local `runtime.db` only, never mirrored |

Workspace cards and `_registry` stay file-based. `Work/INDEX.md` is not rebuilt
under Plane.

## Change detection

Polling (default): the runtime attaches a `ProviderSync` when the provider is a
`taskprovider.ChangeSource`, which Plane is. The poll interval has a 45 s floor
for Plane's 60 requests/minute limit. Fingerprinting and `TaskEvent` creation
are in `plane.ObservePoll` and `ObserveChanges`. Runtime code does not branch on
the provider type.

Webhook: `ObserveWebhook` normalizes Plane's JSON envelopes and reloads through
`Load`. Fleet has no HTTP route for it yet.

## Code

`server/internal/taskprovider/plane`:

| File | Owns |
|---|---|
| `client.go`, `entities.go` | REST, auth, pagination, API errors |
| `mapping.go`, `provider.go` | Fleet task ⇄ Plane work item |
| `flow.go`, `convert.go` | `Flow()` → `taskflow.TaskProvider` for `TaskService` |
| `observe.go` | polling and webhook → `TaskEvent` |

## Limits

- One Fleet project maps to one Plane project for the whole instance. Mapping
  several Plane projects to several Fleet projects is not supported. The client
  builds every URL from one project id, and the poll budget was sized for one
  project (the limit is 60 requests/minute per key across all projects).
- `List()` fetches comments only for `blocked` tasks; `Load()` fetches them all.
- States and labels are cached for about 60 s per provider instance.
- Every Fleet status Fleet writes needs a Plane workflow state (or a
  `statusMap` entry). A missing target state is a hard error.
- Tests use a mocked Plane API. A live run needs a real token and matching
  workflow states.

## Trying it

1. Create `core.config.yaml` with `type: plane` and the fields above.
2. Make the Plane project's states match the Fleet statuses, or set `statusMap`.
3. Export the token variable.
4. Run `make serve` and check that `/api/state` lists the Plane work items.
5. Create, transition and comment through the dashboard or the Manager, and
   check the Plane UI.
6. To go back, remove the file or set `type: markdown`.
