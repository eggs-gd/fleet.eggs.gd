package taskprovider

import (
	"sort"
	"sync"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// Board is the in-memory dashboard projection for tasks, workspaces, projects,
// and registry. It intentionally excludes execution sessions and orphans.
type Board struct {
	mu          sync.RWMutex
	root        string
	statuses    []string
	tasks       map[string]Task
	workspaces  map[string]board.Workspace
	projects    map[string]board.Project
	registry    board.RegistryInfo
	generatedAt time.Time
}

// Snapshot is the board-only poll projection (no runtime sessions).
type Snapshot struct {
	GeneratedAt string             `json:"generated_at"`
	Root        string             `json:"root"`
	Statuses    []string           `json:"statuses"`
	Workspaces  []board.Workspace  `json:"workspaces"`
	Projects    []board.Project    `json:"projects"`
	Tasks       []Task             `json:"tasks"`
	LifeItems   []board.LifeItem   `json:"life_items"`
	Registry    board.RegistryInfo `json:"registry"`
}

func NewBoard(root string) *Board {
	return &Board{
		root:       root,
		statuses:   tasklifecycle.TaskStatuses(),
		tasks:      map[string]Task{},
		workspaces: map[string]board.Workspace{},
		projects:   map[string]board.Project{},
	}
}

func (b *Board) Snapshot() Snapshot {
	if b == nil {
		return Snapshot{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	tasks := make([]Task, 0, len(b.tasks))
	for _, task := range b.tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasklifecycle.TaskPickupLess(tasks[i], tasks[j])
	})

	workspaces := make([]board.Workspace, 0, len(b.workspaces))
	for _, workspace := range b.workspaces {
		workspaces = append(workspaces, workspace)
	}
	sort.Slice(workspaces, func(i, j int) bool {
		if workspaces[i].ID == "_life" {
			return true
		}
		if workspaces[j].ID == "_life" {
			return false
		}
		return workspaces[i].Title < workspaces[j].Title
	})

	projects := make([]board.Project, 0, len(b.projects))
	for _, project := range b.projects {
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].WorkspaceID == "_life" && projects[j].WorkspaceID != "_life" {
			return true
		}
		if projects[j].WorkspaceID == "_life" && projects[i].WorkspaceID != "_life" {
			return false
		}
		if projects[i].WorkspaceID != projects[j].WorkspaceID {
			return projects[i].WorkspaceID < projects[j].WorkspaceID
		}
		return projects[i].Title < projects[j].Title
	})

	generatedAt := b.generatedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC().Truncate(time.Second)
	}

	return Snapshot{
		GeneratedAt: generatedAt.Format(time.RFC3339),
		Root:        b.root,
		Statuses:    append([]string{}, b.statuses...),
		Workspaces:  workspaces,
		Projects:    projects,
		Tasks:       tasks,
		LifeItems:   []board.LifeItem{},
		Registry:    b.registry,
	}
}
