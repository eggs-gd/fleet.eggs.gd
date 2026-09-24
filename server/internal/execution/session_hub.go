package execution

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// sessionHub is execution's own black-box store of live runtime sessions and
// orphaned-doing-task records. listTasks/taskByPath read the board's current
// task list so orphan/session projections stay accurate without the hub
// owning tasks itself.
type sessionHub struct {
	mu   sync.RWMutex
	root string

	sessions map[string]RuntimeSession
	orphans  map[string]OrphanedTask

	listTasks  func() []tasklifecycle.Task
	taskByPath func(string) (tasklifecycle.Task, bool)
}

func newSessionHub(root string, listTasks func() []tasklifecycle.Task, taskByPath func(string) (tasklifecycle.Task, bool)) *sessionHub {
	return &sessionHub{
		root:       root,
		sessions:   map[string]RuntimeSession{},
		orphans:    map[string]OrphanedTask{},
		listTasks:  listTasks,
		taskByPath: taskByPath,
	}
}

func (hub *sessionHub) tasks() []tasklifecycle.Task {
	if hub.listTasks == nil {
		return nil
	}
	return hub.listTasks()
}

func (hub *sessionHub) UpsertSession(session RuntimeSession) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if session.ClaimID == "" {
		return
	}
	if session.VisibilityMode == "" {
		session.VisibilityMode = VisibilityModeForSession(session)
	}
	if session.Capabilities.Provider == "" {
		session.Capabilities = ProviderCapabilityFlags(session.Agent, session.Backend)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if existing, ok := hub.sessions[session.ClaimID]; ok {
		if session.LastEventAt == "" {
			session.LastEventAt = existing.LastEventAt
		}
		if session.LastOutputAt == "" {
			session.LastOutputAt = existing.LastOutputAt
		}
		if session.LastStatusAt == "" {
			session.LastStatusAt = existing.LastStatusAt
		}
		if session.Result == nil && existing.Result != nil {
			session.Result = existing.Result
		}
		if session.LastEvent != "" && session.LastEvent != existing.LastEvent {
			session.LastEventAt = now
		}
		if session.LastMessage != "" && session.LastMessage != existing.LastMessage {
			session.LastOutputAt = now
		}
		if firstNonEmpty(session.ExecutionStatus, session.Status) != firstNonEmpty(existing.ExecutionStatus, existing.Status) {
			session.LastStatusAt = now
		}
	} else {
		if session.LastEvent != "" && session.LastEventAt == "" {
			session.LastEventAt = now
		}
		if session.LastMessage != "" && session.LastOutputAt == "" {
			session.LastOutputAt = now
		}
		if firstNonEmpty(session.ExecutionStatus, session.Status) != "" && session.LastStatusAt == "" {
			session.LastStatusAt = now
		}
	}
	if session.ExecutionStatus != "resumable" {
		session.LastSeenAt = now
	}
	// Runtime-only annotation; never persist control-channel liveness.
	session.ProviderControllable = false
	hub.sessions[session.ClaimID] = session
	_ = persistRuntimeSession(hub.root, session)
	hub.removeOrphanForSessionLocked(session)
}

func (hub *sessionHub) TryReserveSession(session RuntimeSession) (RuntimeSession, string, bool) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if session.ClaimID == "" {
		return RuntimeSession{}, "", false
	}
	if conflict, scope, ok := hub.activeSessionConflictLocked(session.ProjectID, session.Repository, session.Agent); ok {
		return conflict, scope, false
	}
	if orphan, ok := hub.activeOrphanConflictLocked(session.Agent, session); ok {
		return orphanRuntimeSession(orphan), "assignee_orphan", false
	}
	if session.Status == "" {
		session.Status = "starting"
	}
	if session.ExecutionStatus == "" {
		session.ExecutionStatus = "starting"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	session.LastSeenAt = now
	session.LastStatusAt = firstNonEmpty(session.LastStatusAt, now)
	hub.sessions[session.ClaimID] = session
	_ = persistRuntimeSession(hub.root, session)
	hub.removeOrphanForSessionLocked(session)
	return RuntimeSession{}, "", true
}

func (hub *sessionHub) RemoveSession(claimID string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	session, ok := hub.sessions[claimID]
	delete(hub.sessions, claimID)
	if ok {
		hub.removeOrphanForSessionLocked(session)
	}
}

func (hub *sessionHub) Session(claimID string) (RuntimeSession, bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	session, ok := hub.sessions[claimID]
	return session, ok
}

func (hub *sessionHub) OrphanByClaimID(claimID string) (OrphanedTask, bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	claimID = strings.TrimSpace(claimID)
	if claimID == "" {
		return OrphanedTask{}, false
	}
	for _, orphan := range hub.orphans {
		if orphan.ClaimID == claimID {
			return orphan, true
		}
	}
	return OrphanedTask{}, false
}

func (hub *sessionHub) ClearOrphanForSession(session RuntimeSession) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.removeOrphanForSessionLocked(session)
	if session.TaskPath != "" || session.TaskID != "" || session.TaskRef != "" {
		hub.removeOrphanForTaskLocked(tasklifecycle.Task{
			RelativePath: session.TaskPath,
			ID:           session.TaskID,
			Ref:          session.TaskRef,
		})
	}
}

func (hub *sessionHub) UpsertOrphan(orphan OrphanedTask) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	key := orphan.TaskPath
	if key == "" {
		key = orphan.TaskID
	}
	if key == "" {
		return
	}
	if task, ok := hub.taskForOrphanLocked(orphan); ok && !hub.taskIsCurrentOrphanLocked(task) && !leftoverOrphanUnresolved(orphan) {
		delete(hub.orphans, key)
		return
	}
	if existing, ok := hub.orphans[key]; ok {
		orphan = mergeOrphanIdentity(existing, orphan)
	}
	hub.orphans[key] = orphan
}

func (hub *sessionHub) CurrentOrphanForTask(task tasklifecycle.Task) (OrphanedTask, bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	orphan, ok := hub.orphanForTaskLocked(task)
	if !ok || !hub.taskIsCurrentOrphanLocked(task) {
		return OrphanedTask{}, false
	}
	return orphan, true
}

func (hub *sessionHub) ActiveSessionConflict(projectID string, repository string, agent string) (RuntimeSession, string, bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	return hub.activeSessionConflictLocked(projectID, repository, agent)
}

func (hub *sessionHub) ActiveOrphanConflict(assignee string, session RuntimeSession) (RuntimeSession, bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	orphan, ok := hub.activeOrphanConflictLocked(assignee, session)
	if !ok {
		return RuntimeSession{}, false
	}
	return orphanRuntimeSession(orphan), true
}

func (hub *sessionHub) ActiveSessionForTask(task tasklifecycle.Task) (RuntimeSession, bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	return hub.activeSessionForTaskLocked(task)
}

func (hub *sessionHub) activeSessionForTaskLocked(task tasklifecycle.Task) (RuntimeSession, bool) {
	for _, session := range hub.sessions {
		if !session.IsActive() {
			continue
		}
		if sameTaskIdentity(session.TaskPath, task.RelativePath, session.TaskID, task.ID, session.TaskRef, task.Ref) {
			return session, true
		}
	}
	return RuntimeSession{}, false
}

func (hub *sessionHub) activeSessionConflictLocked(projectID string, repository string, agent string) (RuntimeSession, string, bool) {
	for _, session := range hub.sessions {
		if !session.IsActive() {
			continue
		}
		if projectID != "" && session.ProjectID == projectID {
			return session, "project", true
		}
		if repository != "" && session.Repository == repository {
			return session, "repository", true
		}
		if agent != "" && agent != "unassigned" && session.Agent == agent {
			return session, "assignee", true
		}
	}
	return RuntimeSession{}, "", false
}

func (hub *sessionHub) activeOrphanConflictLocked(assignee string, session RuntimeSession) (OrphanedTask, bool) {
	if assignee == "" || assignee == "unassigned" {
		return OrphanedTask{}, false
	}
	for _, orphan := range hub.orphans {
		if orphan.Assignee != assignee {
			continue
		}
		if sameTaskIdentity(session.TaskPath, orphan.TaskPath, session.TaskID, orphan.TaskID, session.TaskRef, orphan.TaskRef) {
			continue
		}
		task, ok := hub.taskForOrphanLocked(orphan)
		if !ok || !hub.taskIsCurrentOrphanLocked(task) {
			continue
		}
		return orphan, true
	}
	return OrphanedTask{}, false
}

func (hub *sessionHub) taskIsCurrentOrphanLocked(task tasklifecycle.Task) bool {
	if task.Status != "doing" {
		return false
	}
	return !hub.hasActiveSessionForTaskLocked(task)
}

func leftoverOrphanUnresolved(orphan OrphanedTask) bool {
	session := RuntimeSession{Status: orphan.ExecutionState, ExecutionStatus: orphan.ExecutionState}
	if session.IsActive() {
		return true
	}
	switch orphan.ExecutionState {
	case "dead", "orphaned", "unknown":
		return true
	default:
		return false
	}
}

func (hub *sessionHub) orphanIsVisibleLocked(orphan OrphanedTask) bool {
	task, ok := hub.taskForOrphanLocked(orphan)
	if ok && hub.taskIsCurrentOrphanLocked(task) {
		return true
	}
	return leftoverOrphanUnresolved(orphan)
}

func (hub *sessionHub) hasActiveSessionForTaskLocked(task tasklifecycle.Task) bool {
	for _, session := range hub.sessions {
		if !session.IsActive() {
			continue
		}
		if task.RelativePath != "" && session.TaskPath == task.RelativePath {
			return true
		}
		if task.ID != "" && session.TaskID == task.ID {
			return true
		}
		if task.Ref != "" && session.TaskRef == task.Ref {
			return true
		}
	}
	return false
}

func (hub *sessionHub) removeOrphanForTaskLocked(task tasklifecycle.Task) {
	for _, key := range orphanTaskKeys(task) {
		delete(hub.orphans, key)
	}
	for key, orphan := range hub.orphans {
		if sameTaskIdentity(task.RelativePath, orphan.TaskPath, task.ID, orphan.TaskID, task.Ref, orphan.TaskRef) {
			delete(hub.orphans, key)
		}
	}
}

func (hub *sessionHub) removeOrphanForSessionLocked(session RuntimeSession) {
	for _, key := range orphanSessionKeys(session) {
		delete(hub.orphans, key)
	}
	for key, orphan := range hub.orphans {
		if sameTaskIdentity(session.TaskPath, orphan.TaskPath, session.TaskID, orphan.TaskID, session.TaskRef, orphan.TaskRef) {
			delete(hub.orphans, key)
		}
	}
}

func (hub *sessionHub) taskForOrphanLocked(orphan OrphanedTask) (tasklifecycle.Task, bool) {
	for _, task := range hub.tasks() {
		if sameTaskIdentity(task.RelativePath, orphan.TaskPath, task.ID, orphan.TaskID, task.Ref, orphan.TaskRef) {
			return task, true
		}
	}
	return tasklifecycle.Task{}, false
}

func (hub *sessionHub) orphanForTaskLocked(task tasklifecycle.Task) (OrphanedTask, bool) {
	for _, orphan := range hub.orphans {
		if sameTaskIdentity(task.RelativePath, orphan.TaskPath, task.ID, orphan.TaskID, task.Ref, orphan.TaskRef) {
			return orphan, true
		}
	}
	return OrphanedTask{}, false
}

func orphanTaskKeys(task tasklifecycle.Task) []string {
	return compactStrings(task.RelativePath, task.Path, task.ID, task.Ref)
}

func orphanSessionKeys(session RuntimeSession) []string {
	return compactStrings(session.TaskPath, session.TaskID, session.TaskRef)
}

func orphanRuntimeSession(orphan OrphanedTask) RuntimeSession {
	return RuntimeSession{
		ClaimID:        firstNonEmpty(orphan.ClaimID, "orphan:"+firstNonEmpty(orphan.TaskRef, orphan.TaskID, orphan.TaskPath)),
		TaskRef:        orphan.TaskRef,
		TaskID:         orphan.TaskID,
		TaskPath:       orphan.TaskPath,
		ProjectID:      orphan.ProjectID,
		Repository:     orphan.Repository,
		Agent:          orphan.Assignee,
		Backend:        orphan.Backend,
		VisibilityMode: orphan.VisibilityMode,
		ProcessID:      orphan.ProcessID,
		CodexSessionDetails: CodexSessionDetails{
			CodexThreadID: orphan.ThreadID,
			CodexTurnID:   orphan.TurnID,
		},
		CursorSessionDetails: CursorSessionDetails{
			CursorChatID: orphan.SessionID,
		},
		LogPath:         orphan.LogPath,
		LastEvent:       orphan.LastEvent,
		LastSeenAt:      orphan.LastActivityAt,
		ClaimedAt:       orphan.DetectedAt,
		Status:          "orphaned",
		ExecutionStatus: firstNonEmpty(orphan.ExecutionState, "orphaned"),
		ErrorMessage:    orphan.Reason,
	}
}

func mergeOrphanIdentity(existing OrphanedTask, next OrphanedTask) OrphanedTask {
	next.ClaimID = firstNonEmpty(next.ClaimID, existing.ClaimID)
	next.ExecutionState = firstNonEmpty(next.ExecutionState, existing.ExecutionState)
	next.Provider = firstNonEmpty(next.Provider, existing.Provider)
	next.Backend = firstNonEmpty(next.Backend, existing.Backend)
	next.VisibilityMode = firstNonEmpty(next.VisibilityMode, existing.VisibilityMode)
	if next.ProcessID == 0 {
		next.ProcessID = existing.ProcessID
	}
	next.ThreadID = firstNonEmpty(next.ThreadID, existing.ThreadID)
	next.SessionID = firstNonEmpty(next.SessionID, existing.SessionID)
	next.TurnID = firstNonEmpty(next.TurnID, existing.TurnID)
	next.LogPath = firstNonEmpty(next.LogPath, existing.LogPath)
	next.LastEvent = firstNonEmpty(next.LastEvent, existing.LastEvent)
	next.LastActivityAt = firstNonEmpty(next.LastActivityAt, existing.LastActivityAt)
	if next.Capabilities.Provider == "" {
		next.Capabilities = existing.Capabilities
	}
	if next.ProviderError == nil {
		next.ProviderError = existing.ProviderError
	}
	if existing.ExecutionState != "" && next.ExecutionState == existing.ExecutionState {
		next.Reason = firstNonEmpty(existing.Reason, next.Reason)
		next.BlockingReason = firstNonEmpty(existing.BlockingReason, next.BlockingReason)
	}
	return next
}

// orphanForTaskSession builds an OrphanedTask record from a task/session
// pair that restart recovery or startup-orphan detection has classified as
// unresolved.
func orphanForTaskSession(task tasklifecycle.Task, session RuntimeSession, state string, reason string) OrphanedTask {
	lastActivityAt := firstNonEmpty(session.LastSeenAt, session.LastEventAt, session.LastOutputAt, session.ExitedAt, session.StartedAt, session.ClaimedAt)
	detectedAt := firstNonEmpty(time.Now().Format(time.RFC3339), lastActivityAt)
	if session.ClaimedAt != "" && session.ClaimID == "" {
		detectedAt = session.ClaimedAt
	}
	provider := firstNonEmpty(session.Agent, task.Assignee)
	orphan := OrphanedTask{
		TaskRef:        firstNonEmpty(session.TaskRef, task.Ref),
		TaskID:         firstNonEmpty(session.TaskID, task.ID),
		TaskPath:       firstNonEmpty(session.TaskPath, task.RelativePath),
		ProjectID:      firstNonEmpty(session.ProjectID, task.ProjectID),
		Repository:     firstNonEmpty(session.Repository, FirstRepository(task)),
		Assignee:       provider,
		ClaimID:        session.ClaimID,
		ExecutionState: state,
		Provider:       provider,
		Backend:        session.Backend,
		VisibilityMode: VisibilityModeForSession(session),
		ProcessID:      session.ProcessID,
		ThreadID:       session.ProviderThreadID(),
		SessionID:      session.ProviderSessionID(),
		TurnID:         session.CodexTurnID,
		LogPath:        session.LogPath,
		LastEvent:      session.LastEvent,
		LastActivityAt: lastActivityAt,
		DetectedAt:     detectedAt,
		Reason:         reason,
		Capabilities:   session.Capabilities,
	}
	if orphan.Capabilities.Provider == "" {
		orphan.Capabilities = ProviderCapabilityFlags(provider, session.Backend)
	}
	if orphan.VisibilityMode == "" {
		orphan.VisibilityMode = "unknown"
	}
	orphan.BlockingReason = orphanBlockingReason(orphan, state)
	return orphan
}

func orphanBlockingReason(orphan OrphanedTask, state string) string {
	task := firstNonEmpty(orphan.TaskRef, orphan.TaskID, orphan.TaskPath, "unknown task")
	agent := firstNonEmpty(orphan.Assignee, orphan.Provider, "assigned agent")
	reason := firstNonEmpty(orphan.Reason, "runtime session is not active")
	return fmt.Sprintf("Unresolved %s execution for %s blocks later %s launches until the task leaves doing or is reclaimed: %s", state, task, agent, reason)
}

// status snapshots the active session strip, session history groups, and
// current orphans, mirroring corechain.Store.Snapshot's equivalent fields.
func (hub *sessionHub) status() Status {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	sessions := make([]RuntimeSession, 0, len(hub.sessions))
	for _, session := range hub.sessions {
		if !session.IsActive() {
			continue
		}
		if session.VisibilityMode == "" {
			session.VisibilityMode = VisibilityModeForSession(session)
		}
		sessions = append(sessions, session)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ClaimedAt < sessions[j].ClaimedAt
	})

	orphans := make([]OrphanedTask, 0, len(hub.orphans))
	for _, orphan := range hub.orphans {
		if !hub.orphanIsVisibleLocked(orphan) {
			continue
		}
		if orphan.ExecutionState == "" {
			orphan.ExecutionState = "orphaned"
		}
		if orphan.VisibilityMode == "" {
			orphan.VisibilityMode = "unknown"
		}
		if orphan.Capabilities.Provider == "" {
			orphan.Capabilities = ProviderCapabilityFlags(firstNonEmpty(orphan.Provider, orphan.Assignee), orphan.Backend)
		}
		if orphan.BlockingReason == "" {
			orphan.BlockingReason = orphanBlockingReason(orphan, orphan.ExecutionState)
		}
		orphans = append(orphans, orphan)
	}
	sort.Slice(orphans, func(i, j int) bool {
		return orphans[i].TaskRef < orphans[j].TaskRef
	})

	return Status{
		RuntimeSessions: sessions,
		SessionGroups:   hub.buildSessionGroupsLocked(),
		OrphanedTasks:   orphans,
	}
}

// buildSessionGroupsLocked groups every known runtime session (live plus
// persisted history in the runtime sqlite database) by task ref/id/path.
// Must be called with hub.mu held (read or write).
func (hub *sessionHub) buildSessionGroupsLocked() []SessionGroup {
	all := map[string]RuntimeSession{}
	if records, err := loadRuntimeSessionRecords(hub.root); err == nil {
		for _, record := range records {
			all[record.ClaimID] = record
		}
	}
	for claimID, session := range hub.sessions {
		if session.VisibilityMode == "" {
			session.VisibilityMode = VisibilityModeForSession(session)
		}
		all[claimID] = session
	}
	if len(all) == 0 {
		return []SessionGroup{}
	}

	groups := map[string]*SessionGroup{}
	order := []string{}
	for _, session := range all {
		key := firstNonEmpty(session.TaskPath, session.TaskID, session.TaskRef)
		if key == "" {
			continue
		}
		group, ok := groups[key]
		if !ok {
			group = &SessionGroup{}
			groups[key] = group
			order = append(order, key)
		}
		group.TaskRef = firstNonEmpty(group.TaskRef, session.TaskRef)
		group.TaskID = firstNonEmpty(group.TaskID, session.TaskID)
		group.TaskPath = firstNonEmpty(group.TaskPath, session.TaskPath)
		group.Sessions = append(group.Sessions, sessionSummary(session))
	}

	sort.Strings(order)
	result := make([]SessionGroup, 0, len(groups))
	for _, key := range order {
		group := groups[key]
		sort.Slice(group.Sessions, func(i, j int) bool {
			return group.Sessions[i].ClaimedAt < group.Sessions[j].ClaimedAt
		})
		assignSessionRoles(group.Sessions)
		result = append(result, *group)
	}
	return result
}

func sessionSummary(session RuntimeSession) SessionSummary {
	capability := ProviderSessionReuseCapability(session.Backend)
	return SessionSummary{
		RuntimeSession: session,
		Resumable:      capability.Level == SessionReuseVerified,
		ReuseNote:      capability.Reason,
	}
}

// assignSessionRoles marks each session in a task's group. Exactly one
// session is "current": the newest that is not explicitly superseded.
// sessions must already be sorted oldest-to-newest by ClaimedAt.
func assignSessionRoles(sessions []SessionSummary) {
	currentIdx := -1
	for i := len(sessions) - 1; i >= 0; i-- {
		if sessions[i].SupersededByClaimID == "" {
			currentIdx = i
			break
		}
	}
	for i := range sessions {
		session := &sessions[i]
		switch {
		case session.SupersededByClaimID != "":
			session.Role = "superseded"
		case i == currentIdx:
			session.Role = "current"
		default:
			session.Role = "historical"
		}
	}
}

func (hub *sessionHub) executionForTask(task tasklifecycle.Task) tasklifecycle.ExecutionState {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if session, ok := hub.activeSessionForTaskLocked(task); ok {
		return executionStateForSession(session)
	}
	if orphan, ok := hub.orphanForTaskLocked(task); ok && hub.taskIsCurrentOrphanLocked(task) {
		return executionStateForOrphan(orphan)
	}
	return tasklifecycle.ExecutionState{
		State:          "none",
		VisibilityMode: "unknown",
		Agent:          task.Assignee,
		Provider:       task.Assignee,
	}
}

// ExecutionStateForSession maps a live session into the task Execution annotation.
func ExecutionStateForSession(session RuntimeSession) tasklifecycle.ExecutionState {
	return executionStateForSession(session)
}

func executionStateForSession(session RuntimeSession) tasklifecycle.ExecutionState {
	state := firstNonEmpty(session.ExecutionStatus, session.Status, "starting")
	return tasklifecycle.ExecutionState{
		State:          state,
		VisibilityMode: VisibilityModeForSession(session),
		SessionPointer: sessionPointerForSession(session),
		ClaimID:        session.ClaimID,
		Agent:          session.Agent,
		Provider:       session.Agent,
		Backend:        session.Backend,
		HostID:         session.HostID,
		HostName:       session.HostName,
		ProcessID:      session.ProcessID,
		ThreadID:       session.ProviderThreadID(),
		SessionID:      session.ProviderSessionID(),
		TurnID:         session.CodexTurnID,
		ThreadTitle:    session.CodexThreadTitle,
		LogPath:        session.LogPath,
		LastEvent:      session.LastEvent,
		LastActivityAt: firstNonEmpty(session.LastSeenAt, session.ExitedAt, session.StartedAt, session.ClaimedAt),
		LastEventAt:    firstNonEmpty(session.LastEventAt, session.LastSeenAt),
		LastOutputAt:   session.LastOutputAt,
		LastStatusAt:   firstNonEmpty(session.LastStatusAt, session.StartedAt, session.ClaimedAt),
		TerminalReason: session.ErrorMessage,
		BlockingReason: session.BlockingReason,
		ProviderError:  session.ProviderError,
		ToolWarning:    toolWarningFromSession(session),
	}
}

func executionStateForOrphan(orphan OrphanedTask) tasklifecycle.ExecutionState {
	return tasklifecycle.ExecutionState{
		State:          firstNonEmpty(orphan.ExecutionState, "orphaned"),
		VisibilityMode: firstNonEmpty(orphan.VisibilityMode, "unknown"),
		SessionPointer: firstNonEmpty(orphan.SessionID, orphan.ThreadID),
		ClaimID:        orphan.ClaimID,
		Agent:          firstNonEmpty(orphan.Assignee, orphan.Provider),
		Provider:       firstNonEmpty(orphan.Provider, orphan.Assignee),
		Backend:        orphan.Backend,
		ProcessID:      orphan.ProcessID,
		ThreadID:       orphan.ThreadID,
		SessionID:      orphan.SessionID,
		TurnID:         orphan.TurnID,
		LogPath:        orphan.LogPath,
		LastEvent:      orphan.LastEvent,
		LastActivityAt: firstNonEmpty(orphan.LastActivityAt, orphan.DetectedAt),
		TerminalReason: orphan.Reason,
		BlockingReason: firstNonEmpty(orphan.BlockingReason, orphanBlockingReason(orphan, firstNonEmpty(orphan.ExecutionState, "orphaned"))),
		ProviderError:  orphan.ProviderError,
	}
}

func sessionPointerForSession(session RuntimeSession) string {
	switch VisibilityModeForSession(session) {
	case string(VisibilityAppVisible):
		return firstNonEmpty(session.RemoteControlURL, session.BackgroundID)
	case string(VisibilityCLIVisible):
		return firstNonEmpty(session.OperatorCommand, session.CursorChatID)
	case string(VisibilityCoreVisible):
		return firstNonEmpty(session.CodexThreadTitle, session.CodexThreadID, session.ProviderThreadID())
	default:
		return session.ProviderSessionID()
	}
}

// revalidateLaunchWaiting re-checks a task's stored launch-wait reason
// against the current session/orphan hub, clearing or replacing it with the
// current blocker. Returns the (possibly updated) task and whether it
// changed.
func (hub *sessionHub) revalidateLaunchWaiting(task tasklifecycle.Task) (tasklifecycle.Task, bool) {
	wait := task.LaunchEvaluation.Waiting
	if wait == nil {
		return task, false
	}

	hub.mu.RLock()
	current, ok := hub.currentLaunchWaitBlockerLocked(task, *wait)
	hub.mu.RUnlock()

	if !ok {
		task.ClearLaunchWaiting()
		return task, true
	}
	if launchWaitMatches(*wait, current) {
		return task, false
	}
	task.MarkLaunchWaiting(current)
	return task, true
}

func (hub *sessionHub) currentLaunchWaitBlockerLocked(task tasklifecycle.Task, wait tasklifecycle.LaunchWait) (tasklifecycle.LaunchWait, bool) {
	switch wait.Scope {
	case "assignee_orphan":
		assignee := firstNonEmpty(wait.Resource, task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee)
		orphan, ok := hub.activeOrphanConflictLocked(assignee, runtimeSessionForTaskIdentity(task))
		if !ok {
			return tasklifecycle.LaunchWait{}, false
		}
		conflict := orphanRuntimeSession(orphan)
		return tasklifecycle.LaunchWait{
			Scope:           "assignee_orphan",
			Resource:        assignee,
			ConflictClaimID: conflict.ClaimID,
			Reason:          LaunchConflictReason(conflict, "assignee_orphan", assignee),
			WaitingSince:    firstNonEmpty(wait.WaitingSince, time.Now().Format(time.RFC3339)),
		}, true
	case "assignee":
		assignee := firstNonEmpty(wait.Resource, task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee)
		for _, session := range hub.sessions {
			if !session.IsActive() || session.Agent == "" || session.Agent != assignee {
				continue
			}
			if sameTaskIdentity(session.TaskPath, task.RelativePath, session.TaskID, task.ID, session.TaskRef, task.Ref) {
				continue
			}
			return tasklifecycle.LaunchWait{
				Scope:           "assignee",
				Resource:        assignee,
				ConflictClaimID: session.ClaimID,
				Reason:          LaunchConflictReason(session, "assignee", assignee),
				WaitingSince:    firstNonEmpty(wait.WaitingSince, time.Now().Format(time.RFC3339)),
			}, true
		}
	case "project":
		for _, session := range hub.sessions {
			if !session.IsActive() || session.ProjectID == "" || session.ProjectID != wait.Resource {
				continue
			}
			if sameTaskIdentity(session.TaskPath, task.RelativePath, session.TaskID, task.ID, session.TaskRef, task.Ref) {
				continue
			}
			return tasklifecycle.LaunchWait{
				Scope:           "project",
				Resource:        wait.Resource,
				ConflictClaimID: session.ClaimID,
				Reason:          LaunchConflictReason(session, "project", wait.Resource),
				WaitingSince:    firstNonEmpty(wait.WaitingSince, time.Now().Format(time.RFC3339)),
			}, true
		}
	case "repository":
		for _, session := range hub.sessions {
			if !session.IsActive() || session.Repository == "" || session.Repository != wait.Resource {
				continue
			}
			if sameTaskIdentity(session.TaskPath, task.RelativePath, session.TaskID, task.ID, session.TaskRef, task.Ref) {
				continue
			}
			return tasklifecycle.LaunchWait{
				Scope:           "repository",
				Resource:        wait.Resource,
				ConflictClaimID: session.ClaimID,
				Reason:          LaunchConflictReason(session, "repository", wait.Resource),
				WaitingSince:    firstNonEmpty(wait.WaitingSince, time.Now().Format(time.RFC3339)),
			}, true
		}
	}
	return tasklifecycle.LaunchWait{}, false
}

func runtimeSessionForTaskIdentity(task tasklifecycle.Task) RuntimeSession {
	return RuntimeSession{
		TaskRef:  task.Ref,
		TaskID:   task.ID,
		TaskPath: task.RelativePath,
		Agent:    firstNonEmpty(task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee),
	}
}

func launchWaitMatches(left tasklifecycle.LaunchWait, right tasklifecycle.LaunchWait) bool {
	return left.Scope == right.Scope &&
		left.Resource == right.Resource &&
		left.ConflictClaimID == right.ConflictClaimID &&
		left.Reason == right.Reason
}
