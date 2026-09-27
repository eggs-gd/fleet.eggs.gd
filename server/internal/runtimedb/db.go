// Package runtimedb stores App runtime state in <runtime-root>/runtime.db:
// session records, session logs, the audit event log, and the manager relay log.
package runtimedb

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

const FileName = "runtime.db"

const schema = `
CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
  claim_id TEXT PRIMARY KEY,
  task_ref TEXT,
  task_id TEXT,
  task_path TEXT,
  project_id TEXT,
  agent TEXT,
  claimed_at TEXT,
  body TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS session_logs (
  claim_id TEXT PRIMARY KEY,
  body TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS events (
  id INTEGER PRIMARY KEY,
  time TEXT,
  type TEXT,
  task_ref TEXT,
  task_id TEXT,
  path TEXT,
  agent TEXT,
  repository TEXT,
  outcome TEXT,
  message TEXT,
  details TEXT
);
CREATE INDEX IF NOT EXISTS events_type ON events(type);
CREATE INDEX IF NOT EXISTS events_task_ref ON events(task_ref);
CREATE TABLE IF NOT EXISTS relay_log (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  body TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS extra_files (
  name TEXT PRIMARY KEY,
  body TEXT NOT NULL
);
`

var cache sync.Map // runtime root -> *sql.DB

// Path is the sqlite file for a runtime root (~/.fleet in production).
func Path(root string) string {
	return filepath.Join(root, FileName)
}

// Open returns the shared database for root, creating the file and schema if needed.
func Open(root string) (*sql.DB, error) {
	if root == "" {
		return nil, fmt.Errorf("runtime root is empty")
	}
	if v, ok := cache.Load(root); ok {
		return v.(*sql.DB), nil
	}
	// The database holds agent session logs, so only the owner may read it.
	// SQLite gives the -wal and -shm files the mode of the main file.
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create runtime root: %w", err)
	}
	if err := ensurePrivateFile(Path(root)); err != nil {
		return nil, err
	}
	dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(Path(root)), RawQuery: "mode=rwc"}
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open runtime db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set wal: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate runtime schema: %w", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := tightenIfPresent(Path(root) + suffix); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	actual, loaded := cache.LoadOrStore(root, db)
	if loaded {
		_ = db.Close()
		return actual.(*sql.DB), nil
	}
	return db, nil
}

// ensurePrivateFile creates path with mode 0600 when it is missing and
// tightens it when it exists.
func ensurePrivateFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("create runtime db: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func tightenIfPresent(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.Chmod(path, 0o600)
}

// CloseAll closes every open runtime database, which checkpoints its WAL. Call
// it once, on the way out.
func CloseAll() {
	cache.Range(func(key, value any) bool {
		_ = value.(*sql.DB).Close()
		cache.Delete(key)
		return true
	})
}
