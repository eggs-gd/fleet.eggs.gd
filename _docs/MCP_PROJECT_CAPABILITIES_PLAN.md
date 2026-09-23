# Project Capabilities — MCP Scoping (Plan)

> **Status: draft plan, pre-inventory.** Authored externally (ChatGPT) from
> conversational context about Core's intent, without access to the actual
> codebase. Treat every persistence/API/provider claim below as a proposal,
> not a fact about the current implementation — Phase 0 below is required
> before any of it is built. See the "Phase 0 Inventory Findings" section
> (appended after this plan once that inventory is done) for how this plan
> reconciles with what Core actually does today.

Ось я б так і кинув Cursor/девелоперу. Спеціально не фіксую наперед
storage/config implementation, де ми ще не знаємо поточну інфраструктуру:
спочатку нехай зробить inventory і поставить питання.

## Overview

Introduce project-scoped capabilities.

MCP servers are registered globally once, but each project explicitly
controls which MCP servers are exposed to agents working in that project.

Core principle:

> Registry is global. Activation is project-scoped.

This solves tool ambiguity and prevents every executor from receiving every
globally configured MCP.

Example:

```
Global MCP Registry
├── LMS
├── Node-RED
├── Gmail
├── Google Drive
└── GitHub

Smart Home
├── LMS ✓
├── Node-RED ✓
└── everything else disabled

Mail
└── Gmail ✓

Career
├── Gmail ✓
├── Google Drive ✓
└── GitHub ✓
```

A project is therefore not necessarily a source-code repository. It is a
context boundary containing instructions, files/knowledge, available
capabilities and work.

Do not introduce specialized `MailAgent`, `SmartHomeAgent`, etc. Existing
general-purpose executors should receive different context and tool sets
depending on project.

## Product behavior

The intended execution flow is:

```
User request
    ↓
Manager
    ↓
select project/context
    ↓
Project
├── directory/context
├── AGENTS.md / instructions
├── enabled MCP capabilities
└── allowed executors
    ↓
Executor
```

Example:

```
"make the volume louder"
        ↓
Manager
        ↓
Smart Home
        ↓
Executor receives:
- Smart Home context
- Smart Home instructions
- LMS MCP
- Node-RED MCP

Executor does NOT receive:
- unrelated filesystem/device tools
- Gmail MCP
- Career tools
- other globally registered MCPs
```

The goal is to reduce the executor's action space rather than solve
ambiguity with increasingly complicated prompts.

## Phase 0 — Infrastructure inventory first

Do not start by building UI.

Inspect the current implementation and document:

```
Concern                 Current implementation
────────────────────────────────────────────────
MCP discovery
MCP configuration
MCP persistence
MCP process lifecycle
Agent MCP injection
Manager MCP injection
Project model
Project persistence
Session creation
Executor launch
AGENTS.md/context loading
CLI-specific MCP behavior
Global MCP configuration
```

Especially determine how Claude, Codex and Cursor currently receive MCP
configuration.

We need to know whether MCP configuration is:

- passed at process launch;
- inherited from CLI/global configuration;
- loaded from project-local configuration;
- injected dynamically;
- impossible to isolate with a particular provider.

Do not assume all providers support identical MCP scoping.

After inventory, identify blockers and ask questions before introducing a
second configuration system.

## Domain model

Conceptually introduce two separate things.

### MCP Registry

Global definition of MCP servers known to Core.

Conceptual model:

```
MCPServer
  id
  name
  transport
  configuration
  status
  source
```

Exact fields must follow existing MCP infrastructure rather than creating
unnecessary abstractions.

### Project Capability Assignment

Many-to-many relationship:

```
Project <-> MCPServer
```

Conceptually:

```
ProjectCapabilities
  projectId
  enabledMcpIds[]
```

Do not duplicate MCP connection/configuration data inside projects. A
project stores only the association.

Therefore:

```
MCP definition → global
MCP credentials/config → global
MCP enabled state → per project
```

### Effective MCP set

There must be one backend resolver responsible for determining MCP
visibility.

Conceptually:

```
ResolveMCPs(projectID) -> []MCPServer
```

Every executor launch for project work must use this resolver.

Do not implement separate filtering logic in:

- UI;
- Claude adapter;
- Codex adapter;
- Cursor adapter;
- scheduler;
- session resume.

Provider adapters should consume the already-resolved capability set.

Desired pipeline:

```
project
   ↓
CapabilityResolver
   ↓
effective MCP set
   ↓
provider adapter
   ↓
executor
```

## Existing global MCP behavior

Do not silently break current installations.

Today some MCPs may effectively be global. Inventory existing behavior and
define a migration strategy.

Preferred end-state:

> Global registry ≠ globally exposed.

However, existing configurations may initially need compatibility behavior.

Before implementing migration, report:

- what MCPs currently exist;
- where they are configured;
- whether project association can be inferred;
- whether changing behavior would break existing sessions/workflows.

No destructive automatic migration.

## Project UI

Add a project-level Capabilities section. Exact placement should follow
existing UI structure rather than redesigning Work.

Example:

```
Smart Home
──────────────────────────────

Capabilities

MCP Servers

✓ Logitech Media Server
✓ Node-RED
○ Gmail
○ Google Drive
○ GitHub

+ Add MCP
```

All globally registered MCP servers are visible.

Toggle means:

```
ON  → available to executors in this project
OFF → unavailable to executors in this project
```

This is project configuration, not MCP deletion.

Keep the UI dense and consistent with existing Settings. No giant cards.

### Add MCP from project

`+ Add MCP` opens MCP configuration UI.

When an MCP is created from inside a project:

```
1. Add MCP to global registry
2. Enable it for current project
3. Leave it disabled for every other project
```

Example:

```
Smart Home → + Add MCP → Home Assistant

Global Registry:
Home Assistant ✓ created

Smart Home:
Home Assistant ✓ enabled

Career:
Home Assistant ○

Mail:
Home Assistant ○
```

This behavior is important.

### Add MCP globally

Global Settings should eventually expose:

```
Settings
└── Integrations / MCP
```

Adding an MCP there:

```
1. Add to global registry
2. Do NOT automatically enable it for projects
```

This may reuse existing Settings Integrations UI if appropriate. Do not
build a duplicate MCP editor.

### Removing MCP

There are two distinct actions.

#### Remove from project

Means:

```
disable project association
```

The MCP remains globally registered and may remain active in other
projects.

#### Delete globally

Before deleting, calculate usage.

Example:

```
Delete "Gmail"?

Used by:
- Mail
- Career
- Content

Deleting it will remove this capability
from 3 projects.
```

Require explicit confirmation. No silent cascading deletion.

## Runtime semantics

When starting a new project executor/session, Core resolves that project's
current MCP capabilities.

Disabled MCPs must not be exposed to that executor where provider
infrastructure allows actual isolation.

Important distinction:

```
configured globally
≠
visible to project
≠
available to executor
```

Tests must verify actual executor visibility, not merely GUI state.

### Existing sessions

Do not unexpectedly mutate or kill running sessions when capability
configuration changes.

Initial conservative semantics:

```
Capability changes
→ affect new sessions / launches
→ existing session remains untouched
```

If provider/session architecture allows safe dynamic capability changes,
document it, but don't introduce it merely for this feature.

UI may show: "Applies to new sessions." if this limitation exists.

## Manager

Manager is special and must not automatically inherit the union of every
project's MCPs. First inspect current behavior.

Desired conceptual separation:

```
Manager capabilities
        ≠
Project capabilities
```

Manager's primary responsibility remains routing/coordinating. For "make
the volume louder", Manager should ideally identify `Smart Home` and
delegate into that context rather than require every Smart Home MCP
globally.

However, some cheap/direct actions may currently be performed by Manager
itself. Do not break that behavior in this phase.

Document current manager tool visibility and ask before changing it. This
is an explicit design question, not something to silently decide during
implementation.

## Agents / executors

Do not create domain-specific agent types. Existing providers remain:

```
Claude
Codex
Cursor
...
```

Project capability configuration controls their tool environment.

Future project-level executor permissions may conceptually look like:

```
Project
├── Context
├── Capabilities
│   └── MCP
└── Executors
    ├── Claude ✓
    ├── Codex ✓
    └── Cursor ○
```

But project-scoped executor enablement is not required in this phase
unless equivalent infrastructure already exists and adding it is trivial.
Do not expand scope just because the model is symmetrical.

## Security

MCP credentials/secrets must not be returned to normal Settings/project
APIs after save.

UI may receive:

```
configured: true
credentialStatus: present
```

not the credential itself.

Project association must reference MCP IDs and must not duplicate
credentials.

No secret values in:

- project files;
- project registry;
- frontend state snapshots;
- logs;
- task metadata.

Follow existing secret-storage behavior where available.

## Diagnostics

Extend diagnostics to detect at least:

```
MCP registered but unhealthy
MCP enabled for project but unavailable
MCP referenced by project but missing from registry
MCP configuration invalid
Provider cannot honor requested project MCP isolation
```

The last one is particularly important. If e.g. some CLI can only see
globally configured MCPs, UI must not claim isolation that runtime does
not actually provide.

Prefer: "Scoped isolation unsupported by this provider" over fake
security/capability boundaries.

## API

Do not commit to exact routes until existing project/settings APIs are
inspected.

Conceptually we need operations equivalent to:

```
GET global MCP registry

CREATE MCP
UPDATE MCP
DELETE MCP

GET project capabilities
ENABLE MCP for project
DISABLE MCP for project

RECHECK MCP
```

Prefer extending existing Settings/project APIs over introducing an
unrelated API family. Backend owns validation and effective capability
resolution.

## Persistence

Do not choose YAML/database/project files before inspecting current
persistence.

Requirements:

- global MCP definitions have one source of truth;
- project associations have one source of truth;
- associations survive Core restart;
- project association references stable MCP IDs;
- project config must not duplicate secrets;
- machine-specific MCP configuration should not accidentally enter a
  public Git repository;
- writes must be atomic where file-backed.

If current `core.local.yaml` from Settings Phase 3 is the appropriate
source, reuse it. Do not create `mcp.local.yaml`, random project YAMLs,
and another registry unless there is a concrete reason.

## Important UX invariant

The GUI should communicate this mental model without requiring the user
to understand MCP configuration internals:

> This project can use these tools.

Not:

> Here's a YAML fragment that gets passed to Claude.

For a normal user, creating a domain should eventually feel approximately
like:

```
Create Project: Smart Home

Context:
~/Projects/smart-home

Capabilities:
☑ LMS
☑ Node-RED

Instructions:
"Music means LMS.
Default player is Studio.
Never change system volume on the host."
```

Done.

## Acceptance scenario

This scenario should be used as the primary end-to-end test.

Global MCP registry contains:

```
LMS
Gmail
GitHub
```

Projects:

```
Smart Home
  enabled MCPs: LMS

Mail
  enabled MCPs: Gmail

Career
  enabled MCPs: Gmail, GitHub
```

Start executor in Smart Home. Expected:

```
LMS     visible
Gmail   NOT visible
GitHub  NOT visible
```

Start executor in Mail:

```
Gmail   visible
LMS     NOT visible
GitHub  NOT visible
```

Add `Node-RED` through:

```
Smart Home → + Add MCP
```

Expected:

```
Global registry:
Node-RED exists

Smart Home:
Node-RED enabled

Mail:
Node-RED disabled

Career:
Node-RED disabled
```

Restart Core. All associations remain intact. Existing active sessions
are not unexpectedly killed.

## Tests

At minimum:

### Backend

- global registry persistence;
- project association persistence;
- effective capability resolver;
- enable/disable;
- duplicate MCP handling;
- global delete usage detection;
- missing MCP reference;
- secret redaction;
- restart persistence.

### Provider integration

For every supported provider, verify whether project-level MCP isolation
is actually honored. Do not test only the config object; verify what the
launched executor receives.

### Frontend

- global MCP list;
- project toggles;
- Add MCP from project automatically enables current project;
- adding globally does not enable any project;
- dirty/save behavior if applicable;
- delete usage warning;
- unhealthy/unsupported state.

## Explicit non-goals

Do not include in this increment unless implementation inventory proves
it is already essentially free:

- specialized Mail/Home/Career agents;
- rewriting Manager routing;
- multi-root;
- remote/mobile control;
- Firebase/cloud relay;
- agent marketplace;
- MCP marketplace;
- automatic MCP installation;
- dynamic modification of running sessions;
- project-specific credentials;
- project-specific copies of MCP definitions;
- arbitrary workflow/concurrency changes;
- Work redesign.

## Implementation order

```
0. Inventory existing MCP + project + provider infrastructure
1. Report blockers/questions
2. Define global registry + project association persistence
3. Implement single effective capability resolver
4. Wire provider launch paths through resolver
5. Add project Capabilities API
6. Add project Capabilities UI
7. Add MCP creation from project
8. Add global usage/delete semantics
9. Diagnostics
10. E2E provider verification
```

Do not fake provider isolation. If step 0 reveals that Claude/Codex/Cursor
handle MCP configuration fundamentally differently, stop after the
inventory and propose the smallest common runtime contract before
proceeding.

---

## Phase 0 Inventory Findings (Claude, 2026-09-22)

Read against the actual `core.eggs.gd` codebase, not from conversational
context. Bottom line up front: **almost none of this plan's assumed
Core-side infrastructure exists yet** — no MCP registry, no project
persistence layer at all. The three providers *do* each have their own
project-scoped MCP mechanism (see the 2026-09-22 correction below,
prompted by the user's review), though not an identical one — Codex's
requires an extra trust step the other two don't.

| Concern | Current implementation |
|---|---|
| MCP discovery | `internal/settings/mcp.go:loadMCP` reads **one** file: `.mcp.json` at the Core root (the repo's own dev `.mcp.json` — `svelte`, `gopls`, `design-patterns`). Read-only, for the Settings > Integrations diagnostics page. |
| MCP configuration | Same file. No create/update/delete API exists — `.mcp.json` is edited by hand. |
| MCP persistence | No Core-owned registry. `loadMCP`'s own source-role string says it outright: *"Core daemon does not currently inject these servers into Claude/Codex/Cursor launches."* |
| MCP process lifecycle | None. Core never starts/stops an MCP server process; it only shells out to check if the configured `command` resolves on PATH (`detectMCP`). |
| Agent MCP injection | **None, for any provider.** `internal/execution/providers/adapters.go` and `runtime.go` build launch commands (`claude --bg …`, `codex app-server --stdio`, cursor-agent invocation) with zero MCP-related flags or config. Whatever MCP set an executor sees today is 100% that provider's own native discovery, invisible to Core. |
| Manager MCP injection | N/A by construction — `internal/manager.Service` is explicitly barred (AGENTS.md, enforced by `internal/server/import_boundaries_test.go`) from importing `internal/execution` or `internal/execution/providers` at all. Manager has no tool/MCP access today, full stop. `"MCP"`/`"mcp"` only appears in `manager/vocabulary.go` as a recognized text token for `@mention` parsing — cosmetic, not functional. |
| Project model | `internal/board.Project` (`projection.go`) is a **derived projection**, rebuilt from a filesystem/git scan on every read (id, workspace_id, title, repositories, technology). It is not a persisted, mutable entity — there is nowhere to hang a `enabledMcpIds[]` field. |
| Project persistence | **Does not exist.** The only per-project UI (`ProjectSettings.svelte` / `lib/projectSettings.js`, added this session) is a read-only hierarchy/technology inspector — no fetch-for-write, no PATCH endpoint, no file backing it. `core.local.yaml` (via `internal/settings/overlay.go`) is the only mutable local-config file, and it is scoped to `scanRoot` + per-agent overlay + the new Manager binding — nothing project-shaped. |
| Session creation / Executor launch | `internal/execution/host_launch.go:startLaunchCandidate` — always claims a real `tasklifecycle.Task` first, reserves a project/agent concurrency slot, then dispatches through `providers.RunBackgroundRemoteSession` / `RunCodexAppServerSession` / `RunCursorVisibleSession`. Every launch is task-bound; there is no project-level "start an executor with these capabilities" primitive independent of a task. |
| AGENTS.md/context loading | Each provider's own CLI reads `AGENTS.md`/equivalent from its working directory natively; Core does not inject or template instructions beyond the per-agent `routingInstructions` overlay field (`Fleet/<agent>.md` "Best At" text, surfaced in Settings > Agents/Workers). |
| CLI-specific MCP behavior | **Confirmed to differ fundamentally** (see below) — this is the plan's key open question, and it does not have a common answer. |
| Global MCP configuration | Only the one Core-root `.mcp.json`, and only for diagnostics display. |

### CLI-specific MCP behavior (the load-bearing finding)

- **Claude Code** — auto-discovers `.mcp.json` from its **current working
  directory** at launch (native Claude Code behavior, nothing to do with
  Core). Core already launches every Claude session with `cmd.Dir` set to
  the task's project working directory. So project-scoped MCP for Claude
  is **almost free**: if each project repo carries its own `.mcp.json`,
  Claude already only sees that project's servers today, with zero Core
  plumbing. Global suppression/allow-list is a separate question (Claude
  also reads `~/.claude.json`/user settings).
- **Cursor** (`cursor-agent`) — same shape: `.cursor/mcp.json` (project,
  CWD-relative) or `~/.cursor/mcp.json` (global), **plus** it already ships
  a native per-server `enable`/`disable` primitive (`cursor-agent mcp
  enable|disable <identifier>`, against "the local approved list"). This
  is the closest thing to the plan's toggle model that already exists,
  natively, in a provider CLI.
- **Codex** — ~~fundamentally different, no per-project mechanism~~
  **CORRECTED (2026-09-22, see below): Codex does have a per-project config
  layer, `<project>/.codex/config.toml`, gated behind explicit trust.**
  The original claim above was wrong — verified two independent ways: `codex
  app-server`'s own `config/read {cwd, includeLayers: true}` RPC method
  (schema description: *"return the effective config as seen from that
  directory ... including any project layers between cwd and the
  project/repo root"*), and a live test against a real spawned app-server.
  See "Correction: Codex project-scoped config" below for the full
  mechanics — the short version is that it exists, is queryable over the
  same protocol Core already speaks to Codex, but is off by default per
  path and gates more than just MCP.

### Correction: Codex project-scoped config (2026-09-22)

The first pass above concluded Codex has no per-project MCP mechanism.
That was wrong, and the user's own review (via a second AI, working from
current OpenAI docs) caught it. Re-verified empirically rather than
taking either side's word for it — spawned a real `codex app-server
--stdio`, wrote a throwaway `<tmp-project>/.codex/config.toml` with a
uniquely-named `[mcp_servers.zzz_project_probe]` entry, and called
`config/read` with `cwd` set to that directory. Findings:

1. **The project layer is real and is detected.** The response's `layers`
   array included an entry `{"name": {"type": "project", "dotCodexFolder":
   "<path>/.codex"}}` whose parsed `config.mcp_servers` contained my test
   server verbatim — Codex parses `<project>/.codex/config.toml` and knows
   about it.
2. **It is disabled by default, per exact absolute path.** That layer
   carried `"disabledReason": "To load project-local config, hooks, and
   exec policies, add <path> as a trusted project in
   ~/.codex/config.toml."` — i.e. project config only takes effect once
   that literal path is marked `trusted` in the **global**
   `~/.codex/config.toml`'s `[projects."<path>"]` table (this machine
   already has over a dozen such entries, presumably from answering
   Codex's own interactive "trust this folder?" prompt over time).
3. **Trust does not cascade to subdirectories.** Confirmed by testing a
   fresh directory *inside* an already-trusted ancestor
   (`/Users/operator/Projects`, itself trusted) — the subdirectory's own
   `.codex/config.toml` layer still came back disabled. Trust is keyed by
   exact path, not prefix.
4. **The gate is not MCP-specific.** The disabled reason names "config,
   hooks, and exec policies" together — trusting a project for its MCP
   servers also trusts it for hook execution and sandbox/exec policy
   overrides. There is no narrower "just trust the MCP list" grant.
5. This is queryable and (presumably) writable through the same
   `codex app-server` protocol Core already drives (`config/read` to
   inspect; the trust table itself lives in the user config layer, which
   `config/batchWrite` — seen in the same schema — can plausibly edit,
   though write access wasn't exercised in this pass).

**Revised conclusion:** Codex is not "unsupported" — it is the *most
Core-mediated* of the three, not the least. Claude and Cursor read their
project file unconditionally; Codex requires Core to also manage a global
trust-list entry per project (in `~/.codex/config.toml`) before its
project file does anything, and that trust grant is coarser than "MCP
only." Any resolver/materializer design needs a trust step for Codex that
Claude/Cursor don't need, and should not claim MCP-only isolation for
Codex without flagging that trusting a project also relaxes its hook/exec
sandboxing — exactly the kind of thing the plan's own diagnostics section
asked to surface honestly rather than fake.

### What this means for the plan

The plan's own escape hatch applies: *"If step 0 reveals that
Claude/Codex/Cursor handle MCP configuration fundamentally differently,
stop after the inventory and propose the smallest common runtime contract
before proceeding."* That's exactly what's true here. Concretely, before
any registry/resolver/UI work:

1. **Project persistence doesn't exist at all yet.** This blocks the plan
   regardless of MCP specifics — `ProjectCapabilities{projectId,
   enabledMcpIds[]}` needs *some* durable, restart-surviving store keyed by
   project id, and today there is no project-scoped store of any kind to
   extend. This is a bigger prerequisite than the plan assumed ("if
   `core.local.yaml` … is appropriate, reuse it" — `core.local.yaml` has no
   project dimension today; it would need one added, or a sibling file
   introduced).
2. **All three providers turn out to support project-scoped MCP via a
   native project-local file** — `.mcp.json` (Claude), `.cursor/mcp.json`
   (Cursor), `.codex/config.toml` (Codex) — see the correction above.
   Codex is the outlier only in needing an *additional* trust-list entry
   in its global `~/.codex/config.toml` before that file is honored, and
   in that trust grant being coarser than MCP alone (also covers hooks and
   exec-policy overrides). This changes the shape of the fix entirely: no
   resolver/adapter-injection layer is needed for any of the three.
3. **The materializer model beats the resolver model.** Rather than
   `CapabilityResolver → provider adapter → runtime injection` (the plan's
   original design), Core's registry can be the source of truth and a
   thin materializer just writes each project's own native file:
   `Core Project Capabilities → Claude: .mcp.json / Cursor: .cursor/mcp.json
   / Codex: .codex/config.toml (+ a trust-list entry)`. Each provider then
   does exactly what it already knows how to do — no adapter-specific
   injection code path in `internal/execution/providers` at all. This also
   makes a project's MCP config portable: open the repo with a bare
   Claude/Cursor/Codex outside Core entirely and the project's MCP set is
   still there.
4. Given (1)-(3), the smallest-common-contract candidate is: *Core writes/
   maintains the three native files per project from one registry, and for
   Codex specifically also manages the corresponding trust-list entry in
   the global `~/.codex/config.toml` (with an explicit, visible "this
   grants hook/exec trust too, not just MCP" warning in the UI before
   doing so)*. That is a real design fork from the plan's original
   resolver-based architecture, but a materially smaller and cleaner one —
   worth confirming before writing any backend code, same as before, just
   for a different reason than originally thought.


*Найцінніше тут навіть не UI з галочками, а вимога спочатку розібратися,
де MCP реально живуть і як кожен CLI їх отримує. Там майже гарантовано
знайдуться архітектурні розбіжності з тим, як ChatGPT це собі уявляв —
і вже після inventory можна буде вирішити persistence/runtime без
фантазій.*
