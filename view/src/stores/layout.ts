import { writable } from 'svelte/store';
import {
  LEFT_WIDTH,
  RIGHT_WIDTH,
  loadAccordionOpen,
  loadAutoRefreshMs,
  loadLeftWidth,
  loadRightPanels,
  loadRightWidth,
  loadSidebarScroll,
  loadThemePref,
  saveAccordionOpen,
  saveAutoRefreshMs,
  saveLeftWidth,
  saveRightPanels,
  saveRightWidth,
  saveSidebarScroll,
  saveThemePref
} from '../lib/layoutPrefs';
import { applyTheme } from '../lib/theme';

/** Chrome the operator arranges (panel widths, open sections, theme); remembered in localStorage. */

export const leftWidth = writable(loadLeftWidth());
export const rightWidth = writable(loadRightWidth());
export const rightPanels = writable(loadRightPanels());
export const accordionOpen = writable(loadAccordionOpen());
export const themePref = writable(loadThemePref());
export const refreshMs = writable(loadAutoRefreshMs());

accordionOpen.subscribe(saveAccordionOpen);

export function dragLeft(dx: number) {
  leftWidth.update((width) => {
    const next = Math.min(LEFT_WIDTH.max, Math.max(LEFT_WIDTH.min, width + dx));
    saveLeftWidth(next);
    return next;
  });
}

export function dragRight(dx: number) {
  rightWidth.update((width) => {
    const next = Math.min(RIGHT_WIDTH.max, Math.max(RIGHT_WIDTH.min, width - dx));
    saveRightWidth(next);
    return next;
  });
}

function togglePanel(key: 'attentionOpen' | 'recentOpen') {
  rightPanels.update((panels) => {
    const next = { ...panels, [key]: !panels[key] };
    saveRightPanels(next);
    return next;
  });
}

export const toggleAttention = () => togglePanel('attentionOpen');
export const toggleRecent = () => togglePanel('recentOpen');

export function setThemePref(pref: string) {
  const saved = saveThemePref(pref);
  themePref.set(saved);
  applyTheme(saved);
}

export function setRefreshMs(ms: number) {
  refreshMs.set(saveAutoRefreshMs(ms));
}

/** Where the sidebar tree was scrolled to, so a reload or refresh keeps the place. */
let sidebarScroll = loadSidebarScroll();
export const sidebarScrollTop = () => sidebarScroll;
export function rememberSidebarScroll(value: number) {
  sidebarScroll = value;
  saveSidebarScroll(value);
}
