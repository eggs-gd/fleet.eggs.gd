package tasklifecycle

import (
	"fmt"
	"sort"
	"strings"
)

// TaskPickupLess orders tasks the way a worker or daemon should pick them up:
// needs_rework before todo before everything else, then ascending priority
// (1 highest), then natural CORE-* ref order, then file path as a final
// deterministic tie-breaker.
func TaskPickupLess(left Task, right Task) bool {
	if statusRank(left.Status) != statusRank(right.Status) {
		return statusRank(left.Status) < statusRank(right.Status)
	}
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	if left.Ref != right.Ref {
		return naturalRefLess(left.Ref, right.Ref)
	}
	return left.RelativePath < right.RelativePath
}

func statusRank(status string) int {
	switch status {
	case "needs_rework":
		return 0
	case "todo":
		return 1
	default:
		return 2
	}
}

// SortTasksForPickup returns a new slice sorted in pickup order.
func SortTasksForPickup(tasks []Task) []Task {
	sorted := append([]Task{}, tasks...)
	sort.Slice(sorted, func(i, j int) bool {
		return TaskPickupLess(sorted[i], sorted[j])
	})
	return sorted
}

// LaunchableCandidatesForAssignee returns, in pickup order, the tasks in
// todo/needs_rework whose effective launch agent (launch.agent, falling back
// to assignee) matches assignee.
func LaunchableCandidatesForAssignee(tasks []Task, assignee string) []Task {
	candidates := []Task{}
	for _, task := range tasks {
		agent := firstNonEmpty(task.Launch.Agent, task.Assignee)
		if !LaunchablePickupStatus(task.Status) || agent != assignee {
			continue
		}
		candidates = append(candidates, task)
	}
	return SortTasksForPickup(candidates)
}

// LaunchablePickupStatus reports whether status is a status daemon/manual
// pickup may launch from.
func LaunchablePickupStatus(status string) bool {
	return status == "needs_rework" || status == "todo"
}

// naturalRefLess compares CORE-<n> style refs numerically by their trailing
// number so CORE-9 sorts before CORE-10.
func naturalRefLess(left string, right string) bool {
	leftPrefix, leftNumber := splitRef(left)
	rightPrefix, rightNumber := splitRef(right)
	if leftPrefix == rightPrefix && leftNumber != rightNumber {
		return leftNumber < rightNumber
	}
	return left < right
}

func splitRef(ref string) (string, int) {
	prefix, number, ok := strings.Cut(ref, "-")
	if !ok {
		return ref, 0
	}
	var parsed int
	_, _ = fmt.Sscanf(number, "%d", &parsed)
	return prefix, parsed
}
