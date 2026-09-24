# Core Manager API

This document defines the Core Manager API: the dedicated voice/text ingestion
and board-command path that replaces **Codex-as-manager** for non-coding work.

Worker agents (Codex, Claude, Cursor) execute tasks. They must not act as the
default manager for voice/task ingestion.

Canonical mutation rules still live in `_docs/DOMAIN_MODEL.md` and the shared
task write path in `internal/corechain`. This API is an ingestion and
classification front door, not a second task store.

## Role Split

| Role | Responsibility | Must not |
|---|---|---|
| **Core Manager API** | Transcribe, classify, normalize, resolve project/repo, create/update tasks via the canonical mutation layer, answer status lookups | Write application code, launch workers as a side effect of chat, invent repository paths |
| **Worker agents** | Implement coding/research tasks assigned on the board | Own voice triage, create Core board policy, spend coding credits on management |
| **Alex** | Review, prioritize (`backlog` → `todo`), close `needs_review` → `done`/`needs_rework` | — |

Codex chat may still be used for **coding** or explicit Manager override during
migration, but it is no longer the default voice/task ingestion path.

## Target Flow

```text
Action Button / HTTP client
        ↓
  Record audio or submit text
        ↓
   Core Manager API
        ↓
 Speech-to-Text (gpt-4o-transcribe)
        ↓
 Deterministic fast-path parser
        ↓ (miss)
 Manager LLM (cheap structured-output model)
        ↓
 Deterministic validation
        ↓
 Task provider (Markdown now / Plane later)
        ↓
 Core mutation layer (CreateTask / PatchTask)
        ↓
 Daemon launches Codex / Claude / Cursor workers
```

## API Surface

Base URL: the Core backoffice server (default `http://127.0.0.1:8787`).

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/manager/text` | Submit plain text for classify → act |
| `POST` | `/api/manager/audio` | Submit audio (`multipart/form-data`, field `audio`) for STT → classify → act |
| `POST` | `/api/manager/command` | Submit an already-normalized deterministic command |
| `GET` | `/api/manager/vocabulary` | Transcription vocabulary / prompt hints from registry + workspaces |
| `GET` | `/api/manager/schema` | JSON Schema for Manager LLM structured output |

Existing board APIs remain canonical for dashboard editors:

- `POST /api/tasks` — create
- `PATCH /api/tasks` — patch status/assignee/priority/comment/body
- `GET /api/state` — board projection

The Manager API **must** call the same mutation layer those endpoints use. It
must not write Markdown files ad hoc.

### `POST /api/manager/text`

Request:

```json
{
  "text": "move CORE-57 to todo",
  "source": "text"
}
```

Response (success example):

```json
{
  "ok": true,
  "path": "deterministic",
  "intent": {
    "kind": "status_change",
    "ref": "CORE-57",
    "status": "todo"
  },
  "result": {
    "action": "patch_task",
    "ref": "CORE-57",
    "path": "Work/core-eggs-gd/tasks/...."
  }
}
```

### `POST /api/manager/audio`

`multipart/form-data`:

- `audio` — recorded blob (webm/wav/m4a/mp3)
- optional `source` — defaults to `voice`

Pipeline: STT → same classify/act path as text.

### `POST /api/manager/command`

Request uses the structured intent shape (see Schema). Used by clients that
already know the command (UI buttons, scripts).

## Manager Intent Schema

Structured Output + JSON Schema. Discriminator field: `kind`.

| `kind` | Meaning | Typical fields |
|---|---|---|
| `task` | Create a new Work item | `project`, `repository`, `title`, `description`, `priority`, `status`, `type`, `assignee` |
| `status_change` | Move an existing task | `ref`, `status` |
| `comment` | Append a review/activity comment | `ref`, `comment`, `comment_author` |
| `board_command` | Board lookup / filter | `board_action` (`show_board`, `show_task`), optional `project`, `ref` |
| `question` | Status/lookup question answered from Core state | `ref` and/or free `query` |
| `assignee_change` | Change assignee | `ref`, `assignee` |
| `priority_change` | Change priority | `ref`, `priority` |
| `cancel` | Soft-cancel → `archived` (requires `confirm: true`) | `ref`, `confirm` |

Expected create-task shape:

```json
{
  "kind": "task",
  "project": "core-eggs-gd",
  "repository": "core.eggs.gd",
  "title": "Replace Codex manager with Core Manager API",
  "description": "...",
  "priority": 2,
  "status": "backlog",
  "type": "architecture",
  "assignee": "unassigned"
}
```

Full schema is served from `GET /api/manager/schema` and embedded in
`internal/manager/schema.go`.

## Deterministic Fast Path

After transcription (or for plain text), Core tries a deterministic parser
**before** spending Manager LLM tokens.

Supported unambiguous patterns (English; numeric refs also match Ukrainian
“задача N” / “таска N” style mentions):

| Example input | Intent |
|---|---|
| `show board` | `board_command` / `show_board` |
| `show CORE-57` | `board_command` / `show_task` |
| `move CORE-57 to todo` | `status_change` |
| `add comment to CORE-57: …` | `comment` |
| `assign CORE-57 to claude` | `assignee_change` |
| `set priority of CORE-57 to 2` / `P2` | `priority_change` |
| `cancel CORE-57` without confirm | failure `unsafe_command` |
| `cancel CORE-57 confirm` | `cancel` with `confirm: true` |

If parsing is ambiguous or incomplete, the request falls through to the Manager
LLM classifier. If the LLM is not configured, Core returns
`llm_not_configured` rather than guessing.

## Task Provider Boundary

```text
Manager Service
    → TaskProvider (interface)
        → MarkdownTaskProvider  (MVP: Work/**/tasks/*.md via corechain)
        → PlaneTaskProvider     (future)
```

The voice/text workflow never imports Markdown layout details. Switching storage
backends means swapping the provider implementation; STT, fast-path, and schema
stay unchanged.

`MarkdownTaskProvider` delegates to:

- `Runtime.CreateTask(TaskCreateRequest)`
- `Runtime.PatchTask(TaskPatch)`
- `Runtime.State()` for lookups

## Transcription Vocabulary

`GET /api/manager/vocabulary` builds a **dynamic** STT / manager-context payload
from live Core state (not a hardcoded product catalog):

- **Projects** — id, title, short summary (from `PROJECT.md`), and aliases
  (ids, titles, repository paths/basenames). Summaries exist so the manager /
  STT prompt can recognize synonyms and informal names, not only exact titles.
- **Workspaces** — same shape, with workspace summaries.
- **Repositories** — registry names and relative paths as aliases.
- **Technologies in context** — effective/detected tags (and profile languages /
  frameworks / runtimes / tooling) collected from the workspaces, projects, and
  repositories currently in state, with STT-friendly spellings (e.g. `go` →
  `Go`, `typescript` → `TypeScript`).
- **Operational terms only as fixed seeds** — agents/statuses and a few Core
  product words (`Codex`, `Claude`, `Cursor`, `MCP`, `Plane`, board statuses).
  Project and stack names come from state.

The `prompt` field is a readable multi-line hint passed into STT (and available
to the manager LLM) that lists projects with their short explanations, then
technologies in context. Pass this into the STT call so recognition prefers
Core domain names over Apple Dictation defaults. Do **not** use Apple Dictation
for Core task ingestion.

## Models

| Step | Suggested model | Notes |
|---|---|---|
| STT | `gpt-4o-transcribe` | With vocabulary prompt |
| Manager classify/normalize | cheap structured-output model (suggested: `gpt-5.6-luna`) | JSON Schema response format only |
| Workers | Codex / Claude / Cursor | Execution only |

Manager must never write code.

## Failure Modes

| Code | When | Client behavior |
|---|---|---|
| `ambiguous_project` | Multiple projects match | Ask Alex to disambiguate |
| `ambiguous_repository` | Multiple repos match | Ask Alex to disambiguate |
| `ambiguous_command` | Fast-path partial match / unclear intent | Fall through to LLM or ask |
| `unsafe_command` | Destructive action without confirm | Require explicit confirm |
| `malformed_model_output` | LLM JSON fails schema/validation | Retry once or ask |
| `provider_error` | Task provider / mutation / STT / LLM transport error | Surface message; do not invent state |
| `not_found` | Unknown `CORE-*` ref | Ask / offer search |
| `llm_not_configured` | Need LLM but no API key/model wired | Keep text; do not guess |
| `stt_not_configured` | Audio submitted but STT unavailable | Ask for text fallback |

## Dashboard Affordance

The backoffice exposes a **Manager** action next to New Task. It opens an
endpoint-driven panel that can:

- submit text to `/api/manager/text`;
- record audio and POST to `/api/manager/audio`;
- show the structured result (created/patched task, lookup, or failure code).

This is the local control-surface hook for the Action Button → Record audio
flow. Phone/shortcut clients can call the same HTTP endpoints.

## Migration Notes

1. Prefer Manager API for new voice/text board work.
2. Keep `_docs/MANAGER.md` in the Data root as the durable **rules** for Markdown shape,
   refs, and Definition Of Done — the Manager API implements those rules in
   code.
3. Update `_docs/OPERATING_MODEL.md` in the Data root: Core Manager API replaces Codex
   Manager as the default capture path.
4. Fleet workers remain execution-only per `Fleet/ROUTING.md` and
   `Fleet/LAUNCH_POLICY.md`.

## Implementation Map

| Area | Location |
|---|---|
| Design (this doc) | `_docs/CORE_MANAGER_API.md` |
| Package | `server/internal/manager/` |
| HTTP routes | `server/internal/server/server.go` |
| Dashboard panel | `view/src/ManagerPanel.svelte` |
| Canonical writes | `internal/corechain` `CreateTask` / `PatchTask` |
