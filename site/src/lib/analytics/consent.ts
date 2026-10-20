import { writable } from 'svelte/store';

/** Google Analytics measurement id. The tag loads only after the visitor accepts. */
const GA_ID = 'G-EZKCMTF6E0';
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

let loaded = false;

function loadGoogleTag() {
  if (loaded) return;
  loaded = true;
  const w = window as unknown as Record<string, unknown>;
  w[`ga-disable-${GA_ID}`] = false;
  const dataLayer = (w.dataLayer = (w.dataLayer as unknown[]) || []);
  // gtag.js only acts on `arguments` objects, not on arrays.
  function gtag(..._args: unknown[]) {
    dataLayer.push(arguments);
  }
  const script = document.createElement('script');
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtag/js?id=${GA_ID}`;
  document.head.appendChild(script);
  gtag('js', new Date());
  gtag('config', GA_ID);
}

/** Applies a stored choice on page load. Nothing is requested from Google unless it was granted. */
export function initConsent() {
  const stored = read();
  consent.set(stored);
  if (stored === 'granted') loadGoogleTag();
}

export function choose(choice: 'granted' | 'denied') {
  try {
    localStorage.setItem(STORAGE_KEY, choice);
  } catch {
    // The choice then only lasts for this page view.
  }
  consent.set(choice);
  if (choice === 'granted') {
    loadGoogleTag();
  } else if (loaded) {
    // A tag already running stops sending once the visitor changes their mind.
    (window as unknown as Record<string, unknown>)[`ga-disable-${GA_ID}`] = true;
  }
}

/** Shows the banner again so the visitor can change the choice. */
export const reopenConsent = () => consent.set('unset');
