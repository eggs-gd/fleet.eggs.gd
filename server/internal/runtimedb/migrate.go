package runtimedb

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const legacyImportedKey = "legacy_imported"

// ImportLegacy copies ~/.fleet/_registry text files into the database once,
// then deletes them. A crash before commit leaves the files in place. A crash
// after commit only deletes leftovers on the next call.
func ImportLegacy(root string) error {
	db, err := Open(root)
	if err != nil {
		return err
	}
	imported, err := metaValue(db, legacyImportedKey)
	if err != nil {
		return err
	}
	if imported == "1" {
		return removeLegacyFiles(root)
	}
	if !legacyPresent(root) {
		return setMeta(db, legacyImportedKey, "1")
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin legacy import: %w", err)
	}
	if err := importLegacy(tx, root); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO meta(key, value) VALUES(?, '1')
ON CONFLICT(key) DO UPDATE SET value = '1'`, legacyImportedKey); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("mark legacy import: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy import: %w", err)
	}
	return removeLegacyFiles(root)
}

func importLegacy(tx *sql.Tx, root string) error {
	reg := filepath.Join(root, "_registry")
	if err := importEvents(tx, filepath.Join(reg, "events.ndjson")); err != nil {
		return err
	}
	if err := importRelay(tx, filepath.Join(reg, "manager-relay.log")); err != nil {
		return err
	}
	return importSessionDir(tx, filepath.Join(reg, "sessions"))
}

func importEvents(tx *sql.Tx, path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	stmt, err := tx.Prepare(`
INSERT INTO events(time, type, task_ref, task_id, path, agent, repository, outcome, message, details)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare events import: %w", err)
	}
	defer stmt.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event Event
		var raw struct {
			Time       string          `json:"time"`
			Type       string          `json:"type"`
			TaskRef    string          `json:"task_ref"`
			TaskID     string          `json:"task_id"`
			Path       string          `json:"path"`
			Agent      string          `json:"agent"`
			Repository string          `json:"repository"`
			Outcome    string          `json:"outcome"`
			Message    string          `json:"message"`
			Details    json.RawMessage `json:"details"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			event = Event{Type: "legacy_unparsed", Message: line}
		} else {
			event = Event{
				Time: raw.Time, Type: raw.Type, TaskRef: raw.TaskRef, TaskID: raw.TaskID,
				Path: raw.Path, Agent: raw.Agent, Repository: raw.Repository,
				Outcome: raw.Outcome, Message: raw.Message,
			}
			if len(raw.Details) > 0 && string(raw.Details) != "null" {
				event.Details = string(raw.Details)
			}
		}
		var details any
		if event.Details != "" {
			details = event.Details
		}
		if _, err := stmt.Exec(event.Time, event.Type, event.TaskRef, event.TaskID, event.Path, event.Agent, event.Repository, event.Outcome, event.Message, details); err != nil {
			return fmt.Errorf("import event: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func importRelay(tx *sql.Tx, path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(body) == 0 {
		return nil
	}
	_, err = tx.Exec(`
INSERT INTO relay_log(id, body) VALUES(1, ?)
ON CONFLICT(id) DO UPDATE SET body = relay_log.body || excluded.body`, string(body))
	if err != nil {
		return fmt.Errorf("import relay log: %w", err)
	}
	return nil
}

func importSessionDir(tx *sql.Tx, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".json":
			if err := importSessionFile(tx, entry.Name(), body); err != nil {
				return err
			}
		case ".log":
			claimID := claimFromBase(entry.Name())
			if _, err := tx.Exec(`
INSERT INTO session_logs(claim_id, body) VALUES(?, ?)
ON CONFLICT(claim_id) DO UPDATE SET body = excluded.body`, claimID, string(body)); err != nil {
				return fmt.Errorf("import session log %s: %w", entry.Name(), err)
			}
		default:
			if _, err := tx.Exec(`
INSERT INTO extra_files(name, body) VALUES(?, ?)
ON CONFLICT(name) DO UPDATE SET body = excluded.body`, entry.Name(), string(body)); err != nil {
				return fmt.Errorf("import extra file %s: %w", entry.Name(), err)
			}
		}
	}
	return nil
}

func importSessionFile(tx *sql.Tx, name string, body []byte) error {
	var head struct {
		ClaimID   string `json:"claim_id"`
		TaskRef   string `json:"task_ref"`
		TaskID    string `json:"task_id"`
		TaskPath  string `json:"task_path"`
		ProjectID string `json:"project_id"`
		Agent     string `json:"agent"`
		ClaimedAt string `json:"claimed_at"`
	}
	_ = json.Unmarshal(body, &head)
	if head.ClaimID == "" {
		head.ClaimID = claimFromBase(name)
	}
	_, err := tx.Exec(`
INSERT INTO sessions(claim_id, task_ref, task_id, task_path, project_id, agent, claimed_at, body)
VALUES(?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(claim_id) DO UPDATE SET
  task_ref=excluded.task_ref,
  task_id=excluded.task_id,
  task_path=excluded.task_path,
  project_id=excluded.project_id,
  agent=excluded.agent,
  claimed_at=excluded.claimed_at,
  body=excluded.body`,
		head.ClaimID, head.TaskRef, head.TaskID, head.TaskPath, head.ProjectID, head.Agent, head.ClaimedAt, string(body))
	if err != nil {
		return fmt.Errorf("import session %s: %w", name, err)
	}
	return nil
}

func legacyPresent(root string) bool {
	reg := filepath.Join(root, "_registry")
	if fileExists(filepath.Join(reg, "events.ndjson")) || fileExists(filepath.Join(reg, "manager-relay.log")) {
		return true
	}
	entries, err := os.ReadDir(filepath.Join(reg, "sessions"))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return true
		}
	}
	return false
}

func removeLegacyFiles(root string) error {
	reg := filepath.Join(root, "_registry")
	for _, name := range []string{"events.ndjson", "manager-relay.log"} {
		if err := os.Remove(filepath.Join(reg, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	sessions := filepath.Join(reg, "sessions")
	entries, err := os.ReadDir(sessions)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.Remove(reg)
			return nil
		}
		return fmt.Errorf("read %s: %w", sessions, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(sessions, entry.Name())); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", entry.Name(), err)
		}
	}
	_ = os.Remove(sessions)
	_ = os.Remove(reg)
	return nil
}

func metaValue(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read meta %s: %w", key, err)
	}
	return value, nil
}

func setMeta(db *sql.DB, key, value string) error {
	_, err := db.Exec(`
INSERT INTO meta(key, value) VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("write meta %s: %w", key, err)
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
