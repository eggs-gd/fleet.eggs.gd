package server

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/eventbus"
	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/executionfinalizer"
	"github.com/eggs-gd/fleet.eggs.gd/internal/health"
	"github.com/eggs-gd/fleet.eggs.gd/internal/providerconfig"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
	tpopen "github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/open"
	"github.com/eggs-gd/fleet.eggs.gd/lib/chain"
	l "github.com/eggs-gd/fleet.eggs.gd/lib/logger"
	"github.com/eggs-gd/fleet.eggs.gd/lib/logger/decorators"
)

// App is the composition-root bundle used by Serve and focused integration tests.
type App struct {
	*Dashboard

	Store         *taskprovider.Board
	Exec          *execution.Service
	TaskStore     taskprovider.Provider
	Listener      chain.Processor
	FinalizerProc chain.ChainProcessor

	errch       chan error
	logger      *l.Logger
	root        string
	runtimeRoot string
}

// ComposeConfig builds the independent sibling services behind one Dashboard.
type ComposeConfig struct {
	CoreRoot         string
	RuntimeRoot      string
	DryRun           bool
	SessionTimeout   time.Duration
	Interval         time.Duration
	TaskProvider     providerconfig.Settings
	LauncherIdentity string
	Logger           *l.Logger
	Health           *health.Monitor
}

// Compose constructs listener, execution, finalizer, and dashboard siblings.
func Compose(cfg ComposeConfig) *App {
	logger := cfg.Logger
	if logger == nil {
		logger = l.NewLogger(l.InfoLevel, &decorators.GontrollerDecorator{})
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	// RuntimeRoot (App's own ~/.fleet home for session/audit state) falls
	// back to CoreRoot when not given explicitly, so callers that don't
	// care about the App/Data split (tests, ad hoc tooling) get one
	// coherent root instead of runtime state silently landing at a
	// relative path under the process's working directory.
	if cfg.RuntimeRoot == "" {
		cfg.RuntimeRoot = cfg.CoreRoot
	}

	errch := make(chan error, 64)
	taskEvents := make(chan taskflow.TaskEvent)
	tasks := make(chan *execution.Task)
	validated := make(chan *execution.Task)
	launched := make(chan *execution.Task)
	executionResults := make(chan taskflow.ExecutionResult, 64)

	store := taskprovider.NewBoard(cfg.CoreRoot)
	taskStore, wiring := tpopen.Open(tpopen.OpenOptions{
		Root:        cfg.CoreRoot,
		RuntimeRoot: cfg.RuntimeRoot,
		Settings:    cfg.TaskProvider,
		Interval:    interval,
		Board:       store,
	})
	taskService := taskprovider.NewService(taskStore)

	if wiring.PollInterval > 0 {
		interval = wiring.PollInterval
	}

	finalizer := executionfinalizer.New(taskService, func(locator string) (tasklifecycle.Task, error) {
		if loaded, err := taskStore.Load(locator); err == nil {
			return loaded, nil
		}
		flowTask, err := taskService.Get(context.Background(), locator)
		if err != nil {
			return tasklifecycle.Task{}, err
		}
		return execution.TaskFromFlow(flowTask), nil
	}, store.UpsertTask)
	events := &auditedBus{bus: eventbus.New(), root: cfg.RuntimeRoot, health: cfg.Health}
	finalizer.WithPublisher(events)

	exec := execution.NewService(execution.ServiceOptions{
		Config: execution.Config{
			Root:             cfg.CoreRoot,
			RuntimeRoot:      cfg.RuntimeRoot,
			DryRun:           cfg.DryRun,
			SessionTimeout:   cfg.SessionTimeout,
			LauncherIdentity: cfg.LauncherIdentity,
			Logger:           logger.Named("Execution"),
		},
		TaskService: taskService,
		ReloadTask: func(locator string) (execution.Task, error) {
			return taskStore.Load(locator)
		},
		Board:            boardHooks(cfg.RuntimeRoot, store),
		Lookup:           store.TaskByPath,
		Planner:          execution.DefaultPlanner(),
		In:               taskEvents,
		Tasks:            tasks,
		Validated:        validated,
		Launched:         launched,
		Errch:            errch,
		ExecutionResults: executionResults,
		ApplyResult: func(result taskflow.ExecutionResult) error {
			_, err := finalizer.Apply(result)
			return err
		},
	})
	exec.NotifyOperatorAttention(func(taskRef, question string) {
		text := "Task " + taskRef + " needs a decision before continuing."
		if detail := strings.TrimSpace(question); detail != "" {
			text += "\n" + detail
		}
		_ = events.Publish(eventbus.Event{
			Channel: eventbus.ChannelTask,
			Type:    "task.needs_attention",
			Text:    text,
			Fields:  map[string]string{"task_id": taskRef},
		})
	})

	listener := taskprovider.NewListenerService(taskStore, interval, taskprovider.ListenerHooks{
		Before: func(locator string) (taskprovider.Task, bool) {
			return store.TaskByPath(locator)
		},
		ActiveSessionForTask: func(task taskprovider.Task) (taskprovider.ActiveSession, bool) {
			session, ok := exec.ActiveSessionForTask(task)
			if !ok {
				return taskprovider.ActiveSession{}, false
			}
			return taskprovider.ActiveSession{
				ClaimID:         session.ClaimID,
				ExecutionStatus: session.ExecutionStatus,
				Status:          session.Status,
			}, true
		},
		RevertActiveExecution: execution.ActiveExecutionGuard(taskService),
		OnTaskChange: func(observed taskprovider.ObservedTaskChange) error {
			store.UpsertTask(observed.Task)
			return nil
		},
	}, taskEvents)
	finalizerProc := executionfinalizer.NewService(finalizer, executionResults, errch)

	dash := NewDashboard(DashboardOptions{
		Root:           cfg.CoreRoot,
		RuntimeRoot:    cfg.RuntimeRoot,
		DryRun:         cfg.DryRun,
		SessionTimeout: cfg.SessionTimeout,
		Store:          store,
		TaskStore:      taskStore,
		TaskService:    taskService,
		Execution:      exec,
		Logger:         logger.Named("Dashboard"),
	})

	return &App{
		Dashboard:     dash,
		Store:         store,
		Exec:          exec,
		TaskStore:     taskStore,
		Listener:      listener,
		FinalizerProc: finalizerProc,
		errch:         errch,
		logger:        logger.Named("App"),
		root:          cfg.CoreRoot,
		runtimeRoot:   cfg.RuntimeRoot,
	}
}

func compose(cfg Config, taskProviderSettings providerconfig.Settings) *App {
	return Compose(ComposeConfig{
		CoreRoot:       cfg.CoreRoot,
		RuntimeRoot:    cfg.RuntimeRoot,
		DryRun:         cfg.DryRun,
		SessionTimeout: cfg.SessionTimeout,
		TaskProvider:   taskProviderSettings,
		Health:         cfg.Health,
	})
}

func (a *App) Bootstrap() error {
	return a.Dashboard.Bootstrap()
}

func (a *App) Surface() DashboardSurface {
	return a.Dashboard
}

func (a *App) TaskService() taskprovider.TaskService {
	return a.Dashboard.TaskService()
}

func (a *App) State() State {
	return a.Dashboard.State()
}

func (a *App) Run(ctx context.Context) {
	go a.logErrors(ctx)
	if a.Exec != nil && a.Dashboard != nil && a.allowsProcessStart() {
		// Wake existing todo/needs_rework tasks after session reconciliation.
		// Must run with the live execution host, not only on Bootstrap, so
		// focused Bootstrap-only tests stay free of accidental process starts.
		a.Exec.LaunchPendingTasks(ctx)
	}
	var wg sync.WaitGroup
	if a.Exec != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.Exec.Process(ctx)
		}()
	}
	for _, proc := range []chain.Processor{a.Listener, a.FinalizerProc} {
		if proc == nil {
			continue
		}
		wg.Add(1)
		go func(p chain.Processor) {
			defer wg.Done()
			p.Process(ctx)
		}(proc)
	}
	wg.Wait()
}

func (a *App) logErrors(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-a.errch:
			if err != nil && a.logger != nil {
				a.logger.Error("chain error", l.Error(err))
				_ = audit.AppendEvent(a.runtimeRoot, audit.Event{Type: "chain_error", Message: err.Error()})
			}
		}
	}
}
