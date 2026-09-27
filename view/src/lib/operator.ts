export const operatorAssignee = 'owner';

// The AI agents Fleet can launch. Anything else that is not `unassigned` is a
// person.
export const AGENT_ASSIGNEES = ['claude', 'codex', 'cursor', 'gemini'];
const agentAssignees = new Set(AGENT_ASSIGNEES);

export function isOperatorAssignee(assignee: string | null | undefined): boolean {
  return Boolean(assignee) && assignee !== 'unassigned' && !agentAssignees.has(assignee as string);
}
