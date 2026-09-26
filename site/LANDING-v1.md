# Fleet

The page is one centered frame, 1120px wide. Each block has a tag at the top-left
of the block (base eyebrow style, or an override like Problem). Tags are not glued
to the headline.

The header sits above the frame: Fleet, Problem, Workflow, Ownership, Install, and Download.

---

# Local control plane for coding agents

## Keep every agent attached to the work it owns.

The headline is full width, above the two columns.

Left:

Fleet is a local control plane for coding-agent work across real projects and
repositories.

Use Codex, Claude Code, Cursor, Gemini, and the tools you already use. Fleet
keeps the work around them durable: what needs to be done, who owns it, what is
running, what is waiting, and what needs your review.

**Download Fleet**  
View install notes

First 5 completed tasks free. Then $9/month.

Right: the real Fleet dashboard, cropped. This is one of the two places the full
product UI appears. The other is Product proof, where the same screen is shown
whole.

---

# Agent chats are easy to start.
# They're hard to operate.

No tag on this block.

**What is it working on?**  
Which repository can it touch?

**Is something already running there?**  
Can another agent start safely?

**Did it actually finish?**  
Where is the result?

**Who reviewed it?**  
Is the work done, or did the agent merely stop talking?

Fleet keeps those answers outside the chat. The agents stay in their native tools.
The work gets a durable operating layer.

---

# Operating loop

## Fleet turns rough input into work that can survive beyond a conversation.

The old headline "Capture. Shape. Launch. Review." is not used. Those four names
are the table.

The table and the picture are full width of the frame.

### 01 — raw input — Capture

Start with a voice note, sentence, link, or half-formed idea.

### 02 — ready task — Shape

Attach the work to a project, repository, owner, dependencies, and priority.

### 03 — safe run — Launch

Check ownership, active sessions, and repository conflicts before handoff.

### 04 — accepted done — Review

Bring artifacts back for human acceptance, rework, or unblock.

Under the table: a real task as it hangs in the list. Do not open it.

Done means accepted by a human. No silent swarm touching the same codebase.

---

# Native tools

## Your agents stay your agents.

Fleet does not replace Codex, Claude Code, Cursor, or Gemini. It doesn't hide
them behind another generic chat interface.

Coding stays where it already works best. Fleet sits around those tools and
coordinates the work between them.

**Native tools stay native. Fleet keeps the operating state.**

Right: four reconstructed agent cards — Claude Code, Codex, Cursor, Gemini CLI.
No rule above the closing line.

---

# Ownership

Plaque on the left, three lines on the right. They are one locked group and scale
together, so the gaps between the lines stay fixed.

**One repository.**  
**One active task.**  
**One owner.**

The plaque is one repository surface: perceptrail is Active, Codex is Running,
Claude Code is Blocked. "Repository currently owned by Codex."

---

# Durable work

## Nothing disappears when the chat ends.

Tasks, decisions, context, results, and review state survive the conversation that
created them.

Right: one finished agent run, not a storage diagram.

- Task #124, Needs review
- Add repository ownership gates
- perceptrail · Codex · feat/repository-ownership
- Work produced: launcher.go, launcher_test.go, Launch policy, Repository ownership
- Agent finished 12 min ago
- Review changes

Do not mention persistence or storage architecture (files, databases, directories,
formats) unless it is part of Fleet's public contract.

---

# Product proof

## From one task to many projects.

Fleet gives you one place to see projects, repositories, dependencies, active
sessions, work waiting for review, and agent availability.

The headline, the sentence, the screenshot, the line under it, and the chips are
all full width of the frame and aligned left.

**The control plane between your intent and the agents doing the work.**

Not another IDE.  
Not another model.  
Not another cloud project-management suite.

The screenshot is the whole Fleet dashboard, not cropped. This is the place the
landing shows the entire product.

---

# Run Fleet locally

## Run coding agents without losing track of the work.

No line under the headline. This block is the close of the page. The old final
screen is not a separate section.

Left, in one box:

First 5 completed tasks are free.

**$9 / month**

**Download Mac** · Linux · Windows

Right:

Fleet runs against your local projects and data. Connect the coding agents you
already use and start with real work.

No install command. No "Read install notes."

Under the next divider is the page footer. For now it only says Fleet. The links
are not decided yet.

---

# Composition

Theme tokens live in `src/styles.css` (`:root`): colors plus type/space scales
`s` / `m` / `l` / `xl` (`--type-title-*`, `--type-subtitle-*`, `--type-text-*`,
`--type-tag-*`, `--space-*`).

Layout constructor lives in `src/lib/layout/`. Each landing block is a section
component in `src/lib/sections/` that composes `Section` (Svelte has no class
extends — OwnershipSection wraps Section, etc.):

- `Section` — tag → band → body → footer
- `SectionTag` / `SectionTitle` / `SectionSubtitle` / `SectionText` / `SectionFoot` — type roles with `size="s|m|l|xl"` (default `m`); sections pick sizes via props, not local `font-size` on those roles
- `Split` / `Stack` — body layouts (`flat` Split keeps children in the section grid; `gap` accepts a size token or raw CSS)
- `ScaleLock` — ownership zoom canvas

Section components: `HeroSection`, `ProblemSection`, `OperatingLoopSection`,
`NativeToolsSection`, `OwnershipSection`, `DurableWorkSection`,
`ProductProofSection`, `InstallSection`. The page only stacks them.

Roles: **title**, **subtitle**, **text** (body copy), **foot** (CTA / closing line).
Title and/or subtitle sit in **band** or in **body**. Foot may be the section
`footer` slot or an in-column `SectionFoot`.

- Hero: tag m, title l, subtitle l, text l; Split(copy | cropped dashboard).
- Problem: muted display title m + title l, Stack(questions), footer statement xl.
- Operating loop: tag m, title s, Stack(pipeline, task row), footer note xl.
- Native tools: tag m, title/subtitle/text s, Split + agent cards.
- Ownership: tag m, title xl (cqi override below desktop), ScaleLock + plaque.
- Durable work: tag m, title l + subtitle s, Split + run record.
- Product proof: tag/title/subtitle m, shot, footer statement xl + chips.
- Close: tag m, title s, price box + lead subtitle l. Page footer underneath.

Real Fleet UI is only the hero crop, the operating-loop task row, and the product
proof dashboard. The other plaques are reconstructed.

Keep the technical details in `CONTENT.md` and docs. This page sells the product
position, not the implementation.
