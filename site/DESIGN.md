# Fleet Site Design Direction

This document records the initial visual direction for Fleet's public landing
page. It exists so future implementation does not drift into generic AI SaaS
styling.

## Product Positioning

Fleet coordinates coding agents across projects.

It is not an IDE replacement and not a new coding agent. Cursor, Claude Code,
Codex, and similar tools keep their native interaction surfaces. Fleet sits
around them as the local orchestration layer:

- discover projects and agent sessions;
- connect/manage workers;
- assign work;
- keep one active task/agent per repository;
- preserve review boundaries instead of letting agent work disappear into chat.

## Reference Study

References:

- <https://cccc.sh/en>
- <https://vibeide.dev/>

Do not copy layout, palette, typography, components, illustrations, or brand
language from either reference.

Useful shared principles:

- product is visible early;
- dense information can still feel designed if the grid is disciplined;
- navigation is direct and functional;
- large typography works when paired with real product artifacts;
- restrained motion and borders feel more credible than decorative effects;
- sections are composed from real workflows, not feature-icon filler.

Fleet's own dashboard is the identity source. The references are a quality bar,
not a style source.

## Visual Character

Fleet should feel:

- technical;
- mature;
- opinionated;
- restrained;
- slightly industrial;
- intentionally designed;
- built by an experienced engineer who cares about design.

Avoid:

- purple/blue AI gradients;
- glowing blobs;
- glassmorphism;
- excessive card grids;
- pill-shaped everything;
- stock illustrations;
- fake metrics/testimonials/logos;
- generic "AI magic" phrasing;
- huge empty centered hero sections;
- framework-default styling.

Prefer:

- strong, compact typography;
- deliberate spacing;
- clear grid;
- subtle rules and borders;
- product screenshots or product-like artifacts;
- terminal/session/task surfaces where appropriate;
- restrained asymmetry;
- useful details over decoration.

## Fleet UI Cues To Carry Forward

From the current Fleet app:

- dense operator-console feel;
- small radius system around `3px`-`6px`;
- thin borders/rules;
- dark and light themes, with dark UI feeling especially product-like;
- compact task/session/status vocabulary;
- muted surfaces with sharp contrast only for state and action;
- utility-first composition, not marketing ornament.

The landing page should elevate this language. It should not simply paste the
dashboard into a hero or make everything as cramped as the app.

## First Viewport Concept

Goal: communicate immediately that Fleet is a local control layer for real
coding agents across repositories.

Recommended composition:

- top navigation with brand, Product/Docs/Releases anchors, and Download CTA;
- left side: concise positioning and primary CTA;
- right side: large product visual based on Fleet's real interaction model;
- supporting line: "Local-first orchestration for coding agents" or similar.

The product visual should show:

- multiple repositories/projects;
- agent/session status;
- task queue/review states;
- a visible rule that only one active task/agent owns a repository at a time.

If real screenshots are unavailable during the first pass, use a carefully
constructed static product mock that is clearly Fleet-like and avoids fake
customer claims. Replace it with real screenshots as soon as possible.

## Typography

Use system UI or a simple local web-safe stack first. Avoid adding font
dependencies until there is a clear reason.

Suggested hierarchy:

- small, uppercase eyebrow with wide but not exaggerated tracking;
- hero heading large but not billboard-empty;
- body copy compact, high contrast, line length controlled;
- UI mock labels small and dense.

Do not scale type with viewport width. Use responsive breakpoints and explicit
sizes.

## Grid And Spacing

Desktop first viewport:

- constrained page width around `1180px`-`1280px`;
- two-column composition, product visual at least half the width;
- header aligned to the same grid;
- enough vertical density that a hint of the next section can appear.

Tablet/mobile:

- product visual remains visible and meaningful;
- do not just stack a huge headline above an unreadable screenshot;
- compress navigation deliberately;
- CTA remains immediately reachable.

## Surfaces, Borders, Radius

Use flat surfaces with subtle separation:

- page background: dark industrial neutral or very controlled light neutral;
- product visual surface: darker/lighter panel with thin border;
- borders: one-pixel, low contrast;
- radius: small (`4px` preferred), not soft SaaS cards;
- shadows: minimal or none; use borders and contrast first.

## Motion

Motion is optional and should be quiet:

- small hover transitions;
- subtle status pulse only if it communicates "running";
- no hero blob movement, particle fields, or decorative AI animation.

## Content Rules

Do not invent:

- testimonials;
- company logos;
- usage numbers;
- fake customer claims;
- fake benchmark claims.

Use precise copy:

- "coordinates coding agents across projects";
- "keeps repository ownership clear";
- "finished work waits for review";
- "use the tools you already use."

Avoid:

- "revolutionary";
- "magical";
- "10x";
- "autonomous workforce";
- generic "AI agent platform" language.

## Technical Direction

Build with:

- SvelteKit;
- static adapter;
- GitHub Pages compatible output;
- GitHub Actions publish workflow.

Keep implementation simple:

- no backend;
- no CMS;
- no auth;
- no analytics unless explicitly requested later;
- no heavyweight component library.

Before considering the first viewport done:

- run the local dev/build path;
- inspect rendered screenshots at roughly `1440px`, `1024px`, and `390px`;
- critique hierarchy, density, product prominence, alignment, and responsive
  behavior;
- iterate after the last meaningful CSS/layout change;
- keep this document updated if the visual language changes.
