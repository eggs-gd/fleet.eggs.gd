package execution

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

type (
	ToolCallEvidence  = executionapi.ToolCallEvidence
	ToolUsageEvidence = executionapi.ToolUsageEvidence
	RequirementInput  = executionapi.RequirementInput
)

// InitSessionToolUsage seeds required-tool evidence for a newly claimed session.
func InitSessionToolUsage(session *RuntimeSession, task tasklifecycle.Task) {
	if session == nil {
		return
	}
	at := time.Now()
	usage := executionapi.InitToolUsage(requirementInputFromTask(task, session), at)
	session.ToolUsage = &usage
}

// ObserveSessionToolCalls merges provider-observed tool calls into session state.
func ObserveSessionToolCalls(session *RuntimeSession, calls []ToolCallEvidence) {
	if session == nil || len(calls) == 0 {
		return
	}
	at := time.Now()
	existing := ToolUsageEvidence{}
	if session.ToolUsage != nil {
		existing = *session.ToolUsage
	}
	merged := executionapi.MergeToolCalls(existing, calls, at)
	session.ToolUsage = &merged
}

// ObserveCodexToolParams extracts and merges tool evidence from one Codex RPC message.
func ObserveCodexToolParams(session *RuntimeSession, method string, raw json.RawMessage) {
	ObserveSessionToolCalls(session, executionapi.ExtractToolCallsFromCodexParams(method, raw, time.Now()))
}

// RefreshSessionToolUsageFromLog rescans the session log (and Cursor agent
// transcript when a chat id is present) and refreshes missing/warning.
func RefreshSessionToolUsageFromLog(session *RuntimeSession, logText string) {
	if session == nil {
		return
	}
	at := time.Now()
	existing := ToolUsageEvidence{}
	if session.ToolUsage != nil {
		existing = *session.ToolUsage
	}
	calls := executionapi.ExtractToolCallsFromSessionLog(logText, at)
	if chatID := strings.TrimSpace(session.CursorChatID); chatID != "" {
		if transcript := executionapi.ReadCursorAgentTranscript(chatID, session.WorkingDir); transcript != "" {
			calls = append(calls, executionapi.ExtractToolCallsFromCursorTranscript(transcript, at)...)
		}
	}
	merged := executionapi.MergeToolCalls(existing, calls, at)
	session.ToolUsage = &merged
}

func requirementInputFromTask(task tasklifecycle.Task, session *RuntimeSession) RequirementInput {
	workingDir := task.LaunchEvaluation.WorkingDir
	repository := task.LaunchEvaluation.Repository
	if session != nil {
		if workingDir == "" {
			workingDir = session.WorkingDir
		}
		if repository == "" {
			repository = session.Repository
		}
	}
	return executionapi.RequirementInputFromTask(
		task.Type,
		task.Title,
		task.Body,
		repository,
		workingDir,
		task.Repositories,
	)
}

func toolWarningFromSession(session RuntimeSession) string {
	if session.ToolUsage == nil {
		return ""
	}
	return executionapi.ToolUsageWarning(*session.ToolUsage)
}
