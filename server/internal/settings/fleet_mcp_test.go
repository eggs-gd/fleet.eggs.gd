package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureManagerMCPWritesProviderFiles(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	claude, err := EnsureManagerMCP(root, "claude", "127.0.0.1:9797", "")
	if err != nil {
		t.Fatal(err)
	}
	if !claude.Matches || claude.URL != "http://127.0.0.1:9797/mcp" {
		t.Fatalf("claude = %#v", claude)
	}

	codex, err := EnsureManagerMCP(root, "codex", "127.0.0.1:9797", "")
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

	gemini, err := EnsureManagerMCP(root, "gemini", ":9797", "")
	if err != nil {
		t.Fatal(err)
	}
	if !gemini.Matches || gemini.URL != "http://127.0.0.1:9797/mcp" {
		t.Fatalf("gemini = %#v", gemini)
	}

	cursor, err := EnsureManagerMCP(root, "cursor", "127.0.0.1:9797", "")
	if err != nil {
		t.Fatal(err)
	}
	if !cursor.Matches || cursor.URL != "http://127.0.0.1:9797/mcp" {
		t.Fatalf("cursor = %#v", cursor)
	}
	for _, rel := range []string{".mcp.json", ".codex/config.toml", ".agents/mcp_config.json", ".cursor/mcp.json"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
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
	if _, err := EnsureManagerMCP(root, "codex", "127.0.0.1:8787", "launch-token"); err != nil {
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
	if !strings.Contains(text, "token=launch-token") {
		t.Fatalf("config = %s", text)
	}
}

func TestClaudeManagerMCPOmitsEmptyFieldsAndKeepsOtherKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := `{"other":{"x":1},"mcpServers":{"mine":{"command":"tool","args":["a"]}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeClaudeManagerMCP(path, "http://127.0.0.1:1/mcp"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var got map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["other"]; !ok {
		t.Fatalf("unknown top-level key lost: %s", data)
	}
	var servers map[string]map[string]any
	if err := json.Unmarshal(got["mcpServers"], &servers); err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["mine"]; !ok {
		t.Fatalf("existing server lost: %s", data)
	}
	manager := servers["manager"]
	if manager["url"] != "http://127.0.0.1:1/mcp" || manager["type"] != "http" {
		t.Fatalf("manager = %#v", manager)
	}
	if _, ok := manager["command"]; ok {
		t.Fatalf("empty command written: %s", data)
	}
	if _, ok := manager["args"]; ok {
		t.Fatalf("null args written: %s", data)
	}
	if _, err := os.Stat(path + ".fleet-bak"); err == nil {
		t.Fatal("a backup file was written")
	}
}

func TestManagerMCPRefusesInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	for name, write := range map[string]func(string, string) error{
		".mcp.json":   writeClaudeManagerMCP,
		"cursor.json": writeCursorManagerMCP,
		"gemini.json": writeGeminiManagerMCP,
	} {
		path := filepath.Join(dir, name)
		broken := "{ // comment\n \"mcpServers\": {"
		if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := write(path, "http://127.0.0.1:1/mcp"); err == nil {
			t.Fatalf("%s: expected an error for invalid JSON", name)
		}
		if data, _ := os.ReadFile(path); string(data) != broken {
			t.Fatalf("%s was modified: %q", name, data)
		}
	}
}
