<script>
  import { onDestroy } from 'svelte';
  import { orphanKey, sessionKey } from './lib/dashboardState.js';
  import { resolveOpenSession } from './lib/sessionOutcome.js';
  import Icon from './Icon.svelte';
  import SessionCard from './SessionCard.svelte';
  import SessionModal from './SessionModal.svelte';

  export let runtimeSessions = [];
  export let closedSessions = [];
  export let orphanedTasks = [];
  export let onControlSession = async () => {};
  export let onReleaseSession = async () => {};
  export let onResolveOrphan = async () => {};
  export let refreshStatus = null;

  let tab = 'active';
  let expanded = false;
  let selected = null;
  let now = Date.now();
  let tick = setInterval(() => (now = Date.now()), 30000);
  onDestroy(() => clearInterval(tick));

  $: activeCount = runtimeSessions.length + orphanedTasks.length;
  $: count = tab === 'active' ? activeCount : closedSessions.length;
  $: emptyText = tab === 'active' ? 'No active sessions.' : 'No closed sessions.';
  $: openSession = resolveOpenSession(selected, runtimeSessions, closedSessions, orphanedTasks);

  function setTick(ms) {
    clearInterval(tick);
    tick = setInterval(() => (now = Date.now()), ms);
  }

  function openSelected(session, kind) {
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
