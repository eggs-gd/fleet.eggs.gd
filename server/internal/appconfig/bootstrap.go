package appconfig

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:template
var templateFS embed.FS

const templateDir = "template"

// EnsureDataRoot creates root from the bundled directory template the first
// time it is seen. An already-existing root (even an empty directory) is
// left untouched — this only fires for a genuinely fresh Data root, such as
// the first run on a new machine.
func EnsureDataRoot(root string) (created bool, err error) {
	if _, statErr := os.Stat(root); statErr == nil {
		return false, nil
	} else if !os.IsNotExist(statErr) {
		return false, fmt.Errorf("stat %s: %w", root, statErr)
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return false, fmt.Errorf("create data root %s: %w", root, err)
	}

	walkErr := fs.WalkDir(templateFS, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(root, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if walkErr != nil {
		return false, fmt.Errorf("bootstrap data root %s: %w", root, walkErr)
	}
	return true, nil
}
