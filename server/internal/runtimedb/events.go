package runtimedb

import (
	"database/sql"
	"fmt"
)

// Event is one audit row. Details is JSON text, or empty when the event has none.
type Event struct {
	Time       string
	Type       string
	TaskRef    string
	TaskID     string
	Path       string
	Agent      string
	Repository string
	Outcome    string
	Message    string
	Details    string
}

// InsertEvent appends one audit event.
func InsertEvent(root string, event Event) error {
	db, err := Open(root)
	if err != nil {
		return err
	}
	var details any
	if event.Details != "" {
		details = event.Details
	}
	_, err = db.Exec(`
INSERT INTO events(time, type, task_ref, task_id, path, agent, repository, outcome, message, details)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.Time, event.Type, event.TaskRef, event.TaskID, event.Path,
		event.Agent, event.Repository, event.Outcome, event.Message, details)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

// Events returns audit events in append order.
func Events(root string) ([]Event, error) {
	db, err := Open(root)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`
SELECT time, type, task_ref, task_id, path, agent, repository, outcome, message, details
FROM events ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var event Event
		var details sql.NullString
		if err := rows.Scan(&event.Time, &event.Type, &event.TaskRef, &event.TaskID, &event.Path, &event.Agent, &event.Repository, &event.Outcome, &event.Message, &details); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		event.Details = details.String
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return events, nil
}

// TaskEvent is one task.* audit row with its cursor id.
type TaskEvent struct {
	ID      int64
	Type    string
	TaskRef string
	Message string
}

// TaskEventsAfter returns task.* events whose row id is greater than after.
func TaskEventsAfter(root string, after int64) ([]TaskEvent, error) {
	db, err := Open(root)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`
SELECT id, type, task_ref, message FROM events
WHERE id > ? AND type LIKE 'task.%'
ORDER BY id`, after)
	if err != nil {
		return nil, fmt.Errorf("list task events: %w", err)
	}
	defer rows.Close()
	var events []TaskEvent
	for rows.Next() {
		var event TaskEvent
		if err := rows.Scan(&event.ID, &event.Type, &event.TaskRef, &event.Message); err != nil {
			return nil, fmt.Errorf("scan task event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list task events: %w", err)
	}
	return events, nil
}
