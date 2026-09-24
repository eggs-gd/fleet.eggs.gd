# Fleet

Workers and execution surfaces.

Fleet describes who or what can do work:

- humans;
- Codex;
- Claude;
- Cursor;
- Gemini;
- local scripts;
- future specialized agents.

Fleet does not own tasks. Work items point to Fleet members through assignment.

## Files

- `ROUTING.md` — task category and assignment policy.
- `LAUNCH_POLICY.md` — launch, lock, and concurrency policy.
- `codex.md` / `claude.md` / `cursor.md` / `gemini.md` — agent capability
  profiles.
- `owner.md` — the human operator's profile. Rename it (and the `owner`
  assignee value) to your own name/slug if you prefer.
