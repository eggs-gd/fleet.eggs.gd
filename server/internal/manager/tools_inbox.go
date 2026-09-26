package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/workfiles"
)

// Inbox captures, promotes, or lists raw notes.
func (s *Service) Inbox(ctx context.Context, action, text, ref, project, title string) Response {
	if strings.TrimSpace(s.DataRoot) == "" {
		return failResponse(Failure{Code: FailureProviderError, Message: "data root is not configured"})
	}
	switch strings.TrimSpace(action) {
	case "list":
		refs, err := workfiles.ListInbox(s.DataRoot)
		if err != nil {
			return failureFromErr(err)
		}
		return Response{OK: true, Result: &Result{Action: "inbox", Detail: refs}}
	case "capture":
		if strings.TrimSpace(text) == "" {
			return failResponse(Failure{Code: FailureValidation, Message: "text is required"})
		}
		ref, path, err := workfiles.CaptureInbox(s.DataRoot, text, time.Now())
		if err != nil {
			return failureFromErr(err)
		}
		return Response{OK: true, Result: &Result{Action: "inbox", Ref: ref, Path: path}}
	case "promote":
		return s.promoteInbox(ctx, ref, project, title, text)
	default:
		return failResponse(Failure{
			Code:        FailureValidation,
			Message:     fmt.Sprintf("unknown inbox action %q", action),
			Suggestions: []string{"capture", "promote", "list"},
		})
	}
}

func (s *Service) promoteInbox(ctx context.Context, ref, project, title, text string) Response {
	if !workfiles.IsInboxRef(ref) {
		return failResponse(Failure{Code: FailureValidation, Message: "promote requires an INBOX ref"})
	}
	item, err := workfiles.ReadInbox(s.DataRoot, ref)
	if errors.Is(err, workfiles.ErrNotFound) {
		return failResponse(Failure{Code: FailureNotFound, Message: fmt.Sprintf("%s not found", strings.ToUpper(strings.TrimSpace(ref)))})
	}
	if err != nil {
		return failureFromErr(err)
	}
	if item.PromotedTo != "" {
		return Response{OK: true, Result: &Result{Action: "inbox", Ref: item.Ref, Path: item.Path, Detail: map[string]any{"already": true, "promoted_to": item.PromotedTo}}}
	}
	body := item.Body
	if strings.TrimSpace(text) != "" {
		body = strings.TrimSpace(text)
	}
	if strings.TrimSpace(title) == "" {
		title = firstLine(body)
	}
	resp := s.createTask(ctx, Intent{
		Kind:        KindTask,
		Project:     project,
		Title:       title,
		Description: body,
		Acceptance:  body,
		SourceInbox: item.Ref,
		Status:      "backlog",
		Type:        "feature",
	}, PathCommand, "inbox")
	if !resp.OK {
		return resp
	}
	if err := workfiles.MarkPromoted(s.DataRoot, item.Ref, resp.Result.Ref, time.Now()); err != nil {
		return failureFromErr(err)
	}
	resp.Result.Detail = map[string]any{"source_inbox": item.Ref, "promoted_to": resp.Result.Ref}
	return resp
}

// Project records an alias, note, or decision on the project card.
func (s *Service) Project(action, project, text string) Response {
	if strings.TrimSpace(s.DataRoot) == "" {
		return failResponse(Failure{Code: FailureProviderError, Message: "data root is not configured"})
	}
	if !workfiles.ValidProjectID(project) {
		return failResponse(Failure{Code: FailureValidation, Message: "project id is required"})
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return failResponse(Failure{Code: FailureValidation, Message: "text is required"})
	}
	action = strings.TrimSpace(action)
	switch action {
	case "alias", "note", "decision":
	default:
		return failResponse(Failure{
			Code:        FailureValidation,
			Message:     fmt.Sprintf("unknown project action %q", action),
			Suggestions: []string{"alias", "note", "decision"},
		})
	}
	path, wrote, err := workfiles.AddProjectNote(s.DataRoot, project, action, text, time.Now())
	if err != nil {
		return failureFromErr(err)
	}
	return Response{OK: true, Result: &Result{Action: "project", Path: path, Detail: map[string]any{"wrote": wrote}}}
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return "Inbox item"
	}
	return line
}
