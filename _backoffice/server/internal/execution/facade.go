package execution

import (
	"context"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/execution/providers"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

type BackgroundAgentStatus = providers.BackgroundAgentStatus

func DefaultPlanner() Planner {
	return providers.NewRegistry()
}

func RunBackgroundRemoteSession(ctx context.Context, session RuntimeSession, task tasklifecycle.Task, opts RunnerOptions) {
	providers.RunBackgroundRemoteSession(ctx, session, task, providerRunnerOptions(opts))
}

func RunCursorVisibleSession(ctx context.Context, session RuntimeSession, task tasklifecycle.Task, opts RunnerOptions) {
	providers.RunCursorVisibleSession(ctx, session, task, providerRunnerOptions(opts))
}

func RunCodexAppServerSession(ctx context.Context, session RuntimeSession, task tasklifecycle.Task, opts RunnerOptions) {
	providers.RunCodexAppServerSession(ctx, session, task, providerRunnerOptions(opts))
}

func RunGeminiHeadlessSession(ctx context.Context, session RuntimeSession, task tasklifecycle.Task, opts RunnerOptions) {
	providers.RunGeminiHeadlessSession(ctx, session, task, providerRunnerOptions(opts))
}

func QueryBackgroundAgentStatus(claudeBinary string, backgroundID string) (BackgroundAgentStatus, bool, error) {
	return providers.QueryBackgroundAgentStatus(claudeBinary, backgroundID)
}

func FetchBackgroundSessionLogs(claudeBinary string, backgroundID string) (string, error) {
	return providers.FetchBackgroundSessionLogs(claudeBinary, backgroundID)
}

func IsTerminalBackgroundState(status BackgroundAgentStatus) bool {
	return providers.IsTerminalBackgroundState(status)
}

func SetBackgroundRemotePollIntervalForTest(interval time.Duration) func() {
	previous := providers.BackgroundRemotePollInterval()
	providers.SetBackgroundRemotePollInterval(interval)
	return func() {
		providers.SetBackgroundRemotePollInterval(previous)
	}
}

func SetBackgroundRemoteControlReadyTimeoutForTest(timeout time.Duration) func() {
	previous := providers.BackgroundRemoteControlReadyTimeout()
	providers.SetBackgroundRemoteControlReadyTimeout(timeout)
	return func() {
		providers.SetBackgroundRemoteControlReadyTimeout(previous)
	}
}

func providerRunnerOptions(opts RunnerOptions) providers.RunnerOptions {
	return providers.RunnerOptions{
		CoreRoot:       opts.CoreRoot,
		SessionTimeout: opts.SessionTimeout,
		Callbacks: providers.RunnerCallbacks{
			ClassifyProviderError:        opts.Callbacks.ClassifyProviderError,
			FailLaunchSession:            opts.Callbacks.FailLaunchSession,
			RecordProviderSessionStart:   opts.Callbacks.RecordProviderSessionStart,
			CompleteProviderSession:      opts.Callbacks.CompleteProviderSession,
			UpsertSession:                opts.Callbacks.UpsertSession,
			AddTaskComment:               opts.Callbacks.AddTaskComment,
			MarkSessionOperatorAttention: opts.Callbacks.MarkSessionOperatorAttention,
			ApplyWorkerReportedResult:    opts.Callbacks.ApplyWorkerReportedResult,
			RegisterSessionControl:       opts.Callbacks.RegisterSessionControl,
			UnregisterSessionControl:     opts.Callbacks.UnregisterSessionControl,
		},
	}
}
