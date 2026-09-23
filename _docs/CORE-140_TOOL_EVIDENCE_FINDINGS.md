# CORE-140 — Tool evidence req/missing investigation

## Verdict

The plaque text `req go_diagnostics missing go_diagnostics` was primarily a
**detection + presentation bug**, not proof that MCP ran and was ignored.

| Layer | Finding |
|---|---|
| UI chips | Required tools were listed twice: once as `req X`, again as `missing X`. |
| Cursor stdout log | `_registry/sessions/*.log` for Cursor contains prompt + final text only — no `tool_use` / `CallMcpTool` lines — so log scanners never saw evidence. |
| Cursor transcripts | Real tool events live in `~/.cursor/projects/.../agent-transcripts/<chat_id>/`. Core now reads them. |
| MCP availability | Daemon Cursor often has **no MCP servers** attached (empty `GetMcpTools`). That is a real launch/MCP gap (CORE-32), separate from chip duplication. |
| Profile default | Non-Core repos were wrongly defaulted to Go/`go_diagnostics` for any `feature`/`bug`. Narrowed to Core-tree contexts. |

## Evidence sampled

- `core-139-*.log`, `core-137-*.log` (Cursor): no MCP/tool tokens in stdout.
- Codex `core-131-*.log`: structured RPC present, but task was Python career-wizard; Core still required `go_diagnostics` via the old default.
- This session: `GetMcpTools` returned an empty server catalog.

## Artifacts

- UI: single chip per required tool (`formatRequiredToolChip`)
- Detection: `cursor_transcript.go` + finalize/refresh wiring
- Requirements: Core-tree-only Go default
- Docs: `_docs/AGENT_TOOL_EVIDENCE.md`
