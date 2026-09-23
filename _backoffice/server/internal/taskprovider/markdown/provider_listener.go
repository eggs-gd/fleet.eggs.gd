package markdown

import (
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider"
	"github.com/eggs-gd/core.eggs.gd/lib/chain"
)

func (p *Provider) Listener(interval time.Duration, hooks taskprovider.ListenerHooks, out chan<- taskprovider.TaskEvent) chain.Processor {
	return NewListenerService(p, interval, ListenerHooks{
		ObserveTaskHooks: ObserveTaskHooks{
			Before: hooks.Before,
			ActiveSessionForTask: func(task taskprovider.Task) (ActiveSession, bool) {
				if hooks.ActiveSessionForTask == nil {
					return ActiveSession{}, false
				}
				session, ok := hooks.ActiveSessionForTask(task)
				if !ok {
					return ActiveSession{}, false
				}
				return ActiveSession{
					ClaimID:         session.ClaimID,
					ExecutionStatus: session.ExecutionStatus,
					Status:          session.Status,
				}, true
			},
			RevertActiveExecution: func(locator string, session ActiveSession) (taskprovider.Task, error) {
				if hooks.RevertActiveExecution == nil {
					return taskprovider.Task{}, nil
				}
				return hooks.RevertActiveExecution(locator, taskprovider.ActiveSession{
					ClaimID:         session.ClaimID,
					ExecutionStatus: session.ExecutionStatus,
					Status:          session.Status,
				})
			},
		},
	}, out)
}
