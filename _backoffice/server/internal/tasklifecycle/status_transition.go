package tasklifecycle

var allowedTaskStatusTransitions = map[string]map[string]bool{
	"backlog": {
		"todo":     true,
		"archived": true,
	},
	"todo": {
		"doing":    true,
		"blocked":  true,
		"backlog":  true,
		"archived": true,
	},
	"needs_rework": {
		"doing":    true,
		"blocked":  true,
		"todo":     true,
		"archived": true,
	},
	"doing": {
		"needs_review": true,
		"needs_rework": true,
		"blocked":      true,
		"todo":         true,
		"done":         true,
	},
	"blocked": {
		"needs_review": true,
		"needs_rework": true,
		"todo":         true,
		"archived":     true,
	},
	"needs_review": {
		"needs_rework": true,
		"todo":         true,
		"done":         true,
		"archived":     true,
	},
	"done": {
		"archived": true,
	},
	"archived": {
		"backlog": true,
	},
}

// AllowedTaskStatusTransition reports whether the state machine allows a
// task to move from one status to another. A no-op transition (from == to)
// is always allowed.
func AllowedTaskStatusTransition(from string, to string) bool {
	if from == to {
		return true
	}
	return allowedTaskStatusTransitions[from][to]
}

// AllowedNextStatuses returns the statuses a task may move to from `from`,
// in canonical TaskStatuses order. The no-op self-transition is omitted.
func AllowedNextStatuses(from string) []string {
	next := allowedTaskStatusTransitions[from]
	if len(next) == 0 {
		return nil
	}
	out := make([]string, 0, len(next))
	for _, status := range taskStatuses {
		if next[status] {
			out = append(out, status)
		}
	}
	return out
}

// StatusTransitionMap is the full from→next lookup used by /api/state so the
// dashboard can render only legal operator moves.
func StatusTransitionMap() map[string][]string {
	out := make(map[string][]string, len(taskStatuses))
	for _, from := range taskStatuses {
		out[from] = AllowedNextStatuses(from)
	}
	return out
}
