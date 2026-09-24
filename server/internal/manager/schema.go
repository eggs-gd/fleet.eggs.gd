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
    "repository": { "type": "string" },
    "title": { "type": "string" },
    "description": { "type": "string" },
    "priority": { "type": "integer", "minimum": 1, "maximum": 5 },
    "status": {
      "type": "string",
      "enum": ["backlog", "needs_rework", "todo", "doing", "blocked", "needs_review", "done", "archived"]
    },
    "type": {
      "type": "string",
      "enum": ["feature", "bug", "research", "review", "maintenance", "decision"]
    },
    "assignee": {
      "type": "string",
      "enum": ["alex", "codex", "claude", "cursor", "gemini", "unassigned"]
    },
    "ref": {
      "type": "string",
      "pattern": "^(CORE|INBOX|LIFE)-[0-9]+$"
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
