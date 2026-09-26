# Core — App (fleet.eggs.gd)

The Core tool itself: a Go backend + Svelte dashboard that orchestrate
tasks, projects, and AI worker agents. This repository is the product's
source code, build tooling, and engineering documentation — it ships to an
operator as a built binary; it has no knowledge of any specific operator's
data and makes no assumptions about where that data lives on their disk.

Fleet manages work, not artifacts. A task lives in Fleet, and the work
happens in a **workspace**: a folder that can hold code, documents, research,
or anything an AI worker or a person can work on. Code is the most developed
kind of workspace today, not the only one. A task assigned to a person needs no
artifact at all.

An operator's own Inbox/Work/Fleet/Archive content and generated
`_registry` state — their **Data** — is a separate, independent repository
with no dependency in either direction; nothing here references its
layout or its docs. App reads and writes wherever Data lives through a
`--root <path>` flag, which is optional: without it, App resolves the
Data root from `~/.fleet/app.json`. `~/.fleet` is App's one fixed, sticky
home directory on the operator's machine — it holds `app.json` and, by
default, a `workspace/` subdirectory that becomes the Data root on a
genuinely first run, bootstrapped (copied) from App's own bundled template
(`internal/appconfig`) if it doesn't exist yet. Settings > General > Data
root shows the effective path and source, and can relocate it anywhere
(moves the directory and updates the pointer in `~/.fleet/app.json`; a
restart is required to apply) — `~/.fleet` itself never moves, only the
workspace it points at.

For local development against this repo's own checkout of Data (a
separate clone, not a subtree of this repo), `make serve` passes `--root`
explicitly at a sibling `../Data` directory by default — purely a dev
convenience, overridable, and irrelevant to how a packaged build behaves
on an operator's machine.

## Shape

```text
App
|-- server/    Go backend (task orchestration, project scan, agent launch, HTTP API)
|-- view/      Svelte/Vite dashboard frontend
|-- site/      static SvelteKit landing (GitHub Pages)
|-- _docs/     roadmaps, specs, and architecture notes
|-- AGENTS.md  canonical agent instruction file for this tree
`-- Makefile   build/serve/test entry points
```

## Maintenance Entry Point

Use `make` from this directory:

```bash
make help
make build
make test
make check
make serve
make site
```

`make`, `make build`, and `make check` do not build `site/`. `make site` runs
`npm run check` and `npm run build` in `site/`.

By default `make serve` points `--root` at `../Data` (a sibling directory,
a separate clone of the Data repo — a dev convenience only) and serves the
built dashboard from this repo's own `view/dist`. Override the
Data location for a different local layout:

```bash
make serve DATA_ROOT=/absolute/path/to/Data
make scan PROJECTS_ROOTS="/absolute/path/one /absolute/path/two"
make serve ADDR=127.0.0.1:8787
```

## Principle

App does not own or guess at anyone's project data. Everything it reads —
task Markdown, registry state, per-machine config overlay — comes from the
Data root passed to it at launch. See `AGENTS.md` for engineering
conventions and `_docs/` for architecture.
