package runtimedb

import (
	"path/filepath"
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
