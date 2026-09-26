---
name: review
description: Accept work or send it back for rework.
---

Call `manager_task` and compare the result to the acceptance criteria in the task body. Do not accept because the worker says it is done.

Call `manager_review`:

- `verdict=accept` when every criterion is met. That sets `done` and releases ownership.
- `verdict=rework` otherwise. That sets `needs_rework` and stores your comment on the task.

The comment names the missed criterion. It does not redesign the task.

Example: the criterion is "invoice status is visible". A change that only adds a button fails. Send it back with that sentence.
