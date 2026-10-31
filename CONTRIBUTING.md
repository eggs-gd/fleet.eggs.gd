# Contributing

## Set up

```bash
make setup       # npm ci for the dashboard and the site
make build       # dashboard, then bin/fleet with the dashboard embedded
make check       # go vet, the Go tests, and the dashboard tests
```

You need Go 1.27.3 or newer (the `go` line in `server/go.mod` fetches the toolchain
for you) and Node 22.12 or newer (22.17 for the landing page in `site/`). `make help` lists every target.

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

## SonarQube (optional)

A maintainer can run the SonarQube analysis locally. You need the
[SonarScanner CLI](https://docs.sonarsource.com/sonarqube-community-build/analyzing-source-code/scanners/sonarscanner)
(`brew install sonar-scanner`), [golangci-lint](https://golangci-lint.run/) v2
(`brew install golangci-lint`; its report is imported into SonarQube, config in
`.golangci.yml`), a SonarQube server and a user token for it. The
server address and the token stay out of the repository: put them in the
environment, or in a `.env` file at the root (it is git-ignored):

```bash
SONAR_HOST_URL=http://your-server:9000
SONAR_TOKEN=<a SonarQube user token>
```

```bash
make sonar       # tests with coverage, then the analysis; fails when the quality gate is red
```

`scripts/sonar-scope` decides what gets analyzed: on the main branch (`origin/HEAD`)
the main analysis, on any other branch the pull request when `gh` finds an open one,
otherwise that branch. A feature branch never overwrites the main branch's state.
Branch and pull request analysis need the
[community branch plugin](https://github.com/mc1arke/sonarqube-community-branch-plugin)
on the server; without it SonarQube Community only knows one branch. Do not run
`sonar-scanner` directly from a feature branch: without those arguments it is
published as the main branch.

`git config core.hooksPath .githooks` turns on two hooks. **pre-commit** is the local gate:
golangci-lint on the Go modules with staged files, and a secrets scan of the staged files when the
SonarQube CLI (`brew install sonarqube-cli`) is installed; a finding stops the commit, and nothing
goes to the server. **pre-push** runs `make sonar` when code the analysis covers changed: the report
of the pushed commit goes to the server, and a red quality gate stops the push. After every push, once it has finished, run `make sonar-publish`: the push's analysis was made while GitHub did not know the commit yet, so the pull request's check could not land on it; this sends the scan again, on the same reports. Both do nothing
unless the two variables are set; skip one with `--no-verify`. `make sonar SONAR_GATE=false` sends
the report without waiting for the gate. The setup and the reasons for it are in the organization's
[SonarQube standard](https://github.com/eggs-gd/.github/blob/main/docs/sonarqube.md).

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

## License of your contribution

Fleet is licensed under the [Apache License 2.0](LICENSE). By opening a pull
request you agree that your contribution is licensed under the same terms
(section 5 of the license). Please also read the [Code of Conduct](CODE_OF_CONDUCT.md).
