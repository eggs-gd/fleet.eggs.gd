<script>
  import { onDestroy, onMount } from 'svelte';
  import { elapsedClock, orphanRecoveryLabel, toolUsageHasMissing } from './lib/taskDisplay.js';
  import {
    elapsedClockPrecise,
    sessionOutcomeLabel,
    sessionOutcomeTone
  } from './lib/sessionOutcome.js';
  import SessionDetail from './SessionDetail.svelte';

  export let session;
  export let kind = 'closed';
  export let now = Date.now();
  export let onClose = () => {};
  export let onControlSession = async () => {};
  export let onReleaseSession = async () => {};
  export let onResolveOrphan = async () => {};

  $: live = kind === 'live';
  $: orphan = kind === 'orphan';
  $: tone = sessionOutcomeTone(session, kind);
  $: statusLabel = orphan
    ? orphanRecoveryLabel(session)
    : live && tone === 'live'
      ? `Running ${elapsedClockPrecise(session.started_at, now) || elapsedClock(session.started_at, now) || 'now'}`
      : sessionOutcomeLabel(session, kind);

  function closeFromBackdrop(event) {
    if (event.target === event.currentTarget) onClose();
  }

  function onKey(event) {
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
