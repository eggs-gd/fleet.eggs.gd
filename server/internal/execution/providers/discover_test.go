package providers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyCodexDiscoveryNonCanonicalWhenOnlyChatGPTAppExists(t *testing.T) {
	row := classifyCodexDiscovery(AgentDiscovery{
		ID:   "codex",
		Name: "Codex",
		DetectedExecutables: []DetectedExecutable{
			{Path: chatGPTAppCodex, Source: "bundled"},
		},
	}, "", "", errors.New("codex is not on PATH"))
	if row.Status != AgentNonCanonical {
		t.Fatalf("status = %q, want %s", row.Status, AgentNonCanonical)
	}
	if row.Canonical {
		t.Fatal("ChatGPT.app-only install must not be canonical")
	}
	if row.EffectiveExecutable != "" {
		t.Fatalf("effective = %q, ChatGPT.app must not become effective", row.EffectiveExecutable)
	}
	if len(row.DetectedExecutables) != 1 || row.DetectedExecutables[0].Path != chatGPTAppCodex {
		t.Fatalf("detected = %#v", row.DetectedExecutables)
	}
}

func TestClassifyCodexDiscoveryReadyKeepsChatGPTAppAsOtherInstall(t *testing.T) {
	resolved := "/opt/homebrew/bin/codex"
	row := classifyCodexDiscovery(AgentDiscovery{
		ID:   "codex",
		Name: "Codex",
		DetectedExecutables: []DetectedExecutable{
			{Path: resolved, Source: "path"},
			{Path: chatGPTAppCodex, Source: "bundled"},
		},
	}, resolved, "path", nil)
	if row.Status != AgentReady || !row.Canonical {
		t.Fatalf("row = %#v", row)
	}
	if row.EffectiveExecutable != resolved || row.EffectiveSource != "path" {
		t.Fatalf("effective = %q %q", row.EffectiveExecutable, row.EffectiveSource)
	}
	if len(row.OtherInstallations) != 1 || row.OtherInstallations[0] != chatGPTAppCodex {
		t.Fatalf("other = %#v", row.OtherInstallations)
	}
}

func TestIsChatGPTAppCodex(t *testing.T) {
	if !isChatGPTAppCodex(chatGPTAppCodex) {
		t.Fatal("expected ChatGPT.app path to be non-canonical")
	}
	if isChatGPTAppCodex("/opt/homebrew/bin/codex") {
		t.Fatal("standalone PATH install is canonical")
	}
}

func TestDiscoverAgentsOnlyRegisteredProviders(t *testing.T) {
	installFakeExecutable(t, "claude")
	installFakeExecutable(t, "codex")
	installFakeExecutable(t, "cursor-agent")
	installFakeExecutable(t, "agy")
	got := DiscoverAgents()
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4 (claude, codex, cursor, gemini — only providers.NewRegistry())", len(got))
	}
	want := []string{"claude", "codex", "cursor", "gemini"}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("got[%d].ID = %q, want %q", i, got[i].ID, id)
		}
	}
}

func TestResolveCodexPrefersEnvThenOverlayThenPath(t *testing.T) {
	pathBin := installFakeExecutable(t, "codex")
	overlay := installNamedFake(t, "overlay-codex")
	envBin := installNamedFake(t, "env-codex")

	ApplyLocalAgents(map[string]LocalAgent{"codex": {Executable: overlay}})
	t.Cleanup(func() { ApplyLocalAgents(nil) })

	resolved, source, err := resolveCodexBinaryWithSource()
	if err != nil || resolved != overlay || source != "core.local.yaml" {
		t.Fatalf("overlay resolve = %q %q %v", resolved, source, err)
	}

	t.Setenv("CORE_CODEX_BINARY", envBin)
	resolved, source, err = resolveCodexBinaryWithSource()
	if err != nil || resolved != envBin || source != "env" {
		t.Fatalf("env resolve = %q %q %v", resolved, source, err)
	}

	t.Setenv("CORE_CODEX_BINARY", "")
	ApplyLocalAgents(nil)
	resolved, source, err = resolveCodexBinaryWithSource()
	if err != nil || resolved != pathBin || source != "path" {
		t.Fatalf("path resolve = %q %q %v", resolved, source, err)
	}
}

func TestAgentEnabledDefaultsTrue(t *testing.T) {
	ApplyLocalAgents(nil)
	if !AgentEnabled("codex") {
		t.Fatal("unset overlay must leave agents enabled")
	}
	off := false
	ApplyLocalAgents(map[string]LocalAgent{"codex": {Enabled: &off}})
	t.Cleanup(func() { ApplyLocalAgents(nil) })
	if AgentEnabled("codex") {
		t.Fatal("overlay enabled=false must disable new launches")
	}
	if !AgentEnabled("claude") {
		t.Fatal("other agents stay enabled")
	}
}

func installNamedFake(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
