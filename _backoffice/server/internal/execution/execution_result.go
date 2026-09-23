package execution

import (
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

// executionResultFromWorker converts a CORE-97 WorkerResult (or nil) into the
// canonical taskflow.ExecutionResult used by TaskService.ReportExecution.
// Session log path is folded into Summary so Finalizer comments keep an
// operator-visible pointer without extending the ExecutionResult contract.
func ExecutionResultFromWorker(task tasklifecycle.Task, result *tasklifecycle.WorkerResult, logPath string) taskflow.ExecutionResult {
	return ExecutionResultFromWorkerWithTools(task, result, logPath, nil)
}

// ExecutionResultFromWorkerWithTools folds optional tool-usage warnings into the
// Finalizer-facing summary so missing required tools are visible before review.
func ExecutionResultFromWorkerWithTools(task tasklifecycle.Task, result *tasklifecycle.WorkerResult, logPath string, tools *ToolUsageEvidence) taskflow.ExecutionResult {
	er := executionResultFromWorkerBase(task, result, logPath)
	if tools != nil {
		er.Summary = executionapi.WithToolUsageWarning(er.Summary, *tools)
	}
	return er
}

func executionResultFromWorkerBase(task tasklifecycle.Task, result *tasklifecycle.WorkerResult, logPath string) taskflow.ExecutionResult {
	taskID := firstNonEmpty(task.RelativePath, task.Path, task.ID)
	if result == nil {
		return withSessionLog(taskflow.ExecutionResult{
			TaskID:  taskID,
			Outcome: taskflow.ExecutionCompleted,
			Summary: "Agent execution completed.",
		}, logPath)
	}

	outcome, ok := taskflow.NormalizeExecutionOutcome(result.Outcome)
	if !ok {
		// Preserve the raw outcome so ReportExecution rejects it and the
		// finalizer can fall back to a blocked invalid-outcome comment.
		outcome = taskflow.ExecutionOutcome(strings.TrimSpace(result.Outcome))
	}

	er := taskflow.ExecutionResult{
		TaskID:    taskID,
		Outcome:   outcome,
		Summary:   WorkerOutcomeHeadline(outcome, result.Outcome),
		Tests:     append([]string{}, result.Tests...),
		Artifacts: append([]string{}, result.Artifacts...),
		Question:  strings.TrimSpace(result.Question),
		Error:     strings.TrimSpace(result.Error),
	}
	if summary := strings.TrimSpace(result.Summary); summary != "" {
		if er.Summary == "" {
			er.Summary = summary
		} else {
			er.Summary += "\n\n" + summary
		}
	}
	if er.Question == "" && len(result.Blockers) > 0 {
		er.Question = joinNonEmptyLines(result.Blockers)
	}
	if notes := strings.TrimSpace(result.ReviewNotes); notes != "" {
		er.Summary = strings.TrimSpace(er.Summary + "\n\nReview notes: " + notes)
	}
	if hint := strings.TrimSpace(result.SuggestedNextStatus); hint != "" {
		er.Summary = strings.TrimSpace(er.Summary + "\n\nWorker-suggested next status: `" + hint +
			"` (informational only; Core's lifecycle layer decides the actual transition).")
	}
	return withSessionLog(er, logPath)
}

// executionResultFromRuntimeStatus builds an ExecutionResult for process-level
// failures (timeout, cancel, launch failure) when no worker JSON payload exists.
func ExecutionResultFromRuntimeStatus(task tasklifecycle.Task, outcome taskflow.ExecutionOutcome, summary string, errText string, logPath string) taskflow.ExecutionResult {
	return withSessionLog(taskflow.ExecutionResult{
		TaskID:  firstNonEmpty(task.RelativePath, task.Path, task.ID),
		Outcome: outcome,
		Summary: strings.TrimSpace(summary),
		Error:   strings.TrimSpace(errText),
	}, logPath)
}

func WorkerOutcomeHeadline(outcome taskflow.ExecutionOutcome, raw string) string {
	switch outcome {
	case taskflow.ExecutionCompleted:
		return "Agent execution completed."
	case taskflow.ExecutionBlocked:
		return "Agent execution reported blocked."
	case taskflow.ExecutionNeedsRework:
		return "Agent execution reported the work needs rework before it is done."
	case taskflow.ExecutionFailed:
		return "Agent execution reported failure."
	case taskflow.ExecutionNeedsInput:
		return "Agent execution reported it is waiting on operator input."
	case taskflow.ExecutionCancelled:
		return "Agent execution was cancelled."
	case taskflow.ExecutionTimedOut:
		return "Agent execution timed out."
	case taskflow.ExecutionOrphaned:
		return "Agent execution was orphaned."
	default:
		return "Agent execution succeeded but returned invalid result outcome `" + strings.TrimSpace(raw) + "`."
	}
}

func withSessionLog(result taskflow.ExecutionResult, logPath string) taskflow.ExecutionResult {
	logPath = strings.TrimSpace(logPath)
	if logPath == "" {
		return result
	}
	note := "Session log: `" + logPath + "`"
	if strings.TrimSpace(result.Summary) == "" {
		result.Summary = note
	} else {
		result.Summary += "\n\n" + note
	}
	return result
}

func joinNonEmptyLines(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, "; ")
}
