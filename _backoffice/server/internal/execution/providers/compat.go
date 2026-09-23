package providers

import "github.com/eggs-gd/core.eggs.gd/internal/executionapi"

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

func ProviderCapabilityFlags(provider string, backend string) executionapi.ProviderCapabilities {
	return executionapi.ProviderCapabilityFlags(provider, backend)
}

func ProviderSessionReuseCapability(backend string) executionapi.SessionReuseCapability {
	return executionapi.ProviderSessionReuseCapability(backend)
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

func CodexRemoteThreadTitle(ref string, title string, id string) string {
	return executionapi.CodexRemoteThreadTitle(ref, title, id)
}
