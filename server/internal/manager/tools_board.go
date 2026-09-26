package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
)

var workerAgents = []string{"claude", "codex", "cursor", "gemini"}

const (
	defaultBoardLimit = 50
	maxBoardLimit     = 200
	summaryRunes      = 240
)

// BoardQuery selects a slice of the board. Detail "summary" adds the first
// lines of each task's description, which is what a review pass over many tasks
// needs.
type BoardQuery struct {
	View    string `json:"view,omitempty"`
	Project string `json:"project,omitempty"`
	Status  string `json:"status,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

// ShowBoard returns the live board slice named by q.View, filtered and paged.
func (s *Service) ShowBoard(ctx context.Context, q BoardQuery) Response {
	tasks, err := s.flowTasks(ctx)
	if err != nil {
		return failureFromErr(err)
	}
	view := strings.TrimSpace(q.View)
	if view == "" {
		view = "all"
	}
	detail := strings.TrimSpace(q.Detail)
	if detail != "" && detail != "cards" && detail != "summary" {
		return failResponse(Failure{Code: FailureValidation, Message: fmt.Sprintf("unknown detail %q", detail), Suggestions: []string{"cards", "summary"}})
	}
	var shown []taskflow.Task
	switch view {
	case "all":
		shown = tasks
	case "needs_attention":
		shown = filterTasks(tasks, func(task taskflow.Task) bool {
			return task.Status == taskflow.StatusBlocked || strings.TrimSpace(task.BlockedReason) != ""
		})
	case "blocked":
		shown = filterTasks(tasks, func(task taskflow.Task) bool { return task.Status == taskflow.StatusBlocked })
	case "in_review":
		shown = filterTasks(tasks, func(task taskflow.Task) bool { return task.Status == "needs_review" })
	default:
		return failResponse(Failure{
			Code:        FailureValidation,
			Message:     fmt.Sprintf("unknown view %q", view),
			Suggestions: []string{"needs_attention", "blocked", "in_review", "all"},
		})
	}
	if project := strings.TrimSpace(q.Project); project != "" {
		shown = filterTasks(shown, func(task taskflow.Task) bool {
			return strings.EqualFold(firstNonEmpty(task.ProjectID, task.Project, task.WorkspaceID), project) ||
				strings.EqualFold(task.Project, project) || strings.EqualFold(task.WorkspaceID, project)
		})
	}
	if status := strings.TrimSpace(q.Status); status != "" {
		shown = filterTasks(shown, func(task taskflow.Task) bool { return strings.EqualFold(string(task.Status), status) })
	}
	total := len(shown)
	limit := q.Limit
	if limit <= 0 {
		limit = defaultBoardLimit
	}
	if limit > maxBoardLimit {
		limit = maxBoardLimit
	}
	offset := q.Offset
	if offset < 0 || offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := shown[offset:end]
	cards := cardsOf(page)
	if detail == "summary" {
		for i, task := range page {
			cards[i]["summary"] = excerpt(firstNonEmpty(task.Summary, task.Body), summaryRunes)
		}
	}
	out := map[string]any{
		"view":      view,
		"total":     total,
		"tasks":     cards,
		"queues":    queuesByRepository(tasks),
		"ownership": ownershipOf(tasks),
	}
	if end < total {
		out["next_offset"] = end
	}
	return Response{OK: true, Result: &Result{Action: "board", Detail: out}}
}

func excerpt(text string, max int) string {
	text = strings.Join(strings.Fields(text), " ")
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "…"
}

// Task returns one card, its dependencies, blockers, session, and activity tail.
func (s *Service) Task(ctx context.Context, ref string) Response {
	tasks, err := s.flowTasks(ctx)
	if err != nil {
		return failureFromErr(err)
	}
	ref = strings.ToUpper(strings.TrimSpace(ref))
	var found *taskflow.Task
	refs := make([]string, 0, len(tasks))
	byRef := map[string]taskflow.Task{}
	for _, task := range tasks {
		refs = append(refs, task.Ref)
		byRef[strings.ToUpper(task.Ref)] = task
		if strings.EqualFold(task.Ref, ref) {
			copy := task
			found = &copy
		}
	}
	if found == nil {
		return failResponse(Failure{
			Code:        FailureNotFound,
			Message:     fmt.Sprintf("task %s not found", ref),
			Suggestions: suggestRefs(refs, ref),
		})
	}
	var blockers []string
	if reason := strings.TrimSpace(found.BlockedReason); reason != "" {
		blockers = append(blockers, reason)
	}
	for _, dep := range found.DependsOn {
		other, ok := byRef[strings.ToUpper(dep)]
		if !ok {
			blockers = append(blockers, dep+" is not on the board")
			continue
		}
		if other.Status != taskflow.StatusDone && other.Status != "archived" {
			blockers = append(blockers, dep+" is "+string(other.Status))
		}
	}
	session := "none"
	if found.Status == taskflow.StatusDoing && strings.TrimSpace(found.Assignee) != "" {
		session = found.Assignee
	}
	comments := found.Comments
	if len(comments) > 5 {
		comments = comments[len(comments)-5:]
	}
	return Response{OK: true, Result: &Result{Action: "show_task", Ref: found.Ref, Status: string(found.Status), Detail: map[string]any{
		"task":       cardOf(*found),
		"body":       found.Body,
		"depends_on": found.DependsOn,
		"blockers":   blockers,
		"session":    session,
		"activity":   comments,
	}}}
}

// Workers lists agents and the task each one currently holds.
func (s *Service) Workers(ctx context.Context) Response {
	tasks, err := s.flowTasks(ctx)
	if err != nil {
		return failureFromErr(err)
	}
	held := ownershipOf(tasks)
	busy := map[string]string{}
	for _, item := range held {
		busy[item.Assignee] = item.Ref
	}
	var available []string
	for _, agent := range workerAgents {
		if busy[agent] == "" {
			available = append(available, agent)
		}
	}
	return Response{OK: true, Result: &Result{Action: "workers", Detail: map[string]any{
		"available": available,
		"ownership": held,
	}}}
}

// Events returns task.* audit rows after the given id.
func (s *Service) Events(since int64) Response {
	if strings.TrimSpace(s.RuntimeRoot) == "" {
		return failResponse(Failure{Code: FailureProviderError, Message: "runtime root is not configured"})
	}
	events, err := audit.TaskEventsAfter(s.RuntimeRoot, since)
	if err != nil {
		return failureFromErr(err)
	}
	result := &Result{Action: "events", Detail: events}
	if problem, failing := s.Health.Has("audit"); failing {
		result.Warnings = []string{"task events may be missing since " + problem.Since + ": " + problem.Error}
	}
	return Response{OK: true, Result: result}
}

func (s *Service) flowTasks(ctx context.Context) ([]taskflow.Task, error) {
	if s == nil || s.Tasks == nil {
		return nil, Failure{Code: FailureProviderError, Message: "task service is not configured"}
	}
	return s.Tasks.List(ctx, taskflow.TaskFilter{})
}

type ownership struct {
	Assignee   string `json:"assignee"`
	Ref        string `json:"ref"`
	Repository string `json:"repository,omitempty"`
}

func ownershipOf(tasks []taskflow.Task) []ownership {
	var held []ownership
	for _, task := range tasks {
		if task.Status != taskflow.StatusDoing && task.Status != taskflow.StatusBlocked && task.Status != "needs_review" {
			continue
		}
		assignee := strings.TrimSpace(task.Assignee)
		if assignee == "" || assignee == "unassigned" {
			continue
		}
		repo := ""
		if len(task.Repositories) > 0 {
			repo = task.Repositories[0]
		}
		held = append(held, ownership{Assignee: assignee, Ref: task.Ref, Repository: repo})
	}
	return held
}

func queuesByRepository(tasks []taskflow.Task) map[string]int {
	queues := map[string]int{}
	for _, task := range tasks {
		if task.Status != "todo" && task.Status != taskflow.StatusDoing && task.Status != "backlog" {
			continue
		}
		repo := "unassigned"
		if len(task.Repositories) > 0 && strings.TrimSpace(task.Repositories[0]) != "" {
			repo = task.Repositories[0]
		}
		queues[repo]++
	}
	return queues
}

func cardsOf(tasks []taskflow.Task) []map[string]any {
	out := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, cardOf(task))
	}
	return out
}

func cardOf(task taskflow.Task) map[string]any {
	return map[string]any{
		"ref":        task.Ref,
		"title":      task.Title,
		"status":     task.Status,
		"assignee":   task.Assignee,
		"project":    firstNonEmpty(task.ProjectID, task.Project, task.WorkspaceID),
		"depends_on": task.DependsOn,
	}
}

func filterTasks(tasks []taskflow.Task, keep func(taskflow.Task) bool) []taskflow.Task {
	var out []taskflow.Task
	for _, task := range tasks {
		if keep(task) {
			out = append(out, task)
		}
	}
	return out
}

func suggestRefs(refs []string, query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	var hits []string
	for _, ref := range refs {
		if query != "" && strings.Contains(strings.ToLower(ref), query) {
			hits = append(hits, ref)
		}
	}
	if len(hits) == 0 {
		hits = append(hits, refs...)
	}
	if len(hits) > 5 {
		hits = hits[:5]
	}
	return hits
}
