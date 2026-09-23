// Mirrors `_backoffice/server/internal/tasklifecycle/status_transition.go`.
// Prefer `/api/state` `status_transitions` when present.

export const STATUS_TRANSITIONS = {
  backlog: ['todo', 'archived'],
  todo: ['doing', 'blocked', 'backlog', 'archived'],
  needs_rework: ['doing', 'blocked', 'todo', 'archived'],
  doing: ['needs_review', 'needs_rework', 'blocked', 'todo', 'done'],
  blocked: ['needs_review', 'needs_rework', 'todo', 'archived'],
  needs_review: ['needs_rework', 'todo', 'done', 'archived'],
  done: ['archived'],
  archived: ['backlog']
};

const ACTION_ORDER = ['done', 'todo', 'needs_review', 'backlog', 'archived', 'blocked'];

const ACTION_META = {
  todo: { label: 'To work', icon: 'play' },
  blocked: { label: 'Block', icon: 'x-circle' },
  needs_review: { label: 'Review', icon: 'check-circle' },
  done: { label: 'Complete', icon: 'checks-circle' },
  archived: { label: 'Archive', icon: 'archive' },
  backlog: { label: 'Restore', icon: 'undo' }
};

const nextStatuses = (from, transitions = STATUS_TRANSITIONS) => {
  const source = transitions && Object.keys(transitions).length ? transitions : STATUS_TRANSITIONS;
  return source[from] ?? STATUS_TRANSITIONS[from] ?? [];
};

export const canTransitionStatus = (from, to, transitions) =>
  from === to || nextStatuses(from, transitions).includes(to);

export const statusAction = (from, to) => {
  const meta = ACTION_META[to];
  if (!meta) return null;
  let label = meta.label;
  if (to === 'backlog' && from === 'todo') label = 'Backlog';
  return {
    status: to,
    label,
    icon: meta.icon,
    title: `${label}`
  };
};

// Operator plaque/list buttons. `doing` is the launcher's claim, not a
// one-click PATCH. `needs_rework` is a board status, not the "send back
// to the queue" action — that is `todo` / To work.
export const actionsForStatus = (status, transitions) => {
  const allowed = new Set(nextStatuses(status, transitions));
  return ACTION_ORDER.filter((to) => allowed.has(to))
    .map((to) => statusAction(status, to))
    .filter(Boolean);
};
