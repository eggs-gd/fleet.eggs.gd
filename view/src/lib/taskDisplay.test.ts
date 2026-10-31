import assert from 'node:assert/strict';
import test from 'node:test';

import {
  agentColorClass,
  latestCommentText,
  operatorPause,
  parseDependsOnDraft
} from './taskDisplay.ts';
import type { Task } from './types.ts';

const doing = (text: string): Task =>
  ({ status: 'doing', comments: [{ text }] }) as unknown as Task;

test('a paused execution state is waiting with its reason as the question', () => {
  const task = {
    status: 'doing',
    execution: { state: 'waiting_input', blocking_reason: 'Which database?' }
  } as unknown as Task;
  assert.deepEqual(operatorPause(task), { waiting: true, question: 'Which database?' });
});

test('the question is read from the latest comment, up to the next section', () => {
  const task = doing(
    'Waiting on operator input. Question: Use sqlite or postgres?   Artifacts: a.txt Session log: x'
  );
  assert.deepEqual(operatorPause(task), { waiting: true, question: 'Use sqlite or postgres?' });
  const last = doing('Waiting on operator input. Question: Ship it? Tests/checks: none');
  assert.equal(operatorPause(last).question, 'Ship it?');
  const noSection = doing('waiting on operator input. Question: open end');
  assert.equal(operatorPause(noSection).question, 'open end');
});

test('without a Question marker the whole comment is the question', () => {
  const task = doing('Waiting on operator input, please advise');
  assert.equal(operatorPause(task).question, 'Waiting on operator input, please advise');
});

test('other tasks are not paused', () => {
  assert.equal(operatorPause(doing('just a note')).waiting, false);
  assert.equal(
    operatorPause({
      status: 'todo',
      comments: [{ text: 'waiting on operator input' }]
    } as unknown as Task).waiting,
    false
  );
});

test('latestCommentText skips blank comments', () => {
  const task = { comments: [{ text: 'first' }, { text: '  ' }] } as unknown as Task;
  assert.equal(latestCommentText(task), 'first');
  assert.equal(latestCommentText(null), '');
});

test('dependency drafts split on commas and newlines and drop repeats', () => {
  assert.deepEqual(parseDependsOnDraft('A-1, B-2\nA-1\n\n C-3 '), ['A-1', 'B-2', 'C-3']);
  assert.deepEqual(parseDependsOnDraft(null), []);
});

test('only the known agents get their own color', () => {
  assert.equal(agentColorClass('Claude'), 'agent-claude');
  assert.equal(agentColorClass('mystery'), 'agent-other');
  assert.equal(agentColorClass(undefined), 'agent-other');
});
