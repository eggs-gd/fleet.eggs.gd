package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func TestClassifyManagerErrorNeverLeaksProviderOutput(t *testing.T) {
	signIn := errors.New("gemini manager session: exit status 1: Authentication required. Please visit the URL to log in:\n  https://accounts.google.com/o/oauth2/auth?code_challenge=SECRET&state=SECRET\nWaiting for authentication")
	cases := []struct {
		name     string
		err      error
		wantCode string
	}{
		{"sign-in", signIn, sessionAuthRequired},
		{"timeout", fmt.Errorf("start: %w", context.DeadlineExceeded), sessionTimedOut},
		{"missing binary", fmt.Errorf("start: %w", exec.ErrNotFound), sessionNotInstalled},
		{"other", errors.New("boom\nsecond line https://example.com/private?token=SECRET"), sessionProviderError},
	}
	for _, c := range cases {
		code, message := classifyManagerError("gemini", c.err)
		if code != c.wantCode {
			t.Errorf("%s: code = %q, want %q", c.name, code, c.wantCode)
		}
		for _, leak := range []string{"SECRET", "https://", "code_challenge"} {
			if strings.Contains(message, leak) {
				t.Errorf("%s: message leaks %q: %s", c.name, leak, message)
			}
		}
		if strings.Contains(message, "\n") || len(message) > 300 {
			t.Errorf("%s: message is not short and single-line: %q", c.name, message)
		}
	}
}

func TestFirstLineRemovesLinksAndTruncates(t *testing.T) {
	if got := firstLine("\n  see https://x.example/a?b=c now\nnext"); got != "see [link removed] now" {
		t.Fatalf("got %q", got)
	}
	if got := firstLine(strings.Repeat("é", 500)); len([]rune(got)) != 201 {
		t.Fatalf("truncated to %d runes", len([]rune(got)))
	}
	if got := firstLine(" \n "); got != "no details" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteManagerSessionErrorSendsJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeManagerSessionError(rec, "gemini", errString("workspace is not the data root"))
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 400 || body["code"] != sessionInvalid || body["error"] != "workspace is not the data root" {
		t.Fatalf("status %d body %v", rec.Code, body)
	}
	rec = httptest.NewRecorder()
	writeManagerSessionError(rec, "gemini", errors.New("Authentication required. https://accounts.example/oauth?state=SECRET"))
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["code"] != sessionAuthRequired || strings.Contains(rec.Body.String(), "SECRET") {
		t.Fatalf("body %s", rec.Body.String())
	}
}
