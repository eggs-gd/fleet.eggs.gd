<script lang="ts">
  import { onDestroy } from 'svelte';
  import { orphanKey, sessionKey } from './lib/dashboardState';
  import { resolveOpenSession, type SelectedSessionRef } from './lib/sessionOutcome';
  import Icon from './Icon.svelte';
  import SessionCard from './SessionCard.svelte';
  import SessionModal from './SessionModal.svelte';
  import type { Session } from './lib/types';

  interface RefreshStatus {
    label: string;
    warning: boolean;
    busy: boolean;
    busyLabel: string;
    updatedAt: string;
    updatedText: string;
  }

  export let runtimeSessions: Session[] = [];
  export let closedSessions: Session[] = [];
  export let orphanedTasks: Session[] = [];
  export let onControlSession: (
    session: Session,
    action: string,
    input?: string
  ) => Promise<void> = async () => {};
  export let onReleaseSession: (
    session: Session,
    action: string,
    targetStatus: string
  ) => Promise<void> = async () => {};
  export let onResolveOrphan: (session: Session, status: string) => Promise<void> = async () => {};
  export let refreshStatus: RefreshStatus | null = null;

  let tab = 'active';
  let expanded = false;
  let selected: SelectedSessionRef | null = null;
  let now = Date.now();
  let tick = setInterval(() => (now = Date.now()), 30000);
  onDestroy(() => clearInterval(tick));

  $: activeCount = runtimeSessions.length + orphanedTasks.length;
  $: count = tab === 'active' ? activeCount : closedSessions.length;
  $: emptyText = tab === 'active' ? 'No active sessions.' : 'No closed sessions.';
  $: openSession = resolveOpenSession(selected, runtimeSessions, closedSessions, orphanedTasks);

  function setTick(ms: number) {
    clearInterval(tick);
    tick = setInterval(() => (now = Date.now()), ms);
  }

  function openSelected(session: Session, kind: string) {
    selected = {
      key: kind === 'orphan' ? orphanKey(session) : sessionKey(session),
      kind,
      snapshot: session
    };
    now = Date.now();
    setTick(1000);
  }

  function closeSelected() {
    selected = null;
    setTick(30000);
  }
</script>

<section class="runtime-strip" class:is-expanded={expanded} aria-label="Sessions">
  <header class="runtime-strip-header">
    <strong>Sessions</strong>
    <div class="session-tabs" role="tablist" aria-label="Session status">
      <button
        type="button"
        role="tab"
        aria-selected={tab === 'active'}
        class:active={tab === 'active'}
        on:click={() => (tab = 'active')}
      >
        Active
      </button>
      <button
        type="button"
        role="tab"
        aria-selected={tab === 'closed'}
        class:active={tab === 'closed'}
        on:click={() => (tab = 'closed')}
      >
        Closed
      </button>
    </div>
    <span>{count}</span>
    {#if tab === 'active' && orphanedTasks.length}
      <span class="runtime-strip-orphan-flag" title="Fleet lost track of these sessions">
        {orphanedTasks.length} orphaned
      </span>
    {/if}
    {#if refreshStatus}
      <span
        class="runtime-strip-refresh"
        class:refresh-warning={refreshStatus.warning}
        aria-live={refreshStatus.warning ? 'polite' : 'off'}
      >
        <span class="refresh-indicator" class:is-busy={refreshStatus.busy} aria-hidden="true">
          <span class="refresh-spinner"></span>
        </span>
        {refreshStatus.label}
        {#if refreshStatus.updatedText}&nbsp;·&nbsp;{refreshStatus.updatedText}{/if}
      </span>
    {/if}
    <button
      type="button"
      class="view-all-sessions"
      class:active={expanded}
      aria-expanded={expanded}
      title={expanded ? 'Collapse sessions' : 'Expand sessions'}
      on:click={() => (expanded = !expanded)}
    >
      View all
      <Icon name={expanded ? 'collapse' : 'expand'} size={13} />
    </button>
  </header>

  <div class="session-grid">
    {#if tab === 'active'}
      {#each runtimeSessions as session (sessionKey(session))}
        <SessionCard {session} kind="live" {now} onOpen={openSelected} />
      {/each}
      {#each orphanedTasks as orphan (orphanKey(orphan))}
        <SessionCard
          session={orphan}
          kind="orphan"
          {now}
          onOpen={openSelected}
          {onResolveOrphan}
          {onReleaseSession}
        />
      {/each}
    {:else}
      {#each closedSessions as session (sessionKey(session))}
        <SessionCard {session} kind="closed" {now} onOpen={openSelected} />
      {/each}
    {/if}

    {#if tab === 'active' ? !activeCount : !closedSessions.length}
      <p class="empty">{emptyText}</p>
    {/if}
  </div>
</section>

{#if openSession}
  <SessionModal
    session={openSession.session}
    kind={openSession.kind}
    {now}
    onClose={closeSelected}
    {onControlSession}
    {onReleaseSession}
    {onResolveOrphan}
  />
{/if}

<style>
  .runtime-strip {
    flex: 0 0 auto;
  }

  .runtime-strip-header {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    margin-bottom: var(--space-md);
  }

  .runtime-strip-header strong {
    font-size: var(--text-md);
  }

  .runtime-strip-header span {
    min-width: 22px;
    border-radius: var(--radius-full);
    background: var(--surface-muted);
    color: var(--text-muted);
    font-size: var(--text-xs);
    font-weight: 700;
    padding: var(--space-2xs) var(--space-sm);
    text-align: center;
  }

  .runtime-strip-refresh {
    display: inline-flex;
    align-items: center;
    gap: var(--space-sm);
    margin-left: auto;
    color: var(--text-faint);
    font-size: var(--text-xs);
    white-space: nowrap;
  }

  .runtime-strip-refresh.refresh-warning {
    color: var(--amber-text);
    font-weight: 600;
  }

  .refresh-indicator {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 12px;
    height: 12px;
    opacity: 0;
    visibility: hidden;
  }

  .refresh-indicator.is-busy {
    opacity: 1;
    visibility: visible;
  }

  .refresh-spinner {
    display: block;
    width: 10px;
    height: 10px;
    border: 2px solid var(--border-strong);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: refresh-spin 0.7s linear infinite;
  }

  .session-tabs {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2xs);
  }

  .session-tabs button {
    min-height: 26px;
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-muted);
    font-size: var(--text-sm);
    font-weight: 600;
    padding: 0 var(--space-lg);
  }

  .session-tabs button:hover {
    background: var(--surface-muted);
    color: var(--text);
  }

  .session-tabs button.active {
    background: var(--surface-muted);
    color: var(--text);
  }

  .view-all-sessions {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    min-height: 0;
    margin-left: auto;
    border: none;
    background: transparent;
    color: var(--accent);
    font-size: var(--text-sm);
    font-weight: 600;
    padding: 0 var(--space-xs);
  }

  .view-all-sessions.active {
    color: var(--accent-hover);
  }

  .runtime-strip-orphan-flag {
    min-width: 0 !important;
    border-radius: var(--radius-full);
    background: var(--amber-soft) !important;
    color: var(--amber-text) !important;
  }

  .session-grid {
    display: flex;
    gap: 0;
    overflow-x: auto;
    padding-bottom: 0;
    scrollbar-width: thin;
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    background: var(--surface);
  }

  .runtime-strip.is-expanded .session-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    overflow-x: hidden;
    overflow-y: auto;
    max-height: 42vh;
    gap: 1px;
    background: var(--border);
  }

  .session-grid > :global(.empty) {
    flex: 1 1 auto;
    margin: 0;
    padding: var(--space-lg);
  }
</style>
