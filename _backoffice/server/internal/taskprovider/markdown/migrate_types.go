package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

// migrateLegacyTodoTaskTypes rewrites type: todo to type: feature on Markdown
// task cards. "todo" is a lifecycle status, not a task type. Called during
// Work index regeneration / maintenance so old cards converge without a
// separate one-shot migration command.
func migrateLegacyTodoTaskTypes(root string) error {
	matches, err := filepath.Glob(filepath.Join(root, "Work", "*", "tasks", "*.md"))
	if err != nil {
		return fmt.Errorf("list task files for type migration: %w", err)
	}
	for _, path := range matches {
		if filepath.Base(path) == "README.md" {
			continue
		}
		if err := migrateLegacyTodoTaskTypeFile(path); err != nil {
			return fmt.Errorf("migrate type on %s: %w", path, err)
		}
	}
	return nil
}

func migrateLegacyTodoTaskTypeFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	frontmatter, body, err := splitMarkdown(string(data))
	if err != nil {
		// Incomplete or non-card markdown under tasks/ — leave untouched.
		return nil
	}
	if frontmatterScalar(frontmatter, "type") != "todo" {
		return nil
	}
	updated, err := setFrontmatterValue(frontmatter, "type", tasklifecycle.DefaultTaskType)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, []byte("---\n"+updated+"---\n\n"+body), 0o644)
}

func frontmatterScalar(frontmatter string, key string) string {
	for _, line := range strings.Split(frontmatter, "\n") {
		currentKey, value, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, "  ") {
			continue
		}
		if strings.TrimSpace(currentKey) != key {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return ""
}
