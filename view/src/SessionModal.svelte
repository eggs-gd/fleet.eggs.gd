<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { elapsedClock, orphanRecoveryLabel, toolUsageHasMissing } from './lib/taskDisplay';
  import {
    elapsedClockPrecise,
    sessionOutcomeLabel,
    sessionOutcomeTone
  } from './lib/sessionOutcome';
  import SessionDetail from './SessionDetail.svelte';
  import type { Session } from './lib/types';

  export let session: Session;
  export let kind = 'closed';
  export let now = Date.now();
  export let onClose: () => void = () => {};
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

  $: live = kind === 'live';
  $: orphan = kind === 'orphan';
  $: tone = sessionOutcomeTone(session, kind);
  $: statusLabel = orphan
    ? orphanRecoveryLabel(session)
    : live && tone === 'live'
      ? `Running ${elapsedClockPrecise(session.started_at, now) || elapsedClock(session.started_at, now) || 'now'}`
      : sessionOutcomeLabel(session, kind);

  function closeFromBackdrop(event: MouseEvent) {
    if (event.target === event.currentTarget) onClose();
  }

  function onKey(event: KeyboardEvent) {
    if (event.key === 'Escape') onClose();
  }

  onMount(() => {
    window.addEventListener('keydown', onKey);
  });
  onDestroy(() => window.removeEventListener('keydown', onKey));
</script>

<section class="modal-backdrop" role="presentation" on:click={closeFromBackdrop}>
  <div
    class="task-modal session-modal"
    class:session-modal--live={live}
    class:tool-evidence-warning={toolUsageHasMissing(session)}
    role="dialog"
    aria-modal="true"
    aria-labelledby="session-title"
    tabindex="-1"
  >
    <header>
      <div>
        <p class="eyebrow">
          {session.agent || session.assignee || 'session'} · {session.repository ||
            session.project_id ||
            'session'}
        </p>
        <h2 id="session-title">
          {session.task_title || session.task_ref || session.task_id || 'Session'}
        </h2>
        <p
          class={`session-modal-status session-card-status--${tone}`}
          class:is-live={live && tone === 'live'}
        >
          <span class="status-dot"></span>
          {statusLabel}
        </p>
      </div>
      <div class="task-modal-actions">
        <button type="button" on:click={onClose}>Close</button>
      </div>
    </header>
    <SessionDetail {session} {kind} {onControlSession} {onReleaseSession} {onResolveOrphan} />
  </div>
</section>

<style>
  .session-modal {
    width: min(640px, 100%);
  }

  .session-modal-status {
    display: inline-flex;
    align-items: center;
    gap: var(--space-sm);
    margin: var(--space-md) 0 0;
    font-size: var(--text-md);
    font-weight: 700;
  }

  .session-modal-status .status-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--text-faint);
  }

  /* .session-message/.session-meta are SessionDetail's own classes, rendered
     as this modal's child. */
  .session-modal :global(.session-message) {
    overflow: visible;
    white-space: normal;
  }

  .session-modal :global(.session-meta code),
  .session-modal :global(.session-meta span),
  .session-modal :global(.session-meta a) {
    white-space: normal;
  }
</style>
