package settings

import (
	"reflect"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/execution/providers"
	"github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
)

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

func TestOverlayRoundTripWithEveryField(t *testing.T) {
	in := Overlay{
		ScanRoots:      []string{"/a b", "/c"},
		SessionTimeout: "10m",
		Launch:         "live",
		Agents: map[string]AgentOverlay{
			"claude": {Enabled: boolPtr(false), Executable: "/usr/bin/claude", RoutingInstructions: "line one\n\n  indented two"},
			"codex":  {},
		},
		Manager: ManagerOverlay{Agent: "codex", ThreadID: "t-1", Workspace: "w", BoundAt: "2026-01-01T00:00:00Z"},
	}
	out, err := ParseOverlay(EncodeOverlay(in))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out.ScanRoots, in.ScanRoots) || out.SessionTimeout != "10m" || out.Launch != "live" {
		t.Fatalf("top level = %+v", out)
	}
	claude := out.Agents["claude"]
	if claude.Enabled == nil || *claude.Enabled || claude.Executable != "/usr/bin/claude" || claude.RoutingInstructions != in.Agents["claude"].RoutingInstructions {
		t.Fatalf("claude = %+v", claude)
	}
	if _, ok := out.Agents["codex"]; ok {
		t.Fatal("an empty agent must not be written")
	}
	if out.Manager != in.Manager {
		t.Fatalf("manager = %+v", out.Manager)
	}
}

func TestParseOverlayInlineAndBlockValues(t *testing.T) {
	text := "# comment\nsessionTimeout: \"5m\"\nscanRoots: /one\nscanRoots:\n  - /two\nagents:\n  gemini:\n    enabled: true\n    routingInstructions: short text\n  cursor:\n    routingInstructions: >\n      folded\n      text\n"
	out, err := ParseOverlay([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if out.SessionTimeout != "5m" || !reflect.DeepEqual(out.ScanRoots, []string{"/one", "/two"}) {
		t.Fatalf("out = %+v", out)
	}
	if g := out.Agents["gemini"]; g.Enabled == nil || !*g.Enabled || g.RoutingInstructions != "short text" {
		t.Fatalf("gemini = %+v", g)
	}
	if c := out.Agents["cursor"]; c.RoutingInstructions != "folded\ntext" {
		t.Fatalf("cursor = %q", c.RoutingInstructions)
	}
}

func TestParseOverlayErrors(t *testing.T) {
	cases := map[string]string{
		"unknown top key":     "mystery: 1\n",
		"unknown manager key": "manager:\n  mystery: 1\n",
		"unknown agent":       "agents:\n  nobody:\n",
		"unknown agent key":   "agents:\n  claude:\n    mystery: 1\n",
		"bad boolean":         "agents:\n  claude:\n    enabled: maybe\n",
		"no colon":            "just words\n",
		"empty scan root":     "scanRoots:\n  - \n",
		"unexpected indent":   "launch: live\n      deep: 1\n",
	}
	for name, text := range cases {
		if _, err := ParseOverlay([]byte(text)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := ParseOverlay([]byte("agents:\n  claude:\n    enabled: maybe\n")); err == nil || !strings.Contains(err.Error(), "agents.claude.enabled") {
		t.Errorf("error should name the key, got %v", err)
	}
}

func TestMergeAgentPatch(t *testing.T) {
	cur := AgentOverlay{Executable: "old", RoutingInstructions: "old text"}
	got := mergeAgentPatch(cur, AgentPatch{Enabled: boolPtr(true), Executable: strPtr(" new "), RoutingInstructions: strPtr(" x ")})
	if got.Enabled == nil || !*got.Enabled || got.Executable != "new" || got.RoutingInstructions != "x" {
		t.Fatalf("got %+v", got)
	}
	if kept := mergeAgentPatch(cur, AgentPatch{}); kept != cur {
		t.Fatalf("an empty patch changes nothing, got %+v", kept)
	}
}

func TestMergeManagerPatch(t *testing.T) {
	base := ManagerOverlay{Agent: "codex", ThreadID: "t1", BoundAt: "then"}
	if got := mergeManagerPatch(base, ManagerPatch{}); got != base {
		t.Fatalf("unchanged binding keeps its time, got %+v", got)
	}
	got := mergeManagerPatch(base, ManagerPatch{ThreadID: strPtr("t2")})
	if got.ThreadID != "t2" || got.BoundAt == "then" || got.BoundAt == "" {
		t.Fatalf("a new thread is stamped, got %+v", got)
	}
	if got := mergeManagerPatch(base, ManagerPatch{Agent: strPtr("")}); got != (ManagerOverlay{}) {
		t.Fatalf("without an agent the binding is cleared, got %+v", got)
	}
}

func TestMergeOverlayDropsEmptiedAgents(t *testing.T) {
	base := Overlay{Agents: map[string]AgentOverlay{"claude": {Executable: "x"}}}
	out := MergeOverlay(base, PatchRequest{Agents: map[string]AgentPatch{"claude": {Executable: strPtr("")}, "codex": {Enabled: boolPtr(false)}}})
	if _, ok := out.Agents["claude"]; ok {
		t.Fatal("an agent with nothing left is removed")
	}
	if c := out.Agents["codex"]; c.Enabled == nil || *c.Enabled {
		t.Fatalf("codex = %+v", c)
	}
}

func TestCountClosedSessions(t *testing.T) {
	mk := func(claim, status string) execution.SessionSummary {
		return execution.SessionSummary{RuntimeSession: executionapi.RuntimeSession{ClaimID: claim, Status: status}}
	}
	groups := []execution.SessionGroup{
		{Sessions: []execution.SessionSummary{mk("a", "succeeded"), mk("b", "failed"), mk("c", "running")}},
		{Sessions: []execution.SessionSummary{mk("a", "succeeded"), mk("d", "timed_out")}},
	}
	closed, failed := countClosedSessions(groups)
	if closed != 3 || failed != 2 {
		t.Fatalf("closed %d failed %d", closed, failed)
	}
}

func TestClassifySessions(t *testing.T) {
	runtime := []executionapi.RuntimeSession{
		{ClaimID: "1", Status: "running"},
		{ClaimID: "2", Status: "waiting_input"},
		{ClaimID: "3", Status: "succeeded"},
	}
	orphans := []execution.OrphanedTask{{ExecutionState: "resumable"}, {Reason: "gone"}}
	got := classifySessions(runtime, nil, orphans)
	if got.Active != 1 || got.HITL != 1 || got.Orphaned != 2 || got.OrphanedResumable != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestCollectIssues(t *testing.T) {
	on := BoolField{Value: true}
	agents := Agents{Providers: []Agent{
		{Name: "Claude", Enabled: on, Status: providers.AgentNonCanonical, Error: "wrong"},
		{Name: "Codex", Enabled: on, Status: providers.AgentNotFound, Expected: "codex"},
		{Name: "Off", Enabled: BoolField{}, Status: providers.AgentNotFound},
	}}
	projects := Projects{
		Roots:     []ProjectRoot{{Path: "/gone", Reachable: false}, {Path: "/here", Reachable: true}},
		DataPaths: []DataPath{{Name: "Data root", Path: "/d", Exists: true, Writable: false}, {Name: "Other", Exists: true}},
	}
	integrations := Integrations{MCP: []MCPServer{{Name: "svelte", Configured: true, Detected: false, Expected: "npx"}, {Name: "ok", Configured: true, Detected: true}}}
	issues := collectIssues(projects, agents, integrations, SessionCounts{Orphaned: 2, OrphanedResumable: 1})
	codes := []string{}
	for _, i := range issues {
		codes = append(codes, i.Code)
	}
	want := []string{"agent_non_canonical", "agent_not_found", "root_unavailable", "data_not_writable", "mcp_undetected", "orphaned_sessions"}
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("codes = %v, want %v", codes, want)
	}
	if got := collectIssues(Projects{}, Agents{}, Integrations{}, SessionCounts{}); len(got) != 0 {
		t.Fatalf("nothing wrong, got %v", got)
	}
}
