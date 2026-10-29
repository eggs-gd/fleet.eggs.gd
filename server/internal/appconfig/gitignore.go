package appconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureIgnored adds entry to root/.gitignore when that file exists and does
// not already list it, so a file written later cannot be committed by accident.
// A root without a .gitignore is left alone.
func EnsureIgnored(root, entry string) error {
	path := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read .gitignore: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		data = append(data, '\n')
	}
	data = append(data, entry+"\n"...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("update .gitignore: %w", err)
	}
	return nil
}
