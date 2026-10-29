import { writable } from 'svelte/store';
import { apiFetch } from '../lib/api';
import type { AnyRecord } from '../lib/types';

/**
 * The prompt to pick a Manager provider. It opens by itself on a first run, and
 * from Settings → Agents whenever there is no Manager session. Skipping the
 * first-run prompt hides it until the next reload.
 */

const SETUP_SKIPPED_KEY = 'fleet.managerSetupSkipped';

/** Whether the dialog is showing. */
export const managerSetupNeeded = writable(false);
/** Opened from Settings, so it is a plain cancel and not a "set up later". */
export const managerSetupFromSettings = writable(false);
export const managerProviders = writable<AnyRecord[]>([]);
/** Bumps when a session was created, so Settings can reload what it shows. */
export const managerCreated = writable(0);

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

export function closeManagerSetup() {
  managerSetupNeeded.set(false);
  managerSetupFromSettings.set(false);
}

export function finishManagerSetup() {
  closeManagerSetup();
  managerCreated.update((n) => n + 1);
}

async function fetchSetup(): Promise<AnyRecord | null> {
  try {
    const response = await apiFetch('/api/manager/setup', { cache: 'no-store' });
    return response.ok ? await response.json() : null;
  } catch {
    return null;
  }
}

export async function loadManagerSetup() {
  const body = await fetchSetup();
  if (!body) {
    managerSetupNeeded.set(false);
    return;
  }
  managerProviders.set(body.providers ?? []);
  managerSetupNeeded.set(Boolean(body.needed) && !setupSkipped());
}

/** Opens the dialog on request, whatever was skipped before. */
export async function openManagerSetup() {
  const body = await fetchSetup();
  if (body) managerProviders.set(body.providers ?? []);
  managerSetupFromSettings.set(true);
  managerSetupNeeded.set(true);
}
