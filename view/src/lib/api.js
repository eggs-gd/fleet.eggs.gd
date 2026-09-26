export function apiFetch(input, init = {}) {
  const headers = new Headers(init.headers || {});
  if (!headers.has('Authorization') && typeof document !== 'undefined') {
    const token = document.querySelector('meta[name="fleet-token"]')?.getAttribute('content') || '';
    if (token) headers.set('Authorization', `Bearer ${token}`);
  }
  return fetch(input, { ...init, headers });
}
