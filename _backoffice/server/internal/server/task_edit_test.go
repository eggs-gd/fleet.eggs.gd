package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/eggs-gd/core.eggs.gd/internal/audit"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider"
	tpopen "github.com/eggs-gd/core.eggs.gd/internal/taskprovider/open"
)

// These tests cover board-projection side effects of task mutation (the
// dashboard projection, the generated Work/INDEX.md, and the event log) that
// Markdown itself no longer knows about — those are wired in by
// tpopen.NewMarkdown. Lifecycle-only mutation behavior (patch
// validation, status transitions, claim semantics, optimistic concurrency)
// is covered directly in the taskprovider/markdown package.

func TestTaskStoreCreateWritesTaskAndRefreshesIndex(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-create-task.md")
	store := tpopen.NewMarkdown(root, taskprovider.NewBoard(root))

	task, err := store.CreateFile(tasklifecycle.TaskCreate{
		Path:    taskPath,
		Content: testTaskMarkdown("CORE-991", "Created task", "backlog"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Ref != "CORE-991" {
		t.Fatalf("ref = %q, want CORE-991", task.Ref)
	}

	data, err := os.ReadFile(filepath.Join(root, "Work", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "`CORE-991`") {
		t.Fatalf("index missing created task\n%s", data)
	}
}

func TestClaimTaskForLaunchMovesNeedsReworkTaskToDoing(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-test-task.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-996", "Fix me again", "needs_rework"))

	task, err := tpopen.NewMarkdown(root, nil).Claim(taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "doing" {
		t.Fatalf("status = %q, want doing", task.Status)
	}

	file, err := os.Open(audit.EventLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event audit.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if event.Type != "task_claimed" {
			continue
		}
		if event.Message != "Task claimed for launch from needs_rework" {
			t.Fatalf("message = %q, want needs_rework pickup", event.Message)
		}
		if event.Details["pickup_status"] != "needs_rework" {
			t.Fatalf("pickup_status = %#v, want needs_rework", event.Details["pickup_status"])
		}
		return
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	t.Fatal("task_claimed event missing")
}

func TestClaimTaskForLaunchAllowsOnlyOneConcurrentClaim(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-test-task.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-992", "Concurrent claim", "todo"))

	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < cap(results); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := tpopen.NewMarkdown(root, nil).Claim(taskPath)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	claimed := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, tasklifecycle.ErrTaskAlreadyClaimed):
			claimed++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successes = %d, want 1", successes)
	}
	if claimed != 7 {
		t.Fatalf("already claimed errors = %d, want 7", claimed)
	}

	data, err := os.ReadFile(filepath.Join(root, "Work", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "| `doing` | 1 |") {
		t.Fatalf("index was not rebuilt after claim\n%s", data)
	}

	file, err := os.Open(audit.EventLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	seenClaimed := false
	seenRejected := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event audit.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if event.Type == "task_claimed" {
			seenClaimed = true
		}
		if event.Type == "task_claim_rejected" {
			seenRejected = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if !seenClaimed || !seenRejected {
		t.Fatalf("claim events missing: claimed=%t rejected=%t", seenClaimed, seenRejected)
	}
}
