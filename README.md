# Core — App

The Core tool itself: a Go backend + Svelte dashboard that orchestrate
tasks, projects, and AI worker agents. This tree is the product's source
code, build tooling, and engineering documentation — not anyone's actual
data.

Your own Inbox/Work/Fleet/Archive content and generated `_registry` state
live in a separate **Data** tree — see `../Data/README.md`. App reads and
writes Data through a `--root <path>` flag, which is optional: without it,
App resolves the Data root from `~/.fleet/app.json`. `~/.fleet` is App's
one fixed, sticky home directory — it holds `app.json` and, by default, a
`workspace/` subdirectory that is the Data root on a genuinely first run,
bootstrapped (copied) from App's own bundled template (`internal/appconfig`)
if it doesn't exist yet. Settings > General > Data root shows the effective
path and source, and can relocate it anywhere (moves the directory and
updates the pointer in `~/.fleet/app.json`; a restart is required to
apply) — `~/.fleet` itself never moves, only the workspace it points at.
This repo's own `make serve` always passes `--root` explicitly at
`../Data`, so none of this affects local development.

## Shape

```text
App
|-- _backoffice/
|   |-- scripts/   deterministic Python utilities (registry scan, workspace generation)
|   |-- server/    Go backend (task orchestration, agent launch/session runtime, HTTP API)
|   `-- view/      Svelte/Vite dashboard frontend
|-- _docs/         architecture notes, plans, decisions (engineering-facing)
|-- AGENTS.md       canonical agent instruction file for this tree
`-- Makefile        build/serve/test entry points
```

## Maintenance Entry Point

Use `make` from this directory:

```bash
make help
make build
make test
make check
make serve
```

By default `make serve` points `--root` at `../Data` (a sibling directory)
and serves the built dashboard from this tree's own
`_backoffice/view/dist`. Override the Data location for a different layout:

```bash
make serve DATA_ROOT=/absolute/path/to/Data
make scan PROJECTS_ROOT=~/Projects
make serve ADDR=127.0.0.1:8787
```

## Principle

App does not own or guess at anyone's project data. Everything it reads —
task Markdown, registry state, per-machine config overlay — comes from the
Data root passed to it at launch. See `AGENTS.md` for engineering
conventions and `_docs/` for architecture.
