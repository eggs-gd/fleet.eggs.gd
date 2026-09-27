# Agent Instructions — App

This file is the canonical agent instruction file for the **App**
repository (`fleet.eggs.gd` — the Core tool itself: `server`
Go backend, `view` Svelte dashboard, and `site` static landing). Codex reads
this file from the repository root. `CLAUDE.md`, `GEMINI.md`,
`.gemini/settings.json`, and `.cursor/rules/agents.mdc` all point here
instead of duplicating these rules — keep durable rules in this file only.

> **App vs Data.** App and Data are two fully independent repositories
> with no dependency in either direction — App ships as a built binary to
> an operator who has never seen this source tree, let alone a `Data/`
> folder sitting next to it. An operator's own Inbox/Work/Fleet/Archive
> content lives wherever they configured it — resolved at runtime through
> `--root <path>` / `~/.fleet/app.json` (see `internal/appconfig` and
> `README.md`), never a hardcoded relative path. The only place a literal
> `../Data` shows up is `Makefile`'s dev-only default, for contributors who
> keep a separate local clone of the Data repo next to this one to test
> against — that convenience must never leak into application code.

---

# MCP Tools

You have access to MCP servers for Svelte, Go, and design-pattern research.
Use them proactively, not only when the user explicitly asks.

### Svelte MCP

#### 1. list-sections
Use this FIRST to discover all available documentation sections. Returns a
structured list with titles, use_cases, and paths. When asked about Svelte
or SvelteKit topics, ALWAYS use this tool at the start of the chat to find
relevant sections.

#### 2. get-documentation
Retrieves full documentation content for specific sections. Accepts single
or multiple sections. After calling list-sections, you MUST analyze the
returned sections (especially use_cases) and then use get-documentation to
fetch ALL sections relevant to the user's task.

#### 3. svelte-autofixer
Analyzes Svelte code and returns issues and suggestions. You MUST use this
tool whenever writing Svelte code before sending it to the user. Keep
calling it until no issues or suggestions are returned.

#### 4. playground-link
Generates a Svelte Playground link with the provided code. After completing
the code, ask the user if they want a playground link. Only call this tool
after user confirmation and NEVER if code was written to files in their
project.

### Go MCP (gopls)

#### 5. go_diagnostics
Runs compile-time and static analysis checks on Go files. Returns errors
and warnings. You MUST call this tool on every Go file you create or
modify, BEFORE marking the task complete. Fix all reported issues and
re-run until clean.

#### 6. go_vulncheck
Scans dependencies for known vulnerabilities affecting the code paths you
touched. You MUST call this tool after any change to go.mod or go.sum, or
when adding a new import from an external package.

#### 7. go_references / go_symbol_references
Finds all usages of a function, type, or symbol across the workspace. You
MUST use this tool BEFORE modifying or removing any exported symbol, to
confirm you understand every call site. Also use it BEFORE writing new
code: search for existing symbols that solve a similar problem — if
something similar exists, extend it instead of writing a duplicate.

#### 8. go_package_api
Shows the public API surface of a package without reading all its source
files. Use this when working in an unfamiliar package to understand its
existing abstractions before adding new ones.

### Design Patterns MCP

Installed at `$AGENTS_TOOLS_DIR/design_patterns_mcp` (`AGENTS_TOOLS_DIR`
defaults to `$HOME/.agents`). This tool is OPTIONAL — it may not be
available on every machine that runs this agent. If it is NOT in your tool
list, fall back to the neighboring-files check from the "Code Style"
section below — do not treat its absence as a task blocker, and do not
report it as an error to the user. If it IS present, using it before
writing a non-trivial new abstraction is REQUIRED, not optional — see below.

#### 9. find_patterns
Hybrid search (semantic + keyword) over 700+ documented design patterns
given a natural-language problem description. If this tool is present in
your tool list, you MUST use it BEFORE writing any non-trivial new
abstraction (a new interface, a new coordination mechanism, a new
cross-cutting concern) to check whether a well-known pattern already fits.
This tool suggests options; it does not mandate using one. Prefer the
simplest option it returns, and only reach for a heavier pattern (e.g.
Strategy, Observer) when simpler options clearly don't fit — this repo's
own "Go — write it like Go" rules (no interface-just-in-case, no
Manager/Factory/Builder by default, no wrapper structs) still override
anything this tool suggests.

#### 10. get_pattern_details
Retrieves full detail and code examples for a specific pattern by name. Use
after `find_patterns` has narrowed down a candidate, before implementing
it. Same optionality as `find_patterns` above.

#### 11. search_patterns
Keyword/semantic/hybrid search across the catalog by name or concept (e.g.
"circuit breaker", "event sourcing") when you already know roughly what
you're looking for. Use it to look up a pattern the user or a doc mentions
by name, or to compare a few related patterns before picking one. Same
optionality as `find_patterns` above.

`count_patterns` and `get_health_status` are introspection-only — skip them
during normal work.

---

# Task Manager Engineering Conventions

These conventions apply to work inside `server` (Go) and
`view` (Svelte/SvelteKit).

## Project Configuration

- **Language**: TypeScript (frontend), Go (backend)
- **Package Manager**: npm (frontend)
- **Add-ons**: sveltekit-adapter, prettier, mcp

## Code Style — before writing code

1. Read 2-3 neighboring files in the same package first — style must be
   consistent with what already exists, not your own preference.
2. If it seems like a new abstraction is needed (interface, new package,
   new layer) — first check whether the task is solved by simply adding a
   function to an existing file. Default answer is "no, not needed."
3. File size limit: ~200-300 lines. A file growing past that is a signal
   to split by responsibility, not evidence that it's a "normal" file.

## Go — write it like Go, not like Java or JS

Every extra line costs something. Default-forbidden unless justified:

- **Interface "just in case."** Do not create an `interface` if there is
  only one implementation. An interface appears when a second implementer
  exists or a test genuinely needs a mock.
- **Getters/setters.** No `GetTitle()`/`SetTitle()`. Access fields
  directly: `t.Title`.
- **Empty wrapper structs.** Don't wrap a `string`/`int`/`[]Task` in a
  struct with one field unless it has real behavior in methods.
- **Manager/Factory/Builder by default.** If construction is just
  `&Task{...}`, don't write `NewTaskFactory().Build()`. A constructor
  `NewTask(...)` is fine when there's real init logic (validation,
  defaults); if it just copies args into fields, skip it.
- **Deep nesting / exception-style error hierarchies.** Go has no
  exceptions — don't imitate them with a pyramid of custom error types.
  Wrap with `fmt.Errorf("read task %s: %w", id, err)` and move on.
- **"utils"/"common"/"helpers" dumping-ground package.** A function
  belongs to the package that owns its domain.

Required:

- **Early return**, max 1 nesting level in the happy path.
- **Small packages with a clear single responsibility.**
- **Short names in short scopes**: `err`, `ctx`, `i`, `t`. Long descriptive
  names only at package level or when context isn't obvious within 3+ lines.
- **Standard library first** before adding a dependency.
- **Table-driven tests** for logic with multiple cases.

### Example — what NOT to write vs what TO write

Bad (Java/JS mindset):
```go
type ITaskRepository interface {
    GetTaskById(id string) (*Task, error)
}

type TaskRepositoryImpl struct {
    basePath string
}

func NewTaskRepositoryImpl(basePath string) *TaskRepositoryImpl {
    return &TaskRepositoryImpl{basePath: basePath}
}

func (r *TaskRepositoryImpl) GetTaskById(id string) (*Task, error) {
    filePath := r.buildFilePath(id)
    if !r.fileExists(filePath) {
        return nil, NewTaskNotFoundException(id)
    }
    content, readErr := r.readFileContent(filePath)
    if readErr != nil {
        return nil, NewTaskReadException(id, readErr)
    }
    return r.parseTaskContent(content)
}
```

Good (Go mindset) — always write it this way:

```go
func GetTask(dir, id string) (*Task, error) {
    data, err := os.ReadFile(filepath.Join(dir, id+".md"))
    if err != nil {
        return nil, fmt.Errorf("get task %s: %w", id, err)
    }
    return parseTask(data)
}
```

## Repository architecture

The repository contains several independent application areas. Do not model them as one end-to-end pipeline.

### Main modules

**Task domain**
Owns the canonical task model:

- task fields and statuses;
- task updates;
- comments;
- filtering;
- validation of task state transitions;
- normalized task events.

This module must not know:

- how tasks are physically stored;
- how agents are launched;
- how HTTP endpoints are implemented;
- how the dashboard renders data.

**Task storage**
Provides synchronous task persistence operations:

```text
Create
Get
List
Update
AddComment
```

Storage implementations may use:

- Markdown files;
- a remote task-management API;
- another future backend.

Provider-specific concerns stay inside the storage implementation:

- Markdown parsing and serialization;
- filesystem paths and indexes;
- remote API identifiers;
- pagination;
- authentication;
- provider-specific status mapping.

The rest of the application works only with the common task model and storage interface.
Do not leak Markdown documents, filesystem paths, or remote API payloads into task-domain or execution code.

**Task change listener**
Observes changes produced by the configured task source and emits normalized task events.
Possible sources include:

- filesystem watching for Markdown storage;
- polling or webhooks for a remote task manager.

Listeners may run concurrently, but downstream code receives one normalized event stream:

```text
storage-specific change
→ reload canonical task
→ normalize TaskEvent
→ publish event
```

The listener is only an event source. It must not launch agents or contain execution logic.

**Manager**
Handles operator intent and works synchronously through the task-storage API.
Responsibilities:

- interpret normalized manager commands;
- create and update tasks;
- query board/task state;
- add comments;
- return responses to the operator.

The Manager must not:

- subscribe to task events;
- launch agents;
- manage process lifecycles;
- access Markdown or remote APIs directly;
- become the owner of execution state.

The Manager is a client of the task API, not the central runtime.

**Execution runtime**
Consumes normalized task events and decides whether work should start.
Responsibilities:

- inspect task changes;
- check whether a task became executable;
- enforce project and agent concurrency limits;
- acquire and release execution locks;
- resolve repository and agent configuration;
- start and monitor agent processes;
- publish a normalized execution result.

Execution code must not know whether the task came from Markdown, Plane, or another provider.
Its input is a typed task event, never a filesystem path or provider payload.

**Finalizer**
Consumes normalized execution results and maps them to task updates.
Current responsibilities are limited to runtime completion semantics:

- completed;
- failed;
- blocked;
- needs input;
- cancelled;
- timed out;
- orphaned or not shut down cleanly.

The launcher/runtime must determine the final process/session outcome before passing it to the Finalizer.
The Finalizer then updates the task through the task API, for example:

```text
completed      → needs_review
failed         → blocked
needs_input    → pause: task stays doing, session waiting_input, 1-1-1 slot held
timed_out      → blocked
orphaned       → blocked
```

Quality gates, architecture review, lint-review agents, and additional board stages are separate concerns and must not be added to the Finalizer unless the task explicitly requires them.

**HTTP API and dashboard**
The dashboard is a stateless client.
It communicates through narrow HTTP APIs:

- task API for tasks, statuses, and comments;
- runtime API for sessions, logs, and controls;
- manager API for text/audio/structured commands.

The dashboard must not contain task or execution business logic.
Polling is acceptable. Do not introduce WebSockets or internal event-bus coupling unless explicitly required.

### Application startup

The server entry point is the composition root.
It must construct and start independent services rather than one monolithic runtime containing the entire flow.
Conceptually, startup should expose separate siblings such as:

```text
Task storage / task engine
Manager service
Task change listener
Execution runtime
Finalizer
HTTP server
```

The exact package and type names may evolve during refactoring.
The architectural requirement is that these services have independent responsibilities and lifecycles.
Do not create runtime concepts such as `Contour`, `Topology`, or named pipeline sections merely to describe the architecture. The separation must be visible in actual ownership and startup code.

### Control flows

There are two independent flows.

**Operator flow**

```text
Human / voice / chat / dashboard
→ Manager or task HTTP API
→ task storage
→ response to operator
```

This flow ends after the task operation succeeds.

**Execution flow**

```text
task source listener
→ normalized TaskEvent
→ execution runtime
→ agent process
→ ExecutionResult
→ Finalizer
→ task storage
```

Do not join these flows into one continuous Chain of Responsibility.

### Dependency direction

Allowed dependencies:

```text
HTTP / Manager
    → task application API
        → task domain
        → task storage interface

storage implementations
    → task domain

listeners
    → storage interface
    → task events

execution runtime
    → task domain/events
    → repository and agent configuration

finalizer
    → execution result
    → task application API
```

Forbidden dependencies:

```text
task domain           → HTTP
task domain           → Markdown parser
task domain           → remote provider payloads
manager               → launcher/runtime internals
execution runtime     → Markdown files
execution runtime     → Plane API
storage implementation→ agent launcher
dashboard             → provider implementation
agent                 → direct task-storage mutation
```

### Markdown ownership

Markdown parsing and serialization belong only to the Markdown storage implementation.
Filesystem watching belongs to the Markdown change listener.
No generic execution step may parse Markdown or receive filesystem paths as its application input.

### General architectural rules

- Prefer extending an existing module over creating a parallel flow.
- Do not wrap the old architecture in new interfaces without changing the actual control flow.
- Do not encode architecture as metadata.
- Keep provider-specific code at the system boundary.
- Use typed domain objects after crossing the boundary.
- Agents return execution results; they do not directly control task lifecycle.
- New code should replace or delete obsolete paths, not merely add another layer beside them.

The exact package and file names are intentionally not specified because the repository is being refactored. Preserve these responsibility boundaries, but map them onto the simplest structure supported by the current code.

## Site

`site/` is the public landing inside this repository, not a second repo.
MCP config, this file, and GitHub Actions stay at the App root. Pages
deploys from `App/.github/workflows/pages.yml` and publishes `site/build`.

The landing is static SvelteKit (`@sveltejs/adapter-static`). It does not
grow a backend, auth, CMS, or the dashboard. Dashboard identity is the
style source: thin rules, muted surfaces, small radii. Read
`site/README.md`, `site/DESIGN.md`, and `site/IMPLEMENTATION.md` before
changing the landing direction.

From `site/`, before finishing landing work:

```sh
npm run check
npm run build
```

Keep route modules TypeScript. Do not start a long-running dev server
unless asked. If a browser preview is used, inspect desktop and mobile
widths.

## Svelte frontend

The dashboard in `view/` is small — don't bring in React-world enterprise patterns:

- No Redux / heavy state management. Built-in Svelte stores are enough.
- One component = one file; don't split into presentational/container if the component is already short.
- Backend calls go through a thin `api.ts` wrapper over `fetch`, no generated client SDKs.
- Don't create a component until it's used at least twice OR the parent file exceeds ~150 lines.

## Before handing off a result

- [ ] Does a similar function/pattern already exist? (check neighboring files, use go_references/go_package_api before writing new code)
- [ ] Any interface with a single implementation? → remove
- [ ] Any getter/setter without logic? → remove, access field directly
- [ ] Any if/else nesting deeper than 1 level in the happy path? → early return
- [ ] go_diagnostics and go_vulncheck run clean on touched files?
- [ ] svelte-autofixer run clean on touched Svelte files?
- [ ] Any new file duplicating existing logic? → merge instead
