export interface SettingsSection {
  id: string;
  label: string;
}

export const SETTINGS_SECTIONS: SettingsSection[] = [
  { id: 'general', label: 'General' },
  { id: 'agents', label: 'Agents' },
  { id: 'workflow', label: 'Workflow' },
  { id: 'integrations', label: 'Integrations' },
  { id: 'diagnostics', label: 'Diagnostics' }
];

export const settingsSectionLabel = (id: string): string =>
  SETTINGS_SECTIONS.find((section) => section.id === id)?.label || id;

export const isSettingsNav = (globalNav: string): boolean => globalNav === 'settings';
