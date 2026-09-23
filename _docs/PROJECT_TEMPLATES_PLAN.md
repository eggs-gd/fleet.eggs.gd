# User-Defined Project Templates (Plan)

> **Status: draft plan, pre-inventory.** Authored externally (ChatGPT) from
> conversational context about Core's intent, without access to the actual
> codebase. Treat every persistence/API/executor claim below as a proposal,
> not a fact about the current implementation — the "Phase 0 questions"
> section below must be answered from real code before any of it is built.
> See "Phase 0 Findings" (appended after this plan) for that reconciliation.
> Companion to `_docs/MCP_PROJECT_CAPABILITIES_PLAN.md` — the two are
> deliberately not coupled in v1.

Я б зберіг це окремим design doc поруч із MCP Scoping. І тут спеціально не
треба прив'язувати implementation до поточної фабрики сильніше, ніж
потрібно.

## Overview

Add support for user-defined project templates that can be compared with
and applied to existing projects.

A template represents the user's own preferred project structure,
conventions, instructions, configuration files, agent setup, CI setup,
documentation structure, or any other reusable project content.

Templates are not a requirement of Core. Core must continue to work with
projects that:

- have no template;
- do not use `AGENTS.md`;
- have no agent-specific configuration;
- follow completely different conventions;
- were created before Core existed.

The feature exists because users may have their own reusable project
conventions and want an easy way to apply or audit them across projects.

Core must not treat any particular template as canonical.

## Core principle

> Templates are user-owned reusable project context, not Core
> architecture.

Conceptually:

```
User template
     │
     ├──── Compare ──── Project
     │
     └──── Apply ────── Project
```

Core does not need to understand the semantic meaning of files inside the
template. For example, one user's template might contain:

```
my-agent-setup/
├── AGENTS.md
├── CLAUDE.md
├── .cursor/
├── .github/
└── docs/
```

Another user's:

```
company-baseline/
├── README.md
├── SECURITY.md
├── .github/
├── docker/
└── scripts/
```

Both are equally valid.

## Non-goal: no Core project standard

Do not introduce concepts such as:

```
Core-compatible project
Core standard
Canonical Core structure
Required AGENTS.md
Required agent files
```

There is no such requirement. The user's current personal convention
around `AGENTS.md`, Claude, Codex, Cursor, Gemini and MCPs is a user
preference, not a Core invariant. Core should remain opinion-free about
repository structure.

## Primary use case

A user has developed their preferred project setup over time. Older
projects may:

- lack it entirely;
- contain an older version;
- have partially compatible files;
- contain valuable project-specific modifications that must not be
  overwritten.

Today the user manually asks an agent: *"Review this project and bring
its agent infrastructure in line with my current template."* This
feature makes that workflow first-class.

## Template registry

Core should maintain a lightweight registry of templates.

Conceptually:

```
ProjectTemplate
  id
  name
  path
  description?
```

Example:

```
Templates

My Agent Setup
~/Projects/_templates/agent-ready

Go Service Baseline
~/Projects/_templates/go-service

Company CI
~/Projects/_templates/company-ci
```

Important: Core does not need to own or copy the template directory. The
preferred initial model is to reference an existing directory controlled
by the user. This allows the user to:

- edit it with any IDE;
- put it under Git;
- sync it independently;
- maintain its history;
- use it without Core;
- share it independently from Core.

## Add Template

Initial UX:

```
Settings
└── Project Templates

Project Templates
─────────────────────────────────

My Agent Setup
~/Projects/_templates/agent-ready

Company CI
~/Templates/company-ci

[ + Add Template ]
```

`+ Add Template` initially only needs:

```
Name
Directory
Optional description
```

Do not build a template editor inside Core in the first version. The
filesystem/IDE remains the editor.

## No predefined templates initially

Preferred initial state:

```
Project Templates

No templates configured.

Add a directory containing files and instructions
you commonly apply to projects.

[ + Add existing directory ]
```

Do not silently install the author's personal template as the Core
default. A sample template may eventually be shipped or linked as an
example, but must not imply: *this is the correct way to structure a
Core project.* If examples are added later, they should be explicitly
labelled as examples.

## Project integration

A project should expose template actions.

Conceptually:

```
Project: Perceptrail

Templates
──────────────────────────

My Agent Setup

[ Compare ] [ Apply ]
```

A project does not need to permanently belong to exactly one template.
Avoid introducing `project.templateId` as a required relationship unless
implementation later proves this useful.

Initially, a template is better understood as an input to an operation:

```
Compare(Project, Template)
Apply(Project, Template)
```

This keeps the feature generic. The same project could legitimately
receive `My Agent Setup`, `Company CI`, `Security Baseline`
independently.

## Compare

`Compare` is a read-only agent operation. Core provides the executor
with:

```
project path
template path
operation instructions
```

Conceptual task:

```
Compare the target project against the selected template.

Template:
<template path>

Target:
<project path>

Inspect both directory trees and relevant file contents.

Identify meaningful differences between the target project
and the conventions represented by the template.

Preserve awareness of project-specific configuration.

Do not modify any files.

Return:
- missing template elements
- legacy/outdated equivalents
- conflicting structures
- project-specific content that must be preserved
- recommended changes
```

The executor performs the semantic comparison. Core itself does not need
a complicated Markdown/config diff engine.

### Compare result

Result should eventually be understandable in the UI.

Example:

```
My Agent Setup

⚠ Differences found

✓ AGENTS.md exists
⚠ CLAUDE.md uses older structure
✕ Cursor instructions missing
✕ Gemini instructions missing
✓ Project-specific instructions detected

[ View report ]
[ Apply ]
```

Do not require structured findings in the first implementation if the
current task/executor architecture naturally produces a textual report.
Start with the simplest representation compatible with existing task
results. Structured findings can be added later.

## Apply

`Apply` is not `cp -R`. This is critical.

Existing projects may contain:

- valuable instructions;
- modified configuration;
- provider-specific files;
- custom documentation;
- newer project-specific content.

Therefore the default Apply operation should be performed by an
executor.

Conceptual task:

```
Apply the selected project template to the target project.

Template:
<template path>

Target:
<project path>

The template represents user preferences and conventions,
not files that must blindly overwrite the target.

Inspect both before changing anything.

Preserve project-specific information.
Merge or adapt existing files where appropriate.
Create missing files where appropriate.
Remove or replace legacy structure only when justified.

Do not blindly overwrite existing project files.

After changes:
- verify the resulting structure
- summarize modifications
- report unresolved conflicts
```

This fits the existing Core execution model:

```
User
  ↓
Apply Template
  ↓
Core creates task
  ↓
Executor receives project + template context
  ↓
Executor modifies project
  ↓
existing review / HITL / completion lifecycle
```

Do not invent a second execution engine for templates.

## Safety

Template application modifies a real project and must use existing
execution safeguards. At minimum:

- never silently overwrite project files from the UI;
- use normal executor/session lifecycle;
- preserve existing HITL semantics;
- expose resulting changes through whatever review mechanism currently
  exists;
- never delete project-specific content simply because it does not exist
  in the template.

If the project is under Git, existing Git-based review/diff mechanisms
should be reused where available. Do not require Git solely for this
feature.

## Template evolution

Templates are expected to change.

Example:

```
September:
My Agent Setup v1

October:
user edits template directory

November:
same registered template now contains newer conventions
```

Core should use the current contents of the referenced directory when
Compare/Apply runs. Do not duplicate the template into Core storage
unless there is a concrete technical reason. Therefore the user's
existing Git history can serve as template version history. Core does
not need template versioning in the first implementation.

## Missing template path

Because Core references external directories, paths may disappear.

Possible state:

```
My Agent Setup
~/Projects/_templates/agent-ready

⚠ Directory unavailable
```

Actions:

```
Change path
Remove from registry
Recheck
```

Do not delete anything from disk when removing a template from Core.

## Relationship to Project Capabilities / MCP Scoping

This feature and MCP scoping are separate features. Do not couple their
initial implementations.

Project Templates operate primarily on project content/context. Project
Capabilities control the actual tools exposed to executors.

Conceptually they may eventually cooperate:

```
Template:
"My Agent Setup"

Suggested capabilities:
- Architecture Patterns
- TypeScript
```

But do not implement this automatically in v1. In particular, a file
inside a template must not silently enable a global MCP capability. Any
future template → capability behavior requires an explicit design.

## Relationship to agents

Templates are provider-independent. Core must not encode special
semantics such as:

```
CLAUDE.md means Claude support
.cursor means Cursor support
AGENTS.md means generic support
```

unless Core already has such semantics elsewhere for independent
reasons. For this feature these are simply files. Their meaning is
interpreted by the executor applying/comparing the template. This lets a
user invent any convention without requiring Core changes.

## Storage

Before implementation, inspect the configuration infrastructure produced
by Settings Phase 3. Prefer storing only registry metadata:

```
projectTemplates:
  - id: agent-setup
    name: My Agent Setup
    path: /Users/.../Projects/_templates/agent-ready
```

Do not store template contents in configuration. Do not introduce
another config file if the existing local overlay is appropriate.
Template paths are machine-local configuration and should not
accidentally enter the public Core repository.

## API

Exact API shape should follow existing Settings/project APIs.
Conceptually required operations are:

```
LIST templates
ADD template
UPDATE template metadata/path
REMOVE template registration
RECHECK template path

COMPARE project with template
APPLY template to project
```

`REMOVE` means: forget this template registration — not: delete
directory from filesystem.

Compare/Apply should preferably enter the existing task/session
infrastructure rather than becoming synchronous HTTP operations.

## UI placement

Do not overbuild navigation before inspecting the current Settings
structure. Likely global location:

```
Settings
└── Project Templates
```

or an appropriate subsection under Projects & Data. Project-level UI
only needs access to registered templates and actions.

Example:

```
Templates
──────────────────────────

My Agent Setup

[ Compare ]   [ Apply ]

Company CI

[ Compare ]   [ Apply ]
```

If there are many templates later, use a selector instead. Do not
optimize for that now.

## Diagnostics

Useful states:

```
ready
missing_directory
unreadable
invalid_path
```

Potential diagnostics:

```
Template "My Agent Setup" directory is missing

Template directory cannot be read

Template path points to a file instead of directory
```

Do not semantically validate template contents. An empty directory is
technically a valid template, even if useless.

## First acceptance scenario

User creates `~/Projects/_templates/my-agent-setup/` containing their
personal current conventions.

In Core:

```
Settings
→ Project Templates
→ Add Template

Name:
My Agent Setup

Directory:
~/Projects/_templates/my-agent-setup
```

Core registers it. User opens an old project:

```
Perceptrail
→ Templates
→ My Agent Setup
→ Compare
```

Core creates a read-only executor task. Executor reports differences.
User chooses `Apply`. Core creates a normal modifying task using the
same project and template. Executor:

- inspects both;
- preserves project-specific information;
- adapts legacy structure;
- adds missing pieces;
- reports changes.

Normal session/review lifecycle continues. No Core-specific repository
standard has been introduced.

## Explicit non-goals

Do not implement in the first increment:

- mandatory project templates;
- automatic application on project discovery;
- Core-defined canonical project structure;
- author's personal conventions hardcoded into Core;
- template marketplace;
- remote template repository;
- template version management;
- template inheritance;
- template composition engine;
- validation DSL;
- `if Go then X` rules;
- automatic MCP enablement;
- automatic agent/provider configuration;
- background drift scanning;
- scheduled compliance checks;
- custom merge engine;
- arbitrary filesystem deletion;
- mobile/remote functionality.

## Phase 0 — questions before implementation

Before coding, inspect the existing architecture and answer:

1. Where should machine-local template registry metadata live given
   Phase 3 config?
2. How does Core currently create a task programmatically for a project?
3. Can the current executor safely receive/read an additional directory
   outside the project root?
4. Do Claude/Codex/Cursor differ in filesystem sandbox/access rules that
   affect reading the template directory?
5. Can Compare reliably be enforced as read-only with current executor
   infrastructure?
6. What existing review/diff lifecycle can Apply reuse?
7. Does task/session persistence currently support recording which
   template triggered the operation?
8. Does the project UI already have an appropriate place for these
   actions?
9. Are template paths valid if Core later runs on a server while
   executor/project lives on another machine?

Question 9 is especially important. Don't prematurely solve distributed
filesystem semantics, but don't design a path-only contract that becomes
impossible to extend once Core and executors live on different hosts.

## Implementation order

```
0. Architecture inventory + answer open questions
1. Template registry model/persistence
2. Settings list + Add/Edit/Remove/Recheck
3. Project → template selection
4. Compare through existing executor/task lifecycle
5. Apply through existing executor/task lifecycle
6. Review/result integration
7. Tests + browser pass
```

---

*І я б тут реально попросив девелопера зупинитися після пункту 0 і
повернутися з питаннями. Бо тут є одна потенційно жирна архітектурна
міна: зараз `project path` і `template path` можуть бути двома
локальними директоріями на одному Mac. А ми буквально щойно обговорювали
майбутню схему, де Core живе на сервері, а executor і repo — на Mac.
Якщо зараз зробити template identity = absolute filesystem path, потім
це може вилізти боком. Тому саму фічу я б уже зберіг у документацію саме
так, а реалізацію — після того, як він розкаже, як вона лягає на ваш
нинішній runtime.*

---

## Phase 0 Findings (Claude, 2026-09-22)

Answering the plan's own nine questions against the actual codebase.

**1. Where should machine-local template registry metadata live?**
`internal/settings/overlay.go` (`core.local.yaml`) is the one machine-local
mutable config file today — `Overlay{ScanRoot, Agents, Manager}`, plus a
hand-rolled codec in `overlay_codec.go` (no YAML library). Adding a
`Templates []TemplateOverlay{ID, Name, Path, Description}` field there is
the natural fit — same pattern just used for the Manager binding in this
same file, gitignored, atomic-write already handled by `SaveOverlay`.

**2. How does Core currently create a task programmatically for a
project?** `POST /api/tasks` → `internal/server/dashboard.go` decodes a
`tasklifecycle.TaskCreateRequest{Title, Request, Project, Repository,
Status, Type, Assignee, Priority, AssignmentReason, DependsOn}` and calls
`DashboardSurface.CreateTask(...)`. That is the one programmatic entry
point — Compare/Apply would construct this same struct (no separate task
API needed, matching the plan's "do not invent a second execution
engine").

**3–4. Can the executor read a directory outside the project root, and do
providers differ?** They differ, concretely:
- **Codex** — `codexThreadStartParams` (`runtime.go:1358`) always sends
  `sandbox: "workspace-write"` scoped to `cwd: session.WorkingDir`, with
  no additional-readable-roots parameter passed anywhere in Core today.
  Codex's own CLI exposes `-c 'sandbox_permissions=["disk-full-read-access"]'`
  as an override, but Core never sets it — an unmodified Codex launch may
  not be able to read a template directory that lives outside the task's
  project root. Unverified without a live test; flagged, not assumed.
- **Claude** — launched via plain `claude --bg --remote-control … --permission-mode
  acceptEdits`, no OS-level sandbox flag at all. `--permission-mode` only
  governs edit-approval prompts, not filesystem visibility — a Claude
  session today can almost certainly already read any path the host
  process can, template directory included.
- **Cursor** — not inspected this pass; likely closer to Claude's
  no-sandbox model given `cursor-agent`'s plain process launch, but this
  needs its own check before relying on it.
This is a real, unresolved per-provider gap, not a detail — Compare
sending "read `<template path>`" to Codex may need an explicit sandbox
permission Core does not grant today.

**5. Can Compare be enforced as read-only?** No such primitive exists.
The task/status domain has no "read-only" or permission-scoped mode —
Codex's `approvalPolicy: "never"` is actually the opposite of restrictive
(auto-approves everything), and Claude's `acceptEdits` auto-accepts file
edits. Today "read-only" would be enforced only by prompt instruction
("do not modify any files"), same as the plan already expects
("Compare... Do not modify any files" as instruction text) — there is no
technical backstop, which matches the plan's own framing but is worth
stating plainly: a misbehaving/compromised executor is not prevented from
writing during a nominal "Compare".

**6. What review/diff lifecycle can Apply reuse?** The existing
`needs_review` task status + HITL session-state contract
(`internal/settings/workflow.go`'s `HITLContract`,
`taskflow.MapExecutionOutcomeToStatus`) is a direct fit — Apply's executor
run finishes like any other task, lands on `needs_review`, and the
existing dashboard review UI applies unchanged. No new lifecycle needed,
confirming the plan's assumption here.

**7. Does task/session persistence support recording which template
triggered an operation?** No structured field — `tasklifecycle.Task` has
no metadata/tags/provenance bag beyond `Source` (provider-origin, not
operation-origin) and free-text `Body`/`AssignmentReason`. Recording
"this task came from Compare(ProjectX, TemplateY)" today would have to go
into the task body text, same convention already used for
AssignmentReason-style provenance elsewhere. Adding a structured field is
a small, additive schema change if this turns out to matter later.

**8. Does the project UI already have a place for these actions?** Yes —
`ProjectSettings.svelte` / `lib/projectSettings.js` (added this session)
is already the per-project settings page (`PROJECT_SETTINGS_VIEW`, reached
via the project header's own "Settings" tab). It is currently a read-only
hierarchy/technology inspector with no write path, but it is the correct
existing location to add a "Templates" section — no new navigation
concept required.

**9. Are template paths (and project paths) valid once Core/executor/repo
can live on different hosts?** No — this concern is real and already
latent everywhere, not specific to templates. Every path in the system
today is an absolute local filesystem path assumed to be on the same
machine as the Core process: `board.Project.Path`/`RelativePath`,
`RuntimeSession.WorkingDir`, `overlay.ScanRoot`, `_registry/repositories.json`'s
`path` field. There is no existing "logical project identity independent
of host" abstraction to hang a remote-safe `TemplateOverlay.Path` on —
adding one now would be solving it for templates alone while the same gap
stays open for projects themselves. Confirms the plan's own instinct: not
worth solving distributed-filesystem semantics for this feature in
isolation, but worth flagging as the same open question the Core/Mac
split conversation already raised, so `TemplateOverlay` should not be
designed as if `path` were guaranteed portable — e.g. keep it a plain
string now, but do not let it leak into anything (registry ids,
comparisons) that assumes two paths on two hosts are comparable.

### Net read

Nothing here blocks the plan's Phase 1 (registry + Settings CRUD) — that
part is genuinely cheap given `core.local.yaml`'s existing pattern.
The load-bearing open question before Compare/Apply specifically is **3/4
(Codex sandbox read access to an external template directory)** — worth a
five-minute live check (spawn a Codex thread with `cwd` inside a task
project and ask it to read a file one directory above) before committing
to "Compare = one executor task" as the mechanism for all three
providers.
