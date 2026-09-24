import assert from 'node:assert/strict';
import test from 'node:test';

import { AGENTS_TABS, SETTINGS_SECTIONS, isSettingsNav, settingsSectionLabel } from './settingsNav.js';

test('settings sections match the product nav and no longer list agents or manager', () => {
  assert.deepEqual(
    SETTINGS_SECTIONS.map((section) => section.id),
    ['general', 'projects', 'workflow', 'integrations', 'diagnostics']
  );
});

test('the Agents page has its own Workers/Manager tabs', () => {
  assert.deepEqual(AGENTS_TABS, [
    { id: 'agents', label: 'Workers' },
    { id: 'manager', label: 'Manager' }
  ]);
});

test('agents rail is its own settings-shell destination, not a Settings sub-section', () => {
  assert.equal(isSettingsNav('agents'), true);
  assert.equal(isSettingsNav('settings'), true);
  assert.equal(isSettingsNav('work'), false);
  assert.equal(settingsSectionLabel('projects'), 'Projects & Data');
  assert.equal(settingsSectionLabel('agents'), 'Agents');
});
