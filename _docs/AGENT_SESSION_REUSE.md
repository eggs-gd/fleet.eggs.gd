# Agent session reuse

Fleet allows one active agent session per project and repository at a time
(`Fleet/LAUNCH_POLICY.md`). Within that limit, one agent keeps working through
a project's queue in one continuing provider session: the same conversation, a
new task each time. Fleet keeps at most one persistent session per
`(project, agent)` pair.

## How it works

When a task launch is claimed:

1. If a session was already resumed or superseded for this exact task, use it.
2. Otherwise look up the most recent session this agent ran in the same
   project (`previousSessionForAgentProject`,
   `internal/execution/session_registry.go`). If one exists and the backend's
   resume capability allows it, the new session carries that identity forward
   and resumes instead of starting fresh.
3. Otherwise start a fresh session.

After a worker outcome other than `needs_input`, a Claude background session is
stopped (`claude stop <id>`). A stopped session is what lets the next resume
land on the same session. It stays resumable, and a person can still
`claude attach <id>`. `needs_input` is a pause: the process keeps running, the
task is not finalized, and the slot stays held until a person answers or
releases the session.

Session records are the ones in `runtime.db`. The lookup is scoped to project
and agent instead of only the task.

## Resume support by backend

| Backend | Level | Notes |
|---|---|---|
| `background-remote` (Claude) | verified | Protocol below. |
| `cursor-visible` (Cursor) | verified | `cursor-agent --resume <chat-id>`, chat id created before the first prompt. |
| `codex-app-server` (Codex) | unverified | `thread/resume`; on failure Fleet starts a fresh thread and records `session_resume_fallback`. |
| `gemini-headless` (Gemini) | verified | `agy --conversation <id>`: the same conversation id comes back and the earlier turn is remembered. Carries `gemini_conversation_id` and `gemini_project_id`. |

Claude, Cursor and Gemini have no fallback: a failed resume is a launch failure, since
silently forking a copy defeats the reuse. The level is stated by each adapter's `SessionReuse()` and shown in Settings → Agents.

## Claude resume protocol

`claude --bg --resume` resumes the same session only when all three hold.
Otherwise it silently starts a copy.

1. **Full session UUID.** `claude agents --json` returns a short `id` (for
   `stop`, `logs`, `attach`) and a full `sessionId`. `--resume` needs the
   `sessionId`. Fleet captures it while polling.
2. **Stopped first.** A session that is still running, even idle, is forked.
3. **No other flags.** `claude --bg --resume <full-id> "<prompt>"` and nothing
   else. Repeating `--remote-control`, `--name` or `--permission-mode` makes
   Claude treat the request as diverging and fork. The resumed session keeps
   its saved options, its id and its remote-control URL.

A resume that returns a different background id than expected is logged as a
warning in the session log. Fleet does not reconcile it.

## Limit: one session

Reuse is capped at one session per agent per project on purpose. There is no
concurrency setting and no `worker1`, `worker2` naming. With one active agent
per repository, each agent can use whatever local git habits it likes. More
than one concurrent session needs a decision first about branch or worktree
isolation, hand-off and merge rules, and who may push, and about how the
launch-policy limits change from "at most one" to "at most N". Add the
setting only after that design exists.
