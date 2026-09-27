package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestServeUntilDoneStopsWhenTheContextEnds(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "ok") })}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serveUntilDone(ctx, srv, l) }()

	resp, err := http.Get("http://" + l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("shutdown returned %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serveUntilDone did not stop")
	}
	if _, err := http.Get("http://" + l.Addr().String()); err == nil {
		t.Fatal("server still accepts connections after shutdown")
	}
}

func TestGuardLimitsRequestBodies(t *testing.T) {
	const token = "tok"
	var read int64
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := io.Copy(io.Discard, r.Body)
		read = n
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		}
	})
	handler := guardLocal("127.0.0.1:8787", token, next)
	post := func(path string, size int) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(strings.Repeat("a", size)))
		req.Host = "127.0.0.1:8787"
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if rec := post("/api/tasks", jsonBodyLimit-1); rec.Code != http.StatusOK {
		t.Fatalf("body under the limit: %d", rec.Code)
	}
	if rec := post("/api/tasks", jsonBodyLimit+1); rec.Code != http.StatusRequestEntityTooLarge || read > jsonBodyLimit {
		t.Fatalf("oversized JSON body: status %d, read %d", rec.Code, read)
	}
	if bodyLimit("/api/manager/audio") <= jsonBodyLimit {
		t.Fatal("the audio upload must allow more than a JSON body")
	}
}
