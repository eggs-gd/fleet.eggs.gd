# Technology

Fleet keeps its runtime technology narrow.

## Runtime: Go

One long-running Go binary, `fleet`, does everything at run time: the HTTP
server, the embedded dashboard, task validation and indexing, the project
scanner, the MCP endpoint and the agent launcher. Storage is `modernc.org/sqlite`
(pure Go, no cgo) for runtime state, and Markdown files for tasks and project
cards. It builds for macOS, Linux and Windows (`GOOS=windows go build` runs in
CI), and macOS is the supported platform, see [windows](specs/windows.md).

## Dashboard: Svelte and Vite

`view/` is a Svelte 5 app built with Vite. Vite is build-time only. The
dashboard is copied into `server/internal/webui/dist` and embedded in the
binary with `//go:embed`, so a built `fleet` needs no separate files. A binary
built without the dashboard serves a placeholder and `serve` refuses to start;
pass `--backoffice-dir <dir with index.html>` to serve another build.

The dashboard reads live state from `/api/state` and mutates through the JSON
API. It does not use generated static data.

## Site: SvelteKit

`site/` is the public landing page, a static SvelteKit build published to
GitHub Pages by its own workflow. It is independent of the binary.

## Entry points

The `Makefile` is the single entry point for local work:

| Target | Does |
|---|---|
| `make setup` | install dashboard and site dependencies |
| `make build` | build the dashboard and embed it into `bin/fleet` |
| `make serve` | build, then run `bin/fleet serve` |
| `make test`, `make vet`, `make check` | Go tests, vet, and the full gate |
| `make scan`, `make rebuild-index` | project scan, `Work/INDEX.md` rebuild |
| `make site` | build the landing page |
| `make version`, `make fmt`, `make clean` | version, formatting, cleanup |

Machine-local overrides (`DATA_ROOT`, `ADDR`, `LIVE`, `PROJECTS_ROOTS`) go in
`Makefile.local`, which is not committed.

The CLI has four commands: `fleet serve`, `fleet scan`, `fleet rebuild-index`,
`fleet version`.

## Project scan

Repository discovery and workspace cards are part of the Go server.

- `fleet scan` walks the configured scan roots and rewrites `_registry`.
- `fleet serve` watches every saved scan root every 2 s. It rewrites
  `_registry` when repositories appear or disappear, keeps each
  `Work/<id>/PROJECT.md` in sync with the registry, and reloads the board.
- Settings → Rescan runs the same pass immediately.
