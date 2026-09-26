---
name: route
description: Choose a worker when routing confidence is low.
---

Call `manager_route` with the draft. If it returns a worker and `confident=true`, assign that worker.

If `confident=false` or it reports a conflict, do not flip a coin. Tell the person the options and why, then wait. Record the choice with `manager_command` `kind=assignee_change` and a one-line reason in `comment`.

A direct instruction from the person beats the recommendation.

Example: the tool says claude is preferred but the person already said "give this to Codex". Assign `codex` and say that in the handoff comment.
