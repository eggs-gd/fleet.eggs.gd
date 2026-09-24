package providers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseGeminiStreamJSONExtractsTerminalResult(t *testing.T) {
	raw := []byte(`{"event":"init","conversation_id":"abc","init":{"cwd":"/tmp"}}
{"event":"step_update","step_update":{"conversation_id":"abc","step_index":0,"state":"DONE","step_type":"user_input"}}
{"event":"step_update","step_update":{"conversation_id":"abc","step_index":1,"state":"DONE","step_type":"agent_response","text_delta":"pong\n"}}
{"event":"result","result":{"conversation_id":"abc","status":"SUCCESS","response":"pong\n","duration_seconds":2.2,"num_turns":1}}
`)
	got := parseGeminiStreamJSON(raw)
	if got.ConversationID != "abc" || got.Status != "SUCCESS" || got.Response != "pong\n" || got.ErrorText != "" {
		t.Fatalf("got = %#v", got)
	}
}

func TestParseGeminiStreamJSONHandlesEmptyAndGarbageInput(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte(""), []byte("\n\n"), []byte("not json\nalso not json\n")} {
		got := parseGeminiStreamJSON(raw)
		if got != (geminiResultEvent{}) {
			t.Fatalf("raw=%q got = %#v, want zero value", raw, got)
		}
	}
}

func TestParseGeminiStreamJSONLastResultEventWins(t *testing.T) {
	raw := []byte(`{"event":"result","result":{"conversation_id":"first","status":"ERROR","response":""}}
{"event":"result","result":{"conversation_id":"second","status":"SUCCESS","response":"done"}}
`)
	got := parseGeminiStreamJSON(raw)
	if got.ConversationID != "second" || got.Status != "SUCCESS" || got.Response != "done" {
		t.Fatalf("got = %#v", got)
	}
}

func TestParseGeminiStreamJSONCapturesErrorField(t *testing.T) {
	raw := []byte(`{"event":"result","result":{"conversation_id":"abc","status":"ERROR","response":"","error":{"code":"RATE_LIMITED","message":"slow down"}}}
`)
	got := parseGeminiStreamJSON(raw)
	if got.Status != "ERROR" || got.ErrorText == "" {
		t.Fatalf("got = %#v", got)
	}
}

func writeGeminiProjectFile(t *testing.T, home, id, folderURI string, modTime time.Time) {
	t.Helper()
	dir := filepath.Join(home, ".gemini", "config", "projects")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, id+".json")
	body := `{"id":"` + id + `","name":"test","projectResources":{"resources":[{"folderUri":"` + folderURI + `"}]}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}

func TestFindGeminiProjectIDMatchesFolderURI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	workDir := filepath.Join(home, "Projects", "demo")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeGeminiProjectFile(t, home, "unrelated-project", "file:///somewhere/else", time.Now())
	writeGeminiProjectFile(t, home, "matching-project", "file://"+workDir, time.Now())

	got := findGeminiProjectID(workDir)
	if got != "matching-project" {
		t.Fatalf("got = %q, want matching-project", got)
	}
}

func TestFindGeminiProjectIDPicksNewestOnDuplicates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	workDir := filepath.Join(home, "Projects", "demo")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	older := time.Now().Add(-time.Hour)
	newer := time.Now()
	writeGeminiProjectFile(t, home, "older-project", "file://"+workDir, older)
	writeGeminiProjectFile(t, home, "newer-project", "file://"+workDir, newer)

	got := findGeminiProjectID(workDir)
	if got != "newer-project" {
		t.Fatalf("got = %q, want newer-project (most recently modified)", got)
	}
}

func TestFindGeminiProjectIDReturnsEmptyWhenNoneMatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeGeminiProjectFile(t, home, "unrelated-project", "file:///somewhere/else", time.Now())

	got := findGeminiProjectID(filepath.Join(home, "Projects", "demo"))
	if got != "" {
		t.Fatalf("got = %q, want empty", got)
	}
}
