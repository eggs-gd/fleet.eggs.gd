import { apiFetch } from '../lib/api';
import { operatorAssignee } from '../lib/operator';
import type { AnyRecord, Session } from '../lib/types';
import { error, loadData } from './board';
import { patchTask } from './tasks';

/** Operator actions on running (or orphaned) agent sessions. */

export async function controlSession(
  session: Session,
  action: string,
  input = '',
  targetStatus = ''
) {
  const payload: AnyRecord = { claim_id: session.claim_id, action, input };
  if (targetStatus) payload.target_status = targetStatus;
  const response = await apiFetch('/api/sessions/control', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  if (!response.ok) throw new Error(await response.text());
  await loadData();
}

export async function releaseSession(
  session: Session,
  action = 'release',
  targetStatus = 'needs_review'
) {
  try {
    error.set('');
    await controlSession(session, action, '', targetStatus);
  } catch (err: any) {
    error.set(err.message);
  }
}

/** Settles a task whose runtime session vanished, as blocked or by moving it on. */
export async function resolveOrphan(orphan: AnyRecord, status: string) {
  const path = orphan.task_path;
  if (!path) return;
  try {
    await patchTask(path, {
      status,
      comment:
        status === 'blocked'
          ? `Marked blocked from orphaned runtime state: ${orphan.blocking_reason || orphan.reason || 'runtime session is not active'}`
          : `Resolved orphaned runtime state by moving task to ${status}.`,
      comment_author: operatorAssignee
    });
    await loadData();
  } catch (err: any) {
    error.set(err.message);
  }
}
