import assert from 'node:assert/strict';
import test from 'node:test';

import {
  projectAccordionKey,
  pruneAccordionOpen,
  workspaceAccordionKey
} from './dashboardState.ts';
import {
  archiveListMode,
  groupedByMilestone,
  groupedByWorkspaceAndProject,
  shouldGroupByMilestone,
  milestoneGroupStatusCounts
} from './taskDisplay.ts';
import { buildSidebarTree, projectInitial, projectOpenCount } from './projectTree.ts';

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
  const unity = tree.find((node) => node.workspace.id === 'unity')!;
  const core = tree.find((node) => node.workspace.id === 'core-eggs-gd')!;

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
  assert.equal(unity.rootNode!.project.id, 'unity');
  assert.equal(core.isGroup, false);
  assert.equal(core.leafProject!.id, 'core-eggs-gd');
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
  const core = groups.find((group) => group.id === 'core-eggs-gd')!;
  const unity = groups.find((group) => group.id === 'unity')!;

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

test('shouldGroupByMilestone is true once a task opts in, or the selected project defines one', () => {
  const noMilestones = [{ id: 'acme/web', title: 'Web' }];
  const withMilestone = [
    { id: 'acme/web', title: 'Web', milestones: [{ id: 'v1-launch', title: 'v1 launch' }] }
  ];

  assert.equal(shouldGroupByMilestone([{ status: 'todo' }], noMilestones, ''), false);
  assert.equal(
    shouldGroupByMilestone([{ status: 'todo' }, { milestone: 'v1-launch' }], noMilestones, ''),
    true
  );
  // A freshly created, still-empty milestone must be visible without any
  // task opted in yet — otherwise creating one looks like it silently failed.
  assert.equal(shouldGroupByMilestone([{ status: 'todo' }], withMilestone, 'acme/web'), true);
  // But only when that project is the one actually selected.
  assert.equal(shouldGroupByMilestone([{ status: 'todo' }], withMilestone, ''), false);
});

test('groupedByMilestone trims the breadcrumb to what the selection does not already pin down', () => {
  const workspaces = [{ id: 'acme', title: 'Acme' }];
  const projects = [
    {
      id: 'acme/web',
      title: 'Web',
      workspace_id: 'acme',
      milestones: [
        { id: 'v1-launch', title: 'v1 launch', state: 'active' as const, version: 'v1.0.0' }
      ]
    }
  ];
  const tasks = [
    { project_id: 'acme/web', workspace_id: 'acme', milestone: 'v1-launch', status: 'doing' },
    { project_id: 'acme/web', workspace_id: 'acme', milestone: 'v1-launch', status: 'todo' },
    { project_id: 'acme/web', workspace_id: 'acme', status: 'backlog' }
  ];

  const allProjects = groupedByMilestone(tasks, projects, workspaces, '', '');
  assert.deepEqual(
    allProjects.map((g) => g.title),
    ['Acme / Web / v1 launch', 'Acme / Web / Unscheduled']
  );

  const withinWorkspace = groupedByMilestone(tasks, projects, workspaces, '', 'acme');
  assert.deepEqual(
    withinWorkspace.map((g) => g.title),
    ['Web / v1 launch', 'Web / Unscheduled']
  );

  const withinProject = groupedByMilestone(tasks, projects, workspaces, 'acme/web', 'acme');
  assert.deepEqual(
    withinProject.map((g) => g.title),
    ['v1 launch', 'Unscheduled']
  );

  const launch = withinProject.find((g) => g.milestoneId === 'v1-launch')!;
  assert.equal(launch.tasks.length, 2);
  assert.equal(launch.state, 'active');
  assert.equal(launch.version, 'v1.0.0');
  const unscheduled = withinProject.find((g) => g.milestoneId === '')!;
  assert.equal(unscheduled.tasks.length, 1);
});

test('groupedByMilestone seeds an empty group for a milestone with no tasks yet, only for the selected project', () => {
  const workspaces = [{ id: 'acme', title: 'Acme' }];
  const projects = [
    {
      id: 'acme/web',
      title: 'Web',
      workspace_id: 'acme',
      milestones: [{ id: 'v2-beta', title: 'v2 beta', state: 'parked' as const }]
    }
  ];
  const tasks = [{ project_id: 'acme/web', workspace_id: 'acme', status: 'backlog' }];

  const withinProject = groupedByMilestone(tasks, projects, workspaces, 'acme/web', 'acme');
  assert.deepEqual(
    withinProject.map((g) => g.title),
    ['v2 beta', 'Unscheduled']
  );
  const beta = withinProject.find((g) => g.milestoneId === 'v2-beta')!;
  assert.equal(beta.tasks.length, 0);
  assert.equal(beta.state, 'parked');

  // Not seeded when no project is selected — avoids dumping every empty
  // milestone from every project into a broad, unscoped view.
  const allProjects = groupedByMilestone(tasks, projects, workspaces, '', '');
  assert.deepEqual(
    allProjects.map((g) => g.milestoneId),
    ['']
  );
});

test('groupedByMilestone keeps Unscheduled as a drop target when every task is scheduled', () => {
  const workspaces = [{ id: 'acme', title: 'Acme' }];
  const projects = [
    {
      id: 'acme',
      title: 'Acme',
      workspace_id: 'acme',
      milestones: [{ id: 'v1', title: 'v1', state: 'active' as const }]
    }
  ];
  const tasks = [{ project_id: 'acme', workspace_id: 'acme', milestone: 'v1', status: 'todo' }];

  const groups = groupedByMilestone(tasks, projects, workspaces, 'acme', 'acme');
  assert.deepEqual(
    groups.map((g) => g.milestoneId),
    ['v1', '']
  );
  assert.equal(groups[1].tasks.length, 0);
});

test('milestoneGroupStatusCounts sorts by count, most first', () => {
  const counts = milestoneGroupStatusCounts([
    { status: 'doing' },
    { status: 'todo' },
    { status: 'doing' },
    { status: 'todo' },
    { status: 'todo' }
  ]);
  assert.deepEqual(counts, [
    { status: 'todo', count: 3 },
    { status: 'doing', count: 2 }
  ]);
});
