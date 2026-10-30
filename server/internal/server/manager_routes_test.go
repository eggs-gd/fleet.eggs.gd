package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/manager"
	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
)

func managerMux(t *testing.T) (*http.ServeMux, Config) {
	t.Helper()
	cfg := Config{CoreRoot: t.TempDir(), RuntimeRoot: t.TempDir()}
	mux := http.NewServeMux()
	registerManagerRoutes(mux, manager.NewService(nil, nil), cfg)
	return mux, cfg
}

func call(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestManagerRoutesRejectTheWrongMethod(t *testing.T) {
	mux, _ := managerMux(t)
	for path, method := range map[string]string{
		"/api/manager/schema":     http.MethodPost,
		"/api/manager/vocabulary": http.MethodPost,
		"/api/manager/binding":    http.MethodPost,
		"/api/manager/threads":    http.MethodPost,
		"/api/manager/text":       http.MethodGet,
		"/api/manager/command":    http.MethodGet,
		"/api/manager/audio":      http.MethodGet,
	} {
		if rec := call(mux, method, path, ""); rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s = %d", method, path, rec.Code)
		}
	}
}

func TestManagerReadRoutes(t *testing.T) {
	mux, cfg := managerMux(t)
	if rec := call(mux, http.MethodGet, "/api/manager/schema", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "kind") {
		t.Fatalf("schema = %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(mux, http.MethodGet, "/api/manager/vocabulary", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "terms") {
		t.Fatalf("vocabulary = %d", rec.Code)
	}
	if rec := call(mux, http.MethodGet, "/api/manager/binding", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"bound":false`) {
		t.Fatalf("binding = %d %s", rec.Code, rec.Body.String())
	}
	if err := settings.SaveOverlay(cfg.CoreRoot, settings.Overlay{Manager: settings.ManagerOverlay{Agent: "codex", ThreadID: "t-1"}}); err != nil {
		t.Fatal(err)
	}
	if rec := call(mux, http.MethodGet, "/api/manager/binding", ""); !strings.Contains(rec.Body.String(), `"bound":true`) {
		t.Fatalf("a saved binding is reported: %s", rec.Body.String())
	}
}

func TestManagerThreadsNeedsAnAgent(t *testing.T) {
	mux, _ := managerMux(t)
	if rec := call(mux, http.MethodGet, "/api/manager/threads", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("no agent = %d", rec.Code)
	}
	// An agent that cannot be listed is a bad gateway, not a crash.
	if rec := call(mux, http.MethodGet, "/api/manager/threads?agent=unknown-agent", ""); rec.Code != http.StatusBadGateway {
		t.Fatalf("unknown agent = %d %s", rec.Code, rec.Body.String())
	}
}

func TestManagerTextAndCommandValidateTheirBodies(t *testing.T) {
	mux, _ := managerMux(t)
	if rec := call(mux, http.MethodPost, "/api/manager/text", "{not json"); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json = %d", rec.Code)
	}
	if rec := call(mux, http.MethodPost, "/api/manager/text", `{"text":"  "}`); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "text is required") {
		t.Fatalf("blank text = %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(mux, http.MethodPost, "/api/manager/command", "{not json"); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json = %d", rec.Code)
	}
	if rec := call(mux, http.MethodPost, "/api/manager/command", `{"intent":{"kind":"teleport"}}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown kind = %d %s", rec.Code, rec.Body.String())
	}
}

func multipartAudio(t *testing.T, field string, content string, contentType string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if field != "" {
		part, err := w.CreateFormFile(field, "clip.wav")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte(content))
	}
	_ = w.WriteField("source", "voice")
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

func TestManagerAudio(t *testing.T) {
	mux, _ := managerMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/manager/audio", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("not multipart = %d", rec.Code)
	}

	body, ct := multipartAudio(t, "", "", "")
	req = httptest.NewRequest(http.MethodPost, "/api/manager/audio", body)
	req.Header.Set("Content-Type", ct)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "audio file field is required") {
		t.Fatalf("no file = %d %s", rec.Code, rec.Body.String())
	}

	// Speech-to-text is not configured here, which the API reports as unavailable.
	body, ct = multipartAudio(t, "audio", "RIFF....", "")
	req = httptest.NewRequest(http.MethodPost, "/api/manager/audio", body)
	req.Header.Set("Content-Type", ct)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("no STT = %d %s", rec.Code, rec.Body.String())
	}
}

func TestManagerResponseStatus(t *testing.T) {
	cases := []struct {
		name string
		resp manager.Response
		want int
	}{
		{"ok", manager.Response{OK: true}, http.StatusOK},
		{"failure without detail", manager.Response{}, http.StatusBadRequest},
		{"stt", manager.Response{Failure: &manager.Failure{Code: manager.FailureSTTNotConfigured}}, http.StatusServiceUnavailable},
		{"llm", manager.Response{Failure: &manager.Failure{Code: manager.FailureLLMNotConfigured}}, http.StatusServiceUnavailable},
		{"not found", manager.Response{Failure: &manager.Failure{Code: manager.FailureNotFound}}, http.StatusNotFound},
		{"provider", manager.Response{Failure: &manager.Failure{Code: manager.FailureProviderError}}, http.StatusBadGateway},
		{"validation", manager.Response{Failure: &manager.Failure{Code: manager.FailureValidation}}, http.StatusBadRequest},
	}
	for _, c := range cases {
		if got := managerResponseStatus(c.resp); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}
