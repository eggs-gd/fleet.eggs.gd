package providers

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	execution "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

const methodTurnStart = "turn/start"

const methodTurnInterrupt = "turn/interrupt"

func RunCodexAppServerSession(ctx context.Context, session RuntimeSession, task tasklifecycle.Task, opts RunnerOptions) {
	logFile, err := runtimedb.OpenLog(opts.RuntimeRoot, session.LogPath, true)
	if err != nil {
		failLaunch(opts, session, task, err)
		return
	}
	defer logFile.Close()
	if len(session.Command) == 0 {
		failLaunch(opts, session, task, errors.New("launch command is empty"))
		return
	}

	idleTimeout := opts.SessionTimeout
	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(sessionCtx, session.Command[0], session.Command[1:]...)
	cmd.Dir = session.WorkingDir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		failLaunch(opts, session, task, err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		failLaunch(opts, session, task, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		failLaunch(opts, session, task, err)
		return
	}
	_, _ = fmt.Fprintf(logFile, "[core] starting codex app-server agent=%s binary=%s cwd=%s idle_timeout=%s command=%q\n\n", session.Agent, session.Command[0], session.WorkingDir, idleTimeout, session.Command)
	if err := cmd.Start(); err != nil {
		failLaunch(opts, session, task, err)
		return
	}

	session.ProcessID = cmd.Process.Pid
	session.StartedAt = time.Now().Format(time.RFC3339)
	session.Status = "starting"
	session.ExecutionStatus = "waiting_for_visible_session"
	session.LastEvent = "waiting_for_visible_session"
	session.VisibilityMode = firstNonEmpty(session.VisibilityMode, string(execution.VisibilityFleetVisible))
	session.Capabilities = ProviderCapabilityFlags(session.Agent, session.Backend)
	upsertSession(opts, session)
	_, _ = fmt.Fprintf(logFile, "[core] provider visible-session wait started (codex thread/start handshake)\n")
	recordProviderStart(opts, session, task, "Codex app-server process started; waiting for visible session", map[string]any{
		"claim_id":        session.ClaimID,
		"process_id":      session.ProcessID,
		"backend":         BackendCodexAppServer,
		"visibility_mode": session.VisibilityMode,
		"log_path":        session.LogPath,
		"handshake":       "waiting_for_visible_session",
	})

	messages := make(chan CodexRPCMessage, 64)
	readDone := make(chan error, 1)
	go readCodexStdout(stdout, logFile, messages, readDone)
	go copyCodexStderr(stderr, logFile)

	client := NewCodexClient(stdin, logFile)
	controls := make(chan SessionControlRequest, 8)
	if opts.Callbacks.RegisterSessionControl != nil {
		opts.Callbacks.RegisterSessionControl(session.ClaimID, controls)
		defer opts.Callbacks.UnregisterSessionControl(session.ClaimID)
	}
	if err := driveCodexTurn(sessionCtx, cancel, client, messages, controls, &session, task, idleTimeout, opts); err != nil {
		if session.CodexThreadID != "" && session.CodexTurnID != "" {
			_, _ = client.request(methodTurnInterrupt, map[string]any{
				"threadId": session.CodexThreadID,
				"turnId":   session.CodexTurnID,
			})
		}
		cancel()
		_ = cmd.Wait()
		if session.ExecutionStatus != "operator_attention" && session.ExecutionStatus != "cancelled" {
			session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "codex app-server", err, readSessionLog(opts.RuntimeRoot, session.LogPath))
			if session.ProviderError != nil {
				session.ExecutionStatus = "provider_error"
			}
		}
		finishCodexAppServerSession(opts, session, task, "failed", err.Error())
		return
	}

	_ = stdin.Close()
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
		cancel()
		<-waitDone
	}
	select {
	case <-readDone:
	default:
	}
	finishCodexAppServerSession(opts, session, task, "exited", "")
}

func HandleCodexSessionControl(client *CodexClient, session *RuntimeSession, control SessionControlRequest) error {
	switch control.Action {
	case "interrupt":
		if session.CodexThreadID == "" || session.CodexTurnID == "" {
			return errors.New("codex interrupt requires a known thread id and turn id")
		}
		_, err := client.request(methodTurnInterrupt, map[string]any{"threadId": session.CodexThreadID, "turnId": session.CodexTurnID})
		session.LastEvent = "operator_interrupt"
		session.ExecutionStatus = "running"
		session.BlockingReason = ""
		return err
	case "cancel":
		if session.CodexThreadID != "" && session.CodexTurnID != "" {
			_, _ = client.request(methodTurnInterrupt, map[string]any{"threadId": session.CodexThreadID, "turnId": session.CodexTurnID})
		}
		session.LastEvent = "operator_cancel"
		session.ExecutionStatus = "cancelled"
		session.BlockingReason = ""
		return nil
	case "input", "continue", "resume":
		if session.CodexThreadID == "" {
			return errors.New("codex input requires a known thread id")
		}
		input := strings.TrimSpace(control.Input)
		if input == "" {
			return errors.New("codex input/continue requires non-empty input")
		}
		id, err := client.request(methodTurnStart, codexTurnStartParams(session.CodexThreadID, input))
		if err != nil {
			return err
		}
		session.CodexTurnID = ""
		session.LastEvent = fmt.Sprintf("operator_%s:%d", control.Action, id)
		session.ExecutionStatus = "running"
		session.BlockingReason = ""
		return nil
	default:
		return fmt.Errorf("unsupported session control action %q", control.Action)
	}
}

func finishCodexAppServerSession(opts RunnerOptions, session RuntimeSession, task tasklifecycle.Task, status string, errorMessage string) {
	session.Status = status
	if session.ExecutionStatus == "" {
		session.ExecutionStatus = tasklifecycle.ExecutionStatusFor(status, errorMessage)
	}
	if status == "failed" && session.ProviderError == nil {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "codex app-server", errors.New(errorMessage), readSessionLog(opts.RuntimeRoot, session.LogPath))
		if session.ProviderError != nil {
			session.ExecutionStatus = "provider_error"
		}
	}
	session.ErrorMessage = errorMessage
	session.ExitedAt = time.Now().Format(time.RFC3339)
	session.LastEvent = firstNonEmpty(session.LastEvent, "app_server_ended")
	logText := readSessionLog(opts.RuntimeRoot, session.LogPath)
	refreshSessionToolUsageFromText(&session, logText)
	if session.Result == nil {
		if result := tasklifecycle.ParseWorkerResult(logText); result != nil {
			session.Result = result
			session.LastEvent = "worker_result_reported"
			if summary := strings.TrimSpace(result.Summary); summary != "" {
				session.LastMessage = summary
			}
		}
	}
	if session.Result != nil && session.ExecutionStatus != "cancelled" && session.ProviderError == nil && status != "failed" {
		if tasklifecycle.WorkerResultNeedsInput(session.Result) {
			session.ExecutionStatus = "waiting_input"
			session.Status = "waiting_input"
			session.ExitedAt = ""
		} else {
			session.ExecutionStatus = "succeeded"
		}
	}
	_ = runtimedb.AppendLog(opts.RuntimeRoot, session.LogPath, fmt.Sprintf("\n[core] codex app-server session ended status=%s %s\n", status, errorMessage))
	completeProviderSession(opts, session, task, firstNonEmpty(errorMessage, "Codex app-server session ended"), map[string]any{
		"claim_id":        session.ClaimID,
		"codex_thread_id": session.CodexThreadID,
		"codex_turn_id":   session.CodexTurnID,
		"status":          status,
		"exited_at":       session.ExitedAt,
		"log_path":        session.LogPath,
	})
}

func driveCodexTurn(ctx context.Context, cancel context.CancelFunc, client *CodexClient, messages <-chan CodexRPCMessage, controls <-chan SessionControlRequest, session *RuntimeSession, task tasklifecycle.Task, idleTimeout time.Duration, opts RunnerOptions) error {
	id, err := client.request("initialize", codexInitializeParams())
	if err != nil {
		return err
	}
	if _, err := waitForCodexResponse(ctx, messages, id, client, session, task, idleTimeout, opts); err != nil {
		return err
	}
	if err := client.notify("initialized", nil); err != nil {
		return err
	}
	threadID, err := startOrResumeCodexThread(ctx, client, messages, session, task, idleTimeout, opts)
	if err != nil {
		return err
	}
	session.Status = "running"
	session.ExecutionStatus = "running"
	session.LastEvent = firstNonEmpty(session.LastEvent, "thread_ready")
	upsertSession(opts, *session)
	id, err = client.request(methodTurnStart, codexTurnStartParams(threadID, task.LaunchEvaluation.Prompt))
	if err != nil {
		return err
	}
	response, err := waitForCodexResponse(ctx, messages, id, client, session, task, idleTimeout, opts)
	if err != nil {
		return err
	}
	if turnID := CodexTurnID(response.Result); turnID != "" {
		session.CodexTurnID = turnID
		session.LastEvent = methodTurnStart
		upsertSession(opts, *session)
	}
	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()
	attentionNotified := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			reason := fmt.Sprintf("No Codex JSON-RPC activity observed for %s. Session was not interrupted; use continue, interrupt, or cancel.", idleTimeout)
			session.ExecutionStatus = "operator_attention"
			session.LastEvent = "idle_operator_attention"
			session.BlockingReason = reason
			upsertSession(opts, *session)
			if !attentionNotified {
				markOperatorAttention(opts, *session, task, reason)
				attentionNotified = true
			}
			resetTimer(timer, idleTimeout)
		case control := <-controls:
			if err := HandleCodexSessionControl(client, session, control); err != nil {
				control.Done <- err
				continue
			}
			upsertSession(opts, *session)
			control.Done <- nil
			if control.Action == "cancel" {
				session.ExecutionStatus = "cancelled"
				upsertSession(opts, *session)
				cancel()
				return context.Canceled
			}
			attentionNotified = false
			resetTimer(timer, idleTimeout)
		case msg, ok := <-messages:
			if !ok {
				return errors.New("codex app-server stdout closed before turn/completed")
			}
			attentionNotified = false
			resetTimer(timer, idleTimeout)
			if msg.Method != "" && len(msg.ID) > 0 {
				_ = client.respondError(msg.ID, -32601, "Fleet launcher does not service Codex server-initiated requests")
				session.LastEvent = "server_request:" + msg.Method
				session.ExecutionStatus = "waiting_input"
				upsertSession(opts, *session)
				continue
			}
			if updateCodexSessionFromMessage(session, msg, opts) {
				if msg.Method == "turn/completed" {
					status := codexTurnStatus(msg.Params)
					if status == "" || status == "completed" {
						session.ExecutionStatus = "succeeded"
						return nil
					}
					if status == "interrupted" {
						session.ExecutionStatus = "cancelled"
					}
					return fmt.Errorf("codex turn completed with status %q", status)
				}
			}
		}
	}
}

func waitForCodexResponse(ctx context.Context, messages <-chan CodexRPCMessage, id int, client *CodexClient, session *RuntimeSession, task tasklifecycle.Task, idleTimeout time.Duration, opts RunnerOptions) (CodexRPCMessage, error) {
	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()
	attentionNotified := false
	for {
		select {
		case <-ctx.Done():
			return CodexRPCMessage{}, ctx.Err()
		case <-timer.C:
			reason := fmt.Sprintf("No Codex JSON-RPC response observed for %s. Session was not interrupted; operator attention may be required.", idleTimeout)
			session.ExecutionStatus = "operator_attention"
			session.LastEvent = "idle_operator_attention"
			session.BlockingReason = reason
			upsertSession(opts, *session)
			if !attentionNotified {
				markOperatorAttention(opts, *session, task, reason)
				attentionNotified = true
			}
			resetTimer(timer, idleTimeout)
		case msg, ok := <-messages:
			if !ok {
				return CodexRPCMessage{}, errors.New("codex app-server stdout closed before response")
			}
			attentionNotified = false
			resetTimer(timer, idleTimeout)
			if msg.Method != "" && len(msg.ID) > 0 {
				_ = client.respondError(msg.ID, -32601, "Fleet launcher does not service Codex server-initiated requests")
				session.LastEvent = "server_request:" + msg.Method
				session.ExecutionStatus = "waiting_input"
				upsertSession(opts, *session)
				continue
			}
			if codexMessageID(msg) == id {
				if msg.Error != nil {
					return CodexRPCMessage{}, errors.New(msg.Error.Message)
				}
				return msg, nil
			}
			updateCodexSessionFromMessage(session, msg, opts)
		}
	}
}

func startOrResumeCodexThread(ctx context.Context, client *CodexClient, messages <-chan CodexRPCMessage, session *RuntimeSession, task tasklifecycle.Task, idleTimeout time.Duration, opts RunnerOptions) (string, error) {
	if strings.TrimSpace(session.CodexThreadID) != "" {
		id, err := client.request("thread/resume", codexThreadResumeParams(*session))
		if err == nil {
			response, waitErr := waitForCodexResponse(ctx, messages, id, client, session, task, idleTimeout, opts)
			if waitErr == nil {
				if threadID := CodexThreadID(response.Result); threadID != "" {
					session.CodexThreadID = threadID
					session.ResumeOutcome = "resumed"
					session.LastEvent = "thread/resume"
					upsertSession(opts, *session)
					if err := setCodexThreadName(ctx, client, messages, session, task, threadID, idleTimeout, opts); err != nil {
						return "", err
					}
					return threadID, nil
				}
			}
		}
		session.ResumeOutcome = "fallback_new"
		session.CodexThreadID = ""
		upsertSession(opts, *session)
	}

	id, err := client.request("thread/start", codexThreadStartParams(*session))
	if err != nil {
		return "", err
	}
	response, err := waitForCodexResponse(ctx, messages, id, client, session, task, idleTimeout, opts)
	if err != nil {
		return "", err
	}
	threadID := CodexThreadID(response.Result)
	if threadID == "" {
		return "", errors.New("codex app-server thread/start response did not include thread.id")
	}
	session.CodexThreadID = threadID
	session.LastEvent = "thread/start"
	upsertSession(opts, *session)
	if err := setCodexThreadName(ctx, client, messages, session, task, threadID, idleTimeout, opts); err != nil {
		return "", err
	}
	return threadID, nil
}

func setCodexThreadName(ctx context.Context, client *CodexClient, messages <-chan CodexRPCMessage, session *RuntimeSession, task tasklifecycle.Task, threadID string, idleTimeout time.Duration, opts RunnerOptions) error {
	title := strings.TrimSpace(firstNonEmpty(session.CodexThreadTitle, CodexRemoteThreadTitle(task.Ref, task.Title, task.ID)))
	if title == "" {
		return nil
	}
	session.CodexThreadTitle = title
	id, err := client.request("thread/name/set", codexThreadSetNameParams(threadID, title))
	if err != nil {
		return err
	}
	if _, err := waitForCodexResponse(ctx, messages, id, client, session, task, idleTimeout, opts); err != nil {
		return fmt.Errorf("codex thread/name/set failed: %w", err)
	}
	session.LastEvent = "thread/name/set"
	upsertSession(opts, *session)
	return nil
}
