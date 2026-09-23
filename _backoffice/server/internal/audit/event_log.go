// Package audit records append-only runtime events.
package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const eventLogRelativePath = "_registry/events.ndjson"

var eventLogMu sync.Mutex

type Event struct {
	Time       string         `json:"time"`
	Type       string         `json:"type"`
	TaskRef    string         `json:"task_ref,omitempty"`
	TaskID     string         `json:"task_id,omitempty"`
	Path       string         `json:"path,omitempty"`
	Agent      string         `json:"agent,omitempty"`
	Repository string         `json:"repository,omitempty"`
	Outcome    string         `json:"outcome,omitempty"`
	Message    string         `json:"message,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
}

func EventLogPath(root string) string {
	return filepath.Join(root, eventLogRelativePath)
}

func AppendEvent(root string, event Event) error {
	if event.Time == "" {
		event.Time = time.Now().Format(time.RFC3339)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	path := EventLogPath(root)
	eventLogMu.Lock()
	defer eventLogMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(data, '\n'))
	return err
}
