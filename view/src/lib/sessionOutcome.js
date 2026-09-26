import { orphanKey, sessionKey } from './dashboardState.js';

const HITL_OUTCOMES = new Set(['blocked', 'needs_input', 'waiting_input', 'needs_rework']);
const HITL_STATUSES = new Set(['waiting_input', 'operator_attention', 'stalled', 'needs_input']);
const FAIL_OUTCOMES = new Set(['failed']);
const FAIL_STATUSES = new Set(['failed', 'provider_error', 'timed_out', 'dead', 'terminal']);
const SUCCESS_OUTCOMES = new Set(['completed']);
const SUCCESS_STATUSES = new Set(['succeeded', 'released', 'completed']);
const LIVE_STATUSES = new Set(['queued', 'claimed', 'starting', 'running', 'resumable']);

export const sessionStatus = (session) =>
  String(
    session?.execution_status || session?.status || session?.execution_state || ''
  ).toLowerCase();

export const sessionWorkerOutcome = (session) =>
  String(session?.result?.outcome || '').toLowerCase();

export const sessionOutcomeTone = (session, kind = 'closed') => {
  if (kind === 'orphan') return 'orphan';
  const status = sessionStatus(session);
  const outcome = sessionWorkerOutcome(session);
  if (status === 'orphaned') return 'orphan';
  if (HITL_OUTCOMES.has(outcome)) return 'hitl';
  if (FAIL_OUTCOMES.has(outcome) || FAIL_STATUSES.has(status)) return 'fail';
  if (HITL_STATUSES.has(status)) return 'hitl';
  if (kind === 'live' || LIVE_STATUSES.has(status)) return 'live';
  if (SUCCESS_OUTCOMES.has(outcome) || SUCCESS_STATUSES.has(status)) return 'success';
  return 'muted';
};

const titleCase = (value) =>
  String(value || '')
    .replace(/_/g, ' ')
    .replace(/^\w/, (ch) => ch.toUpperCase());

export const sessionOutcomeLabel = (session, kind = 'closed') => {
  const tone = sessionOutcomeTone(session, kind);
  const status = sessionStatus(session);
  const outcome = sessionWorkerOutcome(session);
  if (tone === 'orphan') return 'Orphaned';
  if (tone === 'hitl') {
    if (outcome === 'needs_rework') return 'Needs rework';
    if (outcome === 'blocked') return 'Blocked';
    return 'HITL';
  }
  if (tone === 'fail') {
    if (status === 'provider_error') return 'Provider error';
    if (status === 'timed_out') return 'Timed out';
    if (status === 'dead') return 'Dead';
    if (status === 'terminal') return 'Terminal';
    if (outcome === 'failed') return 'Failed';
    return titleCase(status) || 'Failed';
  }
  if (tone === 'success') {
    if (status === 'released') return 'Released';
    return 'Succeeded';
  }
  if (tone === 'live') return titleCase(status) || 'Running';
  return titleCase(status) || 'Closed';
};

export const elapsedClockPrecise = (startedAt, now = Date.now()) => {
  if (!startedAt) return '';
  const startMs = Date.parse(startedAt);
  if (!Number.isFinite(startMs)) return '';
  const totalSeconds = Math.max(0, Math.floor((now - startMs) / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
  }
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
};

export const resolveOpenSession = (selected, live = [], closed = [], orphans = []) => {
  if (!selected) return null;
  if (selected.kind === 'orphan') {
    const session = orphans.find((item) => orphanKey(item) === selected.key) || selected.snapshot;
    return session ? { session, kind: 'orphan' } : null;
  }
  const liveHit = live.find((item) => sessionKey(item) === selected.key);
  if (liveHit) return { session: liveHit, kind: 'live' };
  const closedHit = closed.find((item) => sessionKey(item) === selected.key);
  if (closedHit) return { session: closedHit, kind: 'closed' };
  return selected.snapshot ? { session: selected.snapshot, kind: selected.kind } : null;
};
