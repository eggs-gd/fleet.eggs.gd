import assert from 'node:assert/strict';
import test from 'node:test';

import {
  capabilityRows,
  launchesAutomatically,
  plainText,
  reuseSummary
} from './agentCapabilities.ts';

const gemini = {
  live_ready: true,
  enabled: { value: true },
  capabilities: {
    can_show_app_visible_link: 'no',
    can_accept_operator_input: 'yes',
    can_resume_session: 'yes',
    can_list_sessions: false
  },
  session_reuse: { level: 'verified', can_attempt: true, reason: 'ok' }
};

test('capabilityRows names every flag and marks missing ones unknown', () => {
  const rows = capabilityRows(gemini);
  const byKey = Object.fromEntries(rows.map((row) => [row.key, row]));
  assert.equal(byKey.can_show_app_visible_link.value, 'no');
  assert.equal(byKey.can_accept_operator_input.value, 'yes');
  assert.equal(byKey.can_list_sessions.value, 'no');
  assert.equal(byKey.can_query_thread_status.value, 'unknown');
  assert.equal(byKey.can_query_thread_status.tone, 'warn');
  assert.ok(rows.every((row) => row.label.length > 10));
});

test('capabilityRows tolerates an agent without capabilities', () => {
  assert.equal(capabilityRows({}).length, 7);
  assert.equal(capabilityRows(undefined).length, 7);
});

test('reuseSummary explains the level in words', () => {
  assert.equal(reuseSummary(gemini).label, 'Confirmed to continue the same session');
  assert.equal(
    reuseSummary({ session_reuse: { level: 'unverified' } }).label,
    'Attempted, not confirmed end to end'
  );
  assert.equal(reuseSummary({}).level, 'unsupported');
});

test('launchesAutomatically needs a ready adapter that is enabled', () => {
  assert.equal(launchesAutomatically(gemini), true);
  assert.equal(launchesAutomatically({ ...gemini, enabled: { value: false } }), false);
  assert.equal(launchesAutomatically({ ...gemini, live_ready: false }), false);
});

test('plainText drops the backticks the server puts around commands', () => {
  assert.equal(plainText('run `agy --conversation <id>` now'), 'run agy --conversation <id> now');
  assert.equal(plainText(undefined), '');
});
