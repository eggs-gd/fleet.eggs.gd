import assert from 'node:assert/strict';
import test from 'node:test';

import {
  EXECUTION_HISTORY_LIMITS,
  applyTaskUpdate,
  archiveProjectAccordionKey,
  archiveWorkspaceAccordionKey,
  dedupeTasksByIdentity,
  executionHistoryGroupKey,
  executionHistoryPanelKey,
  flattenClosedSessions,
  groupActivityAt,
  groupLatestSession,
  isAccordionOpen,
  isArchivedStatus,
  isBoardStatus,
  partitionTasksByBoard,
  prepareExecutionHistory,
  projectAccordionKey,
  pruneAccordionOpen,
  reconcileSelectedTaskFromState,
  refreshStatusPresentation,
  repositoryAccordionKey,
  shouldShowRefreshBusy,
  sameTaskRecord,
  sessionActivityAt,
  sessionAllowsOperatorRelease,
  sessionGroupKey,
  sessionGroupProjectId,
  sessionIsActive,
  taskTitleForSessionGroup,
  updateAccordionOpen,
  visibleSessionsForGroup,
  workspaceAccordionKey
} from './dashboardState.ts';

const firstPoll = {
  workspaces: [{ id: 'core-eggs-gd', title: 'Core' }],
  projects: [
    {
      id: 'core-eggs-gd',
      title: 'Core',
      technology: {
        repositories: [
          {
            relative_path: 'core.eggs.gd',
            effective_tags: ['svelte'],
            detected_tags: ['go', 'svelte']
          }
        ]
      }
    }
  ],
  tasks: [
    {
      id: 'work-67',
      ref: 'CORE-67',
      path: '/core/Work/core-eggs-gd/tasks/core-67.md',
      title: 'Preserve dashboard UI state'
    }
  ],
  session_groups: [
    {
      task_ref: 'CORE-67',
      task_id: 'work-67',
      task_path: '/core/Work/core-eggs-gd/tasks/core-67.md',
      sessions: [
        {
          claim_id: 'c67-1',
          role: 'current',
          status: 'exited',
          claimed_at: '2026-08-01T10:00:00+03:00'
        }
      ]
    }
  ]
};

const secondPoll = {
  ...firstPoll,
  projects: [
    {
      ...firstPoll.projects[0],
      technology: {
        repositories: [
          {
            relative_path: 'core.eggs.gd',
            effective_tags: ['svelte', 'go'],
            detected_tags: ['go', 'svelte', 'vite']
          }
        ]
      }
    }
  ],
  tasks: [
    {
      ...firstPoll.tasks[0],
      title: 'Preserve dashboard UI state during polling'
    }
  ]
};

test('refreshStatusPresentation keeps stable labels across idle/busy/error polls', () => {
  const idle = refreshStatusPresentation({
    lastRefreshAt: '2026-08-04T12:00:00+03:00',
    formatTime: () => '12:00:00 PM'
  });
  assert.equal(idle.label, 'Auto-refresh every 5s');
  assert.equal(idle.warning, false);
  assert.equal(idle.busy, false);
  assert.equal(idle.busyLabel, 'Refreshing');
  assert.equal(idle.updatedText, 'Updated 12:00:00 PM');
  assert.equal(idle.updatedAt, '2026-08-04T12:00:00+03:00');

  const busy = refreshStatusPresentation({
    refreshing: true,
    lastRefreshAt: '2026-08-04T12:00:00+03:00',
    formatTime: () => '12:00:00 PM'
  });
  assert.equal(busy.label, idle.label);
  assert.equal(busy.busy, true);
  assert.equal(busy.busyLabel, idle.busyLabel);
  assert.equal(busy.updatedText, idle.updatedText);

  const failed = refreshStatusPresentation({
    refreshError: 'Failed to load Core state (503)',
    lastRefreshAt: '2026-08-04T12:00:00+03:00',
    formatTime: () => '12:00:00 PM'
  });
  assert.equal(failed.warning, true);
  assert.equal(failed.label, 'Auto-refresh failed: Failed to load Core state (503)');
  assert.equal(failed.updatedText, idle.updatedText);

  const beforeFirstSuccess = refreshStatusPresentation({});
  assert.equal(beforeFirstSuccess.updatedAt, '');
  assert.equal(beforeFirstSuccess.updatedText, '');
  assert.equal(beforeFirstSuccess.busyLabel, 'Refreshing');
});

test('shouldShowRefreshBusy stays quiet for fast auto-polls', () => {
  assert.equal(shouldShowRefreshBusy({ manual: true, elapsedMs: 0 }), true);
  assert.equal(shouldShowRefreshBusy({ manual: false, elapsedMs: 0 }), false);
  assert.equal(shouldShowRefreshBusy({ manual: false, elapsedMs: 319 }), false);
  assert.equal(shouldShowRefreshBusy({ manual: false, elapsedMs: 320 }), true);
  assert.equal(shouldShowRefreshBusy({ manual: false, elapsedMs: 500, delayMs: 400 }), true);
});

test('accordion state survives repeated polling for unchanged project technology rows', () => {
  const project = firstPoll.projects[0];
  const repo = project.technology.repositories[0];
  const workspaceKey = workspaceAccordionKey(firstPoll.workspaces[0]);
  const projectKey = projectAccordionKey(project);
  const repoKey = repositoryAccordionKey(project, repo);
  const historyPanelKey = executionHistoryPanelKey();
  const historyGroupKey = executionHistoryGroupKey(firstPoll.session_groups[0]);
  const archiveWorkspaceKey = archiveWorkspaceAccordionKey(firstPoll.workspaces[0]);
  const archiveProjectKey = archiveProjectAccordionKey(project);

  let openState = {};
  openState = updateAccordionOpen(openState, workspaceKey, true);
  openState = updateAccordionOpen(openState, projectKey, true);
  openState = updateAccordionOpen(openState, repoKey, true);
  openState = updateAccordionOpen(openState, historyPanelKey, true);
  openState = updateAccordionOpen(openState, historyGroupKey, true);
  openState = updateAccordionOpen(openState, archiveWorkspaceKey, false);
  openState = updateAccordionOpen(openState, archiveProjectKey, false);

  openState = pruneAccordionOpen(openState, firstPoll);
  openState = pruneAccordionOpen(openState, secondPoll);

  assert.equal(isAccordionOpen(openState, workspaceKey), true);
  assert.equal(isAccordionOpen(openState, projectKey), true);
  assert.equal(isAccordionOpen(openState, repoKey), true);
  assert.equal(isAccordionOpen(openState, historyPanelKey), true);
  assert.equal(isAccordionOpen(openState, historyGroupKey), true);
  assert.equal(isAccordionOpen(openState, archiveWorkspaceKey), false);
  assert.equal(isAccordionOpen(openState, archiveProjectKey), false);
  assert.equal(
    reconcileSelectedTaskFromState(firstPoll.tasks[0], secondPoll)?.title,
    'Preserve dashboard UI state during polling'
  );
});

test('missing items close without clearing unrelated expanded accordions', () => {
  const project = firstPoll.projects[0];
  const repo = project.technology.repositories[0];
  const workspaceKey = workspaceAccordionKey(firstPoll.workspaces[0]);
  const projectKey = projectAccordionKey(project);
  const repoKey = repositoryAccordionKey(project, repo);
  const missingProjectKey = projectAccordionKey({ id: 'deleted-project' });
  const missingHistoryKey = executionHistoryGroupKey({ task_ref: 'CORE-GONE' });

  const openState = pruneAccordionOpen(
    {
      [workspaceKey]: true,
      [projectKey]: true,
      [repoKey]: true,
      [missingProjectKey]: true,
      [executionHistoryPanelKey()]: true,
      [missingHistoryKey]: true
    },
    secondPoll
  );

  assert.equal(isAccordionOpen(openState, workspaceKey), true);
  assert.equal(isAccordionOpen(openState, projectKey), true);
  assert.equal(isAccordionOpen(openState, repoKey), true);
  assert.equal(isAccordionOpen(openState, executionHistoryPanelKey()), true);
  assert.equal(openState[missingProjectKey], undefined);
  assert.equal(openState[missingHistoryKey], undefined);
  assert.equal(reconcileSelectedTaskFromState({ path: '/missing.md' }, secondPoll), null);
});

test('sessionGroupKey prefers task_path, then task_id, then task_ref', () => {
  assert.equal(
    sessionGroupKey({
      task_path: 'Work/core-eggs-gd/tasks/x.md',
      task_id: 'work-x',
      task_ref: 'CORE-1'
    }),
    'Work/core-eggs-gd/tasks/x.md'
  );
  assert.equal(sessionGroupKey({ task_id: 'work-x', task_ref: 'CORE-1' }), 'work-x');
  assert.equal(sessionGroupKey({ task_ref: 'CORE-1' }), 'CORE-1');
  assert.equal(sessionGroupKey({}), 'session-group');
});

test('prepareExecutionHistory ranks recent groups and keeps expand filters', () => {
  const now = Date.parse('2026-08-02T12:00:00+03:00');
  const tasks = [{ ref: 'CORE-10', id: 'work-10', title: 'Ten', path: 'Work/a/tasks/10.md' }];
  const groups = [
    {
      task_ref: 'CORE-1',
      task_path: 'Work/a/tasks/1.md',
      sessions: [
        {
          claim_id: 'old-terminal',
          role: 'historical',
          status: 'exited',
          agent: 'codex',
          claimed_at: '2026-07-01T10:00:00+03:00',
          exited_at: '2026-07-01T11:00:00+03:00'
        },
        {
          claim_id: 'current-1',
          role: 'current',
          status: 'exited',
          agent: 'codex',
          claimed_at: '2026-08-02T09:00:00+03:00',
          exited_at: '2026-08-02T09:30:00+03:00'
        }
      ]
    },
    {
      task_ref: 'CORE-10',
      task_id: 'work-10',
      task_path: 'Work/a/tasks/10.md',
      sessions: [
        {
          claim_id: 'current-10',
          role: 'current',
          status: 'running',
          execution_status: 'running',
          agent: 'cursor',
          claimed_at: '2026-08-02T11:00:00+03:00',
          last_output_at: '2026-08-02T11:50:00+03:00'
        }
      ]
    }
  ];

  const collapsed = prepareExecutionHistory(groups, { tasks, now });
  assert.equal(collapsed.groups[0].group.task_ref, 'CORE-10');
  assert.equal(collapsed.groups[0].title, 'Ten');
  assert.equal(collapsed.groups[0].agent, 'cursor');
  assert.equal(collapsed.groups[0].status, 'running');
  assert.equal(collapsed.groups[1].visibleSessions.length, 1);
  assert.equal(collapsed.groups[1].visibleSessions[0].claim_id, 'current-1');
  assert.equal(collapsed.groups[1].hiddenSessionCount, 1);

  const expandedKey = executionHistoryGroupKey(groups[0]);
  const expanded = prepareExecutionHistory(groups, {
    tasks,
    now,
    expandedGroupKeys: new Set([expandedKey])
  });
  assert.equal(expanded.groups[1].visibleSessions.length, 2);
  assert.equal(expanded.groups[1].hiddenSessionCount, 0);
});

test('visibleSessionsForGroup respects per-task limits', () => {
  const group = {
    sessions: Array.from({ length: 8 }, (_, index) => ({
      claim_id: `s${index}`,
      role: index === 7 ? 'current' : 'historical',
      status: 'exited',
      claimed_at: `2026-08-0${(index % 9) + 1}T10:00:00+03:00`
    }))
  };
  const visible = visibleSessionsForGroup(group, {
    expanded: true,
    limits: { ...EXECUTION_HISTORY_LIMITS, maxSessionsPerTask: 3 }
  });
  assert.equal(visible.sessions.length, 3);
  assert.equal(visible.sessions[0].claim_id, 's7');
  assert.equal(visible.hiddenCount, 5);
});

test('session activity helpers pick current role and active statuses', () => {
  assert.equal(sessionIsActive({ execution_status: 'running' }), true);
  assert.equal(sessionIsActive({ execution_status: 'waiting_input' }), true);
  assert.equal(sessionIsActive({ status: 'exited' }), false);
  assert.equal(
    sessionAllowsOperatorRelease({ claim_id: 'a', execution_status: 'resumable' }),
    true
  );
  assert.equal(sessionAllowsOperatorRelease({ claim_id: 'a', execution_status: 'dead' }), true);
  assert.equal(
    sessionAllowsOperatorRelease({
      claim_id: 'a',
      execution_status: 'running',
      provider_controllable: true
    }),
    false
  );
  assert.equal(sessionAllowsOperatorRelease({ execution_status: 'resumable' }), false);
  assert.equal(
    sessionAllowsOperatorRelease({ claim_id: 'a', execution_status: 'succeeded' }),
    false
  );
  assert.equal(sessionActivityAt({ last_output_at: 'b', claimed_at: 'a' }), 'b');
  assert.equal(
    groupLatestSession({
      sessions: [
        { claim_id: 'a', role: 'historical' },
        { claim_id: 'b', role: 'current' }
      ]
    })?.claim_id,
    'b'
  );
  assert.equal(
    groupActivityAt({
      sessions: [{ claimed_at: 'a' }, { last_event_at: 'c', claimed_at: 'b' }]
    }),
    'c'
  );
  assert.equal(
    taskTitleForSessionGroup({ task_ref: 'CORE-1' }, [{ ref: 'CORE-1', title: 'One' }]),
    'One'
  );
});

test('sameTaskRecord matches abs/rel paths and only uses fallback against candidates', () => {
  const backlog = {
    id: 'work-87',
    ref: 'CORE-87',
    path: '/core/Work/core-eggs-gd/tasks/core-87.md',
    relative_path: 'Work/core-eggs-gd/tasks/core-87.md',
    status: 'backlog'
  };
  const other = {
    id: 'work-1',
    ref: 'CORE-1',
    path: '/core/Work/core-eggs-gd/tasks/core-1.md',
    relative_path: 'Work/core-eggs-gd/tasks/core-1.md',
    status: 'todo'
  };
  const patch = {
    id: 'work-87',
    ref: 'CORE-87',
    path: '/core/Work/core-eggs-gd/tasks/core-87.md',
    relative_path: 'Work/core-eggs-gd/tasks/core-87.md',
    status: 'todo'
  };

  assert.equal(sameTaskRecord(backlog, patch), true);
  assert.equal(sameTaskRecord(backlog, { ...patch, path: backlog.relative_path }), true);
  assert.equal(sameTaskRecord(backlog, patch, backlog.path), true);
  assert.equal(sameTaskRecord(other, patch, backlog.path), false);
  assert.equal(sameTaskRecord(other, patch, patch.path), false);
});

test('applyTaskUpdate moves a backlog task to the board without leaving a duplicate', () => {
  const tasks = [
    {
      id: 'work-1',
      ref: 'CORE-1',
      path: '/core/Work/a/tasks/1.md',
      relative_path: 'Work/a/tasks/1.md',
      status: 'todo',
      title: 'Keep me'
    },
    {
      id: 'work-87',
      ref: 'CORE-87',
      path: '/core/Work/a/tasks/87.md',
      relative_path: 'Work/a/tasks/87.md',
      status: 'backlog',
      title: 'Leave backlog'
    }
  ];

  const updated = applyTaskUpdate(
    tasks,
    {
      id: 'work-87',
      ref: 'CORE-87',
      path: '/core/Work/a/tasks/87.md',
      relative_path: 'Work/a/tasks/87.md',
      status: 'todo',
      title: 'Leave backlog'
    },
    '/core/Work/a/tasks/87.md'
  );

  const partitioned = partitionTasksByBoard(updated);
  assert.equal(updated.length, 2);
  assert.equal(partitioned.backlog.length, 0);
  assert.equal(partitioned.board.length, 2);
  assert.equal(partitioned.board.filter((task) => task.ref === 'CORE-87').length, 1);
  assert.equal(partitioned.board.find((task) => task.ref === 'CORE-87')?.status, 'todo');
});

test('applyTaskUpdate returns a board task to backlog and collapses identity duplicates', () => {
  const duplicated = [
    {
      id: 'work-87',
      ref: 'CORE-87',
      path: '/core/Work/a/tasks/87.md',
      relative_path: 'Work/a/tasks/87.md',
      status: 'backlog',
      title: 'Stale backlog copy'
    },
    {
      id: 'work-87',
      ref: 'CORE-87',
      path: '/core/Work/a/tasks/87.md',
      relative_path: 'Work/a/tasks/87.md',
      status: 'todo',
      title: 'Board copy'
    },
    {
      id: 'work-2',
      ref: 'CORE-2',
      path: '/core/Work/a/tasks/2.md',
      relative_path: 'Work/a/tasks/2.md',
      status: 'doing',
      title: 'Other'
    }
  ];

  const updated = applyTaskUpdate(
    duplicated,
    {
      id: 'work-87',
      ref: 'CORE-87',
      relative_path: 'Work/a/tasks/87.md',
      status: 'backlog',
      title: 'Back on backlog'
    },
    'Work/a/tasks/87.md'
  );
  const partitioned = partitionTasksByBoard(updated);

  assert.equal(updated.length, 2);
  assert.equal(partitioned.backlog.length, 1);
  assert.equal(partitioned.backlog[0].title, 'Back on backlog');
  assert.equal(partitioned.board.length, 1);
  assert.equal(partitioned.board[0].ref, 'CORE-2');
});

test('applyTaskUpdate archives a done task in place', () => {
  const tasks = [
    {
      id: 'work-123',
      ref: 'CORE-123',
      path: '/core/Work/a/tasks/123.md',
      relative_path: 'Work/a/tasks/123.md',
      status: 'done',
      title: 'Ship it'
    }
  ];

  const updated = applyTaskUpdate(
    tasks,
    {
      id: 'work-123',
      ref: 'CORE-123',
      path: '/core/Work/a/tasks/123.md',
      relative_path: 'Work/a/tasks/123.md',
      status: 'archived',
      title: 'Ship it'
    },
    '/core/Work/a/tasks/123.md'
  );
  const partitioned = partitionTasksByBoard(updated);

  assert.equal(updated.length, 1);
  assert.equal(updated[0].status, 'archived');
  assert.equal(updated[0].ref, 'CORE-123');
  assert.equal(partitioned.board.length, 0);
  assert.equal(partitioned.archived.length, 1);
  assert.equal(partitioned.archived[0].ref, 'CORE-123');
});

test('partitionTasksByBoard keeps archived out of the active board', () => {
  const partitioned = partitionTasksByBoard([
    { ref: 'CORE-1', status: 'todo', title: 'Active' },
    { ref: 'CORE-2', status: 'done', title: 'Finished' },
    { ref: 'CORE-3', status: 'archived', title: 'History' },
    { ref: 'CORE-4', status: 'backlog', title: 'Later' }
  ]);

  assert.deepEqual(
    partitioned.board.map((task) => task.ref),
    ['CORE-1', 'CORE-2']
  );
  assert.deepEqual(
    partitioned.archived.map((task) => task.ref),
    ['CORE-3']
  );
  assert.deepEqual(
    partitioned.backlog.map((task) => task.ref),
    ['CORE-4']
  );
  assert.equal(isBoardStatus('archived'), false);
  assert.equal(isArchivedStatus('archived'), true);
  assert.equal(isBoardStatus('done'), true);
});

test('applyTaskUpdate restores an archived task to backlog', () => {
  const updated = applyTaskUpdate(
    [
      {
        id: 'work-123',
        ref: 'CORE-123',
        path: '/core/Work/a/tasks/123.md',
        relative_path: 'Work/a/tasks/123.md',
        status: 'archived',
        title: 'Ship it'
      }
    ],
    {
      id: 'work-123',
      ref: 'CORE-123',
      path: '/core/Work/a/tasks/123.md',
      relative_path: 'Work/a/tasks/123.md',
      status: 'backlog',
      title: 'Ship it'
    },
    '/core/Work/a/tasks/123.md'
  );
  const partitioned = partitionTasksByBoard(updated);

  assert.equal(updated[0].status, 'backlog');
  assert.equal(partitioned.archived.length, 0);
  assert.equal(partitioned.backlog.length, 1);
  assert.equal(partitioned.board.length, 0);
});

test('dedupeTasksByIdentity prefers board copy over backlog regardless of order', () => {
  const backlogFirst = dedupeTasksByIdentity([
    {
      id: 'work-87',
      ref: 'CORE-87',
      path: '/abs/Work/a/tasks/87.md',
      relative_path: 'Work/a/tasks/87.md',
      status: 'backlog',
      title: 'Stale'
    },
    {
      id: 'work-87',
      ref: 'CORE-87',
      path: '/abs/Work/a/tasks/87.md',
      relative_path: 'Work/a/tasks/87.md',
      status: 'todo',
      title: 'Fresh'
    }
  ]);
  const todoFirst = dedupeTasksByIdentity([
    {
      id: 'work-87',
      ref: 'CORE-87',
      path: '/abs/Work/a/tasks/87.md',
      relative_path: 'Work/a/tasks/87.md',
      status: 'todo',
      title: 'Fresh'
    },
    {
      id: 'work-87',
      ref: 'CORE-87',
      path: '/abs/Work/a/tasks/87.md',
      relative_path: 'Work/a/tasks/87.md',
      status: 'backlog',
      title: 'Stale'
    }
  ]);

  for (const polled of [backlogFirst, todoFirst]) {
    const partitioned = partitionTasksByBoard(polled);
    assert.equal(polled.length, 1);
    assert.equal(polled[0].status, 'todo');
    assert.equal(polled[0].title, 'Fresh');
    assert.equal(partitioned.backlog.length, 0);
    assert.equal(partitioned.board.length, 1);
  }
});

test('prepareExecutionHistory can keep only closed sessions for a selected project', () => {
  const tasks = [
    { ref: 'CORE-10', id: 'work-10', project_id: 'core-eggs-gd', title: 'Ten' },
    { ref: 'CORE-11', id: 'work-11', project_id: 'unity/foo', title: 'Eleven' }
  ];
  const groups = [
    {
      task_ref: 'CORE-10',
      task_id: 'work-10',
      sessions: [
        {
          claim_id: 'live',
          role: 'current',
          status: 'running',
          execution_status: 'running',
          claimed_at: '2026-08-02T11:00:00+03:00'
        },
        {
          claim_id: 'done-10',
          role: 'historical',
          status: 'exited',
          claimed_at: '2026-08-02T09:00:00+03:00'
        }
      ]
    },
    {
      task_ref: 'CORE-11',
      task_id: 'work-11',
      sessions: [
        {
          claim_id: 'done-11',
          role: 'current',
          status: 'exited',
          claimed_at: '2026-08-02T10:00:00+03:00'
        }
      ]
    }
  ];

  const now = Date.parse('2026-08-02T12:00:00+03:00');
  const closed = prepareExecutionHistory(groups, { tasks, closedOnly: true, now });
  assert.equal(closed.groups.length, 2);
  assert.equal(
    closed.groups.find((item) => item.group.task_ref === 'CORE-10')?.visibleSessions.length,
    1
  );
  assert.equal(
    closed.groups.find((item) => item.group.task_ref === 'CORE-10')?.visibleSessions[0].claim_id,
    'done-10'
  );

  const filtered = prepareExecutionHistory(groups, {
    tasks,
    closedOnly: true,
    projectId: 'unity/foo',
    now
  });
  assert.equal(filtered.groups.length, 1);
  assert.equal(filtered.groups[0].group.task_ref, 'CORE-11');
  assert.equal(sessionGroupProjectId(groups[0], tasks), 'core-eggs-gd');
});

test('flattenClosedSessions lists closed sessions as cards without grouping', () => {
  const tasks = [
    { ref: 'CORE-10', id: 'work-10', project_id: 'core-eggs-gd', title: 'Ten' },
    { ref: 'CORE-11', id: 'work-11', project_id: 'unity/foo', title: 'Eleven' }
  ];
  const groups = [
    {
      task_ref: 'CORE-10',
      task_id: 'work-10',
      sessions: [
        {
          claim_id: 'live',
          role: 'current',
          status: 'running',
          execution_status: 'running',
          claimed_at: '2026-08-02T11:00:00+03:00'
        },
        {
          claim_id: 'done-10',
          role: 'historical',
          status: 'exited',
          claimed_at: '2026-08-02T09:00:00+03:00',
          exited_at: '2026-08-02T09:30:00+03:00'
        }
      ]
    },
    {
      task_ref: 'CORE-11',
      task_id: 'work-11',
      sessions: [
        {
          claim_id: 'done-11',
          role: 'current',
          status: 'exited',
          claimed_at: '2026-08-02T10:00:00+03:00',
          exited_at: '2026-08-02T10:20:00+03:00'
        }
      ]
    }
  ];

  const closed = flattenClosedSessions(groups, tasks);
  assert.equal(closed.length, 2);
  assert.equal(closed[0].claim_id, 'done-11');
  assert.equal(closed[0].task_title, 'Eleven');
  assert.equal(closed[1].claim_id, 'done-10');
  assert.equal(closed[1].task_title, 'Ten');
});

test('flattenClosedSessions keeps HITL waiting_input off Closed', () => {
  const groups = [
    {
      task_ref: 'CORE-HITL',
      sessions: [
        {
          claim_id: 'hitl-live',
          role: 'current',
          status: 'waiting_input',
          execution_status: 'waiting_input',
          result: { outcome: 'needs_input' },
          claimed_at: '2026-09-12T10:00:00+03:00'
        }
      ]
    }
  ];
  assert.equal(flattenClosedSessions(groups, []).length, 0);
  assert.equal(sessionIsActive(groups[0].sessions[0]), true);
});

test('flattenClosedSessions caps the result instead of growing without bound', () => {
  const groups = Array.from({ length: 80 }, (_, i) => ({
    task_ref: `CORE-${i}`,
    sessions: [
      {
        claim_id: `done-${i}`,
        role: 'current',
        status: 'exited',
        exited_at: `2026-01-01T00:${String(i % 60).padStart(2, '0')}:00Z`
      }
    ]
  }));
  assert.equal(flattenClosedSessions(groups, []).length, 60);
  assert.equal(flattenClosedSessions(groups, [], 5).length, 5);
  assert.equal(flattenClosedSessions(groups, [], 0).length, 80);
});
