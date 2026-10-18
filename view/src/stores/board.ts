import { derived, get, writable } from 'svelte/store';
import { apiFetch } from '../lib/api';
import {
  dedupeTasksByIdentity,
  flattenClosedSessions,
  pruneAccordionOpen,
  REFRESH_BUSY_DELAY_MS,
  refreshStatusPresentation,
  sessionIsActive
} from '../lib/dashboardState';
import { operatorPause } from '../lib/taskDisplay';
import type { AnyRecord, Project, Session, SessionGroup, Task, Workspace } from '../lib/types';
import { accordionOpen, refreshMs } from './layout';

/** The last `/api/state` snapshot and everything read straight off it. */

export const data = writable<AnyRecord | null>(null);
/** Set while there is no snapshot at all; once one exists failures go to `refreshError`. */
export const error = writable('');
export const refreshError = writable('');
/** A failed operator action (patch, drop, delete), shown on the work area. */
export const actionError = writable('');
export const lastRefreshAt = writable('');
/** Drives the busy affordance; polls only raise it after a short delay. */
export const refreshBusy = writable(false);

export const refreshStatus = derived(
  [refreshBusy, refreshError, lastRefreshAt],
  ([$busy, $error, $at]) =>
    refreshStatusPresentation({ refreshing: $busy, refreshError: $error, lastRefreshAt: $at })
);

export const workspaces = derived(data, ($data) => ($data?.workspaces ?? []) as Workspace[]);
export const projects = derived(data, ($data) => ($data?.projects ?? []) as Project[]);
export const tasks = derived(data, ($data) => ($data?.tasks ?? []) as Task[]);
export const people = derived(data, ($data) => ($data?.people ?? []) as string[]);
export const statuses = derived(data, ($data) => ($data?.statuses ?? []) as string[]);
export const orphanedTasks = derived(data, ($data) => ($data?.orphaned_tasks ?? []) as Session[]);
export const runtimeSessions = derived(data, ($data) =>
  (($data?.runtime_sessions ?? []) as Session[]).filter(sessionIsActive)
);
export const closedSessions = derived([data, tasks], ([$data, $tasks]) =>
  flattenClosedSessions(($data?.session_groups ?? []) as SessionGroup[], $tasks)
);
/** Sessions that still hold a task: running ones plus orphans awaiting an operator decision. */
export const liveSessions = derived([runtimeSessions, orphanedTasks], ([$runtime, $orphans]) => [
  ...$runtime,
  ...$orphans
]);

/** What waits on the operator: paused for input, blocked, or ready for review. */
export const attentionTasks = derived(tasks, ($tasks) => [
  ...$tasks.filter((task) => operatorPause(task).waiting),
  ...$tasks.filter((task) => task.status === 'blocked'),
  ...$tasks.filter((task) => task.status === 'needs_review')
]);

const loadListeners: Array<(state: AnyRecord) => void> = [];

/** Runs with each freshly polled snapshot, just before it replaces the current one. */
export const onStateLoaded = (listener: (state: AnyRecord) => void) => {
  loadListeners.push(listener);
};

let inFlight = false;
let busyTimer: number | null = null;

function clearBusyTimer() {
  if (busyTimer == null) return;
  window.clearTimeout(busyTimer);
  busyTimer = null;
}

export async function loadData({ manual = false } = {}) {
  if (inFlight) return;
  inFlight = true;
  clearBusyTimer();
  if (manual) {
    refreshBusy.set(true);
  } else {
    // Fast auto-polls stay visually quiet so the top menu does not flash every 5s.
    busyTimer = window.setTimeout(() => {
      refreshBusy.set(true);
      busyTimer = null;
    }, REFRESH_BUSY_DELAY_MS);
  }
  try {
    const response = await apiFetch('/api/state', { cache: 'no-store' });
    if (!response.ok) {
      throw new Error(`Failed to load Fleet state (${response.status})`);
    }
    const next = await response.json();
    next.tasks = dedupeTasksByIdentity(next.tasks ?? []);
    accordionOpen.update((open) => pruneAccordionOpen(open, next));
    loadListeners.forEach((listener) => listener(next));
    data.set(next);
    error.set('');
    refreshError.set('');
    actionError.set('');
    lastRefreshAt.set(new Date().toISOString());
  } catch (err: any) {
    if (get(data)) refreshError.set(err.message);
    else error.set(err.message);
  } finally {
    clearBusyTimer();
    inFlight = false;
    refreshBusy.set(false);
  }
}

export const requestManualRefresh = () => loadData({ manual: true });

/** Polls the server at the operator's chosen interval; returns the stop function. */
export function startPolling(): () => void {
  let timer: number | null = null;
  const stopInterval = refreshMs.subscribe((ms) => {
    if (timer != null) window.clearInterval(timer);
    timer = window.setInterval(() => loadData(), ms);
  });
  return () => {
    stopInterval();
    if (timer != null) window.clearInterval(timer);
    clearBusyTimer();
  };
}
