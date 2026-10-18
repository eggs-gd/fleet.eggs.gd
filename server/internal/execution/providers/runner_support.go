package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	execution "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func ReadSessionLog(root string, logPath string) string {
	return readSessionLog(root, logPath)
}

func mergeSessionToolCalls(session *RuntimeSession, calls []execution.ToolCallEvidence) {
	if session == nil || len(calls) == 0 {
		return
	}
	existing := execution.ToolUsageEvidence{}
	if session.ToolUsage != nil {
		existing = *session.ToolUsage
	}
	merged := execution.MergeToolCalls(existing, calls, time.Now())
	session.ToolUsage = &merged
}

func refreshSessionToolUsageFromText(session *RuntimeSession, text string) {
	if session == nil {
		return
	}
	at := time.Now()
	calls := execution.ExtractToolCallsFromSessionLog(text, at)
	if chatID := strings.TrimSpace(session.CursorChatID); chatID != "" {
		if transcript := execution.ReadCursorAgentTranscript(chatID, session.WorkingDir); transcript != "" {
			calls = append(calls, execution.ExtractToolCallsFromCursorTranscript(transcript, at)...)
		}
	}
	if strings.TrimSpace(text) == "" && len(calls) == 0 {
		return
	}
	mergeSessionToolCalls(session, calls)
}

func truncateDisplayText(text string) string {
	const maxLen = 240
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen]
}

func resetTimer(timer *time.Timer, duration time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(duration)
}

func openSessionLogFile(root string, session RuntimeSession) (io.WriteCloser, error) {
	return runtimedb.OpenLog(root, session.LogPath, true)
}

func validateSessionCommand(session RuntimeSession) error {
	if len(session.Command) == 0 {
		return errors.New("launch command is empty")
	}
	if strings.TrimSpace(session.Command[0]) == "" {
		return errors.New("launch command binary is empty")
	}
	return nil
}

func readSessionLog(root string, logPath string) string {
	return runtimedb.ReadLog(root, logPath)
}

func appendSessionLogLine(logFile io.Writer, format string, args ...any) {
	if logFile == nil {
		return
	}
	_, _ = fmt.Fprintf(logFile, format, args...)
}

func providerCommandOutput(timeout time.Duration, binary string, args ...string) ([]byte, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, binary, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return out, ctx.Err()
	}
	return out, err
}

func classifyProviderError(opts RunnerOptions, provider string, backend string, operation string, err error, output string) *tasklifecycle.ProviderError {
	if opts.Callbacks.ClassifyProviderError == nil {
		return nil
	}
	return opts.Callbacks.ClassifyProviderError(provider, backend, operation, err, output)
}

func failLaunch(opts RunnerOptions, session RuntimeSession, task tasklifecycle.Task, err error) {
	if opts.Callbacks.FailLaunchSession != nil {
		opts.Callbacks.FailLaunchSession(session, task, err)
	}
}

func recordProviderStart(opts RunnerOptions, session RuntimeSession, task tasklifecycle.Task, message string, details map[string]any) {
	if opts.Callbacks.RecordProviderSessionStart != nil {
		opts.Callbacks.RecordProviderSessionStart(session, task, message, details)
		return
	}
	upsertSession(opts, session)
}

func completeProviderSession(opts RunnerOptions, session RuntimeSession, task tasklifecycle.Task, exitMessage string, details map[string]any) {
	if opts.Callbacks.CompleteProviderSession != nil {
		opts.Callbacks.CompleteProviderSession(session, task, exitMessage, details)
		return
	}
	upsertSession(opts, session)
}

func upsertSession(opts RunnerOptions, session RuntimeSession) {
	if opts.Callbacks.UpsertSession != nil {
		opts.Callbacks.UpsertSession(session)
	}
}

func addTaskComment(opts RunnerOptions, task tasklifecycle.Task, comment string) {
	if opts.Callbacks.AddTaskComment == nil {
		return
	}
	locator := firstNonEmpty(task.RelativePath, task.Path, task.ID)
	if locator == "" {
		return
	}
	_ = opts.Callbacks.AddTaskComment(locator, comment)
}

func markOperatorAttention(opts RunnerOptions, session RuntimeSession, task tasklifecycle.Task, reason string) {
	if opts.Callbacks.MarkSessionOperatorAttention != nil {
		opts.Callbacks.MarkSessionOperatorAttention(session, task, reason)
		return
	}
	addTaskComment(opts, task, tasklifecycle.OperatorAttentionComment(reason, session.LogPath))
}

func applyWorkerReportedResult(opts RunnerOptions, session *RuntimeSession, task tasklifecycle.Task, result *tasklifecycle.WorkerResult) {
	if opts.Callbacks.ApplyWorkerReportedResult != nil {
		opts.Callbacks.ApplyWorkerReportedResult(session, task, result)
		return
	}
	session.Result = result
	upsertSession(opts, *session)
}
