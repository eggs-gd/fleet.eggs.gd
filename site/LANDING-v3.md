# Fleet

Keep the product concept:

**remote agents -> too many separate sessions -> one manager -> rough intent becomes executable work -> Fleet runs the deterministic layer underneath -> native agents and humans do the work -> repository ownership keeps execution safe -> review and human input stay explicit**

Do not introduce a new story. Compress the existing one.

The implementation lives in `src/lib/sections_v2`. `ManagerLoopSection` is the loop animation after the hero: a chat line becomes a backlog task, a yellow card returns as Needs Attention, and after a reply it flies back green into Review. It is not the Remote Human Input section below. A back-to-top control appears after scrolling and links to `#top`.

Header:

**Fleet**, **Manager**, **Workers**, **Execution**, **Download**

---

# Hero / Manager

**Implemented.** Hero and Manager are no longer two sections — they were merged into one
`HeroSection.svelte` (`id="product"`) so the diagram reads as a direct continuation of the
origin story instead of a second, separately-titled block. There is no more standalone
`ManagerSection.svelte`; the nav's "Manager" link points at `#product`.

## I wanted one manager for coding-agent work.

The origin-story paragraphs briefly went through a `<blockquote>` treatment (66.6% width, left
accent rule) but that was reverted — they're back to plain centered `SectionText` paragraphs,
same as the rest of the hero, no quote styling:

That is pretty much how Fleet started.

Remote sessions let me use coding agents from my phone.

It worked — until I had Codex in one project, Claude in another, Cursor somewhere else, and my own memory acting as the orchestrator.

I didn't want remote access to a bunch of coding agents.

**I wanted one manager.**

Fleet became that manager: rough intent in, executable work out, real agents underneath.

All of the above, plus everything below, is centered (`text-align: center` + `margin: 0 auto`
on each block). The diagram and its internal absolute-positioned scene stay full width — only
the surrounding text/quote/CTA blocks are centered.

Directly under that copy — same section, no heading in between — sits a full-width **animated
scene diagram** (`.manager-scene`, pure CSS keyframes, no JS/canvas/SVG animation library):

```text
chaotic input (voice note, screenshot, pasted chat, broken-icons note)
  -> chat/You panel
  -> Manager box
  -> Backlog (4-slot conveyor, bottom-in / top-out)
  -> worker cards: gemini / claude / codex / cursor
```

Concretely, what's built:

- A pile of "junk" travelers (voice-note waveform chip, two pixel-art thumbnails, a pasted
  chat snippet, a broken-icons note) loop from the chat input into distinct pile slots and
  fade back into the manager — replacing an earlier, since-removed idea of static decorative
  chips (they read as dead weight within a few seconds; everything now stays on the animated
  conveyor).
- A "? which repo" chip flies manager → chat-input (turns amber = needs the user) → back into
  the pile (recolors to normal) — the missing-context loop from the original spec, done as
  motion instead of a paragraph.
- A task chip (`CORE-162` → `163` → `164` → `165`, four staggered copies on one shared
  20s-cycle keyframe so the number visibly increments each time instead of repeating) flies
  manager → the bottom slot of the backlog.
- The backlog is a real 4-slot conveyor: a task chip enters at the bottom slot, shifts up
  through 3 slots, then — as the same element, no hand-off to a second "dispatch" node — flies
  from the top slot straight to whichever worker card is free next. Landing coordinates per
  card are taken from measured `getBoundingClientRect()`, not eyeballed, so it lands on the
  card rather than near it.
- Worker cards are rectangles (icon + name + colored status dot), not plain text pills. Icons
  are real open-source brand marks (Simple Icons) for Gemini, Claude, and Cursor; Codex/OpenAI
  has no open-source mark available (OpenAI had its icon pulled from Simple Icons at its own
  request), so that card uses a neutral "X" monogram until a real asset is supplied. The status
  dot flips green → amber → green in sync with the moment its card's incoming task chip lands.

Caption under the diagram:

A voice note, a screenshot, a half-formed sentence — the manager sorts it into the right
project, shapes it into a scoped task, and queues it for a worker. If the project or the
intent is unclear, it comes back with a question instead of a guess.

Footer, full width:

**Three sentences in. Usable work out.**

Then, at the very bottom of the merged section (not mid-page anymore):

**Download** — a smart split button, centered next to a colored `Early access` badge (amber pill).
See "Download" below. No pricing text anywhere on the page.

No real Fleet screenshot is used anywhere in this section. The diagram is a reconstructed,
labeled scene, not a product screenshot — this replaces the original spec's two-column
"copy left / real Fleet screenshot right" hero plan, which was never implemented.

Additional behavior mentioned in the earlier spec (decomposition creates thinking before
locking implementation) is **not yet represented**, in copy or visual — still open.

---

# Workers

## The agents are still the agents.

Fleet does not replace Codex, Claude Code, Cursor, Gemini, or any other coding tool with another generic chat UI.

They remain native workers with different strengths, limits, subscriptions, and launch behavior.

Fleet discovers what is available, validates whether it can run, and lets the manager assign work without pretending every agent is the same.

Humans are workers too.

A task graph can include product decisions, architecture investigation, implementation, review, and manual unblocking.

Some nodes go to agents.

Some go to me.

**Human-in-the-loop is part of the workflow, not an exception path.**

**Implemented, text differs from the spec above.** The section renders the original three
paragraphs and the closing line as they were before the visual work; the "Fleet discovers what
is available…" and "Some nodes go to agents / to me" lines are not on the page. Copy is
centered, the roster spans full width.

Visual: not the `Human → Claude → Codex → Human` pipeline (it sold multi-agent dependencies,
which is not this section's point). Instead a roster of four named workers, so Fleet visibly
does not turn agents into anonymous `worker-1..3`:

- Cards: Claude Code, Codex, Cursor, You. Each has two role lines (Architecture / Large context,
  Implementation / Focused tasks, Local work / Interactive, Product / Decisions), a status
  and a badge (`canonical ✓`, `active session`, `human`). No invented benchmarking.
- Each worker has its own muted color (Claude terracotta, Codex teal, Cursor rose, You sand),
  applied to the card's top accent, icon and role lines. The task that flies to that worker
  uses the same color, so the routing reads without studying it. Colors avoid the
  status colors (green available, amber busy) and blue/purple.
- Status label and dot change together: `available` green → `busy` amber while a task lands.
  Cursor stays `active` amber.
- Calm pure-CSS loop (16s): four tasks arrive one at a time from the top and settle on
  their worker — auth architecture → Claude, mobile nav regression → Codex, onboarding
  behavior → You, UI interaction → Cursor.
- Icons: same set as the Hero scene; Codex is the "X" monogram.
- `prefers-reduced-motion` disables the motion; mobile uses a 2×2 grid.

---

# Execution

**Implemented.** The old "Deterministic Layer" section (`A manager still needs something real
to manage.` with the Manager / Fleet responsibility cards) was cut entirely: it described how
Fleet is built, not what it does for the user. Its point now surfaces through the guarantee
below. What remains is one section, `OrchestratorSection.svelte` (`id="orchestrator"`, with an
inner `id="ownership"` that the nav's "Execution" link targets). The file name is a leftover
from the old section.

## Parallel across projects. Serialized where code can collide.

Fleet deliberately keeps one active worker on one repository at a time.
Different repositories can move in parallel.
The same repository cannot be silently rewritten by two workers at once.

If an agent is running, waiting for human input, or needs review, the work is still owned.

Blocked is not done. Needs attention is not done. A stopped chat is not accepted work.

**Done means reviewed and accepted.**

The whole section is centered (title and text); the diagram spans full width.

Visual: three side-by-side project panels (`perceptrail`, `fleet`, `journal`) — a pure-CSS
animated backlog per project, same technique as the Hero scene. It replaces an earlier
progress-bar board (three repos with worker rows), which was dropped because it did not show
the queue.

- Panels are equal fixed height, lists top-aligned. Backlog depth differs per panel: 2, 3 and
  5 tasks.
- Each task row: `tag | #id title | status | agent`. Tag (`bug` / `feat` / `chore`) is a
  fixed column snapped right; title takes the free space, snapped left, truncated with an
  ellipsis; status and agent are fixed columns snapped right, so "Needs attention" never
  collides with the agent.
- Agent is an icon, not text: real Gemini, Claude and Cursor marks in muted brand colors, the
  same set as the Hero scene; Codex is the neutral "X" monogram (no open-source OpenAI mark).
  The agent is visible from the moment the task appears in the queue, not only when it runs.
- Tasks enter at the bottom of a panel's queue, step up one slot at a time, and the top slot
  is the active one: green border, a thin progress bar along the bottom edge, and a status
  that cycles Running → Needs attention → Review → Accepted. Queued rows are dimmed.
- Exactly one task per panel is ever active. Each panel's cycle is split into equal slot
  phases, so active windows of the staggered copies never overlap (an earlier version had two
  tasks stacking in the top slot).
- Each panel has its own character: `perceptrail` — one task breezes through, the next gets
  stuck on Needs attention; `fleet` — long Review; `journal` — tasks fly through fast with
  no blocking.
- `prefers-reduced-motion` falls back to a static snapshot.

Task numbers, titles and agents are invented placeholder content, not real data.

The point is safety, not maximal swarm throughput.

---

# Remote Human Input

**Hidden.** Not rendered on the page. The copy below is kept as reference only.

## Human input should not require sitting at the desk.

This is still the original reason Fleet exists.

If an agent needs me, Fleet should preserve enough state for me to see what is waiting, re-enter the right session, answer from the phone, and let the work continue.

The dashboard is still first-class when I am at the computer.

The manager is better when the input is rough, remote, or cross-project.

Both operate the same work.

Visual: Fleet attention state paired with a phone/remote-session cue if available. If not available, keep this section mostly typographic and do not fake a mobile product UI.

---

# Dashboard

## Chat when the input is rough. Dashboard when the work is exact.

The dashboard is not secondary.

It is where exact work gets operated directly: assign workers, change state, add context, launch tasks, review results, reopen, close, archive.

Use the strongest full Fleet screenshot here.

It should feel like the proof that all previous claims are real.

**Implemented.** The real screenshot stays static (`fleet-work-dashboard.png`, never
animated); title and text are centered. The screenshot is a fresh 2× capture (2880×1800, dark theme) of the running app, with the
Sessions strip switched to the **Closed** tab (there are no active sessions to show, but plenty
of closed ones). Six numbered callouts (badge, line and framed label in the soft mint `--good`
green, with a glow, sized relative to the image so they scale with it; white was rejected because
the UI already has white) fade in one after another when the section scrolls into view, driven by
a small `IntersectionObserver` (the only JS animation on the page; everything else is CSS). Each
points at the middle of its block. For the right-hand panels (04, 05) the number sits on the right,
the label on the left, ~10% in from the image edge:

- `01 sessions` — the Sessions strip (Closed tab);
- `02 projects` — the left sidebar project list;
- `03 current work` — the task list;
- `04 needs attention` — the Needs Attention list;
- `05 recent completed` — the Recent Completed block;
- `06 manager` — the chat input at the bottom, which is the manager.

Labels reuse the words from the animations above so the section reads as the payoff: the
same concepts, now as the real UI. Callouts overlay part of the screenshot; on mobile only
the numbers are shown. Without motion preference all callouts are visible immediately.

**One place to see projects, tasks, sessions, ownership, attention, and review.**

---

# Download

**Implemented; all pricing removed.** No free-trial, price, or "no credit card" copy is on the
page anymore (the App has no such mechanism yet). The product is presented as `fleet.eggs.gd`
(header brand, footer, page title) with a colored **Early access** badge in the header and next
to every Download button.

## Run Fleet against your local projects and agents.

Fleet works with your local projects and the coding agents you already use.

**Download** — `DownloadButton.svelte`, used in the hero and here:

- Fetches `releases/latest` of the App repo (`eggs-gd/fleet.eggs.gd`) client-side
  and picks the asset for the visitor's OS/arch. Label states what was detected, e.g.
  `Download for macOS · Apple silicon`. macOS defaults to Apple silicon (Intel is not
  reliably detectable in Safari/Firefox); Chromium's architecture hint is used when present.
- A caret opens a dropdown of every available build (macOS Apple silicon / Intel, Windows, Linux).
- If no release exists (or the API fails) the button is disabled and reads `Coming soon`.
- Expected asset names contain `macos|darwin`, `windows`, or `linux`, and `arm64|aarch64` (else
  amd64 is assumed); checksum/sbom files are ignored. Releases are published in this same App repository.
- The header "Download" button scrolls to this section.

Footer: **fleet.eggs.gd**

---

# Composition Rules

Animation temperature is deliberately different per section: Hero/Manager is lively and a bit chaotic; Execution is mechanical (queues, slots, state transitions); Workers is a calm routing loop; Dashboard is almost static.

This version should be meaningfully shorter than v2.

Target structure (as implemented):

- hero + manager (one section, `HeroSection.svelte`, `id="product"`);
- execution (`OrchestratorSection.svelte`, animated per-project backlogs);
- workers;
- ~~remote human input~~ (hidden for now);
- dashboard;
- download.

Avoid separate sections for every idea from v2. Merge them.

Specific compressions:

- `One manager`, `Task shaping`, and `Manager lifecycle` become **Manager** — and Manager itself
  was then merged straight into **Hero**, one step further than the original plan called for.
  There is no separate Manager section or heading anymore.
- `Orchestrator` was folded into **Execution**; the architecture-lecture framing was dropped.
- `Workers` and `Humans are workers too` become **Workers**.
- `Ownership` and `Blocked is not done` become **Execution** (now also holding the old Deterministic Layer point).
- `Remote HITL` and `Dashboard` remain separate only if the page rhythm benefits from it.
- `That's Fleet` is removed as a separate section. Its summary belongs in the hero.

Use real Fleet screenshots as evidence.

Use reconstructed diagrams only for concepts the product UI cannot show yet.

Do not fake controls.

Do not add generic SaaS polish, gradients, glass, glows, stock imagery, or AI illustrations.

The tone should be compact, causal, and technical:

**I needed this -> this created that problem -> Fleet handles it this way.**

The reader should understand the product without reading an RFC.
