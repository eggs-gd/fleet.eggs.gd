package appconfig

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Resolution sources for ResolveDataRoot, exposed so callers can log/report
// how the effective Data root was picked.
const (
	SourceFlag    = "flag"
	SourceConfig  = "config"
	SourceDefault = "default"
)

// ResolveDataRoot picks the effective Data root: an explicit --root flag
// wins, then the saved pointer config, then DefaultDataRoot. The default is
// persisted on first use so it becomes visible (and editable) in Settings
// instead of being an invisible in-memory fallback.
func ResolveDataRoot(flagRoot string) (root, source string, err error) {
	if strings.TrimSpace(flagRoot) != "" {
		abs, err := filepath.Abs(flagRoot)
		if err != nil {
			return "", "", fmt.Errorf("resolve --root: %w", err)
		}
		return abs, SourceFlag, nil
	}

	cfg, err := Load()
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(cfg.DataRoot) != "" {
		abs, err := ExpandHome(cfg.DataRoot)
		if err != nil {
			return "", "", fmt.Errorf("resolve configured data root: %w", err)
		}
		return abs, SourceConfig, nil
	}

	abs, err := ExpandHome(DefaultDataRoot)
	if err != nil {
		return "", "", err
	}
	if err := Save(Config{DataRoot: abs}); err != nil {
		return "", "", fmt.Errorf("save default data root: %w", err)
	}
	return abs, SourceDefault, nil
}
