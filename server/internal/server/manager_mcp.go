package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/eggs-gd/fleet.eggs.gd/internal/manager"
	"github.com/eggs-gd/fleet.eggs.gd/internal/managerskills"
)

// registerManagerMCPRoute mounts the Manager API as MCP tools directly on
// this already-running fleet serve instance, at /mcp. Handlers call service
// in-process (no HTTP round-trip to itself) — this is the same service the
// /api/manager/* routes use, not a second instance.
//
// Distributed as a URL, not a spawned subprocess: an operator's .mcp.json
// just needs "type": "http", "url": "http://<addr>/mcp" — no Go toolchain,
// no path to an App source checkout, works identically on any machine
// running a built App binary. See ../../_docs/MANAGER.md.
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
			"assignees, and statuses — the same board projection Fleet itself " +
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

	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_board", Description: "Read the live board. view is needs_attention, blocked, in_review, or all. Filter by project or status. detail=summary adds the start of each description. Results are paged: pass next_offset back as offset."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in boardMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.ShowBoard(ctx, in), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_task", Description: "Read one task: card, full description, dependencies, blockers, session, and recent activity."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in refMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Task(ctx, in.Ref), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_workers", Description: "List available agents and the task each busy agent holds."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Workers(ctx), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_events", Description: "List task events recorded after the given audit row id. This does not push events into the chat."},
		func(_ context.Context, _ *mcp.CallToolRequest, in eventsMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Events(in.Since), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_resolve_project", Description: "Rank project candidates for a phrase. Verdict is confident, ambiguous, or none."},
		func(_ context.Context, _ *mcp.CallToolRequest, in phraseMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.ResolveProject(in.Phrase), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_similar", Description: "Find duplicate or related tasks before creating one."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in textMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Similar(ctx, in.Text), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_route", Description: "Recommend a worker from an explicit instruction, category, then a free agent."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in draftMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Route(ctx, in.Draft), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_validate", Description: "Dry-run a draft. Returns fixes and writes nothing."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in draftMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Validate(ctx, in.Draft), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_inbox", Description: "capture, promote, list, or show Inbox notes. list gives status/preview per item; show gives one item's full text and every task it produced (promoted_to). Promote creates one task and links it; call it again on the same ref for a second task from the same input."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in inboxMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Inbox(ctx, in.Action, in.Text, in.Ref, in.Project, in.Title), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_alias", Description: "Record another name for a project so it is recognized next time. Repeating an alias does not duplicate it. What a project is belongs in its repository, not here."},
		func(_ context.Context, _ *mcp.CallToolRequest, in aliasMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Alias(in.Project, in.Alias), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_project_facts", Description: "Read-only. The project's current one-line summary, whether it is still just a guess (summary_source), and its repository's own README (or, for a group, each member's), to write a real description from. Writes nothing."},
		func(_ context.Context, _ *mcp.CallToolRequest, in projectMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.ProjectFacts(in.Project), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_describe", Description: "Record the project's one-paragraph description and mark it confirmed, so the scanner stops overwriting it with a guess. Call this only after the person has approved the wording you read from manager_project_facts."},
		func(_ context.Context, _ *mcp.CallToolRequest, in describeMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Describe(in.Project, in.Summary), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_answer", Description: "Record the person's answer, move a waiting task forward, and let the worker continue."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in answerMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Answer(ctx, in.Ref, in.Text, in.Status), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_review", Description: "accept sets done and releases ownership. rework sets needs_rework and stores the comment."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in reviewMCPInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Review(ctx, in.Ref, in.Verdict, in.Comment), nil
		})
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "manager_update", Description: "Change project, repository, depends_on, or the description (body) of an existing task. Only the fields you set change. Each is checked against the board first."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in manager.UpdateInput) (*mcp.CallToolResult, manager.Response, error) {
			return nil, service.Update(ctx, in), nil
		})

	for _, name := range managerskills.Names {
		desc, err := managerskills.Description(name)
		if err != nil {
			panic(fmt.Sprintf("manager skill %s is not embedded: %v", name, err))
		}
		skillName := name
		mcpServer.AddPrompt(&mcp.Prompt{Name: skillName, Description: desc}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			body, err := managerskills.Read(service.DataRoot, skillName)
			if err != nil {
				return nil, err
			}
			return &mcp.GetPromptResult{
				Description: desc,
				Messages: []*mcp.PromptMessage{{
					Role:    "user",
					Content: &mcp.TextContent{Text: body},
				}},
			}, nil
		})
	}

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{Stateless: true})
	mux.Handle("/mcp", handler)
}

// emptyMCPInput is the input type for tools that take no arguments —
// AddTool requires an object-shaped In type, so a bare empty struct stands
// in for "nothing to pass".
type emptyMCPInput struct{}

type boardMCPInput = manager.BoardQuery

type refMCPInput struct {
	Ref string `json:"ref"`
}

type eventsMCPInput struct {
	Since int64 `json:"since"`
}

type phraseMCPInput struct {
	Phrase string `json:"phrase"`
}

type textMCPInput struct {
	Text string `json:"text"`
}

type draftMCPInput struct {
	Draft manager.Intent `json:"draft"`
}

type inboxMCPInput struct {
	Action  string `json:"action"`
	Text    string `json:"text,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Project string `json:"project,omitempty"`
	Title   string `json:"title,omitempty"`
}

type aliasMCPInput struct {
	Project string `json:"project"`
	Alias   string `json:"alias"`
}

type projectMCPInput struct {
	Project string `json:"project"`
}

type describeMCPInput struct {
	Project string `json:"project"`
	Summary string `json:"summary"`
}

type answerMCPInput struct {
	Ref    string `json:"ref"`
	Text   string `json:"text"`
	Status string `json:"status,omitempty"`
}

type reviewMCPInput struct {
	Ref     string `json:"ref"`
	Verdict string `json:"verdict"`
	Comment string `json:"comment,omitempty"`
}
