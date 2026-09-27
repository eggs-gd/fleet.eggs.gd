# _registry

Generated, machine-readable project-discovery state — which
repositories/projects exist and how they relate. Do not hand-edit —
regenerate with `core scan` / `make scan` from App.

This is portfolio state only. Runtime execution state (session records/logs,
the audit event log, the Manager relay log) is not portfolio content and
does not live here — it's in `~/.fleet/runtime.db`, App's own process
history.

## Current Files

The project sniffer writes:

- `repositories.json` — discovered git repositories and metadata;
- `project-groups.json` — candidate logical groupings;
- `bootstrap-report.md` — human-readable scan report;
- `deletion-candidates.json` — repositories worth reviewing for cleanup;
- `deletion-candidates.md` — human-readable cleanup review list;
- `protection-rules.json` — human-reviewed cleanup exclusions and intentional
  workspace/submodule rules.

## Group Semantics

Top-level folders that contain multiple repositories are treated as intentional
workspace groups, not weak duplicate hints.

Other signals still need review:

- `same git remote` can mean duplicate local checkouts;
- `same base repository name` can mean related products or old/test copies;
- `nested repositories` can mean submodules, vendored sources, or intentional
  child repositories.

## Deletion Candidate Semantics

Deletion candidates are never commands. They are review prompts.

Strong cleanup signals include duplicate local checkouts of the same remote and
nested repositories. Weaker signals include names such as `old`, `test`,
`backup`, or `playground`, missing remotes, and missing README files.

Nothing in Fleet should delete a repository unless a human explicitly confirms
that specific path.

Human-reviewed protection rules live in `protection-rules.json`. Use them for
intentional workspace layouts, nested dependency checkouts, and known
experimental copies that must not be suggested for cleanup.
