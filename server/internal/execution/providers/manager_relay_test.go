package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestFakeCodexManagerServer is not a test: it is the body of a fake
// `codex app-server` for the manager relay, selected by FLEET_FAKE_CODEX_MANAGER.
func TestFakeCodexManagerServer(t *testing.T) {
	mode := os.Getenv("FLEET_FAKE_CODEX_MANAGER")
	if mode == "" {
		t.Skip("only runs as a child process of the relay tests")
	}
	if mode == "crash" {
		os.Exit(3)
	}
	out := os.Stdout
	send := func(line string) { _, _ = out.WriteString(line + "\n") }
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(in.Bytes(), &req) != nil || req.Method == "" || len(req.ID) == 0 {
			continue
		}
		id := string(req.ID)
		reply := func(result string) { send(`{"id":` + id + `,"result":` + result + `}`) }
		switch req.Method {
		case "initialize":
			if mode == "server-request" {
				send(`{"id":900,"method":"item/tool/call","params":{}}`)
			}
			reply(`{}`)
		case "thread/resume":
			switch mode {
			case "resume-error":
				send(`{"id":` + id + `,"error":{"message":"thread is busy"}}`)
			case "resume-other":
				reply(`{"thread":{"id":"thr-other"}}`)
			case "hang":
				// never answers
			default:
				reply(`{"thread":{"id":"thr-1"}}`)
			}
		case "turn/start":
			reply(`{"turn":{"id":"turn-1"}}`)
			status := "completed"
			if mode == "turn-failed" {
				status = "failed"
			}
			send(`{"method":"item/agentMessage/delta","params":{"delta":"hi from fake"}}`)
			send(`{"method":"turn/completed","params":{"turn":{"id":"turn-1","status":"` + status + `"}}}`)
		case "thread/list":
			reply(`{"data":[{"id":"a","name":" First ","cwd":"/w","updatedAt":1700000000,"preview":"x"},{"id":"a","name":"dup"},{"id":"b","preview":"Fleet event: skip"}]}`)
		case "thread/start":
			reply(`{"thread":{"id":"thr-new"}}`)
		case "thread/name/set":
			reply(`{}`)
		}
	}
}

// useFakeCodexManager points FLEET_CODEX_BINARY at a script that runs the fake
// server in this test binary, whatever arguments codex would get.
func useFakeCodexManager(t *testing.T, mode string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake codex is a shell script")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(t.TempDir(), "codex")
	body := "#!/bin/sh\nexec '" + exe + "' -test.run='^TestFakeCodexManagerServer$'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FLEET_CODEX_BINARY", script)
	t.Setenv("FLEET_FAKE_CODEX_MANAGER", mode)
}

func TestRelayCodexManagerMessage(t *testing.T) {
	useFakeCodexManager(t, "ok")
	res, err := RelayCodexManagerMessage(context.Background(), t.TempDir(), " thr-1 ", "hello", 20*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ThreadID != "thr-1" || res.ReplyText != "hi from fake" {
		t.Fatalf("result = %+v", res)
	}
}

func TestRelayCodexManagerMessageServicesNoServerRequests(t *testing.T) {
	useFakeCodexManager(t, "server-request")
	res, err := RelayCodexManagerMessage(context.Background(), t.TempDir(), "thr-1", "hello", 20*time.Second, nil)
	if err != nil || res.ReplyText != "hi from fake" {
		t.Fatalf("result %+v err %v", res, err)
	}
}

func TestRelayCodexManagerMessageFailures(t *testing.T) {
	cases := []struct {
		name, mode, thread, prompt, message string
	}{
		{"blank prompt", "ok", "thr-1", "  ", "prompt is required"},
		{"unbound", "ok", " ", "hi", "manager session is not bound"},
		{"resume refused", "resume-error", "thr-1", "hi", "bound manager session thr-1"},
		{"resumed as another thread", "resume-other", "thr-1", "hi", "resumed as thr-other"},
		{"turn fails", "turn-failed", "thr-1", "hi", `status "failed"`},
		{"process dies", "crash", "thr-1", "hi", "closed"},
	}
	for _, c := range cases {
		useFakeCodexManager(t, c.mode)
		_, err := RelayCodexManagerMessage(context.Background(), t.TempDir(), c.thread, c.prompt, 20*time.Second, nil)
		if err == nil || !strings.Contains(err.Error(), c.message) {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.message)
		}
	}
}

func TestRelayCodexManagerMessageTimesOut(t *testing.T) {
	useFakeCodexManager(t, "hang")
	started := time.Now()
	_, err := RelayCodexManagerMessage(context.Background(), t.TempDir(), "thr-1", "hi", 600*time.Millisecond, nil)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v", err)
	}
	if time.Since(started) > 10*time.Second {
		t.Fatal("the relay did not stop at its deadline")
	}
}

func TestRelayWithoutACodexBinary(t *testing.T) {
	t.Setenv("FLEET_CODEX_BINARY", "/definitely/not/here")
	if _, err := RelayCodexManagerMessage(context.Background(), t.TempDir(), "t", "hi", time.Second, nil); err == nil || !strings.Contains(err.Error(), "resolve codex binary") {
		t.Fatalf("err = %v", err)
	}
	if _, err := ListCodexManagerThreads(context.Background(), time.Second, nil); err == nil {
		t.Fatal("listing needs the binary too")
	}
	if _, err := StartCodexManagerThread(context.Background(), t.TempDir(), nil); err == nil {
		t.Fatal("starting a thread needs the binary too")
	}
}

func TestListCodexManagerThreads(t *testing.T) {
	useFakeCodexManager(t, "ok")
	threads, err := ListCodexManagerThreads(context.Background(), 20*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || threads[0].ID != "a" || threads[0].Name != "First" || threads[0].Cwd != "/w" || threads[0].UpdatedAt == "" {
		t.Fatalf("threads = %+v", threads)
	}
}

func TestStartCodexManagerThread(t *testing.T) {
	useFakeCodexManager(t, "ok")
	id, err := StartCodexManagerThread(context.Background(), t.TempDir(), nil)
	if err != nil || id != "thr-new" {
		t.Fatalf("id %q err %v", id, err)
	}
}
