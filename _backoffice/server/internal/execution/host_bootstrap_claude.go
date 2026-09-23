package execution

import (
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

type persistedSessionValidation struct {
	Resumable       bool
	Status          string
	ExecutionStatus string
	Reason          string
	ProviderError   *tasklifecycle.ProviderError
	LastEvent       string
	WorkerResult    *tasklifecycle.WorkerResult
}

func (s *Service) validatePersistedClaudeBackgroundRemoteSession(session RuntimeSession) persistedSessionValidation {
	backgroundID := strings.TrimSpace(session.BackgroundID)
	if backgroundID == "" {
		return persistedSessionValidation{
			Status:          "dead",
			ExecutionStatus: "dead",
			Reason:          "persisted Claude background session has no background id",
			LastEvent:       "missing_background_id",
		}
	}
	claudeBinary := persistedClaudeBinary(session)
	transcript, err := FetchBackgroundSessionLogs(claudeBinary, backgroundID)
	if err != nil {
		pe := ClassifyProviderError("claude", BackendBackgroundRemote, "claude logs", err, "")
		if pe == nil {
			pe = tasklifecycle.NewProviderError("claude", "provider_unavailable", "claude logs", "Claude provider control path is unavailable.", err.Error())
		}
		return persistedSessionValidation{
			Status:          pe.Kind,
			ExecutionStatus: pe.Kind,
			Reason:          pe.Reason,
			ProviderError:   pe,
			LastEvent:       "provider_reconciliation_failed",
		}
	}
	workerResult := tasklifecycle.ParseWorkerResult(transcript)
	status, found, err := QueryBackgroundAgentStatus(claudeBinary, backgroundID)
	if err != nil {
		pe := ClassifyProviderError("claude", BackendBackgroundRemote, "claude agents --json", err, "")
		if pe == nil {
			pe = tasklifecycle.NewProviderError("claude", "provider_unavailable", "claude agents --json", "Claude provider session list is unavailable.", err.Error())
		}
		return persistedSessionValidation{
			Status:          pe.Kind,
			ExecutionStatus: pe.Kind,
			Reason:          pe.Reason,
			ProviderError:   pe,
			LastEvent:       "provider_reconciliation_failed",
			WorkerResult:    workerResult,
		}
	}
	if !found {
		return persistedSessionValidation{
			Status:          "dead",
			ExecutionStatus: "dead",
			Reason:          "Claude background session is no longer listed by `claude agents --json`",
			LastEvent:       "provider_session_missing",
			WorkerResult:    workerResult,
		}
	}
	if IsTerminalBackgroundState(status) {
		state := firstNonEmpty(status.State, status.Status, "terminal")
		return persistedSessionValidation{
			Status:          "terminal",
			ExecutionStatus: "terminal",
			Reason:          "Claude background session is terminal according to `claude agents --json`: " + state,
			LastEvent:       "provider_session_terminal",
			WorkerResult:    workerResult,
		}
	}
	return persistedSessionValidation{Resumable: true}
}

func persistedClaudeBinary(session RuntimeSession) string {
	if len(session.Command) > 0 && strings.TrimSpace(session.Command[0]) != "" {
		return session.Command[0]
	}
	return "claude"
}

func staleClaudeBackgroundSessionComment(session RuntimeSession) string {
	return tasklifecycle.StaleProviderSessionComment(
		session.ProviderError,
		providerErrorContextFor(session),
		firstNonEmpty(session.ErrorMessage, "provider session could not be verified"),
		firstNonEmpty(session.ExecutionStatus, session.Status, "dead"),
	)
}
