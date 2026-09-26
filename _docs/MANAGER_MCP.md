# Manager MCP

`internal/server/manager_mcp.go` mounts the Manager API as MCP tools
directly on `core serve`, at `/mcp`. The Manager is a Claude, Codex, Cursor,
or Gemini session whose folder is the data root. It reads and changes the
board through these tools instead of hand-parsing `Work/INDEX.md`,
`_registry/*.json`, and task Markdown files. The judgment (when to ask, how to
split, what counts as ready) lives in the Manager skills, see
[manager-skills](specs/manager-skills.md).

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

Every tool call goes to the same `*manager.Service`. Failures come back as
typed errors with a suggestion (`fail-with-fix`), and a call that worked but
needs the person's attention carries `warnings`. The behavior of each tool is
specified in [manager-skills](specs/manager-skills.md).

- **Schema and vocabulary.** `manager_schema` is the Intent JSON schema
  `manager_command` expects. `manager_vocabulary` lists the valid project
  ids, workspace ids, assignees, and statuses.
- **`manager_command`.** Executes one structured `Intent` (see
  `internal/manager/types.go`): create a task (`kind: task`, with
  `depends_on`, `repositories`, `acceptance_criteria`, `source_inbox`),
  change status/assignee/priority, add a comment, cancel (`kind: cancel`,
  requires `confirm: true`), or look up a task or the board (`board_command`,
  `question`). Wraps `Service.SubmitCommand`: deterministic, no
  LLM classification, because the calling agent is already the LLM.
- **Read.** `manager_board` (views, `project`/`status` filters,
  `detail=summary`, `limit`/`offset` paging), `manager_task` (full
  description, blockers, session, recent activity), `manager_workers`,
  `manager_events` (`task.*` audit rows after a row id; nothing is pushed
  into the session).
- **Decide.** `manager_resolve_project`, `manager_similar`, `manager_route`,
  `manager_validate` (dry run, writes nothing).
- **Write.** `manager_inbox` (capture, promote, list), `manager_project`
  (alias, note, decision on the project card), `manager_answer`,
  `manager_review`, `manager_update` (project, repository, `depends_on`, and
  description of an existing task).

The Manager skills are also served as MCP prompts under the same names
(`intake`, `shape-task`, `resolve-project`, `route`, `triage-attention`,
`review`, `briefing`). A prompt returns the canonical file from the data root.

`manager_text`/`manager_audio` (the fast-path + LLM-classifier pipeline
behind `POST /api/manager/text`) are deliberately not exposed as MCP tools:
the classifier is currently `UnconfiguredClassifier` (a stub), and an agent
calling this MCP server is already an LLM that should construct a
structured Intent directly via `manager_command` rather than round-tripping
through a second, unconfigured classification step.

## Registration

`core serve` writes the Fleet MCP entry into the provider files of the data
root before a Manager session is created or adopted: `.mcp.json` (Claude),
`.codex/config.toml` (Codex), `.cursor/mcp.json` (Cursor), and
`.agents/mcp_config.json` (Gemini). Keys and servers already in those files
are kept, and a file that is not valid JSON is refused, not overwritten. The
bundled template ships the same files with the URL only:

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

The URL Fleet writes carries the per-install launch token in the query
(`?token=...`), which `/mcp` requires. The token is created once in
`~/.fleet/launch-token` and does not change between restarts, so these files
stay tracked and stable. It is valid only for `127.0.0.1` with the server's own
Host, but a query string reaches logs, so do not publish the URL. The address
matches `core serve`'s `--addr`, default `127.0.0.1:8787`. Requires `core
serve` to be running.
