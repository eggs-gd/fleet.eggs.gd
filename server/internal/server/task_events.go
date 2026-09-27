package server

import (
	"fmt"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/eventbus"
	"github.com/eggs-gd/fleet.eggs.gd/internal/health"
)

// auditedBus records every task event in the audit log before it reaches the
// bus. manager_events reads only that log. The row is written where the event
// is published, so a failed write comes back to the publisher, is reported to
// Health, and does not depend on a subscriber goroutine. The event still goes
// to the bus, and the task change that caused it stays applied.
type auditedBus struct {
	bus    *eventbus.Bus
	root   string
	health *health.Monitor
}

func (a *auditedBus) Publish(event eventbus.Event) error {
	if event.Channel != eventbus.ChannelTask || strings.TrimSpace(a.root) == "" {
		return a.bus.Publish(event)
	}
	err := audit.AppendEvent(a.root, audit.Event{
		Type:    event.Type,
		TaskRef: taskRefOf(event),
		Message: event.Text,
	})
	if err != nil {
		err = a.health.Fail("audit", fmt.Errorf("record %s in the audit log: %w", event.Type, err))
	} else {
		a.health.OK("audit")
	}
	if pubErr := a.bus.Publish(event); err == nil {
		err = pubErr
	}
	return err
}

// taskRefOf is the human ref of the task an event is about. Events name the task
// by provider locator (task_id), and carry the ref when the publisher knew it.
func taskRefOf(event eventbus.Event) string {
	if ref := event.Fields["task_ref"]; ref != "" {
		return ref
	}
	return event.Fields["task_id"]
}
