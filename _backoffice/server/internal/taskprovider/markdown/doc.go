// Package markdown is the real Markdown/file-backed TaskProvider (CORE-102).
//
// It owns every filesystem-coupled concern for task cards:
//
//   - path resolution under Work/<project>/tasks/*.md
//   - file reading/writing and atomic replaces
//   - frontmatter parsing and Markdown serialization
//   - task id/file-base generation and Core ref allocation hooks
//   - comments stored in the "## Review Comments" section
//   - file-specific validation (path safety, frontmatter shape)
//   - index rebuild / projection hooks via Hooks
//
// Domain rules (status transitions, create input normalization, Core-wide
// CORE-N ref counters, blocked-reason derivation, launch gates, finalization)
// stay in tasklifecycle and are called from this package. Callers that need
// the application contract use taskflow.TaskProvider; the legacy
// taskprovider.Provider surface remains for Runtime until CORE-104/105/107.
//
// fswalker must not parse or create tasks. It only detects filesystem
// changes; this package reloads the affected card and emits a normalized
// TaskEvent{Before,After,Source} via ObserveFileChange (CORE-103 / CORE-111).
// Generic execution consumes TaskEvent and loads through TaskService — not
// filesystem paths.
package markdown
