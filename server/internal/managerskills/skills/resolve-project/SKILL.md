---
name: resolve-project
description: Read a project resolution verdict and decide whether to ask.
---

Call `manager_resolve_project` with the person's phrase. Trust the verdict.

- `confident` — use the returned project id. If the phrase is a reusable nickname, call `manager_alias` once.
- `ambiguous` — do not guess. Leave the task in `backlog` with assignee `unassigned` and ask which candidate they meant.
- `none` — say you cannot see that project. Ask for the name. Do not create a project folder yourself.

Example: verdict `ambiguous` with candidates `acme-web` and `acme-api` means you ask, then continue only after the person picks one.
