package settings

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CodexTrustWarning is the one sentence shown when Fleet trusts a data root
// for Codex. Trust covers hooks and exec policy, not only MCP.
const CodexTrustWarning = "Trusting this folder also allows Codex hooks and exec policy, not only the manager MCP server."

// FleetMCPStatus says whether the Manager provider's native config contains
// the Fleet MCP server and whether its URL matches this process.
type FleetMCPStatus struct {
	Provider    string `json:"provider,omitempty"`
	Path        string `json:"path,omitempty"`
	URL         string `json:"url,omitempty"`
	ExpectedURL string `json:"expected_url"`
	Written     bool   `json:"written"`
	Matches     bool   `json:"matches"`
	Trusted     *bool  `json:"trusted,omitempty"`
}

// ManagerMCPURL is the HTTP URL providers load for this process.
func ManagerMCPURL(addr, token string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = "127.0.0.1:8787"
	}
	base := "http://" + strings.TrimRight(addr, "/") + "/mcp"
	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		switch host {
		case "", "0.0.0.0", "::", "[::]":
			host = "127.0.0.1"
		}
		base = "http://" + net.JoinHostPort(host, port) + "/mcp"
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return base
	}
	return base + "?token=" + url.QueryEscape(token)
}

// EnsureManagerMCP writes the Fleet MCP server into every provider file at
// root before a Manager session is created or adopted. Codex trust still
// runs only when Codex is the selected Manager.
func EnsureManagerMCP(root, agent, addr, token string) (FleetMCPStatus, error) {
	agent = strings.TrimSpace(agent)
	switch agent {
	case "claude", "codex", "gemini", "cursor":
	default:
		return FleetMCPStatus{}, fmt.Errorf("unknown manager agent %q", agent)
	}
	url := ManagerMCPURL(addr, token)
	abs, err := filepath.Abs(root)
	if err != nil {
		return FleetMCPStatus{}, err
	}
	if err := writeClaudeManagerMCP(filepath.Join(abs, ".mcp.json"), url); err != nil {
		return FleetMCPStatus{}, err
	}
	if err := writeCodexManagerMCP(filepath.Join(abs, ".codex", "config.toml"), url); err != nil {
		return FleetMCPStatus{}, err
	}
	if err := writeGeminiManagerMCP(filepath.Join(abs, ".agents", "mcp_config.json"), url); err != nil {
		return FleetMCPStatus{}, err
	}
	if err := writeCursorManagerMCP(filepath.Join(abs, ".cursor", "mcp.json"), url); err != nil {
		return FleetMCPStatus{}, err
	}
	if agent == "codex" {
		if err := trustCodexProject(abs); err != nil {
			return FleetMCPStatus{}, err
		}
	}
	return InspectManagerMCP(abs, agent, addr, token), nil
}

// InspectManagerMCP reads the provider file without writing it.
func InspectManagerMCP(root, agent, addr, token string) FleetMCPStatus {
	agent = strings.TrimSpace(agent)
	expected := ManagerMCPURL(addr, token)
	status := FleetMCPStatus{Provider: agent, ExpectedURL: expected}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	switch agent {
	case "claude":
		status.Path = filepath.Join(abs, ".mcp.json")
		status.URL = claudeManagerURL(status.Path)
	case "codex":
		status.Path = filepath.Join(abs, ".codex", "config.toml")
		status.URL = codexManagerURL(status.Path)
		trusted := codexProjectTrusted(abs)
		status.Trusted = &trusted
	case "gemini":
		status.Path = filepath.Join(abs, ".agents", "mcp_config.json")
		status.URL = geminiManagerURL(status.Path)
	case "cursor":
		status.Path = filepath.Join(abs, ".cursor", "mcp.json")
		status.URL = cursorManagerURL(status.Path)
	default:
		return status
	}
	status.Written = status.URL != ""
	status.Matches = status.URL == expected
	if status.Trusted != nil {
		status.Matches = status.Matches && *status.Trusted
	}
	return status
}

func writeClaudeManagerMCP(path, url string) error {
	return upsertMCPServer(path, "manager", struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}{Type: "http", URL: url})
}

func writeCursorManagerMCP(path, url string) error {
	return upsertMCPServer(path, "manager", struct {
		URL string `json:"url"`
	}{URL: url})
}

func writeGeminiManagerMCP(path, url string) error {
	return upsertMCPServer(path, "manager", struct {
		HTTPURL string `json:"httpUrl"`
	}{HTTPURL: url})
}

// upsertMCPServer sets mcpServers[name] in a provider JSON file. It keeps every
// other key and server as it was, and refuses to touch a file it cannot parse
// instead of replacing it.
func upsertMCPServer(path, name string, entry any) error {
	top := map[string]json.RawMessage{}
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if len(strings.TrimSpace(string(existing))) > 0 {
			if err := json.Unmarshal(existing, &top); err != nil {
				return fmt.Errorf("%s is not valid JSON, refusing to overwrite it: %w", path, err)
			}
		}
	case !os.IsNotExist(err):
		return err
	}
	servers := map[string]json.RawMessage{}
	if raw, ok := top["mcpServers"]; ok && string(raw) != "null" {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return fmt.Errorf("%s: mcpServers is not an object, refusing to overwrite it: %w", path, err)
		}
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	servers[name] = encoded
	top["mcpServers"], err = json.Marshal(servers)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := backupOnce(path, existing); err != nil {
		return err
	}
	return writePrivateFile(path, data)
}

// backupOnce keeps the first pre-Fleet copy of a provider file next to it, so a
// user's own configuration can always be recovered.
func backupOnce(path string, existing []byte) error {
	if len(strings.TrimSpace(string(existing))) == 0 {
		return nil
	}
	backup := path + ".fleet-bak"
	if _, err := os.Stat(backup); err == nil {
		return nil
	}
	return os.WriteFile(backup, existing, 0o600)
}

func writeCodexManagerMCP(path, url string) error {
	var existing string
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}
	body := "url = " + strconv.Quote(url) + "\n"
	next := upsertTOMLSection(existing, "[mcp_servers.manager]", body)
	if next == existing {
		return nil
	}
	if err := backupOnce(path, []byte(existing)); err != nil {
		return err
	}
	return writePrivateFile(path, []byte(next))
}

func trustCodexProject(absRoot string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("codex trust: %w", err)
	}
	path := filepath.Join(home, ".codex", "config.toml")
	var existing string
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}
	header := `[projects.` + strconv.Quote(absRoot) + `]`
	if tomlHasHeader(existing, header) {
		return nil
	}
	next := strings.TrimRight(existing, "\n")
	if next != "" {
		next += "\n\n"
	}
	next += header + "\ntrust_level = \"trusted\"\n"
	if err := backupOnce(path, []byte(existing)); err != nil {
		return err
	}
	return writePrivateFile(path, []byte(next))
}

func claudeManagerURL(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var file mcpFile
	if json.Unmarshal(data, &file) != nil || file.Servers == nil {
		return ""
	}
	return strings.TrimSpace(file.Servers["manager"].URL)
}

func cursorManagerURL(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var file struct {
		Servers map[string]struct {
			URL string `json:"url"`
		} `json:"mcpServers"`
	}
	if json.Unmarshal(data, &file) != nil || file.Servers == nil {
		return ""
	}
	return strings.TrimSpace(file.Servers["manager"].URL)
}

func geminiManagerURL(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var file struct {
		Servers map[string]struct {
			HTTPURL string `json:"httpUrl"`
		} `json:"mcpServers"`
	}
	if json.Unmarshal(data, &file) != nil || file.Servers == nil {
		return ""
	}
	return strings.TrimSpace(file.Servers["manager"].HTTPURL)
}

func codexManagerURL(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return tomlSectionValue(string(data), "[mcp_servers.manager]", "url")
}

func codexProjectTrusted(absRoot string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		return false
	}
	header := `[projects.` + strconv.Quote(absRoot) + `]`
	if !tomlHasHeader(string(data), header) {
		return false
	}
	return tomlSectionValue(string(data), header, "trust_level") == "trusted"
}

func upsertTOMLSection(content, header, body string) string {
	var out []string
	skipping := false
	for _, line := range strings.Split(content, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			skipping = trim == header
		}
		if skipping {
			continue
		}
		out = append(out, line)
	}
	text := strings.TrimRight(strings.Join(out, "\n"), "\n")
	if text != "" {
		text += "\n\n"
	}
	text += header + "\n" + body
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text
}

func tomlHasHeader(content, header string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == header {
			return true
		}
	}
	return false
}

func tomlSectionValue(content, header, key string) string {
	in := false
	for _, line := range strings.Split(content, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			if in {
				return ""
			}
			in = trim == header
			continue
		}
		if !in || trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		name, rest, ok := strings.Cut(trim, "=")
		if !ok || strings.TrimSpace(name) != key {
			continue
		}
		return unquoteTOML(strings.TrimSpace(rest))
	}
	return ""
}

func unquoteTOML(value string) string {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
	}
	return strings.Trim(value, `"'`)
}

func writePrivateFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
