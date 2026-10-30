package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	execution "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func RunCursorVisibleSession(ctx context.Context, session RuntimeSession, task tasklifecycle.Task, opts RunnerOptions) {
	logFile, err := openSessionLogFile(opts.RuntimeRoot, session)
	if err != nil {
		failLaunch(opts, session, task, err)
		return
	}
	defer logFile.Close()

	if err := validateSessionCommand(session); err != nil {
		failLaunch(opts, session, task, err)
		return
	}
	binary := session.Command[0]
	chatID := strings.TrimSpace(session.CursorChatID)
	if chatID == "" {
		session.Status = "starting"
		session.ExecutionStatus = "waiting_for_visible_session"
		session.LastEvent = "waiting_for_visible_session"
		upsertSession(opts, session)
		appendSessionLogLine(logFile, "[core] provider visible-session wait started (cursor-agent create-chat)\n")
		chatID, err = createCursorChat(ctx, opts, &session, logFile)
		if err != nil {
			failLaunch(opts, session, task, err)
			return
		}
	} else {
		session.ResumeAttempted = true
		session.ResumeOutcome = "attempting"
	}

	baseArgs := append([]string{}, session.Command[1:]...)
	command := append([]string{binary, "--resume", chatID}, baseArgs...)
	session.Command = command
	session.CursorChatID = chatID
	session.BackgroundID = chatID
	session.OperatorCommand = CursorOperatorCommand(binary, chatID, session.WorkingDir)
	session.VisibilityMode = string(execution.VisibilityCLIVisible)
	session.Capabilities = ProviderCapabilityFlags(session.Agent, session.Backend)
	session.StartedAt = time.Now().Format(time.RFC3339)
	session.Status = "running"
	session.ExecutionStatus = "running"
	session.LastEvent = "cursor_chat_ready"
	if session.ResumeAttempted {
		session.ResumeOutcome = "resumed"
	}
	upsertSession(opts, session)

	appendSessionLogLine(logFile, "\n[core] cursor chat id=%s\n", chatID)
	appendSessionLogLine(logFile, "[core] operator resume command: %s\n", session.OperatorCommand)
	appendSessionLogLine(logFile, "[core] starting cursor-visible agent=%s cwd=%s command=%q\n\n", session.Agent, session.WorkingDir, command)
	addTaskComment(opts, task, tasklifecycle.CursorSessionLiveComment(chatID, session.OperatorCommand, session.LogPath))

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	cmd := exec.CommandContext(runCtx, command[0], command[1:]...)
	cmd.Dir = session.WorkingDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "cursor-agent start", err, "")
		failLaunch(opts, session, task, err)
		return
	}
	session.ProcessID = cmd.Process.Pid
	session.LastEvent = "process_started"
	recordProviderStart(opts, session, task, "Cursor visible agent process started", map[string]any{
		"claim_id":       session.ClaimID,
		"process_id":     session.ProcessID,
		"cursor_chat_id": chatID,
		"log_path":       session.LogPath,
	})

	err = cmd.Wait()
	session.ExitedAt = time.Now().Format(time.RFC3339)
	settleCursorStatus(opts, &session, err, runCtx.Err())
	if session.Status == "failed" && session.ErrorMessage != "" {
		appendSessionLogLine(logFile, "\n[core] cursor session failed: %s\n", session.ErrorMessage)
	}
	adoptCursorResult(opts, &session)
	completeProviderSession(opts, session, task, firstNonEmpty(session.ErrorMessage, "Cursor visible agent process exited"), map[string]any{
		"process_id":     session.ProcessID,
		"cursor_chat_id": chatID,
		"exit_code":      session.ExitCode,
	})
}

// createCursorChat runs `cursor-agent create-chat` and returns the new chat id.
// On failure it records the classified provider error on session.
func createCursorChat(ctx context.Context, opts RunnerOptions, session *RuntimeSession, logFile io.Writer) (string, error) {
	binary := session.Command[0]
	createCtx, cancelCreate := context.WithTimeout(ctx, 30*time.Second)
	defer cancelCreate()
	var createOutput bytes.Buffer
	cmd := exec.CommandContext(createCtx, binary, "create-chat")
	cmd.Dir = session.WorkingDir
	cmd.Stdout = io.MultiWriter(&createOutput, logFile)
	cmd.Stderr = io.MultiWriter(&createOutput, logFile)
	appendSessionLogLine(logFile, "[core] creating cursor chat id command=%q cwd=%s\n\n", []string{binary, "create-chat"}, session.WorkingDir)
	if err := cmd.Run(); err != nil {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "cursor-agent create-chat", err, createOutput.String())
		return "", fmt.Errorf("cursor chat creation failed: %w", err)
	}
	chatID := ParseCursorChatID(createOutput.String())
	if chatID == "" {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "cursor-agent create-chat", nil, createOutput.String())
		return "", errors.New("could not parse Cursor chat id from `cursor-agent create-chat` output")
	}
	return chatID, nil
}

// settleCursorStatus turns how the process ended into the session status.
func settleCursorStatus(opts RunnerOptions, session *RuntimeSession, waitErr error, runErr error) {
	session.Status = "exited"
	session.ExecutionStatus = "succeeded"
	session.LastEvent = "process_exited"
	if waitErr == nil {
		return
	}
	session.Status = "failed"
	session.ExecutionStatus = "failed"
	session.ErrorMessage = waitErr.Error()
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		session.ExitCode = exitErr.ExitCode()
	}
	if runErr != nil {
		session.ExecutionStatus = "cancelled"
		session.ErrorMessage = runErr.Error()
	}
	session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "cursor-agent output", waitErr, readSessionLog(opts.RuntimeRoot, session.LogPath))
	if session.ProviderError != nil {
		session.ExecutionStatus = "provider_error"
	}
}

// adoptCursorResult takes the worker result from the session log and pauses the
// session when the worker asked for input.
func adoptCursorResult(opts RunnerOptions, session *RuntimeSession) {
	transcript := readSessionLog(opts.RuntimeRoot, session.LogPath)
	if session.Result == nil {
		if transcript != "" {
			refreshSessionToolUsageFromText(session, transcript)
			adoptWorkerResult(session, tasklifecycle.ParseWorkerResult(transcript))
		}
	} else {
		refreshSessionToolUsageFromText(session, transcript)
	}
	settleReportedResult(session)
}
