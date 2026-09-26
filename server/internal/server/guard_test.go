package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardRejectsForeignHostOriginMissingTokenAndNonJSON(t *testing.T) {
	const token = "abc123"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := guardLocal("127.0.0.1:8787", token, next)

	foreign := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	foreign.Host = "evil.example:8787"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, foreign)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("foreign host = %d", rec.Code)
	}

	origin := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	origin.Host = "127.0.0.1:8787"
	origin.Header.Set("Origin", "https://evil.example")
	origin.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, origin)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("foreign origin = %d", rec.Code)
	}

	missing := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	missing.Host = "localhost:8787"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, missing)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing token = %d", rec.Code)
	}

	badJSON := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader("not-json"))
	badJSON.Host = "127.0.0.1:8787"
	badJSON.Header.Set("Authorization", "Bearer "+token)
	badJSON.Header.Set("Content-Type", "text/plain")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, badJSON)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("non-json = %d", rec.Code)
	}
}

func TestGuardAllowsEmptyPostAudioAndQueryToken(t *testing.T) {
	const token = "abc123"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := guardLocal("127.0.0.1:8787", token, next)

	empty := httptest.NewRequest(http.MethodPost, "/api/settings/recheck", nil)
	empty.Host = "127.0.0.1:8787"
	empty.Header.Set("Authorization", "Bearer "+token)
	empty.ContentLength = 0
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, empty)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("empty post = %d", rec.Code)
	}

	audio := httptest.NewRequest(http.MethodPost, "/api/manager/audio", strings.NewReader("audio"))
	audio.Host = "[::1]:8787"
	audio.Header.Set("Authorization", "Bearer "+token)
	audio.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, audio)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("audio = %d", rec.Code)
	}

	query := httptest.NewRequest(http.MethodPost, "/mcp?token="+token, strings.NewReader(`{}`))
	query.Host = "127.0.0.1:8787"
	query.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, query)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("query token = %d", rec.Code)
	}
}

func TestInjectLaunchToken(t *testing.T) {
	page := injectLaunchToken("<head></head>", "tok")
	if !strings.Contains(page, `<meta name="fleet-token" content="tok" />`) {
		t.Fatalf("page = %s", page)
	}
}

func TestGuardEdgeCases(t *testing.T) {
	const token = "abc123"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := guardLocal("127.0.0.1:8787", token, next)
	do := func(method, path, host, origin, contentType, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Host = host
		req.Header.Set("Authorization", "Bearer "+token)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	cases := []struct {
		name string
		rec  *httptest.ResponseRecorder
		want int
	}{
		{"origin null", do(http.MethodPost, "/api/tasks", "127.0.0.1:8787", "null", "application/json", "{}"), http.StatusForbidden},
		{"https origin", do(http.MethodGet, "/api/state", "127.0.0.1:8787", "https://127.0.0.1:8787", "", ""), http.StatusForbidden},
		{"wrong port", do(http.MethodGet, "/api/state", "127.0.0.1:9999", "", "", ""), http.StatusForbidden},
		{"ipv6 loopback", do(http.MethodGet, "/api/state", "[::1]:8787", "", "", ""), http.StatusNoContent},
		{"mixed case host", do(http.MethodGet, "/api/state", "LocalHost:8787", "", "", ""), http.StatusNoContent},
		{"put text", do(http.MethodPut, "/api/tasks", "127.0.0.1:8787", "", "text/plain", "x"), http.StatusUnsupportedMediaType},
		{"patch json with charset", do(http.MethodPatch, "/api/tasks", "127.0.0.1:8787", "", "application/json; charset=utf-8", "{}"), http.StatusNoContent},
		{"mcp needs token", nil, http.StatusUnauthorized},
	}
	noToken := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	noToken.Host = "127.0.0.1:8787"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, noToken)
	cases[len(cases)-1].rec = rec
	for _, c := range cases {
		if c.rec.Code != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.rec.Code, c.want)
		}
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q", got)
	}
}

func TestServeBackofficeBlocksTraversalAndSetsNoStore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html><head></head></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/../secret.txt", nil)
	rec := httptest.NewRecorder()
	serveBackoffice(rec, req, dir, "tok")
	if strings.Contains(rec.Body.String(), "top secret") {
		t.Fatal("path traversal reached a file outside the backoffice dir")
	}
	if !strings.Contains(rec.Body.String(), `content="tok"`) {
		t.Fatalf("token not injected: %s", rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", rec.Header().Get("Cache-Control"))
	}
}

func TestLoadLaunchTokenIsStablePerRuntimeRoot(t *testing.T) {
	root := t.TempDir()
	first, err := loadLaunchToken(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadLaunchToken(root)
	if err != nil || first != second || len(first) < 32 {
		t.Fatalf("first=%q second=%q err=%v", first, second, err)
	}
	if info, err := os.Stat(filepath.Join(root, "launch-token")); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("token file mode: %v %v", info, err)
	}
	a, _ := loadLaunchToken("")
	b, _ := loadLaunchToken("")
	if a == b {
		t.Fatal("empty runtime root must give a fresh token each time")
	}
}
