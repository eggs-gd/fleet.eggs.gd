# Task Flow Architecture

Canonical application contracts for Core task orchestration.

This document is the architectural source of truth for the task-flow refactor
series (`CORE-100` … `CORE-107`): `TaskService`, providers, commands vs
events, and outcome mapping. Later agents must follow these boundaries
instead of inventing a parallel design from scattered task cards or from the
half-abstraction that grew around `CORE-94` / `CORE-95` / `CORE-96`.

**Contours supersede the single vertical pipeline.** Independent entry
points (Manager, `TaskEvent` execution, `ExecutionResult` finalization,
Dashboard REST) are defined in `_docs/TASK_FLOW_CONTOURS.md` (`CORE-108` …
`CORE-114`). Do not implement or preserve one renamed end-to-end conveyor
from this document’s §2 diagram alone.

Related durable contracts:

- `_docs/TASK_FLOW_CONTOURS.md` — independent contours; no single
  FsWalker→agent chain (`CORE-108`…).
- `_docs/DOMAIN_MODEL.md` — task schema and status state machine.
- `../../Data/_docs/OPERATING_MODEL.md` — Manager / worker / Alex roles.
- `_docs/CORE_MANAGER_API.md` — Manager capture surface.
- `_docs/AGENT_LAUNCHER.md` — launch eligibility and worker prompt shape.
- Go package `internal/taskflow` — compile-checked target contract types.

Do not migrate providers, implement Plane, or build an event bus in the
architecture pass. Implementation slices are `CORE-101` through `CORE-107`.

---

## 1. Problem statement

The current abstraction is wrong.

Core originally had a filesystem-first pipeline:

```text
fswalker
→ markdown parser
→ typed Task
→ validation
→ launcher
→ agent
```

That worked because Markdown was not a provider behind an interface. It was
deeply integrated into the first steps of the chain:

- filesystem changes were the trigger;
- Markdown parsing created the typed task;
- the same chain then continued into execution.

A provider package was added for Plane, but Markdown was not actually moved
behind the same application boundary. In practice:

```text
providers/
├── plane
└── no real markdown provider
```

because Markdown still lives inside `fswalker + parser + early chain steps`
(`corechain.FsWalker`, `corechain.Projector`, `tasklifecycle.LoadTaskFile`,
and create/mutate paths that still assume files).

This produced a fake abstraction:

- Plane is treated as a provider;
- Markdown remains part of the execution pipeline;
- provider-specific `if` logic appears inside the chain
  (`newTaskProvider` switches, `ProviderSync` vs `FsWalker`, index rebuild
  branches);
- code volume increased instead of decreasing;
- command handling, persistence, and execution orchestration remain mixed;
- the manager has its own `manager.TaskProvider` that is not the same
  contract as `taskprovider.Provider`.

Useful pieces from `CORE-94`–`CORE-97` remain (lifecycle package, worker
result protocol, Plane adapter sketch). The next pass must refactor the
boundary instead of adding more provider branches.

**What must be replaced or moved:**

| Current piece | Problem | Target home |
|---|---|---|
| `FsWalker` → `Projector` → task parse in the generic chain | Filesystem is the generic execution input | Markdown change adapter only; emits `TaskEvent` |
| `tasklifecycle.LoadTaskFile` / frontmatter I/O used as shared core | Markdown leaks into “generic” code | `MarkdownProvider` |
| `taskprovider.Provider` used directly by runtime/manager/finalizer | No application service; no write serialization | Callers use `TaskService` only |
| `manager.TaskProvider` parallel interface | Second fake boundary | Delete once manager talks to `TaskService` |
| Direct `Mutate` / file writes from launcher / agents | Lifecycle and storage mixed | Finalizer + `TaskService` |
| Provider `Type()` branches in orchestration | Storage mechanics leak upward | Delete from generic chains |

---

## 2. Target model

> **Superseded control-flow shape.** The diagrams below show shared
> `TaskService` / provider / event *contracts*. They must **not** be read as
> permission to keep one continuous Manager→Service→Event→Launch→Agent→
> Finalizer conveyor. Independent contours are mandatory — see
> `_docs/TASK_FLOW_CONTOURS.md`.

Separate the system into clear responsibilities (shared services, not one
chain):

```text
Manager / Commands          → TaskService → TaskProvider
Provider change adapters    → TaskService reconcile → TaskEvent
Execution (from TaskEvent)  → Agent → ExecutionResult
Finalizer (from result)     → TaskService → TaskProvider
Dashboard REST              → TaskService / RuntimeService
```

After the contours refactor (authoritative picture in
`_docs/TASK_FLOW_CONTOURS.md`):

```text
Contour 1: Human → Manager → TaskService (stops here)

Storage/detect: Markdown/Plane ↔ TaskService → TaskEvent
                      ↑
                 FsWalker (Markdown only) / Plane webhook|poll

Contour 2: TaskEvent → Execution Chain → Agent → ExecutionResult

Contour 3: ExecutionResult → Finalizer → TaskService

Dashboard: REST → TaskService / RuntimeService (no Go channels)
```

Core orchestrates tasks. Providers store and observe them. Providers must not
leak into manager or launcher code.

---

## 3. Responsibility boundaries

### 3.1 Manager / commands

The manager handles human/operator intent.

Sources may include:

- Core Manager API / voice input;
- Codex manager chat (override only);
- dashboard actions;
- API calls;
- deterministic board commands.

**May:**

- parse intent;
- resolve project and repository;
- validate task input against domain rules;
- create or update tasks through `TaskService`;
- return confirmation to the operator.

**Must not:**

- launch agents;
- watch files;
- know Plane endpoints;
- know Markdown paths or frontmatter layout;
- mutate provider storage directly;
- own a second task-provider interface.

Example:

```text
Voice / Codex
→ parse request
→ resolve project
→ TaskService.Create(...)
→ "Created CORE-81"
```

### 3.2 TaskService

`TaskService` is the single application-level API for all task mutations.

All callers use it:

- manager;
- dashboard;
- launcher (claims / reads needed for launch);
- finalizer.

No caller talks directly to Markdown or Plane.

Suggested public contract (canonical Go shapes live in `internal/taskflow`):

```go
type TaskService interface {
    Create(ctx context.Context, input CreateTask) (Task, error)
    Get(ctx context.Context, id string) (Task, error)
    List(ctx context.Context, filter TaskFilter) ([]Task, error)
    Claim(ctx context.Context, id string) (Task, error)
    Transition(ctx context.Context, id string, to Status, meta TransitionMeta) error
    AddComment(ctx context.Context, id string, text string) error
    ReportExecution(ctx context.Context, result ExecutionResult) error
}
```

Notes:

- `id` is the opaque task identity used by the active provider (today often
  the locator / relative path / `plane://…` string, or a Core `ref` once the
  service resolves it). Callers treat it as opaque.
- `Get` / `List` are reads: they may hit an in-memory snapshot or the
  provider directly and **do not** pass through the write queue.
- `Transition` enforces `_docs/DOMAIN_MODEL.md` status rules in one place.
- `ReportExecution` is the preferred post-execution entry: the Finalizer
  calls it with a typed `ExecutionResult`; the service applies the
  outcome→status table below (comment + transition). Equivalently, a thin
  Finalizer may call `AddComment` + `Transition` itself using the same
  table — but never the provider.

### 3.3 Serialized mutation queue

All task **writes** are serialized inside `TaskService`.

```text
Manager ───────┐
Dashboard ─────┤
Launcher ──────┼→ TaskCommand channel → TaskService loop → TaskProvider
Finalizer ─────┘
```

Requirements:

- one goroutine processes mutations in order;
- callers do not receive direct access to the channel;
- public methods enqueue commands and await a result;
- validation and transition rules live in one place;
- reads do not need to pass through the write queue.

Internal shape (private to the service package):

```go
type taskCommand struct {
    run    func(context.Context, TaskProvider) (any, error)
    result chan taskCommandResult
}
```

This protects the single-process Core runtime from concurrent writes. It does
**not** replace provider-side optimistic concurrency or reconciliation for
external systems (Plane etag/version, Markdown expected-updated-at, etc.).

Do not build a general event-bus framework. Use a private Go channel inside
the process.

### 3.4 TaskProvider

Both Markdown and Plane implement the same persistence contract.

```go
type TaskProvider interface {
    Create(ctx context.Context, input CreateTask) (Task, error)
    Get(ctx context.Context, id string) (Task, error)
    List(ctx context.Context, filter TaskFilter) ([]Task, error)
    Update(ctx context.Context, task Task) error
    AddComment(ctx context.Context, id string, text string) error
}
```

Exact method names may evolve slightly during `CORE-101`/`CORE-102` as long
as:

- manager / launcher / finalizer never see provider-specific types;
- claim/launch remains expressible via service-level `Claim` (serialized
  pickup check + `doing` write) or `Transition` to `doing`; both still only
  talk to `TaskProvider.Update`;
- provider-specific details stay inside the adapter package.

`taskprovider.Provider` (`Type` / `List` / `Load` / `CreateFromRequest` /
`Mutate` / `Claim`) is the **current** half-abstraction. It is replaced by
the `TaskProvider` + `TaskService` split above; do not keep growing
`Type()`-based orchestration branches.

#### MarkdownProvider

Owns:

- path resolution;
- file reading/writing;
- frontmatter parsing;
- Markdown serialization;
- task ID / ref generation hooks still required locally;
- index rebuild hooks if still required;
- file-specific validation.

`fswalker` must **not** parse tasks or own task creation logic. It only
detects filesystem changes and asks `MarkdownProvider` to reload the
affected task.

#### PlaneProvider

Owns:

- REST API calls;
- Plane IDs and Core ref mapping;
- state/status mappings;
- comments;
- pagination;
- API errors;
- provider-specific metadata;
- webhook payload normalization or polling.

No Plane-specific `if` statements in generic manager, dashboard, launcher,
or finalizer code.

### 3.5 Provider change sources and events

Provider change detection is separate from task persistence.

Normalize all provider-specific notifications into one internal event:

```go
type TaskEvent struct {
    TaskID string
    Before *Task // nil when the task did not previously exist
    After  Task
    Source TaskEventSource
}
```

`TaskEventSource` values: `markdown_fs`, `plane_poll`, `plane_webhook`,
`task_service`. Consumers inspect `Before`/`After`; do not create per-status
channels. A migration-era `Kind` may remain until Contour 2 eligibility is
fully delta-derived (`CORE-111`).

**Commands vs events:**

| Concept | Meaning |
|---|---|
| Command (`TaskCommand`) | Requested mutation (create, transition, comment, …) |
| Event (`TaskEvent`) | Mutation or external change already observed |

Do not conflate them. Do not put `fswalker` inside the generic launcher
chain. `fswalker` is a Markdown-specific event adapter.

Markdown:

```text
fswalker
→ detect changed file
→ MarkdownProvider reloads task
→ emit TaskEvent
```

Plane:

```text
webhook or polling
→ PlaneProvider reloads task
→ emit TaskEvent
```

After that point, the execution pipeline is provider-independent.

### 3.6 Execution chain

The launcher consumes normalized task events:

```text
TaskEvent
→ load current task through TaskService
→ validate launchability
→ resolve repository
→ resolve agent
→ resolve MCP profile
→ claim task (via TaskService)
→ launch agent
→ track execution
→ return typed ExecutionResult
```

The execution chain must **not**:

- parse Markdown;
- call Plane directly;
- decide how tasks are stored;
- update task files or provider records directly.

### 3.7 Agents and ExecutionResult

Agents do not control task lifecycle. They only return a typed result.

```go
type ExecutionResult struct {
    TaskID      string
    ExecutionID string // runtime claim / session id
    Agent       string
    Outcome     ExecutionOutcome
    Summary     string
    Tests       []string
    Artifacts   []string
    Question    string
    Error       string
}
```

Canonical outcomes:

| Outcome | Meaning |
|---|---|
| `completed` | Work finished with physical artifacts; ready for Alex review |
| `failed` | Execution or work failed |
| `needs_input` | Missing info/approval/access (replaces worker `waiting_input`) |
| `needs_rework` | Worker judges the same task needs another pass before review |
| `blocked` | Worker reports a durable board blocker |
| `cancelled` | Runtime/operator cancelled the session |
| `timed_out` | Runtime timed out the session |
| `orphaned` | Runtime recovered a dead/stale session with no resumable worker |

Compatibility with `CORE-97` `WorkerResult`:

- existing payloads remain accepted during migration;
- `waiting_input` maps to `needs_input`;
- `SuggestedNextStatus` / `Blockers` / `ReviewNotes` fold into comment text
  via the Finalizer (informational only; never applied as a raw status
  write from the worker).

The agent does **not** call `Transition(needs_review)`. Contour 3
(Finalizer) does, through `TaskService`, after consuming `ExecutionResult`.

### 3.8 Finalizer

The Finalizer converts execution outcomes into task transitions. It is an
independent Contour 3 entry point (`ExecutionResult` channel →
`ExecutionFinalizer` → `TaskService.ReportExecution`), not the last step of
Contour 2.

| Outcome | TaskService actions |
|---|---|
| `completed` (+ quality gates passed) | `AddComment(summary…)` → `Transition(needs_review)` |
| `failed` | `AddComment(error…)` → `Transition(blocked)` |
| `needs_input` | `AddComment(question…)` → keep `doing` / session `waiting_input` (HITL pause; 1-1-1 slot held) |
| `needs_rework` | `AddComment(…)` → `Transition(needs_rework)` |
| `blocked` | `AddComment(…)` → `Transition(blocked)` |
| `cancelled` | `AddComment(…)` → `Transition(blocked)` (MVP; not return-to-pickup — see `Fleet/LAUNCH_POLICY.md`) |
| `timed_out` | `AddComment(timeout…)` → `Transition(blocked)` |
| `orphaned` | `AddComment(…)` → `Transition(blocked)` |

Preferred API: `TaskService.ReportExecution(result)` applies this table in
one place. Bot-produced work still stops at `needs_review`; Alex alone
moves to `done`.

This prevents task state and runtime state from diverging.

### 3.9 Dashboard / runtime split

The dashboard is not the manager and not the source of business logic.
It sits outside the three contours (`_docs/TASK_FLOW_CONTOURS.md` §7).

| Surface | Responsibility |
|---|---|
| `TaskService` | tasks, statuses, comments, writes |
| `RuntimeService` (`corechain.RuntimeService`) | sessions, workers, locks, projection snapshot |

Dashboard writes go through `TaskService` (via
`corechain.DashboardSurface.CreateTask` / `PatchTask` adapters).

Dashboard runtime reads go through `RuntimeService.State` /
`ControlSession`.

Dashboard must not subscribe to Go channels, call providers, or
participate in Contour 2/3. Guardrails: `guardrails_test.go` (CORE-114).

Dashboard reads may use a concurrent in-memory snapshot or provider reads.
Do not serialize all reads behind the mutation queue.

---

## 4. Canonical contracts

Go types are finalized in `_backoffice/server/internal/taskflow`. Summary:

### 4.1 Task

Provider-neutral in-memory task. Domain fields follow
`_docs/DOMAIN_MODEL.md`. Storage locators (`Path` / `RelativePath` today)
remain opaque strings; generic code must not parse them as filesystem paths.

Statuses (canonical):

```text
backlog | needs_rework | todo | doing | blocked | needs_review | done | archived
```

Allowed transitions: exactly as in `_docs/DOMAIN_MODEL.md` and
`tasklifecycle.AllowedTaskStatusTransition` today. `TaskService.Transition`
is the only application write path that applies them.

### 4.2 TaskProvider

Persistence adapter only. See §3.4.

### 4.3 TaskService

Application API + private write queue. See §3.2–§3.3.

### 4.4 TaskCommand

Internal to `TaskService`. Requested mutation. Not an event. Not exported to
manager/launcher callers.

### 4.5 TaskEvent

Normalized “change observed” signal for the execution contour. See §3.5 and
`_docs/TASK_FLOW_CONTOURS.md` §4.3.

Target shape (`CORE-111`): `TaskID`, `Before *Task`, `After Task`,
`Source TaskEventSource`. Consumers inspect `Before`/`After`; do not create
per-status channels.

A migration-era `Kind` field may remain until Contour 2 eligibility is fully
derived from deltas.

### 4.6 ExecutionResult

Typed agent/runtime outcome. See §3.7–§3.8 and
`_docs/TASK_FLOW_CONTOURS.md` §6.1 (adds `ExecutionID`, `Agent`, `orphaned`;
keeps `Tests` / `Artifacts` / `Question`).

### 4.7 Status / transition rules

Owned by domain model + `TaskService` validation:

- readiness: `backlog → todo`;
- durable launch lock: `todo|needs_rework → doing` (claim);
- bot completion: `doing → needs_review` only via Finalizer after
  `completed`;
- Alex close: `needs_review → done`;
- operator recovery: `blocked → needs_review` is allowed for manual board /
  TaskService patches when Alex verifies artifacts without re-running the
  agent; Finalizer still only acts from `doing` and must not use this path;
- workers never write status themselves.

---

## 5. Important constraints

- Do not add provider checks inside generic chain steps.
- Do not create parallel manager implementations or parallel provider
  interfaces for the same job.
- Do not keep Markdown parsing inside `fswalker`.
- Do not let launchers or agents update providers directly.
- Do not build a general event-bus framework.
- Use Go channels inside the current process for the write queue and for
  local event delivery.
- Keep commands and events distinct.
- Preserve existing behavior while removing the incorrect abstraction.
- Prefer deleting obsolete pipeline code over wrapping it.
- The refactor should reduce conceptual duplication and ideally reduce
  total code.
- Do not pretend the existing `taskprovider.Provider` half-abstraction is
  the end state; it is a migration waypoint.

---

## 6. Migration map (current → target)

| Current code | Target |
|---|---|
| `manager.TaskProvider` / `RuntimeTaskProvider` | Manager → `TaskService` only; delete parallel interface |
| `corechain.Runtime.CreateTask` / `PatchTask` | Thin wrappers over `TaskService` during migration, then remove |
| `taskprovider.Provider` | Split: persistence → `TaskProvider`; callers → `TaskService` |
| `tasklifecycle.TaskStore` file I/O | Move into `MarkdownProvider` (`CORE-102`) |
| `tasklifecycle` status / finalize / validate | Stay domain-owned; invoked from `TaskService` / Finalizer |
| `corechain.FsWalker` | Markdown event adapter only (`CORE-103`) |
| `corechain.Projector` task parse | `MarkdownProvider.Reload` + emit `TaskEvent` |
| `corechain.ProviderSync` | Plane (or generic) change adapter → `TaskEvent` |
| `corechain.Validator` / launch gates | Execution chain step; loads task via `TaskService` |
| `execution` launch gates + `internal/execution/providers` session runners | Execution chain; no provider writes |
| `tasklifecycle.WorkerResult` + `FinalizeSucceededExecution` | `ExecutionResult` + Finalizer via `ReportExecution` (`CORE-105`) |
| `taskprovider/plane` | Real `PlaneProvider` behind `taskflow.TaskProvider` via `Flow()` + observe (`CORE-106`) |
| Provider `Type()` branches in runtime | Delete in cleanup (`CORE-107`) — replaced by `providerWiring` + `ChangeSource` |

Ordered implementation tasks:

1. `CORE-100` — this document + contracts (done when landed).
2. `CORE-101` — serialized `TaskService`.
3. `CORE-102` — real `MarkdownProvider`.
4. `CORE-103` — provider events / `TaskEvent`.
5. `CORE-104` — manager/commands through `TaskService`.
6. `CORE-105` — launcher + finalizer through `TaskService`.
7. `CORE-106` — `PlaneProvider`.
8. `CORE-107` — remove obsolete filesystem coupling.

Each of `CORE-101`–`CORE-107` must treat this file as architectural truth.

---

## 7. Out of scope for CORE-100

- Migrating providers or rewriting the chain.
- Implementing Plane.
- Building a general event bus.
- Editing generated indexes (`Work/INDEX.md`) by hand.
- Letting workers finalize task status.

Physical artifacts for this architecture pass:

- `_docs/TASK_FLOW_ARCHITECTURE.md` (this file);
- `_backoffice/server/internal/taskflow` contract types;
- follow-up task cards pointing here.

---

## 8. Final ownership after CORE-107

Cleanup (`CORE-107`) removes the remaining filesystem-first and
`Type()`-string coupling. The live ownership model:

| Concern | Owner | Must not |
|---|---|---|
| Detect filesystem changes | `corechain.FsWalker` | Parse tasks, write status, know Plane |
| Parse/persist Markdown cards | `taskprovider/markdown` | Leak into launcher/finalizer |
| Plane API + poll/webhook observe | `taskprovider/plane` (`ChangeSource`) | Appear as `if Type()=="plane"` in generic chain |
| Application writes | `taskflow.TaskService` | Callers talking to Markdown/Plane directly |
| Normalize change → launch | `TaskEvent` → `ExecutionEntry` | Take file paths as launcher input |
| Post-execution transitions | Finalizer via `ReportExecution` | Agents editing task status |
| Dashboard/manager mutations | `TaskService` (`Create`/`Patch`/…) | Parallel `manager.TaskProvider` |
| `Work/INDEX.md` | Markdown factory wiring only | Rebuild under Plane |

Runtime composition root (`newTaskProvider`) may switch on config to
construct Markdown vs Plane. After that point, generic steps use
capability wiring (`providerWiring.pollInterval`,
`rebuildDerivedViews`) and optional interfaces (`ChangeSource`) — never
`Provider.Type()` string branches.

`taskprovider.Provider` remains a migration-era bootstrap/List/Load
surface. New write paths use `TaskService` only.

---

## 9. CORE-115 review and CORE-116 package split

CORE-115 found that the contours were represented by independent wiring but
the package split was incomplete and `ContourTopology` remained metadata-only
architecture. CORE-116 replaces that map with concrete package boundaries:
`audit` owns event logging, `board` owns dashboard projections, `taskprovider/markdown`
owns Markdown persistence plus filesystem detection, `execution` owns provider-neutral launch
admission, `executionfinalizer` owns outcome application, and
`executionstate` owns session persistence support. Shared launch/session/
capability contracts now live in `internal/execution`, while provider-specific
backend identity, provider session helper parsing, and provider runners now
live in `internal/execution/providers`, not in `corechain`. Proxy constructors and
legacy wrapper types were removed from `corechain` in a rework pass; the
composition root imports sibling packages directly.

`corechain` remains the composition root and still owns runtime scheduling, but
provider session runners now live in `internal/execution/providers` alongside
the provider planners and helper parsers they supervise.
Its in-memory runtime `Store` was also split internally into separate board/task
projection state and execution/session state, even though the composition root
still exposes one dashboard-facing facade.
Composition must call ownership packages directly (no `NewFsWalker` /
`NewValidator` proxies or legacy test-only wrapper structs). Import-boundary,
runtime-wiring, and no-proxy tests are the live guardrails;
`HasContinuousFsWalkerToAgent` and `ContourTopology` are removed.
