package markdown

import (
	"context"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/lib/chain"
)

type ListenerHooks struct {
	ObserveTaskHooks
}

// NewListenerService owns Markdown file watching plus task-file normalization.
// It uses the runtime Provider instance so AfterMutate/AfterObserve hooks stay attached.
func NewListenerService(
	provider *Provider,
	interval time.Duration,
	hooks ListenerHooks,
	taskOut chan<- taskflow.TaskEvent,
) chain.Processor {
	if provider == nil {
		provider = New("", Hooks{})
	}
	return chain.NewEntryPoint(taskOut, &listenerService{
		walker:   &FsWalker{root: provider.root, interval: interval, seen: map[string]fileSignature{}},
		provider: provider,
		hooks:    hooks,
	})
}

type listenerService struct {
	walker   *FsWalker
	provider *Provider
	hooks    ListenerHooks
}

func (service *listenerService) Start(chout chan<- taskflow.TaskEvent, ctx context.Context) {
	events := make(chan FileEvent)
	go service.walker.Start(events, ctx)

	for {
		select {
		case <-ctx.Done():
			service.walker.Stop()
			return
		case event := <-events:
			if !IsTaskFile(event.Path) {
				continue
			}
			observed, err := service.provider.ObserveTaskFileChange(event.Path, service.hooks.ObserveTaskHooks)
			if err != nil {
				continue
			}
			if err := service.provider.applyObservedChange(observed); err != nil {
				continue
			}
			select {
			case <-ctx.Done():
				service.walker.Stop()
				return
			case chout <- observed.Event:
			}
		}
	}
}

func (service *listenerService) Decorate(event taskflow.TaskEvent) (taskflow.TaskEvent, error) {
	if strings.TrimSpace(event.TaskID) == "" {
		return taskflow.TaskEvent{}, chain.ErrSkippedItem
	}
	return event, nil
}

func (service *listenerService) Stop() {}

func (p *Provider) applyObservedChange(observed ObservedTaskChange) error {
	if err := RebuildWorkIndex(p.root); err != nil {
		return err
	}
	if p.hooks.AfterObserve == nil {
		return nil
	}
	return p.hooks.AfterObserve(observed)
}
