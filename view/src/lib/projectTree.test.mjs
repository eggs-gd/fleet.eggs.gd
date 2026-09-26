import assert from 'node:assert/strict';
import test from 'node:test';

import {
  projectAccordionKey,
  pruneAccordionOpen,
  workspaceAccordionKey
} from './dashboardState.js';
import { archiveListMode, groupedByWorkspaceAndProject } from './taskDisplay.js';
import { buildSidebarTree, projectInitial, projectOpenCount } from './projectTree.js';

test('buildSidebarTree nests projects by id prefix and keeps a standalone workspace as one row', () => {
  const workspaces = [
    { id: 'unity', title: 'Unity', kind: 'workspace_group' },
    { id: 'core-eggs-gd', title: 'Core' }
  ];
  const projects = [
    { id: 'unity', title: 'Unity', workspace_id: 'unity' },
    { id: 'unity/foo', title: 'Foo', workspace_id: 'unity' },
    { id: 'unity/foo/bar', title: 'Bar', workspace_id: 'unity' },
    { id: 'core-eggs-gd', title: 'Core', workspace_id: 'core-eggs-gd' }
  ];
  const tasks = [
    { project_id: 'unity/foo', workspace_id: 'unity', status: 'todo' },
    { project_id: 'unity/foo/bar', workspace_id: 'unity', status: 'doing' },
    { project_id: 'unity/foo/bar', workspace_id: 'unity', status: 'archived' },
    { project_id: 'core-eggs-gd', workspace_id: 'core-eggs-gd', status: 'todo' }
  ];

  const tree = buildSidebarTree(workspaces, projects, tasks);
  const unity = tree.find((node) => node.workspace.id === 'unity');
  const core = tree.find((node) => node.workspace.id === 'core-eggs-gd');

  assert.equal(unity.isGroup, true);
  assert.equal(unity.children.length, 1);
  assert.equal(unity.children[0].project.id, 'unity/foo');
  assert.equal(unity.children[0].children[0].project.id, 'unity/foo/bar');
  assert.equal(unity.children[0].children[0].taskCount, 1);
  // The workspace's own root project (id === workspace.id) must stay reachable
  // as a node even when the workspace also has real child projects — it used
  // to be silently dropped (excluded from children, and leafProject is null
  // for groups), making it unclickable in the sidebar.
  assert.ok(
    unity.rootNode,
    'workspace-root project should still be reachable when the workspace has children'
  );
  assert.equal(unity.rootNode.project.id, 'unity');
  assert.equal(core.isGroup, false);
  assert.equal(core.leafProject.id, 'core-eggs-gd');
  assert.equal(projectOpenCount(tasks, 'unity/foo'), 1);
  assert.equal(projectInitial('eGGs.gd', 'eggs'), 'E');
});

test('archive grouping marks a standalone project as a leaf and a space as a group', () => {
  const workspaces = [
    { id: 'unity', title: 'Unity', kind: 'workspace_group' },
    { id: 'core-eggs-gd', title: 'Core' }
  ];
  const projects = [
    { id: 'unity', title: 'Unity', workspace_id: 'unity' },
    { id: 'unity/foo', title: 'Foo', workspace_id: 'unity' },
    { id: 'core-eggs-gd', title: 'Core', workspace_id: 'core-eggs-gd' }
  ];
  const tasks = [
    { project_id: 'unity/foo', workspace_id: 'unity', status: 'archived' },
    { project_id: 'core-eggs-gd', workspace_id: 'core-eggs-gd', status: 'archived' }
  ];

  const groups = groupedByWorkspaceAndProject(tasks, workspaces, projects);
  const core = groups.find((group) => group.id === 'core-eggs-gd');
  const unity = groups.find((group) => group.id === 'unity');

  assert.equal(core.isGroup, false);
  assert.equal(core.projects.length, 1);
  assert.equal(core.projects[0].id, 'core-eggs-gd');
  assert.equal(unity.isGroup, true);
  assert.equal(unity.projects[0].id, 'unity/foo');
});

test('archive list mode follows the current project filter', () => {
  assert.equal(archiveListMode('core-eggs-gd', ''), 'flat');
  assert.equal(archiveListMode('', 'eggs-gd-prod'), 'projects');
  assert.equal(archiveListMode('', ''), 'workspaces');
});

test('sidebar workspace and project accordion keys survive prune on poll', () => {
  const state = {
    workspaces: [{ id: 'unity', title: 'Unity' }],
    projects: [
      { id: 'unity/foo', title: 'Foo' },
      { id: 'unity/foo/bar', title: 'Bar' }
    ],
    tasks: [],
    session_groups: []
  };
  const workspaceKey = workspaceAccordionKey(state.workspaces[0]);
  const projectKey = projectAccordionKey(state.projects[0]);
  const nestedKey = projectAccordionKey(state.projects[1]);
  const stale = pruneAccordionOpen(
    {
      [workspaceKey]: false,
      [projectKey]: true,
      [nestedKey]: false,
      'sidebar:workspace:unity': true,
      'project:gone': true
    },
    state
  );

  assert.equal(stale[workspaceKey], false);
  assert.equal(stale[projectKey], true);
  assert.equal(stale[nestedKey], false);
  assert.equal(stale['sidebar:workspace:unity'], undefined);
  assert.equal(stale['project:gone'], undefined);
});
