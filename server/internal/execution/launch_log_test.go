package execution

import (
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func TestClassifyLaunchLogTreatsNonPickupStatusAsRoutineSkip(t *testing.T) {
	decision := ClassifyLaunchLog(Task{
		Ref:    "CORE-24",
		Status: "done",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			FailedGates: []string{`status:pickup: status is "done", expected "needs_rework" or "todo"`},
		},
	})

	if decision.Category != LaunchLogSkipped {
		t.Fatalf("category = %q, want %q", decision.Category, LaunchLogSkipped)
	}
	if !decision.Routine {
		t.Fatal("non-pickup status skip should be routine")
	}
	if decision.Event != "launch_skipped" {
		t.Fatalf("event = %q, want launch_skipped", decision.Event)
	}
}

func TestClassifyLaunchLogKeepsValidationFailuresBlocked(t *testing.T) {
	decision := ClassifyLaunchLog(Task{
		Ref:    "CORE-26",
		Status: "todo",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			FailedGates: []string{"repository: repositories is empty"},
		},
	})

	if decision.Category != LaunchLogNotReady {
		t.Fatalf("category = %q, want %q", decision.Category, LaunchLogNotReady)
	}
	if decision.Event != "launch_not_ready" {
		t.Fatalf("event = %q, want launch_not_ready", decision.Event)
	}
}

func TestClassifyLaunchLogTreatsDependsOnAsVisibleSkip(t *testing.T) {
	decision := ClassifyLaunchLog(Task{
		Ref:    "CORE-147",
		Status: "todo",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			FailedGates: []string{`depends_on: CORE-144 status is "needs_review", want "done"`},
		},
	})

	if decision.Category != LaunchLogSkipped {
		t.Fatalf("category = %q, want %q", decision.Category, LaunchLogSkipped)
	}
	if decision.Routine {
		t.Fatal("depends_on skip should be operator-visible, not routine")
	}
	if decision.Event != "launch_skipped" {
		t.Fatalf("event = %q, want launch_skipped", decision.Event)
	}
}

func TestClassifyLaunchLogTreatsUnavailableAgentAsVisibleSkip(t *testing.T) {
	decision := ClassifyLaunchLog(Task{
		Ref:    "CORE-62",
		Status: "todo",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			FailedGates: []string{"agent_live_ready: Codex CLI live daemon launch is not verified yet."},
		},
	})

	if decision.Category != LaunchLogSkipped {
		t.Fatalf("category = %q, want %q", decision.Category, LaunchLogSkipped)
	}
	if decision.Routine {
		t.Fatal("unavailable assigned agent skip should be visible, not routine")
	}
	if decision.Event != "launch_skipped" {
		t.Fatalf("event = %q, want launch_skipped", decision.Event)
	}
}

func TestClassifyLaunchLogTreatsMissingExecutableAsNotReady(t *testing.T) {
	decision := ClassifyLaunchLog(Task{
		Ref:    "CORE-150",
		Status: "todo",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			FailedGates: []string{"agent_executable: Codex CLI is not available on the daemon PATH. Install the standalone Codex CLI or configure the Codex binary path, then retry this task."},
		},
	})

	if decision.Category != LaunchLogNotReady {
		t.Fatalf("category = %q, want %q", decision.Category, LaunchLogNotReady)
	}
	if decision.Event != "launch_not_ready" {
		t.Fatalf("event = %q, want launch_not_ready", decision.Event)
	}
}

func TestClassifyLaunchLogTreatsVisibilityGateAsVisibleSkip(t *testing.T) {
	decision := ClassifyLaunchLog(Task{
		Ref:    "CORE-70",
		Status: "todo",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			VisibilityMode: "headless",
			FailedGates:    []string{"agent_visibility: backend process visibility is headless"},
		},
	})

	if decision.Category != LaunchLogSkipped {
		t.Fatalf("category = %q, want %q", decision.Category, LaunchLogSkipped)
	}
	if decision.Routine {
		t.Fatal("visibility skip should be operator-facing, not routine")
	}
}

func TestClassifyLaunchLogDetectsCandidate(t *testing.T) {
	decision := ClassifyLaunchLog(Task{
		Ref:    "CORE-26",
		Status: "todo",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Launchable: true,
		},
	})

	if decision.Category != LaunchLogCandidate {
		t.Fatalf("category = %q, want %q", decision.Category, LaunchLogCandidate)
	}
	if decision.Stage != "launch-plan" {
		t.Fatalf("stage = %q, want launch-plan", decision.Stage)
	}
}
