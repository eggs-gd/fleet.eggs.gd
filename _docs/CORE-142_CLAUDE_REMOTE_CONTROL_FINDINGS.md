# CORE-142 — Claude sessions start then go idle / “blocked by timeout”

Date: 2026-08-05

## Symptom

Daemon-launched Claude tasks show as started, then after ~10 minutes sit in
operator attention / look “blocked by timeout”. Nothing appears in Claude
desktop or mobile.

## What Core is doing

Claude launches as:

```text
claude --bg --remote-control --name <ref> --permission-mode acceptEdits <prompt>
```

Core then polls `claude agents --json` and `claude logs <background_id>` for:

1. an app-visible remote-control URL (`Continue here, on your phone, or at
   https://claude.ai/code/session_…`);
2. transcript growth / terminal agent state.

## Evidence

| Session | Started | Background ID | Remote URL | Transcript | Outcome |
|---|---|---|---|---|---|
| CORE-115 | 2026-08-04 01:20 | `06a53116` | yes | 188 KB | succeeded |
| CORE-116 | 2026-08-04 06:27 | `<session-b>` | **no** | 1424 B frozen | idle attention → released |
| CORE-140 | 2026-08-05 22:57 | `<session-a>` | **no** | 1424 B frozen | idle attention → released |

Logs:

- `_registry/sessions/core-115-1785795599685230000.log` — has
  `[core] remote-control url: https://claude.ai/code/session_…`
- `_registry/sessions/core-116-1785814046297652000.log` — background id only,
  then idle attention
- `_registry/sessions/core-140-1785959837248928000.log` — same hollow pattern

Regression window: after CORE-115 (~01:20) and before CORE-116 (~06:27) on
2026-08-04. Launch flags and binary path
(`/Users/operator/.local/bin/claude`) are unchanged.

## Classification

| Category | Verdict |
|---|---|
| Core launcher flags wrong | Ruled out (same command as last good session) |
| Core idle timeout logic | Working as designed — symptom, not root cause |
| Core URL parser too strict | Possible secondary only; frozen 1424-byte transcript means Claude never produced agent output either |
| Claude remote-control registration / auth / Desktop | **Primary** |

Hollow sessions: local background id exists, but Claude never registers an
app-visible remote-control session. Without that URL, nothing shows on
desktop/mobile and Core cannot reset idle activity from real work.

## Core fix shipped in this task

Fail fast when a background session never publishes a remote-control URL
and shows no further transcript/state activity within
`BackgroundRemoteControlReadyTimeout` (default **2m**, capped by the
session idle timeout). Kind: `remote_control_unavailable`. Core stops the
hollow background agent and blocks the task with a provider comment instead
of waiting 10m for idle attention with a misleading “open the remote
session” message.

Sessions that keep producing transcript/state activity without a parseable
URL are left running (possible parser mismatch — inspect `claude logs`).
Idle attention remains for sessions that **did** announce a URL and later
went quiet (true HITL wait).

## Operator recovery checklist

Run on the daemon host:

```bash
/Users/operator/.local/bin/claude --version
/Users/operator/.local/bin/claude agents --json
/Users/operator/.local/bin/claude logs <session-a>
/Users/operator/.local/bin/claude logs <session-b>

# Clean zombies if still listed
/Users/operator/.local/bin/claude stop <session-a>
/Users/operator/.local/bin/claude stop <session-b>

# Smoke
/Users/operator/.local/bin/claude --bg --remote-control --name CORE-142-SMOKE \
  --permission-mode acceptEdits 'Reply OK only.'
# wait ~30s, then claude agents --json + claude logs <id>, then stop
```

Also confirm Claude Desktop is running and logged in on the daemon host —
remote-control app visibility depends on that path.

Interpretation:

- Smoke has no `Continue here…` / session URL → Claude-side (re-auth, restart
  Desktop, CLI update/downgrade).
- Smoke has URL but Core misses it → broaden `ParseRemoteControlURL` with a
  regression test (do not guess without a live `claude logs` excerpt).
- Optional after CLI works:
  `CORE_MANUAL_LIVE_VERIFY=1 go test ./internal/server/... -run TestRuntimeManualVerifyClaudeBackgroundRemoteSession -v -timeout 90s`
