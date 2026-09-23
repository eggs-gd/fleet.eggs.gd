package tasklifecycle

import "errors"

// Sentinel errors shared by Markdown and Plane providers for claim/conflict/
// transition failures. Providers return these so callers can errors.Is them
// without knowing which storage backend produced the failure.
var (
	ErrTaskAlreadyClaimed     = errors.New("task is already claimed")
	ErrTaskConflict           = errors.New("task mutation conflict")
	ErrTaskTransitionRejected = errors.New("task status transition rejected")
)
