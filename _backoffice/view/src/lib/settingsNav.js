export const SETTINGS_SECTIONS = [
  { id: 'general', label: 'General' },
  { id: 'projects', label: 'Projects & Data' },
  { id: 'workflow', label: 'Workflow' },
  { id: 'integrations', label: 'Integrations' },
  { id: 'diagnostics', label: 'Diagnostics' }
];

// The Agents rail destination (Work / Agents / Settings) is its own page,
// not one of Settings' own sub-sections. It has two tabs of its own —
// Workers (executor providers, section id stays 'agents' to match the
// existing snapshot/draft field) and Manager (moved out of Settings) — both
// rendered through the same SettingsView data machinery but outside the
// "Settings" sidebar/nav shell.
export const AGENTS_TABS = [
  { id: 'agents', label: 'Workers' },
  { id: 'manager', label: 'Manager' }
];

export const settingsSectionLabel = (id) => (id === 'agents' ? 'Agents' : SETTINGS_SECTIONS.find((section) => section.id === id)?.label || id);

export const isSettingsNav = (globalNav) => globalNav === 'settings' || globalNav === 'agents';
