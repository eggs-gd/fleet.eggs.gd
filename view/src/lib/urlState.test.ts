import assert from 'node:assert/strict';
import test from 'node:test';
import { boardSearch, readBoardLocation, reconcileBoardLocation } from './urlState.ts';

test('board search omits defaults and round-trips the rest', () => {
  const location = readBoardLocation('?project=acme&filter=mine&q=invoice&sessions=closed');
  assert.equal(location.project, 'acme');
  assert.equal(location.filter, 'mine');
  assert.equal(location.view, 'board');
  assert.equal(location.sessions, 'closed');
  assert.equal(boardSearch(location), '?project=acme&filter=mine&q=invoice&sessions=closed');
  assert.equal(boardSearch(readBoardLocation('')), '');
});

test('reconcile drops selection the data no longer has and falls back to the board view', () => {
  const known = {
    projectIds: new Set(['acme']),
    workspaceIds: new Set(['acme-ws']),
    views: ['board', 'archive', 'projectSettings'],
    viewNeedingProject: 'projectSettings'
  };
  const stale = reconcileBoardLocation(
    { project: 'gone', workspace: 'gone', view: 'projectSettings' },
    known
  );
  assert.deepEqual(stale, { project: '', workspace: '', view: 'board' });

  const kept = reconcileBoardLocation(
    { project: 'acme', workspace: '', view: 'projectSettings' },
    known
  );
  assert.deepEqual(kept, { project: 'acme', workspace: '', view: 'projectSettings' });

  assert.equal(
    reconcileBoardLocation({ project: '', workspace: '', view: 'nonsense' }, known).view,
    'board'
  );
});
