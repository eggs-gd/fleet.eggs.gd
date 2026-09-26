package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
)

// UpdateInput changes fields of an existing task. Only the fields that are set
// change. The task service cannot rename a task or change its type.
type UpdateInput struct {
	Ref        string    `json:"ref"`
	Project    string    `json:"project,omitempty"`
	Repository string    `json:"repository,omitempty"`
	DependsOn  *[]string `json:"depends_on,omitempty"`
	Body       *string   `json:"body,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

// Update rewrites the project, repository, dependencies, or description of a
// task after checking each one against the board.
func (s *Service) Update(ctx context.Context, in UpdateInput) Response {
	if strings.TrimSpace(in.Project) == "" && strings.TrimSpace(in.Repository) == "" && in.DependsOn == nil && in.Body == nil {
		return failResponse(Failure{Code: FailureValidation, Message: "nothing to update", Suggestions: []string{"project", "repository", "depends_on", "body"}})
	}
	current, err := findByRef(ctx, s.Tasks, in.Ref)
	if err != nil {
		return s.missingTask(ctx, in.Ref, err)
	}
	patch := taskflow.PatchInput{Actor: "manager", Reason: strings.TrimSpace(in.Reason), DependsOn: in.DependsOn, Body: in.Body}
	var fixes []string
	if project := strings.TrimSpace(in.Project); project != "" {
		found, err := s.confidentProject(project)
		if err != nil {
			return failureFromErr(err)
		}
		if found == "" {
			fixes = append(fixes, fmt.Sprintf("project %q is not one confident match; call manager_resolve_project", project))
		}
		patch.Project = found
	}
	if repo := strings.TrimSpace(in.Repository); repo != "" {
		if known := s.knownRepositories(); len(known) > 0 && !repoKnown(known, repo) {
			fixes = append(fixes, "unknown repository "+repo)
		}
		patch.Repository = repo
	}
	if in.DependsOn != nil {
		fixes = append(fixes, s.checkDependencies(ctx, current.Ref, *in.DependsOn)...)
	}
	if len(fixes) > 0 {
		return failResponse(Failure{Code: FailureValidation, Message: "update needs changes", Suggestions: fixes})
	}
	task, err := patchViaService(ctx, s.Tasks, current.Ref, patch)
	if err != nil {
		return failureFromErr(err)
	}
	return Response{OK: true, Result: &Result{Action: "update", Ref: task.Ref, Status: task.Status, Detail: task}}
}

// confidentProject returns the project id when phrase names exactly one
// project, and "" otherwise.
func (s *Service) confidentProject(phrase string) (string, error) {
	exact, _, err := s.projectCandidates(phrase)
	if err != nil {
		return "", err
	}
	if len(exact) != 1 {
		return "", nil
	}
	return exact[0], nil
}
