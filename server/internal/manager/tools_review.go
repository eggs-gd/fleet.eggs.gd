package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
)

// Answer records a comment and, when the task is waiting, moves it back to todo.
func (s *Service) Answer(ctx context.Context, ref, text, status string) Response {
	text = strings.TrimSpace(text)
	if text == "" {
		return failResponse(Failure{Code: FailureValidation, Message: "text is required"})
	}
	current, err := findByRef(ctx, s.Tasks, ref)
	if err != nil {
		return s.missingTask(ctx, ref, err)
	}
	next := strings.TrimSpace(status)
	if next == "" && (current.Status == string(taskflow.StatusBlocked) || current.Status == "needs_rework") {
		next = "todo"
	}
	patch := taskflow.PatchInput{Comment: text, CommentAuthor: "owner", Actor: "manager"}
	if next != "" {
		patch.Status = taskflow.Status(next)
	}
	task, err := patchViaService(ctx, s.Tasks, ref, patch)
	if err != nil {
		return failureFromErr(err)
	}
	return Response{OK: true, Result: &Result{Action: "answer", Ref: task.Ref, Status: task.Status, Detail: task}}
}

// Review accepts work or sends it back.
func (s *Service) Review(ctx context.Context, ref, verdict, comment string) Response {
	verdict = strings.TrimSpace(verdict)
	comment = strings.TrimSpace(comment)
	switch verdict {
	case "accept":
		task, err := patchViaService(ctx, s.Tasks, ref, taskflow.PatchInput{Status: taskflow.StatusDone, Actor: "manager"})
		if err != nil {
			return s.missingTask(ctx, ref, err)
		}
		return Response{OK: true, Result: &Result{Action: "review", Ref: task.Ref, Status: task.Status}}
	case "rework":
		if comment == "" {
			return failResponse(Failure{Code: FailureValidation, Message: "rework requires a comment"})
		}
		task, err := patchViaService(ctx, s.Tasks, ref, taskflow.PatchInput{
			Status:        "needs_rework",
			Comment:       "## Rework\n" + comment,
			CommentAuthor: "owner",
			Actor:         "manager",
		})
		if err != nil {
			return s.missingTask(ctx, ref, err)
		}
		return Response{OK: true, Result: &Result{Action: "review", Ref: task.Ref, Status: task.Status}}
	default:
		return failResponse(Failure{
			Code:        FailureValidation,
			Message:     fmt.Sprintf("unknown verdict %q", verdict),
			Suggestions: []string{"accept", "rework"},
		})
	}
}

func (s *Service) missingTask(ctx context.Context, ref string, err error) Response {
	var fail Failure
	if !errors.As(err, &fail) {
		return failureFromErr(err)
	}
	if fail.Code == FailureNotFound {
		tasks, listErr := s.flowTasks(ctx)
		if listErr == nil {
			refs := make([]string, 0, len(tasks))
			for _, task := range tasks {
				refs = append(refs, task.Ref)
			}
			fail.Suggestions = suggestRefs(refs, ref)
		}
	}
	return failResponse(fail)
}
