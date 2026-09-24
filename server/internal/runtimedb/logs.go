package runtimedb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const logFlushBytes = 64 * 1024

// Log is an open session transcript. Writes stay in memory until 64KB or Close,
// and ReadLog includes that tail so a reader sees text that is not flushed yet.
type Log struct {
	mu      sync.Mutex
	root    string
	claimID string
	key     string
	pending strings.Builder
	closed  bool
}

var (
	openMu   sync.Mutex
	openLogs = map[string]*Log{}
)

// OpenLog truncates or resumes the session log at logPath.
// logPath is either relative to root (`_registry/sessions/<claim>.log`) or absolute.
func OpenLog(root, logPath string, truncate bool) (*Log, error) {
	runtimeRoot, claimID, err := locateLog(root, logPath)
	if err != nil {
		return nil, err
	}
	db, err := Open(runtimeRoot)
	if err != nil {
		return nil, err
	}
	if truncate {
		if _, err := db.Exec(`
INSERT INTO session_logs(claim_id, body) VALUES(?, '')
ON CONFLICT(claim_id) DO UPDATE SET body = ''`, claimID); err != nil {
			return nil, fmt.Errorf("truncate session log %s: %w", claimID, err)
		}
	}
	log := &Log{root: runtimeRoot, claimID: claimID, key: logKey(runtimeRoot, claimID)}
	openMu.Lock()
	openLogs[log.key] = log
	openMu.Unlock()
	return log, nil
}

func (l *Log) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return 0, os.ErrClosed
	}
	l.pending.Write(p)
	if l.pending.Len() < logFlushBytes {
		return len(p), nil
	}
	if err := l.flushLocked(); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (l *Log) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	err := l.flushLocked()
	if err != nil {
		l.mu.Unlock()
		return err
	}
	l.closed = true
	l.mu.Unlock()
	openMu.Lock()
	if cur := openLogs[l.key]; cur == l {
		delete(openLogs, l.key)
	}
	openMu.Unlock()
	return nil
}

// AppendLog adds text to an open log, or to the stored row when the log is closed.
func AppendLog(root, logPath, text string) error {
	if text == "" {
		return nil
	}
	runtimeRoot, claimID, err := locateLog(root, logPath)
	if err != nil {
		return err
	}
	openMu.Lock()
	log := openLogs[logKey(runtimeRoot, claimID)]
	openMu.Unlock()
	if log != nil {
		_, err := log.Write([]byte(text))
		return err
	}
	return appendLogBody(runtimeRoot, claimID, text)
}

// ReadLog returns the session transcript, including an open writer's unflushed tail.
func ReadLog(root, logPath string) string {
	runtimeRoot, claimID, err := locateLog(root, logPath)
	if err != nil {
		return ""
	}
	openMu.Lock()
	log := openLogs[logKey(runtimeRoot, claimID)]
	openMu.Unlock()
	if log != nil {
		if text, ok := log.snapshot(); ok {
			return text
		}
	}
	body, err := logBody(runtimeRoot, claimID)
	if err != nil {
		return ""
	}
	return body
}

func (l *Log) snapshot() (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return "", false
	}
	stored, err := logBody(l.root, l.claimID)
	if err != nil {
		return l.pending.String(), true
	}
	return stored + l.pending.String(), true
}

func (l *Log) flushLocked() error {
	if l.pending.Len() == 0 {
		return nil
	}
	chunk := l.pending.String()
	l.pending.Reset()
	if err := appendLogBody(l.root, l.claimID, chunk); err != nil {
		l.pending.WriteString(chunk)
		return err
	}
	return nil
}

func appendLogBody(root, claimID, chunk string) error {
	db, err := Open(root)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
INSERT INTO session_logs(claim_id, body) VALUES(?, ?)
ON CONFLICT(claim_id) DO UPDATE SET body = session_logs.body || excluded.body`, claimID, chunk)
	if err != nil {
		return fmt.Errorf("append session log %s: %w", claimID, err)
	}
	return nil
}

func logBody(root, claimID string) (string, error) {
	db, err := Open(root)
	if err != nil {
		return "", err
	}
	var body string
	err = db.QueryRow(`SELECT body FROM session_logs WHERE claim_id = ?`, claimID).Scan(&body)
	if err != nil {
		return "", err
	}
	return body, nil
}

func logKey(root, claimID string) string {
	return root + "\x00" + claimID
}

func locateLog(root, logPath string) (string, string, error) {
	if strings.TrimSpace(logPath) == "" {
		return "", "", fmt.Errorf("session log path is empty")
	}
	if filepath.IsAbs(logPath) {
		return splitAbsLog(logPath)
	}
	if strings.TrimSpace(root) == "" {
		return "", "", fmt.Errorf("runtime root is empty")
	}
	claimID := claimFromBase(logPath)
	if claimID == "" {
		return "", "", fmt.Errorf("session log path %q has no claim id", logPath)
	}
	return root, claimID, nil
}

func splitAbsLog(path string) (string, string, error) {
	clean := filepath.Clean(path)
	claimID := claimFromBase(clean)
	dir := filepath.Dir(clean)
	if filepath.Base(dir) != "sessions" || filepath.Base(filepath.Dir(dir)) != "_registry" {
		return "", "", fmt.Errorf("session log path %q is not under _registry/sessions", path)
	}
	if claimID == "" {
		return "", "", fmt.Errorf("session log path %q has no claim id", path)
	}
	return filepath.Dir(filepath.Dir(dir)), claimID, nil
}

func claimFromBase(path string) string {
	base := filepath.Base(filepath.FromSlash(path))
	claimID := strings.TrimSuffix(base, filepath.Ext(base))
	if claimID == "." || claimID == string(filepath.Separator) {
		return ""
	}
	return strings.TrimSpace(claimID)
}
