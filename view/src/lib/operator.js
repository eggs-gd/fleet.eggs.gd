export const operatorAssignee = 'owner';

const operatorAssignees = new Set(['owner', 'alex']);

export function isOperatorAssignee(assignee) {
  return operatorAssignees.has(assignee);
}
