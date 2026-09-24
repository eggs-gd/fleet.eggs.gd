package execution

import (
	"encoding/json"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func persistRuntimeSession(root string, session RuntimeSession) error {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(session.ClaimID) == "" {
		return nil
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return runtimedb.UpsertSession(root, runtimedb.Session{
		ClaimID:   session.ClaimID,
		TaskRef:   session.TaskRef,
		TaskID:    session.TaskID,
		TaskPath:  session.TaskPath,
		ProjectID: session.ProjectID,
		Agent:     session.Agent,
		ClaimedAt: session.ClaimedAt,
		Body:      string(data),
	})
}

func PersistRuntimeSession(root string, session RuntimeSession) error {
	return persistRuntimeSession(root, session)
}

// LoadRuntimeSessionRecords reads persisted session records for restart
// reconciliation and reuse lookups.
func LoadRuntimeSessionRecords(root string) ([]RuntimeSession, error) {
	return loadRuntimeSessionRecords(root)
}

// PreviousSessionForTask finds the most recent persisted runtime session
// belonging to the given task.
func PreviousSessionForTask(root string, task tasklifecycle.Task) (RuntimeSession, bool) {
	return previousSessionForTask(root, task)
}

// loadRuntimeSessionRecords reads persisted session records for restart
// reconciliation and reuse lookups. A single unreadable or schema-incompatible
// record (e.g. an older persisted shape from before a session field changed
// type) must not take down the whole daemon on startup: skip and continue
// instead of aborting the read for every other record.
func loadRuntimeSessionRecords(root string) ([]RuntimeSession, error) {
	if strings.TrimSpace(root) == "" {
		return []RuntimeSession{}, nil
	}
	bodies, err := runtimedb.SessionBodies(root)
	if err != nil {
		return nil, err
	}
	sessions := []RuntimeSession{}
	for _, body := range bodies {
		var session RuntimeSession
		if err := json.Unmarshal([]byte(body), &session); err != nil {
			continue
		}
		if session.ClaimID == "" {
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

// previousSessionForTask finds the most recent persisted runtime session
// belonging to the given task (matched by path/id/ref, same identity rule as
// sameTaskIdentity), regardless of whether it is still active. Persisted
// session records are never deleted, so this covers sessions that already
// finished, not just the live in-memory set. This is the basis for CORE-77
// relaunch reuse/supersede linking: a task can be relaunched long after its
// previous session ended.
func previousSessionForTask(root string, task tasklifecycle.Task) (RuntimeSession, bool) {
	sessions, err := loadRuntimeSessionRecords(root)
	if err != nil {
		return RuntimeSession{}, false
	}
	var best RuntimeSession
	found := false
	for _, session := range sessions {
		if !sameTaskIdentity(session.TaskPath, task.RelativePath, session.TaskID, task.ID, session.TaskRef, task.Ref) {
			continue
		}
		if !found || session.ClaimedAt > best.ClaimedAt {
			best = session
			found = true
		}
	}
	return best, found
}

func sameTaskIdentity(leftPath, rightPath, leftID, rightID, leftRef, rightRef string) bool {
	return (leftPath != "" && leftPath == rightPath) || (leftID != "" && leftID == rightID) || (leftRef != "" && leftRef == rightRef)
}

// previousSessionForAgentProject finds the most recent persisted session for
// the same (project, agent) pair, regardless of which task it ran.
//
// This is the 1-1-1 session model (_docs/AGENT_SESSION_REUSE.md): Core keeps
// exactly one live-or-stopped session per agent per project and feeds new
// tasks into it instead of always starting fresh. It is deliberately scoped
// to a single persistent session per (project, agent) — running N sessions
// per agent per project is out of scope here and needs its own git-flow /
// worktree coordination rules first (concurrent agents in one repository
// would need branch or worktree isolation Core does not define yet); do not
// extend this lookup to return multiple candidates without that design.
func previousSessionForAgentProject(root string, projectID string, agent string) (RuntimeSession, bool) {
	projectID = strings.TrimSpace(projectID)
	agent = strings.TrimSpace(agent)
	if projectID == "" || agent == "" {
		return RuntimeSession{}, false
	}
	sessions, err := loadRuntimeSessionRecords(root)
	if err != nil {
		return RuntimeSession{}, false
	}
	var best RuntimeSession
	found := false
	for _, session := range sessions {
		if session.ProjectID != projectID || session.Agent != agent {
			continue
		}
		if !found || session.ClaimedAt > best.ClaimedAt {
			best = session
			found = true
		}
	}
	return best, found
}
