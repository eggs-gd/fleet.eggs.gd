package executionstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Session is the durable runtime-session record persisted under
// _registry/sessions. Concrete provider detail fields stay on the
// composition-root RuntimeSession projection; this package owns the
// session/orphan maps and persistence helpers.
type Session struct {
	ClaimID   string `json:"claim_id"`
	TaskRef   string `json:"task_ref"`
	TaskID    string `json:"task_id"`
	TaskPath  string `json:"task_path"`
	ProjectID string `json:"project_id"`
	Agent     string `json:"agent"`
	Status    string `json:"status"`
}

// Orphan is a doing-task lock without an attached live session.
type Orphan struct {
	TaskRef    string `json:"task_ref"`
	TaskID     string `json:"task_id"`
	TaskPath   string `json:"task_path"`
	ClaimID    string `json:"claim_id,omitempty"`
	Assignee   string `json:"assignee"`
	DetectedAt string `json:"detected_at"`
	Reason     string `json:"reason"`
}

// Registry owns session and orphan maps separately from board task state.
type Registry struct {
	mu       sync.RWMutex
	root     string
	sessions map[string]Session
	orphans  map[string]Orphan
	updated  time.Time
}

// NewRegistry builds an empty session/orphan registry rooted at coreRoot.
func NewRegistry(root string) *Registry {
	return &Registry{
		root:     root,
		sessions: map[string]Session{},
		orphans:  map[string]Orphan{},
	}
}

// Persist writes one session record to _registry/sessions.
func Persist(root, claimID string, value any) error {
	if claimID == "" {
		return nil
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(root, "_registry", "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, claimID+".json"), append(data, '\n'), 0o644)
}

// UpsertSession stores a session identity record.
func (r *Registry) UpsertSession(session Session) {
	if r == nil || session.ClaimID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ClaimID] = session
	r.updated = time.Now().UTC().Truncate(time.Second)
	_ = Persist(r.root, session.ClaimID, session)
}

// RemoveSession drops a session by claim id.
func (r *Registry) RemoveSession(claimID string) {
	if r == nil || claimID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, claimID)
	r.updated = time.Now().UTC().Truncate(time.Second)
}

// UpsertOrphan stores an orphan lock keyed by task path/id.
func (r *Registry) UpsertOrphan(orphan Orphan) {
	if r == nil {
		return
	}
	key := orphan.TaskPath
	if key == "" {
		key = orphan.TaskID
	}
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orphans[key] = orphan
	r.updated = time.Now().UTC().Truncate(time.Second)
}

// SessionCount returns active stored sessions (test/helper).
func (r *Registry) SessionCount() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}

// OrphanCount returns stored orphans (test/helper).
func (r *Registry) OrphanCount() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.orphans)
}
