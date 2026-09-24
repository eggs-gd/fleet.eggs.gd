package execution

import "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"

type (
	TaskContext = executionapi.TaskContext
	Plan        = executionapi.Plan
	Agent       = executionapi.Agent
	Registry    = executionapi.Registry
)

func NewRegistry() *Registry {
	return executionapi.NewRegistry()
}

var ErrUnknownAgent = executionapi.ErrUnknownAgent
