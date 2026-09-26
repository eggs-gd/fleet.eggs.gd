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

Content strategy lives in `CONTENT.md`. Update that document first, then move
accepted content into Svelte.

The first implementation pass built:

- navigation/header;
- first viewport/hero;
- primary product visual;
- primary CTA.

Later sections should not be treated as final just because they exist in code.
Plan the content in `CONTENT.md`, then implement the accepted structure.

See `DESIGN.md` before implementing.

## Agent Development

Repository instructions live in `AGENTS.md`. `CLAUDE.md` and Cursor rules
point there so agents share one source of truth.

Local MCP servers are declared in `.mcp.json`:

- Svelte documentation and autofix tools;
- gopls for consistency with neighboring Fleet repos;
- optional design-pattern research.

Before finishing code work, run:

```sh
npm run check
npm run build
```
