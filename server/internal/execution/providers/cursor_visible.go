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
		createCtx, cancelCreate := context.WithTimeout(ctx, 30*time.Second)
		var createOutput bytes.Buffer
		cmd := exec.CommandContext(createCtx, binary, "create-chat")
		cmd.Dir = session.WorkingDir
		cmd.Stdout = io.MultiWriter(&createOutput, logFile)
		cmd.Stderr = io.MultiWriter(&createOutput, logFile)
		appendSessionLogLine(logFile, "[core] creating cursor chat id command=%q cwd=%s\n\n", []string{binary, "create-chat"}, session.WorkingDir)
		err := cmd.Run()
		cancelCreate()
		if err != nil {
			session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "cursor-agent create-chat", err, createOutput.String())
			failLaunch(opts, session, task, fmt.Errorf("cursor chat creation failed: %w", err))
			return
		}
		chatID = ParseCursorChatID(createOutput.String())
		if chatID == "" {
			session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "cursor-agent create-chat", nil, createOutput.String())
			failLaunch(opts, session, task, errors.New("could not parse Cursor chat id from `cursor-agent create-chat` output"))
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
	session.Status = "exited"
	session.ExecutionStatus = "succeeded"
	session.LastEvent = "process_exited"
	if err != nil {
		session.Status = "failed"
		session.ExecutionStatus = "failed"
		session.ErrorMessage = err.Error()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			session.ExitCode = exitErr.ExitCode()
		}
		if runCtx.Err() != nil {
			session.ExecutionStatus = "cancelled"
			session.ErrorMessage = runCtx.Err().Error()
		}
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "cursor-agent output", err, readSessionLog(opts.RuntimeRoot, session.LogPath))
		if session.ProviderError != nil {
			session.ExecutionStatus = "provider_error"
		}
	}
	if session.Status == "failed" && session.ErrorMessage != "" {
		appendSessionLogLine(logFile, "\n[core] cursor session failed: %s\n", session.ErrorMessage)
	}
	if session.Result == nil {
		if transcript := readSessionLog(opts.RuntimeRoot, session.LogPath); transcript != "" {
			refreshSessionToolUsageFromText(&session, transcript)
			if result := tasklifecycle.ParseWorkerResult(transcript); result != nil {
				session.Result = result
				session.LastEvent = "worker_result_reported"
				if summary := strings.TrimSpace(result.Summary); summary != "" {
					session.LastMessage = summary
				}
			}
		}
	} else {
		refreshSessionToolUsageFromText(&session, readSessionLog(opts.RuntimeRoot, session.LogPath))
	}
	if session.Result != nil && session.ExecutionStatus != "cancelled" && session.ProviderError == nil {
		if tasklifecycle.WorkerResultNeedsInput(session.Result) {
			session.ExecutionStatus = "waiting_input"
			session.Status = "waiting_input"
			session.ExitedAt = ""
		} else {
			session.ExecutionStatus = "succeeded"
		}
	}
	completeProviderSession(opts, session, task, firstNonEmpty(session.ErrorMessage, "Cursor visible agent process exited"), map[string]any{
		"process_id":     session.ProcessID,
		"cursor_chat_id": chatID,
		"exit_code":      session.ExitCode,
	})
}
