import { isArchivedStatus } from './dashboardState.ts';
import { isOperatorAssignee } from './operator.ts';
import {
  operatorPause,
  priorityValue,
  taskInBacklog,
  taskPickupCompare,
  taskProjectId,
  taskUpdatedCompare
} from './taskDisplay.ts';
import type { Project, Session, Task, Workspace } from './types.ts';

export interface Chip {
  id: string;
  label: string;
  count: number;
}

export interface BoardSelection {
  projectId: string;
  workspaceId: string;
  query: string;
  sort: string;
}

export const selectedProjectOf = (
  projects: Project[],
  workspaces: Workspace[],
  projectId: string,
  workspaceId: string
): Project | Workspace | null =>
  projects.find((project) => project.id === projectId) ||
  workspaces.find((workspace) => workspace.id === workspaceId) ||
  null;

/** Tasks inside the current project/workspace scope matching the search box. */
export const visibleTasksFor = (tasks: Task[], selection: BoardSelection): Task[] => {
  const needle = selection.query.trim().toLowerCase();
  return tasks
    .filter((task) => {
      const identityMatch = selection.projectId
        ? taskProjectId(task) === selection.projectId
        : !selection.workspaceId || task.workspace_id === selection.workspaceId;
      const haystack =
        `${task.ref} ${task.title} ${task.type} ${task.project_id} ${task.project} ${task.repositories?.join(' ') ?? ''}`.toLowerCase();
      return identityMatch && (!needle || haystack.includes(needle));
    })
    .sort(selection.sort === 'priority' ? taskPickupCompare : taskUpdatedCompare);
};

export const backlogTasksFor = (visible: Task[], liveSessions: Session[]): Task[] =>
  visible.filter((task) => taskInBacklog(task, liveSessions));

export const activeTasksFor = (visible: Task[]): Task[] =>
  visible.filter((task) => !isArchivedStatus(task.status));

export const archivedTasksFor = (visible: Task[]): Task[] =>
  visible.filter((task) => isArchivedStatus(task.status));

/** The quick filters that look beyond the backlog, at every non-archived task. */
const WHOLE_BOARD_FILTERS = new Set(['blocked', 'review', 'done', 'attention']);

export const listTasksFor = (quickFilter: string, active: Task[], backlog: Task[]): Task[] =>
  WHOLE_BOARD_FILTERS.has(quickFilter) ? active : backlog;

export const matchesQuickFilter = (task: Task, quickFilter: string): boolean => {
  switch (quickFilter) {
    case 'open':
      return task.status !== 'done';
    case 'rework':
      return task.status === 'needs_rework';
    case 'mine':
      return isOperatorAssignee(task.assignee);
    case 'high':
      return priorityValue(task) <= 2;
    case 'bug':
      return task.type === 'bug';
    case 'feature':
      return task.type === 'feature';
    case 'blocked':
      return task.status === 'blocked';
    case 'review':
      return task.status === 'needs_review';
    case 'done':
      return task.status === 'done';
    case 'attention':
      return (
        task.status === 'blocked' || task.status === 'needs_review' || operatorPause(task).waiting
      );
    default:
      return true;
  }
};

// The Sidebar's Views list can set quickFilter to a value with no matching
// toolbar chip (blocked/review/done/attention) — without a label for it the
// chip row would show no active state while the list is silently filtered.
const SIDEBAR_ONLY_CHIP_LABELS: Record<string, string> = {
  blocked: 'Blocked',
  review: 'In Review',
  done: 'Done',
  attention: 'Needs Attention'
};

const FIXED_CHIPS = ['all', 'open', 'rework', 'mine', 'high', 'bug', 'feature'];
const FIXED_CHIP_LABELS: Record<string, string> = {
  all: 'All',
  open: 'Open',
  rework: 'Rework',
  mine: 'Mine',
  high: 'High',
  bug: 'Bug',
  feature: 'Feature'
};

export const chipsFor = (backlog: Task[], quickFilter: string, filteredCount: number): Chip[] => {
  const chips = FIXED_CHIPS.map((id) => ({
    id,
    label: FIXED_CHIP_LABELS[id],
    count: backlog.filter((task) => matchesQuickFilter(task, id)).length
  }));
  const extra = SIDEBAR_ONLY_CHIP_LABELS[quickFilter];
  return extra ? [...chips, { id: quickFilter, label: extra, count: filteredCount }] : chips;
};
