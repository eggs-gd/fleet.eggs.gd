package taskprovider

import "github.com/eggs-gd/core.eggs.gd/internal/taskflow"

// ObservedChange is one provider-backed task mutation already observed and
// normalized into a TaskEvent. Change adapters feed Event into the execution
// chain; Task refreshes the in-memory dashboard store.
type ObservedChange struct {
	Event taskflow.TaskEvent
	Task  Task
}

// ChangeSource is an optional capability for providers that discover external
// changes by polling (or equivalent). Markdown uses FsWalker instead and does
// not implement this.
//
// Runtime wires taskprovider.NewListenerService, which polls ChangeSource when
// the active provider exposes it. Generic chain code must not branch on
// Provider.Type() strings or concrete adapter types (CORE-107).
type ChangeSource interface {
	ObserveChanges() ([]ObservedChange, error)
}
