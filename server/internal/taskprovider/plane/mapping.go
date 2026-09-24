package plane

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// Core <-> Plane priority mapping. Plane's priority enum
// (urgent/high/medium/low/none) is coarser in name but the same cardinality
// as Core's 1-5 scale, so this is a direct, deterministic bijection — no
// project-specific configuration needed, unlike status.
var corePriorityToPlane = map[int]string{1: "urgent", 2: "high", 3: "medium", 4: "low", 5: "none"}
var planePriorityToCore = map[string]int{"urgent": 1, "high": 2, "medium": 3, "low": 4, "none": 5}

func planePriorityForCore(priority int) string {
	if name, ok := corePriorityToPlane[priority]; ok {
		return name
	}
	return "none"
}

func corePriorityForPlane(priority string) int {
	if value, ok := planePriorityToCore[strings.ToLower(strings.TrimSpace(priority))]; ok {
		return value
	}
	return 5
}

// groupFallbackStatus maps a Plane workflow state's *group* (backlog /
// unstarted / started / completed / cancelled — the only classification
// every Plane state is guaranteed to carry) onto the closest Core status.
// Used only when a state's name does not match the configured/identity
// status map, so an unrecognized-but-valid Plane state degrades to a
// reasonable Core status instead of Core failing to load the task at all.
var groupFallbackStatus = map[string]string{
	"backlog":   "backlog",
	"unstarted": "todo",
	"started":   "doing",
	"completed": "done",
	"cancelled": "archived",
}

// effectiveStatusMap returns the Core-status -> Plane-state-name mapping:
// identity by default (a Core status maps to a Plane state literally named
// the same thing, case-insensitively), overridden per status by
// settings.StatusMap. Determinism first (Ground Rule 1): no fuzzy guessing
// beyond the documented group fallback in statusForState.
func (p *Provider) effectiveStatusMap() map[string]string {
	out := make(map[string]string, len(tasklifecycle.TaskStatuses()))
	for _, status := range tasklifecycle.TaskStatuses() {
		out[status] = status
	}
	for status, name := range p.settings.StatusMap {
		if strings.TrimSpace(name) != "" {
			out[status] = name
		}
	}
	return out
}

// stateIDForStatus resolves a Core status to the Plane state UUID to write.
// Returns an error (not a guess) when the target project has no state
// matching the configured/identity name — a missing state is a setup
// problem the operator must fix, not something to paper over.
func (p *Provider) stateIDForStatus(status string, states []stateObj) (string, error) {
	wantName := p.effectiveStatusMap()[status]
	wantLower := strings.ToLower(wantName)
	for _, s := range states {
		if strings.ToLower(s.Name) == wantLower {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("plane: project %s has no workflow state named %q for Core status %q (configure taskProvider.statusMap in %s or rename/add a matching state)", p.settings.ProjectID, wantName, status, "core.config.yaml")
}

// statusForState resolves a Plane work item's state back to a Core status:
// exact name match against the effective status map first, then the state's
// group as a coarse fallback.
func (p *Provider) statusForState(state *stateRef, states []stateObj) string {
	if state == nil {
		return "todo"
	}
	name := state.Name
	group := state.Group
	if name == "" || group == "" {
		for _, s := range states {
			if s.ID == state.ID {
				name, group = s.Name, s.Group
				break
			}
		}
	}

	reverse := map[string]string{}
	for status, mappedName := range p.effectiveStatusMap() {
		reverse[strings.ToLower(mappedName)] = status
	}
	if status, ok := reverse[strings.ToLower(name)]; ok {
		return status
	}
	if status, ok := groupFallbackStatus[strings.ToLower(group)]; ok {
		return status
	}
	return "todo"
}

// Core-specific enum metadata (assignee, task type) has no native Plane
// field that fits a closed worker-identity vocabulary (Plane "assignees" are
// real workspace members/UUIDs, not "claude"/"codex"/"cursor"/"unassigned").
// Rather than requiring a Plane member account per Core worker, both are
// carried as project labels — visible in the Plane UI, queryable, and
// requiring no Plane user-management setup. See the architecture note for
// the alternatives considered (custom fields, a dedicated Core member per
// worker) and why labels won.
const assigneeLabelPrefix = "core:assignee:"
const typeLabelPrefix = "core:type:"

func labelNames(ids []string, labels []labelObj) []string {
	byID := make(map[string]string, len(labels))
	for _, l := range labels {
		byID[l.ID] = l.Name
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := byID[id]; ok {
			names = append(names, name)
		}
	}
	return names
}

func assigneeFromLabels(names []string) string {
	for _, name := range names {
		if value, ok := strings.CutPrefix(name, assigneeLabelPrefix); ok && value != "" {
			return value
		}
	}
	return "unassigned"
}

func typeFromLabels(names []string) string {
	for _, name := range names {
		if value, ok := strings.CutPrefix(name, typeLabelPrefix); ok && value != "" {
			return tasklifecycle.NormalizeTaskType(value)
		}
	}
	return tasklifecycle.DefaultTaskType
}

// ensureLabel returns the id of a label named exactly name (case
// insensitive), creating it in the Plane project if it does not exist yet,
// and returns the (possibly extended) label cache alongside it.
func (p *Provider) ensureLabel(ctx context.Context, name string, labels []labelObj) (string, []labelObj, error) {
	for _, l := range labels {
		if strings.EqualFold(l.Name, name) {
			return l.ID, labels, nil
		}
	}
	created, err := p.client.CreateLabel(ctx, name)
	if err != nil {
		return "", labels, err
	}
	labels = append(labels, *created)
	p.mu.Lock()
	p.labels = labels
	p.labelsAt = time.Now()
	p.mu.Unlock()
	return created.ID, labels, nil
}

// replaceLabelWithPrefix drops any existing label id whose name starts with
// prefix (e.g. a stale "core:assignee:codex" when reassigning to claude) and
// appends newID, leaving every other label on the item untouched.
func replaceLabelWithPrefix(existingIDs []string, prefix string, newID string, labels []labelObj) []string {
	byID := make(map[string]labelObj, len(labels))
	for _, l := range labels {
		byID[l.ID] = l
	}
	out := make([]string, 0, len(existingIDs)+1)
	for _, id := range existingIDs {
		if l, ok := byID[id]; ok && strings.HasPrefix(l.Name, prefix) {
			continue
		}
		out = append(out, id)
	}
	return append(out, newID)
}

func repositoriesFor(repository string) []string {
	if strings.TrimSpace(repository) == "" {
		return []string{}
	}
	return []string{repository}
}

// locatorFor builds the opaque taskprovider locator (Task.RelativePath) for
// a Plane work item — see internal/taskprovider/provider.go's package doc
// for the locator contract every provider must honor.
func locatorFor(workspace string, projectID string, workItemID string) string {
	return "plane://" + workspace + "/" + projectID + "/" + workItemID
}

func workItemIDFromLocator(locator string) (string, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(locator), "plane://")
	if trimmed == "" {
		return "", fmt.Errorf("plane: task locator is required")
	}
	parts := strings.Split(trimmed, "/")
	id := parts[len(parts)-1]
	if id == "" {
		return "", fmt.Errorf("plane: invalid task locator %q", locator)
	}
	return id, nil
}

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

func stripHTML(text string) string {
	return strings.TrimSpace(html.UnescapeString(htmlTagPattern.ReplaceAllString(text, "")))
}

// formatCommentHTML renders a Core review comment the same
// "author: text" convention the Markdown adapter's "## Review Comments"
// section already uses (see tasklifecycle's appendReviewComment), so a
// human reading either backend sees the same authorship convention. Plane
// supplies its own created_at/actor metadata, so unlike the Markdown
// convention this does not also embed a timestamp.
func formatCommentHTML(author string, text string) string {
	author = strings.TrimSpace(author)
	if author == "" {
		author = "alex"
	}
	return "<p>" + html.EscapeString(author) + ": " + html.EscapeString(strings.TrimSpace(text)) + "</p>"
}

func commentsFromPlane(items []workComment) []tasklifecycle.Comment {
	out := make([]tasklifecycle.Comment, 0, len(items))
	for _, item := range items {
		text := stripHTML(item.CommentHTML)
		if text == "" {
			continue
		}
		author, body, ok := strings.Cut(text, ": ")
		if !ok {
			out = append(out, tasklifecycle.Comment{CreatedAt: item.CreatedAt, Text: text})
			continue
		}
		out = append(out, tasklifecycle.Comment{Author: author, CreatedAt: item.CreatedAt, Text: body})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out
}

func firstLine(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
			trimmed = trimmed[:idx]
		}
		return strings.TrimSpace(trimmed)
	}
	return ""
}
