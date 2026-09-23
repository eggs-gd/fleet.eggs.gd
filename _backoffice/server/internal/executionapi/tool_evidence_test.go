package executionapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSelectRequirementProfilesBySignals(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   RequirementInput
		want []string
	}{
		{
			name: "go server paths",
			in: RequirementInput{
				TaskType:   "feature",
				Title:      "Fix launch gate",
				Body:       "Touch `_backoffice/server/internal/execution/host_launch.go`.",
				Repository: "core.eggs.gd",
			},
			want: []string{ProfileGo},
		},
		{
			name: "svelte view paths",
			in: RequirementInput{
				TaskType: "bug",
				Title:    "Dashboard jitter",
				Body:     "Update `_backoffice/view/src/App.svelte` and styles.",
			},
			want: []string{ProfileMCPDocs, ProfileSvelte},
		},
		{
			name: "architecture refactor",
			in: RequirementInput{
				TaskType:   "feature",
				Title:      "Split corechain module boundaries",
				Body:       "Refactor architecture ownership without new Contour wrappers.",
				Repository: "core.eggs.gd",
			},
			want: []string{ProfileArchitecture, ProfileGo},
		},
		{
			name: "coding default go when unclear in core tree",
			in: RequirementInput{
				TaskType:   "maintenance",
				Title:      "Tidy docs",
				Body:       "Update wording only.",
				Repository: "core.eggs.gd",
			},
			want: []string{ProfileGo},
		},
		{
			name: "non-go external repo does not default go",
			in: RequirementInput{
				TaskType:   "bug",
				Title:      "Cover letter channel",
				Body:       "Fix Python cover generation prompts.",
				Repository: "eGGs.gd.prod/career-wizard",
				WorkingDir: "/Users/me/Projects/eGGs.gd.prod/career-wizard",
			},
			want: nil,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SelectRequirementProfiles(tc.in)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("SelectRequirementProfiles() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRequiredToolsForProfiles(t *testing.T) {
	t.Parallel()
	got := RequiredToolsForProfiles([]string{ProfileGo, ProfileSvelte, ProfileArchitecture, ProfileMCPDocs})
	want := []string{ToolFindPatterns, ToolGoDiagnostics, ToolListSections, ToolSvelteAutofixer}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("RequiredToolsForProfiles() = %v, want %v", got, want)
	}
}

func TestExtractToolCallsFromCodexParams(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{
		"item": {
			"id": "item_1",
			"type": "mcpToolCall",
			"server": "user-gopls",
			"tool": "go_diagnostics",
			"result": "ok: 0 issues"
		}
	}`)
	at := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	calls := ExtractToolCallsFromCodexParams("item/completed", raw, at)
	if len(calls) != 1 {
		t.Fatalf("ExtractToolCallsFromCodexParams() len = %d, want 1: %#v", len(calls), calls)
	}
	if calls[0].Name != ToolGoDiagnostics {
		t.Fatalf("name = %q, want %q", calls[0].Name, ToolGoDiagnostics)
	}
	if calls[0].Status != "completed" {
		t.Fatalf("status = %q, want completed", calls[0].Status)
	}
	if !strings.Contains(calls[0].Summary, "0 issues") {
		t.Fatalf("summary = %q, want result snippet", calls[0].Summary)
	}
	if calls[0].Source != toolEvidenceSourceRPC {
		t.Fatalf("source = %q", calls[0].Source)
	}
}

func TestExtractToolCallsFromSessionLogIgnoresWorkerSelfReport(t *testing.T) {
	t.Parallel()
	logText := `{"outcome":"completed","summary":"I ran go_diagnostics and svelte-autofixer","tests":["go_diagnostics"]}`
	calls := ExtractToolCallsFromSessionLog(logText, time.Now())
	if len(calls) != 0 {
		t.Fatalf("self-report-only log produced evidence %#v", calls)
	}
}

func TestExtractToolCallsFromSessionLogFindsCursorMCPInvocation(t *testing.T) {
	t.Parallel()
	logText := strings.Join([]string{
		"[cursor] thinking",
		`CallMcpTool toolName=go_diagnostics server=user-gopls`,
		`{"type":"tool_use","name":"svelte-autofixer","input":{}}`,
		`{"outcome":"completed","summary":"done"}`,
	}, "\n")
	calls := ExtractToolCallsFromSessionLog(logText, time.Now())
	names := map[string]bool{}
	for _, call := range calls {
		names[call.Name] = true
	}
	if !names[ToolGoDiagnostics] || !names[ToolSvelteAutofixer] {
		t.Fatalf("names = %#v, want go_diagnostics and svelte-autofixer", names)
	}
}

func TestExtractToolCallsFromCursorTranscriptCallMcpTool(t *testing.T) {
	t.Parallel()
	transcript := strings.Join([]string{
		`{"role":"user","message":{"content":[{"type":"text","text":"run go_diagnostics please"}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"CallMcpTool","input":{"server":"user-gopls","toolName":"go_diagnostics","arguments":{"files":["a.go"]}}}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"svelte-autofixer","input":{"code":"<script></script>"}}]}}`,
	}, "\n")
	calls := ExtractToolCallsFromCursorTranscript(transcript, time.Now())
	names := map[string]bool{}
	for _, call := range calls {
		names[call.Name] = true
		if call.Source != toolEvidenceSourceCursorTranscript {
			t.Fatalf("source = %q", call.Source)
		}
	}
	if !names[ToolGoDiagnostics] || !names[ToolSvelteAutofixer] {
		t.Fatalf("names = %#v, want go_diagnostics and svelte-autofixer", names)
	}
	// Prose mention in the user message must not count as evidence by itself.
	if len(calls) != 2 {
		t.Fatalf("len(calls) = %d, want 2 (prose-only mention ignored)", len(calls))
	}
}

func TestCursorProjectSlug(t *testing.T) {
	t.Parallel()
	got := cursorProjectSlug("/Users/operator/Projects/core.eggs.gd")
	want := "Users-dukobpa3-Projects-core-eggs-gd"
	if got != want {
		t.Fatalf("cursorProjectSlug() = %q, want %q", got, want)
	}
}

func TestFindCursorAgentTranscriptMissing(t *testing.T) {
	t.Parallel()
	if got := FindCursorAgentTranscript("", "/tmp"); got != "" {
		t.Fatalf("empty chat id should miss, got %q", got)
	}
	if got := FindCursorAgentTranscript("no-such-chat-id-for-core-140", "/tmp/not-a-real-workspace"); got != "" {
		t.Fatalf("missing transcript should be empty, got %q", got)
	}
}

func TestProjectToolUsageMissingAndWarning(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC)
	usage := ProjectToolUsage(
		[]string{ProfileGo, ProfileSvelte},
		[]string{ToolGoDiagnostics, ToolSvelteAutofixer},
		[]ToolCallEvidence{{Name: ToolGoDiagnostics, Source: toolEvidenceSourceRPC, Status: "completed"}},
		at,
	)
	if len(usage.Missing) != 1 || usage.Missing[0] != ToolSvelteAutofixer {
		t.Fatalf("missing = %#v", usage.Missing)
	}
	if !strings.Contains(usage.Warning, ToolSvelteAutofixer) {
		t.Fatalf("warning = %q", usage.Warning)
	}
	if !usage.HasMissing() {
		t.Fatal("HasMissing() = false")
	}
}

func TestListSectionsSatisfiedByGetDocumentationAlias(t *testing.T) {
	t.Parallel()
	usage := ProjectToolUsage(
		[]string{ProfileMCPDocs},
		[]string{ToolListSections},
		[]ToolCallEvidence{{Name: ToolGetDocumentation}},
		time.Now(),
	)
	if len(usage.Missing) != 0 {
		t.Fatalf("missing = %#v, want none via alias", usage.Missing)
	}
}

func TestWithToolUsageWarning(t *testing.T) {
	t.Parallel()
	evidence := ToolUsageEvidence{
		Required: []string{ToolGoDiagnostics},
		Missing:  []string{ToolGoDiagnostics},
		Warning:  ToolUsageWarning(ToolUsageEvidence{Required: []string{ToolGoDiagnostics}, Missing: []string{ToolGoDiagnostics}}),
	}
	got := WithToolUsageWarning("Agent execution completed.", evidence)
	if !strings.Contains(got, "Missing required tool evidence") {
		t.Fatalf("summary missing warning: %q", got)
	}
	// Idempotent append.
	again := WithToolUsageWarning(got, evidence)
	if strings.Count(again, "Missing required tool evidence") != 1 {
		t.Fatalf("warning duplicated: %q", again)
	}
}

func TestInitToolUsageSeedsRequired(t *testing.T) {
	t.Parallel()
	usage := InitToolUsage(RequirementInput{
		TaskType: "feature",
		Title:    "Show tool evidence",
		Body:     "Update RuntimeSessions.svelte and execution session model under _backoffice/server.",
	}, time.Now())
	if len(usage.Required) == 0 {
		t.Fatal("expected required tools")
	}
	if len(usage.Missing) != len(usage.Required) {
		t.Fatalf("initial missing=%v required=%v", usage.Missing, usage.Required)
	}
}
