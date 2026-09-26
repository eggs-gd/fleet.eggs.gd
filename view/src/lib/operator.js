export const operatorAssignee = 'owner';

// Anything that is neither unassigned nor an AI agent is a person.
const agentAssignees = new Set(['claude', 'codex', 'cursor', 'gemini']);

export function isOperatorAssignee(assignee) {
  return Boolean(assignee) && assignee !== 'unassigned' && !agentAssignees.has(assignee);
}
