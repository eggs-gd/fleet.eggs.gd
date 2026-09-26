package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOverlayRoundTripAndAtomicSave(t *testing.T) {
	root := t.TempDir()
	enabled := false
	original := Overlay{
		ScanRoots: []string{root},
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
	if len(got.ScanRoots) != 1 || got.ScanRoots[0] != root {
		t.Fatalf("scanRoots = %#v", got.ScanRoots)
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

func TestSessionTimeoutRoundTripAndFlagOverride(t *testing.T) {
	root := t.TempDir()
	original := Overlay{SessionTimeout: "30m"}
	if err := SaveOverlay(root, original); err != nil {
		t.Fatal(err)
	}
	got, err := LoadOverlay(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionTimeout != "30m" {
		t.Fatalf("sessionTimeout = %q", got.SessionTimeout)
	}
	fromFile, err := ResolveServeTimeout(false, time.Minute, got)
	if err != nil || fromFile != 30*time.Minute {
		t.Fatalf("overlay timeout = %s err=%v", fromFile, err)
	}
	fromFlag, err := ResolveServeTimeout(true, 5*time.Minute, got)
	if err != nil || fromFlag != 5*time.Minute {
		t.Fatalf("flag timeout = %s err=%v", fromFlag, err)
	}
	fallback, err := ResolveServeTimeout(false, 0, Overlay{})
	if err != nil || fallback != DefaultSessionTimeout {
		t.Fatalf("default timeout = %s err=%v", fallback, err)
	}
	if _, err := ResolveServeTimeout(false, 0, Overlay{SessionTimeout: "nope"}); err == nil {
		t.Fatal("expected invalid sessionTimeout")
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

func TestParseOverlayReadsLegacyScanRootAndList(t *testing.T) {
	legacy, err := ParseOverlay([]byte("scanRoot: /one\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(legacy.ScanRoots) != 1 || legacy.ScanRoots[0] != "/one" {
		t.Fatalf("legacy = %#v", legacy.ScanRoots)
	}

	listed, err := ParseOverlay([]byte("scanRoots:\n  - /a\n  - /b\n  - /a\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(normalizeScanRoots(listed.ScanRoots), ",") != "/a,/b" {
		t.Fatalf("list = %#v", listed.ScanRoots)
	}

	encoded := string(EncodeOverlay(Overlay{ScanRoots: []string{"/a", "/b", "/a"}}))
	if !strings.Contains(encoded, "scanRoots:\n") || strings.Contains(encoded, "scanRoot:") {
		t.Fatalf("encoded = %s", encoded)
	}

	if _, err := ParseOverlay([]byte("nope: 1\n")); err == nil {
		t.Fatal("unknown key must fail")
	}
}

func TestValidateExecutableRejectsRelativePath(t *testing.T) {
	if _, err := ValidateExecutable("codex"); err == nil {
		t.Fatal("relative path must fail")
	}
}

func TestScannerBusy(t *testing.T) {
	started := make(chan struct{})
	block := make(chan struct{})
	scanner := NewScanner(func(scanRoots []string, registryDir string) error {
		close(started)
		<-block
		return nil
	}, nil)
	if err := scanner.Start([]string{"/tmp"}, "/tmp/_registry"); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := scanner.Start([]string{"/tmp"}, "/tmp/_registry"); err != ErrScanBusy {
		t.Fatalf("err = %v", err)
	}
	close(block)
}
