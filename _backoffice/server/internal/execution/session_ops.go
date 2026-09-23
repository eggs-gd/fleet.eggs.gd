package execution

// Session hub is the system of record for live sessions/orphans.
// BoardHooks must not own these maps — composition only supplies task hooks.

func (s *Service) UpsertSession(session RuntimeSession) {
	if s == nil || s.hub == nil {
		return
	}
	s.hub.UpsertSession(session)
}

func (s *Service) TryReserveSession(session RuntimeSession) (RuntimeSession, string, bool) {
	if s == nil || s.hub == nil {
		return RuntimeSession{}, "", false
	}
	return s.hub.TryReserveSession(session)
}

func (s *Service) RemoveSession(claimID string) {
	if s == nil || s.hub == nil {
		return
	}
	s.hub.RemoveSession(claimID)
}

func (s *Service) Session(claimID string) (RuntimeSession, bool) {
	if s == nil || s.hub == nil {
		return RuntimeSession{}, false
	}
	return s.hub.Session(claimID)
}

func (s *Service) ClearOrphanForSession(session RuntimeSession) {
	if s == nil || s.hub == nil {
		return
	}
	s.hub.ClearOrphanForSession(session)
}

func (s *Service) UpsertOrphan(orphan OrphanedTask) {
	if s == nil || s.hub == nil {
		return
	}
	s.hub.UpsertOrphan(orphan)
}

func (s *Service) upsertSession(session RuntimeSession) {
	s.UpsertSession(session)
}

func (s *Service) tryReserveSession(session RuntimeSession) (RuntimeSession, string, bool) {
	return s.TryReserveSession(session)
}

func (s *Service) removeSession(claimID string) {
	s.RemoveSession(claimID)
}

func (s *Service) session(claimID string) (RuntimeSession, bool) {
	return s.Session(claimID)
}

func (s *Service) clearOrphanForSession(session RuntimeSession) {
	s.ClearOrphanForSession(session)
}

func (s *Service) recordOrphan(task Task, session RuntimeSession, state, reason string) {
	if s == nil || s.hub == nil {
		return
	}
	orphan := orphanForTaskSession(task, session, state, reason)
	orphan.ProviderError = session.ProviderError
	if session.Capabilities.Provider != "" {
		orphan.Capabilities = session.Capabilities
	}
	s.hub.UpsertOrphan(orphan)
}

func (s *Service) hasOrphanForTask(task Task) bool {
	if s == nil || s.hub == nil {
		return false
	}
	_, ok := s.hub.CurrentOrphanForTask(task)
	return ok
}

func (s *Service) ActiveOrphanConflict(assignee string, session RuntimeSession) (RuntimeSession, bool) {
	if s == nil || s.hub == nil {
		return RuntimeSession{}, false
	}
	return s.hub.ActiveOrphanConflict(assignee, session)
}

func (s *Service) ActiveSessionForTask(task Task) (RuntimeSession, bool) {
	return s.activeSessionForTask(task)
}

func (s *Service) activeSessionForTask(task Task) (RuntimeSession, bool) {
	if s == nil || s.hub == nil {
		return RuntimeSession{}, false
	}
	return s.hub.ActiveSessionForTask(task)
}

func (s *Service) orphanSession(claimID string) (RuntimeSession, bool) {
	if s == nil || s.hub == nil {
		return RuntimeSession{}, false
	}
	orphan, ok := s.hub.OrphanByClaimID(claimID)
	if !ok {
		return RuntimeSession{}, false
	}
	return orphanRuntimeSession(orphan), true
}
