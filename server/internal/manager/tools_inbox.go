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
		items, err := workfiles.ListInbox(s.DataRoot)
		if err != nil {
			return failureFromErr(err)
		}
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			out = append(out, map[string]any{
				"ref": item.Ref, "status": item.Status, "preview": item.Preview,
				"promoted_to": item.PromotedTo,
			})
		}
		return Response{OK: true, Result: &Result{Action: "inbox", Detail: out}}
	case "show":
		item, err := s.readInbox(ref)
		if err != nil {
			return failureFromErr(err)
		}
		if item == nil {
			return failResponse(Failure{Code: FailureNotFound, Message: fmt.Sprintf("%s not found", strings.ToUpper(strings.TrimSpace(ref)))})
		}
		return Response{OK: true, Result: &Result{Action: "inbox", Ref: item.Ref, Path: item.Path, Detail: map[string]any{
			"status": item.Status, "body": item.Body, "promoted_to": item.PromotedTo,
		}}}
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
			Suggestions: []string{"capture", "promote", "list", "show"},
		})
	}
}

// readInbox returns nil (not an error) when ref is not found, so callers can
// give a not-found failure with the operator-facing ref case.
func (s *Service) readInbox(ref string) (*workfiles.InboxItem, error) {
	item, err := workfiles.ReadInbox(s.DataRoot, ref)
	if errors.Is(err, workfiles.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// promoteInbox creates one task from a captured item. Calling it more than
// once on the same ref is expected when one input decomposes into several
// tasks: each call creates another task and links it back onto the item
// (see Service.createTask), instead of refusing after the first.
func (s *Service) promoteInbox(ctx context.Context, ref, project, title, text string) Response {
	if !workfiles.IsInboxRef(ref) {
		return failResponse(Failure{Code: FailureValidation, Message: "promote requires an INBOX ref"})
	}
	item, err := s.readInbox(ref)
	if err != nil {
		return failureFromErr(err)
	}
	if item == nil {
		return failResponse(Failure{Code: FailureNotFound, Message: fmt.Sprintf("%s not found", strings.ToUpper(strings.TrimSpace(ref)))})
	}
	body := item.Body
	if strings.TrimSpace(text) != "" {
		body = strings.TrimSpace(text)
	}
	if strings.TrimSpace(title) == "" {
		title = firstLine(body)
	}
	return s.createTask(ctx, Intent{
		Kind:        KindTask,
		Project:     project,
		Title:       title,
		Description: body,
		SourceInbox: item.Ref,
		Status:      "backlog",
		Type:        "feature",
	}, PathCommand, "inbox")
}

// Alias records an alias for a project, so the Manager can recognize the
// name the person uses. It is the only thing the Manager writes on a project
// card: what a project is lives in its repository.
func (s *Service) Alias(project, alias string) Response {
	if strings.TrimSpace(s.DataRoot) == "" {
		return failResponse(Failure{Code: FailureProviderError, Message: "data root is not configured"})
	}
	if !workfiles.ValidProjectID(project) {
		return failResponse(Failure{Code: FailureValidation, Message: "project id is required"})
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return failResponse(Failure{Code: FailureValidation, Message: "alias is required"})
	}
	path, wrote, err := workfiles.AddProjectAlias(s.DataRoot, project, alias)
	if err != nil {
		return failureFromErr(err)
	}
	return Response{OK: true, Result: &Result{Action: "alias", Path: path, Detail: map[string]any{"wrote": wrote}}}
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return "Inbox item"
	}
	return line
}
