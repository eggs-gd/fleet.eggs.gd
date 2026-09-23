package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOverlayRoundTripAndAtomicSave(t *testing.T) {
	root := t.TempDir()
	enabled := false
	original := Overlay{
		ScanRoot: root,
		Agents: map[string]AgentOverlay{
			"codex": {
				Enabled:             &enabled,
				Executable:          filepath.Join(root, "codex"),
				RoutingInstructions: "Use for repo implementation.",
			},
		},
	}
	if err := os.WriteFile(filepath.Join(root, "codex"), []byte("#!/bin/sh\necho 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveOverlay(root, original); err != nil {
		t.Fatal(err)
	}
	got, err := LoadOverlay(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ScanRoot != root {
		t.Fatalf("scanRoot = %q", got.ScanRoot)
	}
	agent := got.Agent("codex")
	if agent.Enabled == nil || *agent.Enabled {
		t.Fatalf("enabled = %#v", agent.Enabled)
	}
	if agent.Executable != original.Agents["codex"].Executable {
		t.Fatalf("executable = %q", agent.Executable)
	}
	if agent.RoutingInstructions != "Use for repo implementation." {
		t.Fatalf("routing = %q", agent.RoutingInstructions)
	}
}

func TestOverlayManagerRoundTrip(t *testing.T) {
	root := t.TempDir()
	original := Overlay{Manager: ManagerOverlay{Agent: "codex", ThreadID: "thread-1", BoundAt: "2026-09-14T12:00:00Z"}}
	if err := SaveOverlay(root, original); err != nil {
		t.Fatal(err)
	}
	got, err := LoadOverlay(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Manager != original.Manager {
		t.Fatalf("manager = %#v, want %#v", got.Manager, original.Manager)
	}
}

func TestMergeOverlaySetsAndClearsManagerBinding(t *testing.T) {
	base := Overlay{}
	agent := "claude"
	threadID := "thread-9"
	bound := MergeOverlay(base, PatchRequest{Manager: &ManagerPatch{Agent: &agent, ThreadID: &threadID}})
	if bound.Manager.Agent != "claude" || bound.Manager.ThreadID != "thread-9" || bound.Manager.BoundAt == "" {
		t.Fatalf("manager = %#v", bound.Manager)
	}

	empty := ""
	cleared := MergeOverlay(bound, PatchRequest{Manager: &ManagerPatch{Agent: &empty}})
	if cleared.Manager != (ManagerOverlay{}) {
		t.Fatalf("manager should clear fully, got %#v", cleared.Manager)
	}
}

func TestDecodePatchRejectsUnknownManagerAgent(t *testing.T) {
	_, err := DecodePatch([]byte(`{"manager":{"agent":"grok","threadId":"x"}}`))
	if err == nil || !strings.Contains(err.Error(), "unknown manager agent") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseOverlayRejectsUnknownKeys(t *testing.T) {
	_, err := ParseOverlay([]byte("foo: bar\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown overlay key") {
		t.Fatalf("err = %v", err)
	}
}

func TestDecodePatchRejectsUnknownJSON(t *testing.T) {
	_, err := DecodePatch([]byte(`{"theme":"dark"}`))
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	_, err = DecodePatch([]byte(`{"agents":{"grok":{"enabled":true}}}`))
	if err == nil || !strings.Contains(err.Error(), "unknown agent") {
		t.Fatalf("err = %v", err)
	}
}

func TestMergeOverlayClearsEmptyAgent(t *testing.T) {
	on := true
	base := Overlay{Agents: map[string]AgentOverlay{"codex": {Enabled: &on, Executable: "/bin/codex"}}}
	empty := ""
	got := MergeOverlay(base, PatchRequest{Agents: map[string]AgentPatch{
		"codex": {Executable: &empty, Enabled: nil},
	}})
	if got.Agents["codex"].Executable != "" {
		t.Fatalf("executable should clear, got %#v", got.Agents["codex"])
	}
}

func TestValidateExecutableRejectsRelative(t *testing.T) {
	if _, err := ValidateExecutable("codex"); err == nil {
		t.Fatal("relative path must fail")
	}
}

func TestScannerBusy(t *testing.T) {
	started := make(chan struct{})
	block := make(chan struct{})
	scanner := NewScanner(func(scanRoot, registryDir string) error {
		close(started)
		<-block
		return nil
	}, nil)
	if err := scanner.Start("/tmp", "/tmp/_registry"); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := scanner.Start("/tmp", "/tmp/_registry"); err != ErrScanBusy {
		t.Fatalf("err = %v", err)
	}
	close(block)
}
