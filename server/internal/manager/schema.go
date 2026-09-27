package manager

// IntentJSONSchema is the Structured Output schema for Manager LLM responses.
// Served at GET /api/manager/schema.
const IntentJSONSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://fleet.eggs.gd/schemas/manager-intent.json",
  "title": "CoreManagerIntent",
  "type": "object",
  "additionalProperties": false,
  "required": ["kind"],
  "properties": {
    "kind": {
      "type": "string",
      "enum": [
        "task",
        "status_change",
        "comment",
        "board_command",
        "question",
        "assignee_change",
        "priority_change",
        "cancel"
      ]
    },
    "project": { "type": "string" },
    "repository": { "type": "string", "description": "Single repository. Use repositories when a task touches more than one." },
    "repositories": { "type": "array", "items": { "type": "string" } },
    "depends_on": { "type": "array", "items": { "type": "string" }, "description": "Prerequisite task refs. The task cannot start until each one is done." },
    "title": { "type": "string" },
    "description": { "type": "string", "description": "What to do, in the person's words made precise. Not the raw input." },
    "context": { "type": "string", "description": "What the worker needs beyond the request: why it is wanted, limits, and any decision already made. The worker sees only the task, so put the reasoning here." },
    "acceptance_criteria": { "type": "string", "description": "Required before a task leaves backlog. Empty criteria fail manager_validate." },
    "source_inbox": { "type": "string", "description": "INBOX ref this task was promoted from." },
    "priority": { "type": "integer", "minimum": 1, "maximum": 5, "description": "1 is highest, 5 is lowest. Defaults to 5 when omitted." },
    "status": {
      "type": "string",
      "description": "Allowed: backlog, needs_rework, todo, doing, blocked, needs_review, done, archived. Defaults to backlog.",
      "enum": ["backlog", "needs_rework", "todo", "doing", "blocked", "needs_review", "done", "archived"]
    },
    "type": {
      "type": "string",
      "enum": ["feature", "bug", "research", "review", "maintenance", "decision"]
    },
    "assignee": {
      "type": "string",
      "description": "unassigned, an agent (claude, codex, cursor, gemini), or a person listed as Fleet/<name>.md."
    },
    "ref": {
      "type": "string",
      "pattern": "^[A-Z][A-Z0-9]{1,5}-[0-9]+$"
    },
    "comment": { "type": "string" },
    "comment_author": { "type": "string" },
    "board_action": {
      "type": "string",
      "enum": ["show_board", "show_task"]
    },
    "query": { "type": "string" },
    "confirm": { "type": "boolean" },
    "raw_transcript": { "type": "string" }
  },
  "allOf": [
    {
      "if": { "properties": { "kind": { "const": "task" } } },
      "then": { "required": ["project", "title", "description"] }
    },
    {
      "if": { "properties": { "kind": { "const": "status_change" } } },
      "then": { "required": ["ref", "status"] }
    },
    {
      "if": { "properties": { "kind": { "const": "comment" } } },
      "then": { "required": ["ref", "comment"] }
    },
    {
      "if": { "properties": { "kind": { "const": "board_command" } } },
      "then": { "required": ["board_action"] }
    },
    {
      "if": { "properties": { "kind": { "const": "assignee_change" } } },
      "then": { "required": ["ref", "assignee"] }
    },
    {
      "if": { "properties": { "kind": { "const": "priority_change" } } },
      "then": { "required": ["ref", "priority"] }
    },
    {
      "if": { "properties": { "kind": { "const": "cancel" } } },
      "then": { "required": ["ref", "confirm"] }
    }
  ]
}`
