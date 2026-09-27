package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func TestBuildReadModelDoesNotLeakSecretsOrInventProviders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Work", "INDEX.md"), "# Work\n")
	writeFile(t, filepath.Join(root, "core.config.yaml"), `taskProvider:
  type: markdown
  workspace: eggs_gd
  baseUrl: https://api.plane.so
  tokenEnv: PLANE_API_TOKEN
  coreProject: core_eggs_gd
`)
	writeFile(t, filepath.Join(root, ".mcp.json"), `{
  "mcpServers": {
    "gopls": {"type": "stdio", "command": "gopls", "args": ["mcp"]}
  }
}`)
	writeFile(t, filepath.Join(root, "Fleet", "codex.md"), "# Codex\n\n## Best At\n\n- Repository implementation.\n\n## Use For\n\n- feature\n")
	const secret = "plane-token-should-never-appear-in-settings"
	t.Setenv("PLANE_API_TOKEN", secret)

	snap := Build(Input{
		Version:        "0.0.1",
		Addr:           "127.0.0.1:8787",
		StartedAt:      time.Now().Add(-time.Minute),
		CoreRoot:       root,
		DryRun:         true,
		SessionTimeout: 10 * time.Minute,
		RuntimeSessions: []executionapi.RuntimeSession{{
			ClaimID:         "hitl-1",
			Status:          "waiting_input",
			ExecutionStatus: "waiting_input",
			Result:          &tasklifecycle.WorkerResult{Outcome: "needs_input"},
		}},
		OrphanedTasks: []execution.OrphanedTask{{
			ClaimID:        "orphan-1",
			ExecutionState: "resumable",
			Reason:         "orphaned but resumable",
		}},
	})

	if snap.Projects.ScanRootsWritable != true {
		t.Fatal("scan roots must be writable")
	}
	if snap.Projects.TaskBackend.Active != "markdown" {
		t.Fatalf("active backend = %q", snap.Projects.TaskBackend.Active)
	}
	if snap.Projects.TaskBackend.Plane != nil || snap.Projects.TaskBackend.PlaneDeclared {
		t.Fatal("task-provider details outside markdown must stay out of Settings")
	}
	if !snap.Projects.MultiRootSupported {
		t.Fatal("multi-root scan must be supported")
	}

	ids := []string{}
	for _, agent := range snap.Agents.Providers {
		ids = append(ids, agent.ID)
	}
	if strings.Join(ids, ",") != "claude,codex,cursor,gemini" {
		t.Fatalf("providers = %v (only the registered providers.NewRegistry() set must appear)", ids)
	}
	for _, agent := range snap.Agents.Providers {
		if !agent.Enabled.Writable || !agent.ConfiguredExecutable.Writable || !agent.RoutingInstructions.Writable {
			t.Fatalf("agent %s writable flags = %#v", agent.ID, agent)
		}
		if agent.EffectiveExecutable.Writable {
			t.Fatalf("effective executable must not be writable: %#v", agent.EffectiveExecutable)
		}
	}

	hitl := false
	for _, row := range snap.Workflow.RuntimeOutcomes {
		if strings.Contains(row.Outcome, "needs_input") {
			hitl = true
			if row.Task != "doing" || row.ReleasesSlot || row.Session != "waiting_input" {
				t.Fatalf("HITL row = %#v", row)
			}
		}
	}
	if !hitl {
		t.Fatal("missing live HITL outcome row")
	}
	if snap.Workflow.HITL.EqualsClosed || snap.Workflow.HITL.ReleasesSlot || snap.Workflow.HITL.TaskStatus != "doing" {
		t.Fatalf("HITL contract = %#v", snap.Workflow.HITL)
	}
	if snap.Manager.Session != nil {
		t.Fatalf("manager session = %#v, want none", snap.Manager.Session)
	}
	if snap.Diagnostics.Sessions.HITL != 1 {
		t.Fatalf("HITL sessions = %d", snap.Diagnostics.Sessions.HITL)
	}
	if snap.Diagnostics.Sessions.Orphaned != 1 || snap.Diagnostics.Sessions.OrphanedResumable != 1 {
		t.Fatalf("orphans = %+v", snap.Diagnostics.Sessions)
	}

	payload, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), secret) {
		t.Fatal("Settings JSON leaked PLANE_API_TOKEN value")
	}
	if strings.Contains(strings.ToLower(string(payload)), "plane") {
		t.Fatal("Settings JSON still mentions plane")
	}
}

func TestBuildEnvWinsOverOverlayExecutable(t *testing.T) {
	root := t.TempDir()
	overlayBin := filepath.Join(root, "overlay-codex")
	envBin := filepath.Join(root, "env-codex")
	writeFile(t, overlayBin, "#!/bin/sh\necho overlay\n")
	writeFile(t, envBin, "#!/bin/sh\necho env\n")
	if err := os.Chmod(overlayBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(envBin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, OverlayFileName), "agents:\n  codex:\n    executable: "+overlayBin+"\n")
	t.Setenv("FLEET_CODEX_BINARY", envBin)

	snap := Build(Input{CoreRoot: root})
	var codex Agent
	for _, agent := range snap.Agents.Providers {
		if agent.ID == "codex" {
			codex = agent
		}
	}
	if codex.ConfiguredExecutable.Value != overlayBin || codex.ConfiguredExecutable.Source != SourceLocal {
		t.Fatalf("configured = %#v", codex.ConfiguredExecutable)
	}
	if codex.EffectiveExecutable.Value != envBin || codex.EffectiveExecutable.Source != SourceEnv {
		t.Fatalf("effective = %#v", codex.EffectiveExecutable)
	}
	if codex.EffectiveExecutable.OverriddenBy != "FLEET_CODEX_BINARY" {
		t.Fatalf("overridden_by = %q", codex.EffectiveExecutable.OverriddenBy)
	}
	if codex.EffectiveExecutable.Writable {
		t.Fatal("effective must not look writable when env wins")
	}
}

func TestExtractSectionBestAt(t *testing.T) {
	got := extractSection("# Codex\n\n## Best At\n\n- Impl.\n\n## Use For\n\n- feature\n", "Best At")
	if got != "- Impl." {
		t.Fatalf("got %q", got)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
