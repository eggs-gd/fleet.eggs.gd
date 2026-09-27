package audit

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
)

// Event is one append-only runtime audit record.
type Event struct {
	ID         int64          `json:"id,omitempty"`
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

// AppendEvent writes one audit event into the runtime database.
func AppendEvent(root string, event Event) error {
	if event.Time == "" {
		event.Time = time.Now().Format(time.RFC3339)
	}
	row := runtimedb.Event{
		Time: event.Time, Type: event.Type, TaskRef: event.TaskRef, TaskID: event.TaskID,
		Path: event.Path, Agent: event.Agent, Repository: event.Repository,
		Outcome: event.Outcome, Message: event.Message,
	}
	if len(event.Details) > 0 {
		raw, err := json.Marshal(event.Details)
		if err != nil {
			return err
		}
		row.Details = string(raw)
	}
	return runtimedb.InsertEvent(root, row)
}

// ReadEvents returns audit events in append order.
func ReadEvents(root string) ([]Event, error) {
	rows, err := runtimedb.Events(root)
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(rows))
	for _, row := range rows {
		event := Event{
			Time: row.Time, Type: row.Type, TaskRef: row.TaskRef, TaskID: row.TaskID,
			Path: row.Path, Agent: row.Agent, Repository: row.Repository,
			Outcome: row.Outcome, Message: row.Message,
		}
		if row.Details != "" {
			if err := json.Unmarshal([]byte(row.Details), &event.Details); err != nil {
				return nil, err
			}
		}
		events = append(events, event)
	}
	return events, nil
}

// EventLogText is the audit log rendered as NDJSON, for substring checks.
func EventLogText(root string) (string, error) {
	events, err := ReadEvents(root)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	for _, event := range events {
		raw, err := json.Marshal(event)
		if err != nil {
			return "", err
		}
		buf.Write(raw)
		buf.WriteByte('\n')
	}
	return buf.String(), nil
}

// ManagerEventsAfter returns task.* and project.* audit rows newer than
// after — everything manager_events surfaces.
func ManagerEventsAfter(root string, after int64) ([]Event, error) {
	rows, err := runtimedb.ManagerEventsAfter(root, after)
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(rows))
	for _, row := range rows {
		event := Event{ID: row.ID, Type: row.Type, TaskRef: row.TaskRef, Path: row.Path, Message: row.Message}
		if row.Details != "" {
			if err := json.Unmarshal([]byte(row.Details), &event.Details); err != nil {
				return nil, err
			}
		}
		out = append(out, event)
	}
	return out, nil
}
