package markdown

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eggs-gd/fleet.eggs.gd/internal/mdfile"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
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
	doc, err := mdfile.ParseDoc(string(data))
	if err != nil {
		// Incomplete or non-card markdown under tasks/ — leave untouched.
		return nil
	}
	if value, _ := doc.Scalar("type"); value != "todo" {
		return nil
	}
	doc.SetString("type", tasklifecycle.DefaultTaskType)
	return writeFileAtomic(path, []byte(doc.Render(doc.Body())), 0o644)
}
