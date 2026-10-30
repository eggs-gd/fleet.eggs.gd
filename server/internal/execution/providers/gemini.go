package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

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
	projectID := strings.TrimSpace(session.GeminiProjectID)
	if projectID == "" {
		projectID = findGeminiProjectID(session.WorkingDir)
	}
	conversationID := strings.TrimSpace(session.GeminiConversationID)

	command := geminiCommand(&session, projectID, conversationID)
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
	recordGeminiProject(logFile, &session, binary, projectID)
	settleGeminiStatus(opts, &session, err, runCtx.Err(), event)
	if session.Status == "failed" && session.ErrorMessage != "" {
		appendSessionLogLine(logFile, "\n[core] gemini session failed: %s\n", session.ErrorMessage)
	}
	adoptGeminiResult(opts, &session, event)

	completeProviderSession(opts, session, task, firstNonEmpty(session.ErrorMessage, "Gemini headless agent process exited"), map[string]any{
		"process_id":      session.ProcessID,
		"project_id":      session.GeminiProjectID,
		"conversation_id": session.GeminiConversationID,
		"exit_code":       session.ExitCode,
	})
}

// geminiCommand is the `agy` argv for this turn.
func geminiCommand(session *RuntimeSession, projectID string, conversationID string) []string {
	baseArgs := session.Command[1:]
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
	return append([]string{session.Command[0]}, args...)
}

// recordGeminiProject keeps the antigravity project id on the session (looking
// it up after a --new-project launch) and the command to resume it by hand.
func recordGeminiProject(logFile io.Writer, session *RuntimeSession, binary string, projectID string) {
	if projectID != "" {
		session.GeminiProjectID = projectID
	} else if discovered := findGeminiProjectID(session.WorkingDir); discovered != "" {
		session.GeminiProjectID = discovered
		appendSessionLogLine(logFile, "\n[core] registered antigravity project id=%s for cwd=%s\n", discovered, session.WorkingDir)
	} else {
		appendSessionLogLine(logFile, "\n[core] could not discover an antigravity project id for cwd=%s after --new-project\n", session.WorkingDir)
	}
	if session.GeminiProjectID != "" && session.GeminiConversationID != "" {
		session.OperatorCommand = GeminiOperatorCommand(binary, session.GeminiProjectID, session.GeminiConversationID)
		appendSessionLogLine(logFile, "\n[core] operator resume command: %s\n", session.OperatorCommand)
	}
}

// settleGeminiStatus turns how the process ended into the session status.
func settleGeminiStatus(opts RunnerOptions, session *RuntimeSession, waitErr error, runErr error, event geminiResultEvent) {
	session.Status = "exited"
	switch {
	case waitErr != nil:
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
		session.ProviderError = classifyProviderError(opts, session.Agent, session.Backend, "agy output", waitErr, readSessionLog(opts.RuntimeRoot, session.LogPath))
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
}

// adoptGeminiResult takes the worker result (or the plain reply) from the
// final response and pauses the session when the worker asked for input.
func adoptGeminiResult(opts RunnerOptions, session *RuntimeSession, event geminiResultEvent) {
	switch {
	case session.Result != nil:
		refreshSessionToolUsageFromText(session, readSessionLog(opts.RuntimeRoot, session.LogPath))
	case event.Response != "":
		refreshSessionToolUsageFromText(session, event.Response)
		if result := tasklifecycle.ParseWorkerResult(event.Response); result != nil {
			adoptWorkerResult(session, result)
		} else if session.LastMessage == "" {
			session.LastMessage = event.Response
		}
	}
	if session.Result != nil && session.ExecutionStatus != "cancelled" && session.ProviderError == nil &&
		tasklifecycle.WorkerResultNeedsInput(session.Result) {
		session.ExecutionStatus = "waiting_input"
		session.Status = "waiting_input"
		session.ExitedAt = ""
	}
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
