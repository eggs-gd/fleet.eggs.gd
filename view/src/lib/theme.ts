export type ThemePref = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

export const THEME_PREFS: ThemePref[] = ['light', 'dark', 'system'];

export const normalizeThemePref = (value: unknown): ThemePref =>
  THEME_PREFS.includes(value as ThemePref) ? (value as ThemePref) : 'system';

export const resolveTheme = (pref: unknown, systemDark = false): ResolvedTheme => {
  const mode = normalizeThemePref(pref);
  if (mode === 'dark') return 'dark';
  if (mode === 'light') return 'light';
  return systemDark ? 'dark' : 'light';
};

export const systemPrefersDark = (): boolean => {
  if (typeof matchMedia !== 'function') return false;
  return matchMedia('(prefers-color-scheme: dark)').matches;
};

export const applyTheme = (pref: unknown): ResolvedTheme => {
  const mode = normalizeThemePref(pref);
  const resolved = resolveTheme(mode, systemPrefersDark());
  const root = document.documentElement;
  root.dataset.theme = resolved;
  root.style.colorScheme = resolved;
  const color = resolved === 'dark' ? '#111318' : '#f4f5f7';
  let meta = document.querySelector('meta[name="theme-color"]');
  if (!meta) {
    meta = document.createElement('meta');
    meta.setAttribute('name', 'theme-color');
    document.head.appendChild(meta);
  }
  meta.setAttribute('content', color);
  return resolved;
};

export const watchSystemTheme = (onChange: () => void): (() => void) => {
  if (typeof matchMedia !== 'function') return () => {};
  const mq = matchMedia('(prefers-color-scheme: dark)');
  const handler = () => onChange();
  mq.addEventListener('change', handler);
  return () => mq.removeEventListener('change', handler);
};
