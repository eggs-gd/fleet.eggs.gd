package plane

import (
	"context"
	"fmt"
	"html"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/projecttag"
	"github.com/eggs-gd/fleet.eggs.gd/internal/providerconfig"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
)

type (
	Task              = tasklifecycle.Task
	TaskPatch         = tasklifecycle.TaskPatch
	TaskCreateRequest = tasklifecycle.TaskCreateRequest
)

// requestTimeout bounds every Plane API call this provider makes. Chosen to
// comfortably exceed normal latency while still failing a hung request
// before it blocks a daemon mutation/launch path indefinitely.
const requestTimeout = 20 * time.Second

// cacheTTL bounds how long states/labels are trusted between refetches.
// Both change rarely (an operator editing project workflow/labels in the
// Plane UI) relative to Core's task pickup cadence, so caching them avoids
// spending a meaningful fraction of Plane's 60 req/min budget on metadata
// that is not the thing actually changing.
const cacheTTL = 60 * time.Second

// Provider implements taskprovider.Provider against one Plane
// workspace/project, per providerconfig.PlaneSettings. Use Flow() for the
// taskflow.TaskProvider application contract (CORE-106). Setup and metadata
// placement: _docs/PLANE_TASK_PROVIDER.md.
//
// Provider also implements taskprovider.ChangeSource so Runtime can attach
// the listener without Type() or *plane.Provider branches (CORE-107).
type Provider struct {
	client   *Client
	settings providerconfig.PlaneSettings
	coreRoot string

	mu       sync.Mutex
	states   []stateObj
	statesAt time.Time
	labels   []labelObj
	labelsAt time.Time

	pollMu      sync.Mutex
	pollTracker *PollTracker
}

// New builds a Plane-backed taskprovider.Provider. coreRoot is needed only
// to allocate refs from the tag of the mapped Core project and the same
// _registry/counters.json the Markdown adapter uses (refs must never collide
// between providers — see tasklifecycle.AllocateNextRef).
func New(settings providerconfig.PlaneSettings, apiToken string, coreRoot string) *Provider {
	client := NewClient(ClientConfig{
		BaseURL:   settings.BaseURL,
		Workspace: settings.Workspace,
		ProjectID: settings.ProjectID,
		APIToken:  apiToken,
	})
	return &Provider{client: client, settings: settings, coreRoot: coreRoot}
}

var (
	_ taskprovider.Provider     = (*Provider)(nil)
	_ taskprovider.ChangeSource = (*Provider)(nil)
)

func (p *Provider) Type() string { return "plane" }

// ObserveChanges implements taskprovider.ChangeSource. Dedup state lives on
// the provider so corechain never imports plane.PollTracker.
func (p *Provider) ObserveChanges() ([]taskprovider.ObservedChange, error) {
	p.pollMu.Lock()
	defer p.pollMu.Unlock()
	if p.pollTracker == nil {
		p.pollTracker = NewPollTracker()
	}

	changes, err := p.ObservePoll(p.pollTracker)
	if err != nil {
		return nil, err
	}
	out := make([]taskprovider.ObservedChange, 0, len(changes))
	for _, change := range changes {
		out = append(out, taskprovider.ObservedChange{
			Event: change.Event,
			Task:  change.Task,
		})
	}
	return out, nil
}

func requestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), requestTimeout)
}

// ensureProjectID resolves p.client.projectID by matching settings.CoreProject
// against the workspace's project list (case-insensitively, by Plane
// identifier or name), when config did not pin a raw project UUID. An
// explicit settings.ProjectID always wins and skips the lookup entirely.
//
// Resolution is cached for the lifetime of the Provider once it succeeds —
// nothing in Plane's own workspace/project identifiers is expected to
// change without an operator noticing and restarting the daemon.
func (p *Provider) ensureProjectID(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client.projectID != "" {
		return nil
	}
	projects, err := p.client.ListProjects(ctx)
	if err != nil {
		return fmt.Errorf("plane: resolve project for core project %q: %w", p.settings.CoreProject, err)
	}
	for _, project := range projects {
		if strings.EqualFold(project.Identifier, p.settings.CoreProject) || strings.EqualFold(project.Name, p.settings.CoreProject) {
			// Keep both in sync: client.projectID drives request URLs,
			// settings.ProjectID is what locatorFor/RelativePath and error
			// messages elsewhere in this package read directly.
			p.client.projectID = project.ID
			p.settings.ProjectID = project.ID
			return nil
		}
	}
	return fmt.Errorf("plane: no project in workspace %q matches core project %q by identifier or name — pin taskProvider.project explicitly or rename the Plane project", p.settings.Workspace, p.settings.CoreProject)
}

// List returns every Plane work item Core created (or synced) in the
// configured project. Comments are fetched only for "blocked" tasks — the
// one status where Core actually reads Comments to derive BlockedReason for
// the dashboard/pickup surfaces — to keep a full List() call within Plane's
// 60 requests/minute budget regardless of board size. A single-task detail
// view always has full comments via Load.
func (p *Provider) List() ([]Task, error) {
	ctx, cancel := requestContext()
	defer cancel()

	if err := p.ensureProjectID(ctx); err != nil {
		return nil, err
	}

	items, err := p.client.ListWorkItems(ctx)
	if err != nil {
		return nil, err
	}
	states, err := p.cachedStates(ctx)
	if err != nil {
		return nil, err
	}
	labels, err := p.cachedLabels(ctx)
	if err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(items))
	for _, item := range items {
		var comments []workComment
		if p.statusForState(item.State, states) == "blocked" {
			comments, err = p.client.ListComments(ctx, item.ID)
			if err != nil {
				return nil, err
			}
		}
		tasks = append(tasks, p.taskFromWorkItem(item, states, labels, comments))
	}
	sort.Slice(tasks, func(i, j int) bool { return tasklifecycle.TaskPickupLess(tasks[i], tasks[j]) })
	return tasks, nil
}

// Load returns one task with full comment history, by its
// "plane://workspace/project/work-item-id" locator.
func (p *Provider) Load(locator string) (Task, error) {
	id, err := workItemIDFromLocator(locator)
	if err != nil {
		return Task{}, err
	}
	ctx, cancel := requestContext()
	defer cancel()

	if err := p.ensureProjectID(ctx); err != nil {
		return Task{}, err
	}

	item, err := p.client.GetWorkItem(ctx, id)
	if err != nil {
		return Task{}, err
	}
	states, err := p.cachedStates(ctx)
	if err != nil {
		return Task{}, err
	}
	labels, err := p.cachedLabels(ctx)
	if err != nil {
		return Task{}, err
	}
	comments, err := p.client.ListComments(ctx, item.ID)
	if err != nil {
		return Task{}, err
	}
	return p.taskFromWorkItem(*item, states, labels, comments), nil
}

// CreateFromRequest validates req with the same rules the Markdown adapter
// uses (tasklifecycle.NormalizeTaskCreateRequest), allocates a ref from the
// tag of the Core project this Plane project is mapped to, and creates the corresponding Plane work item with that ref recorded
// in external_id/external_source.
func (p *Provider) CreateFromRequest(req TaskCreateRequest) (Task, error) {
	normalized, err := tasklifecycle.NormalizeTaskCreateRequest(req)
	if err != nil {
		return Task{}, err
	}
	if !tasklifecycle.KnownAssignee(p.coreRoot, normalized.Assignee) {
		return Task{}, fmt.Errorf("unknown assignee %q: use unassigned, an agent, or a person with a Fleet/<name>.md file", normalized.Assignee)
	}
	if normalized.Project != "" && !strings.EqualFold(normalized.Project, p.settings.CoreProject) {
		return Task{}, fmt.Errorf("plane: project %q is not served by this provider (configured for %q)", normalized.Project, p.settings.CoreProject)
	}

	tag, err := projecttag.ForProject(p.coreRoot, p.settings.CoreProject)
	if err != nil {
		return Task{}, err
	}
	ref, err := tasklifecycle.AllocateNextRef(p.coreRoot, tag)
	if err != nil {
		return Task{}, err
	}

	ctx, cancel := requestContext()
	defer cancel()

	if err := p.ensureProjectID(ctx); err != nil {
		return Task{}, err
	}

	states, err := p.cachedStates(ctx)
	if err != nil {
		return Task{}, err
	}
	stateID, err := p.stateIDForStatus(normalized.Status, states)
	if err != nil {
		return Task{}, err
	}
	labels, err := p.cachedLabels(ctx)
	if err != nil {
		return Task{}, err
	}
	assigneeLabelID, labels, err := p.ensureLabel(ctx, assigneeLabelPrefix+normalized.Assignee, labels)
	if err != nil {
		return Task{}, err
	}
	typeLabelID, labels, err := p.ensureLabel(ctx, typeLabelPrefix+normalized.Type, labels)
	if err != nil {
		return Task{}, err
	}

	description := normalized.Request
	if normalized.AssignmentReason != "" {
		description += "\n\nAssignment reason: " + normalized.AssignmentReason
	}
	priority := 5
	if normalized.Priority != nil {
		priority = *normalized.Priority
	}

	payload := map[string]any{
		"name":                 normalized.Title,
		"description_stripped": description,
		"description_html":     "<p>" + htmlParagraphs(description) + "</p>",
		"priority":             planePriorityForCore(priority),
		"state":                stateID,
		"external_id":          ref,
		"external_source":      externalSource,
		"labels":               []string{assigneeLabelID, typeLabelID},
	}
	item, err := p.client.CreateWorkItem(ctx, payload)
	if err != nil {
		return Task{}, err
	}
	return p.taskFromWorkItem(*item, states, labels, nil), nil
}

// Mutate applies a status transition, priority change, assignee
// reassignment, review comment, and/or body replacement to one Plane work
// item. Status transition legality is enforced with the identical
// tasklifecycle rules the Markdown adapter uses, so "which transitions are
// allowed" never depends on which provider is active.
func (p *Provider) Mutate(patch TaskPatch) (Task, error) {
	id, err := workItemIDFromLocator(patch.Path)
	if err != nil {
		return Task{}, err
	}
	ctx, cancel := requestContext()
	defer cancel()

	if err := p.ensureProjectID(ctx); err != nil {
		return Task{}, err
	}

	current, err := p.client.GetWorkItem(ctx, id)
	if err != nil {
		return Task{}, err
	}
	states, err := p.cachedStates(ctx)
	if err != nil {
		return Task{}, err
	}
	labels, err := p.cachedLabels(ctx)
	if err != nil {
		return Task{}, err
	}

	payload := map[string]any{}
	if patch.Status != "" {
		currentStatus := p.statusForState(current.State, states)
		if !patch.AllowStatusOverride && !tasklifecycle.AllowedTaskStatusTransition(currentStatus, patch.Status) {
			return Task{}, fmt.Errorf("%w: %s -> %s", tasklifecycle.ErrTaskTransitionRejected, currentStatus, patch.Status)
		}
		stateID, err := p.stateIDForStatus(patch.Status, states)
		if err != nil {
			return Task{}, err
		}
		payload["state"] = stateID
	}
	if patch.Priority != nil {
		if *patch.Priority < 1 || *patch.Priority > 5 {
			return Task{}, fmt.Errorf("unknown task priority %d", *patch.Priority)
		}
		payload["priority"] = planePriorityForCore(*patch.Priority)
	}
	if patch.Assignee != "" {
		assigneeLabelID, updatedLabels, err := p.ensureLabel(ctx, assigneeLabelPrefix+patch.Assignee, labels)
		if err != nil {
			return Task{}, err
		}
		labels = updatedLabels
		payload["labels"] = replaceLabelWithPrefix(current.Labels, assigneeLabelPrefix, assigneeLabelID, labels)
	}
	if strings.TrimSpace(patch.Project) != "" {
		return Task{}, fmt.Errorf("plane provider does not support moving tasks between projects")
	}
	if patch.Body != nil {
		payload["description_stripped"] = *patch.Body
		payload["description_html"] = "<p>" + htmlParagraphs(*patch.Body) + "</p>"
	}
	// DependsOn is Markdown-backed for CORE-148; Plane has no durable field yet.
	// Ignore the patch key so status/assignee updates from TaskService still work.

	updated := current
	if len(payload) > 0 {
		updated, err = p.client.UpdateWorkItem(ctx, id, payload)
		if err != nil {
			return Task{}, err
		}
	}
	if strings.TrimSpace(patch.Comment) != "" {
		if err := p.client.AddComment(ctx, id, formatCommentHTML(patch.CommentAuthor, patch.Comment)); err != nil {
			return Task{}, err
		}
	}

	comments, err := p.client.ListComments(ctx, id)
	if err != nil {
		return Task{}, err
	}
	return p.taskFromWorkItem(*updated, states, labels, comments), nil
}

// Claim mirrors MarkdownProvider.Claim semantics exactly (reject
// unless the task is in a launchable pickup status, then transition to
// "doing"), reusing the same status set and sentinel error so a caller
// cannot tell from error type/behavior which provider rejected the claim.
func (p *Provider) Claim(locator string) (Task, error) {
	current, err := p.Load(locator)
	if err != nil {
		return Task{}, err
	}
	if !tasklifecycle.LaunchablePickupStatus(current.Status) {
		return Task{}, fmt.Errorf("%w: status is %q", tasklifecycle.ErrTaskAlreadyClaimed, current.Status)
	}
	return p.Mutate(TaskPatch{Path: locator, Status: "doing"})
}

func (p *Provider) cachedStates(ctx context.Context) ([]stateObj, error) {
	p.mu.Lock()
	if p.states != nil && time.Since(p.statesAt) < cacheTTL {
		states := p.states
		p.mu.Unlock()
		return states, nil
	}
	p.mu.Unlock()

	states, err := p.client.ListStates(ctx)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	p.states, p.statesAt = states, time.Now()
	p.mu.Unlock()
	return states, nil
}

func (p *Provider) cachedLabels(ctx context.Context) ([]labelObj, error) {
	p.mu.Lock()
	if p.labels != nil && time.Since(p.labelsAt) < cacheTTL {
		labels := p.labels
		p.mu.Unlock()
		return labels, nil
	}
	p.mu.Unlock()

	labels, err := p.client.ListLabels(ctx)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	p.labels, p.labelsAt = labels, time.Now()
	p.mu.Unlock()
	return labels, nil
}

func (p *Provider) taskFromWorkItem(item workItem, states []stateObj, labels []labelObj, comments []workComment) Task {
	names := labelNames(item.Labels, labels)
	status := p.statusForState(item.State, states)
	assignee := assigneeFromLabels(names)

	task := Task{
		SchemaVersion: 1,
		ID:            "plane:" + item.ID,
		Ref:           item.ExternalID,
		Title:         item.Name,
		Type:          typeFromLabels(names),
		Status:        status,
		Priority:      corePriorityForPlane(item.Priority),
		Project:       p.settings.CoreProject,
		ProjectID:     p.settings.CoreProject,
		WorkspaceID:   p.settings.CoreProject,
		Repositories:  repositoriesFor(p.settings.Repository),
		Assignee:      assignee,
		Source:        "plane",
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
		Summary:       firstLine(item.DescriptionStripped, item.Name),
		Body:          item.DescriptionStripped,
		Comments:      commentsFromPlane(comments),
		Path:          "",
		RelativePath:  locatorFor(p.settings.Workspace, p.settings.ProjectID, item.ID),
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent: assignee,
		},
	}
	if status == "blocked" {
		task.BlockedReason = tasklifecycle.DeriveBlockedReason(task)
	}
	return task
}

func htmlParagraphs(text string) string {
	escaped := html.EscapeString(strings.TrimSpace(text))
	return strings.ReplaceAll(escaped, "\n", "<br/>")
}
