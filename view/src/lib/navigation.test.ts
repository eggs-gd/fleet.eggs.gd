import assert from 'node:assert/strict';
import test from 'node:test';
import {
  changeChip,
  normalizeLocation,
  reconcileSelection,
  selectAllProjects,
  selectNav,
  selectProject,
  selectQuickView,
  selectSettingsSection,
  selectWorkspace
} from './navigation.ts';
import { PROJECT_SETTINGS_VIEW } from './projectSettings.ts';
import { readBoardLocation } from './urlState.ts';

const base = readBoardLocation('');

test('unknown URL values fold to the defaults', () => {
  const location = normalizeLocation({ ...base, nav: 'weird', sessions: 'other' });
  assert.equal(location.nav, 'work');
  assert.equal(location.sessions, 'active');
  assert.equal(
    normalizeLocation({ ...base, nav: 'settings', sessions: 'closed' }).sessions,
    'closed'
  );
});

test('selecting a project clears the workspace, leaves settings and shows the board', () => {
  const next = selectProject(
    { ...base, workspace: 'acme', nav: 'settings', view: 'archive' },
    'api'
  );
  assert.equal(next.project, 'api');
  assert.equal(next.workspace, '');
  assert.equal(next.nav, 'work');
  assert.equal(next.view, 'board');
});

test('a workspace selection cannot stay on a project settings view', () => {
  const onSettings = { ...base, project: 'api', view: PROJECT_SETTINGS_VIEW };
  assert.equal(selectWorkspace(onSettings, 'acme').view, 'board');
  assert.equal(selectAllProjects(onSettings).view, 'board');
  // other views are kept
  assert.equal(selectWorkspace({ ...base, view: 'archive' }, 'acme').view, 'archive');
  assert.equal(selectAllProjects({ ...base, view: 'archive' }).view, 'archive');
  assert.equal(selectWorkspace(onSettings, 'acme').project, '');
});

test('select all keeps the nav, quick views reset the scope', () => {
  assert.equal(selectAllProjects({ ...base, nav: 'settings' }).nav, 'settings');
  const archived = selectQuickView({ ...base, project: 'api', filter: 'mine' }, 'archived');
  assert.deepEqual(
    [archived.project, archived.view, archived.filter, archived.nav],
    ['', 'archive', 'all', 'work']
  );
  const blocked = selectQuickView({ ...base, view: 'archive', nav: 'settings' }, 'blocked');
  assert.deepEqual([blocked.view, blocked.filter, blocked.nav], ['board', 'blocked', 'work']);
});

test('picking a chip returns to the board', () => {
  const next = changeChip({ ...base, view: 'archive' }, 'bug');
  assert.deepEqual([next.filter, next.view], ['bug', 'board']);
});

test('nav changes', () => {
  assert.deepEqual(
    [selectNav(base, 'agents').nav, selectNav(base, 'agents').section],
    ['settings', 'agents']
  );
  assert.equal(selectNav(base, 'settings').nav, 'settings');
  // returning to work from a view that only exists elsewhere shows the board
  assert.equal(selectNav({ ...base, nav: 'settings', view: 'mystery' }, 'work').view, 'board');
  assert.equal(selectNav({ ...base, nav: 'settings', view: 'archive' }, 'work').view, 'archive');
  const section = selectSettingsSection(base, 'workflow');
  assert.deepEqual([section.section, section.nav], ['workflow', 'settings']);
});

test('selection the data no longer backs is dropped', () => {
  const stale = { ...base, project: 'gone', workspace: 'acme', view: PROJECT_SETTINGS_VIEW };
  const next = reconcileSelection(stale, ['api'], ['acme']);
  assert.deepEqual([next.project, next.workspace], ['', 'acme']);
  // settings need a project or workspace to be about
  const orphaned = reconcileSelection({ ...stale, workspace: 'gone' }, ['api'], ['acme']);
  assert.equal(orphaned.view, 'board');
});
