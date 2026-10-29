// Package taskprovider defines the Provider surface Fleet uses for bootstrap
// List/Load, claims and optional ChangeSource polling.
//
// The application contract is taskflow.TaskProvider + TaskService
// (_docs/ARCHITECTURE.md). Markdown and Plane both implement:
//
//   - taskprovider.Provider for Runtime bootstrap and claim paths;
//   - Provider.Flow() as taskflow.TaskProvider for TaskService;
//   - Plane also implements ChangeSource so the listener polls via
//     taskprovider.NewListenerService without Type() string branches.
//
// Manager, dashboard, launcher, and finalizer writes go through TaskService.
//
// A locator (the string argument to Load/Claim, and TaskPatch.Path/
// TaskCreate.Path) is an opaque, provider-scoped identifier for one task.
// Callers must treat it as opaque.
package taskprovider

import "github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"

// Task, TaskPatch, and TaskCreateRequest are the provider-neutral types
// every adapter reads and produces.
type (
	Task              = tasklifecycle.Task
	TaskPatch         = tasklifecycle.TaskPatch
	TaskCreateRequest = tasklifecycle.TaskCreateRequest
)

// Provider is the contract Fleet orchestration uses to read and claim tasks.
// Write paths go through taskflow.TaskService.
type Provider interface {
	Type() string
	List() ([]Task, error)
	Load(locator string) (Task, error)
	CreateFromRequest(req TaskCreateRequest) (Task, error)
	Mutate(patch TaskPatch) (Task, error)
	Claim(locator string) (Task, error)
}
