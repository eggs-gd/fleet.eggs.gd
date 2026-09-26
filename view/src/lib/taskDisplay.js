/** Pure display helpers shared by backoffice dashboard components. */

export const assigneeLabel = (value) => value || 'unassigned';

export const priorityValue = (task) => {
  const value = Number(task?.priority);
  return value >= 1 && value <= 5 ? value : 5;
};

export const priorityLabel = (task) => `P${priorityValue(task)}`;

export const refNumber = (task) =>
  Number((task?.ref || '').split('-')[1]) || Number.MAX_SAFE_INTEGER;

export const taskPickupCompare = (a, b) => {
  const priorityDiff = priorityValue(a) - priorityValue(b);
  if (priorityDiff) return priorityDiff;
  const refDiff = refNumber(a) - refNumber(b);
  if (refDiff) return refDiff;
  return (a.relative_path || '').localeCompare(b.relative_path || '');
};

export const taskUpdatedCompare = (a, b) => {
  const stamp = (task) => task?.updated_at || task?.created_at || '';
  const diff = stamp(b).localeCompare(stamp(a));
  if (diff) return diff;
  return taskPickupCompare(a, b);
};

export const taskAgent = (task) =>
  task?.execution?.agent || task?.launch?.agent || task?.assignee || '';

export const hasProjectTasks = (tasks, workspaceId) =>
  tasks.some((task) => task.workspace_id === workspaceId);

export const workspaceRank = (tasks, workspace) => {
  return hasProjectTasks(tasks, workspace.id) ? 1 : 2;
};

export const workspaceTitle = (workspaces, workspaceId) =>
  workspaces.find((workspace) => workspace.id === workspaceId)?.title || workspaceId;

export const projectById = (projects, projectId) =>
  projects.find((project) => project.id === projectId) || {
    id: projectId,
    title: projectId,
    workspace_id: projectId.includes('/') ? projectId.split('/')[0] : projectId,
    repositories: []
  };

export const projectTitle = (projects, projectId) =>
  projectById(projects, projectId).title || projectId;

export const columnStatuses = (column) => column.statuses || [column.id];

export const taskProjectId = (task) => task.project_id || task.project || task.workspace_id;

export const launchEvaluation = (task) => task.launch_evaluation || {};

export const launchOutcome = (task) => launchEvaluation(task).outcome || '';

export const execution = (task) => task.execution || { state: 'none', visibility_mode: 'unknown' };

export const executionState = (task) => execution(task).state || 'none';

export const executionVisibility = (task) => execution(task).visibility_mode || 'unknown';

export const executionPointer = (task) => execution(task).session_pointer || '';

export const executionLabel = (task) => `${executionState(task)} · ${executionVisibility(task)}`;

export const launchVisibility = (task) => launchEvaluation(task).visibility_mode || '';

export const launchVisibilityLabel = (task) => {
  const mode = launchVisibility(task);
  return mode ? `visibility ${mode}` : '';
};

export const executionClass = (task) => {
  const state = executionState(task);
  if (['running', 'starting', 'claimed', 'resumable'].includes(state)) return 'execution-live';
  if (['waiting_input', 'operator_attention', 'stalled'].includes(state))
    return 'execution-waiting';
  if (['orphaned', 'dead', 'terminal', 'unknown'].includes(state)) return 'execution-orphaned';
  if (['failed', 'timed_out', 'cancelled'].includes(state)) return 'execution-failed';
  if (['succeeded', 'completed'].includes(state)) return 'execution-complete';
  return 'execution-none';
};

export const showExecutionSignal = (task) =>
  task.status === 'doing' || executionState(task) !== 'none';

export const executionReason = (task) =>
  execution(task).blocking_reason || execution(task).terminal_reason || '';

const pauseStates = ['waiting_input', 'operator_attention', 'stalled'];

export const latestCommentText = (task) => {
  const comments = task?.comments || [];
  for (let i = comments.length - 1; i >= 0; i -= 1) {
    const text = String(comments[i]?.text || '').trim();
    if (text) return text;
  }
  return '';
};

export const operatorPause = (task) => {
  const state = executionState(task);
  if (pauseStates.includes(state)) {
    return { waiting: true, question: executionReason(task) };
  }
  if (task?.status !== 'doing') return { waiting: false, question: '' };
  const text = latestCommentText(task);
  if (!/waiting on operator input/i.test(text)) return { waiting: false, question: '' };
  const match = text.match(
    /Question:\s*([\s\S]*?)(?=\s+Artifacts:|\s+Session log:|\s+Tests\/checks:|$)/
  );
  return { waiting: true, question: (match?.[1] || text).trim() };
};

export const cardStatusLabel = (task) => {
  if (operatorPause(task).waiting) return 'waiting for a decision';
  return String(task?.status || '').replaceAll('_', ' ');
};

export const sessionCoversTask = (session, task) => {
  if (!session || !task) return false;
  return (
    (session.task_path &&
      (task.path === session.task_path || task.relative_path === session.task_path)) ||
    (session.task_id && task.id === session.task_id) ||
    (session.task_ref && task.ref === session.task_ref)
  );
};

export const taskShownOnRight = (task) =>
  task?.status === 'done' ||
  task?.status === 'blocked' ||
  task?.status === 'needs_review' ||
  operatorPause(task).waiting;

export const taskInBacklog = (task, liveSessions = []) => {
  if (!task || task.status === 'archived') return false;
  if (taskShownOnRight(task)) return false;
  return !liveSessions.some((session) => sessionCoversTask(session, task));
};

export const providerErrorReason = (record) => record?.provider_error?.reason || '';

export const providerErrorDetail = (record) => record?.provider_error?.detail || '';

export const providerErrorKind = (record) => record?.provider_error?.kind || '';

export const providerErrorSuggestedAction = (record) =>
  record?.provider_error?.suggested_action || '';

export const providerErrorRetryPolicy = (record) => record?.provider_error?.retry_policy || '';

export const sessionRoleClass = (session) => `session-role role-${session?.role || 'historical'}`;

export const sessionResumeLabel = (session) => {
  if (!session?.resume_attempted) return '';
  if (session.resume_outcome === 'resumed') return 'resumed prior session';
  if (session.resume_outcome === 'fallback_new') return 'resume failed, started new session';
  return 'resume in progress';
};

export const remoteThreadTitle = (record) =>
  record?.codex_thread_title || record?.thread_title || '';

export const remoteHostLabel = (record) => record?.host_name || record?.host_id || '';

export const codexRemoteInstruction = (record) => {
  const title =
    remoteThreadTitle(record) ||
    record?.task_ref ||
    record?.task_id ||
    record?.thread_id ||
    record?.codex_thread_id ||
    '';
  const host = remoteHostLabel(record);
  const project = record?.repository || record?.project_id || '';
  return [
    'Open Codex Remote',
    host ? `host ${host}` : '',
    project ? `project ${project}` : '',
    title ? `thread ${title}` : ''
  ]
    .filter(Boolean)
    .join(' · ');
};

export const orphanRecoveryLabel = (orphan) => {
  const state = orphan?.execution_state || 'orphaned';
  if (state === 'resumable') return 'orphaned but resumable';
  if (state === 'dead' || state === 'terminal') return 'orphaned and dead';
  if (state === 'unknown') return 'unknown provider capability';
  return state;
};

export const orphanCapabilitySummary = (orphan) => {
  const caps = orphan?.capabilities;
  if (!caps) return '';
  return [
    `detect ${caps.can_detect_running_session || 'unknown'}`,
    `resume ${caps.can_resume_session || 'unknown'}`,
    `query ${caps.can_query_thread_status || 'unknown'}`,
    `link ${caps.can_show_app_visible_link || 'unknown'}`,
    `input ${caps.can_accept_operator_input || 'unknown'}`,
    `terminal ${caps.can_confirm_terminal_outcome || 'unknown'}`
  ].join(' · ');
};

export const executionIdentity = (record) =>
  [
    remoteThreadTitle(record) ? `title ${remoteThreadTitle(record)}` : '',
    remoteHostLabel(record) ? `host ${remoteHostLabel(record)}` : '',
    record?.claim_id ? `claim ${record.claim_id}` : '',
    record?.process_id ? `pid ${record.process_id}` : '',
    record?.thread_id ? `thread ${record.thread_id}` : '',
    record?.codex_thread_id ? `thread ${record.codex_thread_id}` : '',
    record?.cursor_chat_id ? `cursor ${record.cursor_chat_id}` : '',
    record?.session_id && record.session_id !== record.thread_id
      ? `session ${record.session_id}`
      : '',
    record?.turn_id ? `turn ${record.turn_id}` : '',
    record?.codex_turn_id ? `turn ${record.codex_turn_id}` : '',
    record?.log_path ? record.log_path : ''
  ].filter(Boolean);

export const toolUsage = (record) => record?.tool_usage || null;

export const toolUsageWarning = (record) =>
  record?.tool_usage?.warning || record?.tool_warning || '';

export const toolUsageHasMissing = (record) => {
  const missing = record?.tool_usage?.missing;
  return Array.isArray(missing) && missing.length > 0;
};

export const toolUsageProfiles = (record) => {
  const profiles = record?.tool_usage?.profiles;
  return Array.isArray(profiles) ? profiles : [];
};

export const toolUsageRequired = (record) => {
  const required = record?.tool_usage?.required;
  return Array.isArray(required) ? required : [];
};

export const toolUsageUsed = (record) => {
  const used = record?.tool_usage?.used;
  return Array.isArray(used) ? used : [];
};

export const toolUsageMissing = (record) => {
  const missing = record?.tool_usage?.missing;
  return Array.isArray(missing) ? missing : [];
};

// One chip per required tool: "missing X" when unmet, "req X" when observed.
// Do not also render a separate missing list — that produced "req X missing X".
export const formatRequiredToolChip = (tool, record) => {
  if (!tool) return '';
  return toolUsageMissing(record).includes(tool) ? `missing ${tool}` : `req ${tool}`;
};

export const formatToolCall = (call) => {
  if (!call?.name) return '';
  const when = call.called_at ? new Date(call.called_at).toLocaleTimeString() : '';
  const bits = [call.name, call.status, when].filter(Boolean);
  return bits.join(' · ');
};

export async function copyText(value) {
  if (!value) return;
  await navigator.clipboard?.writeText(value);
}

export const launchReason = (task) => {
  const evaluation = launchEvaluation(task);
  if (evaluation.waiting?.reason) return evaluation.waiting.reason;
  return evaluation.failed_gates?.[0] || '';
};

export const blockedReason = (task) =>
  task?.status === 'blocked' ? task?.blocked_reason || launchReason(task) : '';

export const dependsOnList = (task) =>
  Array.isArray(task?.depends_on)
    ? task.depends_on.map((ref) => String(ref || '').trim()).filter(Boolean)
    : [];

export const dependsOnDraft = (task) => dependsOnList(task).join(', ');

export const parseDependsOnDraft = (text) => {
  const seen = new Set();
  const out = [];
  for (const part of String(text || '').split(/[\n,]+/)) {
    const ref = part.trim();
    if (!ref) continue;
    const key = ref.toUpperCase();
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(ref);
  }
  return out;
};

export const showLaunchSignal = (task) => {
  const outcome = launchOutcome(task);
  return outcome && outcome !== 'launchable';
};

const KNOWN_AGENT_COLORS = new Set(['claude', 'codex', 'cursor', 'gemini']);

export const agentColorClass = (agent) => {
  const name = String(agent || '').toLowerCase();
  return KNOWN_AGENT_COLORS.has(name) ? `agent-${name}` : 'agent-other';
};

export const elapsedClock = (startedAt, now = Date.now()) => {
  if (!startedAt) return '';
  const startMs = Date.parse(startedAt);
  if (!Number.isFinite(startMs)) return '';
  const totalMinutes = Math.max(0, Math.floor((now - startMs) / 60000));
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}`;
};

export const timeAgo = (iso, now = Date.now()) => {
  if (!iso) return '';
  const ms = Date.parse(iso);
  if (!Number.isFinite(ms)) return '';
  const diffMinutes = Math.round((now - ms) / 60000);
  if (diffMinutes < 1) return 'just now';
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.round(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.round(diffHours / 24);
  return `${diffDays}d ago`;
};

export const archiveListMode = (selectedProjectId, selectedWorkspaceId) => {
  if (selectedProjectId) return 'flat';
  if (selectedWorkspaceId) return 'projects';
  return 'workspaces';
};

export const groupedByWorkspaceAndProject = (sourceTasks, workspaces, projects) => {
  const workspaceIds = Array.from(new Set(sourceTasks.map((task) => task.workspace_id))).sort(
    (a, b) => workspaceTitle(workspaces, a).localeCompare(workspaceTitle(workspaces, b))
  );
  return workspaceIds.map((workspaceId) => {
    const workspaceTasks = sourceTasks.filter((task) => task.workspace_id === workspaceId);
    const projectIds = Array.from(new Set(workspaceTasks.map(taskProjectId))).sort((a, b) =>
      projectTitle(projects, a).localeCompare(projectTitle(projects, b))
    );
    const source = workspaces.find((workspace) => workspace.id === workspaceId);
    const projectGroups = projectIds.map((projectId) => ({
      id: projectId,
      title: projectTitle(projects, projectId),
      source: projectById(projects, projectId),
      tasks: workspaceTasks.filter((task) => taskProjectId(task) === projectId)
    }));
    const isGroup =
      source?.kind === 'workspace_group' ||
      projectGroups.length > 1 ||
      (projectGroups.length === 1 && projectGroups[0].id !== workspaceId);
    return {
      id: workspaceId,
      title: workspaceTitle(workspaces, workspaceId),
      source,
      isGroup,
      tasks: workspaceTasks,
      projects: projectGroups
    };
  });
};
