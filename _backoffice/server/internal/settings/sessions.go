package settings

import (
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/execution"
	"github.com/eggs-gd/core.eggs.gd/internal/executionapi"
)

func classifySessions(runtime []executionapi.RuntimeSession, groups []execution.SessionGroup, orphans []execution.OrphanedTask) SessionCounts {
	counts := SessionCounts{Orphaned: len(orphans)}
	for _, orphan := range orphans {
		if orphanResumable(orphan) {
			counts.OrphanedResumable++
		}
	}
	for _, session := range runtime {
		if hitlSession(session) {
			counts.HITL++
			continue
		}
		if session.IsActive() {
			counts.Active++
		}
	}
	seen := map[string]bool{}
	for _, group := range groups {
		for _, session := range group.Sessions {
			key := session.ClaimID
			if key == "" {
				key = session.TaskPath + ":" + session.StartedAt + ":" + session.Status
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			if session.IsActive() {
				continue
			}
			counts.Closed++
			if failedSession(session.RuntimeSession) {
				counts.Failed++
			}
		}
	}
	return counts
}

func hitlSession(session executionapi.RuntimeSession) bool {
	if !session.IsActive() {
		return false
	}
	if session.Result != nil {
		switch strings.TrimSpace(session.Result.Outcome) {
		case "needs_input", "waiting_input":
			return true
		}
	}
	switch firstNonEmpty(session.ExecutionStatus, session.Status) {
	case "waiting_input", "operator_attention", "stalled", "needs_input":
		return true
	default:
		return false
	}
}

func failedSession(session executionapi.RuntimeSession) bool {
	if session.Result != nil && strings.TrimSpace(session.Result.Outcome) == "failed" {
		return true
	}
	switch firstNonEmpty(session.ExecutionStatus, session.Status) {
	case "failed", "provider_error", "timed_out", "dead", "terminal":
		return true
	default:
		return false
	}
}

func orphanResumable(orphan execution.OrphanedTask) bool {
	if strings.EqualFold(orphan.ExecutionState, "resumable") {
		return true
	}
	return strings.Contains(strings.ToLower(orphan.Reason+" "+orphan.BlockingReason), "resumable")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
