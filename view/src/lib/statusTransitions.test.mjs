import assert from 'node:assert/strict';
import test from 'node:test';

import { actionsForStatus, canTransitionStatus, STATUS_TRANSITIONS } from './statusTransitions.js';

test('STATUS_TRANSITIONS matches the Go operator map', () => {
  assert.deepEqual(STATUS_TRANSITIONS.needs_review, ['needs_rework', 'todo', 'done', 'archived']);
  assert.deepEqual(STATUS_TRANSITIONS.done, ['archived']);
  assert.deepEqual(STATUS_TRANSITIONS.blocked, ['needs_review', 'needs_rework', 'todo', 'archived']);
  assert.deepEqual(STATUS_TRANSITIONS.archived, ['backlog']);
  assert.equal(canTransitionStatus('blocked', 'done'), false);
  assert.equal(canTransitionStatus('needs_review', 'done'), true);
});

test('review plaques offer complete, to work, archive', () => {
  const actions = actionsForStatus('needs_review');
  assert.deepEqual(
    actions.map((action) => [action.status, action.label, action.icon]),
    [
      ['done', 'Complete', 'checks-circle'],
      ['todo', 'To work', 'play'],
      ['archived', 'Archive', 'archive']
    ]
  );
});

test('completed plaques offer archive only', () => {
  const actions = actionsForStatus('done');
  assert.equal(actions.length, 1);
  assert.equal(actions[0].status, 'archived');
  assert.equal(actions[0].icon, 'archive');
});

test('doing and needs_rework are not one-click plaque actions', () => {
  assert.ok(STATUS_TRANSITIONS.todo.includes('doing'));
  assert.ok(STATUS_TRANSITIONS.needs_review.includes('needs_rework'));
  assert.equal(
    actionsForStatus('todo').some((action) => action.status === 'doing'),
    false
  );
  assert.equal(
    actionsForStatus('needs_review').some((action) => action.status === 'needs_rework'),
    false
  );
});

test('API transition map overrides the fallback', () => {
  const actions = actionsForStatus('done', { done: ['todo'] });
  assert.deepEqual(
    actions.map((action) => action.status),
    ['todo']
  );
});
