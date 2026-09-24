package open

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/providerconfig"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/plane"
)

// planePollMinInterval floors how often a ChangeSource provider is polled.
// Plane's documented rate limit is 60 requests/minute per API key.
const planePollMinInterval = 45 * time.Second

// OpenOptions configures which Provider backs task read/mutation.
type OpenOptions struct {
	Root        string
	RuntimeRoot string
	Settings    providerconfig.Settings
	Interval    time.Duration
	Board       *taskprovider.Board
}

// Wiring carries factory-time capabilities so composition never branches on
// Provider.Type() strings (CORE-107).
type Wiring struct {
	PollInterval time.Duration
}

// Open builds the configured Provider and listener wiring.
func Open(opts OpenOptions) (taskprovider.Provider, Wiring) {
	switch strings.ToLower(strings.TrimSpace(opts.Settings.Type)) {
	case "plane":
		token := os.Getenv(opts.Settings.Plane.TokenEnv)
		pollInterval := opts.Interval
		if pollInterval < planePollMinInterval {
			pollInterval = planePollMinInterval
		}
		return plane.New(opts.Settings.Plane, token, opts.Root), Wiring{PollInterval: pollInterval}
	default:
		return NewMarkdown(opts.Root, opts.RuntimeRoot, opts.Board), Wiring{}
	}
}

// NewMarkdown builds a Markdown provider wired to keep board projection,
// Work/INDEX.md, and the audit event log in sync with every task mutation.
// board may be nil. runtimeRoot (App's own ~/.fleet home) is where the
// audit event log lives — never root, which is the operator's Data.
func NewMarkdown(root, runtimeRoot string, board *taskprovider.Board) *markdown.Provider {
	return markdown.New(root, markdown.Hooks{
		AfterMutate: func(task taskprovider.Task, eventType string, message string, details map[string]any) error {
			if board != nil {
				board.UpsertTask(task)
			}
			if err := markdown.RebuildWorkIndex(root); err != nil {
				return err
			}
			emitTaskEvent(runtimeRoot, eventType, task, message, details)
			_ = audit.AppendEvent(runtimeRoot, audit.Event{
				Type:    "index_rebuilt",
				Path:    "Work/INDEX.md",
				Message: "Work index rebuilt after task mutation",
			})
			return nil
		},
		AfterObserve: func(observed markdown.ObservedTaskChange) error {
			task := observed.After
			if board != nil {
				board.UpsertTask(task)
			}
			if observed.GuardApplied {
				emitTaskEvent(runtimeRoot, "task_lifecycle_finalization_deferred", task, "Active execution prevented direct task lifecycle finalization", map[string]any{
					"claim_id":         observed.Session.ClaimID,
					"execution_status": firstNonEmpty(observed.Session.ExecutionStatus, observed.Session.Status),
					"attempted_status": firstNonEmpty(observed.ObservedStatus, task.Status),
				})
				return nil
			}
			var before taskprovider.Task
			hadBefore := false
			if observed.Before != nil {
				before = *observed.Before
				hadBefore = true
			}
			changes := taskProjectionChangeSummary(before, task, hadBefore)
			emitTaskEvent(runtimeRoot, "core_projected", task, "Task file projected into runtime state", map[string]any{
				"stage":      "project",
				"kind":       "task",
				"event_kind": observed.Event.Kind,
				"task_event": observed.Event.Kind,
				"source":     string(observed.Event.Source),
				"changes":    changes,
			})
			_ = audit.AppendEvent(runtimeRoot, audit.Event{
				Type:    "index_rebuilt",
				Path:    "Work/INDEX.md",
				Message: "Work index rebuilt after file event",
			})
			return nil
		},
		OnEvent: func(task taskprovider.Task, eventType string, message string) {
			emitTaskEvent(runtimeRoot, eventType, task, message, nil)
		},
	})
}

func emitTaskEvent(runtimeRoot string, eventType string, task taskprovider.Task, message string, extra map[string]any) {
	details := map[string]any{
		"status":            task.Status,
		"priority":          task.Priority,
		"failed_gates":      task.LaunchEvaluation.FailedGates,
		"passed_gates":      task.LaunchEvaluation.PassedGates,
		"launchable":        task.LaunchEvaluation.Launchable,
		"launch_outcome":    task.LaunchEvaluation.Outcome,
		"launch_waiting":    task.LaunchEvaluation.Waiting,
		"working_dir":       task.LaunchEvaluation.WorkingDir,
		"command":           task.LaunchEvaluation.Command,
		"launch_config":     task.Launch,
		"launch_evaluation": task.LaunchEvaluation,
	}
	for key, value := range extra {
		details[key] = value
	}
	_ = audit.AppendEvent(runtimeRoot, audit.Event{
		Type:       eventType,
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		Path:       task.RelativePath,
		Agent:      firstNonEmpty(task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee),
		Repository: task.LaunchEvaluation.Repository,
		Outcome:    task.LaunchEvaluation.Outcome,
		Message:    message,
		Details:    details,
	})
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
