package taskprovider

import (
	"fmt"
	"path/filepath"

	"github.com/eggs-gd/core.eggs.gd/internal/board"
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
