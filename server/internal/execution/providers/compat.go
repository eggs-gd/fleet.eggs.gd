package providers

import "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"

type (
	Agent                 = executionapi.Agent
	TaskContext           = executionapi.TaskContext
	Plan                  = executionapi.Plan
	RuntimeSession        = executionapi.RuntimeSession
	SessionControlRequest = executionapi.SessionControlRequest
	ProviderCapabilities  = executionapi.ProviderCapabilities
	ClaudeSessionDetails  = executionapi.ClaudeSessionDetails
	CodexSessionDetails   = executionapi.CodexSessionDetails
	CursorSessionDetails  = executionapi.CursorSessionDetails
	GeminiSessionDetails  = executionapi.GeminiSessionDetails
)

const (
	BackendBackgroundRemote = executionapi.BackendBackgroundRemote
	BackendCodexAppServer   = executionapi.BackendCodexAppServer
	BackendCursorVisible    = executionapi.BackendCursorVisible
	BackendGeminiHeadless   = executionapi.BackendGeminiHeadless
)

func BuildPrompt(task TaskContext) string {
	return executionapi.BuildPrompt(task)
}

func ParseRemoteControlURL(logOutput string) string {
	return executionapi.ParseRemoteControlURL(logOutput)
}

func ParseCursorChatID(output string) string {
	return executionapi.ParseCursorChatID(output)
}

func CursorOperatorCommand(binary string, chatID string, workingDir string) string {
	return executionapi.CursorOperatorCommand(binary, chatID, workingDir)
}

func GeminiOperatorCommand(binary string, projectID string, conversationID string) string {
	return executionapi.GeminiOperatorCommand(binary, projectID, conversationID)
}

func CodexRemoteThreadTitle(ref string, title string, id string) string {
	return executionapi.CodexRemoteThreadTitle(ref, title, id)
}
