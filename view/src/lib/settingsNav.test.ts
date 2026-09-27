import assert from 'node:assert/strict';
import test from 'node:test';

import { SETTINGS_SECTIONS, isSettingsNav, settingsSectionLabel } from './settingsNav.ts';

test('settings sections include one Agents tab', () => {
  assert.deepEqual(
    SETTINGS_SECTIONS.map((section) => section.id),
    ['general', 'agents', 'workflow', 'integrations', 'diagnostics']
  );
  assert.equal(settingsSectionLabel('agents'), 'Agents');
  assert.equal(settingsSectionLabel('general'), 'General');
});

test('agents is a settings section, not its own rail destination', () => {
  assert.equal(isSettingsNav('settings'), true);
  assert.equal(isSettingsNav('agents'), false);
  assert.equal(isSettingsNav('work'), false);
});
