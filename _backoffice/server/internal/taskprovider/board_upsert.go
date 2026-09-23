package taskprovider

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/board"
)

func (b *Board) UpsertTask(task Task) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	key := taskStorageKey(task)
	if key == "" {
		return
	}
	for existingKey, existing := range b.tasks {
		if existingKey == key {
			continue
		}
		if tasksShareIdentity(task, existing) || existingKey == task.Path || existingKey == task.RelativePath || existingKey == task.ID {
			delete(b.tasks, existingKey)
		}
	}
	b.tasks[key] = task
	b.ensureWorkspace(task)
	b.ensureProject(task)
	b.generatedAt = time.Now().UTC().Truncate(time.Second)
}

func taskStorageKey(task Task) string {
	if task.RelativePath != "" {
		return task.RelativePath
	}
	if task.Path != "" {
		return task.Path
	}
	return task.ID
}

func tasksShareIdentity(left Task, right Task) bool {
	if sameTaskIdentity(left.RelativePath, right.RelativePath, left.ID, right.ID, left.Ref, right.Ref) {
		return true
	}
	if left.Path != "" && right.Path != "" && left.Path == right.Path {
		return true
	}
	if left.Path != "" && right.RelativePath != "" && left.Path == right.RelativePath {
		return true
	}
	if left.RelativePath != "" && right.Path != "" && left.RelativePath == right.Path {
		return true
	}
	return false
}

func sameTaskIdentity(leftPath, rightPath, leftID, rightID, leftRef, rightRef string) bool {
	if leftPath != "" && rightPath != "" && leftPath == rightPath {
		return true
	}
	if leftID != "" && rightID != "" && leftID == rightID {
		return true
	}
	if leftRef != "" && rightRef != "" && leftRef == rightRef {
		return true
	}
	return false
}

func (b *Board) TaskByPath(path string) (Task, bool) {
	if b == nil {
		return Task{}, false
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	if path == "" {
		return Task{}, false
	}
	if task, ok := b.tasks[path]; ok {
		return task, true
	}
	for _, task := range b.tasks {
		if task.Path == path || task.RelativePath == path {
			return task, true
		}
	}
	return Task{}, false
}

func (b *Board) UpsertWorkspace(workspace board.Workspace) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if workspace.ID == "" {
		return
	}
	if existing, ok := b.workspaces[workspace.ID]; ok {
		workspace.Repositories = mergeStrings(workspace.Repositories, existing.Repositories)
	}
	workspace.Technology = b.technologySummaryForRepositoriesLocked(workspace.Repositories)
	b.workspaces[workspace.ID] = workspace
	for _, project := range board.ProjectsFromWorkspace(workspace) {
		b.upsertProjectLocked(project)
	}
	b.generatedAt = time.Now().UTC().Truncate(time.Second)
}

func (b *Board) SetRegistry(registry board.RegistryInfo) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	b.registry = registry
	b.refreshTechnologySummariesLocked()
	b.generatedAt = time.Now().UTC().Truncate(time.Second)
}

func (b *Board) upsertProjectLocked(project board.Project) {
	if project.ID == "" {
		return
	}
	if existing, ok := b.projects[project.ID]; ok {
		project.Repositories = mergeStrings(project.Repositories, existing.Repositories)
		if project.Summary == "" {
			project.Summary = existing.Summary
		}
	}
	project.Technology = b.technologySummaryForRepositoriesLocked(project.Repositories)
	b.projects[project.ID] = project
}

func (b *Board) ensureProject(task Task) {
	id := task.ProjectID
	if id == "" {
		id = task.Project
	}
	if id == "" {
		id = task.WorkspaceID
	}
	if existing, ok := b.projects[id]; ok {
		existing.Repositories = mergeStrings(existing.Repositories, task.Repositories)
		existing.Technology = b.technologySummaryForRepositoriesLocked(existing.Repositories)
		b.projects[id] = existing
		return
	}

	b.projects[id] = board.Project{
		ID:           id,
		WorkspaceID:  firstNonEmpty(task.WorkspaceID, task.Project),
		Title:        board.ProjectTitleFromID(id),
		Kind:         "task_project",
		Source:       "task",
		Repositories: append([]string{}, task.Repositories...),
		RelativePath: task.RelativePath,
		Technology:   b.technologySummaryForRepositoriesLocked(task.Repositories),
	}
}

func (b *Board) ensureWorkspace(task Task) {
	id := task.WorkspaceID
	if id == "" {
		id = task.Project
	}
	if id == "" {
		id = "unassigned"
	}
	if existing, ok := b.workspaces[id]; ok {
		existing.Repositories = mergeStrings(existing.Repositories, task.Repositories)
		existing.Technology = b.technologySummaryForRepositoriesLocked(existing.Repositories)
		b.workspaces[id] = existing
		return
	}

	projectPath := ""
	relativePath := ""
	if task.Path != "" {
		projectPath = filepath.Join(filepath.Dir(filepath.Dir(task.Path)), "PROJECT.md")
		if rel, err := filepath.Rel(b.root, projectPath); err == nil {
			relativePath = filepath.ToSlash(rel)
		}
	}

	b.workspaces[id] = board.Workspace{
		ID:           id,
		Title:        id,
		Kind:         "workspace",
		ReviewStatus: "runtime",
		Status:       "active",
		Repositories: append([]string{}, task.Repositories...),
		Path:         projectPath,
		RelativePath: relativePath,
		Technology:   b.technologySummaryForRepositoriesLocked(task.Repositories),
	}
}

func (b *Board) refreshTechnologySummariesLocked() {
	for id, workspace := range b.workspaces {
		workspace.Technology = b.technologySummaryForRepositoriesLocked(workspace.Repositories)
		b.workspaces[id] = workspace
	}
	for id, project := range b.projects {
		project.Technology = b.technologySummaryForRepositoriesLocked(project.Repositories)
		b.projects[id] = project
	}
}

func (b *Board) technologySummaryForRepositoriesLocked(paths []string) board.TechnologySummary {
	if len(paths) == 0 || len(b.registry.Repositories) == 0 {
		return board.TechnologySummary{
			EffectiveTags: []string{},
			DetectedTags:  []string{},
			Repositories:  []board.RepositoryTechnology{},
		}
	}

	wanted := map[string]bool{}
	for _, path := range paths {
		if path == "" {
			continue
		}
		wanted[path] = true
	}

	effectiveTags := []string{}
	detectedTags := []string{}
	repositories := []board.RepositoryTechnology{}
	for _, repo := range b.registry.Repositories {
		if !wanted[repo.RelativePath] {
			continue
		}
		effectiveTags = append(effectiveTags, repo.EffectiveTags...)
		detectedTags = append(detectedTags, repo.DetectedTags...)
		repositories = append(repositories, repo)
	}

	return board.TechnologySummary{
		EffectiveTags: sortedUniqueStrings(effectiveTags),
		DetectedTags:  sortedUniqueStrings(detectedTags),
		Repositories:  repositories,
	}
}

func mergeStrings(left []string, right []string) []string {
	seen := map[string]bool{}
	merged := make([]string, 0, len(left)+len(right))
	for _, value := range append(left, right...) {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		merged = append(merged, value)
	}
	return merged
}

func sortedUniqueStrings(values []string) []string {
	seen := map[string]bool{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
