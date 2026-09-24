package runtimedb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogTailIsVisibleBeforeClose(t *testing.T) {
	root := t.TempDir()
	log, err := OpenLog(root, "_registry/sessions/claim-1.log", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := log.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if got := ReadLog(root, "_registry/sessions/claim-1.log"); got != "hello" {
		t.Fatalf("open log = %q", got)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	if got := ReadLog(root, "_registry/sessions/claim-1.log"); got != "hello" {
		t.Fatalf("closed log = %q", got)
	}
	abs := filepath.Join(root, "_registry", "sessions", "claim-1.log")
	log, err = OpenLog("", abs, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := ReadLog(root, "_registry/sessions/claim-1.log"); got != "" {
		t.Fatalf("truncated log = %q", got)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRelayAppends(t *testing.T) {
	root := t.TempDir()
	relay, err := OpenRelay(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := relay.Write([]byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := relay.Close(); err != nil {
		t.Fatal(err)
	}
	relay, err = OpenRelay(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := relay.Write([]byte("two")); err != nil {
		t.Fatal(err)
	}
	if err := relay.Close(); err != nil {
		t.Fatal(err)
	}
	body, err := relayBody(root)
	if err != nil {
		t.Fatal(err)
	}
	if body != "onetwo" {
		t.Fatalf("relay = %q", body)
	}
}

func TestImportLegacyMovesFilesOnce(t *testing.T) {
	root := t.TempDir()
	reg := filepath.Join(root, "_registry")
	sessions := filepath.Join(reg, "sessions")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(reg, "events.ndjson"), "{\"type\":\"launch_skipped\",\"task_ref\":\"CORE-1\"}\n\n{\"type\":\"nope\"\n")
	write(filepath.Join(reg, "manager-relay.log"), "relay-line\n")
	write(filepath.Join(sessions, "core-1.json"), "{\n  \"claim_id\": \"core-1\",\n  \"task_ref\": \"CORE-1\",\n  \"agent\": \"cursor\"\n}\n")
	write(filepath.Join(sessions, "core-1.log"), "log body")
	write(filepath.Join(sessions, "core-1.prompt.txt"), "prompt")

	if err := ImportLegacy(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(reg); !os.IsNotExist(err) {
		t.Fatalf("legacy dir still present: %v", err)
	}
	events, err := Events(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Type != "launch_skipped" || events[1].Type != "legacy_unparsed" {
		t.Fatalf("events = %#v", events)
	}
	bodies, err := SessionBodies(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 1 || !strings.Contains(bodies[0], `"claim_id": "core-1"`) {
		t.Fatalf("sessions = %#v", bodies)
	}
	if got := ReadLog(root, "_registry/sessions/core-1.log"); got != "log body" {
		t.Fatalf("log = %q", got)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	var extra, relay string
	if err := db.QueryRow(`SELECT body FROM extra_files WHERE name = ?`, "core-1.prompt.txt").Scan(&extra); err != nil {
		t.Fatal(err)
	}
	if extra != "prompt" {
		t.Fatalf("extra = %q", extra)
	}
	if err := db.QueryRow(`SELECT body FROM relay_log WHERE id = 1`).Scan(&relay); err != nil {
		t.Fatal(err)
	}
	if relay != "relay-line\n" {
		t.Fatalf("relay = %q", relay)
	}

	if err := os.MkdirAll(reg, 0o755); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(reg, "events.ndjson"), "{\"type\":\"again\"}\n")
	if err := ImportLegacy(root); err != nil {
		t.Fatal(err)
	}
	events, err = Events(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("second import duplicated events: %#v", events)
	}
	if _, err := os.Stat(filepath.Join(reg, "events.ndjson")); !os.IsNotExist(err) {
		t.Fatalf("leftover events file: %v", err)
	}
}
