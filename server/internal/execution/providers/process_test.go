package providers

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
)

func sessionLogPath(t *testing.T, claim string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "_registry", "sessions", claim+".log")
}

func TestStartProcessRequiresACommand(t *testing.T) {
	if _, err := StartProcess(context.Background(), Plan{}, sessionLogPath(t, "c1"), time.Minute); err == nil {
		t.Fatal("started with no command")
	}
}

func TestStartProcessRunsTheCommandAndLogsItsOutput(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "agent.sh", `echo "hello from agent"; echo "oops" >&2`)
	logPath := sessionLogPath(t, "c1")

	process, err := StartProcess(context.Background(), Plan{Agent: "claude", Command: []string{script}, WorkingDir: dir}, logPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := process.Cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	process.Close()
	process.Close() // closing twice must be harmless

	log := runtimedb.ReadLog("", logPath)
	for _, want := range []string{"[core] starting agent=claude", "hello from agent", "oops"} {
		if !strings.Contains(log, want) {
			t.Errorf("log lacks %q:\n%s", want, log)
		}
	}
	if process.Timeout != 30*time.Minute {
		t.Fatalf("default timeout = %s", process.Timeout)
	}
}

func TestStartProcessKillsTheCommandWhenItsTimeoutPasses(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "slow.sh", `sleep 30`)
	process, err := StartProcess(context.Background(), Plan{Command: []string{script}, WorkingDir: dir}, sessionLogPath(t, "c2"), 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer process.Close()
	done := make(chan error, 1)
	go func() { done <- process.Cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a killed process reported success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the process outlived its timeout")
	}
}

func TestStartProcessReportsABinaryThatDoesNotExist(t *testing.T) {
	logPath := sessionLogPath(t, "c3")
	if _, err := StartProcess(context.Background(), Plan{Command: []string{"/definitely/not/a/binary"}}, logPath, time.Minute); err == nil {
		t.Fatal("started a missing binary")
	}
	if log := runtimedb.ReadLog("", logPath); !strings.Contains(log, "[core] start failed") {
		t.Fatalf("the failure is not in the log:\n%s", log)
	}
}
