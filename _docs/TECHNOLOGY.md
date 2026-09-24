# Technology Decisions

Core intentionally keeps runtime technology narrow.

## Runtime

Use Go for the long-running Core runtime:

- daemon;
- local HTTP server;
- static backoffice serving;
- task validation/indexing;
- future agent launcher.

Go gives Core one small binary and avoids mixing UI tooling with orchestration.

## Maintenance Entry Point

Use the top-level `Makefile` as the cheap declarative entry point for local
maintenance while Core is still bootstrapping.

Current targets:

- `make scan`;
- `make workspaces`;
- `make build`;
- `make test`;
- `make check`;
- `make serve`.

The Makefile does not add runtime behavior. It only documents and wires existing
scripts, Svelte build commands, and Go runtime commands.

## Backoffice

Use Svelte/Vite for the local backoffice UI.

Svelte/Vite is build-time only. The runtime should not depend on the Vite dev
server.

Expected flow:

```bash
make serve
```

The Go runtime serves `view/dist`.

The UI reads live state from the Go server at `/api/state`. The runtime does
not use generated static JSON.

## Project Scan

Repository discovery and workspace cards are part of the Go server.

- `make scan` / `core scan` walks the projects tree and rewrites `_registry`.
- `make workspaces` / `core workspaces` rewrites `Work/<id>/PROJECT.md` from that registry, then rebuilds `Work/INDEX.md`.
- `core serve` watches every saved scan root and rewrites `_registry` when repositories appear or disappear, then reloads the board. It does not rewrite workspace cards.
- Settings → Rescan runs that same pass immediately.

## Entry Point Direction

Avoid adding unrelated launch points.

Future direction:

```text
core scan
core generate-workspaces
core validate
core serve
```

Until Go commands replace the Python scripts, keep the scripts small and
deterministic.
