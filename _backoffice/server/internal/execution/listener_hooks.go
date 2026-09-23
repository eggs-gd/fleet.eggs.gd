package execution

import (
	"context"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider"
)

// ActiveExecutionGuard returns the revert hook that forces a task back to
// doing when an active runtime session still owns it.
func ActiveExecutionGuard(service taskprovider.TaskService) func(locator string, session taskprovider.ActiveSession) (taskprovider.Task, error) {
	return func(locator string, session taskprovider.ActiveSession) (taskprovider.Task, error) {
		flowReverted, err := service.Patch(context.Background(), locator, taskflow.PatchInput{
			Status:              taskflow.StatusDoing,
			Comment:             tasklifecycle.ActiveExecutionGuardComment(session.ClaimID),
			CommentAuthor:       "core",
			Actor:               "runtime",
			AllowStatusOverride: true,
		})
		if err != nil {
			return taskprovider.Task{}, err
		}
		return TaskFromFlow(flowReverted), nil
	}
}
