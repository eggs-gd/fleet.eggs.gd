# Fleet

The page is one centered frame, 1120px wide. Keep the existing visual
system and section constructor. This document replaces the old generic
SaaS narrative with the actual story of why Fleet exists and how it
works.

The page should read like a technical product story, not
`problem → features → CTA`.

Each section advances the same causal chain:

**remote agents → too many separate sessions → one manager → intent
becomes executable work → the manager needs real machinery → Fleet
orchestrates workers safely → human-in-the-loop from anywhere**

Use real Fleet screenshots as evidence, not decoration.

The header sits above the frame: **Fleet**, **Why**, **Manager**,
**Workers**, **Execution**, **Download**.

------------------------------------------------------------------------

# Fleet

## I wanted to give my coding agents work from a park bench.

That is pretty much how Fleet started.

Remote sessions already let me reach coding agents when I wasn't at my
desk. So I started using them from my phone.

It worked.

For about five minutes.

Soon I had Codex working on one project, Claude on another, Cursor open
on a third --- and I had to remember who was doing what, find the right
session, explain context again, decide what should run next, and keep
checking whether anything got stuck.

I didn't want remote access to a bunch of coding agents.

**I wanted one manager.**

**Download Fleet**\
First 5 completed tasks free. Then \$9/month.

Right: a real manager interaction, preferably paired with a compact
Fleet view. Do not lead with a terminal. Do not lead with a generic
dashboard if the screenshot does not help explain the manager.

------------------------------------------------------------------------

# One manager

## All the projects it needs to understand.

I have roughly 80 projects on disk.

I don't want to choose a project every time I have an idea. And I
definitely don't want to load 80 repositories into one model context.

The manager gets a lightweight map of the projects Fleet knows about:
enough context to understand what each project is, what is happening
there, and where new work probably belongs.

Then I can send:

> The mobile layout breaks here.

Or a screenshot.

Or a voice note from outside.

Or a link and three half-formed sentences.

If the project is obvious, the manager routes the work there.

If it isn't, it asks.

**The manager is the front door to the work, not another project
selector.**

Visual: manager conversation on the left or top, project resolution /
created task on the right or below. Make the transition from rough input
to a specific project visible.

------------------------------------------------------------------------

# From rough intent to executable work

## I also got tired of writing three-page prompts for coding agents.

A coding agent can execute a good task remarkably well.

Writing the good task is still work.

Sometimes I know exactly what needs to happen. Sometimes I have a
screenshot and approximately three sentences worth of patience.

The manager uses project context to turn that rough intent into work an
executor can actually pick up: scope, relevant context, constraints,
expected result, dependencies, and a sensible worker.

When one task is enough, it creates one task.

When the work needs decomposition, it creates a chain.

When there is not enough information to specify implementation safely,
it does **not** have to invent the missing context.

It can create the missing thinking first.

Example visual:

``` text
Investigate current authentication flow
                ↓
Document constraints and proposed change
                ↓
Implement the change
                ↓
Verify migration
```

The implementation tasks stay locked by the investigation until the
prerequisite is complete.

That first task can go to an agent.

Or it can go to me.

**Three sentences in. A usable task graph out.**

Visual: real task dependency UI if available. If not, reconstruct only
the dependency graph; do not fake product controls that do not exist.

------------------------------------------------------------------------

# The manager runs the lifecycle too

Creating a task is only the beginning.

The same manager can operate the work after intake:

-   put something in the backlog or make it ready now;
-   choose a worker or use the configured default;
-   start a task;
-   add context, comments, screenshots, or links;
-   tell me what is running, blocked, or waiting for review;
-   reopen, close, or archive work;
-   choose the next ready tasks by priority and launch them.

If a choice is ambiguous, the manager can ask instead of guessing.

The point is not to replace the UI with chat.

The point is that I should be able to operate the same system whether I
am at the desk or holding a phone.

Visual: one manager conversation that performs more than task creation
--- for example, status → add context → launch next ready task.

------------------------------------------------------------------------

# A manager still needs something real to manage

The manager could understand what I wanted.

It could understand the projects.

It could formulate the work.

But somebody still had to find the right worker, launch the session,
attach it to the repository, keep track of ownership, react to state
changes, and make sure the next task did not start at the wrong time.

So I built the missing layer underneath it.

**That layer became Fleet.**

Fleet is the deterministic part of the system.

The manager deals with intent, context, decomposition, and decisions.

Fleet deals with projects, tasks, dependencies, workers, sessions,
repository ownership, and execution state.

Visual: this is the first place for the full real Fleet dashboard. It
should prove that there is an actual operating system underneath the
manager.

------------------------------------------------------------------------

# Workers

## The agents are still the agents.

Fleet does not try to replace Codex, Claude Code, Cursor, or Gemini with
another coding interface.

They remain real tools with different strengths, different
subscriptions, different limits, and different reasons to use them.

Fleet detects the agents available on the machine and validates their
installation.

A canonical installation is good to go.

A non-canonical installation gets a warning and guidance instead of
silently failing later.

Workers can also have roles and defaults, so the manager does not need
an explicit executor in every request.

Visual: real agent settings/discovery UI. Show canonical and warning
states if the current product UI supports them.

Fleet also tracks agent availability and quota state.

**Current:** Fleet can monitor that state.

**Next:** let the manager use it when assigning work --- because the
theoretically perfect worker is not useful if its quota is exhausted for
the next four hours.

Do not present quota-aware automatic routing as already shipped until it
is.

------------------------------------------------------------------------

# Humans are workers too

Fleet does not require every node in a task graph to be an AI agent.

I registered myself as a worker.

That means a project can contain a sequence like:

``` text
Human — decide product behaviour
              ↓
Claude — investigate affected architecture
              ↓
Codex — implement the change
              ↓
Human — review
```

If a feature needs product thinking before implementation, the manager
can assign that thinking to me and lock downstream work behind it.

I make the decision, attach the context or artifact, complete my task,
and the rest becomes executable.

The same mechanism works for any number of human workers.

**Human-in-the-loop is not an exception path. It is part of the task
graph.**

Visual: mixed human/agent dependency chain.

------------------------------------------------------------------------

# Execution

## One worker. One repository. One active task.

This is deliberate.

A lot of multi-agent tooling treats parallel development inside one
repository as the goal, then adds branches, worktrees, PRs, merge
coordination, and conflict resolution to make it survivable.

Fleet takes a different approach.

Most of my agent tasks take minutes, sometimes an hour. During that time
an agent can touch a surprisingly large part of a project.

I don't need two agents rewriting the same repository at once.

So while a worker owns an active task, Fleet holds the repository for
that work.

Another task cannot take the same repository until ownership is
released.

Different repositories are independent.

With roughly 80 projects, serialization inside a repository does not
mean serial execution across Fleet.

It means:

``` text
Repo A → Codex
Repo B → Claude
Repo C → Cursor
Repo D → Human
...
```

all at the same time, without manufacturing merge conflicts inside each
project.

**Parallel across projects. Serialized where the code can collide.**

Visual: ownership plaque / repository surface. Reuse the existing
Ownership visual language, but explain the invariant instead of
presenting it as a slogan.

------------------------------------------------------------------------

# Blocked is not done

This sounds obvious until an orchestrator gets it wrong.

Sometimes an agent cannot continue without a human decision.

That task is not finished.

The repository still belongs to it.

The session still belongs to it.

The next task must not start.

Fleet keeps ownership while the work is waiting for human input.

``` text
RUNNING
   ↓
NEEDS ATTENTION
   ↓
RUNNING
   ↓
NEEDS REVIEW
   ↓
DONE
```

Only the appropriate transition releases the work.

Otherwise "human in the loop" turns into two workers touching the same
repository while the first one is still waiting for an answer.

Visual: real task/session state if available. `Needs Attention` should
visibly retain worker/session/repository ownership.

------------------------------------------------------------------------

# Human in the loop doesn't mean human at the desk

This was the original reason for building the thing.

Fleet launches and tracks sessions in a way that lets the agent's own
remote-session experience remain useful.

If an agent needs me, I can get back into that session from my phone,
answer it, and let the work continue.

I don't need to remote-desktop into my development machine just to
answer one question.

Visual: desktop Fleet task paired with the corresponding session on a
phone.

Small note, if it fits the layout:

**Yes, getting a session launched from the command line to show up
correctly on the phone was annoying. It was also worth it.**

------------------------------------------------------------------------

# The dashboard is still first-class

Sometimes I don't want to talk to the manager.

I'm already at the computer. I know the project. I know the task. I know
who should run it.

So I use the dashboard.

Tasks can be created and operated directly: assign workers, change
state, add context, launch work, review results, reopen, close, archive.

The manager and the dashboard operate the same work.

One is convenient when the input is rough, remote, or cross-project.

The other is convenient when I already know exactly what I want.

Visual: whole Fleet dashboard, uncropped. This is the second and final
large product screenshot.

------------------------------------------------------------------------

# That's Fleet

It started because I wanted to give coding agents work from my phone.

Then managing separate agents became another job.

So I made one manager.

The manager needed enough context to understand all my projects.

Then it needed to turn rough intent into properly shaped work.

Then that work needed dependencies, workers, sessions, ownership, state,
review, and human intervention.

And all of that needed something deterministic underneath it so the
intelligent part could not accidentally turn execution into chaos.

So I built Fleet.

**One manager above the work.\
Real coding agents underneath.\
Fleet in between keeping execution honest.**

------------------------------------------------------------------------

# Run Fleet locally

First **5 completed tasks** are free.

Failed, cancelled, or merely created tasks do not consume the trial.

## \$9 / month

**Download Mac** · Linux · Windows

Fleet works with your local projects and the coding agents you already
use.

No credit card to try it.

No terminal screenshot in this section. The primary action is the
download.

Under the next divider is the page footer.

For now:

**Fleet**

------------------------------------------------------------------------

# Composition

Keep the existing theme tokens in `src/styles.css` (`:root`): colors
plus type/space scales `s` / `m` / `l` / `xl`.

Keep the layout constructor in `src/lib/layout/`.

The landing should continue to be composed from reusable section
primitives rather than one-off page CSS:

-   `Section` --- tag → band → body → footer
-   `SectionTag`
-   `SectionTitle`
-   `SectionSubtitle`
-   `SectionText`
-   `SectionFoot`
-   `Split`
-   `Stack`
-   `ScaleLock`

The current visual system is useful. The old narrative is not.

Sections may be renamed or regrouped to fit the new story. Do not
preserve the old section count merely because the components already
exist.

Suggested page-level sections:

-   `HeroSection`
-   `ManagerSection`
-   `TaskShapingSection`
-   `ManagerLifecycleSection`
-   `OrchestratorSection`
-   `WorkersSection`
-   `HumanWorkerSection`
-   `OwnershipSection`
-   `NeedsAttentionSection`
-   `RemoteHitlSection`
-   `DashboardSection`
-   `CloseSection`

Several narrative sections may share the same underlying visual
component.

Do not make every section a full-screen marketing panel. The page should
feel like one continuous technical story with alternating prose and
proof.

Use this rhythm:

**2--5 short paragraphs → concrete mechanism → screenshot / diagram →
next problem**

------------------------------------------------------------------------

# Visual rules

The existing restrained visual direction stays:

-   technical, mature, slightly industrial;
-   dense enough to feel like an operator tool;
-   thin rules;
-   small radii;
-   no gradients;
-   no glow;
-   no glass;
-   no stock imagery;
-   no fake metrics;
-   no generic AI illustrations;
-   no giant empty centered hero;
-   no terminal as the product identity.

The page frame remains approximately 1120--1280px on desktop and must
still work at 1440, 1024, and 390 widths.

Real Fleet UI is evidence. Use screenshots where they demonstrate a
claim.

Reconstructed diagrams are acceptable for concepts such as task
dependencies or repository ownership, but they must not imply controls
or features that do not exist.

Do not repeat the same dashboard screenshot in five sections.

------------------------------------------------------------------------

# Copy rules

Do not fall back to generic SaaS language.

Avoid phrases like:

-   "Keep your..."
-   "No need to..."
-   "Unlock..."
-   "Supercharge..."
-   "AI-powered..."
-   "Seamlessly..."
-   "Control plane" as the main explanation of the product.

Technical terms are fine when they explain an actual mechanism.

Prefer causal language:

**I wanted X → that created problem Y → so I built Z → then Z exposed
the next problem.**

The landing should explain **why each mechanism exists**, not merely
name the mechanism.

The manager is the main differentiator and must appear before
orchestration internals.

The dashboard is first-class, but it is not the product's opening
explanation.

Do not present Fleet as another chat UI.

Do not present Fleet as another IDE.

Do not present Fleet as a generic project-management system.

Do not promise a specific persistence implementation. In particular, do
not mention Markdown, plain files, directories, SQL, or any other
storage mechanism as part of the public product contract.

Historical origin may mention "a few scripts" or "a simple local
orchestrator" if useful, but current internal storage is not marketing
copy.

Clearly distinguish implemented behaviour from roadmap behaviour,
especially quota-aware worker assignment.

The final page should make a technically experienced reader understand,
without reading separate docs:

1.  why Fleet exists;
2.  why the manager matters;
3.  how rough intent becomes executable work;
4.  how the manager differs from the deterministic orchestrator;
5.  what workers are;
6.  why humans can be workers;
7.  why repository execution is deliberately serialized;
8.  what happens when an agent needs human input;
9.  how remote sessions fit the original use case;
10. why the dashboard still exists.
