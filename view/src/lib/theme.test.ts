import assert from 'node:assert/strict';
import test from 'node:test';

import { normalizeThemePref, resolveTheme, THEME_PREFS } from './theme.ts';

test('theme prefs are light, dark, or system', () => {
  assert.deepEqual(THEME_PREFS, ['light', 'dark', 'system']);
  assert.equal(normalizeThemePref('dark'), 'dark');
  assert.equal(normalizeThemePref('light'), 'light');
  assert.equal(normalizeThemePref('system'), 'system');
  assert.equal(normalizeThemePref('nope'), 'system');
  assert.equal(normalizeThemePref(null), 'system');
});

test('resolveTheme honors explicit prefs over the OS', () => {
  assert.equal(resolveTheme('light', true), 'light');
  assert.equal(resolveTheme('dark', false), 'dark');
});

test('system theme follows prefers-color-scheme', () => {
  assert.equal(resolveTheme('system', true), 'dark');
  assert.equal(resolveTheme('system', false), 'light');
});
