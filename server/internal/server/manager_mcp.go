package server

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/eggs-gd/fleet.eggs.gd/internal/manager"
)

// registerManagerMCPRoute mounts the Manager API as MCP tools directly on
// this already-running core serve instance, at /mcp. Handlers call service
// in-process (no HTTP round-trip to itself) — this is the same service the
// /api/manager/* routes use, not a second instance.
//
// Distributed as a URL, not a spawned subprocess: an operator's .mcp.json
// just needs "type": "http", "url": "http://<addr>/mcp" — no Go toolchain,
// no path to an App source checkout, works identically on any machine
// running a built App binary. See ../../_docs/MANAGER_MCP.md.
func registerManagerMCPRoute(mux *http.ServeMux, service *manager.Service) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "core-manager", Version: "0.0.1"}, nil)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "manager_schema",
		Description: "Return the JSON schema for the Intent object manager_command " +
			"expects. Call once at the start of a session to learn the exact shape " +
			"instead of guessing field names.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ emptyMCPInput) (*mcp.CallToolResult, any, error) {
		schema, err := manager.SchemaObject()
		if err != nil {
			return nil, nil, err
		}
		return nil, schema, nil
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "manager_vocabulary",
		Description: "Return the current valid project ids, workspace ids, " +
			"assignees, and statuses — the same board projection Core itself " +
			"uses to validate project/repository references. Call this instead " +
			"of reading Work/INDEX.md or _registry/*.json by hand.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ emptyMCPInput) (*mcp.CallToolResult, manager.Vocabulary, error) {
		return nil, service.Vocabulary(), nil
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "manager_command",
		Description: "Execute one structured Manager intent: create a task " +
			"(kind=task), change status/assignee/priority, add a comment, " +
			"cancel (archive, requires confirm=true), or look up a task/board " +
			"(kind=board_command or question). This writes the same durable " +
			"Markdown task cards a human editing Work/<project>/tasks/*.md by " +
			"hand would produce — call this instead of writing those files " +
			"directly. See manager_schema for the Intent shape and " +
			"manager_vocabulary for valid project/assignee values.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in manager.CommandRequest) (*mcp.CallToolResult, manager.Response, error) {
		return nil, service.SubmitCommand(ctx, in), nil
	})

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{Stateless: true})
	mux.Handle("/mcp", handler)
}

// emptyMCPInput is the input type for tools that take no arguments —
// AddTool requires an object-shaped In type, so a bare empty struct stands
// in for "nothing to pass".
type emptyMCPInput struct{}
