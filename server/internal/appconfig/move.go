package appconfig

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// MoveDataRoot relocates the Data root from oldRoot to newRoot. A no-op
// when the two paths are equal or oldRoot does not exist. Refuses to move
// onto a destination that already holds files, to avoid silently merging
// or clobbering unrelated content.
func MoveDataRoot(oldRoot, newRoot string) error {
	if oldRoot == newRoot {
		return nil
	}
	if _, err := os.Stat(oldRoot); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("stat %s: %w", oldRoot, err)
	}

	if info, err := os.Stat(newRoot); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("destination %s exists and is not a directory", newRoot)
		}
		entries, err := os.ReadDir(newRoot)
		if err != nil {
			return fmt.Errorf("read %s: %w", newRoot, err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("destination %s already exists and is not empty", newRoot)
		}
		if err := os.Remove(newRoot); err != nil {
			return fmt.Errorf("remove empty destination %s: %w", newRoot, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", newRoot, err)
	}

	if err := os.MkdirAll(filepath.Dir(newRoot), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(newRoot), err)
	}

	if err := os.Rename(oldRoot, newRoot); err != nil {
		if !errors.Is(err, syscall.EXDEV) {
			return fmt.Errorf("move %s to %s: %w", oldRoot, newRoot, err)
		}
		if err := copyDir(oldRoot, newRoot); err != nil {
			return fmt.Errorf("copy %s to %s: %w", oldRoot, newRoot, err)
		}
		if err := os.RemoveAll(oldRoot); err != nil {
			return fmt.Errorf("remove old root %s after copy: %w", oldRoot, err)
		}
	}
	return nil
}

// copyDir is the fallback for moving across filesystems, where os.Rename
// returns EXDEV (e.g. the Data root and its new location are on different
// volumes).
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm()|0o700)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
