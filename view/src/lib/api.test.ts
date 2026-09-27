import assert from 'node:assert/strict';
import test from 'node:test';

import { readApiError } from './api.ts';

const failed = (body: string, status = 400) => new Response(body, { status });

test('a JSON error from the server is shown as its message', async () => {
  const message = await readApiError(
    failed(JSON.stringify({ error: 'Gemini is not signed in.', code: 'auth_required' }))
  );
  assert.equal(message, 'Gemini is not signed in.');
});

test('anything else is cut to one short line', async () => {
  assert.equal(await readApiError(failed('first line\nsecond line')), 'first line');
  const long = await readApiError(failed('x'.repeat(500)));
  assert.equal(long.length, 201);
  assert.equal(await readApiError(failed('', 502)), 'Request failed (502)');
});
