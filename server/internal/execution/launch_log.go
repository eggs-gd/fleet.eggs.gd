package execution

import (
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

type LaunchLogCategory string

const (
	LaunchLogCandidate LaunchLogCategory = "candidate"
	LaunchLogSkipped   LaunchLogCategory = "skipped"
	LaunchLogNotReady  LaunchLogCategory = "not_ready"
)

type LaunchLogClassification struct {
	Category LaunchLogCategory
	Event    string
	Stage    string
	Reason   string
	Routine  bool
}

func ClassifyLaunchLog(task tasklifecycle.Task) LaunchLogClassification {
	if task.LaunchEvaluation.Launchable {
		return LaunchLogClassification{
			Category: LaunchLogCandidate,
			Event:    "launch_candidate",
			Stage:    "launch-plan",
			Reason:   "Task passed launch validation",
		}
	}

	reason := firstBlocker(task)
	if reason == "" {
		return LaunchLogClassification{
			Category: LaunchLogNotReady,
			Event:    "launch_not_ready",
			Stage:    "validate",
			Reason:   "Task is not launchable and has no explicit blocker",
		}
	}

	if strings.HasPrefix(reason, "status:pickup:") {
		return LaunchLogClassification{
			Category: LaunchLogSkipped,
			Event:    "launch_skipped",
			Stage:    "validate",
			Reason:   reason,
			Routine:  true,
		}
	}

	if strings.HasPrefix(reason, "depends_on:") {
		return LaunchLogClassification{
			Category: LaunchLogSkipped,
			Event:    "launch_skipped",
			Stage:    "validate",
			Reason:   reason,
		}
	}

	if strings.HasPrefix(reason, "agent_executable:") {
		return LaunchLogClassification{
			Category: LaunchLogNotReady,
			Event:    "launch_not_ready",
			Stage:    "validate",
			Reason:   reason,
		}
	}

	if strings.HasPrefix(reason, "agent_live_ready:") || strings.HasPrefix(reason, "agent_visibility:") {
		return LaunchLogClassification{
			Category: LaunchLogSkipped,
			Event:    "launch_skipped",
			Stage:    "validate",
			Reason:   reason,
		}
	}

	return LaunchLogClassification{
		Category: LaunchLogNotReady,
		Event:    "launch_not_ready",
		Stage:    "validate",
		Reason:   reason,
	}
}

func firstBlocker(task tasklifecycle.Task) string {
	if len(task.LaunchEvaluation.FailedGates) == 0 {
		return ""
	}
	return task.LaunchEvaluation.FailedGates[0]
}
