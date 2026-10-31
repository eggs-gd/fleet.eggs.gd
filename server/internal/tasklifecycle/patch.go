package tasklifecycle

import (
	"strconv"
	"strings"
)

// TaskPatch is a partial update applied to one task.
type TaskPatch struct {
	Path       string `json:"path"`
	Status     string `json:"status"`
	Priority   *int   `json:"priority"`
	Assignee   string `json:"assignee"`
	Project    string `json:"project"`
	Repository string `json:"repository"`
	// Milestone is a pointer so callers can distinguish omit from clear: nil
	// leaves it unchanged, &"" clears it, &id sets it. A plain empty string
	// can't express "clear" the way it does for Assignee/Project, which are
	// never legitimately empty on a real task.
	Milestone *string `json:"milestone"`
	// DependsOn is a pointer so callers can distinguish omit from clear
	// (empty list). Dashboard/UI use this for hard launch blockers.
	DependsOn           *[]string `json:"depends_on"`
	Body                *string   `json:"body"`
	Comment             string    `json:"comment"`
	CommentAuthor       string    `json:"comment_author"`
	ExpectedUpdatedAt   string    `json:"expected_updated_at"`
	ExpectedHash        string    `json:"expected_hash"`
	AllowStatusOverride bool      `json:"-"`
}

// TaskCreate writes a brand-new task card verbatim at Path (Markdown provider).
// Plane ignores Path/Content and uses its own create API.
type TaskCreate struct {
	Path    string
	Content string
}

// TaskPatchChangeSummary describes what a patch actually changed, for
// logging/event-detail purposes.
func TaskPatchChangeSummary(before Task, after Task, patch TaskPatch) []string {
	beforeProject := firstNonEmpty(before.ProjectID, before.Project)
	afterProject := firstNonEmpty(after.ProjectID, after.Project)
	beforeDeps := strings.Join(NormalizeDependsOn(before.DependsOn), ",")
	afterDeps := strings.Join(NormalizeDependsOn(after.DependsOn), ",")
	rows := []struct {
		label, from, to string
		changed         bool
	}{
		{"status", before.Status, after.Status, before.Status != "" && before.Status != after.Status},
		{"assignee", before.Assignee, after.Assignee, before.Assignee != "" && before.Assignee != after.Assignee},
		{"priority", strconv.Itoa(before.Priority), strconv.Itoa(after.Priority), before.Priority != 0 && before.Priority != after.Priority},
		{"project", beforeProject, afterProject, beforeProject != "" && beforeProject != afterProject},
		{"path", before.RelativePath, after.RelativePath, before.RelativePath != "" && after.RelativePath != "" && before.RelativePath != after.RelativePath},
		{"milestone", before.Milestone, after.Milestone, patch.Milestone != nil && before.Milestone != after.Milestone},
		{"depends_on", beforeDeps, afterDeps, patch.DependsOn != nil && beforeDeps != afterDeps},
	}
	var changes []string
	for _, row := range rows {
		if row.changed {
			changes = append(changes, row.label+"="+row.from+"->"+row.to)
		}
	}
	if patch.Body != nil {
		changes = append(changes, "body_changed=true")
	}
	if strings.TrimSpace(patch.Comment) != "" {
		changes = append(changes, "comment_added=true")
	}
	return changes
}
