package markdown

import (
	"context"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
)

// Compile-time check: Markdown satisfies the legacy Runtime provider surface.
// taskflow.TaskProvider is exposed via Flow() because List() signatures differ
// between the two contracts (Go has no method overloading).
var _ taskprovider.Provider = (*Provider)(nil)

func (p *Provider) flowCreate(_ context.Context, input taskflow.CreateTask) (taskflow.Task, error) {
	req := tasklifecycle.TaskCreateRequest{
		Title:            input.Title,
		Request:          input.Description,
		Project:          input.Project,
		Repository:       input.Repository,
		Status:           string(input.Status),
		Type:             input.Type,
		Assignee:         input.Assignee,
		AssignmentReason: input.AssignmentReason,
		Acceptance:       input.Acceptance,
		Context:          input.Context,
		SourceInbox:      input.SourceInbox,
		DependsOn:        append([]string{}, input.DependsOn...),
	}
	if input.Priority > 0 {
		priority := input.Priority
		req.Priority = &priority
	}
	created, err := p.CreateFromRequest(req)
	if err != nil {
		return taskflow.Task{}, err
	}
	return toFlowTask(created), nil
}

func (p *Provider) flowGet(_ context.Context, id string) (taskflow.Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return taskflow.Task{}, taskflow.ErrEmptyTaskID
	}
	task, err := p.Load(id)
	if err != nil {
		return taskflow.Task{}, err
	}
	return toFlowTask(task), nil
}

func (p *Provider) flowList(_ context.Context, filter taskflow.TaskFilter) ([]taskflow.Task, error) {
	all, err := p.List()
	if err != nil {
		return nil, err
	}
	out := make([]taskflow.Task, 0, len(all))
	for _, task := range all {
		if filter.Project != "" && task.Project != filter.Project && task.ProjectID != filter.Project {
			continue
		}
		if filter.Status != "" && task.Status != string(filter.Status) {
			continue
		}
		if filter.Assignee != "" && task.Assignee != filter.Assignee {
			continue
		}
		if filter.Ref != "" && task.Ref != filter.Ref {
			continue
		}
		out = append(out, toFlowTask(task))
	}
	return out, nil
}

func (p *Provider) flowUpdate(ctx context.Context, task taskflow.Task) error {
	locator := strings.TrimSpace(task.Locator)
	if locator == "" {
		locator = strings.TrimSpace(task.ID)
	}
	if locator == "" {
		return taskflow.ErrEmptyTaskID
	}
	// Status / assignee / priority writebacks from TaskService. Body is only
	// patched when the application explicitly set it (non-empty); empty means
	// "leave body unchanged" so status transitions do not rewrite markdown.
	patch := tasklifecycle.TaskPatch{
		Path:                locator,
		Status:              string(task.Status),
		Assignee:            task.Assignee,
		AllowStatusOverride: taskflow.StatusOverrideAllowed(ctx),
	}
	if task.Priority > 0 {
		priority := task.Priority
		patch.Priority = &priority
	}
	dependsOn := append([]string{}, task.DependsOn...)
	patch.DependsOn = &dependsOn
	if strings.TrimSpace(task.Body) != "" {
		body := task.Body
		patch.Body = &body
	}
	if project, repository, ok := taskflow.ProjectPatchFrom(ctx); ok {
		patch.Project = project
		patch.Repository = repository
	}
	_, err := p.Mutate(patch)
	return err
}

func (p *Provider) flowAddComment(ctx context.Context, id string, text string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return taskflow.ErrEmptyTaskID
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return taskflow.ErrEmptyComment
	}
	author := taskflow.CommentAuthorFrom(ctx)
	if author == "" {
		author = tasklifecycle.SystemCommentAuthor
	}
	_, err := p.Mutate(tasklifecycle.TaskPatch{
		Path:          id,
		Comment:       text,
		CommentAuthor: author,
	})
	return err
}

func toFlowTask(task tasklifecycle.Task) taskflow.Task {
	locator := task.RelativePath
	if locator == "" {
		locator = task.Path
	}
	comments := make([]taskflow.Comment, 0, len(task.Comments))
	for _, c := range task.Comments {
		comments = append(comments, taskflow.Comment{
			Author:    c.Author,
			CreatedAt: c.CreatedAt,
			Text:      c.Text,
		})
	}
	return taskflow.Task{
		SchemaVersion:    task.SchemaVersion,
		ID:               task.ID,
		Ref:              task.Ref,
		Title:            task.Title,
		Type:             task.Type,
		Status:           taskflow.Status(task.Status),
		Priority:         task.Priority,
		Project:          task.Project,
		ProjectID:        task.ProjectID,
		WorkspaceID:      task.WorkspaceID,
		Repositories:     append([]string{}, task.Repositories...),
		DependsOn:        append([]string{}, task.DependsOn...),
		Assignee:         task.Assignee,
		AssignmentReason: task.AssignmentReason,
		Source:           task.Source,
		SourceInbox:      task.SourceInbox,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
		Summary:          task.Summary,
		Body:             task.Body,
		Comments:         comments,
		BlockedReason:    task.BlockedReason,
		Locator:          locator,
	}
}
