# Gemini

## Best At

- (not yet characterized — this worker was just wired up; fill in once it has
  a track record on real tasks).

## Use For

- (undecided — start with small/low-risk tasks until behavior is observed).

## Avoid As Default For

- (undecided).

## Fleet Launcher

- Gemini launches through Google's Antigravity CLI, binary `agy` (not
  `gemini`). Daemon launches require `agy` on PATH (`exec.LookPath("agy")`).
  Optional override: `agents.gemini.executable` in `core.local.yaml`.
- Backend is `gemini-headless`, visibility class `headless`: `agy --print
  <prompt> --output-format stream-json` is a synchronous call that blocks
  until the turn's terminal `result` event, not a background process like
  Claude or Codex. There is nothing to reattach to after a Fleet restart.
- On the first launch in a working directory, Fleet also passes
  `--new-project` so `agy` registers that directory and auto-loads
  `AGENTS.md`, `GEMINI.md`, and project-scoped `.agents/mcp_config.json`.
  Fleet discovers the project id afterward from
  `~/.gemini/config/projects/*.json`. Later launches in that directory reuse
  `--project <id>`.
- Resume is `--conversation <id>`, a fresh process invocation, not stdin
  into a live process.
- Terminal status mapping (`result.status`): `SUCCESS` → `succeeded`,
  `WAITING` → `waiting_input`, `CANCELED`/`INTERRUPTED` → `cancelled`,
  anything else → `failed`.
- The Manager server is the project file `.agents/mcp_config.json`. `agy mcp
  add` writes a global config and is not how Fleet registers the Manager.

