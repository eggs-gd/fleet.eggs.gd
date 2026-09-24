package markdown

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/mdfile"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// IsTaskFile reports whether path points at a canonical task Markdown card
// (Work/<project>/tasks/<file>.md, excluding README.md).
func IsTaskFile(path string) bool {
	return filepath.Ext(path) == ".md" &&
		filepath.Base(path) != "README.md" &&
		filepath.Base(filepath.Dir(path)) == "tasks"
}

// LoadTaskFile reads and parses one canonical task Markdown file into a Task.
func LoadTaskFile(root string, taskFile string) (tasklifecycle.Task, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	taskFile, err = filepath.Abs(taskFile)
	if err != nil {
		return tasklifecycle.Task{}, err
	}

	fm, body, err := mdfile.ReadMarkdown(taskFile)
	if err != nil {
		return tasklifecycle.Task{}, err
	}

	workspaceID := filepath.Base(filepath.Dir(filepath.Dir(taskFile)))
	status := mdfile.Scalar(fm, "status", "todo")
	if !tasklifecycle.IsKnownTaskStatus(status) {
		status = "todo"
	}
	rel, _ := filepath.Rel(root, taskFile)
	slug := strings.TrimSuffix(filepath.Base(taskFile), filepath.Ext(taskFile))
	project := mdfile.Scalar(fm, "project", workspaceID)
	repositories := mdfile.List(fm, "repositories")
	dependsOn := tasklifecycle.NormalizeDependsOn(mdfile.List(fm, "depends_on"))
	projectID := tasklifecycle.ResolveTaskProjectID(workspaceID, project, repositories)

	task := tasklifecycle.Task{
		SchemaVersion:    mdfile.IntValue(fm, "schema_version", 0),
		ID:               mdfile.Scalar(fm, "id", slug),
		Ref:              mdfile.Scalar(fm, "ref", ""),
		Title:            mdfile.Scalar(fm, "title", slug),
		Type:             tasklifecycle.NormalizeTaskType(mdfile.Scalar(fm, "type", tasklifecycle.DefaultTaskType)),
		Status:           status,
		Priority:         taskPriority(fm),
		Project:          project,
		ProjectID:        projectID,
		WorkspaceID:      workspaceID,
		Repositories:     repositories,
		DependsOn:        dependsOn,
		Assignee:         mdfile.Scalar(fm, "assignee", "unassigned"),
		AssignmentReason: mdfile.Scalar(fm, "assignment_reason", ""),
		Source:           mdfile.Scalar(fm, "source", ""),
		CreatedAt:        mdfile.Scalar(fm, "created_at", ""),
		UpdatedAt:        mdfile.Scalar(fm, "updated_at", ""),
		Launch: tasklifecycle.Launch{
			Agent:      mdfile.Scalar(fm, "launch.agent", ""),
			Mode:       mdfile.Scalar(fm, "launch.mode", ""),
			AutoCommit: mdfile.Boolean(fm, "launch.auto_commit"),
			AutoPush:   mdfile.Boolean(fm, "launch.auto_push"),
			AutoPR:     mdfile.Boolean(fm, "launch.auto_pr"),
		},
		Summary:  mdfile.FirstParagraph(body),
		Body:     body,
		Comments: parseReviewComments(body),
		Path:     taskFile,
		RelativePath: filepath.ToSlash(rel),
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent: firstNonEmpty(mdfile.Scalar(fm, "launch.agent", ""), mdfile.Scalar(fm, "assignee", "unassigned")),
		},
	}
	task.BlockedReason = tasklifecycle.DeriveBlockedReason(task)
	return task, nil
}

func taskPriority(fm mdfile.Frontmatter) int {
	value := mdfile.IntValue(fm, "priority", 5)
	if value < 1 || value > 5 {
		return 5
	}
	return value
}

func parseReviewComments(body string) []tasklifecycle.Comment {
	section := markdownSection(body, "Review Comments")
	if section == "" {
		return []tasklifecycle.Comment{}
	}

	comments := []tasklifecycle.Comment{}
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		raw := strings.TrimPrefix(line, "- ")
		createdAt, rest, ok := strings.Cut(raw, " — ")
		if !ok {
			comments = append(comments, tasklifecycle.Comment{Text: raw})
			continue
		}
		author, text, ok := strings.Cut(rest, ": ")
		if !ok {
			comments = append(comments, tasklifecycle.Comment{CreatedAt: createdAt, Text: rest})
			continue
		}
		comments = append(comments, tasklifecycle.Comment{
			Author:    author,
			CreatedAt: createdAt,
			Text:      text,
		})
	}
	return comments
}

func markdownSection(body string, title string) string {
	lines := strings.Split(body, "\n")
	heading := "## " + title
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}

	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			end = i
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func formatComment(author string, text string) string {
	author = strings.TrimSpace(author)
	if author == "" {
		author = "alex"
	}
	text = mdfile.Whitespace.ReplaceAllString(strings.TrimSpace(text), " ")
	return "- " + time.Now().Format(time.RFC3339) + " — " + author + ": " + text
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
