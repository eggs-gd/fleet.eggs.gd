import { writable } from 'svelte/store';
import { apiFetch } from '../lib/api';
import type { AnyRecord } from '../lib/types';

/** First-run prompt to pick a Manager provider; skipping hides it until the next reload. */

const SETUP_SKIPPED_KEY = 'fleet.managerSetupSkipped';

export const managerSetupNeeded = writable(false);
export const managerProviders = writable<AnyRecord[]>([]);

function setupSkipped() {
  try {
    return localStorage.getItem(SETUP_SKIPPED_KEY) === '1';
  } catch {
    return false;
  }
}

export function skipManagerSetup() {
  try {
    localStorage.setItem(SETUP_SKIPPED_KEY, '1');
  } catch {
    // The dialog only stays closed until the next reload.
  }
  managerSetupNeeded.set(false);
}

export async function loadManagerSetup() {
  try {
    const response = await apiFetch('/api/manager/setup', { cache: 'no-store' });
    if (!response.ok) return;
    const body = await response.json();
    managerProviders.set(body?.providers ?? []);
    managerSetupNeeded.set(Boolean(body?.needed) && !setupSkipped());
  } catch {
    managerSetupNeeded.set(false);
  }
}
