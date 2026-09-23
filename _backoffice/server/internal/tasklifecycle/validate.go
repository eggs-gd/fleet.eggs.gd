package tasklifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DependencySatisfiedStatus is the only durable status that unblocks a
// depends_on gate. Conservative default (CORE-148): prerequisites must be
// fully accepted (`done`), not merely awaiting review.
const DependencySatisfiedStatus = "done"

// EvaluateLaunch runs the launch-eligibility gates against task and records
// the outcome on task.LaunchEvaluation: pickup status, assignee, Fleet
// profile existence, a single-repository requirement, and hard depends_on
// prerequisites. coreRoot is used to resolve the Fleet profile file and the
// launch working directory. knownTasks is the current board/snapshot used to
// resolve depends_on refs; pass nil/empty when no board is available.
//
// This is pure lifecycle validation: it has no knowledge of concurrency
// slots, active runtime sessions, or the chain.Processor pipeline that calls
// it in corechain.
func EvaluateLaunch(task *Task, coreRoot string, knownTasks []Task) {
	task.LaunchEvaluation = LaunchEvaluation{
		Agent: firstNonEmpty(task.Launch.Agent, task.Assignee),
	}
	requirePickupStatus(task)
	requireAssignee(task)
	requireDependsOn(task, knownTasks)
	requireFleetProfile(task, coreRoot)
	resolveRepository(task, coreRoot)
	task.ResolveLaunchEvaluation()
}

func requirePickupStatus(task *Task) {
	if !LaunchablePickupStatus(task.Status) {
		task.FailLaunchGate("status:pickup", fmt.Sprintf("status is %q, expected \"needs_rework\" or \"todo\"", task.Status))
		return
	}
	task.Pass("status:pickup")
}

func requireAssignee(task *Task) {
	if task.HasLaunchGateFailures() {
		return
	}
	if task.Assignee == "" || task.Assignee == "unassigned" {
		task.FailLaunchGate("assignee", "assignee is not set")
		return
	}
	task.Pass("assignee")
}

func requireDependsOn(task *Task, knownTasks []Task) {
	if task.HasLaunchGateFailures() {
		return
	}
	deps := NormalizeDependsOn(task.DependsOn)
	if len(deps) == 0 {
		task.Pass("depends_on")
		return
	}

	unresolved := UnresolvedDependencies(*task, knownTasks)
	if len(unresolved) == 0 {
		task.Pass("depends_on")
		return
	}
	task.FailLaunchGate("depends_on", strings.Join(unresolved, "; "))
}

func requireFleetProfile(task *Task, coreRoot string) {
	if task.HasLaunchGateFailures() {
		return
	}
	agent := firstNonEmpty(task.Launch.Agent, task.Assignee)
	if agent == "" || agent == "unassigned" {
		task.FailLaunchGate("fleet_profile", "agent is not set")
		return
	}

	path := filepath.Join(coreRoot, "Fleet", strings.ToLower(agent)+".md")
	if _, err := os.Stat(path); err != nil {
		task.FailLaunchGate("fleet_profile", fmt.Sprintf("Fleet profile not found for agent %q", agent))
		return
	}

	task.LaunchEvaluation.FleetProfile = path
	task.Pass("fleet_profile")
}

func resolveRepository(task *Task, coreRoot string) {
	if task.HasLaunchGateFailures() {
		return
	}
	if len(task.Repositories) == 0 {
		task.FailLaunchGate("repository", "repositories is empty")
		return
	}
	if len(task.Repositories) > 1 {
		task.FailLaunchGate("repository", "automatic launch requires exactly one repository")
		return
	}

	task.LaunchEvaluation.Repository = task.Repositories[0]
	task.LaunchEvaluation.WorkingDir = filepath.Join(coreRoot, "..", task.LaunchEvaluation.Repository)
	task.Pass("repository")
}

// DependencySatisfied reports whether status unblocks a depends_on gate.
func DependencySatisfied(status string) bool {
	return strings.TrimSpace(status) == DependencySatisfiedStatus
}

// NormalizeDependsOn trims blanks and drops empty entries while preserving order.
func NormalizeDependsOn(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, 0, len(refs))
	seen := map[string]bool{}
	for _, raw := range refs {
		ref := strings.TrimSpace(raw)
		if ref == "" {
			continue
		}
		key := strings.ToUpper(ref)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ref)
	}
	return out
}

// UnresolvedDependencies returns human-readable reasons for each depends_on
// entry that is missing or not yet DependencySatisfiedStatus.
func UnresolvedDependencies(task Task, knownTasks []Task) []string {
	deps := NormalizeDependsOn(task.DependsOn)
	if len(deps) == 0 {
		return nil
	}
	var unresolved []string
	for _, dep := range deps {
		if dependencyRefersToSelf(task, dep) {
			unresolved = append(unresolved, fmt.Sprintf("%s is a self-dependency", dep))
			continue
		}
		found, ok := FindDependency(knownTasks, dep)
		if !ok {
			unresolved = append(unresolved, fmt.Sprintf("%s not found", dep))
			continue
		}
		if DependencySatisfied(found.Status) {
			continue
		}
		unresolved = append(unresolved, fmt.Sprintf(
			"%s status is %q, want %q",
			dep,
			found.Status,
			DependencySatisfiedStatus,
		))
	}
	return unresolved
}

// FindDependency resolves a depends_on token against known tasks by human
// ref (CORE-144), canonical id, or opaque locator/path.
func FindDependency(knownTasks []Task, ref string) (Task, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Task{}, false
	}
	for _, candidate := range knownTasks {
		if dependencyMatches(candidate, ref) {
			return candidate, true
		}
	}
	return Task{}, false
}

// TasksDependingOn returns pickup-status tasks whose depends_on list includes
// the given completed task (by ref, id, or locator).
func TasksDependingOn(knownTasks []Task, completed Task) []Task {
	var out []Task
	for _, candidate := range knownTasks {
		if !LaunchablePickupStatus(candidate.Status) {
			continue
		}
		deps := NormalizeDependsOn(candidate.DependsOn)
		if len(deps) == 0 {
			continue
		}
		for _, dep := range deps {
			if dependencyMatches(completed, dep) {
				out = append(out, candidate)
				break
			}
		}
	}
	return out
}

func dependencyMatches(task Task, ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(task.Ref), ref) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(task.ID), ref) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(task.RelativePath), ref) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(task.Path), ref) {
		return true
	}
	return false
}

func dependencyRefersToSelf(task Task, dep string) bool {
	return dependencyMatches(task, dep)
}
