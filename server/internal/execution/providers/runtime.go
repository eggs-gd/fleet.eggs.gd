package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	execution "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

type RunnerCallbacks struct {
	ClassifyProviderError        func(provider string, backend string, operation string, err error, output string) *tasklifecycle.ProviderError
	FailLaunchSession            func(session RuntimeSession, task tasklifecycle.Task, err error)
	RecordProviderSessionStart   func(session RuntimeSession, task tasklifecycle.Task, message string, details map[string]any)
	CompleteProviderSession      func(session RuntimeSession, task tasklifecycle.Task, exitMessage string, details map[string]any)
	UpsertSession                func(session RuntimeSession)
	AddTaskComment               func(locator string, comment string) error
	MarkSessionOperatorAttention func(session RuntimeSession, task tasklifecycle.Task, reason string)
	ApplyWorkerReportedResult    func(session *RuntimeSession, task tasklifecycle.Task, result *tasklifecycle.WorkerResult)
	RegisterSessionControl       func(claimID string, ch chan SessionControlRequest)
	UnregisterSessionControl     func(claimID string)
}

type RunnerOptions struct {
	RuntimeRoot    string
	SessionTimeout time.Duration
	Callbacks      RunnerCallbacks
}

type RunningProcess struct {
	Cmd     *exec.Cmd
	LogFile io.WriteCloser
	Stdin   io.WriteCloser
	Context context.Context
	Cancel  context.CancelFunc
	Timeout time.Duration
}

func (process *RunningProcess) Close() {
	if process.Cancel != nil {
		process.Cancel()
	}
	if process.LogFile != nil {
		_ = process.LogFile.Close()
	}
	if process.Stdin != nil {
		_ = process.Stdin.Close()
	}
}

func StartProcess(ctx context.Context, plan Plan, logPath string, timeout time.Duration) (*RunningProcess, error) {
	if len(plan.Command) == 0 {
		return nil, errors.New("launch command is empty")
	}
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	logFile, err := runtimedb.OpenLog("", logPath, true)
	if err != nil {
		return nil, err
	}
	sessionCtx, cancel := context.WithTimeout(ctx, timeout)
	command := append([]string{}, plan.Command...)
	cmd := exec.CommandContext(sessionCtx, command[0], command[1:]...)
	cmd.Dir = plan.WorkingDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	_, _ = fmt.Fprintf(logFile, "[core] starting agent=%s backend=%s cwd=%s timeout=%s command=%q\n\n", plan.Agent, firstNonEmpty(plan.Backend, "process"), plan.WorkingDir, timeout, command)
	var stdin io.WriteCloser
	if plan.InitialInput != "" && plan.Backend != "terminal" {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			cancel()
			_ = logFile.Close()
			return nil, err
		}
	}
	if err := cmd.Start(); err != nil {
		_, _ = fmt.Fprintf(logFile, "[core] start failed: %s\n", err)
		cancel()
		_ = logFile.Close()
		return nil, err
	}
	if stdin != nil {
		go func() {
			time.Sleep(2 * time.Second)
			_, _ = fmt.Fprintf(logFile, "[core] writing initial stdin input bytes=%d and leaving stdin open\n\n", len(plan.InitialInput))
			if _, err := io.WriteString(stdin, plan.InitialInput); err != nil {
				_, _ = fmt.Fprintf(logFile, "[core] initial stdin write failed: %s\n", err)
			}
		}()
	}
	return &RunningProcess{
		Cmd:     cmd,
		LogFile: logFile,
		Stdin:   stdin,
		Context: sessionCtx,
		Cancel:  cancel,
		Timeout: timeout,
	}, nil
}

var backgroundSessionIDPattern = regexp.MustCompile(`backgrounded · (\S+) ·`)

// backgroundRemotePollInterval and backgroundRemoteControlReadyTimeout back
// BackgroundRemotePollInterval()/BackgroundRemoteControlReadyTimeout() below.
// They are atomic rather than plain vars because a test can reset them
// (SetBackgroundRemotePollIntervalForTest / …ReadyTimeoutForTest) while a
// background session goroutine from that same test's launch is still polling
// in RunBackgroundRemoteSession — a plain var there is an unsynchronized
// concurrent read/write, a real data race caught by `go test -race` that can
// corrupt unrelated state in whatever test happens to run next.
var backgroundRemotePollInterval atomic.Int64
var backgroundRemoteControlReadyTimeout atomic.Int64

func init() {
	backgroundRemotePollInterval.Store(int64(5 * time.Second))
	backgroundRemoteControlReadyTimeout.Store(int64(2 * time.Minute))
}

// BackgroundRemotePollInterval is how often RunBackgroundRemoteSession polls
// `claude agents --json` / `claude logs` while a background session is live.
func BackgroundRemotePollInterval() time.Duration {
	return time.Duration(backgroundRemotePollInterval.Load())
}

// SetBackgroundRemotePollInterval overrides the poll interval; see
// execution.SetBackgroundRemotePollIntervalForTest for the test-facing
// save/restore wrapper.
func SetBackgroundRemotePollInterval(d time.Duration) {
	backgroundRemotePollInterval.Store(int64(d))
}

// BackgroundRemoteControlReadyTimeout is how long Core waits after a
// successful `claude --bg --remote-control` dispatch for the phone/app
// remote-control URL to appear in `claude logs`. A background id without a
// URL is a hollow session (CORE-142): nothing appears on desktop/mobile and
// the transcript never grows. Fail that case as provider_error instead of
// waiting the full idle-attention threshold with a misleading "open the
// remote session" comment.
func BackgroundRemoteControlReadyTimeout() time.Duration {
	return time.Duration(backgroundRemoteControlReadyTimeout.Load())
}

// SetBackgroundRemoteControlReadyTimeout overrides the ready timeout; see
// execution.SetBackgroundRemoteControlReadyTimeoutForTest for the test-facing
// save/restore wrapper.
func SetBackgroundRemoteControlReadyTimeout(d time.Duration) {
	backgroundRemoteControlReadyTimeout.Store(int64(d))
}

type BackgroundAgentStatus struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	State     string `json:"state"`
	SessionID string `json:"sessionId"`
}

func RunBackgroundRemoteSession(ctx context.Context, session RuntimeSession, task tasklifecycle.Task, opts RunnerOptions) {
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
	command := session.Command
	claudeBinary := command[0]
	resuming := strings.TrimSpace(session.SessionID) != ""
	if resuming {
		// 1-1-1 session reuse (_docs/AGENT_SESSION_REUSE.md): resume must be
		// the full session UUID with no other flags repeated (--remote-control
		// / --name / --permission-mode are restored from the session's own
		// saved options) — passing them again, or the short BackgroundID
		// instead of SessionID, forks a copy instead of continuing the same
		// session. The prompt is always the original command's last argv.
		prompt := command[len(command)-1]
		command = []string{claudeBinary, "--bg", "--resume", session.SessionID, prompt}
		session.Command = command
	}
	appendSessionLogLine(logFile, "[core] dispatching background-remote agent=%s cwd=%s resuming=%v command=%q\n\n", session.Agent, session.WorkingDir, resuming, command)

	dispatchCtx, cancelDispatch := context.WithTimeout(ctx, 30*time.Second)
	defer cancelDispatch()
	var dispatchOutput bytes.Buffer
	cmd := exec.CommandContext(dispatchCtx, command[0], command[1:]...)
	cmd.Dir = session.WorkingDir
	cmd.Stdout = io.MultiWriter(&dispatchOutput, logFile)
	cmd.Stderr = io.MultiWriter(&dispatchOutput, logFile)
	if err := cmd.Run(); err != nil {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "background dispatch", err, dispatchOutput.String())
		failLaunch(opts, session, task, fmt.Errorf("background dispatch failed: %w", err))
		return
	}

	match := backgroundSessionIDPattern.FindStringSubmatch(dispatchOutput.String())
	if len(match) < 2 {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "background dispatch", nil, dispatchOutput.String())
		failLaunch(opts, session, task, errors.New("could not parse background session id from `claude --bg` dispatch output"))
		return
	}
	backgroundID := match[1]

	// A resume that actually forked (e.g. the prior session was somehow
	// still running instead of stopped) comes back as a *different*
	// background id with a "started a copy"/"already running" note instead
	// of a "woke session ... with its saved options" one — Core's own
	// 1-1-1 invariant (_docs/AGENT_SESSION_REUSE.md) just silently broke.
	// This case is not handled beyond logging it: reconciling two live
	// identities for the same (project, agent) is out of scope until the
	// git-flow/worktree question for N>1 agents is settled.
	if resuming && session.BackgroundID != "" && backgroundID != session.BackgroundID {
		appendSessionLogLine(logFile, "[core] WARNING: resume of %s came back as a different background id %s — the prior session was not actually stopped, or Claude forked a copy anyway. Two sessions may now exist for this (project, agent); this is not reconciled automatically.\n", session.BackgroundID, backgroundID)
	}
	session.BackgroundID = backgroundID
	session.StartedAt = time.Now().Format(time.RFC3339)
	session.Status = "starting"
	session.ExecutionStatus = "waiting_for_visible_session"
	session.LastEvent = "waiting_for_visible_session"
	session.VisibilityMode = firstNonEmpty(session.VisibilityMode, string(execution.VisibilityAppVisible))
	session.Capabilities = ProviderCapabilityFlags(session.Agent, session.Backend)
	recordProviderStart(opts, session, task, "Background remote-control agent dispatched; waiting for visible session", map[string]any{
		"claim_id":        session.ClaimID,
		"background_id":   backgroundID,
		"visibility_mode": session.VisibilityMode,
		"log_path":        session.LogPath,
		"handshake":       "waiting_for_visible_session",
	})
	appendSessionLogLine(logFile, "[core] background session id=%s\n", backgroundID)
	appendSessionLogLine(logFile, "[core] provider visible-session wait started (remote-control URL registration)\n")

	idleTimeout := opts.SessionTimeout
	readyTimeout := BackgroundRemoteControlReadyTimeout()
	if readyTimeout <= 0 {
		readyTimeout = 2 * time.Minute
	}
	if idleTimeout > 0 && readyTimeout > idleTimeout {
		readyTimeout = idleTimeout
	}
	urlAnnounced := false
	lastActivity := time.Now()
	lastLogSize := -1
	lastState := ""
	attentionNotified := false
	resultApplied := false

	for {
		select {
		case <-ctx.Done():
			_ = StopBackgroundSession(claudeBinary, backgroundID)
			finishBackgroundRemoteSession(opts, session, task, claudeBinary, "cancelled", "runtime shutting down")
			return
		default:
		}

		if !urlAnnounced {
			url, logErr := ExtractRemoteControlURL(claudeBinary, backgroundID)
			if logErr != nil {
				pe := classifyProviderError(opts, session.Agent, session.Backend, "claude logs", logErr, "")
				if pe != nil && pe.Kind == "control_socket_unavailable" {
					session.ProviderError = pe
					finishBackgroundRemoteSession(opts, session, task, claudeBinary, "failed", pe.Reason)
					return
				}
			}
			if url != "" {
				urlAnnounced = true
				lastActivity = time.Now()
				attentionNotified = false
				session.RemoteControlURL = url
				session.Status = "running"
				session.ExecutionStatus = "running"
				session.LastEvent = "remote_control_ready"
				session.VisibilityMode = string(execution.VisibilityAppVisible)
				session.BlockingReason = ""
				upsertSession(opts, session)
				appendSessionLogLine(logFile, "[core] remote-control url: %s\n", url)
				addTaskComment(opts, task, tasklifecycle.RemoteControlSessionLiveComment(url))
			}
		}

		status, found, statusErr := QueryBackgroundAgentStatus(claudeBinary, backgroundID)
		if statusErr != nil {
			pe := classifyProviderError(opts, session.Agent, session.Backend, "claude agents --json", statusErr, "")
			if pe != nil && pe.Kind == "control_socket_unavailable" {
				session.ProviderError = pe
				finishBackgroundRemoteSession(opts, session, task, claudeBinary, "failed", pe.Reason)
				return
			}
		}
		if found && session.SessionID == "" && status.SessionID != "" {
			// Full session UUID, needed by --resume for project+agent session
			// reuse (_docs/AGENT_SESSION_REUSE.md) — BackgroundID alone forks
			// a copy instead of continuing the same session.
			session.SessionID = status.SessionID
			upsertSession(opts, session)
		}
		if found && IsTerminalBackgroundState(status) {
			captureBackgroundWorkerResult(opts, &session, task, claudeBinary, backgroundID, &resultApplied)
			if strings.EqualFold(status.State, "error") || strings.EqualFold(status.State, "failed") {
				finishBackgroundRemoteSession(opts, session, task, claudeBinary, "failed", "background agent ended with state "+status.State)
			} else {
				finishBackgroundRemoteSession(opts, session, task, claudeBinary, "exited", "")
			}
			return
		}
		if !found {
			captureBackgroundWorkerResult(opts, &session, task, claudeBinary, backgroundID, &resultApplied)
			finishBackgroundRemoteSession(opts, session, task, claudeBinary, "exited", "session no longer listed by `claude agents`")
			return
		}
		state := strings.TrimSpace(status.State)
		if state != "" && state != lastState {
			lastState = state
			lastActivity = time.Now()
			attentionNotified = false
			session.LastEvent = "state:" + state
			if urlAnnounced {
				session.ExecutionStatus = "running"
				session.Status = "running"
			} else {
				session.ExecutionStatus = "waiting_for_visible_session"
				session.Status = "starting"
			}
			session.BlockingReason = ""
			upsertSession(opts, session)
		}

		if transcript, _ := FetchBackgroundSessionLogs(claudeBinary, backgroundID); transcript != "" {
			if size := len(transcript); size != lastLogSize {
				lastLogSize = size
				lastActivity = time.Now()
				attentionNotified = false
				session.LastEvent = "transcript_updated"
				session.LastMessage = fmt.Sprintf("transcript bytes=%d", size)
				if urlAnnounced {
					session.ExecutionStatus = "running"
					session.Status = "running"
				} else {
					session.ExecutionStatus = "waiting_for_visible_session"
					session.Status = "starting"
				}
				session.BlockingReason = ""
				refreshSessionToolUsageFromText(&session, transcript)
				upsertSession(opts, session)
			}
			if !resultApplied {
				if result := tasklifecycle.ParseWorkerResult(transcript); result != nil {
					resultApplied = true
					applyWorkerReportedResult(opts, &session, task, result)
					// 1-1-1 session reuse (_docs/AGENT_SESSION_REUSE.md): once
					// this task's outcome is in, stop the background process
					// so the next task for the same (project, agent) can
					// resume this exact session instead of forking a copy —
					// `claude --bg --resume` only continues under the same id
					// when the prior process is already stopped. Skip this
					// for needs_input/waiting_input: the operator may want to
					// answer directly in the still-live phone/app session.
					if !tasklifecycle.WorkerResultNeedsInput(result) {
						_ = StopBackgroundSession(claudeBinary, backgroundID)
						finishBackgroundRemoteSession(opts, session, task, claudeBinary, "exited", "stopped after task outcome; resumable for the next task in this project")
						return
					}
				}
			}
		}

		// Hollow session (CORE-142 / CORE-150): background id exists but
		// Claude never published an app-visible URL. Prefer a concrete
		// auth/quota/billing classification from provider output; only fall
		// back to remote_control_unavailable when nothing more specific is
		// recognizable.
		if !urlAnnounced && time.Since(lastActivity) > readyTimeout {
			transcript, _ := FetchBackgroundSessionLogs(claudeBinary, backgroundID)
			if excerpt := truncateForSessionLog(transcript, 1200); excerpt != "" {
				appendSessionLogLine(logFile, "[core] claude logs excerpt while waiting for remote-control URL:\n%s\n", excerpt)
			}
			if pe := classifyProviderError(opts, session.Agent, session.Backend, "claude logs", nil, transcript); pe != nil {
				session.ProviderError = pe
				session.LastEvent = pe.Kind
				session.BlockingReason = pe.Reason
				appendSessionLogLine(logFile, "[core] classified hollow-session provider failure kind=%s: %s\n", pe.Kind, pe.Reason)
				_ = StopBackgroundSession(claudeBinary, backgroundID)
				finishBackgroundRemoteSession(opts, session, task, claudeBinary, "failed", pe.Reason)
				return
			}
			reason := fmt.Sprintf("Claude background session %s never published a remote-control URL within %s (app-visible registration failed). Nothing will appear on Claude desktop/mobile until the provider publishes Continue-here/session URL. Check Claude Desktop login, `claude agents --json`, and `claude logs %s`.", backgroundID, readyTimeout, backgroundID)
			appendSessionLogLine(logFile, "[core] %s\n", reason)
			session.ProviderError = tasklifecycle.NewProviderError(session.Agent, "remote_control_unavailable", "claude logs", reason, truncateForSessionLog(transcript, 360))
			session.LastEvent = "remote_control_unavailable"
			session.BlockingReason = reason
			_ = StopBackgroundSession(claudeBinary, backgroundID)
			finishBackgroundRemoteSession(opts, session, task, claudeBinary, "failed", reason)
			return
		}

		if idle := time.Since(lastActivity); idle > idleTimeout {
			if !attentionNotified {
				reason := fmt.Sprintf("No Claude activity observed for %s (idle attention threshold %s). Session was not stopped; open the remote session or cancel explicitly.", idle.Round(time.Second), idleTimeout)
				appendSessionLogLine(logFile, "[core] %s\n", reason)
				markOperatorAttention(opts, session, task, reason)
				session.ExecutionStatus = "operator_attention"
				session.LastEvent = "idle_operator_attention"
				session.BlockingReason = reason
				upsertSession(opts, session)
				attentionNotified = true
			}
		}

		select {
		case <-ctx.Done():
		case <-time.After(BackgroundRemotePollInterval()):
		}
	}
}

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

// RunGeminiHeadlessSession runs one turn through Google's Antigravity CLI
// (`agy`). Unlike the other three backends this is a plain synchronous
// subprocess: `agy --print <prompt> --output-format stream-json` blocks
// until the turn's terminal `result` event and exits — there is no daemon
// to attach to and no background polling. On the first launch for a working
// directory Core also registers it as an antigravity project (`--new-project`)
// so AGENTS.md/GEMINI.md and project-scoped `.agents/mcp_config.json` load;
// every later launch in that directory reuses the discovered project id.
func RunGeminiHeadlessSession(ctx context.Context, session RuntimeSession, task tasklifecycle.Task, opts RunnerOptions) {
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
	baseArgs := append([]string{}, session.Command[1:]...)

	projectID := strings.TrimSpace(session.GeminiProjectID)
	if projectID == "" {
		projectID = findGeminiProjectID(session.WorkingDir)
	}
	conversationID := strings.TrimSpace(session.GeminiConversationID)

	var args []string
	if projectID != "" {
		args = append([]string{"--project", projectID}, baseArgs...)
	} else {
		// First launch in this directory: register it as an antigravity
		// project as part of the real turn (not a separate throwaway call)
		// so AGENTS.md/GEMINI.md and .agents/mcp_config.json load. The
		// created project id is discovered afterward from
		// ~/.gemini/config/projects/*.json — `agy` does not print it.
		args = append([]string{"--new-project"}, baseArgs...)
	}
	if conversationID != "" {
		args = append(args, "--conversation", conversationID)
		session.ResumeAttempted = true
		session.ResumeOutcome = "attempting"
	}
	command := append([]string{binary}, args...)
	session.Command = command
	session.Capabilities = ProviderCapabilityFlags(session.Agent, session.Backend)
	session.StartedAt = time.Now().Format(time.RFC3339)
	session.Status = "running"
	session.ExecutionStatus = "running"
	session.LastEvent = "process_started"
	upsertSession(opts, session)

	appendSessionLogLine(logFile, "[core] starting gemini-headless agent=%s cwd=%s project_id=%q conversation_id=%q command=%q\n\n",
		session.Agent, session.WorkingDir, projectID, conversationID, command)

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	cmd := exec.CommandContext(runCtx, command[0], command[1:]...)
	cmd.Dir = session.WorkingDir
	var stdout bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, logFile)
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "agy start", err, "")
		failLaunch(opts, session, task, err)
		return
	}
	session.ProcessID = cmd.Process.Pid
	recordProviderStart(opts, session, task, "Gemini headless agent process started", map[string]any{
		"claim_id":   session.ClaimID,
		"process_id": session.ProcessID,
		"project_id": projectID,
		"log_path":   session.LogPath,
	})

	err = cmd.Wait()
	session.ExitedAt = time.Now().Format(time.RFC3339)

	event := parseGeminiStreamJSON(stdout.Bytes())
	if event.ConversationID != "" {
		session.GeminiConversationID = event.ConversationID
	}
	if projectID == "" {
		if discovered := findGeminiProjectID(session.WorkingDir); discovered != "" {
			session.GeminiProjectID = discovered
			appendSessionLogLine(logFile, "\n[core] registered antigravity project id=%s for cwd=%s\n", discovered, session.WorkingDir)
		} else {
			appendSessionLogLine(logFile, "\n[core] could not discover an antigravity project id for cwd=%s after --new-project\n", session.WorkingDir)
		}
	} else {
		session.GeminiProjectID = projectID
	}

	session.Status = "exited"
	switch {
	case err != nil:
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
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "agy output", err, readSessionLog(opts.RuntimeRoot, session.LogPath))
		if session.ProviderError != nil {
			session.ExecutionStatus = "provider_error"
		}
	case event.Status == "":
		session.ExecutionStatus = "provider_error"
		session.ErrorMessage = "agy exited without a terminal stream-json `result` event"
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "agy output", nil, readSessionLog(opts.RuntimeRoot, session.LogPath))
	case event.Status == "SUCCESS":
		session.ExecutionStatus = "succeeded"
	case event.Status == "WAITING":
		session.Status = "waiting_input"
		session.ExecutionStatus = "waiting_input"
		session.ExitedAt = ""
	case event.Status == "CANCELED" || event.Status == "INTERRUPTED":
		session.ExecutionStatus = "cancelled"
	default: // ERROR, INVALID, RUNNING, or an unrecognized status
		session.ExecutionStatus = "failed"
		session.ErrorMessage = firstNonEmpty(event.ErrorText, "agy turn ended with status "+event.Status)
	}
	if session.Status == "failed" && session.ErrorMessage != "" {
		appendSessionLogLine(logFile, "\n[core] gemini session failed: %s\n", session.ErrorMessage)
	}

	if session.Result == nil && event.Response != "" {
		refreshSessionToolUsageFromText(&session, event.Response)
		if result := tasklifecycle.ParseWorkerResult(event.Response); result != nil {
			session.Result = result
			session.LastEvent = "worker_result_reported"
			if summary := strings.TrimSpace(result.Summary); summary != "" {
				session.LastMessage = summary
			}
		} else if session.LastMessage == "" {
			session.LastMessage = event.Response
		}
	} else if session.Result != nil {
		refreshSessionToolUsageFromText(&session, readSessionLog(opts.RuntimeRoot, session.LogPath))
	}
	if session.Result != nil && session.ExecutionStatus != "cancelled" && session.ProviderError == nil {
		if tasklifecycle.WorkerResultNeedsInput(session.Result) {
			session.ExecutionStatus = "waiting_input"
			session.Status = "waiting_input"
			session.ExitedAt = ""
		} else if session.ExecutionStatus == "succeeded" {
			session.ExecutionStatus = "succeeded"
		}
	}

	completeProviderSession(opts, session, task, firstNonEmpty(session.ErrorMessage, "Gemini headless agent process exited"), map[string]any{
		"process_id":      session.ProcessID,
		"project_id":      session.GeminiProjectID,
		"conversation_id": session.GeminiConversationID,
		"exit_code":       session.ExitCode,
	})
}

// geminiResultEvent is the terminal event of an `agy --output-format
// stream-json` run (one NDJSON object per line; the last `"event":"result"`
// line wins, matching how a well-formed stream only emits one).
type geminiResultEvent struct {
	ConversationID string
	Status         string
	Response       string
	ErrorText      string
}

func parseGeminiStreamJSON(raw []byte) geminiResultEvent {
	var out geminiResultEvent
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var envelope struct {
			Event  string `json:"event"`
			Result *struct {
				ConversationID string          `json:"conversation_id"`
				Status         string          `json:"status"`
				Response       string          `json:"response"`
				Error          json.RawMessage `json:"error,omitempty"`
			} `json:"result"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue
		}
		if envelope.Event != "result" || envelope.Result == nil {
			continue
		}
		out.ConversationID = envelope.Result.ConversationID
		out.Status = envelope.Result.Status
		out.Response = envelope.Result.Response
		if len(envelope.Result.Error) > 0 && string(envelope.Result.Error) != "null" {
			out.ErrorText = string(envelope.Result.Error)
		}
	}
	return out
}

// geminiProjectRecord mirrors the fields Core reads from one
// ~/.gemini/config/projects/<id>.json file. `agy` owns this registry; Core
// only reads it to find a project id it already created for a directory —
// see RunGeminiHeadlessSession's `--new-project` comment for why this
// filesystem lookup exists instead of the CLI printing the id directly.
type geminiProjectRecord struct {
	ID               string `json:"id"`
	ProjectResources struct {
		Resources []struct {
			FolderURI string `json:"folderUri"`
		} `json:"resources"`
	} `json:"projectResources"`
}

// findGeminiProjectID returns the most recently created antigravity project
// registered for workingDir, or "" if none is found. Multiple project files
// can reference the same folder (agy's `--new-project` is not idempotent
// per directory), so this picks the newest by file modification time.
func findGeminiProjectID(workingDir string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	absDir, err := filepath.Abs(workingDir)
	if err != nil {
		absDir = workingDir
	}
	wantURI := "file://" + absDir
	matches, err := filepath.Glob(filepath.Join(home, ".gemini", "config", "projects", "*.json"))
	if err != nil {
		return ""
	}
	var bestPath, bestID string
	var bestModTime time.Time
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var record geminiProjectRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}
		matched := false
		for _, resource := range record.ProjectResources.Resources {
			if resource.FolderURI == wantURI {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if bestPath == "" || info.ModTime().After(bestModTime) {
			bestPath = path
			bestID = record.ID
			bestModTime = info.ModTime()
		}
	}
	return bestID
}

type CodexRPCMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *CodexRPCError  `json:"error,omitempty"`
}

type CodexRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CodexClient struct {
	stdin   io.Writer
	logFile io.Writer
	mu      sync.Mutex
	nextID  int
}

func NewCodexClient(stdin io.Writer, logFile io.Writer) *CodexClient {
	return &CodexClient{stdin: stdin, logFile: logFile}
}

func CodexThreadStartParams(session RuntimeSession) map[string]any {
	return codexThreadStartParams(session)
}

func CodexThreadSetNameParams(threadID string, name string) map[string]any {
	return codexThreadSetNameParams(threadID, name)
}

func CodexTurnStartParams(threadID string, prompt string) map[string]any {
	return codexTurnStartParams(threadID, prompt)
}

func CodexTurnStatus(raw json.RawMessage) string {
	return codexTurnStatus(raw)
}

func UpdateCodexSessionFromMessage(session *RuntimeSession, msg CodexRPCMessage) bool {
	return updateCodexSessionFromMessage(session, msg, RunnerOptions{})
}

func (client *CodexClient) request(method string, params any) (int, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.nextID++
	msg := map[string]any{"id": client.nextID, "method": method}
	if params != nil {
		msg["params"] = params
	}
	return client.nextID, client.writeLocked(msg)
}

func (client *CodexClient) notify(method string, params any) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	msg := map[string]any{"method": method}
	if params != nil {
		msg["params"] = params
	}
	return client.writeLocked(msg)
}

func (client *CodexClient) respondError(id json.RawMessage, code int, message string) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	msg := map[string]any{
		"id":    json.RawMessage(id),
		"error": map[string]any{"code": code, "message": message},
	}
	return client.writeLocked(msg)
}

func (client *CodexClient) writeLocked(msg map[string]any) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := client.stdin.Write(append(payload, '\n')); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(client.logFile, "[core -> codex] %s\n", payload)
	return nil
}

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
	session.VisibilityMode = firstNonEmpty(session.VisibilityMode, string(execution.VisibilityCoreVisible))
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
			_, _ = client.request("turn/interrupt", map[string]any{
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
		_, err := client.request("turn/interrupt", map[string]any{"threadId": session.CodexThreadID, "turnId": session.CodexTurnID})
		session.LastEvent = "operator_interrupt"
		session.ExecutionStatus = "running"
		session.BlockingReason = ""
		return err
	case "cancel":
		if session.CodexThreadID != "" && session.CodexTurnID != "" {
			_, _ = client.request("turn/interrupt", map[string]any{"threadId": session.CodexThreadID, "turnId": session.CodexTurnID})
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
		id, err := client.request("turn/start", codexTurnStartParams(session.CodexThreadID, input))
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

func CodexThreadID(raw json.RawMessage) string {
	var payload struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
		ThreadID string `json:"threadId"`
	}
	_ = json.Unmarshal(raw, &payload)
	return firstNonEmpty(payload.Thread.ID, payload.ThreadID)
}

func CodexTurnID(raw json.RawMessage) string {
	var payload struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
		TurnID string `json:"turnId"`
	}
	_ = json.Unmarshal(raw, &payload)
	return firstNonEmpty(payload.Turn.ID, payload.TurnID)
}

func CodexMessageFullText(raw json.RawMessage) string {
	var payload struct {
		Delta string `json:"delta"`
		Item  struct {
			Text string `json:"text"`
		} `json:"item"`
	}
	_ = json.Unmarshal(raw, &payload)
	return firstNonEmpty(payload.Delta, payload.Item.Text)
}

func CodexMessageText(raw json.RawMessage) string {
	text := CodexMessageFullText(raw)
	if len(text) > 240 {
		return text[:240]
	}
	return text
}

func IsTerminalBackgroundState(status BackgroundAgentStatus) bool {
	switch strings.ToLower(status.State) {
	case "done", "stopped", "error", "failed":
		return true
	}
	return false
}

func QueryBackgroundAgentStatus(claudeBinary string, backgroundID string) (BackgroundAgentStatus, bool, error) {
	out, err := providerCommandOutput(10*time.Second, claudeBinary, "agents", "--json")
	if err != nil {
		return BackgroundAgentStatus{}, false, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	var sessions []BackgroundAgentStatus
	if err := json.Unmarshal(out, &sessions); err != nil {
		return BackgroundAgentStatus{}, false, err
	}
	for _, entry := range sessions {
		if entry.ID == backgroundID {
			return entry, true, nil
		}
	}
	return BackgroundAgentStatus{}, false, nil
}

func ExtractRemoteControlURL(claudeBinary string, backgroundID string) (string, error) {
	out, err := providerCommandOutput(10*time.Second, claudeBinary, "logs", backgroundID)
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return ParseRemoteControlURL(string(out)), nil
}

func FetchBackgroundSessionLogs(claudeBinary string, backgroundID string) (string, error) {
	if claudeBinary == "" || backgroundID == "" {
		return "", nil
	}
	out, err := providerCommandOutput(10*time.Second, claudeBinary, "logs", backgroundID)
	if err != nil {
		return string(out), fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func StopBackgroundSession(claudeBinary string, backgroundID string) error {
	return exec.Command(claudeBinary, "stop", backgroundID).Run()
}

func ReadSessionLog(root string, logPath string) string {
	return readSessionLog(root, logPath)
}

func finishBackgroundRemoteSession(opts RunnerOptions, session RuntimeSession, task tasklifecycle.Task, claudeBinary string, status string, errorMessage string) {
	session.Status = status
	session.ExecutionStatus = tasklifecycle.LegacyExecutionStatus(status, errorMessage)
	session.ErrorMessage = errorMessage
	session.ExitedAt = time.Now().Format(time.RFC3339)
	session.LastEvent = firstNonEmpty(session.LastEvent, "background_session_ended")
	transcript, fetchErr := FetchBackgroundSessionLogs(claudeBinary, session.BackgroundID)
	if transcript != "" {
		refreshSessionToolUsageFromText(&session, transcript)
		if session.Result == nil {
			if result := tasklifecycle.ParseWorkerResult(transcript); result != nil {
				session.Result = result
				session.LastEvent = "worker_result_reported"
				if summary := strings.TrimSpace(result.Summary); summary != "" {
					session.LastMessage = summary
				}
			}
		}
		if session.ProviderError == nil {
			session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "claude logs", fetchErr, transcript)
		}
	} else if fetchErr != nil && session.ProviderError == nil {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "claude logs", fetchErr, "")
	}
	if session.Result == nil {
		if result := tasklifecycle.ParseWorkerResult(readSessionLog(opts.RuntimeRoot, session.LogPath)); result != nil {
			session.Result = result
			session.LastEvent = "worker_result_reported"
			if summary := strings.TrimSpace(result.Summary); summary != "" {
				session.LastMessage = summary
			}
		}
	}
	if session.ProviderError != nil {
		session.ExecutionStatus = "provider_error"
	} else if session.Result != nil && session.ExecutionStatus != "cancelled" {
		if tasklifecycle.WorkerResultNeedsInput(session.Result) {
			session.ExecutionStatus = "waiting_input"
			session.Status = "waiting_input"
			session.ExitedAt = ""
		} else {
			session.ExecutionStatus = "succeeded"
		}
	}
	_ = runtimedb.AppendLog(opts.RuntimeRoot, session.LogPath, fmt.Sprintf("\n[core] background session ended status=%s %s\n", status, errorMessage))
	if transcript != "" {
		_ = runtimedb.AppendLog(opts.RuntimeRoot, session.LogPath, fmt.Sprintf("\n[core] final transcript (claude logs %s):\n%s\n", session.BackgroundID, transcript))
	}
	completeProviderSession(opts, session, task, firstNonEmpty(errorMessage, "Background agent session ended"), map[string]any{
		"background_id": session.BackgroundID,
		"status":        status,
	})
}

func captureBackgroundWorkerResult(opts RunnerOptions, session *RuntimeSession, task tasklifecycle.Task, claudeBinary string, backgroundID string, resultApplied *bool) {
	if session == nil {
		return
	}
	if resultApplied != nil && *resultApplied {
		return
	}
	if session.Result != nil {
		if resultApplied != nil {
			*resultApplied = true
		}
		return
	}
	transcript, _ := FetchBackgroundSessionLogs(claudeBinary, backgroundID)
	if transcript == "" {
		return
	}
	result := tasklifecycle.ParseWorkerResult(transcript)
	if result == nil {
		return
	}
	if resultApplied != nil {
		*resultApplied = true
	}
	applyWorkerReportedResult(opts, session, task, result)
}

func finishCodexAppServerSession(opts RunnerOptions, session RuntimeSession, task tasklifecycle.Task, status string, errorMessage string) {
	session.Status = status
	if session.ExecutionStatus == "" {
		session.ExecutionStatus = tasklifecycle.LegacyExecutionStatus(status, errorMessage)
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
	id, err = client.request("turn/start", codexTurnStartParams(threadID, task.LaunchEvaluation.Prompt))
	if err != nil {
		return err
	}
	response, err := waitForCodexResponse(ctx, messages, id, client, session, task, idleTimeout, opts)
	if err != nil {
		return err
	}
	if turnID := CodexTurnID(response.Result); turnID != "" {
		session.CodexTurnID = turnID
		session.LastEvent = "turn/start"
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
				_ = client.respondError(msg.ID, -32601, "Core launcher does not service Codex server-initiated requests")
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
				_ = client.respondError(msg.ID, -32601, "Core launcher does not service Codex server-initiated requests")
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
		resumeThreadID := session.CodexThreadID
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
				waitErr = errors.New("codex app-server thread/resume response did not include thread.id")
			}
			err = waitErr
		}
		session.ResumeOutcome = "fallback_new"
		session.CodexThreadID = ""
		upsertSession(opts, *session)
		_ = resumeThreadID
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

func updateCodexSessionFromMessage(session *RuntimeSession, msg CodexRPCMessage, opts RunnerOptions) bool {
	if msg.Method == "" {
		return false
	}
	session.LastEvent = msg.Method
	switch msg.Method {
	case "turn/started", "item/started", "item/completed", "item/agentMessage/delta", "thread/status/changed":
		if session.ExecutionStatus != "cancelled" {
			session.ExecutionStatus = "running"
			session.BlockingReason = ""
		}
	}
	if strings.HasPrefix(msg.Method, "item/") {
		mergeSessionToolCalls(session, execution.ExtractToolCallsFromCodexParams(msg.Method, msg.Params, time.Now()))
	}
	if text := CodexMessageText(msg.Params); text != "" {
		session.LastMessage = truncateDisplayText(text)
	}
	if fullText := CodexMessageFullText(msg.Params); fullText != "" {
		if result := tasklifecycle.ParseWorkerResult(fullText); result != nil {
			session.Result = result
			session.LastEvent = "worker_result_reported"
			if summary := strings.TrimSpace(result.Summary); summary != "" {
				session.LastMessage = truncateDisplayText(summary)
			}
		}
	}
	if turnID := CodexTurnID(msg.Params); turnID != "" {
		session.CodexTurnID = turnID
	}
	if threadID := CodexThreadID(msg.Params); threadID != "" {
		session.CodexThreadID = threadID
	}
	upsertSession(opts, *session)
	return true
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

func readCodexStdout(stdout io.Reader, logFile io.Writer, messages chan<- CodexRPCMessage, done chan<- error) {
	defer close(messages)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(logFile, "[codex -> core] %s\n", line)
		var msg CodexRPCMessage
		if err := json.Unmarshal(line, &msg); err == nil {
			messages <- msg
		} else {
			_, _ = fmt.Fprintf(logFile, "[core] could not parse codex JSON-RPC line: %s\n", err)
		}
	}
	done <- scanner.Err()
}

func copyCodexStderr(stderr io.Reader, logFile io.Writer) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		_, _ = fmt.Fprintf(logFile, "[codex stderr] %s\n", scanner.Text())
	}
}

func codexInitializeParams() map[string]any {
	return map[string]any{
		"clientInfo":   map[string]any{"name": "core_launcher", "title": "Core Agent Launcher", "version": "0.1.0"},
		"capabilities": map[string]any{"experimentalApi": true},
	}
}

func codexThreadStartParams(session RuntimeSession) map[string]any {
	return map[string]any{"cwd": session.WorkingDir, "approvalPolicy": "never", "sandbox": "workspace-write"}
}

func codexThreadResumeParams(session RuntimeSession) map[string]any {
	return map[string]any{"threadId": session.CodexThreadID, "cwd": session.WorkingDir}
}

func codexThreadSetNameParams(threadID string, name string) map[string]any {
	return map[string]any{"threadId": threadID, "name": name}
}

func codexTurnStartParams(threadID string, prompt string) map[string]any {
	return map[string]any{"threadId": threadID, "input": []map[string]string{{"type": "text", "text": prompt}}}
}

func codexTurnStatus(raw json.RawMessage) string {
	var payload struct {
		Turn struct {
			Status string `json:"status"`
		} `json:"turn"`
	}
	_ = json.Unmarshal(raw, &payload)
	return strings.TrimSpace(payload.Turn.Status)
}

func codexMessageID(msg CodexRPCMessage) int {
	if len(msg.ID) == 0 {
		return 0
	}
	var id int
	_ = json.Unmarshal(msg.ID, &id)
	return id
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
