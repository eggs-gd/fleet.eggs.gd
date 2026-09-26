package server

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/eventbus"
	"github.com/eggs-gd/fleet.eggs.gd/internal/health"
)

func collect(bus *eventbus.Bus) (*sync.WaitGroup, *[]eventbus.Event) {
	var mu sync.Mutex
	var got []eventbus.Event
	wg := &sync.WaitGroup{}
	wg.Add(1)
	bus.Subscribe(func(e eventbus.Event) {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
		wg.Done()
	})
	return wg, &got
}

func waitFor(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("event never reached the bus")
	}
}

func TestAuditedBusRecordsBeforeDelivering(t *testing.T) {
	root := t.TempDir()
	mon := health.New()
	bus := eventbus.New()
	wg, got := collect(bus)
	pub := &auditedBus{bus: bus, root: root, health: mon}

	err := pub.Publish(eventbus.Event{Channel: eventbus.ChannelTask, Type: "task.needs_review", Text: "ready", Fields: map[string]string{"task_id": "CORE-1"}})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, wg)
	rows, err := audit.TaskEventsAfter(root, 0)
	if err != nil || len(rows) != 1 || rows[0].TaskRef != "CORE-1" || rows[0].Type != "task.needs_review" {
		t.Fatalf("audit rows = %+v, %v", rows, err)
	}
	if len(*got) != 1 || len(mon.Problems()) != 0 {
		t.Fatalf("delivered=%d problems=%v", len(*got), mon.Problems())
	}
}

func TestAuditedBusFailureIsReturnedAndReportedButEventStillDelivered(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	mon := health.New()
	bus := eventbus.New()
	wg, got := collect(bus)
	pub := &auditedBus{bus: bus, root: filepath.Join(blocker, "runtime"), health: mon}

	err := pub.Publish(eventbus.Event{Channel: eventbus.ChannelTask, Type: "task.needs_attention", Text: "blocked"})
	if err == nil || !strings.Contains(err.Error(), "audit log") {
		t.Fatalf("err = %v", err)
	}
	waitFor(t, wg)
	if _, failing := mon.Has("audit"); !failing || len(*got) != 1 {
		t.Fatalf("failing=%v delivered=%d", failing, len(*got))
	}
}

func TestPreflightReportsEveryProblemTogether(t *testing.T) {
	root := t.TempDir()
	if err := preflight(Config{CoreRoot: root, RuntimeRoot: t.TempDir()}); err != nil {
		t.Fatalf("healthy start: %v", err)
	}
	for _, want := range []string{"_registry/counters.json", ".agents/skills/intake/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(want))); err != nil {
			t.Errorf("preflight did not create %s: %v", want, err)
		}
	}

	missing := filepath.Join(root, "gone")
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := preflight(Config{CoreRoot: missing, RuntimeRoot: filepath.Join(blocker, "rt")})
	if err == nil {
		t.Fatal("broken roots passed preflight")
	}
	for _, want := range []string{"data root is not writable", "runtime database is not writable"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %q:\n%v", want, err)
		}
	}
	if _, statErr := os.Stat(missing); statErr == nil {
		t.Error("preflight created a data root that did not exist")
	}
}

func TestPreflightNamesTheFailingSkillFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".agents"), []byte("in the way"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := preflight(Config{CoreRoot: root, RuntimeRoot: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "manager skills") || !strings.Contains(err.Error(), "install manager skill") {
		t.Fatalf("err = %v", err)
	}
}
