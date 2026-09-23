package taskprovider

import (
	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
)

type (
	TaskService     = taskflow.TaskService
	Service         = taskflow.Service
	Status          = taskflow.Status
	CreateTask      = taskflow.CreateTask
	TaskFilter      = taskflow.TaskFilter
	PatchInput      = taskflow.PatchInput
	TransitionMeta  = taskflow.TransitionMeta
	ExecutionResult = taskflow.ExecutionResult
	TaskEvent       = taskflow.TaskEvent
	TaskEventSource = taskflow.TaskEventSource
	TaskEventKind   = taskflow.TaskEventKind
)

type flowProvider interface {
	Flow() taskflow.TaskProvider
}

// NewService returns the canonical task boundary for callers. During the
// migration it reuses taskflow.Service internally, but callers should depend on
// taskprovider rather than constructing taskflow.Service directly.
func NewService(provider Provider) *Service {
	return taskflow.NewService(flowContract(provider))
}

func flowContract(provider Provider) taskflow.TaskProvider {
	if provider == nil {
		panic("taskprovider: NewService requires a non-nil Provider")
	}
	if flow, ok := provider.(flowProvider); ok {
		return flow.Flow()
	}
	panic("taskprovider: Provider must implement Flow() to back NewService")
}
