package markdown

import (
	"context"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
)

// Flow returns a taskflow.TaskProvider view of this Markdown provider.
//
// *Provider itself implements taskprovider.Provider (Type/List/Load/Mutate/…).
// The application contract uses different List/Create signatures, so TaskService
// wires through Flow() instead of casting *Provider directly.
func (p *Provider) Flow() taskflow.TaskProvider {
	return flowView{inner: p}
}

type flowView struct {
	inner *Provider
}

func (v flowView) Create(ctx context.Context, input taskflow.CreateTask) (taskflow.Task, error) {
	return v.inner.flowCreate(ctx, input)
}

func (v flowView) Get(ctx context.Context, id string) (taskflow.Task, error) {
	return v.inner.flowGet(ctx, id)
}

func (v flowView) List(ctx context.Context, filter taskflow.TaskFilter) ([]taskflow.Task, error) {
	return v.inner.flowList(ctx, filter)
}

func (v flowView) Update(ctx context.Context, task taskflow.Task) error {
	return v.inner.flowUpdate(ctx, task)
}

func (v flowView) AddComment(ctx context.Context, id string, text string) error {
	return v.inner.flowAddComment(ctx, id, text)
}

var _ taskflow.TaskProvider = flowView{}
