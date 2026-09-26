package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureManagerMCPWritesProviderFiles(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	claude, err := EnsureManagerMCP(root, "claude", "127.0.0.1:9797")
	if err != nil {
		t.Fatal(err)
	}
	if !claude.Matches || claude.URL != "http://127.0.0.1:9797/mcp" {
		t.Fatalf("claude = %#v", claude)
	}

	codex, err := EnsureManagerMCP(root, "codex", "127.0.0.1:9797")
	if err != nil {
		t.Fatal(err)
	}
	if !codex.Matches || codex.Trusted == nil || !*codex.Trusted {
		t.Fatalf("codex = %#v", codex)
	}
	trust, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(trust), "trust_level = \"trusted\"") {
		t.Fatalf("trust file = %s", trust)
	}

	gemini, err := EnsureManagerMCP(root, "gemini", ":9797")
	if err != nil {
		t.Fatal(err)
	}
	if !gemini.Matches || gemini.URL != "http://127.0.0.1:9797/mcp" {
		t.Fatalf("gemini = %#v", gemini)
	}

	if _, err := EnsureManagerMCP(root, "cursor", "127.0.0.1:9797"); err == nil || !strings.Contains(err.Error(), "cursor") {
		t.Fatalf("cursor err = %v", err)
	}
}

func TestEnsureCodexMCPKeepsOtherServers(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(root, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[mcp_servers.other]\ncommand = \"echo\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureManagerMCP(root, "codex", "127.0.0.1:8787"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "[mcp_servers.other]") || !strings.Contains(text, "[mcp_servers.manager]") {
		t.Fatalf("config = %s", text)
	}
}
