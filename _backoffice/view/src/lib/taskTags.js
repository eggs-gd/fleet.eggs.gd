/** Shared task tag tones. Every panel should use these helpers, not local color classes. */

export const PILL_TONES = ['red', 'blue', 'purple', 'amber', 'green', 'teal', 'pink', 'indigo', 'cyan', 'slate'];

export const TASK_TYPE_TONES = {
  bug: 'red',
  feature: 'blue',
  research: 'purple',
  investigation: 'purple',
  review: 'amber',
  ui: 'pink',
  docs: 'teal',
  content: 'teal',
  architecture: 'indigo',
  decision: 'indigo',
  refactor: 'cyan',
  test: 'green',
  maintenance: 'slate',
  chore: 'slate',
  infrastructure: 'slate'
};

const toneHash = (value) => [...String(value)].reduce((hash, ch) => (hash * 33 + ch.charCodeAt(0)) >>> 0, 0);

export const normalizeTaskType = (type) => String(type || '').trim().toLowerCase();

export const typeTone = (type) => {
  const key = normalizeTaskType(type);
  if (!key) return 'slate';
  if (TASK_TYPE_TONES[key]) return TASK_TYPE_TONES[key];
  return PILL_TONES[toneHash(key) % PILL_TONES.length];
};

export const typeLabel = (type) => normalizeTaskType(type) || 'task';

export const pillClass = (tone) => `pill pill-tone-${tone}`;

export const typePillClass = (type) => `${pillClass(typeTone(type))} pill-type`;
