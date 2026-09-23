package board

import (
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

// ProjectsFromWorkspace derives the canonical project layer from one workspace card.
func ProjectsFromWorkspace(workspace Workspace) []Project {
	projects := []Project{
		{
			ID:           workspace.ID,
			WorkspaceID:  workspace.ID,
			Title:        workspace.Title,
			Kind:         "workspace",
			Source:       workspace.Source,
			Repositories: append([]string{}, workspace.Repositories...),
			Summary:      workspace.Summary,
			Path:         workspace.Path,
			RelativePath: workspace.RelativePath,
		},
	}

	if workspace.Kind != "workspace_group" {
		return projects
	}

	seen := map[string]bool{workspace.ID: true}
	for _, repository := range workspace.Repositories {
		projectID := tasklifecycle.ProjectIDFromRepository(workspace.ID, repository)
		if seen[projectID] {
			continue
		}
		seen[projectID] = true
		projects = append(projects, Project{
			ID:           projectID,
			WorkspaceID:  workspace.ID,
			Title:        projectTitleFromRepository(repository),
			Kind:         "repository_project",
			Source:       "workspace.repositories",
			Repositories: []string{repository},
			RelativePath: workspace.RelativePath,
		})
	}
	return projects
}

func projectTitleFromRepository(repository string) string {
	repository = strings.Trim(repository, "/")
	if repository == "" {
		return "Untitled Project"
	}
	parts := strings.Split(repository, "/")
	return parts[len(parts)-1]
}

// ProjectTitleFromID derives a display title from a project id path segment.
func ProjectTitleFromID(id string) string {
	id = strings.Trim(id, "/")
	if id == "" {
		return "Untitled Project"
	}
	parts := strings.Split(id, "/")
	return strings.ReplaceAll(parts[len(parts)-1], "-", " ")
}
