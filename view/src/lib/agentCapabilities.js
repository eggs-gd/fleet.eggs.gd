// Plain-language rows for what a provider can do, shown in Settings → Agents.
// The server owns the facts (`capabilities`, `session_reuse` on each agent).
// This file only names them for a person.

const FLAG_ROWS = [
  ['can_show_app_visible_link', 'Link to the vendor app'],
  ['can_accept_operator_input', 'You can continue the conversation'],
  ['can_resume_session', 'Fleet can resume a session'],
  ['can_detect_running_session', 'Fleet finds a running session again after a restart'],
  ['can_query_thread_status', 'Fleet can ask the provider whether it is still running'],
  ['can_confirm_terminal_outcome', 'Fleet can tell when it finished']
];

const BOOL_ROWS = [['can_list_sessions', 'Provider can list its sessions']];

const REUSE_LABELS = {
  verified: 'Confirmed to continue the same session',
  unverified: 'Attempted, not confirmed end to end',
  unsupported: 'Not supported'
};

const flagText = (value) => {
  if (value === 'yes' || value === true) return 'yes';
  if (value === 'no' || value === false) return 'no';
  return 'unknown';
};

const tone = (text) => (text === 'yes' ? 'ok' : text === 'no' ? 'muted' : 'warn');

export function capabilityRows(agent) {
  const caps = agent?.capabilities || {};
  return [...FLAG_ROWS, ...BOOL_ROWS].map(([key, label]) => {
    const value = flagText(caps[key]);
    return { key, label, value, tone: tone(value) };
  });
}

export function reuseSummary(agent) {
  const reuse = agent?.session_reuse || {};
  const level = reuse.level || 'unsupported';
  return {
    level,
    label: REUSE_LABELS[level] || level,
    attempts: reuse.can_attempt === true,
    reason: plainText(reuse.reason)
  };
}

export function launchesAutomatically(agent) {
  return agent?.live_ready === true && agent?.enabled?.value !== false;
}

// Server notes quote commands in backticks. The settings page shows plain text.
export const plainText = (text) => String(text || '').replaceAll('`', '');
