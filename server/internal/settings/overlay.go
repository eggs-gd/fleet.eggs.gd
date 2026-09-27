package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution/providers"
)

const OverlayFileName = "core.local.yaml"

var knownAgentIDs = []string{"claude", "codex", "cursor", "gemini"}

// Overlay is the gitignored machine overlay merged above core.config.yaml
// and below process environment.
type Overlay struct {
	ScanRoots      []string
	SessionTimeout string
	// Launch is "live", "dry-run", or empty for the default (dry-run).
	Launch  string
	Agents  map[string]AgentOverlay
	Manager ManagerOverlay
}

type AgentOverlay struct {
	Enabled             *bool
	Executable          string
	RoutingInstructions string
}

// ManagerOverlay remembers the provider session that is the Manager.
// Conversation stays in that provider's app. Fleet does not send chat into it.
type ManagerOverlay struct {
	Agent     string
	ThreadID  string
	Workspace string
	BoundAt   string
}

func OverlayPath(root string) string {
	return filepath.Join(root, OverlayFileName)
}

func LoadOverlay(root string) (Overlay, error) {
	path := OverlayPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Overlay{Agents: map[string]AgentOverlay{}}, nil
		}
		return Overlay{}, fmt.Errorf("read %s: %w", OverlayFileName, err)
	}
	overlay, err := ParseOverlay(data)
	if err != nil {
		return Overlay{}, fmt.Errorf("parse %s: %w", OverlayFileName, err)
	}
	return overlay, nil
}

func SaveOverlay(root string, overlay Overlay) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create overlay dir: %w", err)
	}
	data := EncodeOverlay(overlay)
	tmp, err := os.CreateTemp(root, ".core.local.yaml.*")
	if err != nil {
		return fmt.Errorf("create overlay temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write overlay temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync overlay temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close overlay temp: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("chmod overlay temp: %w", err)
	}
	if err := os.Rename(tmpName, OverlayPath(root)); err != nil {
		return fmt.Errorf("replace %s: %w", OverlayFileName, err)
	}
	return nil
}

func ApplyOverlay(overlay Overlay) {
	providers.ApplyLocalAgents(overlay.localAgents())
}

func (o Overlay) localAgents() map[string]providers.LocalAgent {
	out := make(map[string]providers.LocalAgent, len(o.Agents))
	for id, agent := range o.Agents {
		out[id] = providers.LocalAgent{
			Enabled:    agent.Enabled,
			Executable: agent.Executable,
		}
	}
	return out
}

// Launch modes. A serve that plans launches without starting agents is the
// default: an agent edits files without asking, so starting them is opt-in.
const (
	LaunchLive   = "live"
	LaunchDryRun = "dry-run"
)

// ResolveLaunchMode decides whether serve only plans launches. --live and
// --dry-run win over core.local.yaml, which wins over the default (dry-run). It
// returns dryRun.
func ResolveLaunchMode(flagLive, flagDryRun bool, overlay Overlay) (dryRun bool, err error) {
	switch {
	case flagLive && flagDryRun:
		return false, fmt.Errorf("--live and --dry-run cannot be used together")
	case flagLive:
		return false, nil
	case flagDryRun:
		return true, nil
	}
	switch strings.TrimSpace(overlay.Launch) {
	case "", LaunchDryRun:
		return true, nil
	case LaunchLive:
		return false, nil
	default:
		return false, fmt.Errorf("%s launch must be %q or %q, got %q", OverlayFileName, LaunchLive, LaunchDryRun, overlay.Launch)
	}
}

const DefaultSessionTimeout = 10 * time.Minute

// ResolveServeTimeout uses an explicit --session-timeout when the flag was
// passed. Otherwise it uses core.local.yaml, then the default. The overlay
// value applies on the next serve; a running process keeps the timeout it
// started with.
func ResolveServeTimeout(flagSet bool, flagValue time.Duration, overlay Overlay) (time.Duration, error) {
	if flagSet {
		if flagValue <= 0 {
			return 0, fmt.Errorf("session-timeout must be positive")
		}
		return flagValue, nil
	}
	raw := strings.TrimSpace(overlay.SessionTimeout)
	if raw == "" {
		return DefaultSessionTimeout, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s sessionTimeout: %w", OverlayFileName, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s sessionTimeout must be positive", OverlayFileName)
	}
	return d, nil
}

func (o Overlay) Agent(id string) AgentOverlay {
	if o.Agents == nil {
		return AgentOverlay{}
	}
	return o.Agents[id]
}

func (o Overlay) clone() Overlay {
	out := Overlay{ScanRoots: append([]string{}, o.ScanRoots...), SessionTimeout: o.SessionTimeout, Launch: o.Launch, Agents: map[string]AgentOverlay{}, Manager: o.Manager}
	for id, agent := range o.Agents {
		out.Agents[id] = agent
	}
	return out
}

func isKnownAgent(id string) bool {
	for _, known := range knownAgentIDs {
		if known == id {
			return true
		}
	}
	return false
}

// EffectiveScanRoots returns the configured project directories.
// An empty list means nothing is scanned. There is no default directory.
func EffectiveScanRoots(overlay Overlay) (values []string, source string) {
	values = normalizeScanRoots(overlay.ScanRoots)
	if len(values) == 0 {
		return nil, ""
	}
	return values, SourceLocal
}

func normalizeScanRoots(roots []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, root)
	}
	return out
}
