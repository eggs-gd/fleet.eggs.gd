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
	// DependsOn is a pointer so callers can distinguish omit from clear
	// (empty list). Dashboard/UI use this for hard launch blockers (CORE-148).
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
	var changes []string
	if before.Status != "" && before.Status != after.Status {
		changes = append(changes, "status="+before.Status+"->"+after.Status)
	}
	if before.Assignee != "" && before.Assignee != after.Assignee {
		changes = append(changes, "assignee="+before.Assignee+"->"+after.Assignee)
	}
	if before.Priority != 0 && before.Priority != after.Priority {
		changes = append(changes, "priority="+strconv.Itoa(before.Priority)+"->"+strconv.Itoa(after.Priority))
	}
	beforeProject := firstNonEmpty(before.ProjectID, before.Project)
	afterProject := firstNonEmpty(after.ProjectID, after.Project)
	if beforeProject != "" && beforeProject != afterProject {
		changes = append(changes, "project="+beforeProject+"->"+afterProject)
	}
	if before.RelativePath != "" && after.RelativePath != "" && before.RelativePath != after.RelativePath {
		changes = append(changes, "path="+before.RelativePath+"->"+after.RelativePath)
	}
	if patch.DependsOn != nil {
		beforeDeps := strings.Join(NormalizeDependsOn(before.DependsOn), ",")
		afterDeps := strings.Join(NormalizeDependsOn(after.DependsOn), ",")
		if beforeDeps != afterDeps {
			changes = append(changes, "depends_on="+beforeDeps+"->"+afterDeps)
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
