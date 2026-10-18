package taskprovider

import (
	"fmt"
	"path/filepath"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
)

// Hydrate loads workspaces, tasks, and registry into the board projection.
func Hydrate(boardProj *Board, root string, tasks Provider) (listed []Task, workspaces []board.Workspace, err error) {
	if boardProj == nil {
		return nil, nil, fmt.Errorf("hydrate board: board is nil")
	}
	if tasks == nil {
		return nil, nil, fmt.Errorf("hydrate board: task provider is nil")
	}

	workspaces, err = loadIndexWorkspaces(root, filepath.Join(root, "Work"))
	if err != nil {
		return nil, nil, err
	}
	for _, workspace := range workspaces {
		boardProj.UpsertWorkspace(workspace)
		boardProj.stampCard(workspace.Path)
	}

	listed, err = tasks.List()
	if err != nil {
		return nil, nil, err
	}
	for _, task := range listed {
		boardProj.UpsertTask(task)
	}

	registry, err := loadRegistry(root)
	if err != nil {
		return nil, nil, err
	}
	boardProj.SetRegistry(registry)
	return listed, workspaces, nil
}

// ReloadRegistry re-reads _registry files into the live board without restarting serve.
func ReloadRegistry(boardProj *Board, root string) error {
	if boardProj == nil {
		return fmt.Errorf("reload registry: board is nil")
	}
	registry, err := loadRegistry(root)
	if err != nil {
		return err
	}
	boardProj.SetRegistry(registry)
	return nil
}

// ReloadWorkspace re-reads one project card into the live board, so edits made
// after Hydrate (milestones, aliases, summary) show without restarting serve.
func ReloadWorkspace(boardProj *Board, root, project string) error {
	if boardProj == nil {
		return fmt.Errorf("reload workspace: board is nil")
	}
	workspace, err := board.LoadWorkspaceFile(root, filepath.Join(root, "Work", project, "PROJECT.md"))
	if err != nil {
		return fmt.Errorf("reload workspace %s: %w", project, err)
	}
	boardProj.UpsertWorkspace(workspace)
	boardProj.stampCard(workspace.Path)
	return nil
}

// SyncCards re-reads every project card whose file changed (or appeared) since
// it was last loaded, so edits made by hand or by the scanner reach the live
// board without a restart. It returns the ids of the workspaces it refreshed.
func SyncCards(boardProj *Board, root string) ([]string, error) {
	if boardProj == nil {
		return nil, fmt.Errorf("sync cards: board is nil")
	}
	matches, err := filepath.Glob(filepath.Join(root, "Work", "*", "PROJECT.md"))
	if err != nil {
		return nil, err
	}
	var refreshed []string
	for _, path := range matches {
		abs, err := filepath.Abs(path)
		if err != nil || !boardProj.stampCard(abs) {
			continue
		}
		workspace, err := board.LoadWorkspaceFile(root, abs)
		if err != nil {
			return refreshed, fmt.Errorf("sync card %s: %w", path, err)
		}
		boardProj.UpsertWorkspace(workspace)
		refreshed = append(refreshed, workspace.ID)
	}
	return refreshed, nil
}
