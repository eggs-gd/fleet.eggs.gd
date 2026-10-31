import { apiFetch, readApiError } from './api.ts';

const trimTrailingSlashes = (path: string): string => {
  let end = path.length;
  while (end > 0 && path[end - 1] === '/') end -= 1;
  return path.slice(0, end);
};

/** The scan roots with `path` added; null when it is blank or already there. */
export const withScanRoot = (roots: string[], path: string): string[] | null => {
  const next = path.trim();
  if (!next) return null;
  const same = (a: string, b: string) => trimTrailingSlashes(a) === trimTrailingSlashes(b);
  if (roots.some((root) => same(root, next))) return null;
  return [...roots, next];
};

/** Adds a folder of git checkouts to what Fleet scans; resolves to the server's warnings. */
export async function addScanRoot(path: string): Promise<string[]> {
  const current = await apiFetch('/api/settings');
  if (!current.ok) throw new Error(await readApiError(current));
  const snapshot = await current.json();
  const roots: string[] = Array.isArray(snapshot?.projects?.scan_roots)
    ? snapshot.projects.scan_roots
    : [];
  const next = withScanRoot(roots, path);
  if (!next) {
    if (!path.trim())
      throw new Error('Enter the path of a folder that holds your git repositories.');
    return [];
  }
  const response = await apiFetch('/api/settings', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ scanRoots: next })
  });
  if (!response.ok) throw new Error(await readApiError(response));
  const result = await response.json();
  return result.warnings || [];
}
