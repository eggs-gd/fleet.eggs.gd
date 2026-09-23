package taskprovider

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/board"
)

func loadIndexWorkspaces(root string, workDir string) ([]board.Workspace, error) {
	matches, err := filepath.Glob(filepath.Join(workDir, "*", "PROJECT.md"))
	if err != nil {
		return nil, err
	}

	workspaces := make([]board.Workspace, 0, len(matches))
	for _, path := range matches {
		workspace, err := board.LoadWorkspaceFile(root, path)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, workspace)
	}
	sort.Slice(workspaces, func(i, j int) bool {
		if workspaces[i].ID == "_life" {
			return true
		}
		if workspaces[j].ID == "_life" {
			return false
		}
		return strings.ToLower(workspaces[i].Title) < strings.ToLower(workspaces[j].Title)
	})
	return workspaces, nil
}
