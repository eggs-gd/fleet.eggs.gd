package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"html"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// loadLaunchToken returns the per-install token kept in the runtime root
// (~/.fleet), creating it once. Being stable, it survives restarts: open
// dashboard tabs and running Manager sessions keep working, and provider
// config files do not change on every launch. An empty root gives a fresh
// in-memory token, which is what tests want.
func loadLaunchToken(runtimeRoot string) (string, error) {
	root := strings.TrimSpace(runtimeRoot)
	if root == "" {
		return newLaunchToken()
	}
	path := filepath.Join(root, "launch-token")
	if data, err := os.ReadFile(path); err == nil {
		if token := strings.TrimSpace(string(data)); len(token) >= 32 {
			return token, nil
		}
	}
	token, err := newLaunchToken()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", err
	}
	return token, nil
}

func newLaunchToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func guardLocal(listenAddr, token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if !hostAllowed(r.Host, listenAddr) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if guardedPath(r.URL.Path) {
			if !originAllowed(r.Header.Get("Origin"), listenAddr) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
			if !launchTokenOK(r, token) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !bodyAllowed(r) {
				http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, bodyLimit(r.URL.Path))
		}
		next.ServeHTTP(w, r)
	})
}

// Request body limits. Task text and MCP calls are small; only the audio
// upload is big.
const (
	jsonBodyLimit  = 4 << 20
	audioBodyLimit = 32 << 20
)

func bodyLimit(path string) int64 {
	if path == "/api/manager/audio" {
		return audioBodyLimit
	}
	return jsonBodyLimit
}

func guardedPath(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/") || path == "/mcp" || strings.HasPrefix(path, "/mcp/")
}

func listenPort(addr string) string {
	_, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil || port == "" {
		return "8787"
	}
	return port
}

func hostAllowed(hostHeader, listenAddr string) bool {
	host, port, err := net.SplitHostPort(strings.TrimSpace(hostHeader))
	if err != nil {
		return false
	}
	if port != listenPort(listenAddr) {
		return false
	}
	switch strings.ToLower(host) {
	case "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}

func originAllowed(origin, listenAddr string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || parsed.Scheme != "http" {
		return false
	}
	return hostAllowed(parsed.Host, listenAddr)
}

func launchTokenOK(r *http.Request, token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(presentedToken(r)), []byte(token)) == 1 {
		return true
	}
	return false
}

func presentedToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if rest, ok := strings.CutPrefix(header, "Bearer "); ok {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}

func bodyAllowed(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return true
	}
	media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if r.URL.Path == "/api/manager/audio" {
		return media == "multipart/form-data"
	}
	if r.ContentLength == 0 {
		return true
	}
	return media == "application/json"
}

// serveBackoffice serves the dashboard from fsys. Any path that is not a file in
// fsys gets index.html, with the launch token injected, so client-side routes
// load. fs.FS refuses paths that leave its root, so a request cannot reach files
// outside the dashboard.
func serveBackoffice(w http.ResponseWriter, r *http.Request, fsys fs.FS, token string) {
	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if rel != "" && rel != "index.html" {
		if info, err := fs.Stat(fsys, rel); err == nil && !info.IsDir() {
			http.ServeFileFS(w, r, fsys, rel)
			return
		}
	}
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, injectLaunchToken(string(data), token))
}

func injectLaunchToken(page, token string) string {
	meta := `<meta name="fleet-token" content="` + html.EscapeString(token) + `" />`
	if strings.Contains(page, "</head>") {
		return strings.Replace(page, "</head>", meta+"\n</head>", 1)
	}
	return meta + page
}
