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
	defer logFile.Close() //nolint:errcheck // the session log; a failed close has nowhere to be reported
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
	if err := (&codexTurn{ctx: sessionCtx, cancel: cancel, client: client, messages: messages, controls: controls, session: &session, task: task, idleTimeout: idleTimeout, opts: opts}).drive(); err != nil {
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
	waitDone := make(chan struct{})
	go func() {
		_ = cmd.Wait() // the exit status was already read from the app-server's messages
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
		cancel()
		<-waitDone
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
		adoptWorkerResult(&session, tasklifecycle.ParseWorkerResult(logText))
	}
	if status != "failed" {
		settleReportedResult(&session)
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

// codexTurn is one Codex turn being driven over the app-server JSON-RPC.
type codexTurn struct {
	ctx         context.Context
	cancel      context.CancelFunc
	client      *CodexClient
	messages    <-chan CodexRPCMessage
	controls    <-chan SessionControlRequest
	session     *RuntimeSession
	task        tasklifecycle.Task
	idleTimeout time.Duration
	opts        RunnerOptions
}

// drive initializes the app-server, starts or resumes the thread, sends the
// task prompt and follows the turn until it completes.
func (t *codexTurn) drive() error {
	id, err := t.client.request("initialize", codexInitializeParams())
	if err != nil {
		return err
	}
	if _, err := t.waitForResponse(id); err != nil {
		return err
	}
	if err := t.client.notify("initialized", nil); err != nil {
		return err
	}
	threadID, err := t.startOrResumeThread()
	if err != nil {
		return err
	}
	t.session.Status = "running"
	t.session.ExecutionStatus = "running"
	t.session.LastEvent = firstNonEmpty(t.session.LastEvent, "thread_ready")
	upsertSession(t.opts, *t.session)
	id, err = t.client.request(methodTurnStart, codexTurnStartParams(threadID, t.task.LaunchEvaluation.Prompt))
	if err != nil {
		return err
	}
	response, err := t.waitForResponse(id)
	if err != nil {
		return err
	}
	if turnID := CodexTurnID(response.Result); turnID != "" {
		t.session.CodexTurnID = turnID
		t.session.LastEvent = methodTurnStart
		upsertSession(t.opts, *t.session)
	}
	return t.awaitCompletion()
}

// noteIdle flags the session for operator attention, notifying once per quiet spell.
func (t *codexTurn) noteIdle(reason string, notified *bool) {
	t.session.ExecutionStatus = "operator_attention"
	t.session.LastEvent = "idle_operator_attention"
	t.session.BlockingReason = reason
	upsertSession(t.opts, *t.session)
	if !*notified {
		markOperatorAttention(t.opts, *t.session, t.task, reason)
		*notified = true
	}
}

// refuseServerRequest answers a request the app-server sent us; the launcher
// services none, so the session waits for the operator.
func (t *codexTurn) refuseServerRequest(msg CodexRPCMessage) bool {
	if msg.Method == "" || len(msg.ID) == 0 {
		return false
	}
	_ = t.client.respondError(msg.ID, -32601, "Fleet launcher does not service Codex server-initiated requests")
	t.session.LastEvent = "server_request:" + msg.Method
	t.session.ExecutionStatus = "waiting_input"
	upsertSession(t.opts, *t.session)
	return true
}

// awaitCompletion follows the turn until turn/completed, the operator cancels,
// or the context ends.
func (t *codexTurn) awaitCompletion() error {
	timer := time.NewTimer(t.idleTimeout)
	defer timer.Stop()
	notified := false
	for {
		select {
		case <-t.ctx.Done():
			return t.ctx.Err()
		case <-timer.C:
			t.noteIdle(fmt.Sprintf("No Codex JSON-RPC activity observed for %s. Session was not interrupted; use continue, interrupt, or cancel.", t.idleTimeout), &notified)
			resetTimer(timer, t.idleTimeout)
		case control := <-t.controls:
			reset, err := t.handleControl(control)
			if err != nil {
				return err
			}
			if reset {
				notified = false
				resetTimer(timer, t.idleTimeout)
			}
		case msg, ok := <-t.messages:
			if !ok {
				return errors.New("codex app-server stdout closed before turn/completed")
			}
			notified = false
			resetTimer(timer, t.idleTimeout)
			if t.refuseServerRequest(msg) {
				continue
			}
			if updateCodexSessionFromMessage(t.session, msg, t.opts) && msg.Method == "turn/completed" {
				return t.completion(msg)
			}
		}
	}
}

// handleControl runs one operator control. reset says the idle timer should
// restart; a non-nil error ends the turn (cancel).
func (t *codexTurn) handleControl(control SessionControlRequest) (bool, error) {
	if err := HandleCodexSessionControl(t.client, t.session, control); err != nil {
		control.Done <- err
		return false, nil //nolint:nilerr // the control error goes back to the caller on control.Done
	}
	upsertSession(t.opts, *t.session)
	control.Done <- nil
	if control.Action != "cancel" {
		return true, nil
	}
	t.session.ExecutionStatus = "cancelled"
	upsertSession(t.opts, *t.session)
	t.cancel()
	return false, context.Canceled
}

func (t *codexTurn) completion(msg CodexRPCMessage) error {
	status := codexTurnStatus(msg.Params)
	if status == "" || status == "completed" {
		t.session.ExecutionStatus = "succeeded"
		return nil
	}
	if status == "interrupted" {
		t.session.ExecutionStatus = "cancelled"
	}
	return fmt.Errorf("codex turn completed with status %q", status)
}

func (t *codexTurn) waitForResponse(id int) (CodexRPCMessage, error) {
	timer := time.NewTimer(t.idleTimeout)
	defer timer.Stop()
	notified := false
	for {
		select {
		case <-t.ctx.Done():
			return CodexRPCMessage{}, t.ctx.Err()
		case <-timer.C:
			t.noteIdle(fmt.Sprintf("No Codex JSON-RPC response observed for %s. Session was not interrupted; operator attention may be required.", t.idleTimeout), &notified)
			resetTimer(timer, t.idleTimeout)
		case msg, ok := <-t.messages:
			if !ok {
				return CodexRPCMessage{}, errors.New("codex app-server stdout closed before response")
			}
			notified = false
			resetTimer(timer, t.idleTimeout)
			if t.refuseServerRequest(msg) {
				continue
			}
			if codexMessageID(msg) == id {
				if msg.Error != nil {
					return CodexRPCMessage{}, errors.New(msg.Error.Message)
				}
				return msg, nil
			}
			updateCodexSessionFromMessage(t.session, msg, t.opts)
		}
	}
}

// startOrResumeThread resumes the session's thread when it has one, and starts
// a new thread when there is none or the resume fails.
func (t *codexTurn) startOrResumeThread() (string, error) {
	if strings.TrimSpace(t.session.CodexThreadID) != "" {
		if threadID, ok := t.tryResumeThread(); ok {
			return threadID, t.nameThread(threadID)
		}
		t.session.ResumeOutcome = "fallback_new"
		t.session.CodexThreadID = ""
		upsertSession(t.opts, *t.session)
	}

	id, err := t.client.request("thread/start", codexThreadStartParams(*t.session))
	if err != nil {
		return "", err
	}
	response, err := t.waitForResponse(id)
	if err != nil {
		return "", err
	}
	threadID := CodexThreadID(response.Result)
	if threadID == "" {
		return "", errors.New("codex app-server thread/start response did not include thread.id")
	}
	t.session.CodexThreadID = threadID
	t.session.LastEvent = "thread/start"
	upsertSession(t.opts, *t.session)
	return threadID, t.nameThread(threadID)
}

func (t *codexTurn) tryResumeThread() (string, bool) {
	id, err := t.client.request("thread/resume", codexThreadResumeParams(*t.session))
	if err != nil {
		return "", false
	}
	response, err := t.waitForResponse(id)
	if err != nil {
		return "", false
	}
	threadID := CodexThreadID(response.Result)
	if threadID == "" {
		return "", false
	}
	t.session.CodexThreadID = threadID
	t.session.ResumeOutcome = "resumed"
	t.session.LastEvent = "thread/resume"
	upsertSession(t.opts, *t.session)
	return threadID, true
}

func (t *codexTurn) nameThread(threadID string) error {
	title := strings.TrimSpace(firstNonEmpty(t.session.CodexThreadTitle, CodexRemoteThreadTitle(t.task.Ref, t.task.Title, t.task.ID)))
	if title == "" {
		return nil
	}
	t.session.CodexThreadTitle = title
	id, err := t.client.request("thread/name/set", codexThreadSetNameParams(threadID, title))
	if err != nil {
		return err
	}
	if _, err := t.waitForResponse(id); err != nil {
		return fmt.Errorf("codex thread/name/set failed: %w", err)
	}
	t.session.LastEvent = "thread/name/set"
	upsertSession(t.opts, *t.session)
	return nil
}
