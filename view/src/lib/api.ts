export function apiFetch(input: RequestInfo | URL, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers || {});
  if (!headers.has('Authorization') && typeof document !== 'undefined') {
    const token = document.querySelector('meta[name="fleet-token"]')?.getAttribute('content') || '';
    if (token) headers.set('Authorization', `Bearer ${token}`);
  }
  return fetch(input, { ...init, headers });
}

// readApiError turns a failed response into a message for the person. The
// server answers with {"error": "...", "code": "..."}; anything else is shown
// as a short line, never as a page of raw output.
export async function readApiError(response: Response): Promise<string> {
  const text = await response.text();
  try {
    const body = JSON.parse(text);
    if (body && typeof body.error === 'string' && body.error) return body.error;
  } catch {
    // not JSON
  }
  const line = text.trim().split('\n')[0] || `Request failed (${response.status})`;
  return line.length > 200 ? `${line.slice(0, 200)}…` : line;
}
