package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution/providers"
)

const OverlayFileName = "core.local.yaml"

var knownAgentIDs = []string{"claude", "codex", "cursor", "gemini"}

// Overlay is the gitignored machine overlay merged above core.config.yaml
// and below process environment.
type Overlay struct {
	ScanRoots []string
	Agents    map[string]AgentOverlay
	Manager   ManagerOverlay
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

func (o Overlay) Agent(id string) AgentOverlay {
	if o.Agents == nil {
		return AgentOverlay{}
	}
	return o.Agents[id]
}

func (o Overlay) clone() Overlay {
	out := Overlay{ScanRoots: append([]string{}, o.ScanRoots...), Agents: map[string]AgentOverlay{}, Manager: o.Manager}
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
