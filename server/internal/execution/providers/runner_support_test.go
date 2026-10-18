package providers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	execution "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func TestValidateSessionCommand(t *testing.T) {
	cases := []struct {
		command []string
		wantErr string
	}{
		{nil, "empty"},
		{[]string{"  "}, "binary is empty"},
		{[]string{"claude", "--x"}, ""},
	}
	for _, c := range cases {
		err := validateSessionCommand(RuntimeSession{Command: c.command})
		if (c.wantErr == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), c.wantErr)) {
			t.Errorf("%q: err = %v, want %q", c.command, err, c.wantErr)
		}
	}
}

func TestTruncateDisplayText(t *testing.T) {
	if got := truncateDisplayText("short"); got != "short" {
		t.Fatalf("short = %q", got)
	}
	if got := truncateDisplayText(strings.Repeat("x", 500)); len(got) != 240 {
		t.Fatalf("long text kept %d bytes, want 240", len(got))
	}
}

func TestResetTimerRearmsAFiredTimerWithoutAStaleTick(t *testing.T) {
	timer := time.NewTimer(time.Millisecond)
	time.Sleep(20 * time.Millisecond) // fired, tick not yet read
	resetTimer(timer, 60*time.Millisecond)
	select {
	case <-timer.C:
		t.Fatal("a stale tick leaked through the reset")
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case <-timer.C:
	case <-time.After(time.Second):
		t.Fatal("the rearmed timer never fired")
	}
}

func TestProviderCommandOutput(t *testing.T) {
	dir := t.TempDir()
	ok := writeScript(t, dir, "ok.sh", `echo "out"; echo "err" >&2`)
	out, err := providerCommandOutput(time.Second, ok)
	if err != nil || !strings.Contains(string(out), "out") || !strings.Contains(string(out), "err") {
		t.Fatalf("out = %q, err = %v (stdout and stderr are both wanted)", out, err)
	}
	slow := writeScript(t, dir, "slow.sh", `exec sleep 5`)
	if _, err := providerCommandOutput(50*time.Millisecond, slow); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a slow command returned %v, want a deadline error", err)
	}
	if _, err := providerCommandOutput(time.Second, "/no/such/binary"); err == nil {
		t.Fatal("a missing binary reported success")
	}
}

func TestRunnerFallbacksWhenACallbackIsNotSet(t *testing.T) {
	var upserts []RuntimeSession
	opts := RunnerOptions{Callbacks: RunnerCallbacks{
		UpsertSession: func(s RuntimeSession) { upserts = append(upserts, s) },
	}}
	session := RuntimeSession{ClaimID: "c", LogPath: "sessions/c.log"}
	task := tasklifecycle.Task{ID: "t", Ref: "CORE-1"}

	recordProviderStart(opts, session, task, "started", nil)
	completeProviderSession(opts, session, task, "done", nil)
	if len(upserts) != 2 {
		t.Fatalf("start/complete fell back to %d upserts, want 2", len(upserts))
	}

	applyWorkerReportedResult(opts, &session, task, &tasklifecycle.WorkerResult{Outcome: "completed"})
	if session.Result == nil || len(upserts) != 3 {
		t.Fatalf("the reported result was not applied and published: %#v, %d", session.Result, len(upserts))
	}

	if classifyProviderError(opts, "codex", "b", "op", errors.New("x"), "") != nil {
		t.Fatal("an unset classifier must classify nothing")
	}
	failLaunch(opts, session, task, errors.New("x")) // no callback: must not panic
}

func TestOperatorAttentionFallsBackToATaskComment(t *testing.T) {
	var comments []string
	var locators []string
	opts := RunnerOptions{Callbacks: RunnerCallbacks{
		AddTaskComment: func(locator, comment string) error {
			locators = append(locators, locator)
			comments = append(comments, comment)
			return nil
		},
	}}
	session := RuntimeSession{LogPath: "sessions/c.log"}
	markOperatorAttention(opts, session, tasklifecycle.Task{RelativePath: "Work/p/tasks/t.md"}, "needs you")
	if len(comments) != 1 || !strings.Contains(comments[0], "needs you") || !strings.Contains(comments[0], "sessions/c.log") {
		t.Fatalf("comment = %v", comments)
	}
	if locators[0] != "Work/p/tasks/t.md" {
		t.Fatalf("locator = %q", locators[0])
	}

	// A task with no locator at all gets no comment rather than a bad write.
	comments = nil
	addTaskComment(opts, tasklifecycle.Task{}, "x")
	if comments != nil {
		t.Fatalf("commented on a task with no locator: %v", comments)
	}
}

func TestMergeSessionToolCallsIsNilSafeAndAccumulates(t *testing.T) {
	mergeSessionToolCalls(nil, []execution.ToolCallEvidence{{Name: "x"}}) // must not panic
	session := &RuntimeSession{}
	mergeSessionToolCalls(session, nil)
	if session.ToolUsage != nil {
		t.Fatal("no calls must leave tool usage unset")
	}
	mergeSessionToolCalls(session, []execution.ToolCallEvidence{{Name: "mcp__svelte__list-sections"}})
	if session.ToolUsage == nil || len(session.ToolUsage.Used) != 1 {
		t.Fatalf("tool usage = %#v", session.ToolUsage)
	}
	refreshSessionToolUsageFromText(session, "") // empty text, no calls: unchanged
	refreshSessionToolUsageFromText(nil, "anything")
}

func TestAppendSessionLogLineToleratesAMissingLog(t *testing.T) {
	appendSessionLogLine(nil, "ignored %d", 1)
	var b strings.Builder
	appendSessionLogLine(&b, "line %d\n", 2)
	if b.String() != "line 2\n" {
		t.Fatalf("got %q", b.String())
	}
}
