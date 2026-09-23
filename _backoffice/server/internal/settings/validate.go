package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/execution/providers"
)

func ValidateScanRoot(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("scan root is required")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("scan root must be an absolute path")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("scan root is not an existing directory")
	}
	if !info.IsDir() {
		return fmt.Errorf("scan root is not a directory")
	}
	return nil
}

func ValidateExecutable(path string) (warning string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("executable must be an absolute path")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("executable does not exist")
	}
	if info.IsDir() || !info.Mode().IsRegular() {
		return "", fmt.Errorf("executable is not a regular file")
	}
	if info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("executable is not executable")
	}
	if providers.ProbeVersion(path) == "" {
		return "executable is runnable but --version produced no output; Save is allowed for wrappers", nil
	}
	return "", nil
}
