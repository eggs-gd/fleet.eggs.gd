# Task Flow Contours

Canonical architecture for independent Core task-flow contours
(`CORE-108` … `CORE-114`).

This document is the architectural source of truth for the contours refactor.
Implementation tasks `CORE-109` through `CORE-114` must follow these
boundaries. The final review task `CORE-115` reconciles code against this
file and `_docs/TASK_FLOW_ARCHITECTURE.md`.

Related durable contracts:

- `_docs/TASK_FLOW_ARCHITECTURE.md` — application contracts (`TaskService`,
  providers, status rules) from `CORE-100`…`CORE-107`. Contours **supersede**
  that document’s single vertical pipeline diagram; shared contracts remain.
- `_docs/DOMAIN_MODEL.md` — task schema and status state machine.
- Data root `_docs/OPERATING_MODEL.md` — Manager / worker / Alex roles.
- `_docs/CORE_MANAGER_API.md` — Manager capture surface.
- `_docs/AGENT_LAUNCHER.md` — launch eligibility and worker prompt shape.
- Go package `internal/taskflow` — compile-checked contract types (migrate
  toward the shapes below during `CORE-111`…`CORE-113`).

This task (`CORE-108`) documents the target. It does **not** implement the
refactor.

---

## 1. Hard rule: no single end-to-end chain

There is **no single end-to-end chain**.

The old continuous pipeline:

```text
fswalker
→ parser
→ task
→ validator
→ launcher
→ agent
→ finalizer
```

must be deleted as one continuous control-flow shape.

Do **not**:

- reorganize its steps into a prettier chain;
- insert providers into the middle of it;
- preserve it under new package names;
- treat “FsWalker emits TaskEvent into the same ChainProcessor that later
  launches agents” as compliance.

After the contours refactor, there must be **no** control-flow path that
begins with `FsWalker` / a filesystem path / a Markdown document / a Plane
payload / a Manager request and ends with an agent as one continuous chain.

The expected result is multiple independent entry points:

| Entry point | Contour |
|---|---|
| Manager / commands | Task management |
| `TaskEvent` channel | Execution |
| `ExecutionResult` channel | Finalization |
| Dashboard REST | Outside contours (poll + command only) |

They share `TaskService`. They do **not** form one chain through it.

---

## 2. Complete architecture

These are separate systems, not one vertical pipeline:

```text
+--------------------------------------+
| CONTOUR 1: TASK MANAGEMENT           |
|                                      |
| Human -> Manager Chain -> TaskService|
+--------------------------------------+

+--------------------------------------+
| TASK STORAGE AND CHANGE DETECTION    |
|                                      |
| Markdown <-> TaskService <-> Plane   |
|    |                       |         |
| FsWalker               Webhook/Poll  |
|    +-------- TaskEvent ----+         |
+--------------------------------------+

+--------------------------------------+
| CONTOUR 2: EXECUTION                 |
|                                      |
| TaskEvent -> Execution Chain -> Agent|
|                              |       |
|                       ExecutionResult|
+--------------------------------------+

+--------------------------------------+
| CONTOUR 3: FINALIZATION              |
|                                      |
| ExecutionResult -> Finalizer         |
|                         |            |
|                    TaskService       |
+--------------------------------------+

+--------------------------------------+
| DASHBOARD                            |
|                                      |
| REST -> TaskService / RuntimeService |
+--------------------------------------+
```

Shared `TaskService` fan-in (not a chain):

```text
Manager contour --------+
Dashboard REST ---------+--> TaskService
Finalizer contour ------+
```

---

## 3. Contour 1: Task Management

```text
Human / Voice / Codex Manager override
              |
              v
        Manager Chain
 parse -> resolve -> decompose
              |
              v
         TaskService API
              |
              v
         Task Provider
       +------+------+
       |             |
   Markdown        Plane
```

The Manager contour **ends** after `TaskService` accepts the command.

It does not continue into scheduling or agent launch.

The Manager **may**:

- create tasks;
- update / patch tasks;
- query tasks;
- add comments;
- return results to the operator.

The Manager **must not**:

- watch task changes;
- launch agents;
- wait for executions;
- finalize executions;
- contain `FsWalker`;
- call Markdown or Plane providers directly.

Dashboard task commands that mutate the board use the same management path
(`TaskService`) where applicable. Dashboard stays outside Contour 1 for
runtime/session concerns (see §6).

---

## 4. Task storage and change detection

Change detection is **not** Contour 1 and **not** an extension of Manager
processing. It feeds a generic event channel that Contour 2 consumes.

### 4.1 Markdown event source

`FsWalker` (package `taskprovider/markdown`; historically in `tasklistener`
and earlier in `corechain`) is
**Markdown-specific filesystem change detection only**.

It is not a generic execution-chain step. It must not parse launch policy,
claim tasks, or start agents.

```text
Filesystem
    |
    v
FsWalker
    | changed path
    v
Markdown Provider reload
    |
    v
TaskService reconcile
    |
    v
TaskEvent channel
```

### 4.2 Plane event source

```text
Plane webhook / polling
          |
          v
 Plane Provider reload
          |
          v
 TaskService reconcile
          |
          v
  TaskEvent channel
```

### 4.3 One generic event channel

Both sources end at the same channel:

```go
taskEvents chan TaskEvent
```

Do not create separate event types or separate channels per status.
Consumers inspect `Before` and `After`.

Canonical event shape (target for `CORE-111`):

```go
type TaskEventSource string

const (
    TaskEventSourceMarkdownFS    TaskEventSource = "markdown_fs"
    TaskEventSourcePlanePoll     TaskEventSource = "plane_poll"
    TaskEventSourcePlaneWebhook  TaskEventSource = "plane_webhook"
    TaskEventSourceTaskService   TaskEventSource = "task_service"
)

type TaskEvent struct {
    TaskID string
    Before *Task // nil when the task did not previously exist
    After  Task  // zero/tombstone semantics for delete are implementation-defined;
                 // deleted tasks must be skippable by Contour 2
    Source TaskEventSource
}
```

Notes:

- `TaskService` publishes one generic update event after canonical state
  changes (including Manager/dashboard writes and provider reconcile).
- Provider-specific payloads (file paths, Plane webhook bodies) must never
  enter Contour 2.
- A `Kind` field may exist temporarily during migration, but Contour 2
  eligibility must be derived from `Before`/`After` (and `Source` for
  diagnostics), not from provider-specific side channels.

---

## 5. Contour 2: Execution

This contour detects eligible work from normalized task changes and starts
agents. It is **not** an extension of Contour 1.

```text
TaskEvent channel
        |
        v
Execution Chain
 inspect delta (Before/After)
 -> validate eligibility
 -> acquire lock / claim
 -> resolve repository
 -> resolve agent
 -> launch runtime
        |
        v
Agent process/session
        |
        v
ExecutionResult channel
```

The execution chain **begins with `TaskEvent`**.

It must **never** begin with:

- a filesystem path;
- a Markdown document;
- a Plane webhook payload;
- a Manager request.

Rules:

- Plane and Markdown are indistinguishable inside Contour 2.
- No provider-specific `if Type() == …` checks in execution steps.
- Contour 2 publishes `ExecutionResult`; it does **not** finalize durable
  task lifecycle (that is Contour 3).
- Claims / reads needed for launch go through `TaskService` only.

---

## 6. Contour 3: Finalization

This contour consumes normalized execution outcomes.

It is **not** appended as the last step of Contour 2’s Chain of
Responsibility. It has its own entry point: the `ExecutionResult` channel
(or equivalent independent consumer wired at composition root).

```text
ExecutionResult channel
          |
          v
      Finalizer
          |
          v
    TaskService API
          |
          v
      Task Provider
```

### 6.1 ExecutionResult

The launcher/runtime normalizes process/session state before publishing:

```go
type ExecutionResult struct {
    TaskID      string
    ExecutionID string // runtime claim / session id
    Agent       string
    Outcome     ExecutionOutcome
    Summary     string
    Error       string
    // Operator-facing extras already used by workers (keep):
    Tests     []string
    Artifacts []string
    Question  string
}
```

Canonical outcomes:

```text
completed
failed
blocked
needs_input
needs_rework
cancelled
timed_out
orphaned
```

Legacy worker string `waiting_input` normalizes to `needs_input`.

### 6.2 Outcome → task state

The Finalizer only maps execution outcome to task state through
`TaskService` (preferably `ReportExecution`):

| Outcome | Task status |
|---|---|
| `completed` | `needs_review` |
| `failed` | `blocked` |
| `timed_out` | `blocked` |
| `orphaned` | `blocked` |
| `cancelled` | `blocked` (MVP chooses blocked, not return-to-pickup; documented in `MapExecutionOutcomeToStatus` / CORE-113) |
| `needs_input` | pause: task stays `doing`, session `waiting_input`, 1-1-1 slot held |
| `needs_rework` | `needs_rework` |
| `blocked` | `blocked` |

No hard quality gates are part of this contour yet (CORE-117). CORE-120 appends
a soft missing-tool warning into Finalizer comments when required MCP tools were
not observed in session evidence; it does not block `needs_review`.

Quality review will later use additional board statuses and dedicated agents.
Bot-produced `completed` work still stops at `needs_review`; Alex alone moves
to `done`.

Naming note: today’s `corechain.CoreFinalizer` is a **change-detection**
step (index rebuild + forward `TaskEvent`). It is **not** Contour 3. Contour
3 is `corechain.ExecutionFinalizer`, which consumes `ExecutionResult` and
calls `TaskService.ReportExecution`.

---

## 7. Dashboard / runtime boundary

The dashboard is outside all three contours.

```text
Dashboard
   +-- REST -> TaskService
   +-- REST -> RuntimeService
```

It polls state and sends commands.

It does **not**:

- subscribe to Go channels;
- participate in execution;
- own business lifecycle logic beyond calling `TaskService`.

### 7.1 Final wiring (CORE-114)

Implemented in `server`:

| REST | Boundary | Implementation |
|---|---|---|
| `GET /api/state` | `RuntimeService.State` | in-memory projection snapshot |
| `POST /api/sessions/control` | `RuntimeService.ControlSession` | operator session commands |
| `PATCH /api/tasks` | `TaskService` via `DashboardSurface.PatchTask` | status/assignee/comment writes |
| `POST /api/tasks` | `TaskService` via `DashboardSurface.CreateTask` | create writes |

Go contracts:

- `corechain.RuntimeService` — poll + session control only;
- `corechain.DashboardSurface` — `RuntimeService` + `TaskService()` +
  Create/Patch adapters;
- `internal/server.registerDashboardRoutes` accepts `DashboardSurface` only.

Structural guardrails (package `internal/corechain` tests + server route
tests) reject channel subscription, contour processor references, and
provider imports from the HTTP dashboard package. The Svelte backoffice
polls `/api/state` and posts commands — it does not open WebSockets or
EventSource streams into Go channels.

---

## 8. Shared TaskService

All mutations are serialized internally:

```text
TaskService method
       |
       v
private command channel
       |
       v
single write loop
       |
       v
TaskProvider
```

Callers never receive the write channel. Reads do not need the write queue.

Do not build a general event-bus framework. Use in-process Go channels for
the write queue, `TaskEvent`, and `ExecutionResult` delivery.

---

## 9. Hard rejection criteria

The contours refactor is **rejected** if:

1. one chain still starts with `FsWalker` / fswalker and ends with an agent;
2. Manager processing flows directly into execution;
3. `FsWalker` remains a generic chain step;
4. Plane and Markdown enter execution through different paths;
5. provider-specific checks remain inside execution steps;
6. Finalizer is merely the final step of the old chain;
7. code was moved between packages but the control flow stayed the same.

Passing means independent entry points exist in wiring and tests — not only
in package names.

### 9.1 Guardrail coverage (CORE-114)

Automated checks live in:

- `internal/corechain/import_boundaries_test.go` — package imports,
  provider-neutral execution source, independent service wiring, and dashboard
  channel/execution coupling;
- `internal/server/dashboard_test.go` — REST → TaskService / RuntimeService
  route binding.

CORE-116 replaced metadata-only topology assertions with Go package-import and
runtime-wiring tests.

---

## 10. Reality vs this document

### After CORE-109…CORE-115

Contours `CORE-109`…`CORE-114` deleted the continuous FsWalker→agent
`ChainProcessor` and introduced independent Process entry points plus
shared channels. `CORE-115` confirmed that split — and also found that
`internal/corechain` remained a package monolith with metadata-only
`ContourTopology` guardrails.

| Topic | After contours (pre CORE-116) | Doc target |
|---|---|---|
| Process shape | Independent Process contours + shared channels | Independent entry points |
| Package shape | Still mostly `corechain` monolith | Real module packages |
| Topology metadata | Hand-maintained `ContourTopology` map | Must not encode architecture as metadata |
| `TaskEvent` / execution / finalizer | Channel contracts matched docs | Keep |
| Manager / Dashboard | Contour 1 + REST boundaries matched | Keep |

### After CORE-116 (rework)

| Topic | Code after CORE-116 rework | Doc target |
|---|---|---|
| Packages | `taskprovider/markdown`, `execution`, `executionfinalizer`, `board`, `executionstate`, `audit`; `corechain` wires them directly | Ownership by lifecycle |
| Proxies | Removed — no `NewFsWalker`/`NewExecutionEntry`/legacy test types in `corechain` | Real imports only |
| Topology metadata | Removed | No metadata architecture |
| Guardrails | Import + wiring tests | Real wiring checks |
| Still in `corechain` | Provider runners, session control, projector/provider sync, joined Store facade | Documented residue in `MODULES.md` |

Shared pieces retained:

- serialized `taskflow.TaskService` write queue;
- Markdown + Plane `Flow()` providers behind the service;
- Plane `ChangeSource` + `ProviderSync`;
- `execution.Entry` as execution gate;
- `ReportExecution` outcome table.

There is **no** single end-to-end ChainProcessor from FsWalker to agent
launch, and architecture is no longer claimed via a topology map.

---

## 11. Implementation map

| Ref | Focus |
|---|---|
| `CORE-108` | This document (architecture only) |
| `CORE-109` | Remove continuous FsWalker→agent chain |
| `CORE-110` | Isolate Contour 1 (Manager → TaskService only) |
| `CORE-111` | Normalize change detection → `TaskEvent{Before,After,Source}` |
| `CORE-112` | Contour 2 starts only from `TaskEvent` |
| `CORE-113` | Contour 3 from `ExecutionResult` |
| `CORE-114` | Dashboard boundary + rejection-criteria guardrails |
| `CORE-115` | Review implementation vs docs |
| `CORE-116` | Real package split; topology metadata removed |

Ordered guidance: break the continuous chain (`CORE-109`) before or while
isolating contours; do not “finish” Contour 2/3 by appending steps onto the
old ChainProcessor.

---

## 12. Out of scope for CORE-108

- Implementing the contour split in Go.
- Migrating `TaskEvent` / `ExecutionResult` types in code.
- Editing generated indexes (`Work/INDEX.md`).
- Letting workers finalize task status to `done`.

---

## 13. CORE-115 review and CORE-116 implementation status

CORE-115 found that the control flow had been split conceptually, but the
implementation still concentrated package ownership in `corechain` and used
`ContourTopology` metadata as an architectural claim rather than enforcement.

CORE-116 removes that metadata and introduces concrete packages: `audit`,
`board`, `taskprovider/markdown`, `execution`, `executionfinalizer`, and
`executionstate`. A rework pass removed `corechain` proxy constructors and
legacy wrapper types; composition imports sibling packages directly. Shared
execution-facing launch/session/capability contracts now live in
`internal/execution`, while Claude/Codex/Cursor/Gemini planners, provider
session helpers, and provider session runners live in
`internal/execution/providers`.
`corechain` remains the composition root for runtime scheduling.

Guardrails are now import-boundary, runtime-wiring, and no-proxy tests. They
assert independent services, ban `corechain` forwarding constructors, and
prevent provider branches from entering `execution`; they do not rely on
`HasContinuousFsWalkerToAgent` or a topology map. The runtime store is also no
longer one flat bag of task/workspace/session/orphan maps: board/task
projection state and execution/session state are separated internally even
though `corechain` still serves one composed dashboard/runtime surface.
