import { writable } from 'svelte/store';

const FAVORITES_KEY = 'core.favoriteProjects';

const readFavoriteIds = () => {
  try {
    const parsed = JSON.parse(localStorage.getItem(FAVORITES_KEY) || '[]');
    return Array.isArray(parsed) ? parsed.filter((id) => typeof id === 'string' && id) : [];
  } catch {
    return [];
  }
};

const writeFavoriteIds = (ids) => {
  try {
    localStorage.setItem(FAVORITES_KEY, JSON.stringify(ids));
  } catch {
    // Best-effort per-browser list, not a source of truth.
  }
};

export const favoriteIds = writable(readFavoriteIds());

export const toggleFavorite = (id) => {
  if (!id) return;
  favoriteIds.update((ids) => {
    const next = ids.includes(id) ? ids.filter((item) => item !== id) : [...ids, id];
    writeFavoriteIds(next);
    return next;
  });
};
