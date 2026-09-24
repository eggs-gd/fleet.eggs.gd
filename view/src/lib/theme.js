export const THEME_PREFS = ['light', 'dark', 'system'];

export const normalizeThemePref = (value) => (THEME_PREFS.includes(value) ? value : 'system');

export const resolveTheme = (pref, systemDark = false) => {
  const mode = normalizeThemePref(pref);
  if (mode === 'dark') return 'dark';
  if (mode === 'light') return 'light';
  return systemDark ? 'dark' : 'light';
};

export const systemPrefersDark = () => {
  if (typeof matchMedia !== 'function') return false;
  return matchMedia('(prefers-color-scheme: dark)').matches;
};

export const applyTheme = (pref) => {
  const mode = normalizeThemePref(pref);
  const resolved = resolveTheme(mode, systemPrefersDark());
  const root = document.documentElement;
  root.dataset.theme = resolved;
  root.style.colorScheme = resolved;
  return resolved;
};

export const watchSystemTheme = (onChange) => {
  if (typeof matchMedia !== 'function') return () => {};
  const mq = matchMedia('(prefers-color-scheme: dark)');
  const handler = () => onChange();
  mq.addEventListener('change', handler);
  return () => mq.removeEventListener('change', handler);
};
