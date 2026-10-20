import { writable } from 'svelte/store';

/**
 * The Google tag itself lives in app.html so Google can detect it. It starts with
 * analytics storage denied (Consent Mode); this module records the visitor's
 * choice and tells the tag.
 */
const STORAGE_KEY = 'fleet.analyticsConsent';

export type Consent = 'granted' | 'denied' | 'unset';

const read = (): Consent => {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    return value === 'granted' || value === 'denied' ? value : 'unset';
  } catch {
    return 'unset';
  }
};

/** What the visitor chose; `unset` means the banner is showing. */
export const consent = writable<Consent>('unset');

const tell = (choice: 'granted' | 'denied') => {
  const gtag = (window as unknown as { gtag?: (...args: unknown[]) => void }).gtag;
  gtag?.('consent', 'update', { analytics_storage: choice });
};

/** Shows the banner on a first visit; a stored choice was already applied by app.html. */
export const initConsent = () => consent.set(read());

export function choose(choice: 'granted' | 'denied') {
  try {
    localStorage.setItem(STORAGE_KEY, choice);
  } catch {
    // The choice then only lasts for this page view.
  }
  consent.set(choice);
  tell(choice);
}

/** Shows the banner again so the visitor can change the choice. */
export const reopenConsent = () => consent.set('unset');
