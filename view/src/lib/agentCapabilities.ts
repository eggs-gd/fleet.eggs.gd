// Plain-language rows for what a provider can do, shown in Settings → Agents.
// The server owns the facts (`capabilities`, `session_reuse` on each agent).
// This file only names them for a person.

import type { Agent } from './types.ts';

const FLAG_ROWS: [string, string][] = [
  ['can_show_app_visible_link', 'Link to the vendor app'],
  ['can_accept_operator_input', 'You can continue the conversation'],
  ['can_resume_session', 'Fleet can resume a session'],
  ['can_detect_running_session', 'Fleet finds a running session again after a restart'],
  ['can_query_thread_status', 'Fleet can ask the provider whether it is still running'],
  ['can_confirm_terminal_outcome', 'Fleet can tell when it finished']
];

const BOOL_ROWS: [string, string][] = [['can_list_sessions', 'Provider can list its sessions']];

const REUSE_LABELS: Record<string, string> = {
  verified: 'Confirmed to continue the same session',
  unverified: 'Attempted, not confirmed end to end',
  unsupported: 'Not supported'
};

const flagText = (value: unknown): 'yes' | 'no' | 'unknown' => {
  if (value === 'yes' || value === true) return 'yes';
  if (value === 'no' || value === false) return 'no';
  return 'unknown';
};

const tone = (text: string): string => (text === 'yes' ? 'ok' : text === 'no' ? 'muted' : 'warn');

export interface CapabilityRow {
  key: string;
  label: string;
  value: string;
  tone: string;
}

export function capabilityRows(agent?: Agent | null): CapabilityRow[] {
  const caps = agent?.capabilities || {};
  return [...FLAG_ROWS, ...BOOL_ROWS].map(([key, label]) => {
    const value = flagText((caps as Record<string, unknown>)[key]);
    return { key, label, value, tone: tone(value) };
  });
}

export interface ReuseSummary {
  level: string;
  label: string;
  attempts: boolean;
  reason: string;
}

export function reuseSummary(agent?: Agent | null): ReuseSummary {
  const reuse = agent?.session_reuse || {};
  const level = reuse.level || 'unsupported';
  return {
    level,
    label: REUSE_LABELS[level] || level,
    attempts: reuse.can_attempt === true,
    reason: plainText(reuse.reason)
  };
}

export function launchesAutomatically(agent?: Agent | null): boolean {
  return agent?.live_ready === true && agent?.enabled?.value !== false;
}

// Server notes quote commands in backticks. The settings page shows plain text.
export const plainText = (text: unknown): string => String(text || '').replaceAll('`', '');
