# Fleet Public Site

Public landing for Fleet. This directory is `site/` inside the App
repository, not a separate repo. GitHub Actions and Releases live at the
App root.

Fleet is a local developer tool for orchestrating coding agents across
multiple projects and repositories. This landing is the public surface:

- landing page;
- install/download entry point;
- documentation entry point.

## Stack Decision

The implementation target is:

- SvelteKit;
- `@sveltejs/adapter-static`;
- static GitHub Pages deployment;
- publish by GitHub Actions.

Do not build a backend, CMS, auth system, analytics pipeline, or server-only
feature in this directory. The generated output must be static files suitable
for GitHub Pages.

## Implementation Scope

The landing copy canon is `LANDING-v3.md`. Visual rules live in `DESIGN.md`.
Change those before changing the page structure in Svelte.

The live page is `src/routes/+page.svelte`: hero, manager loop, workers,
execution, dashboard, and install.

## Agent Development

Developer instructions live in the App root `AGENTS.md`. `CLAUDE.md`,
`GEMINI.md`, and Cursor rules there point at that file. Codex reads it
from the repository root.

Developer MCP servers (Svelte, gopls, design-patterns) are declared on the
App root, not in this directory.

Before finishing code work, run:

```sh
npm run check
npm run build
```
