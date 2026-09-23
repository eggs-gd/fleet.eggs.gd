# Plane N:N project mapping (design, not yet implemented)

Status: design note only. No code changed by this document.

## Problem

`_docs/PLANE_TASK_PROVIDER.md` documents the current, shipped behavior: one
Plane adapter instance, configured with exactly one Plane project UUID
(`taskProvider.project`) mapped to exactly one Core project
(`taskProvider.coreProject`) — for the *entire* `_backoffice/server` daemon.

That daemon is not scoped to one repository. It is Core itself: one process
serving every `Work/<project-id>/` workspace across every tracked
repository (`_registry/repositories.json` currently lists dozens, growing
toward ~50). Core is itself just one of those projects
(`core-eggs-gd` / `core.eggs.gd`) — it gets no special case.

Requirement: each local Core project should be able to live in its own
Plane project, under one Plane workspace, through one running daemon.

The current schema and code cannot express that:

- `providerconfig.PlaneSettings` has singular `ProjectID` and `CoreProject`
  fields (`_backoffice/server/internal/providerconfig/providerconfig.go`).
- `plane.Client` bakes `projectID` into the struct at construction and uses
  it in every request URL (`client.go:62,76,84` —
  `.../projects/{projectID}/...`).
- `plane.Provider` and `plane.ObservePoll`/`ObserveWebhook` read
  `p.settings.ProjectID` / `p.settings.CoreProject` directly wherever they
  need a project id (`provider.go:65,387,398`, `observe.go:97,151`) — there
  is exactly one of each per provider instance.

## Target config shape (v1: discovery on our side only, Plane linkage manual)

Discovery is one-directional. Two different things are getting discovered,
and only one of them is automatic:

- **Which Core projects exist** — already deterministic today
  (`_registry/repositories.json` via `make scan`,
  `Work/<project-id>/PROJECT.md` via `make workspaces`). Core's own tooling
  already knows this without asking Plane anything.
- **Which of those projects has a Plane project behind it, and which one**
  — never guessed. Alex is the only one who writes that value, because it
  requires a decision (does this project get a Plane mirror at all) that
  Core must not make on its own (Ground Rule 4: "Suggestions are not
  decisions").

So: the **key** (`coreProject`) can be added to `core.config.yaml`
automatically whenever Core discovers a new local project it doesn't
already have an entry for — written with an empty value. The **value**
(Plane project reference) is filled in by hand only for the subset that
should actually go to Plane; everything else just sits there with an empty
value and is treated as "not mirrored," same as if the key were absent.

```yaml
taskProvider:
  type: plane
  workspace: <slug>
  baseUrl: https://api.plane.so
  tokenEnv: PLANE_API_TOKEN
  projects:
    core-eggs-gd: <plane-project-ref>   # Alex filled this in
    career-wizard: <plane-project-ref>  # Alex filled this in
    some-new-repo-just-scanned-in: ""   # auto-added key, empty until Alex fills it
statusMap:
  backlog: Backlog
  todo: Todo
  # ... shared across all projects unless a project needs its own override
```

Sync step (where the auto-add lives): extend `make scan` /
`_backoffice/scripts/sniff_projects.py` (or `generate_workspaces.py` —
whichever already runs at the point a new project first gets a
`Work/<project-id>/`) to also ensure `core.config.yaml`'s
`taskProvider.projects` has a key for every known Core project. Rules:

- Additive only — add a missing key with an empty value; never touch a
  key that already has a non-empty value.
- Never remove a key even if the project later disappears from the
  registry (Ground Rule 5: don't delete historical `_registry` state
  automatically) — an orphaned key with an empty or stale value is
  harmless; a silently-dropped mapping to a real Plane project is not.
- This step needs to parse-and-rewrite `core.config.yaml` while leaving
  everything else in the file (`workspace`, `statusMap`, comments,
  formatting) untouched. `providerconfig.go` currently reads this file
  through a hand-rolled pseudo-frontmatter scalar parser
  (`mdfile.ParseFrontmatter`/`mdfile.Scalar`), not a real YAML library —
  fine for reading scalars, but a real read-modify-write-preserving-rest
  needs either a proper YAML library for this one write path, or a
  narrower text-level patch that only touches the `projects:` block.
  Worth deciding explicitly before writing this sync step, not
  discovering the gap while debugging a mangled config file.

Two things to pin down before implementing, both open questions rather than
decided here:

- **Map key = `coreProject`.** Must match the real Core project id exactly
  (the `Work/<project-id>/` folder name, e.g. `core-eggs-gd` — hyphens, not
  underscores) since that's what a task's `project` frontmatter field
  already carries. A key that doesn't match a real project id silently
  never matches any task.
- **Map value = Plane project reference — UUID or identifier?** The
  existing `plane.Client` builds every request URL from a raw project
  *UUID* baked in at construction (`client.go:62,76,84`,
  `.../projects/{projectID}/...`) — it does no identifier→UUID resolution
  today. So as written now, the value must be the real Plane project UUID,
  not the short human identifier (e.g. Plane's issue-prefix code) shown in
  the example above. If a human-friendly identifier is preferred over a
  raw UUID, that needs one extra resolution step (a cheap one-time
  workspace project list call, cached) before this ships — worth deciding
  explicitly rather than discovering it by a failed request later.

`statusMap` stays global for now (one workflow-state naming convention
across the workspace) unless a concrete project needs a different one —
add a per-entry override only when that actually happens, not
speculatively.

`repository` is dropped from this config entirely: Core already resolves
project → repository through the existing registry
(`Work/<project-id>/PROJECT.md`, `_registry/repositories.json`) —
duplicating that fact inside the Plane config would just be a second place
for it to go stale. Confirm this at implementation time rather than assume
it silently.

No backward-compat shim for the old singular `project`/`coreProject` keys:
the feature has not shipped to a real user yet (`core.config.yaml` in this
repo still has placeholder values), so there is nothing to migrate.

Explicitly out of scope for v1, revisit only if it becomes a real
bottleneck: discovering or auto-creating anything on the *Plane* side via
the Plane API (listing existing projects, creating a new one for a repo
that doesn't have one yet). Key discovery (our side) is in scope per
above; value discovery (Plane side) stays a manual decision indefinitely,
not just for v1.

## Code changes required

### `providerconfig.go`

- Replace `PlaneSettings.ProjectID` / `CoreProject` / `Repository` with
  `PlaneSettings.Projects map[string]string` — key is `coreProject`, value
  is the Plane project reference (see open question above on UUID vs
  identifier). A map structurally rules out the duplicate-`coreProject`
  problem a list would need to validate for.
- `Load` parses the `taskProvider.projects` mapping instead of three scalar
  keys.
- Validation: at least one entry; every key and value non-empty; `workspace`
  and `tokenEnv` stay required at the top level as today.

### `plane/client.go`

`projectID` can no longer be fixed at construction. Two ways to get there,
pick the simpler one once the routing shape below is settled:

1. Accept `projectID` as a parameter on each request-building call
   (`urlFor(projectID, suffix)` instead of `c.projectID`), and have callers
   pass the right id per task. One `Client` (one HTTP transport, one auth
   token) shared across all configured projects.
2. Construct one `Client` per configured project, all sharing the same
   underlying `http.Client`/auth. Simpler call sites, more objects.

Option 1 is closer to "one adapter, N projects" and avoids N redundant
`http.Client`s; prefer it unless it makes the diff much larger than option
2.

### `plane/provider.go` / `plane/flow.go`

- `Provider` currently assumes one `(ProjectID, CoreProject)` pair. It needs
  a lookup: given a task's `CoreProject` (already a real field on the Core
  domain model — tasks already carry `project: <project-id>` frontmatter
  per `_docs/DOMAIN_MODEL.md`), resolve the matching Plane project UUID
  before building the request.
- `Create` needs the caller (or the task itself) to say which
  `CoreProject` it belongs to *before* the Plane project id can be
  resolved — this already exists in the domain model, so no new field is
  needed on `Task`, just a resolution step inside the adapter.
- Reject (clear error, not silent fallback) a task whose `CoreProject` has
  no configured mapping — this must not silently write into whatever
  project happens to be first in the list.

### `plane/observe.go` (poll / webhook change detection)

- Today one `ProviderSync` polls one project on one interval. With N
  projects behind one adapter, change detection must cover all of them and
  fan into the same shared `TaskEvent` channel, tagging each event with the
  `CoreProject` it came from (already carried by `Source`/task fields —
  confirm `TaskEvent` construction sets it from the right project, not a
  single hardcoded one).
- Webhook payloads already carry Plane's own project id in the envelope;
  `ObserveWebhook` needs to map that back to the right `CoreProject` using
  the same lookup table as `Create`/`Get`.

### Rate limit at N-project scale

Plane's documented limit is 60 requests/minute **per API key** — shared
across every project polled through that key, not per project. The
existing 45s poll floor
(`_backoffice/server/internal/taskprovider/open/open.go:18`,
`planePollMinInterval`) was sized for one project. At ~50 projects, naive
"poll every project every 45s" blows the budget by roughly 50x.

This needs one of:

- Round-robin polling: one project's poll tick per interval, cycling
  through the list, instead of all projects every tick.
- Prefer webhooks as the primary change source once N is large, keeping
  polling only as a slower fallback/reconciliation pass.
- A shared rate limiter across all Plane HTTP calls (client-level), so the
  budget is enforced structurally instead of by picking an interval that
  happens to fit today's project count.

Whichever is picked, it belongs in `plane/observe.go` /
`taskprovider/open/open.go`, not duplicated per call site.

### Tests

- `providerconfig`: table-driven tests for the map schema — valid map,
  empty map, an entry with an empty key or empty value.
- `plane`: routing tests — `Create`/`Get`/`List`/`Update`/`AddComment` hit
  the correct project URL for a given `CoreProject`; unknown `CoreProject`
  returns a clear error, not a silent default.
- `plane/observe`: poll/webhook events carry the correct `CoreProject` when
  multiple projects are configured; rate-limit behavior under N projects
  (whichever strategy is chosen above) has a regression test.

### Docs

- Update `_docs/PLANE_TASK_PROVIDER.md`: replace the "one Core project ↔
  one Plane project" limitation with the map-based config, and update the
  verification checklist for a multi-project setup.
- This file (`PLANE_MULTI_PROJECT_MAPPING.md`) can be folded into
  `PLANE_TASK_PROVIDER.md` once implemented, or kept as the design record
  — decide at implementation time.

## Out of scope

- Changing the Core task domain model or status machine — `project`
  already exists as a task field; this is routing, not a schema change on
  the Core side.
- Mixing providers (Markdown for some projects, Plane for others) in one
  daemon run — still one `taskProvider.type` for the whole daemon.
- Per-project `statusMap` overrides — add only if a real project needs a
  different workflow-state naming than the shared default.

## Rollout suggestion

Validate the routing logic against 2-3 real projects (including
`core-eggs-gd` itself) before adding all ~50 — mainly to catch the
rate-limit question above with real traffic before it matters at scale.
