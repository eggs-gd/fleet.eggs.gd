package execution

import (
	"strings"
	"testing"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

func TestInitAndObserveSessionToolUsage(t *testing.T) {
	session := &RuntimeSession{ClaimID: "claim-1", WorkingDir: "/tmp/core", Repository: "core.eggs.gd"}
	task := tasklifecycle.Task{
		Type:  "feature",
		Title: "Dashboard tool evidence",
		Body:  "Change `_backoffice/view/src/RuntimeSessions.svelte` and `_backoffice/server/internal/executionapi/session.go`.",
	}
	InitSessionToolUsage(session, task)
	if session.ToolUsage == nil || len(session.ToolUsage.Required) == 0 {
		t.Fatalf("InitSessionToolUsage left empty evidence: %#v", session.ToolUsage)
	}
	ObserveSessionToolCalls(session, []ToolCallEvidence{
		{Name: executionapi.ToolGoDiagnostics, Source: "provider_rpc", Status: "completed", CalledAt: time.Now().UTC().Format(time.RFC3339)},
		{Name: executionapi.ToolSvelteAutofixer, Source: "session_log", Status: "observed"},
		{Name: executionapi.ToolListSections, Source: "session_log", Status: "observed"},
	})
	if session.ToolUsage.HasMissing() {
		t.Fatalf("expected no missing tools after observe, got %#v", session.ToolUsage.Missing)
	}
	state := ExecutionStateForSession(*session)
	if state.ToolWarning != "" {
		t.Fatalf("ToolWarning = %q, want empty", state.ToolWarning)
	}
}

func TestExecutionResultFromWorkerWithToolsAppendsWarning(t *testing.T) {
	task := tasklifecycle.Task{ID: "t1", RelativePath: "Work/x/tasks/a.md"}
	tools := &ToolUsageEvidence{
		Required: []string{executionapi.ToolGoDiagnostics},
		Missing:  []string{executionapi.ToolGoDiagnostics},
	}
	tools.Warning = executionapi.ToolUsageWarning(*tools)
	got := ExecutionResultFromWorkerWithTools(task, &tasklifecycle.WorkerResult{
		Outcome: "completed",
		Summary: "Shipped feature.",
	}, "_registry/sessions/c.log", tools)
	if got.Outcome != taskflow.ExecutionCompleted {
		t.Fatalf("outcome = %s", got.Outcome)
	}
	if !strings.Contains(got.Summary, "Missing required tool evidence: go_diagnostics") {
		t.Fatalf("summary = %q", got.Summary)
	}
}

func TestRefreshSessionToolUsageFromLogProjection(t *testing.T) {
	session := &RuntimeSession{
		ClaimID: "claim-2",
		ToolUsage: &ToolUsageEvidence{
			Profiles: []string{executionapi.ProfileGo},
			Required: []string{executionapi.ToolGoDiagnostics},
			Missing:  []string{executionapi.ToolGoDiagnostics},
		},
	}
	logText := `[codex -> core] {"method":"item/completed","params":{"item":{"type":"mcpToolCall","tool":"go_diagnostics","result":"clean"}}}`
	RefreshSessionToolUsageFromLog(session, logText)
	if session.ToolUsage.HasMissing() {
		t.Fatalf("missing after refresh: %#v", session.ToolUsage)
	}
	if len(session.ToolUsage.Used) == 0 || session.ToolUsage.Used[0].Name != executionapi.ToolGoDiagnostics {
		t.Fatalf("used = %#v", session.ToolUsage.Used)
	}
}
