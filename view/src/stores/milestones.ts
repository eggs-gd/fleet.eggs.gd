import { derived, get, writable } from 'svelte/store';
import {
  createMilestone,
  deleteMilestone,
  milestoneCardId as cardIdOf,
  patchMilestone
} from '../lib/milestones';
import type { MilestoneTaskGroup } from '../lib/taskDisplay';
import type { Project } from '../lib/types';
import { actionError, projects, requestManualRefresh, tasks } from './board';
import { selectedProjectId } from './location';
import { selectedProject } from './views';

/** Milestone create/rename/delete from the work board. */

export interface EditingMilestone {
  cardId: string;
  id: string;
  title: string;
  version: string;
}

export const creatingMilestone = writable(false);
export const editingMilestone = writable<EditingMilestone | null>(null);
export const savingMilestone = writable(false);
export const milestoneError = writable('');

// Milestones live on the workspace card, so a repository project adds them
// to its parent; nested task projects have no card of their own.
export const milestoneCardId = derived(
  [selectedProject, selectedProjectId],
  ([$project, $projectId]) => ($projectId ? cardIdOf($project as Project | null) : '')
);

export function openCreateMilestone() {
  creatingMilestone.set(true);
  milestoneError.set('');
}

export function closeCreateMilestone() {
  creatingMilestone.set(false);
  milestoneError.set('');
}

export function startEditMilestone(group: MilestoneTaskGroup) {
  const cardId = cardIdOf(get(projects).find((project) => project.id === group.projectId));
  if (!cardId || !group.milestoneId) return;
  editingMilestone.set({
    cardId,
    id: group.milestoneId,
    title: group.milestoneTitle,
    version: group.version || ''
  });
  milestoneError.set('');
}

export function closeEditMilestone() {
  editingMilestone.set(null);
  milestoneError.set('');
}

async function save(write: () => Promise<unknown>, close: () => void) {
  if (get(savingMilestone)) return;
  savingMilestone.set(true);
  milestoneError.set('');
  try {
    await write();
    close();
    await requestManualRefresh();
  } catch (err: any) {
    milestoneError.set(err.message);
  } finally {
    savingMilestone.set(false);
  }
}

export async function saveCreatedMilestone(title: string, version: string) {
  const cardId = get(milestoneCardId);
  if (!cardId) return;
  await save(() => createMilestone(cardId, title, version), closeCreateMilestone);
}

export async function saveEditedMilestone(title: string, version: string) {
  const editing = get(editingMilestone);
  if (!editing) return;
  await save(
    () => patchMilestone(editing.cardId, editing.id, { title, version }),
    closeEditMilestone
  );
}

export async function removeMilestone(group: MilestoneTaskGroup) {
  const cardId = cardIdOf(get(projects).find((project) => project.id === group.projectId));
  if (!cardId || !group.milestoneId) return;
  // Count from the whole board, not the rendered group: the delete clears
  // every task of the workspace (archived, filtered out, sibling projects).
  const count = get(tasks).filter(
    (task) => task.workspace_id === cardId && task.milestone === group.milestoneId
  ).length;
  const noun = count === 1 ? 'task' : 'tasks';
  const detail = count ? ` ${count} ${noun} will move to Unscheduled.` : '';
  if (!globalThis.confirm(`Delete milestone "${group.milestoneTitle}"?${detail}`)) return;
  try {
    await deleteMilestone(cardId, group.milestoneId);
    await requestManualRefresh();
  } catch (err: any) {
    actionError.set(err.message);
  }
}
