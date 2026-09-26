# Implementation Plan

This repository is the public static surface for Fleet. It started with a
first-viewport pass and has an experimental structural landing skeleton in
Svelte. The canonical content plan now lives in `CONTENT.md`; update that file
before changing page structure in code.

## Target Stack

- SvelteKit
- `@sveltejs/adapter-static`
- GitHub Pages
- GitHub Actions deployment

## Current Landing Structure

The current Svelte page is a draft. The accepted landing structure should be
planned in `CONTENT.md` first and only then moved into code. The current draft
is structured around existing Fleet App/Data capabilities rather than invented
marketing claims:

- Hero: local-first orchestration for coding agents.
- Why Fleet exists: agent chats are easy to start but hard to coordinate.
- Operating loop: capture, backlog readiness, deterministic launch gates,
  human review.
- Product states: 16:9 placeholders for future screenshots.
- Capabilities: local data root, project/repository ownership, lifecycle,
  routing, launch policy, dependencies, worker outcome protocol, review-first
  completion.
- Install preview: static site will point to releases, install notes, and docs.

The screenshot placeholders are deliberate. Replace each with a real capture
when the App is ready:

- Work dashboard with sidebar project tree, task columns, Needs Attention, and
  active Sessions strip.
- Manager bar with text/voice input, `@project` chips, `@agent` chips, and
  applied task result.
- Task modal with status, priority, project, assignee, `depends_on`, launch
  evaluation, comments, and task body.
- Sessions panel with active, HITL, closed, failed, orphaned, and resumable
  sessions.
- Settings > Agents with worker readiness, executable path, visibility class,
  and routing notes.
- Settings > General/Data root plus registry counts.

## Deployment Notes

The site must build to static files suitable for GitHub Pages. Prefer a GitHub
Actions workflow that:

1. checks out the repository;
2. installs Node dependencies;
3. builds the SvelteKit static output;
4. uploads the generated artifact;
5. deploys to GitHub Pages.

## Validation Checklist

- local dev server runs;
- production build succeeds;
- static output is GitHub Pages compatible;
- desktop screenshot inspected at approximately `1440px`;
- tablet screenshot inspected at approximately `1024px`;
- mobile screenshot inspected at approximately `390px`;
- no fake claims/data;
- primary CTA is clear;
- Fleet product visual is prominent;
- design still matches `DESIGN.md`.
