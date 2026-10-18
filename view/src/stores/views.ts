import { derived } from 'svelte/store';
import {
  activeTasksFor,
  archivedTasksFor,
  backlogTasksFor,
  chipsFor,
  listTasksFor,
  matchesQuickFilter,
  selectedProjectOf,
  visibleTasksFor
} from '../lib/boardView';
import { AGENT_ASSIGNEES } from '../lib/operator';
import { liveSessions, people, projects, tasks, workspaces } from './board';
import { query, quickFilter, selectedProjectId, selectedWorkspaceId, taskSort } from './location';

/** What the board shows for the current selection, search and filter. */

export const selectedProject = derived(
  [projects, workspaces, selectedProjectId, selectedWorkspaceId],
  ([$projects, $workspaces, $project, $workspace]) =>
    selectedProjectOf($projects, $workspaces, $project, $workspace)
);

const visibleTasks = derived(
  [tasks, selectedProjectId, selectedWorkspaceId, query, taskSort],
  ([$tasks, projectId, workspaceId, $query, sort]) =>
    visibleTasksFor($tasks, { projectId, workspaceId, query: $query, sort })
);

const activeTasks = derived(visibleTasks, activeTasksFor);
export const archivedTasks = derived(visibleTasks, archivedTasksFor);
const backlogTasks = derived([visibleTasks, liveSessions], ([$visible, $live]) =>
  backlogTasksFor($visible, $live)
);

/** The tasks the board list renders after the quick filter. */
export const filteredActiveTasks = derived(
  [quickFilter, activeTasks, backlogTasks],
  ([$filter, $active, $backlog]) =>
    listTasksFor($filter, $active, $backlog).filter((task) => matchesQuickFilter(task, $filter))
);

export const chips = derived(
  [backlogTasks, quickFilter, filteredActiveTasks],
  ([$backlog, $filter, $filtered]) => chipsFor($backlog, $filter, $filtered.length)
);

/** Who a task can be given to: nobody, the registered people, or an AI agent. */
export const assigneeChoices = derived(people, ($people) => [
  'unassigned',
  ...$people,
  ...AGENT_ASSIGNEES
]);
