// Package boundary notes (CORE-94 … CORE-102)
//
// # What lives here today
//
// Everything about the task lifecycle *domain*, independent of how a task is
// ultimately stored:
//
//   - the Task model (model.go): Task, Comment, Launch, LaunchEvaluation,
//     ExecutionState, WorkerResult;
//   - status transition rules (status_transition.go);
//   - pickup/priority ordering (priority.go);
//   - create-request normalization and Core-wide CORE-N ref allocation
//     (task_create.go);
//   - patch/create request types (patch.go);
//   - launch validation gates (validate.go);
//   - execution-outcome-to-task-status finalization decisions (finalize.go);
//     daemon runtime prefers taskflow.ReportExecution (CORE-105) and keeps
//     FinalizeSucceededExecution as a domain-level mapping helper;
//   - task-facing provider error classification (provider_error.go).
//
// corechain (the daemon/runtime/dashboard package) depends on this package;
// this package does not, and must not, import corechain.
//
// # What moved out in CORE-102
//
// Markdown file I/O — path resolution, frontmatter parse/serialize, task card
// create/mutate/claim, comment section editing, IsTaskFile — now lives in
// internal/taskprovider/markdown (MarkdownProvider). That package implements
// taskprovider.Provider and exposes Flow() as taskflow.TaskProvider.
//
// Domain helpers NormalizeTaskCreateRequest, AllocateNextWorkRef, and
// DeriveBlockedReason stay here so Plane and Markdown share identical rules.
package tasklifecycle
