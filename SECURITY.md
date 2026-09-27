# Security

## Reporting a problem

Please report a vulnerability privately, using **Report a vulnerability** on this
repository's Security tab, and not in a public issue.

## What Fleet is built to protect

Fleet runs on your computer and starts agents that can change your files. It has
three boundaries.

1. **The local API.** It listens on `127.0.0.1` only. Every request under `/api` and
   `/mcp` must carry the per-install token in `~/.fleet/launch-token`. A request
   with another `Host` or `Origin` is refused, so a web page you visit cannot drive
   Fleet, and bodies must be JSON. The token is stable across restarts and is
   written into the MCP entry Fleet gives your Manager, so keep that file private.
2. **Your files.** `~/.fleet/` is readable by your user only. The session logs of
   your agents are stored there.
3. **The agents.** In live mode an agent runs in the task's repository with the
   permissions listed in the README, and edits files without asking each time.
   Fleet starts in dry-run mode and starts nothing until you turn live mode on.

## What it cannot protect

**Prompt injection.** The text of a task, a comment, or an Inbox note goes to an
agent. If someone else can write that text (an email you forwarded, a web page, a
project synced from another tool), they can try to steer an agent that has write
access to your repository. Fleet marks the task text as data in the prompt and tells
the agent to stop and ask when the text asks for something unrelated to the task,
but no prompt makes that safe. Stay in dry-run mode, or review what agents did,
for work that started from text you do not trust.

**Other users on the machine** can reach the loopback port. The token stops them
only while they cannot read your `~/.fleet/`.

Fleet is not a sandbox, and it makes no promise about what the agents you run will
do. See each agent's own documentation for its sandbox.
