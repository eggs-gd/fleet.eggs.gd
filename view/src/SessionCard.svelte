<script lang="ts">
  import {
    agentColorClass,
    elapsedClock,
    orphanRecoveryLabel,
    toolUsageHasMissing
  } from './lib/taskDisplay';
  import { sessionOutcomeLabel, sessionOutcomeTone } from './lib/sessionOutcome';
  import Icon from './Icon.svelte';
  import type { Session } from './lib/types';

  export let session: Session;
  export let kind = 'live';
  export let now = Date.now();
  export let onOpen: (session: Session, kind: string) => void = () => {};
  export let onResolveOrphan: ((session: Session, status: string) => void) | null = null;
  export let onReleaseSession:
    ((session: Session, action: string, targetStatus: string) => void) | null = null;

  const ORPHAN_RESOLVE_ACTIONS = [
    { status: 'blocked', label: 'Blocked', icon: 'x-circle' },
    { status: 'needs_rework', label: 'Rework', icon: 'refresh' },
    { status: 'todo', label: 'Todo', icon: 'play' },
    { status: 'done', label: 'Done', icon: 'checks-circle' }
  ];

  $: live = kind === 'live';
  $: orphan = kind === 'orphan';
  $: tone = sessionOutcomeTone(session, kind);
  $: status = session.execution_status || session.status || (live ? 'starting' : 'closed');
  $: statusLabel = orphan
    ? orphanRecoveryLabel(session)
    : live && tone === 'live'
      ? `Running ${elapsedClock(session.started_at, now) || status}`
      : live && tone === 'hitl'
        ? `HITL ${elapsedClock(session.started_at, now) || ''}`.trim()
        : sessionOutcomeLabel(session, kind);
  $: showOrphanActions = orphan && (onResolveOrphan || (onReleaseSession && session.claim_id));

  function open() {
    onOpen(session, kind);
  }

  function onKey(event: KeyboardEvent) {
    // Ignore keydowns that bubbled up from a nested action button (e.g.
    // pressing Space to click "Blocked") — only the card itself opening the
    // modal on Enter/Space should reach here.
    if (event.target !== event.currentTarget) return;
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      open();
    }
  }

  function resolve(event: Event, targetStatus: string) {
    event.preventDefault();
    event.stopPropagation();
    onResolveOrphan?.(session, targetStatus);
  }

  function releaseClaim(event: Event) {
    event.preventDefault();
    event.stopPropagation();
    onReleaseSession?.(
      {
        claim_id: session.claim_id,
        execution_status: session.execution_state || 'orphaned',
        status: session.execution_state || 'orphaned'
      },
      'release',
      'needs_review'
    );
  }
</script>

<section
  class={`session-card session-card--${tone}`}
  class:session-card--orphaned={orphan}
  class:tool-evidence-warning={!orphan && toolUsageHasMissing(session)}
  role="button"
  tabindex="0"
  aria-haspopup="dialog"
  on:click={open}
  on:keydown={onKey}
>
  <div class="session-card-head">
    <span
      class={`agent-dot ${orphan ? 'agent-orphaned' : agentColorClass(session.agent || session.assignee)}`}
    ></span>
    <strong class="session-card-agent">{session.agent || session.assignee || 'session'}</strong>
    {#if orphan}
      <span class="session-card-status session-card-status--orphan">
        <Icon name="x-circle" size={13} />
        {statusLabel}
      </span>
    {:else}
      <span
        class={`session-card-status session-card-status--${tone}`}
        class:is-live={live && tone === 'live'}
      >
        <span class="status-dot"></span>
        {statusLabel}
      </span>
    {/if}
  </div>
  <p class="session-card-project">{session.repository || session.project_id}</p>
  <p class="session-card-task">{session.task_title || session.task_ref || session.task_id}</p>
  <p class="session-card-activity">
    {#if orphan}
      {session.blocking_reason || session.reason || 'Fleet lost track of this session.'}
    {:else}
      {session.last_message || session.last_event || session.result?.summary || status}
    {/if}
  </p>
  {#if showOrphanActions}
    <div class="session-card-actions" role="group" aria-label="Resolve orphaned session">
      {#if onResolveOrphan}
        {#each ORPHAN_RESOLVE_ACTIONS as action (action.status)}
          <button
            type="button"
            title={`Move task to ${action.label}`}
            aria-label={action.label}
            on:click={(event) => resolve(event, action.status)}
          >
            <Icon name={action.icon} size={13} />
          </button>
        {/each}
      {/if}
      {#if onReleaseSession && session.claim_id}
        <button
          type="button"
          title="Release claim"
          aria-label="Release claim"
          on:click={releaseClaim}
        >
          <Icon name="undo" size={13} />
        </button>
      {/if}
    </div>
  {/if}
</section>

<style>
  .session-card {
    min-width: 0;
    min-height: 0;
    border: none;
    border-right: 1px solid var(--border);
    border-radius: 0;
    background: var(--surface);
    box-shadow: none;
    padding: var(--space-lg);
    cursor: pointer;
    text-align: left;
  }

  .session-card:last-child {
    border-right: none;
  }

  .session-card:hover,
  .session-card:focus-visible {
    background: var(--surface-muted);
  }

  .session-card:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }

  .session-card--success {
    border-left: 3px solid var(--green);
    background: var(--session-success-bg);
  }

  .session-card--success:hover,
  .session-card--success:focus-visible {
    background: var(--session-success-bg-hover);
  }

  .session-card--fail {
    border-left: 3px solid var(--red);
    background: var(--session-fail-bg);
  }

  .session-card--fail:hover,
  .session-card--fail:focus-visible {
    background: var(--session-fail-bg-hover);
  }

  .session-card--hitl {
    border-left: 3px solid var(--amber);
    background: var(--session-hitl-bg);
  }

  .session-card--hitl:hover,
  .session-card--hitl:focus-visible {
    background: var(--session-hitl-bg-hover);
  }

  .session-card--live {
    border-left: 3px solid var(--green);
  }

  .session-card--orphaned,
  .session-card--orphan {
    border-left: 3px solid var(--amber);
    background: var(--amber-soft);
  }

  /* At-a-glance flag for a live/closed session missing required tool evidence
     — independent of the tone-based left border above, so it stays visible
     whatever the session's outcome color is. */
  .session-card.tool-evidence-warning {
    box-shadow: inset 0 0 0 1px var(--amber);
  }

  .session-card-head {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .session-card-agent {
    font-size: var(--text-md);
    text-transform: capitalize;
  }

  .session-card-status {
    margin-left: auto;
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    max-width: 58%;
    overflow: hidden;
    color: var(--text-muted);
    font-size: var(--text-xs);
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .session-card-project {
    margin: var(--space-md) 0 0;
    overflow: hidden;
    color: var(--text-faint);
    font-size: var(--text-xs);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .session-card-task {
    margin: var(--space-2xs) 0;
    overflow: hidden;
    font-size: var(--text-md);
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .session-card-status .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--text-faint);
  }

  .session-card-status.is-live,
  .session-card-status--live,
  .session-card-status--success {
    color: var(--green-text);
  }

  .session-card-status.is-live .status-dot,
  .session-card-status--live .status-dot,
  .session-card-status--success .status-dot {
    background: var(--green);
  }

  .session-card-status--fail {
    color: var(--red-text);
  }

  .session-card-status--fail .status-dot {
    background: var(--red);
  }

  .session-card-status--hitl {
    color: var(--amber-text);
  }

  .session-card-status--hitl .status-dot {
    background: var(--amber);
  }

  .session-card-status--orphan,
  .session-card-status--orphaned {
    color: var(--red-text);
  }

  .session-card-activity {
    margin: 0;
    overflow: hidden;
    color: var(--text-muted);
    font-size: var(--text-sm);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .session-card-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs);
    margin-top: var(--space-md);
  }

  .session-card-actions button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    min-height: 26px;
    padding: 0;
    border-color: var(--border-strong);
    background: var(--surface);
    color: var(--text-muted);
  }

  .session-card-actions button:hover {
    background: var(--surface-muted);
    color: var(--text);
  }
</style>
