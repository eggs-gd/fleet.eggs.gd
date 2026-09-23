package providers

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Claude struct{}

func (Claude) Name() string {
	return "claude"
}

func (agent Claude) Plan(task TaskContext) (Plan, error) {
	prompt := BuildPrompt(task)
	binary, err := resolveClaudeBinary()
	if err != nil {
		return Plan{
			Agent:           agent.Name(),
			Backend:         "unsupported",
			WorkingDir:      task.WorkingDir,
			Prompt:          prompt,
			VisibilityClass: "unknown",
			LiveReady:       false,
			LiveNotes:       "Claude CLI binary could not be resolved: " + err.Error() + " PATH=" + pathEnvHint(),
		}, nil
	}

	name := firstNonEmpty(task.Ref, task.ID, "core-task")
	return Plan{
		Agent:           agent.Name(),
		Backend:         BackendBackgroundRemote,
		Command:         []string{binary, "--bg", "--remote-control", "--name", name, "--permission-mode", "acceptEdits", prompt},
		WorkingDir:      task.WorkingDir,
		Prompt:          prompt,
		VisibilityClass: "app_visible",
		LiveReady:       true,
		LiveNotes:       "Claude launches as a `claude --bg --remote-control` background agent with `--permission-mode acceptEdits` so every daemon session is app_visible and operator-controllable. File edits are auto-accepted; Bash/git actions are not, so no-auto-commit/push/PR from the launch prompt still holds. Core does not inject stdin/PTY input; it supervises completion via `claude agents --json` polling and exposes the printed remote-control URL so a human can continue the session from the phone. The task `launch.mode` value cannot select a headless Claude print path.",
	}, nil
}

// resolveClaudeBinary finds a runnable `claude` executable.
//
// On this class of machine `claude` is frequently not on the daemon's PATH
// even when it works fine in an interactive terminal (it can be a versioned
// binary bundled inside the Claude.app support directory rather than a
// PATH-installed CLI). Fall back to the newest bundled build before giving up.
func resolveClaudeBinary() (string, error) {
	path, _, err := resolveClaudeBinaryWithSource()
	return path, err
}

func resolveClaudeBinaryWithSource() (string, string, error) {
	if overlay := localExecutable("claude"); overlay != "" {
		if err := requireRunnable(overlay, "configured Claude executable"); err != nil {
			return "", "core.local.yaml", err
		}
		return overlay, "core.local.yaml", nil
	}
	if path, err := exec.LookPath("claude"); err == nil {
		return path, "path", nil
	}
	candidate, err := newestBundledClaude()
	if err != nil {
		return "", "", err
	}
	return candidate, "bundled", nil
}

func newestBundledClaude() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("claude is not on PATH and home directory could not be resolved")
	}
	matches, err := filepath.Glob(filepath.Join(home, "Library", "Application Support", "Claude", "claude-code", "*", "claude.app", "Contents", "MacOS", "claude"))
	if err != nil || len(matches) == 0 {
		return "", errors.New("claude is not on PATH and no bundled Claude.app claude-code binary was found")
	}
	sort.Strings(matches)
	candidate := matches[len(matches)-1]
	if info, err := os.Stat(candidate); err != nil || info.IsDir() {
		return "", errors.New("claude is not on PATH and the newest bundled binary candidate is not runnable")
	}
	return candidate, nil
}

func bundledClaudePaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(home, "Library", "Application Support", "Claude", "claude-code", "*", "claude.app", "Contents", "MacOS", "claude"))
	if err != nil {
		return nil
	}
	sort.Strings(matches)
	return matches
}

type Codex struct{}

func (Codex) Name() string {
	return "codex"
}

func (agent Codex) Plan(task TaskContext) (Plan, error) {
	prompt := BuildPrompt(task)
	binary, err := resolveCodexBinary()
	if err != nil {
		return Plan{
			Agent:           agent.Name(),
			Backend:         "unsupported",
			WorkingDir:      task.WorkingDir,
			Prompt:          prompt,
			VisibilityClass: "unknown",
			LiveReady:       false,
			LiveNotes:       codexMissingCLIDiagnostic() + " (" + err.Error() + ") PATH=" + pathEnvHint(),
		}, nil
	}

	notes := "Resolved Codex binary: " + binary + ". Codex `app-server --stdio` is core_visible (CORE-130): Core owns a JSON-RPC thread/turn control plane with dashboard continue/interrupt/cancel, names threads for Codex Remote, and normal daemon auto-launch is allowed. Sessions appear in Codex Remote on paired clients (not necessarily ChatGPT desktop sidebar history) and lack a Claude-style deep-link URL. `launch.mode: allow_core_visible` is a legacy no-op. `codex exec` remains a local fallback/smoke path, not the primary Core launcher contract."
	if strings.EqualFold(strings.TrimSpace(task.LaunchMode), "allow_core_visible") {
		notes = "Legacy launch.mode=allow_core_visible is accepted but no longer required. " + notes
	}

	return Plan{
		Agent:           agent.Name(),
		Backend:         BackendCodexAppServer,
		Command:         []string{binary, "app-server", "--stdio"},
		WorkingDir:      task.WorkingDir,
		Prompt:          prompt,
		VisibilityClass: "core_visible",
		LiveReady:       true,
		LiveNotes:       notes,
	}, nil
}

// codexMissingCLIDiagnostic is the operator-facing setup message when the
// daemon cannot resolve a standalone Codex CLI (CORE-151).
func codexMissingCLIDiagnostic() string {
	return "Codex CLI is not available on the daemon PATH. Install the standalone Codex CLI or configure the Codex binary path, then retry this task."
}

// resolveCodexBinary finds the official standalone Codex CLI for daemon
// launches (CORE-151). Resolution order:
//  1. CORE_CODEX_BINARY — process env wins over overlay;
//  2. core.local.yaml agents.codex.executable;
//  3. exec.LookPath("codex") — shell-visible standalone install.
//
// ChatGPT.app's bundled binary is intentionally not a fallback: silent use of
// /Applications/ChatGPT.app/Contents/Resources/codex hides daemon PATH setup
// problems (for example a missing ~/.local/bin entry).
func resolveCodexBinary() (string, error) {
	path, _, err := resolveCodexBinaryWithSource()
	return path, err
}

func resolveCodexBinaryWithSource() (string, string, error) {
	if configured := strings.TrimSpace(os.Getenv("CORE_CODEX_BINARY")); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", "env", errors.New("CORE_CODEX_BINARY is set but is not an absolute path")
		}
		if err := requireRunnable(configured, "CORE_CODEX_BINARY"); err != nil {
			return "", "env", err
		}
		return configured, "env", nil
	}
	if overlay := localExecutable("codex"); overlay != "" {
		if err := requireRunnable(overlay, "configured Codex executable"); err != nil {
			return "", "core.local.yaml", err
		}
		return overlay, "core.local.yaml", nil
	}
	if path, err := exec.LookPath("codex"); err == nil {
		return path, "path", nil
	}
	return "", "", errors.New("codex is not on PATH")
}

func requireRunnable(path, label string) error {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return errors.New(label + " is set but is not a runnable file")
	}
	if info.Mode()&0o111 == 0 {
		return errors.New(label + " is set but is not executable")
	}
	return nil
}

type Cursor struct{}

func (Cursor) Name() string {
	return "cursor"
}

func (agent Cursor) Plan(task TaskContext) (Plan, error) {
	prompt := BuildPrompt(task)
	binary, err := resolveCursorAgentBinary()
	if err != nil {
		return Plan{
			Agent:           agent.Name(),
			Backend:         "unsupported",
			WorkingDir:      task.WorkingDir,
			Prompt:          prompt,
			VisibilityClass: "unknown",
			LiveReady:       false,
			LiveNotes:       "Cursor Agent CLI binary could not be resolved: " + err.Error() + " PATH=" + pathEnvHint(),
		}, nil
	}

	return Plan{
		Agent:           agent.Name(),
		Backend:         BackendCursorVisible,
		Command:         cursorAgentVisibleCommand(binary, task.WorkingDir, prompt),
		WorkingDir:      task.WorkingDir,
		Prompt:          prompt,
		VisibilityClass: "cli_visible",
		LiveReady:       true,
		LiveNotes:       "Cursor uses the normal interactive `cursor-agent --trust --workspace <path> <prompt>` CLI path, never `--print`, for daemon-launched tasks (CORE-79). Visibility class is cli_visible, not phone/app_visible: Core creates a Cursor chat with `cursor-agent create-chat`, launches with `--resume <chat_id>`, stores the chat id on the runtime session, captures stdout/stderr in `_registry/sessions/<claim_id>.log`, and exposes `cursor-agent --resume <chat_id> --workspace <path>` for operator inspection/resume. Completion detection is process-exit based. Assign Cursor only as an explicit opt-in worker; do not treat it as Claude-style phone remote-control.",
	}, nil
}

func cursorAgentVisibleCommand(binary string, workingDir string, prompt string) []string {
	return []string{binary, "--trust", "--workspace", workingDir, prompt}
}

func cursorAgentPrintDiagnosticCommand(binary string, workingDir string, prompt string) []string {
	return []string{binary, "--print", "--output-format", "text", "--trust", "--workspace", workingDir, prompt}
}

func resolveCursorAgentBinary() (string, error) {
	path, _, err := resolveCursorAgentBinaryWithSource()
	return path, err
}

func resolveCursorAgentBinaryWithSource() (string, string, error) {
	if overlay := localExecutable("cursor"); overlay != "" {
		if err := requireRunnable(overlay, "configured Cursor executable"); err != nil {
			return "", "core.local.yaml", err
		}
		return overlay, "core.local.yaml", nil
	}
	if path, err := exec.LookPath("cursor-agent"); err == nil {
		return path, "path", nil
	}
	candidate, err := newestBundledCursor()
	if err != nil {
		return "", "", err
	}
	return candidate, "bundled", nil
}

func newestBundledCursor() (string, error) {
	if _, err := os.UserHomeDir(); err != nil {
		return "", errors.New("cursor-agent is not on PATH and home directory could not be resolved")
	}
	matches := bundledCursorPaths()
	if len(matches) == 0 {
		return "", errors.New("cursor-agent is not on PATH and no ~/.local/share/cursor-agent versioned binary was found")
	}
	candidate := matches[len(matches)-1]
	if info, err := os.Stat(candidate); err != nil || info.IsDir() {
		return "", errors.New("cursor-agent is not on PATH and the newest versioned binary candidate is not runnable")
	}
	return candidate, nil
}

func bundledCursorPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(home, ".local", "share", "cursor-agent", "versions", "*", "cursor-agent"))
	if err != nil {
		return nil
	}
	sort.Strings(matches)
	return matches
}

type Gemini struct{}

func (Gemini) Name() string {
	return "gemini"
}

func (agent Gemini) Plan(task TaskContext) (Plan, error) {
	prompt := BuildPrompt(task)
	binary, err := resolveGeminiBinary()
	if err != nil {
		return Plan{
			Agent:           agent.Name(),
			Backend:         "unsupported",
			WorkingDir:      task.WorkingDir,
			Prompt:          prompt,
			VisibilityClass: "unknown",
			LiveReady:       false,
			LiveNotes:       "Antigravity CLI (`agy`) binary could not be resolved: " + err.Error() + " PATH=" + pathEnvHint(),
		}, nil
	}

	return Plan{
		Agent:           agent.Name(),
		Backend:         BackendGeminiHeadless,
		Command:         []string{binary, "--print", prompt, "--output-format", "stream-json"},
		WorkingDir:      task.WorkingDir,
		Prompt:          prompt,
		VisibilityClass: "headless",
		LiveReady:       true,
		LiveNotes:       "Gemini launches through Google's Antigravity CLI (`agy --print <prompt> --output-format stream-json`): a synchronous headless call that blocks until the turn's terminal `result` event, not a background/daemon process like Claude/Codex. Core registers the working directory as an antigravity project on first use (`agy --new-project`, discovered afterward from `~/.gemini/config/projects/*.json` since the CLI does not print the id) so AGENTS.md/GEMINI.md and project-scoped `.agents/mcp_config.json` load the same way an interactive session would; every later launch in that directory reuses `--project <id>`. Resume is `--conversation <id>`, a fresh process invocation, not stdin into a live process.",
	}, nil
}

// resolveGeminiBinary finds a runnable Antigravity CLI. The agent id is
// "gemini" (matching the product this session drives and the existing
// frontend agent lists), but the executable is `agy`.
func resolveGeminiBinary() (string, error) {
	path, _, err := resolveGeminiBinaryWithSource()
	return path, err
}

func resolveGeminiBinaryWithSource() (string, string, error) {
	if overlay := localExecutable("gemini"); overlay != "" {
		if err := requireRunnable(overlay, "configured Gemini executable"); err != nil {
			return "", "core.local.yaml", err
		}
		return overlay, "core.local.yaml", nil
	}
	if path, err := exec.LookPath("agy"); err == nil {
		return path, "path", nil
	}
	return "", "", errors.New("agy (Antigravity CLI) is not on PATH")
}

func pathEnvHint() string {
	path := strings.TrimSpace(os.Getenv("PATH"))
	if path == "" {
		return "(empty)"
	}
	const limit = 180
	if len(path) <= limit {
		return path
	}
	return path[:limit] + "..."
}
