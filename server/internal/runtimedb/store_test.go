package runtimedb

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// reopen drops the cached handle the way a process exit does, so the next
// call has to read everything back from disk.
func reopen(t *testing.T) {
	t.Helper()
	CloseAll()
}

func TestOpenUsesWriteAheadLogging(t *testing.T) {
	root := t.TempDir()
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(CloseAll)
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
	var timeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatal(err)
	}
	if timeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", timeout)
	}
}

func TestEventsSurviveARestartInAppendOrder(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(CloseAll)
	want := []Event{
		{Time: "2026-10-29T10:00:00Z", Type: "task.claimed", TaskRef: "FLEET-1", Agent: "claude", Message: "ünïcode — ✓"},
		{Time: "2026-10-29T10:01:00Z", Type: "task.done", TaskRef: "FLEET-1", Details: `{"ok":true}`},
	}
	for _, event := range want {
		if err := InsertEvent(root, event); err != nil {
			t.Fatal(err)
		}
	}
	reopen(t)

	got, err := Events(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("read %d events, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestManagerEventsAfterSkipsOtherTypesAndEarlierRows(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(CloseAll)
	for _, typ := range []string{"task.claimed", "session.started", "project.created", "task.done"} {
		if err := InsertEvent(root, Event{Type: typ, Message: typ}); err != nil {
			t.Fatal(err)
		}
	}

	all, err := ManagerEventsAfter(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if types := eventTypes(all); types != "task.claimed project.created task.done" {
		t.Fatalf("types = %q", types)
	}

	later, err := ManagerEventsAfter(root, all[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if types := eventTypes(later); types != "project.created task.done" {
		t.Errorf("types after the first cursor = %q", types)
	}
}

func eventTypes(events []TaskEvent) string {
	types := make([]string, len(events))
	for i, event := range events {
		types[i] = event.Type
	}
	return strings.Join(types, " ")
}

func TestUpsertSessionReplacesTheRowForAClaim(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(CloseAll)
	for _, session := range []Session{
		{ClaimID: "b", Body: `{"v":1}`},
		{ClaimID: "a", Body: `{"v":1}`},
		{ClaimID: "b", Body: `{"v":2}`},
		{ClaimID: "", Body: `{"ignored":true}`},
	} {
		if err := UpsertSession(root, session); err != nil {
			t.Fatal(err)
		}
	}
	reopen(t)

	got, err := SessionBodies(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{`{"v":1}`, `{"v":2}`}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("bodies = %v, want %v (ordered by claim id, one row per claim)", got, want)
	}
}

func TestSessionLogKeepsFlushedAndPendingTextAcrossARestart(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(CloseAll)
	log, err := OpenLog(root, "_registry/sessions/claim-1.log", true)
	if err != nil {
		t.Fatal(err)
	}
	// Past the flush threshold, so part of it is on disk and the tail is not.
	big := strings.Repeat("x", logFlushBytes+10)
	if _, err := log.Write([]byte(big)); err != nil {
		t.Fatal(err)
	}
	if _, err := log.Write([]byte("tail")); err != nil {
		t.Fatal(err)
	}
	if got := ReadLog(root, "_registry/sessions/claim-1.log"); got != big+"tail" {
		t.Fatalf("a reader sees %d bytes, want %d", len(got), len(big)+4)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	reopen(t)

	if got := ReadLog(root, "_registry/sessions/claim-1.log"); got != big+"tail" {
		t.Errorf("after restart the log has %d bytes, want %d", len(got), len(big)+4)
	}
	resumed, err := OpenLog(root, "_registry/sessions/claim-1.log", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resumed.Write([]byte("+more")); err != nil {
		t.Fatal(err)
	}
	if err := resumed.Close(); err != nil {
		t.Fatal(err)
	}
	if got := ReadLog(root, "_registry/sessions/claim-1.log"); !strings.HasSuffix(got, "tail+more") {
		t.Errorf("resumed log ends %q, want it to append", got[len(got)-12:])
	}
}

func TestRelayLogAppendsAcrossWriters(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(CloseAll)
	for _, text := range []string{"first\n", "second\n"} {
		relay, err := OpenRelay(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := relay.Write([]byte(text)); err != nil {
			t.Fatal(err)
		}
		if err := relay.Close(); err != nil {
			t.Fatal(err)
		}
	}
	reopen(t)

	body, err := relayBody(root)
	if err != nil {
		t.Fatal(err)
	}
	if body != "first\nsecond\n" {
		t.Errorf("relay body = %q", body)
	}
}

func TestConcurrentWritersAllLand(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(CloseAll)
	const writers, perWriter = 8, 25
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				if err := InsertEvent(root, Event{Type: "task.tick", Message: fmt.Sprint(w, "-", i)}); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	events, err := Events(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != writers*perWriter {
		t.Errorf("stored %d events, want %d", len(events), writers*perWriter)
	}
}
