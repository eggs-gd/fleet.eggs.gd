import assert from 'node:assert/strict';
import test from 'node:test';
import {
  activeTasksFor,
  archivedTasksFor,
  backlogTasksFor,
  chipsFor,
  listTasksFor,
  matchesQuickFilter,
  selectedProjectOf,
  visibleTasksFor
} from './boardView.ts';
import type { Task } from './types.ts';

const task = (fields: Partial<Task>): Task => ({
  ref: 'T-1',
  title: 'A task',
  type: 'feature',
  status: 'todo',
  priority: 3,
  project_id: 'api',
  workspace_id: 'acme',
  updated_at: '2026-10-01T00:00:00Z',
  ...fields
});

const everything = { projectId: '', workspaceId: '', query: '', sort: 'updated' };

test('scope narrows by project, then by workspace', () => {
  const tasks = [
    task({ ref: 'A-1', project_id: 'api', workspace_id: 'acme' }),
    task({ ref: 'B-1', project_id: 'web', workspace_id: 'acme' }),
    task({ ref: 'C-1', project_id: 'ops', workspace_id: 'other' })
  ];
  const refs = (selection: Partial<typeof everything>) =>
    visibleTasksFor(tasks, { ...everything, ...selection })
      .map((t) => t.ref)
      .sort();
  assert.deepEqual(refs({}), ['A-1', 'B-1', 'C-1']);
  assert.deepEqual(refs({ workspaceId: 'acme' }), ['A-1', 'B-1']);
  assert.deepEqual(refs({ projectId: 'web' }), ['B-1']);
  // a project selection wins over a stale workspace id
  assert.deepEqual(refs({ projectId: 'web', workspaceId: 'other' }), ['B-1']);
});

test('search matches ref, title, type and repositories, ignoring case and padding', () => {
  const tasks = [
    task({ ref: 'A-1', title: 'Fix invoice rounding' }),
    task({ ref: 'A-2', title: 'Other', repositories: ['billing-core'] })
  ];
  const find = (query: string) =>
    visibleTasksFor(tasks, { ...everything, query }).map((t) => t.ref);
  assert.deepEqual(find('  INVOICE '), ['A-1']);
  assert.deepEqual(find('billing'), ['A-2']);
  assert.deepEqual(find('a-'), ['A-1', 'A-2'].sort());
  assert.deepEqual(find('nothing'), []);
});

test('sorting by priority puts urgent work first, otherwise newest update first', () => {
  const tasks = [
    task({ ref: 'old-urgent', priority: 1, updated_at: '2026-01-01T00:00:00Z' }),
    task({ ref: 'new-low', priority: 5, updated_at: '2026-10-01T00:00:00Z' })
  ];
  assert.equal(visibleTasksFor(tasks, everything)[0].ref, 'new-low');
  assert.equal(visibleTasksFor(tasks, { ...everything, sort: 'priority' })[0].ref, 'old-urgent');
});

test('archived tasks split off from the active ones', () => {
  const tasks = [task({ ref: 'a', status: 'todo' }), task({ ref: 'b', status: 'archived' })];
  assert.deepEqual(
    activeTasksFor(tasks).map((t) => t.ref),
    ['a']
  );
  assert.deepEqual(
    archivedTasksFor(tasks).map((t) => t.ref),
    ['b']
  );
  assert.deepEqual(
    backlogTasksFor(tasks, []).map((t) => t.ref),
    ['a']
  );
});

test('quick filters', () => {
  assert.ok(matchesQuickFilter(task({ status: 'todo' }), 'open'));
  assert.ok(!matchesQuickFilter(task({ status: 'done' }), 'open'));
  assert.ok(matchesQuickFilter(task({ status: 'needs_rework' }), 'rework'));
  assert.ok(matchesQuickFilter(task({ priority: 2 }), 'high'));
  assert.ok(!matchesQuickFilter(task({ priority: 3 }), 'high'));
  assert.ok(matchesQuickFilter(task({ type: 'bug' }), 'bug'));
  assert.ok(matchesQuickFilter(task({ status: 'blocked' }), 'attention'));
  assert.ok(matchesQuickFilter(task({ status: 'needs_review' }), 'review'));
  assert.ok(matchesQuickFilter(task({}), 'all'));
  assert.ok(matchesQuickFilter(task({}), 'unknown-filter'));
});

test('whole-board filters read the active list, the rest read the backlog', () => {
  const active = [task({ ref: 'active' })];
  const backlog = [task({ ref: 'backlog' })];
  for (const filter of ['blocked', 'review', 'done', 'attention']) {
    assert.equal(listTasksFor(filter, active, backlog), active);
  }
  for (const filter of ['all', 'open', 'mine']) {
    assert.equal(listTasksFor(filter, active, backlog), backlog);
  }
});

test('chips count the backlog and add one for a sidebar-only filter', () => {
  const backlog = [
    task({ type: 'bug', priority: 1 }),
    task({ type: 'feature', status: 'done' }),
    task({ type: 'feature' })
  ];
  const chips = chipsFor(backlog, 'all', 3);
  const count = (id: string) => chips.find((chip) => chip.id === id)?.count;
  assert.deepEqual(
    chips.map((chip) => chip.id),
    ['all', 'open', 'rework', 'mine', 'high', 'bug', 'feature']
  );
  assert.equal(count('all'), 3);
  assert.equal(count('open'), 2);
  assert.equal(count('bug'), 1);
  assert.equal(count('feature'), 2);

  const withExtra = chipsFor(backlog, 'review', 4);
  assert.deepEqual(withExtra.at(-1), { id: 'review', label: 'In Review', count: 4 });
});

test('selected project falls back to the workspace', () => {
  const projects = [{ id: 'api', title: 'API' }];
  const workspaces = [{ id: 'acme', title: 'Acme' }];
  assert.equal(selectedProjectOf(projects, workspaces, 'api', '')?.title, 'API');
  assert.equal(selectedProjectOf(projects, workspaces, '', 'acme')?.title, 'Acme');
  assert.equal(selectedProjectOf(projects, workspaces, '', ''), null);
});
