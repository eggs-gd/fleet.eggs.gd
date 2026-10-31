import assert from 'node:assert/strict';
import test from 'node:test';
import { withScanRoot } from './scanRoots.ts';

test('a new folder is appended', () => {
  assert.deepEqual(withScanRoot(['/a'], ' /b '), ['/a', '/b']);
  assert.deepEqual(withScanRoot([], '~/Projects'), ['~/Projects']);
});

test('blank input and folders already scanned add nothing', () => {
  assert.equal(withScanRoot(['/a'], '   '), null);
  assert.equal(withScanRoot(['/a'], '/a'), null);
  assert.equal(withScanRoot(['/a/'], '/a'), null);
});

test('a root that differs only by trailing slashes is already there', () => {
  assert.equal(withScanRoot(['/a/b/'], '/a/b'), null);
  assert.equal(withScanRoot(['/a/b'], '/a/b///'), null);
  assert.equal(withScanRoot(['/a/b'], '   '), null);
  assert.deepEqual(withScanRoot(['/a/b'], '/a/bc'), ['/a/b', '/a/bc']);
});
