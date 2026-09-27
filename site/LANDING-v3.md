# Fleet landing

The copy canon for the live page. Change this file before changing the page
structure in `src/lib/sections_v2`. Visual rules are in `DESIGN.md`.

Concept: remote agents lead to too many separate sessions, so one manager turns
rough intent into executable work. Fleet runs the deterministic layer
underneath, native agents and humans do the work, repository ownership keeps
execution safe, and review and human input stay explicit. Compress this story;
do not add a new one.

Header: **fleet.eggs.gd** brand, then **Manager**, **Execution**, **Workers**,
**Dashboard**, **Download**. An **Early access** badge sits next to every
Download button. There is no pricing, trial or "no credit card" copy anywhere.
Footer: **fleet.eggs.gd**. A back-to-top control appears after scrolling.

Page order: hero, manager loop, execution, workers, dashboard, download.

Animation temperature differs per section on purpose. The hero is lively and a
bit chaotic, the manager loop is a single guided sequence, workers is a calm
routing loop, execution is mechanical (queues, slots, state changes), and the
dashboard is almost static. All motion is CSS except the dashboard callouts,
which use one `IntersectionObserver`. `prefers-reduced-motion` shows a static
frame everywhere.

---

# Hero (`HeroSection.svelte`, `#product`)

## I wanted one manager for coding-agent work.

The page `<h1>`. Centered copy under it:

That is pretty much how Fleet started.

Remote sessions let me use coding agents from my phone.

It worked — until I had Codex in one project, Claude in another, Cursor
somewhere else, and my own memory acting as the orchestrator.

I didn't want remote access to a bunch of coding agents.

**I wanted one manager.**

The manager is an agent you already use. Fleet gave you that role: rough intent
in, executable work out, real agents underneath.

Under the copy, a full-width animated scene (pure CSS keyframes):

```text
chaotic input (voice note, screenshot, pasted chat, broken-icons note)
  -> chat panel
  -> Manager
  -> Backlog (four-slot conveyor)
  -> worker cards: Gemini, Claude, Codex, Cursor
```

- The input pieces loop from the chat into the pile and back into the manager.
- A "? which repo" chip goes to the chat input (amber, needs the user) and back
  (normal): the missing-context loop as motion.
- Task chips enter the bottom backlog slot, shift up, and fly from the top slot
  to the free worker card, landing where the card was measured. Task numbers
  are placeholders.
- Worker cards show an icon, a name and a status dot that changes as a task
  lands. Gemini, Claude and Cursor use their open-source marks; Codex uses a
  neutral "X" monogram.

Caption: A voice note, a screenshot, a half-formed sentence — the manager sorts
it into the right project, shapes it into a scoped task, and queues it for a
worker. If the project or the intent is unclear, it comes back with a question
instead of a guess.

**Three sentences in. Usable work out.**

Then the **Download** button with the **Early access** badge.

The scene is a labeled reconstruction, not a product screenshot.

---

# Manager loop (`ManagerLoopSection.svelte`)

## The manager is an agent you already use.

One animation. A chat message ("Add Windows support. Start with an
investigation if needed.") becomes a task in the backlog. A yellow card returns
as Needs Attention ("Worker needs a decision before continuing."). After the
reply ("Use native process APIs.") it flies back green and lands in Review.

**When the work needs you, it comes back to the same conversation.**

---

# Execution (`OrchestratorSection.svelte`, `#orchestrator`, inner `#ownership`)

## Parallel across projects. Serialized where code can collide.

Fleet deliberately keeps one active worker on one repository at a time.
Different repositories can move in parallel.
The same repository cannot be silently rewritten by two workers at once.

If an agent is running, waiting for human input, or needs review, the work is
still owned.

Blocked is not done. Needs attention is not done. A stopped chat is not
accepted work.

**Done means reviewed and accepted.**

Visual: three side-by-side project panels (`perceptrail`, `fleet`, `journal`),
each an animated backlog.

- Panels have equal height, with 2, 3 and 5 tasks.
- A row is `tag | #id title | status | agent`. The tag (`bug`, `feat`,
  `chore`), status and agent are fixed right-aligned columns, and the title
  takes the rest and truncates. The agent is an icon, visible from the moment
  the task enters the queue.
- Tasks enter at the bottom and step up. The top slot is the active one: green
  border, a thin progress bar, and a status that cycles Running, Needs
  attention, Review, Accepted. Queued rows are dimmed. Exactly one task per
  panel is active.
- Each panel has its own character: `perceptrail` has one task that stalls on
  Needs attention, `fleet` has a long Review, `journal` moves fast with no
  blocking.

Task numbers, titles and agents are placeholder content, not real data.

The point is safety, not maximal swarm throughput.

---

# Workers (`WorkersSection.svelte`, `#workers`)

## The agents are still the agents.

Fleet does not replace Codex, Claude Code, Cursor, Gemini, or any other coding
tool with another generic chat UI.

They remain native workers with different strengths, limits, subscriptions, and
launch behavior.

Humans are workers too: product decisions, architecture investigation, manual
unblocking, and review can sit in the same task graph as agent work.

**Human-in-the-loop is part of the workflow, not an exception path.**

Visual: a roster of four named workers, so agents never become anonymous
`worker-1..3`. Cards for Claude Code, Codex, Cursor and You, each with two role
lines, a status and a badge (`canonical ✓`, `active session`, `human`). No
invented benchmarks. Each worker has its own muted color, distinct from the
status colors. In a 16 s loop four tasks arrive one at a time and settle on
their worker; the status flips from `available` to `busy` as one lands. On
mobile the cards form a 2×2 grid.

---

# Dashboard (`DashboardSection.svelte`, `#dashboard`)

## Chat when the input is rough. Dashboard when the work is exact.

The dashboard is where exact work gets operated directly: assign workers,
change state, add context, launch tasks, review results, reopen, close,
archive.

The real screenshot (`fleet-work-dashboard.png`, 2880×1800, dark theme, the
Sessions strip on the Closed tab) stays static. Six numbered callouts in the
mint `--good` green fade in one after another when the section scrolls into
view. On mobile only the numbers show.

- `01 sessions` — the Sessions strip
- `02 projects` — the sidebar project list
- `03 current work` — the task list
- `04 needs attention` — the Needs Attention list
- `05 recent completed` — the Recent Completed block
- `06 manager` — the chat input at the bottom

**One place to see projects, tasks, sessions, ownership, attention, and
review.**

---

# Download (`InstallSection.svelte`, `#install`)

## Run Fleet against your local projects and agents.

Fleet works with your local projects and the coding agents you already use.

`DownloadButton.svelte` (in the hero and here) fetches `releases/latest` of the
App repository client-side and picks the asset for the visitor's OS and
architecture. The label says what was detected, for example `Download for macOS
· Apple silicon`. macOS defaults to Apple silicon, and Chromium's architecture
hint is used when present. A caret opens a list of every available build. With
no release, or when the API fails, the button is disabled and reads `Coming
soon`. Asset names contain `macos|darwin`, `windows` or `linux`, and
`arm64|aarch64` (otherwise amd64 is assumed); checksum and SBOM files are
ignored. The header Download link scrolls to this section.

The plan is to replace the button with an install command, see the release
roadmap in `_docs/roadmaps/release.md`.

---

# Rules

- Use real Fleet screenshots as evidence. Use reconstructed diagrams only for
  concepts the product UI cannot show yet. Do not fake controls.
- No generic SaaS polish, gradients, glass, stock imagery or AI illustrations.
- Tone: compact, causal, technical. *I needed this, this created that problem,
  Fleet handles it this way.* A reader should understand the product without
  reading an RFC.
- Not represented on the page: decomposition as thinking before implementation,
  and remote human input as its own section.
