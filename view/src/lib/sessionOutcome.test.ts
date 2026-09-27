import assert from 'node:assert/strict';
import test from 'node:test';

import {
  elapsedClockPrecise,
  resolveOpenSession,
  sessionOutcomeLabel,
  sessionOutcomeTone
} from './sessionOutcome.ts';

test('blocked / needs_input / needs_rework are HITL, not fail', () => {
  assert.equal(
    sessionOutcomeTone({ execution_status: 'succeeded', result: { outcome: 'blocked' } }),
    'hitl'
  );
  assert.equal(
    sessionOutcomeTone({ execution_status: 'succeeded', result: { outcome: 'needs_input' } }),
    'hitl'
  );
  assert.equal(
    sessionOutcomeTone({ execution_status: 'succeeded', result: { outcome: 'needs_rework' } }),
    'hitl'
  );
  assert.equal(
    sessionOutcomeLabel({ execution_status: 'succeeded', result: { outcome: 'blocked' } }),
    'Blocked'
  );
  assert.equal(
    sessionOutcomeLabel({ execution_status: 'succeeded', result: { outcome: 'needs_input' } }),
    'HITL'
  );
  assert.equal(
    sessionOutcomeLabel({ execution_status: 'succeeded', result: { outcome: 'needs_rework' } }),
    'Needs rework'
  );
});

test('live waiting_input is HITL even without a worker outcome', () => {
  assert.equal(sessionOutcomeTone({ execution_status: 'waiting_input' }, 'live'), 'hitl');
  assert.equal(sessionOutcomeLabel({ execution_status: 'waiting_input' }, 'live'), 'HITL');
});

test('completed / succeeded / released are success', () => {
  assert.equal(
    sessionOutcomeTone({ execution_status: 'succeeded', result: { outcome: 'completed' } }),
    'success'
  );
  assert.equal(sessionOutcomeTone({ execution_status: 'released' }), 'success');
  assert.equal(
    sessionOutcomeLabel({ execution_status: 'succeeded', result: { outcome: 'completed' } }),
    'Succeeded'
  );
  assert.equal(sessionOutcomeLabel({ execution_status: 'released' }), 'Released');
});

test('failed worker outcome and provider errors are fail', () => {
  assert.equal(
    sessionOutcomeTone({ execution_status: 'succeeded', result: { outcome: 'failed' } }),
    'fail'
  );
  assert.equal(sessionOutcomeTone({ execution_status: 'provider_error' }), 'fail');
  assert.equal(sessionOutcomeTone({ execution_status: 'failed' }), 'fail');
  assert.equal(sessionOutcomeTone({ execution_status: 'terminal' }), 'fail');
  assert.equal(sessionOutcomeLabel({ execution_status: 'provider_error' }), 'Provider error');
  assert.equal(
    sessionOutcomeLabel({ execution_status: 'succeeded', result: { outcome: 'failed' } }),
    'Failed'
  );
});

test('running live sessions stay live unless HITL or fail', () => {
  assert.equal(sessionOutcomeTone({ execution_status: 'running' }, 'live'), 'live');
  assert.equal(sessionOutcomeLabel({ execution_status: 'running' }, 'live'), 'Running');
  assert.equal(
    sessionOutcomeTone({ execution_status: 'running', result: { outcome: 'blocked' } }, 'live'),
    'hitl'
  );
});

test('orphans are their own tone', () => {
  assert.equal(sessionOutcomeTone({ execution_state: 'orphaned' }, 'orphan'), 'orphan');
  assert.equal(sessionOutcomeLabel({ execution_state: 'orphaned' }, 'orphan'), 'Orphaned');
});

test('elapsedClockPrecise includes seconds', () => {
  const started = '2026-09-12T12:00:00.000Z';
  const now = Date.parse('2026-09-12T12:01:07.000Z');
  assert.equal(elapsedClockPrecise(started, now), '01:07');
});

test('resolveOpenSession follows a live session into closed after it finishes', () => {
  const selected = {
    key: 'claim-1',
    kind: 'live',
    snapshot: { claim_id: 'claim-1', execution_status: 'running' }
  };
  const live = [{ claim_id: 'claim-1', execution_status: 'running', last_event: 'working' }];
  const opened = resolveOpenSession(selected, live, [], [])!;
  assert.equal(opened.kind, 'live');
  assert.equal(opened.session.last_event, 'working');

  const closed = [
    { claim_id: 'claim-1', execution_status: 'succeeded', result: { outcome: 'completed' } }
  ];
  const finished = resolveOpenSession(selected, [], closed, [])!;
  assert.equal(finished.kind, 'closed');
  assert.equal(finished.session.result!.outcome, 'completed');
});
