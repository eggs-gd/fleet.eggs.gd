import assert from 'node:assert/strict';
import test from 'node:test';

import { asText } from './text.ts';

test('asText keeps strings and numbers and empties everything else', () => {
  assert.equal(asText('abc'), 'abc');
  assert.equal(asText(42), '42');
  assert.equal(asText(0), '0');
  assert.equal(asText(10n), '10');
  assert.equal(asText(null), '');
  assert.equal(asText(undefined), '');
  assert.equal(asText(true), '');
  assert.equal(asText({ a: 1 }), '');
  assert.equal(asText(['x']), '');
});
