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

Normalized request in clear language.

## Raw Input

Original user wording, transcript, URL, or source text.

## Project Resolution

- Matched workspace: `project-id`
- Matched repository: `relative/repository/path`
- Resolution method: exact title | alias | repository name | path | fuzzy
- Confidence: high | medium | low

## Acceptance Criteria

- [ ] First expected outcome.
- [ ] Second expected outcome.

## Context

- Workspace: `Work/project-id/PROJECT.md`
- Project owner: `project-id`
- Repository: `relative/repository/path`

## Deliverable

Expected physical artifact:

- File, doc, note, decision, roadmap update, code change, generated output, or
  task-card update required for completion.

Produced artifacts:

- Pending.

## Handoff

Clear next action for the assignee or next agent.

Assignment reason: why this task is assigned to `assignee`.

Launch policy: `todo` and `needs_rework` tasks are daemon-pickup candidates
when they have a supported assignee and exactly one repository. Leave
`launch.agent` empty unless this task intentionally launches a different worker
than `assignee`; do not use `launch.mode` to choose between visible and
headless workers. Invisible/headless daemon workers are not supported.

## Review Comments

Append review notes here when a completed attempt needs changes before it can
move to `done`. If the operator rejects `needs_review` work, set
`status: needs_rework` and make the latest comment the active rework
instruction.

## Activity Log

- YYYY-MM-DDTHH:MM:SS+ZZ:ZZ — Created by Manager.
