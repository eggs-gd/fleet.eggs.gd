---
schema_version: 1
id: work-YYYY-MM-DD-slug
ref: TAG-N
title: "Human-readable task title"
type: feature
status: backlog
priority: 5
project: project-id
repositories:
  - relative/repository/path
depends_on: []
assignee: unassigned
assignment_reason: "Why this worker is the right/default assignee"
source: text
source_inbox:
created_at: YYYY-MM-DDTHH:MM:SS+ZZ:ZZ
updated_at: YYYY-MM-DDTHH:MM:SS+ZZ:ZZ
launch:
  agent:
  mode:
  auto_commit: false
  auto_push: false
  auto_pr: false
---

# Human-readable task title

## Request

What to do, in clear language. Not the raw input: that stays in the Inbox item
when there is one.

## Acceptance Criteria

- [ ] First expected outcome.
- [ ] Second expected outcome.

## Context

Why this is wanted, the limits, and every decision already made. The worker
sees only the task, so write the reasoning here instead of pointing at it. If a
choice should outlive the task, add a criterion that the worker documents it in
the repository's own docs, in the same change. Leave this section out when
there is nothing to add.

## Review Comments

Append review notes here when a completed attempt needs changes before it can
move to `done`. If the operator rejects `needs_review` work, set
`status: needs_rework` and make the latest comment the active rework
instruction.

## Activity Log

- YYYY-MM-DDTHH:MM:SS+ZZ:ZZ — Created.
