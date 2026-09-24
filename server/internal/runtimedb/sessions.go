package runtimedb

import "fmt"

// Session is one persisted runtime-session record. Body is the full JSON document.
type Session struct {
	ClaimID   string
	TaskRef   string
	TaskID    string
	TaskPath  string
	ProjectID string
	Agent     string
	ClaimedAt string
	Body      string
}

// UpsertSession writes one session document, replacing any previous row for the claim.
func UpsertSession(root string, session Session) error {
	if session.ClaimID == "" {
		return nil
	}
	db, err := Open(root)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
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
		session.ClaimID, session.TaskRef, session.TaskID, session.TaskPath,
		session.ProjectID, session.Agent, session.ClaimedAt, session.Body)
	if err != nil {
		return fmt.Errorf("upsert session %s: %w", session.ClaimID, err)
	}
	return nil
}

// SessionBodies returns every stored session document, ordered by claim id.
func SessionBodies(root string) ([]string, error) {
	db, err := Open(root)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT body FROM sessions ORDER BY claim_id`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	var bodies []string
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		bodies = append(bodies, body)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return bodies, nil
}
