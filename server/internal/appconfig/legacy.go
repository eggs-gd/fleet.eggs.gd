package appconfig

import (
	"os"
	"path/filepath"
)

// ReadPath is where to read a root-relative config file from. A file renamed
// between releases keeps working: when only the legacy name exists, that is
// read, so an existing Data root does not lose its settings.
func ReadPath(root, name, legacy string) string {
	current := filepath.Join(root, name)
	if _, err := os.Stat(current); err == nil {
		return current
	}
	old := filepath.Join(root, legacy)
	if _, err := os.Stat(old); err == nil {
		return old
	}
	return current
}
