export const SETTINGS_SECTIONS = [
  { id: 'general', label: 'General' },
  { id: 'agents', label: 'Agents' },
  { id: 'workflow', label: 'Workflow' },
  { id: 'integrations', label: 'Integrations' },
  { id: 'diagnostics', label: 'Diagnostics' }
];

export const settingsSectionLabel = (id) =>
  SETTINGS_SECTIONS.find((section) => section.id === id)?.label || id;

export const isSettingsNav = (globalNav) => globalNav === 'settings';
