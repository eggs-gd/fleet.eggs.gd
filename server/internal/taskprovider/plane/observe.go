package plane

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// ObservedChange is one Plane-backed task mutation already observed and
// normalized into a TaskEvent. The taskprovider listener (or a future webhook HTTP
// handler) feeds Event into the shared TaskEvent channel; Task is the loaded
// snapshot used to refresh the in-memory dashboard store.
type ObservedChange struct {
	Event taskflow.TaskEvent
	Task  tasklifecycle.Task
}

// PollTracker remembers the last-seen fingerprint and After snapshot per Plane
// locator so ObservePoll can emit TaskEvent only when pickup-relevant fields
// change, with Before populated from the prior After.
type PollTracker struct {
	seen  map[string]string
	tasks map[string]taskflow.Task
}

// NewPollTracker returns an empty poll dedup state.
func NewPollTracker() *PollTracker {
	return &PollTracker{seen: map[string]string{}, tasks: map[string]taskflow.Task{}}
}

// ObservePoll lists Plane work items, compares them to the tracker, and
// returns normalized TaskEvents for new or changed tasks. Unchanged tasks
// produce no events. This is the Plane peer of Markdown ObserveFileChange:
// polling detects the change; this method owns reload semantics and the
// TaskEvent shape (Before/After/Source).
func (p *Provider) ObservePoll(tracker *PollTracker) ([]ObservedChange, error) {
	if tracker == nil {
		tracker = NewPollTracker()
	}
	if tracker.seen == nil {
		tracker.seen = map[string]string{}
	}
	if tracker.tasks == nil {
		tracker.tasks = map[string]taskflow.Task{}
	}

	tasks, err := p.List()
	if err != nil {
		return nil, err
	}

	out := make([]ObservedChange, 0, len(tasks))
	for _, task := range tasks {
		locator := taskEventID(task)
		if locator == "" {
			continue
		}
		fingerprint := pollFingerprint(task)
		previous, hadPrevious := tracker.seen[locator]
		if hadPrevious && previous == fingerprint {
			continue
		}
		var before *tasklifecycle.Task
		if prevFlow, ok := tracker.tasks[locator]; ok {
			prev := lifecycleFromFlow(prevFlow)
			before = &prev
		}
		tracker.seen[locator] = fingerprint
		afterFlow := toFlowTask(task)
		tracker.tasks[locator] = afterFlow
		out = append(out, ObservedChange{
			Event: TaskEventFor(before, task, taskflow.TaskEventSourcePlanePoll),
			Task:  task,
		})
	}
	return out, nil
}

// ObserveWebhook normalizes a Plane webhook JSON body into a TaskEvent by
// extracting a work-item id and reloading through Load. Supported shapes are
// deliberately tolerant: Plane's webhook envelope has varied across versions
// (`data.id`, `data.work_item.id`, `issue.id`, top-level `id`). Unsupported
// payloads return an error rather than guessing a locator.
//
// Core does not host a webhook listener in this slice; this method is the
// adapter contract so a future HTTP route can feed the same TaskEvent channel
// the listener already uses without Plane-specific branches in the launcher.
func (p *Provider) ObserveWebhook(payload []byte) (ObservedChange, error) {
	id, err := workItemIDFromWebhook(payload)
	if err != nil {
		return ObservedChange{}, err
	}
	ctx, cancel := requestContext()
	defer cancel()
	if err := p.ensureProjectID(ctx); err != nil {
		return ObservedChange{}, err
	}
	locator := locatorFor(p.settings.Workspace, p.settings.ProjectID, id)
	task, err := p.Load(locator)
	if err != nil {
		return ObservedChange{}, err
	}
	return ObservedChange{
		Event: TaskEventFor(nil, task, taskflow.TaskEventSourcePlaneWebhook),
		Task:  task,
	}, nil
}

// TaskEventFor builds a normalized TaskEvent from a loaded Plane task.
// TaskID is the opaque plane:// locator used with TaskService.Get.
func TaskEventFor(before *tasklifecycle.Task, after tasklifecycle.Task, source taskflow.TaskEventSource) taskflow.TaskEvent {
	if source == "" {
		source = taskflow.TaskEventSourcePlanePoll
	}
	var beforeFlow *taskflow.Task
	if before != nil {
		b := toFlowTask(*before)
		beforeFlow = &b
	}
	return taskflow.NewTaskEvent(taskEventID(after), beforeFlow, toFlowTask(after), source)
}

func taskEventID(task tasklifecycle.Task) string {
	if id := strings.TrimSpace(task.RelativePath); id != "" {
		return id
	}
	return strings.TrimSpace(task.Path)
}

func lifecycleFromFlow(task taskflow.Task) tasklifecycle.Task {
	locator := strings.TrimSpace(task.Locator)
	if locator == "" {
		locator = strings.TrimSpace(task.ID)
	}
	comments := make([]tasklifecycle.Comment, 0, len(task.Comments))
	for _, c := range task.Comments {
		comments = append(comments, tasklifecycle.Comment{
			Author:    c.Author,
			CreatedAt: c.CreatedAt,
			Text:      c.Text,
		})
	}
	return tasklifecycle.Task{
		SchemaVersion:    task.SchemaVersion,
		ID:               task.ID,
		Ref:              task.Ref,
		Title:            task.Title,
		Type:             task.Type,
		Status:           string(task.Status),
		Priority:         task.Priority,
		Project:          task.Project,
		ProjectID:        task.ProjectID,
		WorkspaceID:      task.WorkspaceID,
		Repositories:     append([]string{}, task.Repositories...),
		DependsOn:        append([]string{}, task.DependsOn...),
		Assignee:         task.Assignee,
		AssignmentReason: task.AssignmentReason,
		Source:           task.Source,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
		Summary:          task.Summary,
		Body:             task.Body,
		Comments:         comments,
		BlockedReason:    task.BlockedReason,
		Path:             locator,
		RelativePath:     locator,
	}
}

// pollFingerprint is the subset of task state that matters for pickup/launch
// and dashboard display. Deliberately narrower than a full-struct comparison.
func pollFingerprint(task tasklifecycle.Task) string {
	return strings.Join([]string{
		task.Status,
		strconv.Itoa(task.Priority),
		task.Assignee,
		task.Launch.Agent,
		task.Launch.Mode,
		task.UpdatedAt,
		strings.Join(task.DependsOn, ","),
	}, "|")
}

func workItemIDFromWebhook(payload []byte) (string, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return "", fmt.Errorf("plane: decode webhook payload: %w", err)
	}
	if id := stringField(raw, "id"); looksLikeWorkItemID(id) && !mapHasKey(raw, "event") {
		// Bare work-item body (some delivery modes POST the issue object).
		return id, nil
	}
	for _, key := range []string{"data", "issue", "work_item", "work-item"} {
		if nested, ok := raw[key].(map[string]any); ok {
			if id := extractNestedWorkItemID(nested); id != "" {
				return id, nil
			}
		}
	}
	if id := stringField(raw, "id"); looksLikeWorkItemID(id) {
		return id, nil
	}
	return "", fmt.Errorf("plane: webhook payload has no work-item id")
}

func extractNestedWorkItemID(obj map[string]any) string {
	if id := stringField(obj, "id"); looksLikeWorkItemID(id) {
		return id
	}
	for _, key := range []string{"work_item", "work-item", "issue"} {
		if nested, ok := obj[key].(map[string]any); ok {
			if id := stringField(nested, "id"); looksLikeWorkItemID(id) {
				return id
			}
		}
	}
	return ""
}

func stringField(obj map[string]any, key string) string {
	value, ok := obj[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func mapHasKey(obj map[string]any, key string) bool {
	_, ok := obj[key]
	return ok
}

func looksLikeWorkItemID(id string) bool {
	id = strings.TrimSpace(id)
	return id != "" && !strings.Contains(id, " ")
}
