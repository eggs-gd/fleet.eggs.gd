---
name: shape-task
description: Shape an intent into a task with acceptance criteria.
---

Turn an intent into a goal, acceptance criteria, type, and priority. Do not invent a specification.

Priority runs from 1 (highest) to 5 (lowest) and defaults to 5. Status defaults to `backlog` until criteria exist. Allowed statuses: `backlog`, `needs_rework`, `todo`, `doing`, `blocked`, `needs_review`, `done`, `archived`. Types: `feature`, `bug`, `research`, `review`, `maintenance`, `decision`.

If the change is not ready to implement, do not pretend it is. Create one research task (`type=research`) that asks the open question. Do not create the implementation tasks yet: the research result says how to split the work, and the follow-up tasks are created from it.

Call `manager_validate` before writing. Fix every item it returns. Call `manager_similar` first so you do not create a duplicate.

Example: "make the web UI show invoice status" is not ready if nobody has said which states exist. Research asks that question. Implementation depends on the research ref and stays in `backlog`.

Fixing existing tasks. When the plan changes and old tasks no longer fit, page through `manager_board` with `detail=summary` (use `next_offset` to continue) and open a task with `manager_task` only when the summary is not enough. Correct it with `manager_update` (project, repository, `depends_on`, description), reprioritize or reassign with `manager_command`, and drop it with `kind=cancel` and `confirm=true`. Change only what the person asked for, and list every change you made at the end.
