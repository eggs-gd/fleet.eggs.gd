package execution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

type RunnerCallbacks struct {
	ClassifyProviderError      func(provider string, backend string, operation string, err error, output string) *tasklifecycle.ProviderError
	FailLaunchSession          func(session RuntimeSession, task tasklifecycle.Task, err error)
	RecordProviderSessionStart func(session RuntimeSession, task tasklifecycle.Task, message string, details map[string]any)
	CompleteProviderSession    func(session RuntimeSession, task tasklifecycle.Task, exitMessage string, details map[string]any)
	UpsertSession              func(session RuntimeSession)
	AddTaskComment             func(locator string, comment string) error
	MarkSessionOperatorAttention func(session RuntimeSession, task tasklifecycle.Task, reason string)
	ApplyWorkerReportedResult  func(session *RuntimeSession, task tasklifecycle.Task, result *tasklifecycle.WorkerResult)
	RegisterSessionControl     func(claimID string, ch chan SessionControlRequest)
	UnregisterSessionControl   func(claimID string)
}

type RunnerOptions struct {
	CoreRoot       string
	SessionTimeout time.Duration
	Callbacks      RunnerCallbacks
}

type RunningProcess struct {
	Cmd     *exec.Cmd
	LogFile *os.File
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
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
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

func ReadSessionLog(root string, logPath string) string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(logPath)))
	if err != nil {
		return ""
	}
	return string(data)
}
