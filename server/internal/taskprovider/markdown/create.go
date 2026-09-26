package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func (p *Provider) CreateFromRequest(req tasklifecycle.TaskCreateRequest) (tasklifecycle.Task, error) {
	normalized, err := tasklifecycle.NormalizeTaskCreateRequest(req)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	workProjectID := workProjectDirID(normalized.Project)
	if err := validateProjectForCreate(p.root, workProjectID); err != nil {
		return tasklifecycle.Task{}, err
	}
	// Canonical task cards live under Work/<workspace-folder>/tasks even when the
	// dashboard project picker exposes a nested project id.
	normalized.Project = workProjectID

	ref, err := tasklifecycle.AllocateNextWorkRef(p.root)
	if err != nil {
		return tasklifecycle.Task{}, err
	}

	now := time.Now()
	dateStamp := now.Format("2006-01-02")
	slug := tasklifecycle.SlugifyTitle(normalized.Title)
	if slug == "" {
		slug = "task"
	}
	fileBase, err := uniqueTaskFileBase(p.root, normalized.Project, dateStamp, slug)
	if err != nil {
		return tasklifecycle.Task{}, err
	}

	relativePath := filepath.ToSlash(filepath.Join("Work", normalized.Project, "tasks", fileBase+".md"))
	content := renderCreatedTaskMarkdown(normalized, ref, fileBase, now)

	return p.CreateFile(tasklifecycle.TaskCreate{
		Path:    relativePath,
		Content: content,
	})
}

func workProjectDirID(projectID string) string {
	projectID = strings.Trim(strings.TrimSpace(projectID), "/")
	if projectID == "" {
		return ""
	}
	if i := strings.IndexByte(projectID, '/'); i >= 0 {
		return projectID[:i]
	}
	return projectID
}

func validateProjectForCreate(root string, projectID string) error {
	projectDir := filepath.Join(root, "Work", projectID)
	info, err := os.Stat(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project %q does not exist under Work/", projectID)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("project path Work/%s is not a directory", projectID)
	}
	projectCard := filepath.Join(projectDir, "PROJECT.md")
	if _, err := os.Stat(projectCard); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project %q is missing PROJECT.md", projectID)
		}
		return err
	}
	return nil
}

func uniqueTaskFileBase(root string, projectID string, dateStamp string, slug string) (string, error) {
	tasksDir := filepath.Join(root, "Work", projectID, "tasks")
	base := dateStamp + "-" + slug
	candidate := base
	for i := 2; ; i++ {
		path := filepath.Join(tasksDir, candidate+".md")
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
		if i > 1000 {
			return "", fmt.Errorf("could not allocate unique task filename for %s", base)
		}
	}
}

func renderCreatedTaskMarkdown(req tasklifecycle.TaskCreateRequest, ref string, fileBase string, now time.Time) string {
	stamp := now.Format(time.RFC3339)
	priority := 5
	if req.Priority != nil {
		priority = *req.Priority
	}

	repoLines := "repositories: []"
	matchedRepo := "(none selected)"
	if req.Repository != "" {
		repoLines = "repositories:\n  - " + quoteFrontmatter(req.Repository)
		matchedRepo = "`" + req.Repository + "`"
	}

	requestBody := req.Request
	var body strings.Builder
	body.WriteString("---\n")
	body.WriteString("schema_version: 1\n")
	body.WriteString("id: work-" + fileBase + "\n")
	body.WriteString("ref: " + ref + "\n")
	body.WriteString("title: " + quoteFrontmatter(req.Title) + "\n")
	body.WriteString("type: " + req.Type + "\n")
	body.WriteString("status: " + req.Status + "\n")
	body.WriteString(fmt.Sprintf("priority: %d\n", priority))
	body.WriteString("project: " + req.Project + "\n")
	body.WriteString(repoLines + "\n")
	if len(req.DependsOn) > 0 {
		body.WriteString("depends_on:\n")
		for _, dep := range req.DependsOn {
			body.WriteString("  - " + quoteFrontmatter(dep) + "\n")
		}
	} else {
		body.WriteString("depends_on: []\n")
	}
	body.WriteString("assignee: " + req.Assignee + "\n")
	body.WriteString("assignment_reason: " + quoteFrontmatter(req.AssignmentReason) + "\n")
	body.WriteString("source: text\n")
	body.WriteString("source_inbox:\n")
	body.WriteString("created_at: " + stamp + "\n")
	body.WriteString("updated_at: " + stamp + "\n")
	body.WriteString("launch:\n")
	body.WriteString("  agent:\n")
	body.WriteString("  mode:\n")
	body.WriteString("  auto_commit: false\n")
	body.WriteString("  auto_push: false\n")
	body.WriteString("  auto_pr: false\n")
	body.WriteString("---\n\n")
	body.WriteString("# " + req.Title + "\n\n")
	body.WriteString("## Request\n\n")
	body.WriteString(requestBody + "\n\n")
	body.WriteString("## Raw Input\n\n")
	body.WriteString(requestBody + "\n\n")
	body.WriteString("## Project Resolution\n\n")
	body.WriteString("- Matched workspace: `" + req.Project + "`\n")
	body.WriteString("- Matched repository: " + matchedRepo + "\n")
	body.WriteString("- Resolution method: backoffice dashboard\n")
	body.WriteString("- Confidence: high\n\n")
	body.WriteString("## Acceptance Criteria\n\n")
	body.WriteString("- [ ] Describe the expected outcome.\n\n")
	body.WriteString("## Context\n\n")
	body.WriteString("- Workspace: `Work/" + req.Project + "/PROJECT.md`\n")
	if req.Repository != "" {
		body.WriteString("- Repository: `" + req.Repository + "`\n")
	}
	body.WriteString("\n## Deliverable\n\n")
	body.WriteString("Expected physical artifact:\n\n")
	body.WriteString("- Pending.\n\n")
	body.WriteString("Produced artifacts:\n\n")
	body.WriteString("- Pending.\n\n")
	body.WriteString("## Handoff\n\n")
	body.WriteString("Created from the backoffice dashboard. Refine acceptance criteria and\n")
	body.WriteString("priority before moving out of backlog when needed.\n\n")
	body.WriteString("Assignment reason: " + req.AssignmentReason + "\n\n")
	body.WriteString("Launch policy: `todo` and `needs_rework` tasks are daemon-pickup candidates\n")
	body.WriteString("when they have a supported assignee and exactly one repository. Leave\n")
	body.WriteString("`launch.agent` empty unless this task intentionally launches a different worker\n")
	body.WriteString("than `assignee`. Dashboard creation does not trigger launch, commit, push, or PR.\n\n")
	body.WriteString("## Review Comments\n\n")
	body.WriteString("## Activity Log\n\n")
	body.WriteString("- " + stamp + " — Created from backoffice dashboard.\n")
	return body.String()
}
