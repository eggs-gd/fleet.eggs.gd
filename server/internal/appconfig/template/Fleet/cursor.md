# Cursor

## Best At

- Interactive IDE-driven code edits.
- Local exploratory implementation when the user wants to steer in-editor.
- Focused UI/code changes with immediate visual/context feedback.

## Use For

- `feature`
- `bug`
- editor-driven implementation tasks.

Only assign to Cursor when the user explicitly asks or a task/project
explicitly chooses Cursor. Cursor is now launchable by Core, but it is
CLI-visible rather than phone/app-visible like Claude remote-control.

## Avoid As Default For

- durable planning, _registry maintenance, or tasks that need to be fully handled
  from this Manager chat.

## Core Launcher

Core launches the local Cursor Agent CLI (`cursor-agent`), not the GUI
`cursor` command. The adapter resolves `cursor-agent` from `$PATH`, then
falls back to the newest versioned binary under
`~/.local/share/cursor-agent/versions/`.

Visibility class is `cli_visible`. A session Fleet starts this way stays on
this computer. It does not publish a phone remote-control URL. Phone
visibility, when the person wants it, is a manual `/remote-control` handoff
from Cursor's Agents window, not something Core starts.

Daemon contract:

- The plan never includes `--print`, `-p`, or `--output-format`.
- The runtime calls `cursor-agent create-chat` before starting the task prompt.
- The runtime starts the visible CLI session with
  `cursor-agent --resume <chat-id> --trust --workspace <repo-path> <task-prompt>`.
- Core stores `cursor_chat_id`, the operator command, process id, visibility
  mode `cli_visible`, and the session log path.
- The dashboard shows the Cursor chat id and resume command:
  `cursor-agent --resume <chat-id> --workspace <repo-path>`.
- `cursor-agent ls` lists sessions. Completion is process-exit based.
- Core sets the process cwd and also passes `--workspace`.
- Stdout and stderr go to `_registry/sessions/<claim_id>.log`.
- The daemon user must be authenticated to Cursor and able to write
  `~/.cursor/projects`.
- Core does not pass `--force` or `--yolo`. No auto-commit, auto-push, or
  auto-PR.
- `--print` is historical diagnostics only. It is not a daemon launch.
