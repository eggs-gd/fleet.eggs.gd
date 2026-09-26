// Package eventbus is Fleet's in-process event bus. Publishers name a
// channel and a type. Subscribers do not belong to a channel: a task
// event and a later release event travel the same Publish path.
package eventbus

import (
	"errors"
	"fmt"
	"sync"
)

// Channel is a named stream of events. Task is the only channel with a
// publisher today. The others exist so a later publisher can use the same
// bus without a second delivery scheme.
type Channel string

const (
	ChannelTask          Channel = "task"
	ChannelRelease       Channel = "release"
	ChannelWorker        Channel = "worker"
	ChannelPractice      Channel = "practice"
	ChannelConfiguration Channel = "configuration"
)

// Event is one fact Fleet wants a subscriber to hear. Text is what a person
// should read. Fields carry structured bits (a task id, a version) that
// the text may also mention.
type Event struct {
	Channel Channel
	Type    string
	Text    string
	Fields  map[string]string
}

// Handler receives a copy of a published event. It must not assume it runs
// on the publisher's goroutine, and it must not block the bus: Publish
// returns before handlers run.
type Handler func(Event)

// ErrUnknownChannel is returned when Publish is given a channel this bus
// does not accept. The event is not delivered.
var ErrUnknownChannel = errors.New("unknown event channel")

// Bus delivers events to every subscriber. It has no disk queue and no retry.
type Bus struct {
	mu   sync.Mutex
	subs []Handler
}

// New returns an empty bus.
func New() *Bus {
	return &Bus{}
}

// Subscribe registers handler for every later Publish. The handler is
// invoked on its own goroutine. A panic in one handler does not stop the
// others and does not surface to the publisher.
func (b *Bus) Subscribe(handler Handler) {
	if handler == nil {
		return
	}
	b.mu.Lock()
	b.subs = append(b.subs, handler)
	b.mu.Unlock()
}

// Publish delivers a copy of event to current subscribers and returns
// without waiting for them. An unknown channel is rejected and delivered
// to nobody.
func (b *Bus) Publish(event Event) error {
	if !knownChannel(event.Channel) {
		return fmt.Errorf("%w: %q", ErrUnknownChannel, event.Channel)
	}
	event = event.clone()

	b.mu.Lock()
	subs := append([]Handler(nil), b.subs...)
	b.mu.Unlock()

	for _, handler := range subs {
		go runHandler(handler, event)
	}
	return nil
}

func knownChannel(channel Channel) bool {
	switch channel {
	case ChannelTask, ChannelRelease, ChannelWorker, ChannelPractice, ChannelConfiguration:
		return true
	default:
		return false
	}
}

func runHandler(handler Handler, event Event) {
	defer func() { _ = recover() }()
	handler(event)
}

func (e Event) clone() Event {
	if len(e.Fields) == 0 {
		e.Fields = nil
		return e
	}
	fields := make(map[string]string, len(e.Fields))
	for k, v := range e.Fields {
		fields[k] = v
	}
	e.Fields = fields
	return e
}
