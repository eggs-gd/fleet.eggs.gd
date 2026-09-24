package settings

import (
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func workflowSnapshot() Workflow {
	return Workflow{
		Concurrency: []ConcurrencyScope{
			{Scope: "repository", Limit: 1, Source: "Fleet/LAUNCH_POLICY.md + execution slot hub"},
			{Scope: "project", Limit: 1, Source: "Fleet/LAUNCH_POLICY.md + execution slot hub"},
			{Scope: "assignee", Limit: 1, Source: "Fleet/LAUNCH_POLICY.md + execution slot hub"},
			{Scope: "assignee_orphan", Limit: 1, Source: "Fleet/LAUNCH_POLICY.md + execution slot hub"},
			{Scope: "task", Limit: 1, Source: "Fleet/LAUNCH_POLICY.md"},
		},
		Statuses:            tasklifecycle.TaskStatuses(),
		OperatorTransitions: tasklifecycle.StatusTransitionMap(),
		RuntimeOutcomes:     runtimeOutcomeRows(),
		HITL: HITLContract{
			TaskStatus:           "doing",
			SessionStatus:        "waiting_input",
			ReleasesSlot:         false,
			EqualsClosed:         false,
			Source:               "execution/host_pause.go pauseSessionForOperatorInput; WorkerResultNeedsInput",
			FinalizerIfPublished: string(finalizerStatus(taskflow.ExecutionNeedsInput)),
		},
		Notes: []string{
			"Live HITL (needs_input / waiting_input) does not publish an ExecutionResult; the host pauses instead.",
			"taskflow.MapExecutionOutcomeToStatus still maps needs_input → blocked. That table is Contour 3 if a result is published, not the live pause path.",
			"tasklifecycle.FinalizeSucceededExecution still maps waiting_input/needs_input → blocked. Live host_finalize intercepts HITL first.",
			"Concurrency limits are the 1-1-1 lock, not a single repository-only setting. Not GUI-editable.",
		},
	}
}

func runtimeOutcomeRows() []OutcomeRow {
	return []OutcomeRow{
		row("completed", taskflow.ExecutionCompleted, "closed", true, "host_finalize publish → taskflow.MapExecutionOutcomeToStatus"),
		{
			Outcome:      "needs_input / waiting_input",
			Task:         "doing",
			Session:      "waiting_input",
			ReleasesSlot: false,
			Source:       "execution/host_pause.go (does not publish ExecutionResult)",
		},
		row("needs_rework", taskflow.ExecutionNeedsRework, "closed", true, "host_finalize publish → taskflow.MapExecutionOutcomeToStatus"),
		row("blocked", taskflow.ExecutionBlocked, "closed", true, "host_finalize publish → taskflow.MapExecutionOutcomeToStatus"),
		row("failed", taskflow.ExecutionFailed, "failed", true, "host_finalize publish → taskflow.MapExecutionOutcomeToStatus"),
		row("timed_out", taskflow.ExecutionTimedOut, "closed", true, "host_finalize PublishExecutionOutcome"),
		row("cancelled", taskflow.ExecutionCancelled, "closed", true, "host_finalize PublishExecutionOutcome"),
		{
			Outcome:      "orphaned",
			Task:         "doing until operator resolves, or blocked if Finalizer receives ExecutionOrphaned",
			Session:      "orphaned",
			ReleasesSlot: false,
			Source:       "execution DetectStartupOrphans holds assignee_orphan; MapExecutionOutcomeToStatus(orphaned)=blocked only if published",
		},
	}
}

func row(outcome string, mapped taskflow.ExecutionOutcome, session string, releases bool, source string) OutcomeRow {
	return OutcomeRow{
		Outcome:      outcome,
		Task:         string(finalizerStatus(mapped)),
		Session:      session,
		ReleasesSlot: releases,
		Source:       source,
	}
}

func finalizerStatus(outcome taskflow.ExecutionOutcome) taskflow.Status {
	status, ok := taskflow.MapExecutionOutcomeToStatus(outcome)
	if !ok {
		return ""
	}
	return status
}
