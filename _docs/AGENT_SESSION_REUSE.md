# Agent session reuse (1-1-1 model)

Status: **implemented for 1 agent / 1 project / 1 session. Not extended to
N concurrent sessions per agent per project — see "Out of scope" below
before touching that.**

## Why

Fleet/LAUNCH_POLICY.md already limits Core to one active agent session per
project/repository at a time. Before this change, every new task still
started a brand-new provider session (a new Claude background process, a new
Cursor chat, a new Codex thread) even when the same agent had just finished a
different task in the same project seconds earlier.

Given the launch policy already serializes work per project, it is more
natural for one agent to keep working through a project's queue inside one
continuing session — same conversation, new task each time — than to start
over from zero context every time.

## What changed

Core now keeps at most one persistent session per `(project, agent)` pair:

- When a task's launch is claimed, Core first checks for a session already
  resumed/superseded for *this exact task* (existing CORE-77 relaunch
  behavior, unchanged).
- If none, it now also checks for the most recent session *this agent ran
  anywhere else in the same project* (`previousSessionForAgentProject`,
  `execution/session_registry.go`). If one exists and the backend's resume
  capability allows it, the new session carries that identity forward and
  resumes instead of starting fresh.
- On task completion (outcome other than `needs_input`/`waiting_input`),
  Claude's background-remote session is now explicitly stopped
  (`claude stop <id>`) rather than left running indefinitely — a stopped
  session is what makes the next resume land on the same session instead of
  forking a copy. It stays fully resumable; a human can still `claude attach
  <id>` it directly at any time.
- HITL `needs_input`/`waiting_input` is a pause: Core does not stop the
  Claude process, does not finalize the task, and does not release the
  1-1-1 slot. The rest of that project's queue waits until a human
  answers or explicitly releases the session.

No new persistence file was needed: this reuses the existing per-claim
session records under `~/.fleet/_registry/sessions/*.json` (`persistRuntimeSession`),
just with a new project+agent-scoped lookup instead of only a same-task one.

## Verified Claude resume protocol (2026-09-12, live)

Before this change, Claude's background-remote resume was explicitly marked
unverified and disabled (`ProviderSessionReuseCapability`,
`CanAttempt: false`) — a deliberate decision, not an oversight, per the
CORE-53 finding that injecting stdin into a *live* Claude session never
reached the remote chat. That finding is about a different mechanism
(writing to an already-running process's stdin) than what this feature does
(a fresh non-interactive CLI invocation carrying a new prompt as argv), so it
was worth re-testing rather than assuming it still applied. It was tested
live against a real Claude account on 2026-09-12; findings:

1. **Use the full session UUID, not the short id.** `claude agents --json`
   returns both `id` (short, e.g. `2a1af707` — what `stop`/`logs`/`attach`
   take) and `sessionId` (full UUID). Passing the short id to `--resume`
   silently starts a copy: *"note: started a copy of that conversation as
   ...To continue a session under its own id, pass its full session id."*
2. **The session must be stopped first.** `--resume` on a session that is
   still running (even idle, with `state: done`/`status: idle` — the process
   just hasn't exited) also forks a copy: *"note: session ... is already
   running in the background, so this started a copy."* `claude stop <id>`
   ends the process without discarding the conversation.
3. **Pass no other flags on resume.** `claude --bg --resume <full-id>
   "<prompt>"` — nothing else. Repeating `--remote-control`/`--name`/
   `--permission-mode` (even with the original values) makes Claude treat the
   saved session as diverging from the request and fork again: *"note:
   background session ... keeps its own saved options, so the flags you
   passed started a copy... Without flags, the same command continues ...
   itself."* Done right, the response is *"note: woke session ... with its
   saved options (--remote-control, --name, --permission-mode)"* — same
   session id, same `claude.ai/code/session_...` remote-control URL, prior
   and new transcript both present.

`executionapi/capabilities.go` and `providers/runtime.go` implement exactly
this: capture `sessionId` while polling (`BackgroundAgentStatus.SessionID`),
stop before the caller ever tries to resume, dispatch with only
`--bg --resume <id> "<prompt>"` when a `SessionID` is present.

A resume that comes back with a *different* background id than expected
(meaning the prior session was not actually stopped, or Claude forked anyway)
is logged as a warning in the session log but not reconciled automatically —
see "Out of scope".

## Out of scope — do not build this without deciding git-flow first

This feature deliberately caps at **one** session per agent per project. It
does not add a "max concurrent sessions" setting, and there is no `worker1`/
`worker2`/... naming anywhere in the code yet, even as a stub value.

The reason is not laziness: with exactly one active agent per project, each
agent can keep whatever internal working style it wants (how it edits files,
whether/how it uses git locally) because nothing else touches that
repository at the same time. The moment more than one agent (or more than one
session of the same agent) can be active in the same repository
concurrently, that stops being true — concurrent agents need an actual
answer for branch or worktree isolation, merge/handoff rules, and who is
allowed to push what, before "how many can run at once" becomes a number you
can just increase. None of that is defined yet.

If a future task wants to raise the per-project/per-agent concurrency limit
above 1:

- it needs its own git-flow / worktree coordination design first (a real
  decision, not a default), covering at minimum: does each concurrent
  session get its own worktree or branch, how conflicts are surfaced to
  the operator, and how the launch-policy scopes in `Fleet/LAUNCH_POLICY.md` change
  from "at most one" to "at most N" per scope;
- only then does naming concurrent sessions (`worker1`, `worker2`, ...) and
  a configurable limit belong in code — do not add the config knob first and
  figure out coordination later.

Also out of scope here: reconciling a resume that silently forked (see
above).

Update 2026-09-22: the same 1-1-1 reuse model now also covers Gemini
(`gemini-headless`, Antigravity CLI `agy`) — `ApplySessionReuseAt` and
`ApplyAgentProjectSessionReuse` carry `gemini_conversation_id` /
`gemini_project_id` forward the same way they do Codex's thread id and
Cursor's chat id. Its `ProviderSessionReuseCapability` is registered as
`unverified` (same honesty level as Codex's `thread/resume` was before it
was exercised), not `verified` like Claude/Cursor — see
`_docs/AGENT_LAUNCHER.md`'s "Gemini Headless Notes" for what has and has
not actually been confirmed live.
