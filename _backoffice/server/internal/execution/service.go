package execution

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider"
	"github.com/eggs-gd/core.eggs.gd/lib/chain"
	l "github.com/eggs-gd/core.eggs.gd/lib/logger"
	"github.com/eggs-gd/core.eggs.gd/lib/logger/decorators"
)

// Service owns the execution contour (Entry→Validate→Plan) plus post-launch
// host duties: start processes, schedule slots, session control, and publish
// ExecutionResult for Contour 3.
type Service struct {
	cfg     Config
	Contour chain.Processor
	tasks   taskprovider.TaskService
	reload  func(locator string) (tasklifecycle.Task, error)
	board   BoardHooks
	hub     *sessionHub
	lookup  Lookup
	planner Planner

	launched         <-chan *Task
	errch            <-chan error
	executionResults chan taskflow.ExecutionResult
	// applyResult is an optional synchronous Contour 3 hand-off used while
	// composition still double-writes channel + Apply (idempotent).
	applyResult func(taskflow.ExecutionResult) error

	controlsMu sync.Mutex
	controls   map[string]chan SessionControlRequest
	logger     *l.Logger
}

type ServiceOptions struct {
	Config           Config
	TaskService      taskprovider.TaskService
	ReloadTask       func(locator string) (tasklifecycle.Task, error)
	Board            BoardHooks
	Lookup           Lookup
	Planner          Planner
	In               <-chan taskflow.TaskEvent
	Tasks            chan *Task
	Validated        chan *Task
	Launched         chan *Task
	Errch            chan error
	ExecutionResults chan taskflow.ExecutionResult
	ApplyResult      func(taskflow.ExecutionResult) error
}

// NewService builds the package-owned execution host.
func NewService(opts ServiceOptions) *Service {
	if opts.Planner == nil {
		opts.Planner = DefaultPlanner()
	}
	if opts.ExecutionResults == nil {
		opts.ExecutionResults = make(chan taskflow.ExecutionResult, 64)
	}
	logger := opts.Config.Logger
	if logger == nil {
		logger = l.NewLogger(l.InfoLevel, &decorators.GontrollerDecorator{})
	}

	contour := chain.NewChainProcessor(opts.Errch)
	var onDependencySatisfied func(Task)
	contour.AddStep(NewEntryWithHooks(opts.TaskService, opts.Lookup, func(task Task) {
		if onDependencySatisfied != nil {
			onDependencySatisfied(task)
		}
	}, opts.In, opts.Tasks))
	contour.AddStep(NewValidator(opts.Config.Root, opts.Board.ListTasks, opts.Tasks, opts.Validated))
	contour.AddStep(NewAgentLauncher(opts.Validated, opts.Launched, opts.Planner))

	s := &Service{
		cfg:              opts.Config,
		Contour:          contour,
		tasks:            opts.TaskService,
		reload:           opts.ReloadTask,
		board:            opts.Board,
		lookup:           opts.Lookup,
		planner:          opts.Planner,
		launched:         opts.Launched,
		errch:            opts.Errch,
		executionResults: opts.ExecutionResults,
		applyResult:      opts.ApplyResult,
		controls:         map[string]chan SessionControlRequest{},
		logger:           logger,
	}
	onDependencySatisfied = func(task Task) {
		go s.RequeueDependentsOf(context.Background(), task)
	}
	s.hub = newSessionHub(opts.Config.Root, func() []tasklifecycle.Task {
		if s.board.ListTasks != nil {
			return s.board.ListTasks()
		}
		return nil
	}, func(locator string) (tasklifecycle.Task, bool) {
		if s.board.TaskByPath != nil {
			return s.board.TaskByPath(locator)
		}
		return tasklifecycle.Task{}, false
	})
	return s
}

// Process runs launch admission and the package-owned launched-task consumer.
func (s *Service) Process(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); s.Contour.Process(ctx) }()
	go func() { defer wg.Done(); s.consumeLaunched(ctx) }()
	wg.Wait()
}

func (s *Service) consumeLaunched(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-s.launched:
			if task == nil {
				continue
			}
			s.HandleLaunchResult(ctx, *task)
		}
	}
}

// ExecutionResults is the Contour 3 input channel.
func (s *Service) ExecutionResults() <-chan taskflow.ExecutionResult {
	return s.executionResults
}

// PublishExecutionResult sends a normalized outcome onto the finalizer channel.
func (s *Service) PublishExecutionResult(result taskflow.ExecutionResult) {
	if s == nil || s.executionResults == nil {
		return
	}
	select {
	case s.executionResults <- result:
	default:
		select {
		case s.executionResults <- result:
		default:
			if s.logger != nil {
				s.logger.Warn("execution result channel full; dropped publish",
					l.String("task_id", result.TaskID),
					l.String("outcome", string(result.Outcome)))
			}
		}
	}
}

func (s *Service) registerSessionControl(claimID string, ch chan SessionControlRequest) {
	s.controlsMu.Lock()
	defer s.controlsMu.Unlock()
	s.controls[claimID] = ch
}

// RegisterSessionControl registers a live provider control channel for a claim.
func (s *Service) RegisterSessionControl(claimID string, ch chan SessionControlRequest) {
	s.registerSessionControl(claimID, ch)
}

func (s *Service) unregisterSessionControl(claimID string) {
	s.controlsMu.Lock()
	defer s.controlsMu.Unlock()
	delete(s.controls, claimID)
}

// UnregisterSessionControl removes a live session control channel.
func (s *Service) UnregisterSessionControl(claimID string) {
	s.unregisterSessionControl(claimID)
}

// HasSessionControl reports whether a claim is currently provider-controllable.
func (s *Service) HasSessionControl(claimID string) bool {
	if s == nil {
		return false
	}
	s.controlsMu.Lock()
	defer s.controlsMu.Unlock()
	_, ok := s.controls[claimID]
	return ok
}

// SendSessionControl delivers an operator control request to a live session.
func (s *Service) SendSessionControl(claimID, action, input string) error {
	if s == nil {
		return fmt.Errorf("execution service is not available")
	}
	s.controlsMu.Lock()
	ch := s.controls[claimID]
	s.controlsMu.Unlock()
	if ch == nil {
		return fmt.Errorf("runtime session %q is not controllable", claimID)
	}
	req := SessionControlRequest{Action: action, Input: input, Done: make(chan error, 1)}
	select {
	case ch <- req:
	case <-time.After(5 * time.Second):
		return errors.New("timed out sending session control request")
	}
	select {
	case err := <-req.Done:
		return err
	case <-time.After(10 * time.Second):
		return errors.New("timed out waiting for session control response")
	}
}
