import { apiFetch, readApiError } from './api.ts';
import type { Milestone } from './types.ts';

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
