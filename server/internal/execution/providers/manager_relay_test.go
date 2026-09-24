package providers

import (
	"encoding/json"
	"testing"
)

func TestParseCodexThreadListResultDedupesByID(t *testing.T) {
	// A real codex thread/list response observed in the wild: the same
	// thread id appears twice (once per history entry), newest first.
	raw := json.RawMessage(`{
		"data": [
			{"id": "01a0478a-d63b-7831-975f-19286cf293b3", "name": "Career-space Operator", "cwd": "/repo", "updatedAt": 1789455346},
			{"id": "01a0478a-d63b-7831-975f-19286cf293b3", "name": "Career-space Operator", "cwd": "/repo", "updatedAt": 1789139140},
			{"id": "01a0813a-547e-7f91-8a8b-d83d87e9c058", "name": "CORE-151", "cwd": "/core", "updatedAt": 1789375020}
		]
	}`)

	threads, err := parseCodexThreadListResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 2 {
		t.Fatalf("len(threads) = %d, want 2 (duplicate id must collapse to one row): %#v", len(threads), threads)
	}
	if threads[0].ID != "01a0478a-d63b-7831-975f-19286cf293b3" {
		t.Fatalf("threads[0].ID = %q", threads[0].ID)
	}
	// The first (newest) occurrence of the duplicated id must win.
	if threads[0].UpdatedAt == "" {
		t.Fatal("expected UpdatedAt to be set")
	}
	seen := map[string]bool{}
	for _, th := range threads {
		if seen[th.ID] {
			t.Fatalf("duplicate id survived dedupe: %s", th.ID)
		}
		seen[th.ID] = true
	}
}

func TestParseCodexThreadListResultRejectsInvalidJSON(t *testing.T) {
	_, err := parseCodexThreadListResult(json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestParseCodexThreadListResultEmpty(t *testing.T) {
	threads, err := parseCodexThreadListResult(json.RawMessage(`{"data": []}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 0 {
		t.Fatalf("len(threads) = %d, want 0", len(threads))
	}
}
