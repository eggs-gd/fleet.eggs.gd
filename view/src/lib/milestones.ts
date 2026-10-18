import { apiFetch, readApiError } from './api.ts';
import type { Milestone, Project } from './types.ts';

export async function fetchMilestones(projectId: string): Promise<Milestone[]> {
  const response = await apiFetch(`/api/projects/${encodeURIComponent(projectId)}/milestones`);
  if (!response.ok) {
    throw new Error(await readApiError(response));
  }
  return (await response.json()) ?? [];
}

export async function createMilestone(
  projectId: string,
  title: string,
  version?: string
): Promise<Milestone> {
  const response = await apiFetch(`/api/projects/${encodeURIComponent(projectId)}/milestones`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, version: version || undefined })
  });
  if (!response.ok) {
    throw new Error(await readApiError(response));
  }
  return await response.json();
}

export async function patchMilestone(
  projectId: string,
  id: string,
  patch: { state?: string; title?: string; version?: string }
): Promise<Milestone> {
  const response = await apiFetch(`/api/projects/${encodeURIComponent(projectId)}/milestones`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id, ...patch })
  });
  if (!response.ok) {
    throw new Error(await readApiError(response));
  }
  return await response.json();
}

/** Milestones live on the workspace card, so a repository project of a group
 * reads and writes its parent's. Nested task projects have no card ('' here). */
export function milestoneCardId(
  project: Pick<Project, 'id' | 'kind' | 'workspace_id'> | null | undefined
): string {
  if (!project) return '';
  if (project.kind === 'workspace') return project.id;
  if (project.kind === 'repository_project') return project.workspace_id || '';
  return '';
}

/** Removes a milestone and returns how many tasks it was cleared from. */
export async function deleteMilestone(projectId: string, id: string): Promise<number> {
  const response = await apiFetch(`/api/projects/${encodeURIComponent(projectId)}/milestones`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id })
  });
  if (!response.ok) {
    throw new Error(await readApiError(response));
  }
  return (await response.json())?.cleared ?? 0;
}
