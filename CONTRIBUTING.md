# Contributing

## Set up

```bash
make setup       # npm ci for the dashboard and the site
make build       # dashboard, then bin/fleet with the dashboard embedded
make check       # go vet, the Go tests, and the dashboard tests
```

You need Go 1.26.6 or newer (the `go` line in `server/go.mod` fetches the toolchain
for you) and Node 22.12 or newer. `make help` lists every target.

`make serve` builds and runs `bin/fleet serve`. It uses the Data root saved in
`~/.fleet/app.json`, which is `~/.fleet/workspace` on a first run. To work against
another folder, or to start agents while you develop, put local settings in
`Makefile.local` (it is not committed):

```make
DATA_ROOT := /absolute/path/to/your/data
LIVE := 1
```

For the dashboard alone, `npm run dev` in `view/` proxies to a running
`bin/fleet serve` and reads the token from `~/.fleet/launch-token`.

## How the code is laid out

- `server/internal/server`: HTTP routes and the composition root.
- `server/internal/manager`: the Manager tools (`manager_*` over MCP).
- `server/internal/taskflow`, `taskprovider`, `tasklifecycle`: the task model and
  its Markdown and Plane storage.
- `server/internal/execution`: launching agents and tracking their sessions.
- `server/internal/webui`: holds the dashboard that `make ui` copies in. A checkout
  that has not built it contains only a placeholder, so `go build` and `go test`
  work without Node.

`AGENTS.md` has the engineering rules (how Go is written here, module boundaries,
what the Manager may and may not do). Read it before a larger change.

## Before you open a pull request

- `make fmt` and `make check` pass.
- A change to behavior has a test. Tests do not touch your real `~/.fleet`, and each
  one uses its own temporary directories.
- Docs that describe what you changed are updated.

App and your data are separate: nothing in `server/`, `view/` or `site/` may refer
to a specific person's folders, names, or projects.
