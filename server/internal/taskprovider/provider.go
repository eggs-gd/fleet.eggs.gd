// Package taskprovider defines the migration-era Provider surface Core still
// uses for bootstrap List/Load and optional ChangeSource polling.
//
// The target application contract is taskflow.TaskProvider + TaskService
// (_docs/TASK_FLOW_ARCHITECTURE.md). Markdown and Plane both implement:
//
//   - taskprovider.Provider for Runtime bootstrap/claim legacy paths;
//   - Provider.Flow() as taskflow.TaskProvider for TaskService;
//   - Plane also implements ChangeSource so the listener polls via
//     taskprovider.NewListenerService without Type() string branches (CORE-107).
//
// Manager, dashboard, launcher, and finalizer writes go through TaskService.
// flowadapter remains only for legacy Provider implementations that lack a
// native Flow view.
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

// Provider is the sole contract Core orchestration uses today to read and
// mutate tasks. Prefer taskflow.TaskService for new write paths (CORE-101+).
type Provider interface {
	Type() string
	List() ([]Task, error)
	Load(locator string) (Task, error)
	CreateFromRequest(req TaskCreateRequest) (Task, error)
	Mutate(patch TaskPatch) (Task, error)
	Claim(locator string) (Task, error)
}
