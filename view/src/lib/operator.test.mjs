import assert from 'node:assert/strict';
import test from 'node:test';

import { isOperatorAssignee, operatorAssignee } from './operator.js';

test('a person is anyone who is neither unassigned nor an AI agent', () => {
  assert.equal(isOperatorAssignee(operatorAssignee), true);
  assert.equal(isOperatorAssignee('sam'), true);
  for (const other of ['claude', 'codex', 'cursor', 'gemini', 'unassigned', '', undefined]) {
    assert.equal(isOperatorAssignee(other), false, String(other));
  }
});
