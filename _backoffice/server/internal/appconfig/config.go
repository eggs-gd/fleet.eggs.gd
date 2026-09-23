// Package appconfig resolves where the Data root lives on this machine.
//
// ~/.fleet is App's one fixed, sticky home directory: it holds app.json
// (this package) and, in the future, other App-level config/secrets
// (provider keys, etc.) alongside a workspace/ subdirectory that is the
// default Data root. ~/.fleet itself never moves — its path is how App
// bootstraps without any config to read yet. workspace/ is just the
// default *content* of dataRoot; the operator can repoint dataRoot
// anywhere (see MoveDataRoot), same as any other configured value.
//
// This is deliberately separate from internal/settings' Overlay
// (core.local.yaml): that file lives inside the Data root it describes, so
// it cannot also be where App learns the Data root's own location.
package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultDataRoot is used the first time App runs anywhere and nothing else
// says where Data should live: workspace/ under App's fixed ~/.fleet home.
const DefaultDataRoot = "~/.fleet/workspace"

// Config is the small App-level pointer file, independent of any Data root.
type Config struct {
	DataRoot string `json:"dataRoot"`
}

// PointerPath returns ~/.fleet/app.json.
func PointerPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".fleet", "app.json"), nil
}

// Load reads the pointer file. A missing file is not an error — it returns
// a zero Config, meaning "nothing configured yet".
func Load() (Config, error) {
	path, err := PointerPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the pointer file atomically.
func Save(cfg Config) error {
	path, err := PointerPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode app config: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".app.json.*")
	if err != nil {
		return fmt.Errorf("create app config temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write app config temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close app config temp: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("chmod app config temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

// ExpandHome resolves a leading "~" against the user's home directory and
// makes the result absolute; a path without "~" is just made absolute.
func ExpandHome(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("empty path")
	}
	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if trimmed == "~" {
			return home, nil
		}
		return filepath.Join(home, trimmed[2:]), nil
	}
	return filepath.Abs(trimmed)
}
