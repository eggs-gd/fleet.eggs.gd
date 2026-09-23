package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/board"
	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
)

// TaskSummary is a storage-agnostic task view for Manager responses.
type TaskSummary struct {
	Ref          string   `json:"ref"`
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	Priority     int      `json:"priority"`
	Project      string   `json:"project"`
	Repositories []string `json:"repositories,omitempty"`
	Assignee     string   `json:"assignee,omitempty"`
	Path         string   `json:"path"`
	RelativePath string   `json:"relative_path"`
	Summary      string   `json:"summary,omitempty"`
}

// BoardView is the workspace/registry projection Manager needs for vocabulary
// and project/repository resolution. It intentionally excludes execution sessions.
type BoardView struct {
	Workspaces []board.Workspace
	Projects   []board.Project
	Registry   board.RegistryInfo
}

// BoardReader provides read-only workspace/registry projection for vocabulary
// and project/repository resolution. Task queries and mutations go through
// TaskManagement (TaskService), never through this reader (CORE-110).
type BoardReader interface {
	Board() BoardView
}

func (v BoardView) Board() BoardView { return v }

func summaryFromFlow(task taskflow.Task) TaskSummary {
	path := strings.TrimSpace(task.Locator)
	if path == "" {
		path = strings.TrimSpace(task.ID)
	}
	return TaskSummary{
		Ref:          task.Ref,
		ID:           task.ID,
		Title:        task.Title,
		Status:       string(task.Status),
		Priority:     task.Priority,
		Project:      firstNonEmpty(task.ProjectID, task.Project, task.WorkspaceID),
		Repositories: append([]string{}, task.Repositories...),
		Assignee:     task.Assignee,
		Path:         path,
		RelativePath: path,
		Summary:      task.Summary,
	}
}

// createViaService creates a task through Contour 1 TaskManagement.
func createViaService(ctx context.Context, tasks TaskManagement, title, description, project, repository, status, typ, assignee, reason, source string, priority *int) (TaskSummary, error) {
	if tasks == nil {
		return TaskSummary{}, Failure{Code: FailureProviderError, Message: "task service is not configured"}
	}
	input := taskflow.CreateTask{
		Title:            title,
		Description:      description,
		Project:          project,
		Repository:       repository,
		Status:           taskflow.Status(status),
		Type:             typ,
		Assignee:         assignee,
		AssignmentReason: reason,
		Source:           source,
	}
	if priority != nil {
		input.Priority = *priority
	}
	task, err := tasks.Create(ctx, input)
	if err != nil {
		return TaskSummary{}, Failure{Code: FailureProviderError, Message: err.Error()}
	}
	return summaryFromFlow(task), nil
}

func patchViaService(ctx context.Context, tasks TaskManagement, ref string, patch taskflow.PatchInput) (TaskSummary, error) {
	if tasks == nil {
		return TaskSummary{}, Failure{Code: FailureProviderError, Message: "task service is not configured"}
	}
	id, err := resolveTaskID(ctx, tasks, ref)
	if err != nil {
		return TaskSummary{}, err
	}
	task, err := tasks.Patch(ctx, id, patch)
	if err != nil {
		return TaskSummary{}, Failure{Code: FailureProviderError, Message: err.Error()}
	}
	return summaryFromFlow(task), nil
}

func resolveTaskID(ctx context.Context, tasks TaskManagement, ref string) (string, error) {
	task, err := findByRef(ctx, tasks, ref)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(task.RelativePath)
	if id == "" {
		id = strings.TrimSpace(task.Path)
	}
	if id == "" {
		id = strings.TrimSpace(task.ID)
	}
	if id == "" {
		return "", Failure{Code: FailureNotFound, Message: fmt.Sprintf("task %s has no provider locator", ref)}
	}
	return id, nil
}

func findByRef(ctx context.Context, tasks TaskManagement, ref string) (TaskSummary, error) {
	ref = strings.ToUpper(strings.TrimSpace(ref))
	if ref == "" {
		return TaskSummary{}, Failure{Code: FailureNotFound, Message: "task ref is required"}
	}
	if tasks == nil {
		return TaskSummary{}, Failure{Code: FailureProviderError, Message: "task service is not configured"}
	}
	listed, err := tasks.List(ctx, taskflow.TaskFilter{Ref: ref})
	if err != nil {
		return TaskSummary{}, Failure{Code: FailureProviderError, Message: err.Error()}
	}
	for _, task := range listed {
		if strings.EqualFold(task.Ref, ref) {
			return summaryFromFlow(task), nil
		}
	}
	return TaskSummary{}, Failure{Code: FailureNotFound, Message: fmt.Sprintf("task %s not found", ref)}
}

func listViaService(ctx context.Context, tasks TaskManagement, project, status, ref string) ([]TaskSummary, error) {
	if tasks == nil {
		return nil, Failure{Code: FailureProviderError, Message: "task service is not configured"}
	}
	listed, err := tasks.List(ctx, taskflow.TaskFilter{
		Project: project,
		Status:  taskflow.Status(strings.TrimSpace(status)),
		Ref:     strings.TrimSpace(ref),
	})
	if err != nil {
		return nil, Failure{Code: FailureProviderError, Message: err.Error()}
	}
	out := make([]TaskSummary, 0, len(listed))
	for _, task := range listed {
		if project != "" && !projectMatchesFlow(task, project) {
			continue
		}
		out = append(out, summaryFromFlow(task))
	}
	return out, nil
}

func projectMatchesFlow(task taskflow.Task, project string) bool {
	project = strings.TrimSpace(strings.ToLower(project))
	candidates := []string{task.Project, task.ProjectID, task.WorkspaceID}
	for _, candidate := range candidates {
		if strings.EqualFold(strings.TrimSpace(candidate), project) {
			return true
		}
		if strings.Contains(strings.ToLower(candidate), project) {
			return true
		}
	}
	return false
}
