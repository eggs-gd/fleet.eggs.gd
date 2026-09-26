<script>
  import {
    agentColorClass,
    elapsedClock,
    orphanRecoveryLabel,
    toolUsageHasMissing
  } from './lib/taskDisplay.js';
  import { sessionOutcomeLabel, sessionOutcomeTone } from './lib/sessionOutcome.js';
  import Icon from './Icon.svelte';

  export let session;
  export let kind = 'live';
  export let now = Date.now();
  export let onOpen = () => {};
  export let onResolveOrphan = null;
  export let onReleaseSession = null;

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

  function onKey(event) {
    // Ignore keydowns that bubbled up from a nested action button (e.g.
    // pressing Space to click "Blocked") — only the card itself opening the
    // modal on Enter/Space should reach here.
    if (event.target !== event.currentTarget) return;
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      open();
    }
  }

  function resolve(event, targetStatus) {
    event.preventDefault();
    event.stopPropagation();
    onResolveOrphan?.(session, targetStatus);
  }

  function releaseClaim(event) {
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
      {session.blocking_reason || session.reason || 'Core lost track of this session.'}
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
