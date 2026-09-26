import assert from 'node:assert/strict';
import test from 'node:test';

import { childProjects, parentProjectId, projectSettingsInspect } from './projectSettings.js';

test('parent and child ids follow slash nesting against the real project/workspace id list', () => {
  const ids = [
    'core-eggs-gd',
    'audiophile',
    'audiophile/jivemax-lua',
    'audiophile/jivemax',
    'audiophile/jivemax/jivelite',
    'other'
  ];
  assert.equal(parentProjectId('core-eggs-gd', ids), '');
  assert.equal(parentProjectId('audiophile/jivemax-lua', ids), 'audiophile');
  // Longest-prefix match: a grandchild resolves to its immediate parent, not
  // the top-level workspace — this is the case the old one-level string-split
  // implementation got wrong (it either missed grandchildren entirely from
  // childProjects, or would have flattened them straight under "audiophile").
  assert.equal(parentProjectId('audiophile/jivemax/jivelite', ids), 'audiophile/jivemax');
  assert.deepEqual(
    childProjects(
      [
        { id: 'audiophile/cratune' },
        { id: 'audiophile/jivemax' },
        { id: 'audiophile/jivemax/jivelite' },
        { id: 'other' }
      ],
      'audiophile'
    ).map((p) => p.id),
    ['audiophile/cratune', 'audiophile/jivemax']
  );
  // jivelite is two levels below audiophile, so it must NOT show up as a
  // direct child of audiophile — it's a direct child of audiophile/jivemax.
  assert.deepEqual(
    childProjects(
      [
        { id: 'audiophile/cratune' },
        { id: 'audiophile/jivemax' },
        { id: 'audiophile/jivemax/jivelite' },
        { id: 'other' }
      ],
      'audiophile/jivemax'
    ).map((p) => p.id),
    ['audiophile/jivemax/jivelite']
  );
});

test('standalone project inspects PROJECT.md and the bound repo without header fields', () => {
  const inspect = projectSettingsInspect(
    {
      id: 'core-eggs-gd',
      workspace_id: 'core-eggs-gd',
      title: 'Core',
      kind: 'repository_project',
      source: 'workspace.repositories',
      path: '/Users/me/Projects/core.eggs.gd/Work/core.eggs.gd/PROJECT.md',
      relative_path: 'Work/core.eggs.gd/PROJECT.md',
      repositories: ['core.eggs.gd'],
      technology: {
        repositories: [
          {
            id: 'core-eggs-gd-1',
            name: 'core.eggs.gd',
            relative_path: 'core.eggs.gd',
            remote: 'git@github.com:eggs-gd/core.eggs.gd.git',
            branch: 'main'
          }
        ]
      }
    },
    {
      workspaces: [{ id: 'core-eggs-gd', title: 'Core' }],
      projects: [{ id: 'core-eggs-gd', title: 'Core' }]
    }
  );
  assert.equal(inspect.kind, 'repository_project');
  assert.equal(inspect.projectMd, '/Users/me/Projects/core.eggs.gd/Work/core.eggs.gd/PROJECT.md');
  assert.equal(inspect.membership, null);
  assert.equal(inspect.parent, null);
  assert.equal(inspect.repositories.length, 1);
  assert.equal(inspect.repositories[0].branch, 'main');
  assert.equal(inspect.repositories[0].webUrl, 'https://github.com/eggs-gd/core.eggs.gd');
});

test('nested project shows workspace membership and enriches git nesting from registry', () => {
  const inspect = projectSettingsInspect(
    {
      id: 'audiophile/jivemax-lua',
      workspace_id: 'audiophile',
      title: 'jivemax-lua',
      kind: 'repository_project',
      source: 'workspace.repositories',
      path: '',
      relative_path: 'Work/audiophile/PROJECT.md',
      repositories: ['Audiophile/jivemax-lua'],
      technology: {
        repositories: [
          {
            id: 'jivemax-lua-1',
            relative_path: 'Audiophile/jivemax-lua',
            remote: 'git@github.com:example/jivemax-lua.git',
            branch: 'main'
          }
        ]
      }
    },
    {
      workspaces: [{ id: 'audiophile', title: 'Audiophile', kind: 'workspace_group' }],
      projects: [
        { id: 'audiophile/jivemax-lua', title: 'jivemax-lua' },
        { id: 'audiophile/cratune', title: 'cratune' }
      ],
      repositories: [
        {
          id: 'jivemax-lua-1',
          relative_path: 'Audiophile/jivemax-lua',
          nested_repository_ids: ['luajit-1']
        },
        { id: 'luajit-1', name: 'luajit', relative_path: 'Audiophile/jivemax-lua/luajit' }
      ]
    }
  );
  assert.equal(inspect.membership.id, 'audiophile');
  assert.equal(inspect.parent.id, 'audiophile');
  assert.equal(inspect.parent.title, 'Audiophile');
  assert.equal(inspect.projectMd, 'Work/audiophile/PROJECT.md');
  assert.deepEqual(inspect.repositories[0].nests, ['Audiophile/jivemax-lua/luajit']);
});

test('missing item is empty inspect', () => {
  assert.equal(projectSettingsInspect(null), null);
});
