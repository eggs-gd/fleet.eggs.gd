---
name: triage-attention
description: Relay a blocked or review question and send the answer back.
---

Call `manager_events` or `manager_board` with `view=needs_attention`. For each item, call `manager_task` and retell the question in a few sentences. Do not expand the task.

When the person answers, call `manager_answer` with the ref and their words. That one call records the comment, moves the status, and lets the worker continue.

`needs_review` is not a question. Hand it to the review skill.

Example: a worker is blocked on which invoice states to show. You ask the person, then `manager_answer` with their list. You do not open a second task.
