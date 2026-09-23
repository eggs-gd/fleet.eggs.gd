package execution

import "github.com/eggs-gd/core.eggs.gd/internal/executionapi"

type (
	ClaudeSessionDetails  = executionapi.ClaudeSessionDetails
	CodexSessionDetails   = executionapi.CodexSessionDetails
	CursorSessionDetails  = executionapi.CursorSessionDetails
	RuntimeSession        = executionapi.RuntimeSession
	SessionControlRequest = executionapi.SessionControlRequest
)

func ParseRemoteControlURL(logOutput string) string {
	return executionapi.ParseRemoteControlURL(logOutput)
}

func ParseCursorChatID(output string) string {
	return executionapi.ParseCursorChatID(output)
}

func CursorOperatorCommand(binary string, chatID string, workingDir string) string {
	return executionapi.CursorOperatorCommand(binary, chatID, workingDir)
}

func CodexRemoteThreadTitle(ref string, title string, id string) string {
	return executionapi.CodexRemoteThreadTitle(ref, title, id)
}
