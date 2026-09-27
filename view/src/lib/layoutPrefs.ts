import { normalizeThemePref, type ThemePref } from './theme.ts';

const LEFT_KEY = 'core.split.leftWidth';
const RIGHT_KEY = 'core.split.rightWidth';
const ACCORDION_KEY = 'core.accordionOpen';
const SCROLL_KEY = 'core.sidebarScroll';
const RIGHT_PANELS_KEY = 'core.rightPanels';
const THEME_KEY = 'core.theme';

export interface WidthBounds {
  min: number;
  max: number;
  fallback: number;
}

export const LEFT_WIDTH: WidthBounds = { min: 260, max: 420, fallback: 260 };
export const RIGHT_WIDTH: WidthBounds = { min: 240, max: 480, fallback: 300 };

const readJson = <T>(key: string, fallback: T): T => {
  try {
    const raw = localStorage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch {
    return fallback;
  }
};

const writeJson = (key: string, value: unknown): void => {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Best-effort UI chrome; not a source of truth.
  }
};

const clamp = (value: unknown, { min, max, fallback }: WidthBounds): number => {
  const n = Number(value);
  if (!Number.isFinite(n)) return fallback;
  return Math.min(max, Math.max(min, Math.round(n)));
};

export const loadLeftWidth = (): number =>
  clamp(readJson(LEFT_KEY, LEFT_WIDTH.fallback), LEFT_WIDTH);

export const loadRightWidth = (): number =>
  clamp(readJson(RIGHT_KEY, RIGHT_WIDTH.fallback), RIGHT_WIDTH);

export const saveLeftWidth = (value: unknown): void =>
  writeJson(LEFT_KEY, clamp(value, LEFT_WIDTH));

export const saveRightWidth = (value: unknown): void =>
  writeJson(RIGHT_KEY, clamp(value, RIGHT_WIDTH));

export const loadAccordionOpen = (): Record<string, boolean> => {
  const value = readJson<Record<string, boolean> | null>(ACCORDION_KEY, {});
  return value && typeof value === 'object' ? value : {};
};

export const saveAccordionOpen = (value: Record<string, boolean> | null | undefined): void =>
  writeJson(ACCORDION_KEY, value || {});

export const loadSidebarScroll = (): number => {
  const n = Number(readJson(SCROLL_KEY, 0));
  return Number.isFinite(n) && n > 0 ? n : 0;
};

export const saveSidebarScroll = (value: unknown): void =>
  writeJson(SCROLL_KEY, Math.max(0, Number(value) || 0));

export interface RightPanelsPrefs {
  attentionOpen: boolean;
  recentOpen: boolean;
  attentionShare: number;
}

export const loadRightPanels = (): RightPanelsPrefs => {
  const value = readJson<Partial<RightPanelsPrefs> | null>(RIGHT_PANELS_KEY, null);
  const share = Number(value?.attentionShare);
  return {
    attentionOpen: value?.attentionOpen !== false,
    recentOpen: value?.recentOpen !== false,
    attentionShare: Number.isFinite(share) ? Math.min(0.72, Math.max(0.28, share)) : 0.58
  };
};

export const saveRightPanels = (value: RightPanelsPrefs): void =>
  writeJson(RIGHT_PANELS_KEY, value);

export const loadThemePref = (): ThemePref => normalizeThemePref(readJson(THEME_KEY, 'system'));

export const saveThemePref = (value: unknown): ThemePref => {
  const pref = normalizeThemePref(value);
  writeJson(THEME_KEY, pref);
  return pref;
};

const REFRESH_KEY = 'core.autoRefreshMs';
export const AUTO_REFRESH_MS = [2000, 5000, 10000, 30000];

export const loadAutoRefreshMs = (): number => {
  const n = Number(readJson(REFRESH_KEY, 5000));
  return AUTO_REFRESH_MS.includes(n) ? n : 5000;
};

export const saveAutoRefreshMs = (value: unknown): number => {
  const n = AUTO_REFRESH_MS.includes(Number(value)) ? Number(value) : 5000;
  writeJson(REFRESH_KEY, n);
  return n;
};
