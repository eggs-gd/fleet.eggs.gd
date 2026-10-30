package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	execution "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

const claudeLogsOp = "claude logs"

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
// URL is a hollow session: nothing appears on desktop/mobile and
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
	resuming := strings.TrimSpace(session.SessionID) != ""
	if resuming {
		session.Command = backgroundResumeCommand(session)
	}
	appendSessionLogLine(logFile, "[core] dispatching background-remote agent=%s cwd=%s resuming=%v command=%q\n\n", session.Agent, session.WorkingDir, resuming, session.Command)

	backgroundID, err := dispatchBackgroundSession(ctx, opts, &session, logFile)
	if err != nil {
		failLaunch(opts, session, task, err)
		return
	}
	noteBackgroundDispatched(opts, &session, task, logFile, backgroundID, resuming)

	run := newBackgroundRun(ctx, opts, session, task, logFile, backgroundID)
	run.loop()
}

// backgroundResumeCommand is the 1-1-1 session reuse command
// (_docs/AGENT_SESSION_REUSE.md): resume must be the full session UUID with no
// other flags repeated (--remote-control / --name / --permission-mode are
// restored from the session's own saved options) — passing them again, or the
// short BackgroundID instead of SessionID, forks a copy instead of continuing
// the same session. The prompt is always the original command's last argv.
func backgroundResumeCommand(session RuntimeSession) []string {
	command := session.Command
	return []string{command[0], "--bg", "--resume", session.SessionID, command[len(command)-1]}
}

// dispatchBackgroundSession runs `claude --bg` and returns the background id
// it printed. On failure it records the classified provider error on session.
func dispatchBackgroundSession(ctx context.Context, opts RunnerOptions, session *RuntimeSession, logFile io.Writer) (string, error) {
	dispatchCtx, cancelDispatch := context.WithTimeout(ctx, 30*time.Second)
	defer cancelDispatch()
	var dispatchOutput bytes.Buffer
	command := session.Command
	cmd := exec.CommandContext(dispatchCtx, command[0], command[1:]...)
	cmd.Dir = session.WorkingDir
	cmd.Stdout = io.MultiWriter(&dispatchOutput, logFile)
	cmd.Stderr = io.MultiWriter(&dispatchOutput, logFile)
	if err := cmd.Run(); err != nil {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "background dispatch", err, dispatchOutput.String())
		return "", fmt.Errorf("background dispatch failed: %w", err)
	}
	match := backgroundSessionIDPattern.FindStringSubmatch(dispatchOutput.String())
	if len(match) < 2 {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "background dispatch", nil, dispatchOutput.String())
		return "", errors.New("could not parse background session id from `claude --bg` dispatch output")
	}
	return match[1], nil
}

// noteBackgroundDispatched records the new background id and announces that the
// session is waiting for its remote-control URL.
func noteBackgroundDispatched(opts RunnerOptions, session *RuntimeSession, task tasklifecycle.Task, logFile io.Writer, backgroundID string, resuming bool) {
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
	recordProviderStart(opts, *session, task, "Background remote-control agent dispatched; waiting for visible session", map[string]any{
		"claim_id":        session.ClaimID,
		"background_id":   backgroundID,
		"visibility_mode": session.VisibilityMode,
		"log_path":        session.LogPath,
		"handshake":       "waiting_for_visible_session",
	})
	appendSessionLogLine(logFile, "[core] background session id=%s\n", backgroundID)
	appendSessionLogLine(logFile, "[core] provider visible-session wait started (remote-control URL registration)\n")
}

// backgroundRun is the poll loop of one dispatched background session.
type backgroundRun struct {
	ctx          context.Context
	opts         RunnerOptions
	session      RuntimeSession
	task         tasklifecycle.Task
	logFile      io.Writer
	claudeBinary string
	backgroundID string

	idleTimeout  time.Duration
	readyTimeout time.Duration

	urlAnnounced      bool
	lastActivity      time.Time
	lastLogSize       int
	lastState         string
	attentionNotified bool
	resultApplied     bool
}

func newBackgroundRun(ctx context.Context, opts RunnerOptions, session RuntimeSession, task tasklifecycle.Task, logFile io.Writer, backgroundID string) *backgroundRun {
	idleTimeout := opts.SessionTimeout
	readyTimeout := BackgroundRemoteControlReadyTimeout()
	if readyTimeout <= 0 {
		readyTimeout = 2 * time.Minute
	}
	if idleTimeout > 0 && readyTimeout > idleTimeout {
		readyTimeout = idleTimeout
	}
	return &backgroundRun{
		ctx:          ctx,
		opts:         opts,
		session:      session,
		task:         task,
		logFile:      logFile,
		claudeBinary: session.Command[0],
		backgroundID: backgroundID,
		idleTimeout:  idleTimeout,
		readyTimeout: readyTimeout,
		lastActivity: time.Now(),
		lastLogSize:  -1,
	}
}

func (r *backgroundRun) loop() {
	for {
		select {
		case <-r.ctx.Done():
			_ = StopBackgroundSession(r.claudeBinary, r.backgroundID)
			r.finish("cancelled", "runtime shutting down")
			return
		default:
		}
		if r.poll() {
			return
		}
		select {
		case <-r.ctx.Done():
		case <-time.After(BackgroundRemotePollInterval()):
		}
	}
}

// poll runs one round of checks; true means the session was finished.
func (r *backgroundRun) poll() bool {
	if r.announceURL() || r.followStatus() || r.followTranscript() || r.checkHollow() {
		return true
	}
	r.checkIdle()
	return false
}

func (r *backgroundRun) finish(status string, errorMessage string) {
	finishBackgroundRemoteSession(r.opts, r.session, r.task, r.claudeBinary, status, errorMessage)
}

// progress marks fresh activity and keeps the session row in step with whether
// the remote-control URL is out yet.
func (r *backgroundRun) progress() {
	r.lastActivity = time.Now()
	r.attentionNotified = false
	if r.urlAnnounced {
		r.session.ExecutionStatus = "running"
		r.session.Status = "running"
	} else {
		r.session.ExecutionStatus = "waiting_for_visible_session"
		r.session.Status = "starting"
	}
	r.session.BlockingReason = ""
}

// socketFailure finishes the session when err is the provider's control socket
// being unavailable; any other error is left to the next poll.
func (r *backgroundRun) socketFailure(operation string, err error) bool {
	pe := classifyProviderError(r.opts, r.session.Agent, r.session.Backend, operation, err, "")
	if pe == nil || pe.Kind != "control_socket_unavailable" {
		return false
	}
	r.session.ProviderError = pe
	r.finish("failed", pe.Reason)
	return true
}

func (r *backgroundRun) announceURL() bool {
	if r.urlAnnounced {
		return false
	}
	url, logErr := ExtractRemoteControlURL(r.claudeBinary, r.backgroundID)
	if logErr != nil && r.socketFailure(claudeLogsOp, logErr) {
		return true
	}
	if url == "" {
		return false
	}
	r.urlAnnounced = true
	r.progress()
	r.session.RemoteControlURL = url
	r.session.LastEvent = "remote_control_ready"
	r.session.VisibilityMode = string(execution.VisibilityAppVisible)
	upsertSession(r.opts, r.session)
	appendSessionLogLine(r.logFile, "[core] remote-control url: %s\n", url)
	addTaskComment(r.opts, r.task, tasklifecycle.RemoteControlSessionLiveComment(url))
	return false
}

func (r *backgroundRun) followStatus() bool {
	status, found, statusErr := QueryBackgroundAgentStatus(r.claudeBinary, r.backgroundID)
	if statusErr != nil && r.socketFailure("claude agents --json", statusErr) {
		return true
	}
	if !found {
		captureBackgroundWorkerResult(r.opts, &r.session, r.task, r.claudeBinary, r.backgroundID, &r.resultApplied)
		r.finish("exited", "session no longer listed by `claude agents`")
		return true
	}
	if r.session.SessionID == "" && status.SessionID != "" {
		// Full session UUID, needed by --resume for project+agent session
		// reuse (_docs/AGENT_SESSION_REUSE.md) — BackgroundID alone forks
		// a copy instead of continuing the same session.
		r.session.SessionID = status.SessionID
		upsertSession(r.opts, r.session)
	}
	if IsTerminalBackgroundState(status) {
		captureBackgroundWorkerResult(r.opts, &r.session, r.task, r.claudeBinary, r.backgroundID, &r.resultApplied)
		if strings.EqualFold(status.State, "error") || strings.EqualFold(status.State, "failed") {
			r.finish("failed", "background agent ended with state "+status.State)
		} else {
			r.finish("exited", "")
		}
		return true
	}
	if state := strings.TrimSpace(status.State); state != "" && state != r.lastState {
		r.lastState = state
		r.progress()
		r.session.LastEvent = "state:" + state
		upsertSession(r.opts, r.session)
	}
	return false
}

func (r *backgroundRun) followTranscript() bool {
	transcript, _ := FetchBackgroundSessionLogs(r.claudeBinary, r.backgroundID)
	if transcript == "" {
		return false
	}
	if size := len(transcript); size != r.lastLogSize {
		r.lastLogSize = size
		r.progress()
		r.session.LastEvent = "transcript_updated"
		r.session.LastMessage = fmt.Sprintf("transcript bytes=%d", size)
		refreshSessionToolUsageFromText(&r.session, transcript)
		upsertSession(r.opts, r.session)
	}
	if r.resultApplied {
		return false
	}
	result := tasklifecycle.ParseWorkerResult(transcript)
	if result == nil {
		return false
	}
	r.resultApplied = true
	applyWorkerReportedResult(r.opts, &r.session, r.task, result)
	// 1-1-1 session reuse (_docs/AGENT_SESSION_REUSE.md): once this task's
	// outcome is in, stop the background process so the next task for the same
	// (project, agent) can resume this exact session instead of forking a copy —
	// `claude --bg --resume` only continues under the same id when the prior
	// process is already stopped. Skip this for needs_input/waiting_input: the
	// operator may want to answer directly in the still-live phone/app session.
	if tasklifecycle.WorkerResultNeedsInput(result) {
		return false
	}
	_ = StopBackgroundSession(r.claudeBinary, r.backgroundID)
	r.finish("exited", "stopped after task outcome; resumable for the next task in this project")
	return true
}

// checkHollow handles a hollow session: the background id exists but Claude
// never published an app-visible URL. Prefer a concrete auth/quota/billing
// classification from provider output; only fall back to
// remote_control_unavailable when nothing more specific is recognizable.
func (r *backgroundRun) checkHollow() bool {
	if r.urlAnnounced || time.Since(r.lastActivity) <= r.readyTimeout {
		return false
	}
	transcript, _ := FetchBackgroundSessionLogs(r.claudeBinary, r.backgroundID)
	if excerpt := truncateForSessionLog(transcript, 1200); excerpt != "" {
		appendSessionLogLine(r.logFile, "[core] claude logs excerpt while waiting for remote-control URL:\n%s\n", excerpt)
	}
	session := &r.session
	if pe := classifyProviderError(r.opts, session.Agent, session.Backend, claudeLogsOp, nil, transcript); pe != nil {
		session.ProviderError = pe
		session.LastEvent = pe.Kind
		session.BlockingReason = pe.Reason
		appendSessionLogLine(r.logFile, "[core] classified hollow-session provider failure kind=%s: %s\n", pe.Kind, pe.Reason)
		_ = StopBackgroundSession(r.claudeBinary, r.backgroundID)
		r.finish("failed", pe.Reason)
		return true
	}
	reason := fmt.Sprintf("Claude background session %s never published a remote-control URL within %s (app-visible registration failed). Nothing will appear on Claude desktop/mobile until the provider publishes Continue-here/session URL. Check Claude Desktop login, `claude agents --json`, and `claude logs %s`.", r.backgroundID, r.readyTimeout, r.backgroundID)
	appendSessionLogLine(r.logFile, "[core] %s\n", reason)
	session.ProviderError = tasklifecycle.NewProviderError(session.Agent, "remote_control_unavailable", claudeLogsOp, reason, truncateForSessionLog(transcript, 360))
	session.LastEvent = "remote_control_unavailable"
	session.BlockingReason = reason
	_ = StopBackgroundSession(r.claudeBinary, r.backgroundID)
	r.finish("failed", reason)
	return true
}

// checkIdle asks the operator for attention once when nothing has moved for the
// idle threshold; it never stops the session.
func (r *backgroundRun) checkIdle() {
	idle := time.Since(r.lastActivity)
	if idle <= r.idleTimeout || r.attentionNotified {
		return
	}
	reason := fmt.Sprintf("No Claude activity observed for %s (idle attention threshold %s). Session was not stopped; open the remote session or cancel explicitly.", idle.Round(time.Second), r.idleTimeout)
	appendSessionLogLine(r.logFile, "[core] %s\n", reason)
	markOperatorAttention(r.opts, r.session, r.task, reason)
	r.session.ExecutionStatus = "operator_attention"
	r.session.LastEvent = "idle_operator_attention"
	r.session.BlockingReason = reason
	upsertSession(r.opts, r.session)
	r.attentionNotified = true
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

func finishBackgroundRemoteSession(opts RunnerOptions, session RuntimeSession, task tasklifecycle.Task, claudeBinary string, status string, errorMessage string) {
	session.Status = status
	session.ExecutionStatus = tasklifecycle.ExecutionStatusFor(status, errorMessage)
	session.ErrorMessage = errorMessage
	session.ExitedAt = time.Now().Format(time.RFC3339)
	session.LastEvent = firstNonEmpty(session.LastEvent, "background_session_ended")
	transcript, fetchErr := FetchBackgroundSessionLogs(claudeBinary, session.BackgroundID)
	readFinalTranscript(opts, &session, transcript, fetchErr)
	if session.Result == nil {
		adoptWorkerResult(&session, tasklifecycle.ParseWorkerResult(readSessionLog(opts.RuntimeRoot, session.LogPath)))
	}
	settleFinishedExecutionStatus(&session)
	_ = runtimedb.AppendLog(opts.RuntimeRoot, session.LogPath, fmt.Sprintf("\n[core] background session ended status=%s %s\n", status, errorMessage))
	if transcript != "" {
		_ = runtimedb.AppendLog(opts.RuntimeRoot, session.LogPath, fmt.Sprintf("\n[core] final transcript (claude logs %s):\n%s\n", session.BackgroundID, transcript))
	}
	completeProviderSession(opts, session, task, firstNonEmpty(errorMessage, "Background agent session ended"), map[string]any{
		"background_id": session.BackgroundID,
		"status":        status,
	})
}

// adoptWorkerResult keeps the result the worker reported, if any, on the session.
func adoptWorkerResult(session *RuntimeSession, result *tasklifecycle.WorkerResult) {
	if result == nil {
		return
	}
	session.Result = result
	session.LastEvent = "worker_result_reported"
	if summary := strings.TrimSpace(result.Summary); summary != "" {
		session.LastMessage = summary
	}
}

// readFinalTranscript takes the tool usage, the worker result and a provider
// error from the final `claude logs` output (or from the fetch failure).
func readFinalTranscript(opts RunnerOptions, session *RuntimeSession, transcript string, fetchErr error) {
	if transcript == "" {
		if fetchErr != nil && session.ProviderError == nil {
			session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, claudeLogsOp, fetchErr, "")
		}
		return
	}
	refreshSessionToolUsageFromText(session, transcript)
	if session.Result == nil {
		adoptWorkerResult(session, tasklifecycle.ParseWorkerResult(transcript))
	}
	if session.ProviderError == nil {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, claudeLogsOp, fetchErr, transcript)
	}
}

// settleFinishedExecutionStatus turns the end state into the execution status:
// a provider error wins, then a reported result (which may be a pause for input).
func settleFinishedExecutionStatus(session *RuntimeSession) {
	switch {
	case session.ProviderError != nil:
		session.ExecutionStatus = "provider_error"
	case session.Result == nil || session.ExecutionStatus == "cancelled":
	case tasklifecycle.WorkerResultNeedsInput(session.Result):
		session.ExecutionStatus = "waiting_input"
		session.Status = "waiting_input"
		session.ExitedAt = ""
	default:
		session.ExecutionStatus = "succeeded"
	}
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
