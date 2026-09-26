package tasklifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

var workRefCounterMu sync.Mutex

// DefaultTaskType is used when frontmatter/API omit type. "todo" is a status,
// not a task type — legacy type:todo values normalize to this.
const DefaultTaskType = "feature"

var taskTypes = map[string]bool{
	"feature":     true,
	"bug":         true,
	"research":    true,
	"review":      true,
	"maintenance": true,
	"decision":    true,
}

// NormalizeTaskType maps empty and legacy "todo" types to DefaultTaskType.
// Status "todo" is unrelated and must not be passed here.
func NormalizeTaskType(taskType string) string {
	taskType = strings.TrimSpace(taskType)
	if taskType == "" || taskType == "todo" {
		return DefaultTaskType
	}
	return taskType
}

var taskAssignees = map[string]bool{
	"alex":       true,
	"owner":      true,
	"codex":      true,
	"claude":     true,
	"cursor":     true,
	"gemini":     true,
	"unassigned": true,
}

// TaskCreateRequest is the dashboard/API payload for creating a new Work item.
type TaskCreateRequest struct {
	Title            string `json:"title"`
	Request          string `json:"request"`
	Project          string `json:"project"`
	Repository       string `json:"repository"`
	Status           string `json:"status"`
	Type             string `json:"type"`
	Assignee         string `json:"assignee"`
	Priority         *int   `json:"priority"`
	AssignmentReason string `json:"assignment_reason"`
	// DependsOn lists prerequisite refs that must be done before launch
	// (CORE-148). Optional; empty means no hard dependency gate.
	DependsOn []string `json:"depends_on"`
}

type workRefCounters struct {
	WorkRefPrefix  string   `json:"work_ref_prefix"`
	NextWorkRef    int      `json:"next_work_ref"`
	InboxRefPrefix string   `json:"inbox_ref_prefix"`
	NextInboxRef   int      `json:"next_inbox_ref"`
	LifeRefPrefix  string   `json:"life_ref_prefix"`
	NextLifeRef    int      `json:"next_life_ref"`
	Notes          []string `json:"notes"`
}

// NormalizeTaskCreateRequest validates and fills in defaults for a
// TaskCreateRequest (title/status/type/assignee/priority/assignment_reason).
// It is the provider-neutral half of task creation: both the Markdown
// adapter and the Plane adapter call this so create validation rules never
// drift between providers. It does not touch storage.
func NormalizeTaskCreateRequest(req TaskCreateRequest) (TaskCreateRequest, error) {
	return normalizeTaskCreateRequest(req)
}

func normalizeTaskCreateRequest(req TaskCreateRequest) (TaskCreateRequest, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Request = strings.TrimSpace(req.Request)
	req.Project = strings.TrimSpace(req.Project)
	req.Repository = strings.TrimSpace(req.Repository)
	req.Status = strings.TrimSpace(req.Status)
	req.Type = strings.TrimSpace(req.Type)
	req.Assignee = strings.TrimSpace(req.Assignee)
	req.AssignmentReason = strings.TrimSpace(req.AssignmentReason)

	if req.Title == "" {
		return TaskCreateRequest{}, fmt.Errorf("title is required")
	}
	if req.Request == "" {
		return TaskCreateRequest{}, fmt.Errorf("request/description is required")
	}
	if req.Project == "" {
		return TaskCreateRequest{}, fmt.Errorf("project is required")
	}
	if strings.Contains(req.Project, "..") || strings.ContainsAny(req.Project, `\:`) {
		return TaskCreateRequest{}, fmt.Errorf("project id %q is invalid", req.Project)
	}

	if req.Status == "" {
		req.Status = "backlog"
	}
	if !taskStatusSet[req.Status] {
		return TaskCreateRequest{}, fmt.Errorf("unknown task status %q", req.Status)
	}

	req.Type = NormalizeTaskType(req.Type)
	if !taskTypes[req.Type] {
		return TaskCreateRequest{}, fmt.Errorf("unknown task type %q", req.Type)
	}

	if req.Assignee == "" {
		req.Assignee = "unassigned"
	}
	if !taskAssignees[req.Assignee] {
		return TaskCreateRequest{}, fmt.Errorf("unknown assignee %q", req.Assignee)
	}

	priority := 5
	if req.Priority != nil {
		priority = *req.Priority
	}
	if priority < 1 || priority > 5 {
		return TaskCreateRequest{}, fmt.Errorf("unknown task priority %d", priority)
	}
	req.Priority = &priority

	if req.AssignmentReason == "" {
		if req.Assignee == "unassigned" {
			req.AssignmentReason = "Created from the backoffice dashboard; left unassigned until Alex routes it."
		} else {
			req.AssignmentReason = fmt.Sprintf("Assigned to %s from the backoffice dashboard.", req.Assignee)
		}
	}

	req.DependsOn = NormalizeDependsOn(req.DependsOn)

	return req, nil
}

// AllocateNextWorkRef allocates the next Core-wide CORE-N ref from
// _registry/counters.json. Refs are Core-wide, not provider-scoped, so every
// provider that creates tasks against the same Core root must allocate from
// this single counter.
func AllocateNextWorkRef(root string) (string, error) {
	return allocateRef(root, func(c *workRefCounters) (string, string, *int) {
		return c.WorkRefPrefix, "CORE", &c.NextWorkRef
	}, "next_work_ref")
}

// AllocateNextInboxRef allocates the next INBOX-N ref from the same counters
// file.
func AllocateNextInboxRef(root string) (string, error) {
	return allocateRef(root, func(c *workRefCounters) (string, string, *int) {
		return c.InboxRefPrefix, "INBOX", &c.NextInboxRef
	}, "next_inbox_ref")
}

func allocateRef(root string, pick func(*workRefCounters) (prefix, fallback string, next *int), field string) (string, error) {
	workRefCounterMu.Lock()
	defer workRefCounterMu.Unlock()

	path := filepath.Join(root, "_registry", "counters.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read counters: %w", err)
	}

	var counters workRefCounters
	if err := json.Unmarshal(data, &counters); err != nil {
		return "", fmt.Errorf("parse counters: %w", err)
	}
	prefix, fallback, next := pick(&counters)
	if *next < 1 {
		return "", fmt.Errorf("counters %s must be >= 1", field)
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = fallback
	}

	ref := fmt.Sprintf("%s-%d", prefix, *next)
	*next++

	encoded, err := json.MarshalIndent(counters, "", "  ")
	if err != nil {
		return "", err
	}
	encoded = append(encoded, '\n')
	if err := writeCountersAtomic(path, encoded); err != nil {
		return "", fmt.Errorf("write counters: %w", err)
	}
	return ref, nil
}

func writeCountersAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(0o644); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// SlugifyTitle turns a title into a short lowercase ASCII slug. It returns
// "" when the title has no ASCII letters or digits.
func SlugifyTitle(title string) string {
	lower := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	lastDash := false
	for _, r := range lower {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	slug = nonSlugChars.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	parts := strings.Split(slug, "-")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		kept = append(kept, part)
		if len(kept) >= 8 {
			break
		}
	}
	slug = strings.Join(kept, "-")
	if len(slug) > 72 {
		slug = strings.Trim(slug[:72], "-")
	}
	return slug
}
