package settings

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type mcpFile struct {
	Servers map[string]mcpEntry `json:"mcpServers"`
}

type mcpEntry struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	URL     string   `json:"url"`
}

func loadMCP(root string) Integrations {
	path := filepath.Join(root, ".mcp.json")
	out := Integrations{
		SourceFile: path,
		SourceRole: "Cursor project MCP config (.mcp.json). Core daemon does not currently inject these servers into Claude/Codex/Cursor launches.",
		Notes: []string{
			"Agents and MCP are separate. This list is file-configured, not a Core-owned integration registry.",
			"Reachable is unknown: Phase 2 does not probe MCP endpoints or spend provider quota.",
			"No Install action: Core has no deterministic MCP installer.",
		},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			out.Notes = append(out.Notes, ".mcp.json is not present under the Core root.")
			return out
		}
		out.Notes = append(out.Notes, "failed to read .mcp.json: "+err.Error())
		return out
	}
	var file mcpFile
	if err := json.Unmarshal(data, &file); err != nil {
		out.Notes = append(out.Notes, "failed to parse .mcp.json: "+err.Error())
		return out
	}
	for name, entry := range file.Servers {
		server := MCPServer{
			Name:         name,
			Transport:    firstNonEmpty(entry.Type, "stdio"),
			Command:      strings.TrimSpace(entry.Command),
			Configured:   true,
			Reachable:    "unknown",
			InstallKnown: false,
		}
		server.Detected, server.Expected, server.Error = detectMCP(entry)
		out.MCP = append(out.MCP, server)
	}
	sort.Slice(out.MCP, func(i, j int) bool { return out.MCP[i].Name < out.MCP[j].Name })
	return out
}

func detectMCP(entry mcpEntry) (detected bool, expected string, errText string) {
	cmd := strings.TrimSpace(entry.Command)
	if cmd == "" && strings.TrimSpace(entry.URL) != "" {
		return false, entry.URL, "HTTP MCP URL is configured; Phase 2 does not probe it"
	}
	if cmd == "npx" {
		_, err := exec.LookPath("npx")
		return err == nil, "npx on PATH", lookErr(err, "npx")
	}
	if cmd == "gopls" {
		if path, err := exec.LookPath("gopls"); err == nil {
			return true, path, ""
		}
		// The daemon's PATH is frequently narrower than an interactive
		// shell's (same class of problem documented on resolveClaudeBinary):
		// `go install`'s default GOBIN, ~/go/bin, is often missing from it
		// even though gopls resolves fine at a terminal. Fall back to that
		// well-known default before reporting not-detected.
		if home, err := os.UserHomeDir(); err == nil {
			fallback := filepath.Join(home, "go", "bin", "gopls")
			if info, statErr := os.Stat(fallback); statErr == nil && !info.IsDir() {
				return true, fallback, ""
			}
		}
		return false, "gopls on PATH", "gopls not found on PATH"
	}
	if cmd == "sh" && containsDesignPatterns(entry.Args) {
		nodePath := expandAgentsToolsPath("$AGENTS_TOOLS_DIR/design_patterns_mcp/dist/mcp-server.js")
		expected = nodePath
		_, err := os.Stat(nodePath)
		if err != nil {
			return false, expected, "design-patterns MCP server is not installed at " + nodePath
		}
		return true, expected, ""
	}
	if cmd == "" {
		return false, "", "no command"
	}
	path, err := exec.LookPath(cmd)
	if err != nil {
		return false, cmd + " on PATH", lookErr(err, cmd)
	}
	return true, path, ""
}

func containsDesignPatterns(args []string) bool {
	for _, arg := range args {
		if strings.Contains(arg, "design_patterns_mcp") {
			return true
		}
	}
	return false
}

func expandAgentsToolsPath(pattern string) string {
	home, _ := os.UserHomeDir()
	tools := strings.TrimSpace(os.Getenv("AGENTS_TOOLS_DIR"))
	if tools == "" {
		tools = filepath.Join(home, ".agents")
	}
	out := strings.ReplaceAll(pattern, "$AGENTS_TOOLS_DIR", tools)
	out = strings.ReplaceAll(out, "$HOME", home)
	return os.ExpandEnv(out)
}

func lookErr(err error, name string) string {
	if err == nil {
		return ""
	}
	return name + " not found on PATH"
}
