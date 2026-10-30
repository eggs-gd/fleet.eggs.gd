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
				pe := classifyProviderError(opts, session.Agent, session.Backend, claudeLogsOp, logErr, "")
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

		// Hollow session: background id exists but
		// Claude never published an app-visible URL. Prefer a concrete
		// auth/quota/billing classification from provider output; only fall
		// back to remote_control_unavailable when nothing more specific is
		// recognizable.
		if !urlAnnounced && time.Since(lastActivity) > readyTimeout {
			transcript, _ := FetchBackgroundSessionLogs(claudeBinary, backgroundID)
			if excerpt := truncateForSessionLog(transcript, 1200); excerpt != "" {
				appendSessionLogLine(logFile, "[core] claude logs excerpt while waiting for remote-control URL:\n%s\n", excerpt)
			}
			if pe := classifyProviderError(opts, session.Agent, session.Backend, claudeLogsOp, nil, transcript); pe != nil {
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
			session.ProviderError = tasklifecycle.NewProviderError(session.Agent, "remote_control_unavailable", claudeLogsOp, reason, truncateForSessionLog(transcript, 360))
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
			session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, claudeLogsOp, fetchErr, transcript)
		}
	} else if fetchErr != nil && session.ProviderError == nil {
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, claudeLogsOp, fetchErr, "")
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
