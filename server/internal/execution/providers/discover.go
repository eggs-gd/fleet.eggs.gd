package providers

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	AgentReady        = "ready"
	AgentNotFound     = "not_found"
	AgentNonCanonical = "non_canonical"
	AgentError        = "error"
)

const chatGPTAppCodex = "/Applications/ChatGPT.app/Contents/Resources/codex"

type DetectedExecutable struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

// AgentDiscovery is the live PATH/env/bundled/overlay resolution for one
// registered provider. Settings reads this; it does not invent extra providers.
type AgentDiscovery struct {
	ID                   string               `json:"id"`
	Name                 string               `json:"name"`
	Status               string               `json:"status"`
	ConfiguredExecutable string               `json:"configured_executable,omitempty"`
	DetectedExecutables  []DetectedExecutable `json:"detected_executables,omitempty"`
	EffectiveExecutable  string               `json:"effective_executable,omitempty"`
	EffectiveSource      string               `json:"effective_source,omitempty"`
	Executable           string               `json:"executable,omitempty"`
	Version              string               `json:"version,omitempty"`
	DiscoverySource      string               `json:"discovery_source,omitempty"`
	Canonical            bool                 `json:"canonical"`
	Expected             string               `json:"expected,omitempty"`
	OtherInstallations   []string             `json:"other_installations,omitempty"`
	Error                string               `json:"error,omitempty"`
	VisibilityClass      string               `json:"visibility_class,omitempty"`
	LiveReady            bool                 `json:"live_ready"`
	// Capabilities and SessionReuse come from the adapter, not from probing.
	Capabilities ProviderCapabilities   `json:"capabilities"`
	SessionReuse SessionReuseCapability `json:"session_reuse"`
}

// DiscoverAgents reports Claude, Codex, Cursor, and Gemini — the providers
// registered in NewRegistry().
func DiscoverAgents() []AgentDiscovery {
	rows := []AgentDiscovery{discoverClaude(), discoverCodex(), discoverCursor(), discoverGemini()}
	for i := range rows {
		if agent, ok := capabilityRegistry.Agent(rows[i].ID); ok {
			rows[i].Capabilities = agent.Capabilities()
			rows[i].SessionReuse = agent.SessionReuse()
		}
	}
	return rows
}

func discoverClaude() AgentDiscovery {
	row := AgentDiscovery{
		ID:                   "claude",
		Name:                 "Claude",
		Expected:             "claude on PATH, or newest bundled Claude.app claude-code binary",
		VisibilityClass:      "app_visible",
		ConfiguredExecutable: localExecutable("claude"),
		DetectedExecutables:  detectClaude(),
	}
	path, source, err := resolveClaudeBinaryWithSource()
	if err != nil {
		row.Status = AgentNotFound
		row.Error = err.Error()
		row.OtherInstallations = detectedPaths(row.DetectedExecutables)
		return row
	}
	return finishReady(row, path, source)
}

func detectClaude() []DetectedExecutable {
	var out []DetectedExecutable
	if onPath, err := exec.LookPath("claude"); err == nil {
		out = append(out, DetectedExecutable{Path: onPath, Source: "path"})
	}
	for _, path := range bundledClaudePaths() {
		out = appendDetected(out, path, "bundled")
	}
	return out
}

func discoverCursor() AgentDiscovery {
	row := AgentDiscovery{
		ID:                   "cursor",
		Name:                 "Cursor",
		Expected:             "cursor-agent on PATH, or newest ~/.local/share/cursor-agent/versions/*/cursor-agent",
		VisibilityClass:      "cli_visible",
		ConfiguredExecutable: localExecutable("cursor"),
		DetectedExecutables:  detectCursor(),
	}
	path, source, err := resolveCursorAgentBinaryWithSource()
	if err != nil {
		row.Status = AgentNotFound
		row.Error = err.Error()
		row.OtherInstallations = detectedPaths(row.DetectedExecutables)
		return row
	}
	return finishReady(row, path, source)
}

func detectCursor() []DetectedExecutable {
	var out []DetectedExecutable
	if onPath, err := exec.LookPath("cursor-agent"); err == nil {
		out = append(out, DetectedExecutable{Path: onPath, Source: "path"})
	}
	for _, path := range bundledCursorPaths() {
		out = appendDetected(out, path, "bundled")
	}
	return out
}

func discoverGemini() AgentDiscovery {
	row := AgentDiscovery{
		ID:                   "gemini",
		Name:                 "Gemini",
		Expected:             "agy (Antigravity CLI) on PATH, or core.local.yaml",
		VisibilityClass:      "cli_visible",
		ConfiguredExecutable: localExecutable("gemini"),
		DetectedExecutables:  detectGemini(),
	}
	path, source, err := resolveGeminiBinaryWithSource()
	if err != nil {
		row.Status = AgentNotFound
		row.Error = err.Error()
		row.OtherInstallations = detectedPaths(row.DetectedExecutables)
		return row
	}
	return finishReady(row, path, source)
}

func detectGemini() []DetectedExecutable {
	var out []DetectedExecutable
	if onPath, err := exec.LookPath("agy"); err == nil {
		out = append(out, DetectedExecutable{Path: onPath, Source: "path"})
	}
	return out
}

func discoverCodex() AgentDiscovery {
	row := AgentDiscovery{
		ID:                   "codex",
		Name:                 "Codex",
		Expected:             "standalone `codex` on PATH, FLEET_CODEX_BINARY, or core.local.yaml. ChatGPT.app's bundled binary is not canonical.",
		VisibilityClass:      "fleet_visible",
		ConfiguredExecutable: localExecutable("codex"),
		DetectedExecutables:  detectCodex(),
	}
	resolved, source, resolveErr := resolveCodexBinaryWithSource()
	return classifyCodexDiscovery(row, resolved, source, resolveErr)
}

func detectCodex() []DetectedExecutable {
	var out []DetectedExecutable
	if onPath, err := exec.LookPath("codex"); err == nil {
		out = append(out, DetectedExecutable{Path: onPath, Source: "path"})
	}
	for _, path := range existingChatGPTAppCodex() {
		out = appendDetected(out, path, "bundled")
	}
	return out
}

func classifyCodexDiscovery(row AgentDiscovery, resolved, source string, resolveErr error) AgentDiscovery {
	row.ConfiguredExecutable = firstNonEmpty(row.ConfiguredExecutable, localExecutable("codex"))
	if len(row.DetectedExecutables) == 0 {
		row.DetectedExecutables = detectCodex()
	}

	if resolveErr != nil {
		row.OtherInstallations = detectedPaths(row.DetectedExecutables)
		if len(row.DetectedExecutables) > 0 {
			row.Status = AgentNonCanonical
			row.Canonical = false
			row.Error = resolveErr.Error()
			return row
		}
		row.Status = AgentNotFound
		row.Error = resolveErr.Error()
		return row
	}

	row = finishReady(row, resolved, source)
	if isChatGPTAppCodex(resolved) {
		row.Status = AgentNonCanonical
		row.Canonical = false
		row.LiveReady = false
		row.Error = "resolved Codex executable is the ChatGPT.app bundle; Core requires the standalone CLI on PATH (CORE-151)"
	}
	return row
}

func finishReady(row AgentDiscovery, path, source string) AgentDiscovery {
	row.Executable = path
	row.EffectiveExecutable = path
	row.EffectiveSource = source
	row.DiscoverySource = source
	row.Version = probeVersion(path)
	row.Status = AgentReady
	row.Canonical = true
	row.LiveReady = true
	row.OtherInstallations = excludePath(detectedPaths(row.DetectedExecutables), path)
	return row
}

func appendDetected(out []DetectedExecutable, path, source string) []DetectedExecutable {
	path = filepath.Clean(path)
	for _, existing := range out {
		if filepath.Clean(existing.Path) == path {
			return out
		}
	}
	return append(out, DetectedExecutable{Path: path, Source: source})
}

func detectedPaths(detected []DetectedExecutable) []string {
	out := make([]string, 0, len(detected))
	for _, item := range detected {
		out = append(out, item.Path)
	}
	return out
}

func isChatGPTAppCodex(path string) bool {
	cleaned := filepath.ToSlash(filepath.Clean(path))
	return strings.Contains(cleaned, "/ChatGPT.app/")
}

func existingChatGPTAppCodex() []string {
	info, err := os.Stat(chatGPTAppCodex)
	if err != nil || info.IsDir() {
		return nil
	}
	return []string{chatGPTAppCodex}
}

func excludePath(paths []string, skip string) []string {
	skip = filepath.Clean(skip)
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if filepath.Clean(path) == skip {
			continue
		}
		out = append(out, path)
	}
	return out
}

// ProbeVersion runs `binary --version` with a short timeout. Empty means the
// process failed or printed nothing — wrappers may still be valid launchers.
func ProbeVersion(binary string) string {
	return probeVersion(binary)
}

func probeVersion(binary string) string {
	if strings.TrimSpace(binary) == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(bytes.SplitN(bytes.TrimSpace(out), []byte("\n"), 2)[0]))
	if len(line) > 160 {
		line = strings.TrimSpace(line[:160])
	}
	return line
}
