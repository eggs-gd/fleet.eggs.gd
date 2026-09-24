package server

import (
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
)

// observeTaskFileChange runs the live markdown observe path with runtime store
// lookups and the execution active-session guard.
func observeTaskFileChange(t *testing.T, app *App, taskPath string) markdown.ObservedTaskChange {
	t.Helper()
	provider, ok := app.TaskStore.(*markdown.Provider)
	if !ok {
		t.Fatalf("taskStore type = %T, want *markdown.Provider", app.TaskStore)
	}
	observed, err := provider.ObserveTaskFileChange(taskPath, markdown.ObserveTaskHooks{
		Before: func(locator string) (tasklifecycle.Task, bool) {
			return app.Store.TaskByPath(locator)
		},
		ActiveSessionForTask: func(task tasklifecycle.Task) (markdown.ActiveSession, bool) {
			session, ok := app.Exec.ActiveSessionForTask(task)
			if !ok {
				return markdown.ActiveSession{}, false
			}
			return markdown.ActiveSession{
				ClaimID:         session.ClaimID,
				ExecutionStatus: session.ExecutionStatus,
				Status:          session.Status,
			}, true
		},
		RevertActiveExecution: func(locator string, session markdown.ActiveSession) (tasklifecycle.Task, error) {
			return execution.ActiveExecutionGuard(app.TaskService())(locator, taskprovider.ActiveSession{
				ClaimID:         session.ClaimID,
				ExecutionStatus: session.ExecutionStatus,
				Status:          session.Status,
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return observed
}
