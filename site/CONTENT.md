# Fleet Landing Content Plan

This document is the content source of truth for the public Fleet landing
page. Keep the narrative here first; move it into Svelte only after the story
is coherent.

## Current Content Problem

The first structural implementation pass was too literal:

- it listed internal capabilities before the landing had a clean narrative;
- it exposed too much product machinery at once;
- screenshot placeholders were useful, but the page did not yet explain the
  product in the right order.

Do not continue by adding more UI sections. First fix the content model.

## Audience

Primary audience:

- senior developers using multiple coding agents;
- people who already run Codex, Claude Code, Cursor, Gemini, or similar tools;
- local-first operators who care about files, repos, review, and control.

Secondary audience:

- people evaluating whether Fleet is a real product or just another agent
  wrapper;
- future contributors/agents who need to understand what the public site is
  allowed to claim.

Not the audience:

- generic AI SaaS buyers;
- non-technical productivity users;
- teams looking for a cloud project-management suite;
- people who need a hosted IDE replacement.

## One-Sentence Positioning

Fleet is a local control plane for coding-agent work across real projects and
repositories.

## Short Positioning Variants

Use one of these directions for the hero after review:

1. Keep coding agents attached to the work they own.
2. Run coding agents without losing track of the work.
3. A local control plane for Codex, Claude Code, Cursor, and human review.
4. Turn agent chats into durable local work.

The first variant is the current best fit because it connects task ownership,
repository ownership, and session ownership in one sentence.

## Core Promise

Fleet does not replace coding agents. It coordinates them.

The app should make it clear:

- what work exists;
- which project/repository owns that work;
- which worker owns the current attempt;
- whether the attempt is launchable, running, waiting, blocked, or ready for
  review;
- what physical artifact was produced;
- whether the human accepted it.

## Product Boundary

Fleet is not:

- an IDE;
- a model;
- a chat app;
- a cloud task tracker;
- a magic autonomous engineer;
- a replacement for Git, Cursor, Claude Code, Codex, or code review.

Fleet is:

- a local runtime;
- a task/project/workspace registry;
- a launch policy layer;
- a session visibility layer;
- a review boundary;
- a file-backed operating system for personal agent work.

## Landing Narrative

### 1. Hero

Goal: explain the product in one screen.

Must communicate:

- local-first orchestration;
- works with existing coding agents;
- project/repository ownership;
- visible sessions and review state.

Potential copy:

```text
Keep every agent attached to the work it owns.

Fleet turns rough input into durable local tasks, then coordinates Codex,
Claude Code, Cursor, Gemini, and human review across your real repositories.
The coding stays in native tools; Fleet keeps ownership, launch state, and
review from dissolving into forgotten chats.
```

Primary CTA:

```text
Download Fleet
```

Secondary CTA:

```text
View install path
```

Quiet pricing:

```text
First 5 completed tasks free. Then $9/month.
```

Hero visual:

- current hand-built product visual is acceptable until screenshots exist;
- show repositories, active owner, task queue/review states;
- keep one active owner/repository as a visible rule.

### 2. Problem

Goal: make the user feel the pain before showing internals.

Core idea:

```text
Agent chats are easy to start and hard to operate.
```

Questions this section should raise:

- Which repo is safe to touch?
- Who owns the current attempt?
- Was the output reviewed?
- Did the work produce a real artifact?
- Is another worker already editing the same project?
- Where does context live after the chat ends?

This section should be short. Do not list every feature here.

### 3. Operating Loop

Goal: show the lifecycle as a simple loop.

Recommended steps:

1. Capture
   - voice/text/raw idea enters Manager;
   - output becomes Inbox item or structured task.
2. Shape
   - task gets project, repository, assignee, priority, dependencies;
   - backlog means structured but not ready.
3. Launch
   - todo/needs_rework is eligible;
   - deterministic gates check dependencies, worker readiness, visibility,
     project/repository/assignee locks.
4. Review
   - agents stop at needs_review;
   - done is human acceptance;
   - every completed task needs a physical artifact.

Tone:

- operational, not inspirational;
- "what happens next" instead of "AI magic".

### 4. Product Surfaces

Goal: introduce concrete screens/states without pretending screenshots are
ready.

Use 16:9 placeholders until real captures exist.

Screens to capture later:

1. Work dashboard
   - sidebar project tree;
   - task columns/list;
   - Needs Attention;
   - active Sessions strip.
2. Manager capture
   - text/voice input;
   - `@project` and `@agent` chips;
   - applied task result.
3. Task detail
   - status, priority, project, assignee;
   - `depends_on`;
   - launch evaluation;
   - comments and task body.
4. Runtime sessions
   - active sessions;
   - HITL/waiting input;
   - closed/failed/orphaned/resumable states.
5. Agent settings
   - Claude, Codex, Cursor, Gemini;
   - executable path;
   - readiness;
   - visibility class;
   - routing notes.
6. Data root / registry
   - configured data root;
   - workspaces/projects/repositories counts;
   - generated registry/index state.

Each surface needs:

- one sentence explaining why this screen matters;
- one label naming the exact state to screenshot later;
- no fake UI screenshots.

### 5. Capabilities

Goal: list real capabilities after the user understands the workflow.

Candidate capability groups:

#### Local data root

Plain files:

- Inbox;
- Work;
- Fleet profiles;
- Archive;
- registry;
- notes/decisions/tasks.

#### Project/repository model

Fleet separates:

- workspace;
- project;
- repository.

This is the foundation for routing and launch locks.

#### Task lifecycle

Canonical statuses:

- backlog;
- needs_rework;
- todo;
- doing;
- blocked;
- needs_review;
- done;
- archived.

Important product meaning:

- backlog -> todo is readiness;
- todo -> doing is claim/lock;
- worker output goes to needs_review;
- done is human acceptance.

#### Launch policy

Fleet checks:

- assignee;
- worker readiness;
- visibility class;
- dependencies;
- repository/project/assignee/task locks;
- orphaned doing tasks.

#### Session visibility

Supported classes:

- app_visible;
- cli_visible;
- core_visible;
- headless blocked for auto-launch.

This is a strong product opinion: visible sessions over silent background magic.

#### Worker outcome protocol

Agents report:

- completed;
- blocked;
- failed;
- needs_input;
- needs_rework.

Reports include:

- summary;
- artifacts;
- tests;
- question/error.

#### Review and artifact rule

Definition of done:

- no task is complete if the only result is a chat response;
- finished agent work stops for human review;
- bot-produced work stops at needs_review.

### 6. Install / Distribution

Goal: explain how the public site will eventually become useful.

Current state:

- static public site;
- GitHub Pages;
- downloads via GitHub Releases later;
- app source private/separate;
- Fleet runs locally against configured data root.

Do not overbuild this section until release packaging is real.

## Copy Rules

Prefer:

- "local control plane";
- "coding-agent work";
- "real projects and repositories";
- "durable local tasks";
- "review boundary";
- "visible sessions";
- "native tools stay native".

Avoid:

- "autonomous workforce";
- "10x";
- "magic";
- "AI-powered project management";
- "deploy agents at scale";
- "all-in-one IDE";
- "agent swarm";
- persistence or storage architecture (files, databases, directories, formats), unless it is part of Fleet's public contract.

## Proof Rules

Do not claim:

- customer adoption;
- benchmarks;
- team/cloud features;
- integrations that do not exist;
- screenshots as real before they are captured.

Allowed proof:

- actual app concepts from App/Data;
- real task lifecycle;
- real worker/session states;
- real local data model;
- explicit placeholders for future screenshots.

## Open Content Questions

- Should the first public page sell to solo operators first, or keep the
  language broad enough for small teams later?
- Should pricing stay in hero, or move lower until release packaging exists?
- Should "Manager" be named publicly now, or described as capture/input until
  the UX is stable?
- Should Plane provider support be mentioned, or hidden until it is production
  ready?
- Should "Gemini" appear in hero, or only in agent settings/capabilities?

## Next Implementation Direction

Before touching Svelte again:

1. Review this document.
2. Decide final hero headline and lede.
3. Decide whether the product surfaces section should come before or after
   capabilities.
4. Replace the current structural Svelte content only after this document is
   accepted.
