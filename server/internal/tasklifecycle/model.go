// Package tasklifecycle owns the task lifecycle domain: the Task model,
// status transitions, pickup/priority ordering, create-request normalization,
// launch validation gates, execution-to-task finalization decisions, and
// task-facing provider errors.
//
// Core does not permanently own tasks; it orchestrates them. Markdown file
// I/O lives in internal/taskprovider/markdown (CORE-102). Domain rules stay
// here so every provider shares identical lifecycle semantics.
//
// corechain (the daemon/runtime package) depends on tasklifecycle, never the
// other way around — that keeps the lifecycle rules provider-agnostic and
// testable without daemon/session/dashboard plumbing.
package tasklifecycle

import (
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/mdfile"
)

var taskStatuses = []string{"backlog", "needs_rework", "todo", "doing", "blocked", "needs_review", "done", "archived"}
var taskStatusSet = map[string]bool{
	"backlog":      true,
	"needs_rework": true,
	"todo":         true,
	"doing":        true,
	"blocked":      true,
	"needs_review": true,
	"done":         true,
	"archived":     true,
}

var whitespace = mdfile.Whitespace

// TaskStatuses returns the canonical, ordered set of allowed task statuses.
func TaskStatuses() []string {
	return append([]string{}, taskStatuses...)
}

// IsKnownTaskStatus reports whether status is one of TaskStatuses.
func IsKnownTaskStatus(status string) bool {
	return taskStatusSet[status]
}

// Task is the in-memory representation of one Work item, loaded from its
// canonical Markdown task card.
type Task struct {
	SchemaVersion int      `json:"schema_version"`
	ID            string   `json:"id"`
	Ref           string   `json:"ref"`
	Title         string   `json:"title"`
	Type          string   `json:"type"`
	Status        string   `json:"status"`
	Priority      int      `json:"priority"`
	Project       string   `json:"project"`
	ProjectID     string   `json:"project_id"`
	WorkspaceID   string   `json:"workspace_id"`
	Repositories  []string `json:"repositories"`
	// DependsOn lists prerequisite task refs (e.g. CORE-144), ids, or
	// locators that must reach DependencySatisfiedStatus before launch.
	DependsOn        []string  `json:"depends_on,omitempty"`
	Assignee         string    `json:"assignee"`
	AssignmentReason string    `json:"assignment_reason"`
	Source           string    `json:"source"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
	Launch           Launch    `json:"launch"`
	Summary          string    `json:"summary"`
	Body             string    `json:"body"`
	Comments         []Comment `json:"comments"`
	BlockedReason    string    `json:"blocked_reason,omitempty"`
	Path             string    `json:"path"`
	RelativePath     string    `json:"relative_path"`

	LaunchEvaluation LaunchEvaluation `json:"launch_evaluation"`
	Execution        ExecutionState   `json:"execution"`
}

// ExecutionState is a dashboard-facing snapshot of the live runtime session
// (if any) backing a task. It lives here (rather than corechain) only
// because it is embedded on Task; corechain's Runtime/Store own the actual
// session registry and populate this snapshot.
type ExecutionState struct {
	State          string         `json:"state"`
	VisibilityMode string         `json:"visibility_mode"`
	SessionPointer string         `json:"session_pointer,omitempty"`
	ClaimID        string         `json:"claim_id,omitempty"`
	Agent          string         `json:"agent,omitempty"`
	Provider       string         `json:"provider,omitempty"`
	Backend        string         `json:"backend,omitempty"`
	HostID         string         `json:"host_id,omitempty"`
	HostName       string         `json:"host_name,omitempty"`
	ProcessID      int            `json:"process_id,omitempty"`
	ThreadID       string         `json:"thread_id,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	TurnID         string         `json:"turn_id,omitempty"`
	ThreadTitle    string         `json:"thread_title,omitempty"`
	LogPath        string         `json:"log_path,omitempty"`
	LastEvent      string         `json:"last_event,omitempty"`
	LastActivityAt string         `json:"last_activity_at,omitempty"`
	LastEventAt    string         `json:"last_event_at,omitempty"`
	LastOutputAt   string         `json:"last_output_at,omitempty"`
	LastStatusAt   string         `json:"last_status_change_at,omitempty"`
	TerminalReason string         `json:"terminal_reason,omitempty"`
	BlockingReason string         `json:"blocking_reason,omitempty"`
	ProviderError  *ProviderError `json:"provider_error,omitempty"`
	// ToolWarning is a compact operator-facing note when required tools were
	// not observed during the session (CORE-120). Full evidence lives on the
	// runtime session, not on every task annotation.
	ToolWarning string `json:"tool_warning,omitempty"`
}

// Launch carries the optional launch override/safety flags a task card can
// set in frontmatter.
type Launch struct {
	Agent      string `json:"agent"`
	Mode       string `json:"mode"`
	AutoCommit bool   `json:"auto_commit"`
	AutoPush   bool   `json:"auto_push"`
	AutoPR     bool   `json:"auto_pr"`
}

// Comment is one parsed entry from a task card's "## Review Comments"
// section.
type Comment struct {
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
	Text      string `json:"text"`
}

// LaunchEvaluation records the outcome of running launch validation gates
// against a task.
type LaunchEvaluation struct {
	Agent          string      `json:"agent"`
	Backend        string      `json:"backend"`
	VisibilityMode string      `json:"visibility_mode,omitempty"`
	Repository     string      `json:"repository"`
	WorkingDir     string      `json:"working_dir"`
	FleetProfile   string      `json:"fleet_profile"`
	Command        []string    `json:"command"`
	Prompt         string      `json:"-"`
	InitialInput   string      `json:"-"`
	Launchable     bool        `json:"launchable"`
	Outcome        string      `json:"outcome"`
	FailedGates    []string    `json:"failed_gates"`
	PassedGates    []string    `json:"passed_gates"`
	Waiting        *LaunchWait `json:"waiting,omitempty"`
}

// LaunchWait describes why a launchable-otherwise task is waiting for a
// concurrency slot instead of starting.
type LaunchWait struct {
	Scope           string `json:"scope"`
	Resource        string `json:"resource"`
	ConflictClaimID string `json:"conflict_claim_id"`
	Reason          string `json:"reason"`
	WaitingSince    string `json:"waiting_since"`
}

// WorkerResult is the compact JSON payload a worker agent returns once real
// work is done: {"outcome":"completed","summary":"...", ...}. This is the
// CORE-97 worker outcome protocol, kept as a compatibility shape during the
// CORE-105 migration onto taskflow.ExecutionResult. Workers report through
// this payload; they do not mutate task lifecycle directly. Runtime converts
// WorkerResult → ExecutionResult and Finalizer applies transitions only
// through TaskService.ReportExecution.
type WorkerResult struct {
	// Outcome is the worker's self-reported result. Must be one of
	// WorkerOutcomes for ParseWorkerResult to accept the payload.
	Outcome string `json:"outcome"`
	Summary string `json:"summary,omitempty"`
	// Artifacts lists changed files or created artifacts the worker left
	// behind, relative to the target repository.
	Artifacts []string `json:"artifacts,omitempty"`
	// Tests lists checks/commands the worker ran to validate the change.
	Tests []string `json:"tests,omitempty"`
	// Question is the CORE-105 needs_input prompt for the operator.
	Question string `json:"question,omitempty"`
	// Error is optional failure detail for failed/blocked outcomes.
	Error string `json:"error,omitempty"`
	// Blockers lists concrete missing info/approval/access items when
	// Outcome is "blocked" or "waiting_input" (legacy CORE-97 field;
	// folded into Question / comment text by the Finalizer).
	Blockers []string `json:"blockers,omitempty"`
	// SuggestedNextStatus is an optional hint surfaced in the finalization
	// comment for the human reviewer. It is informational only: Core's
	// lifecycle layer, not the worker, decides the actual task status from
	// Outcome. See Target Contract in CORE-97's task card.
	SuggestedNextStatus string `json:"suggested_next_status,omitempty"`
	// ReviewNotes is optional additional context for the reviewer beyond
	// Summary.
	ReviewNotes string `json:"review_notes,omitempty"`
}

// WorkerOutcomes is the canonical, ordered set of outcome values a worker
// may report. ParseWorkerResult rejects any payload whose outcome is not in
// this set — including empty/placeholder values — so an unrelated JSON
// object elsewhere in captured output (for example the literal example
// payload embedded in the launch prompt itself, which some providers echo
// back into their own transcript) can never be mistaken for a real result.
//
// `needs_input` is the CORE-105 name; `waiting_input` remains accepted as a
// legacy alias and normalizes to needs_input before TaskService writes.
var WorkerOutcomes = []string{"completed", "blocked", "needs_rework", "failed", "waiting_input", "needs_input"}

var workerOutcomeSet = map[string]bool{
	"completed":     true,
	"blocked":       true,
	"needs_rework":  true,
	"failed":        true,
	"waiting_input": true,
	"needs_input":   true,
}

// IsValidWorkerOutcome reports whether outcome is one of WorkerOutcomes.
func IsValidWorkerOutcome(outcome string) bool {
	return workerOutcomeSet[strings.TrimSpace(outcome)]
}

// WorkerResultNeedsInput reports a HITL pause: the worker asked a human
// question and the 1-1-1 launch slot must stay held. Task status stays
// doing; this is not a Finalizer close.
func WorkerResultNeedsInput(result *WorkerResult) bool {
	if result == nil {
		return false
	}
	switch strings.TrimSpace(result.Outcome) {
	case "needs_input", "waiting_input":
		return true
	default:
		return false
	}
}

func (task *Task) Pass(step string) {
	task.LaunchEvaluation.PassedGates = append(task.LaunchEvaluation.PassedGates, step)
}

func (task *Task) FailLaunchGate(step string, reason string) {
	task.LaunchEvaluation.FailedGates = append(task.LaunchEvaluation.FailedGates, step+": "+reason)
}

func (task *Task) HasLaunchGateFailures() bool {
	return len(task.LaunchEvaluation.FailedGates) > 0
}

func (task *Task) ResolveLaunchEvaluation() {
	task.LaunchEvaluation.Launchable = !task.HasLaunchGateFailures() && task.LaunchEvaluation.Waiting == nil
	if task.LaunchEvaluation.Launchable {
		task.LaunchEvaluation.Outcome = "launchable"
		return
	}
	if task.LaunchEvaluation.Waiting != nil {
		task.LaunchEvaluation.Outcome = "waiting"
		return
	}
	task.LaunchEvaluation.Outcome = "not_launchable"
}

func (task *Task) MarkLaunchWaiting(wait LaunchWait) {
	task.LaunchEvaluation.Waiting = &wait
	task.ResolveLaunchEvaluation()
}

func (task *Task) ClearLaunchWaiting() {
	task.LaunchEvaluation.Waiting = nil
	task.ResolveLaunchEvaluation()
}

// DeriveBlockedReason picks the operator-facing explanation for a "blocked"
// task from its comments/launch-evaluation state: the most recent non-Alex
// comment, falling back to the most recent comment, a launch-wait reason, a
// failed launch gate, or the task summary. Exported so every provider
// (MarkdownProvider, Plane) derives blocked_reason with identical rules.
func DeriveBlockedReason(task Task) string {
	if task.Status != "blocked" {
		return ""
	}
	for i := len(task.Comments) - 1; i >= 0; i-- {
		if strings.EqualFold(strings.TrimSpace(task.Comments[i].Author), "alex") {
			continue
		}
		if text := strings.TrimSpace(task.Comments[i].Text); text != "" {
			return text
		}
	}
	for i := len(task.Comments) - 1; i >= 0; i-- {
		if text := strings.TrimSpace(task.Comments[i].Text); text != "" {
			return text
		}
	}
	if task.LaunchEvaluation.Waiting != nil && strings.TrimSpace(task.LaunchEvaluation.Waiting.Reason) != "" {
		return task.LaunchEvaluation.Waiting.Reason
	}
	if len(task.LaunchEvaluation.FailedGates) > 0 {
		return task.LaunchEvaluation.FailedGates[0]
	}
	return strings.TrimSpace(task.Summary)
}

// ResolveTaskProjectID picks the durable project id for a task from its
// workspace folder, frontmatter project, and repository list. Exported so
// MarkdownProvider (and any future file adapter) uses the same rule.
func ResolveTaskProjectID(workspaceID string, project string, repositories []string) string {
	if project != "" && project != workspaceID {
		return project
	}
	if len(repositories) == 1 {
		return ProjectIDFromRepository(workspaceID, repositories[0])
	}
	if project != "" {
		return project
	}
	return workspaceID
}

// ProjectIDFromRepository derives a project id from a workspace id and a
// single repository path. It is exported because corechain's
// workspace/project derivation (non-task) needs the identical rule to keep
// task project ids and workspace project ids consistent.
func ProjectIDFromRepository(workspaceID string, repository string) string {
	repository = strings.Trim(repository, "/")
	if repository == "" {
		return workspaceID
	}
	parts := strings.Split(repository, "/")
	name := parts[len(parts)-1]
	id := slugID(name)
	if id == "" {
		return workspaceID
	}
	if id == workspaceID {
		return id
	}
	return workspaceID + "/" + id
}

func slugID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out []rune
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
			lastDash = false
			continue
		}
		if !lastDash {
			out = append(out, '-')
			lastDash = true
		}
	}
	return strings.Trim(string(out), "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
