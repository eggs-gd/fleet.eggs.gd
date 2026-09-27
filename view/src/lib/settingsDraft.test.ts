import assert from 'node:assert/strict';
import test from 'node:test';

import { draftFromSnapshot, draftsEqual, patchFromDraft } from './settingsDraft.ts';

test('draft keeps overlay routing only', () => {
  const draft = draftFromSnapshot({
    projects: { scan_roots: ['/tmp/Projects'] },
    agents: {
      providers: [
        {
          id: 'codex',
          enabled: { value: true },
          configured_executable: { value: '/opt/codex' },
          routing_instructions: { value: 'Fleet text', source: 'fleet' }
        }
      ]
    }
  });
  assert.deepEqual(draft.scanRoots, ['/tmp/Projects']);
  assert.equal(draft.agents!.codex.executable, '/opt/codex');
  assert.equal(draft.agents!.codex.routingInstructions, '');
});

test('patchFromDraft emits only changed overlay keys', () => {
  const baseline = {
    scanRoots: ['/a'],
    agents: { codex: { enabled: true, executable: '', routingInstructions: '' } }
  };
  const draft = {
    scanRoots: ['/b'],
    agents: { codex: { enabled: false, executable: '', routingInstructions: '' } }
  };
  assert.equal(draftsEqual(draft, baseline), false);
  assert.deepEqual(patchFromDraft(draft, baseline), {
    scanRoots: ['/b'],
    agents: { codex: { enabled: false } }
  });
});

test('draft reads the bound manager session from the snapshot', () => {
  const draft = draftFromSnapshot({
    manager: { provider: 'codex', session: { id: 'thread-1', status: 'bound' } }
  });
  assert.deepEqual(draft.manager, { agent: 'codex', threadId: 'thread-1' });
});

test('draft leaves manager unbound when the snapshot has no session', () => {
  const draft = draftFromSnapshot({
    manager: { provider: 'none (HTTP Manager API)', session: null }
  });
  assert.deepEqual(draft.manager, { agent: '', threadId: '' });
});

test('patchFromDraft emits manager only when the binding changes', () => {
  const baseline = { manager: { agent: '', threadId: '' } };
  const unchanged = { manager: { agent: '', threadId: '' } };
  assert.deepEqual(patchFromDraft(unchanged, baseline), {});

  const bound = { manager: { agent: 'claude', threadId: 'abc-123' } };
  assert.deepEqual(patchFromDraft(bound, baseline), {
    manager: { agent: 'claude', threadId: 'abc-123' }
  });
});

test('launch mode is drafted from the snapshot and only patched when it changes', () => {
  const snapshot = {
    general: { launch_config: { value: 'live' } },
    projects: {},
    agents: {},
    manager: {}
  };
  const baseline = draftFromSnapshot(snapshot);
  assert.equal(baseline.launch, 'live');
  assert.equal(patchFromDraft(baseline, baseline).launch, undefined);
  assert.equal(patchFromDraft({ ...baseline, launch: 'dry-run' }, baseline).launch, 'dry-run');
  assert.equal(patchFromDraft({ ...baseline, launch: '' }, baseline).launch, '');
});
