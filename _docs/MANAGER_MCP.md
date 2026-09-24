# Manager MCP

`internal/server/manager_mcp.go` mounts the Manager API as MCP tools
directly on `core serve`, at `/mcp`, so an agent acting as Manager (most
often Codex chat, per the CORE-92 override path — see
`../../Data/_docs/MANAGER.md`) can create/patch tasks through one
structured call instead of hand-parsing `Work/INDEX.md`, `_registry/*.json`,
and individual task Markdown files.

## Architecture

Streamable HTTP MCP transport (`github.com/modelcontextprotocol/go-sdk`),
served from the same `http.ServeMux` as every other route in `Serve()` —
not a separate process. Tool handlers call `*manager.Service` in-process
(the same instance `/api/manager/*` uses), so there is no HTTP round-trip
to itself and no "is core serve reachable" failure class.

This replaced an earlier design (a standalone `cmd/manager-mcp` binary,
spawned via `go run` and talking to `/api/manager/*` over loopback HTTP).
That doesn't survive distribution: App ships as a built binary to an
operator who has no Go toolchain and no path to an App source checkout for
`.mcp.json` to reference. Mounting the MCP server on the App binary's own
already-running HTTP server means `.mcp.json` only ever needs a URL — the
same `--addr` the dashboard already uses — which is portable across any
machine running a built App binary, with nothing to install or spawn.

## Tools

- `manager_schema` — the Intent JSON schema `manager_command` expects.
  Call once per session instead of guessing field names.
- `manager_vocabulary` — current valid project ids, workspace ids,
  assignees, and statuses (the same board projection Core itself uses to
  validate references). Call this instead of reading `Work/INDEX.md` or
  `_registry/*.json` by hand.
- `manager_command` — executes one structured `Intent` (see
  `internal/manager/types.go`): create a task (`kind: task`), change
  status/assignee/priority, add a comment, cancel (`kind: cancel`,
  requires `confirm: true`), or look up a task/board (`kind: board_command`
  or `question`). Wraps `Service.SubmitCommand` — deterministic, no LLM
  classification involved (the calling agent is already the LLM
  constructing the intent).

`manager_text`/`manager_audio` (the fast-path + LLM-classifier pipeline
behind `POST /api/manager/text`) are deliberately not exposed as MCP tools:
the classifier is currently `UnconfiguredClassifier` (a stub), and an agent
calling this MCP server is already an LLM that should construct a
structured Intent directly via `manager_command` rather than round-tripping
through a second, unconfigured classification step.

## Registration

`Data/.mcp.json` (bundled in App's bootstrap template — see
`internal/appconfig/template/.mcp.json` — so every fresh Data root gets it
on first run):

```json
{
  "mcpServers": {
    "manager": {
      "type": "http",
      "url": "http://127.0.0.1:8787/mcp"
    }
  }
}
```

Matches `core serve`'s default `--addr`. Requires `core serve` to already
be running — an operator using Manager MCP tools is, by definition, working
against a live Core install. If `--addr` is overridden, update the URL to
match.
