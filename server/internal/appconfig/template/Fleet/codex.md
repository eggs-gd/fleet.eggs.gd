# Codex

## Best At

- Repository implementation.
- Deterministic filesystem edits.
- Refactors with tests/verification.
- Bootstrap scripts and generated _registry/workspace files.
- Debugging local code with terminal access.

## Use For

- `feature`
- `bug`
- `maintenance`
- code-focused `review`

## Avoid As Default For

- broad product strategy without implementation;
- long-form prose polish unless tied to repo changes.

## Core Launcher

- Daemon Codex launches require the official standalone `codex` CLI on the
  daemon PATH (`exec.LookPath("codex")`), typically installed so it resolves
  from a shell-visible location such as `~/.local/bin/codex`. Optional
  override: `CORE_CODEX_BINARY` to an absolute path. Core does not fall
  back to the Codex binary bundled inside the ChatGPT app; a missing
  standalone CLI blocks the task with a setup diagnostic.
- Launch/session diagnostics include the resolved binary path.
- Codex App Server (`codex app-server --stdio`) is a Core JSON-RPC control plane.
  Visibility class is `core_visible`: Core can start/resume threads, stream
  events, expose dashboard continue/interrupt/cancel, and normal daemon
  auto-launch is live-ready. Sessions appear in Codex Remote on paired clients
  with Core-set thread titles; they are not Claude-style phone remote-control
  URLs and may not appear in ordinary ChatGPT desktop sidebar history.
- `launch.mode: allow_core_visible` is a legacy no-op. Normal Codex tasks do not
  need it.
- Core starts a persisted Codex thread with `approvalPolicy: "never"` and
  `sandbox: "workspace-write"`, names it with `thread/name/set` as
  `<task ref> · <task title>`, then starts a turn with the task prompt. The
  ready event is `agent_core_control_ready` (not `agent_remote_control_ready`).
- Core stores the Codex `thread.id`, `turn.id`, human thread title, and host
  name/id on the runtime session, streams app-server events to
  `_registry/sessions/<claim_id>.log`, and shows task status, separate
  `execution_status`, visibility class, last event/message, ids, and log path
  in the dashboard runtime strip.
- Prefer Core dashboard controls against the runtime session API for daemon
  sessions; Codex Remote is the paired-client conversation surface.
- Daemon-launched Codex workers report completion as a compact JSON result
  payload. Core Finalizer applies the task transition through TaskService
  after the Codex turn and app-server session are terminal; workers must not
  move their own active daemon task to `needs_review`.
- `codex exec` remains a host-local fallback/smoke tool only and is not used
  on the normal daemon path.
- On relaunch (task reopen, `needs_rework`, retried `todo`), Core looks up the
  most recent prior session for the same task ref and, if it has a known
  `codex_thread_id`, attempts `thread/resume` before falling back to a fresh
  `thread/start` on any RPC error. See "Session Reuse On Relaunch" in
  `Fleet/LAUNCH_POLICY.md`.

