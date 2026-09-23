# Agent Instructions — Data

This is a freshly bootstrapped Data root, created automatically by Core on
first run from its built-in template. Edit this file to describe your own
operating rules — this starting point only documents the directory shape
Core expects.

## Directory Contracts

| Path | Contract |
|---|---|
| `Inbox/` | Raw input only. Anything here may be unstructured and untrusted. |
| `Work/` | Project/workspace folders: metadata, tasks, notes, decisions. |
| `Fleet/` | Worker definitions — humans, AI agents, and execution surfaces. |
| `Archive/` | Closed or intentionally ignored material. |
| `_registry/` | Generated project-discovery state (`core scan` / `make scan`). |
| `_docs/` | Your own decisions and operating notes. |

## Ground Rules

1. Determinism first — read state from the filesystem/git instead of guessing.
2. `_registry/` is generated; do not hand-edit it.
3. Keep personal data local — do not commit secrets or machine-specific files.
