package eventbus

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestPublishDeliversTaskAndReleaseToTheSameSubscriber(t *testing.T) {
	bus := New()
	got := make(chan Event, 2)
	bus.Subscribe(func(event Event) {
		got <- event
	})

	for _, event := range []Event{
		{Channel: ChannelTask, Type: "task.needs_review", Text: "ready"},
		{Channel: ChannelRelease, Type: "release.available", Text: "0.0.2"},
	} {
		if err := bus.Publish(event); err != nil {
			t.Fatal(err)
		}
	}

	seen := map[Channel]string{}
	deadline := time.After(time.Second)
	for len(seen) < 2 {
		select {
		case event := <-got:
			seen[event.Channel] = event.Type
		case <-deadline:
			t.Fatalf("delivered %#v, want task and release", seen)
		}
	}
	if seen[ChannelTask] != "task.needs_review" || seen[ChannelRelease] != "release.available" {
		t.Fatalf("delivered %#v", seen)
	}
}

func TestPublishRejectsUnknownChannel(t *testing.T) {
	bus := New()
	delivered := make(chan struct{}, 1)
	bus.Subscribe(func(Event) {
		delivered <- struct{}{}
	})

	err := bus.Publish(Event{Channel: "dashboard", Type: "nope", Text: "no"})
	if !errors.Is(err, ErrUnknownChannel) {
		t.Fatalf("err = %v, want unknown channel", err)
	}
	select {
	case <-delivered:
		t.Fatal("unknown channel was delivered")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPublishSurvivesSubscriberPanic(t *testing.T) {
	bus := New()
	got := make(chan Event, 1)
	bus.Subscribe(func(Event) { panic("subscriber blew up") })
	bus.Subscribe(func(event Event) { got <- event })

	if err := bus.Publish(Event{Channel: ChannelWorker, Type: "worker.offline", Text: "codex"}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-got:
		if event.Type != "worker.offline" {
			t.Fatalf("type = %q", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("panic in one subscriber ate the event")
	}
}

func TestPublishDoesNotWaitForSubscribers(t *testing.T) {
	bus := New()
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	bus.Subscribe(func(Event) {
		once.Do(func() { close(started) })
		<-release
	})

	done := make(chan struct{})
	go func() {
		if err := bus.Publish(Event{Channel: ChannelPractice, Type: "practice.available", Text: "note"}); err != nil {
			t.Errorf("publish: %v", err)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Publish waited for the subscriber")
	}
	close(release)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("subscriber never ran")
	}
}

func TestPublishCopiesFields(t *testing.T) {
	bus := New()
	got := make(chan Event, 1)
	bus.Subscribe(func(event Event) { got <- event })

	fields := map[string]string{"task_id": "CORE-1"}
	if err := bus.Publish(Event{Channel: ChannelTask, Type: "task.needs_review", Fields: fields}); err != nil {
		t.Fatal(err)
	}
	fields["task_id"] = "mutated"

	select {
	case event := <-got:
		if event.Fields["task_id"] != "CORE-1" {
			t.Fatalf("fields = %#v, publisher mutation leaked", event.Fields)
		}
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
}
