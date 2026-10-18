import { derived, get, writable } from 'svelte/store';
import { apiFetch } from '../lib/api';
import {
  applyTaskUpdate,
  reconcileSelectedTaskFromState,
  sameTaskRecord
} from '../lib/dashboardState';
import { operatorAssignee } from '../lib/operator';
import { canTransitionStatus } from '../lib/statusTransitions';
import {
  BLOCKED_REVIEW_COMMENT_ERROR,
  draftFromTask,
  draftNeedsReviewComment,
  emptyTaskDraft,
  taskPatchFromDraft
} from '../lib/taskDraft';
import type { AnyRecord, Task } from '../lib/types';
import { actionError, data, onStateLoaded, projects, tasks, workspaces } from './board';

/** The task being edited or created, and every write to a task. */

export const selectedTask = writable<Task | null>(null);
export const creatingTask = writable(false);
export const draft = writable(emptyTaskDraft());
export const saving = writable(false);
export const creating = writable(false);
export const editError = writable('');
export const createError = writable('');

export const transitions = derived(data, ($data) => $data?.status_transitions ?? null);

// A refresh that no longer lists the open task closes it; one that does hands
// the modal the latest copy, but keeps what the operator has typed so far.
onStateLoaded((state) => {
  const open = get(selectedTask);
  if (!open) return;
  const latest = reconcileSelectedTaskFromState(open, state);
  selectedTask.set(latest);
  if (!latest) editError.set('');
});

function updateTaskInState(updated: Task, fallbackPath = '') {
  const state = get(data);
  if (!state) return;
  const sameOrphan = (orphan: AnyRecord) =>
    (fallbackPath &&
      (orphan.task_path === fallbackPath || orphan.task_path === updated.relative_path)) ||
    (updated.relative_path && orphan.task_path === updated.relative_path) ||
    (updated.id && orphan.task_id === updated.id) ||
    (updated.ref && orphan.task_ref === updated.ref);
  data.set({
    ...state,
    tasks: applyTaskUpdate(state.tasks ?? [], updated, fallbackPath),
    orphaned_tasks:
      updated.status && updated.status !== 'doing'
        ? ((state.orphaned_tasks ?? []) as AnyRecord[]).filter((orphan) => !sameOrphan(orphan))
        : (state.orphaned_tasks ?? [])
  });
  const open = get(selectedTask);
  if (open && sameTaskRecord(open, updated, fallbackPath)) {
    const merged = { ...open, ...updated };
    selectedTask.set(merged);
    draft.set(draftFromTask(merged));
  }
}

const json = (method: string, body: unknown) => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body)
});

export async function patchTask(path: string, patch: AnyRecord) {
  const response = await apiFetch('/api/tasks', json('PATCH', { path, ...patch }));
  if (!response.ok) throw new Error(await response.text());
  const updated = await response.json();
  const local = {
    ...updated,
    ...(patch.status ? { status: patch.status } : {}),
    ...(patch.priority !== undefined ? { priority: patch.priority } : {}),
    ...(patch.assignee !== undefined ? { assignee: patch.assignee } : {}),
    ...(patch.depends_on !== undefined ? { depends_on: patch.depends_on } : {}),
    ...(patch.project
      ? { project: patch.project, project_id: updated.project_id || patch.project }
      : {})
  };
  updateTaskInState(local, path);
  return local;
}

async function createTask(payload: AnyRecord) {
  const response = await apiFetch('/api/tasks', json('POST', payload));
  if (!response.ok) throw new Error(await response.text());
  const created = await response.json();
  updateTaskInState(created);
  return created;
}

export function openTask(task: Task) {
  selectedTask.set(task);
  creatingTask.set(false);
  draft.set(draftFromTask(task));
  editError.set('');
}

export function closeTask() {
  selectedTask.set(null);
  editError.set('');
}

export function openCreateTask() {
  selectedTask.set(null);
  creatingTask.set(true);
  createError.set('');
}

export function closeCreateTask() {
  creatingTask.set(false);
  createError.set('');
}

export async function saveTaskDetails() {
  const task = get(selectedTask);
  if (!task || get(saving)) return;
  const current = get(draft);
  if (draftNeedsReviewComment(task, current)) {
    editError.set(BLOCKED_REVIEW_COMMENT_ERROR);
    return;
  }
  saving.set(true);
  editError.set('');
  try {
    const patch = taskPatchFromDraft(
      task,
      current,
      get(projects),
      get(workspaces),
      operatorAssignee
    );
    await patchTask(task.path || '', patch);
    closeTask();
  } catch (err: any) {
    editError.set(err.message);
  } finally {
    saving.set(false);
  }
}

export async function saveCreatedTask(payload: AnyRecord) {
  if (get(creating)) return;
  creating.set(true);
  createError.set('');
  try {
    const created = await createTask(payload);
    closeCreateTask();
    openTask(created);
  } catch (err: any) {
    createError.set(err.message);
  } finally {
    creating.set(false);
  }
}

/** A status move from a card button; applied at once and rolled back if the server refuses. */
export async function transitionTask(task: Task, status: string) {
  if (!task?.path || !status) return;
  if (!canTransitionStatus(task.status || '', status, get(data)?.status_transitions)) return;
  try {
    actionError.set('');
    updateTaskInState({ ...task, status }, task.path);
    await patchTask(task.path, { status });
  } catch (err: any) {
    actionError.set(err.message);
    updateTaskInState(task, task.path || '');
  }
}

export async function dropTaskOnMilestone(taskPath: string, milestoneId: string) {
  const task = get(tasks).find((item) => (item.path || item.relative_path) === taskPath);
  if (task && (task.milestone || '') === milestoneId) return;
  try {
    await patchTask(taskPath, { milestone: milestoneId });
  } catch (err: any) {
    actionError.set(err.message);
  }
}
