export const taskKey = (task) => task?.path || task?.relative_path || task?.id || task?.ref || '';

export const sessionKey = (session) =>
  session?.claim_id || `${session?.task_ref || session?.task_id || 'session'}:${session?.agent || ''}:${session?.repository || session?.project_id || ''}`;

export const orphanKey = (orphan) =>
  `${orphan?.task_path || orphan?.task_id || orphan?.task_ref || 'orphan'}:${orphan?.assignee || ''}:${orphan?.repository || orphan?.project_id || ''}`;

export const sessionGroupKey = (group) => group?.task_path || group?.task_id || group?.task_ref || 'session-group';

export const workspaceAccordionKey = (workspace) => `workspace:${workspace?.id || ''}`;

export const projectAccordionKey = (project) => `project:${project?.id || ''}`;

export const repositoryAccordionKey = (project, repo) =>
  `repository:${project?.id || ''}:${repo?.relative_path || repo?.path || repo?.id || ''}`;

export const executionHistoryPanelKey = () => 'execution-history:panel';

export const executionHistoryGroupKey = (group) => `execution-history:${sessionGroupKey(group)}`;

export const archiveWorkspaceAccordionKey = (workspace) => `archive-workspace:${workspace?.id || ''}`;

export const archiveProjectAccordionKey = (project) => `archive-project:${project?.id || ''}`;

export const EXECUTION_HISTORY_LIMITS = {
  maxTaskGroups: 12,
  maxSessionsPerTask: 5,
  hideTerminalOlderThanMs: 24 * 60 * 60 * 1000
};

const ACTIVE_SESSION_STATUSES = new Set([
  'queued',
  'claimed',
  'starting',
  'running',
  'waiting_input',
  'operator_attention',
  'stalled',
  'resumable'
]);

export const isAccordionOpen = (accordionOpen, key, defaultOpen = false) => accordionOpen[key] ?? defaultOpen;

export const updateAccordionOpen = (accordionOpen, key, open) => ({
  ...accordionOpen,
  [key]: open
});

export const sameTaskRecord = (a, b, fallbackPath = '') => {
  if (!a || !b) return false;
  if (fallbackPath && (a.path === fallbackPath || a.relative_path === fallbackPath)) {
    return true;
  }
  return (
    (a.path && b.path && a.path === b.path) ||
    (a.relative_path && b.relative_path && a.relative_path === b.relative_path) ||
    (a.path && b.relative_path && a.path === b.relative_path) ||
    (a.relative_path && b.path && a.relative_path === b.path) ||
    (a.id && b.id && a.id === b.id) ||
    (a.ref && b.ref && a.ref === b.ref)
  );
};

export const applyTaskUpdate = (tasks, updatedTask, fallbackPath = '') => {
  const list = Array.isArray(tasks) ? tasks : [];
  if (!updatedTask) return list;

  let merged = null;
  const next = [];
  for (const task of list) {
    if (!sameTaskRecord(task, updatedTask, fallbackPath)) {
      next.push(task);
      continue;
    }
    if (merged) {
      // Drop identity duplicates so a status move cannot leave a backlog copy behind.
      continue;
    }
    merged = { ...task, ...updatedTask };
    next.push(merged);
  }
  if (!merged) {
    next.push({ ...updatedTask });
  }
  return next;
};

export const preferTaskRecord = (a, b) => {
  if (!a) return b;
  if (!b) return a;
  if (a.updated_at && b.updated_at && a.updated_at !== b.updated_at) {
    return a.updated_at > b.updated_at ? { ...b, ...a } : { ...a, ...b };
  }
  if (a.status === 'backlog' && b.status && b.status !== 'backlog') {
    return { ...a, ...b };
  }
  if (b.status === 'backlog' && a.status && a.status !== 'backlog') {
    return { ...b, ...a };
  }
  return { ...a, ...b };
};

export const dedupeTasksByIdentity = (tasks) => {
  const list = Array.isArray(tasks) ? tasks : [];
  const next = [];
  for (const task of list) {
    const duplicateIndex = next.findIndex((existing) => sameTaskRecord(existing, task));
    if (duplicateIndex >= 0) {
      next[duplicateIndex] = preferTaskRecord(next[duplicateIndex], task);
      continue;
    }
    next.push(task);
  }
  return next;
};

export const isArchivedStatus = (status) => status === 'archived';

export const isBoardStatus = (status) =>
  Boolean(status) && status !== 'backlog' && status !== 'archived';

export const partitionTasksByBoard = (tasks) => {
  const list = Array.isArray(tasks) ? tasks : [];
  return {
    backlog: list.filter((task) => task.status === 'backlog'),
    board: list.filter((task) => isBoardStatus(task.status)),
    archived: list.filter((task) => isArchivedStatus(task.status))
  };
};

const techRepositories = (item) => item?.technology?.repositories || [];

export const accordionKeysForState = (state) => {
  const keys = new Set([executionHistoryPanelKey()]);
  for (const workspace of state?.workspaces ?? []) {
    keys.add(workspaceAccordionKey(workspace));
    keys.add(archiveWorkspaceAccordionKey(workspace));
  }
  for (const project of state?.projects ?? []) {
    keys.add(projectAccordionKey(project));
    keys.add(archiveProjectAccordionKey(project));
    for (const repo of techRepositories(project)) {
      keys.add(repositoryAccordionKey(project, repo));
    }
  }
  for (const group of state?.session_groups ?? []) {
    keys.add(executionHistoryGroupKey(group));
  }
  return keys;
};

export const pruneAccordionOpen = (accordionOpen, state) => {
  const keys = accordionKeysForState(state);
  const nextOpen = {};
  let changed = false;
  for (const [key, open] of Object.entries(accordionOpen)) {
    if (keys.has(key)) {
      nextOpen[key] = open;
    } else {
      changed = true;
    }
  }
  return changed ? nextOpen : accordionOpen;
};

export const reconcileSelectedTaskFromState = (selectedTask, state) => {
  if (!selectedTask) return null;
  return (state?.tasks ?? []).find((task) => sameTaskRecord(task, selectedTask)) || null;
};

export const sessionIsActive = (session) =>
  ACTIVE_SESSION_STATUSES.has(session?.execution_status || session?.status || '');

const OPERATOR_RELEASE_STATUSES = new Set([
  ...ACTIVE_SESSION_STATUSES,
  'dead',
  'provider_unavailable',
  'control_socket_unavailable',
  'orphaned',
  'unknown'
]);

export const sessionAllowsOperatorRelease = (session) => {
  if (!session?.claim_id) return false;
  if (session.provider_controllable) return false;
  const state = session.execution_status || session.status || '';
  return OPERATOR_RELEASE_STATUSES.has(state);
};

export const sessionActivityAt = (session) =>
  session?.last_output_at ||
  session?.last_event_at ||
  session?.last_status_change_at ||
  session?.exited_at ||
  session?.claimed_at ||
  '';

export const groupLatestSession = (group) => {
  const sessions = group?.sessions ?? [];
  return sessions.find((session) => session?.role === 'current') || sessions[sessions.length - 1] || null;
};

export const groupActivityAt = (group) => {
  let latest = '';
  for (const session of group?.sessions ?? []) {
    const at = sessionActivityAt(session);
    if (at > latest) latest = at;
  }
  return latest;
};

export const taskTitleForSessionGroup = (group, tasks = []) => {
  const match = (tasks ?? []).find(
    (task) =>
      (group?.task_path && (task.path === group.task_path || task.relative_path === group.task_path)) ||
      (group?.task_id && task.id === group.task_id) ||
      (group?.task_ref && task.ref === group.task_ref)
  );
  return match?.title || '';
};

const parseTimestampMs = (value) => {
  if (!value) return NaN;
  const ms = Date.parse(value);
  return Number.isFinite(ms) ? ms : NaN;
};

export const visibleSessionsForGroup = (
  group,
  { expanded = false, now = Date.now(), limits = EXECUTION_HISTORY_LIMITS } = {}
) => {
  const sessions = [...(group?.sessions ?? [])].reverse();
  const cutoff = now - (limits.hideTerminalOlderThanMs ?? EXECUTION_HISTORY_LIMITS.hideTerminalOlderThanMs);
  const filtered = sessions.filter((session) => {
    if (session?.role === 'current' || sessionIsActive(session)) return true;
    if (expanded) return true;
    const activityMs = parseTimestampMs(sessionActivityAt(session));
    if (!Number.isFinite(activityMs)) return true;
    return activityMs >= cutoff;
  });
  const maxSessions = limits.maxSessionsPerTask ?? EXECUTION_HISTORY_LIMITS.maxSessionsPerTask;
  const limited = filtered.slice(0, maxSessions);
  const hiddenCount = Math.max(0, (group?.sessions?.length ?? 0) - limited.length);
  return { sessions: limited, hiddenCount };
};

/** Delay before quiet background polls show a busy indicator (avoids 5s flash). */
export const REFRESH_BUSY_DELAY_MS = 320;

/** Fixed-width clock for the last-updated slot (tabular digits, padded fields). */
export const formatRefreshClock = (
  iso,
  formatTime = (value) =>
    new Date(value).toLocaleTimeString(undefined, {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
) => {
  if (!iso) return '';
  try {
    return formatTime(iso) || '';
  } catch {
    return '';
  }
};

/**
 * Whether the refresh strip should show a busy affordance.
 * Manual refresh is immediate; auto-poll waits so fast polls stay visually quiet.
 */
export const shouldShowRefreshBusy = ({
  manual = false,
  elapsedMs = 0,
  delayMs = REFRESH_BUSY_DELAY_MS
} = {}) => Boolean(manual) || elapsedMs >= delayMs;

/** Stable labels for the dashboard auto-refresh strip (reserved-width UI). */
export const refreshStatusPresentation = ({
  refreshing = false,
  refreshError = '',
  lastRefreshAt = '',
  formatTime
} = {}) => {
  const updatedAt = lastRefreshAt || '';
  const clock = formatRefreshClock(updatedAt, formatTime);
  const updatedText = clock ? `Updated ${clock}` : '';
  return {
    label: refreshError ? `Auto-refresh failed: ${refreshError}` : 'Auto-refresh every 5s',
    warning: Boolean(refreshError),
    busy: Boolean(refreshing),
    busyLabel: 'Refreshing',
    updatedAt,
    updatedText
  };
};

export const sessionGroupProjectId = (group, tasks = []) => {
  const task = (tasks ?? []).find(
    (item) =>
      (group?.task_path && (item.path === group.task_path || item.relative_path === group.task_path)) ||
      (group?.task_id && item.id === group.task_id) ||
      (group?.task_ref && item.ref === group.task_ref)
  );
  if (task) return task.project_id || task.project || task.workspace_id || '';
  const session = groupLatestSession(group);
  return session?.project_id || '';
};

export const closedSessionsInGroup = (group) =>
  (group?.sessions ?? []).filter((session) => !sessionIsActive(session));

const sessionWithGroupContext = (session, group, tasks = []) => {
  const projectId = session.project_id || sessionGroupProjectId(group, tasks);
  return {
    ...session,
    task_title: session.task_title || taskTitleForSessionGroup(group, tasks),
    task_ref: session.task_ref || group.task_ref,
    task_id: session.task_id || group.task_id,
    task_path: session.task_path || group.task_path,
    project_id: projectId,
    repository: session.repository || session.project_id || projectId
  };
};

// Unbounded on a long-lived project this would re-render every closed session
// ever recorded on every 5s poll; cap it to the same order of magnitude the
// old grouped Execution History view used (12 task groups x 5 sessions).
export const MAX_CLOSED_SESSIONS = 60;

export const flattenClosedSessions = (groups, tasks = [], limit = MAX_CLOSED_SESSIONS) => {
  const rows = [];
  for (const group of groups ?? []) {
    for (const session of closedSessionsInGroup(group)) {
      rows.push(sessionWithGroupContext(session, group, tasks));
    }
  }
  rows.sort((a, b) => String(sessionActivityAt(b)).localeCompare(String(sessionActivityAt(a))));
  return limit ? rows.slice(0, limit) : rows;
};

export const prepareExecutionHistory = (
  groups,
  {
    tasks = [],
    now = Date.now(),
    limits = EXECUTION_HISTORY_LIMITS,
    expandedGroupKeys = new Set(),
    projectId = '',
    closedOnly = false
  } = {}
) => {
  const maxGroups = limits.maxTaskGroups ?? EXECUTION_HISTORY_LIMITS.maxTaskGroups;
  const source = (groups ?? [])
    .map((group) => {
      if (!closedOnly) return group;
      const sessions = closedSessionsInGroup(group);
      return sessions.length ? { ...group, sessions } : null;
    })
    .filter(Boolean)
    .filter((group) => !projectId || sessionGroupProjectId(group, tasks) === projectId);
  const ranked = source
    .map((group) => {
      const latest = groupLatestSession(group);
      const key = sessionGroupKey(group);
      const accordionKey = executionHistoryGroupKey(group);
      const expanded = expandedGroupKeys.has(accordionKey);
      const visible = visibleSessionsForGroup(group, { expanded, now, limits });
      return {
        key,
        accordionKey,
        group,
        title: taskTitleForSessionGroup(group, tasks),
        latest,
        activityAt: groupActivityAt(group),
        sessionCount: group?.sessions?.length ?? 0,
        agent: latest?.agent || latest?.provider || '',
        status: latest?.execution_status || latest?.status || '',
        expanded,
        visibleSessions: visible.sessions,
        hiddenSessionCount: visible.hiddenCount
      };
    })
    .sort((a, b) => {
      if (a.activityAt !== b.activityAt) return a.activityAt < b.activityAt ? 1 : -1;
      return a.key.localeCompare(b.key);
    });

  const visibleGroups = ranked.slice(0, maxGroups);
  return {
    totalGroups: ranked.length,
    hiddenGroupCount: Math.max(0, ranked.length - visibleGroups.length),
    groups: visibleGroups
  };
};
