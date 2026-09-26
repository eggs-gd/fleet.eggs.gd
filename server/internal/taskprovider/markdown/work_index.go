package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// RebuildWorkIndex regenerates Work/INDEX.md from Markdown task cards and
// workspace PROJECT.md files. This is Markdown-provider derived state.
// Before rendering, it migrates legacy type: todo frontmatter to feature.
func RebuildWorkIndex(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	if err := migrateLegacyTodoTaskTypes(root); err != nil {
		return err
	}

	workDir := filepath.Join(root, "Work")
	tasks, err := New(root, Hooks{}).List()
	if err != nil {
		return err
	}
	workspaces, err := loadIndexWorkspaces(root, workDir)
	if err != nil {
		return err
	}

	projects := projectsFromWorkspaces(workspaces)
	index := renderWorkIndex(tasks, workspaces, projects)
	return os.WriteFile(filepath.Join(workDir, "INDEX.md"), []byte(index), 0o644)
}

// RebuildDerivedViews implements the optional provider capability used at
// bootstrap without branching on Provider.Type().
func (p *Provider) RebuildDerivedViews() error {
	if p == nil {
		return nil
	}
	return RebuildWorkIndex(p.root)
}

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
		return strings.ToLower(workspaces[i].Title) < strings.ToLower(workspaces[j].Title)
	})
	return workspaces, nil
}

func renderWorkIndex(tasks []tasklifecycle.Task, workspaces []board.Workspace, projects []board.Project) string {
	var out strings.Builder
	out.WriteString("# Work Index\n\n")
	out.WriteString("Generated read-only projection from canonical Core task and workspace files. Do not edit by hand.\n\n")
	out.WriteString("## Task Queue\n\n")
	out.WriteString("| Status | Count |\n")
	out.WriteString("|---|---:|\n")
	for _, status := range tasklifecycle.TaskStatuses() {
		out.WriteString(fmt.Sprintf("| `%s` | %d |\n", status, countTasksByStatus(tasks, status)))
	}
	out.WriteString("\n")

	writeTaskSection(&out, "Needs Review", "Tasks have produced physical artifacts and wait for the operator's review.", tasks, "needs_review")
	writeTaskSection(&out, "Needs Rework", "Tasks returned from review and picked before normal todo work.", tasks, "needs_rework")
	writeTaskSection(&out, "Todo", "Tasks are ready for active worker pickup.", tasks, "todo")
	writeTaskSection(&out, "Doing", "Tasks are currently being worked on.", tasks, "doing")
	writeTaskSection(&out, "Blocked", "Tasks that cannot proceed without missing info, approval, access, or failed launch resolution.", tasks, "blocked")
	writeTaskSection(&out, "Done", "Tasks are completed and kept for audit/history.", tasks, "done")
	writeTaskSection(&out, "Backlog", "Tasks are structured but not prioritized for worker pickup.", tasks, "backlog")
	writeTaskSection(&out, "Archived", "Tasks intentionally kept only for history.", tasks, "archived")
	writeProjectSection(&out, projects, tasks)
	writeWorkspaceSection(&out, workspaces)
	return out.String()
}

func writeTaskSection(out *strings.Builder, title string, description string, tasks []tasklifecycle.Task, status string) {
	out.WriteString("## " + title + "\n\n")
	out.WriteString(description + "\n\n")
	out.WriteString("| Priority | Ref | Task | Project | Type | Assignee |\n")
	out.WriteString("|---:|---|---|---|---|---|\n")
	for _, task := range tasks {
		if task.Status != status {
			continue
		}
		out.WriteString(fmt.Sprintf(
			"| %d | `%s` | [%s](%s) | `%s` | `%s` | `%s` |\n",
			task.Priority,
			task.Ref,
			escapeTable(task.Title),
			workIndexLink(task.RelativePath),
			task.ProjectID,
			task.Type,
			task.Assignee,
		))
	}
	out.WriteString("\n")
}

func writeProjectSection(out *strings.Builder, projects []board.Project, tasks []tasklifecycle.Task) {
	out.WriteString("## Projects\n\n")
	out.WriteString("Canonical project layer derived from workspace cards and task ownership.\n\n")
	out.WriteString("| Project | Workspace | Kind | Repositories | Tasks |\n")
	out.WriteString("|---|---|---|---:|---:|\n")
	for _, project := range projects {
		out.WriteString(fmt.Sprintf(
			"| `%s` | `%s` | `%s` | %d | %d |\n",
			project.ID,
			project.WorkspaceID,
			project.Kind,
			len(project.Repositories),
			countTasksByProject(tasks, project.ID),
		))
	}
	out.WriteString("\n")
}

func writeWorkspaceSection(out *strings.Builder, workspaces []board.Workspace) {
	out.WriteString("## Workspaces\n\n")
	out.WriteString("Canonical workspace cards tracked by Core.\n\n")
	out.WriteString("| Workspace | Kind | Repositories | Status |\n")
	out.WriteString("|---|---|---:|---|\n")
	for _, workspace := range workspaces {
		out.WriteString(fmt.Sprintf(
			"| [%s](%s) | `%s` | %d | `%s` |\n",
			escapeTable(workspace.Title),
			workIndexLink(workspace.RelativePath),
			workspace.Kind,
			len(workspace.Repositories),
			workspace.Status,
		))
	}
}

func countTasksByStatus(tasks []tasklifecycle.Task, status string) int {
	count := 0
	for _, task := range tasks {
		if task.Status == status {
			count++
		}
	}
	return count
}

func countTasksByProject(tasks []tasklifecycle.Task, projectID string) int {
	count := 0
	for _, task := range tasks {
		if task.ProjectID == projectID {
			count++
		}
	}
	return count
}

func projectsFromWorkspaces(workspaces []board.Workspace) []board.Project {
	seen := map[string]bool{}
	var projects []board.Project
	for _, workspace := range workspaces {
		for _, project := range board.ProjectsFromWorkspace(workspace) {
			if seen[project.ID] {
				continue
			}
			seen[project.ID] = true
			projects = append(projects, project)
		}
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].WorkspaceID != projects[j].WorkspaceID {
			return projects[i].WorkspaceID < projects[j].WorkspaceID
		}
		return projects[i].ID < projects[j].ID
	})
	return projects
}

func escapeTable(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func workIndexLink(relativePath string) string {
	return strings.TrimPrefix(relativePath, "Work/")
}
