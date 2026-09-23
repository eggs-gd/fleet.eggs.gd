<script>
  import {
    codexRemoteInstruction,
    copyText,
    executionIdentity,
    formatRequiredToolChip,
    formatToolCall,
    orphanCapabilitySummary,
    providerErrorKind,
    providerErrorReason,
    providerErrorRetryPolicy,
    providerErrorSuggestedAction,
    remoteHostLabel,
    remoteThreadTitle,
    sessionResumeLabel,
    toolUsageMissing,
    toolUsageProfiles,
    toolUsageRequired,
    toolUsageUsed,
    toolUsageWarning
  } from './lib/taskDisplay.js';
  import { sessionAllowsOperatorRelease } from './lib/dashboardState.js';
  import { sessionStatus } from './lib/sessionOutcome.js';

  export let session;
  export let kind = 'closed';
  export let onControlSession = async () => {};
  export let onReleaseSession = async () => {};
  export let onResolveOrphan = async () => {};

  $: live = kind === 'live';
  $: orphan = kind === 'orphan';
  $: status = sessionStatus(session) || (live ? 'starting' : 'closed');
</script>

{#if orphan}
  <div class="session-meta">
    <span>{session.visibility_mode || 'unknown'}</span>
    {#each executionIdentity(session) as item (item)}
      <code>{item}</code>
    {/each}
    {#if session.last_activity_at}
      <span>last {new Date(session.last_activity_at).toLocaleString()}</span>
    {/if}
  </div>
  {#if orphanCapabilitySummary(session)}
    <p class="session-message" title={session.capabilities?.notes || ''}>{orphanCapabilitySummary(session)}</p>
  {/if}
  <div class="session-actions">
    {#each ['blocked', 'needs_rework', 'todo', 'done'] as nextStatus (nextStatus)}
      <button type="button" on:click={() => onResolveOrphan(session, nextStatus)}>{nextStatus}</button>
    {/each}
    {#if session.claim_id}
      <button
        type="button"
        on:click={() =>
          onReleaseSession(
            {
              claim_id: session.claim_id,
              execution_status: session.execution_state || 'orphaned',
              status: session.execution_state || 'orphaned'
            },
            'release',
            'needs_review'
          )}
      >
        Release claim
      </button>
    {/if}
  </div>
{:else}
  <div class="session-meta">
    <span>{status}</span>
    <span>{session.visibility_mode || 'unknown'}</span>
    {#if session.resumable === false}
      <span class="session-role role-unresumable" title="This provider/backend cannot auto-resume this session">not auto-resumable</span>
    {/if}
    {#if sessionResumeLabel(session)}
      <span title="Resume outcome the launcher recorded for this session">{sessionResumeLabel(session)}</span>
    {/if}
    {#if session.supersedes_claim_id}
      <span title="This session supersedes an earlier session for this task">supersedes {session.supersedes_claim_id}</span>
    {/if}
    {#if session.remote_control_url}
      <a href={session.remote_control_url} target="_blank" rel="noreferrer">Open session</a>
    {:else if session.operator_command}
      <code>{session.operator_command}</code>
    {:else if session.codex_thread_title || session.codex_thread_id}
      <code>{session.codex_thread_title || session.codex_thread_id}</code>
    {/if}
    {#if session.claim_id}
      <code>claim {session.claim_id}</code>
    {/if}
    {#if session.process_id}
      <code>pid {session.process_id}</code>
    {/if}
    {#if remoteThreadTitle(session)}
      <code>title {remoteThreadTitle(session)}</code>
    {/if}
    {#if remoteHostLabel(session)}
      <code>host {remoteHostLabel(session)}</code>
    {/if}
    {#if session.last_event}
      <span>{session.last_event}</span>
    {/if}
    {#if session.last_event_at}
      <span>event {new Date(session.last_event_at).toLocaleTimeString()}</span>
    {/if}
    {#if session.last_output_at}
      <span>output {new Date(session.last_output_at).toLocaleTimeString()}</span>
    {/if}
    {#if session.last_status_change_at}
      <span>status {new Date(session.last_status_change_at).toLocaleTimeString()}</span>
    {/if}
    {#if session.codex_thread_id}
      <code>thread {session.codex_thread_id}</code>
    {/if}
    {#if session.codex_turn_id}
      <code>turn {session.codex_turn_id}</code>
    {/if}
    {#if session.cursor_chat_id}
      <code>cursor {session.cursor_chat_id}</code>
    {/if}
    {#if session.background_id}
      <code>bg {session.background_id}</code>
    {/if}
    {#if session.log_path}
      <code>{session.log_path}</code>
    {/if}
  </div>
  {#if session.tool_usage}
    <section class="tool-evidence" aria-label="Tool usage evidence">
      <strong>Tool evidence</strong>
      {#if toolUsageProfiles(session).length}
        <div class="session-meta">
          {#each toolUsageProfiles(session) as profile (profile)}
            <span>profile {profile}</span>
          {/each}
        </div>
      {/if}
      <div class="session-meta">
        {#each toolUsageRequired(session) as tool (tool)}
          <span class:tool-missing={toolUsageMissing(session).includes(tool)}
            >{formatRequiredToolChip(tool, session)}</span
          >
        {:else}
          <span>no required tools</span>
        {/each}
        {#each toolUsageUsed(session) as call (formatToolCall(call))}
          <code title={call.summary || call.source || ''}>{formatToolCall(call)}</code>
        {/each}
      </div>
      {#if toolUsageWarning(session)}
        <p class="session-message tool-warning-text">{toolUsageWarning(session)}</p>
      {/if}
    </section>
  {/if}
  {#if session.agent === 'codex'}
    <p class="session-message">{codexRemoteInstruction(session)}</p>
    <div class="session-actions">
      {#if session.codex_thread_id}
        <button type="button" on:click={() => copyText(session.codex_thread_id)}>Copy thread id</button>
      {/if}
      {#if session.codex_turn_id}
        <button type="button" on:click={() => copyText(session.codex_turn_id)}>Copy turn id</button>
      {/if}
      {#if remoteThreadTitle(session)}
        <button type="button" on:click={() => copyText(remoteThreadTitle(session))}>Copy title</button>
      {/if}
    </div>
  {/if}
  {#if session.operator_command}
    <p class="session-message">Resume: <code>{session.operator_command}</code></p>
  {/if}
  {#if session.capabilities?.notes}
    <p class="session-message">{session.capabilities.notes}</p>
  {/if}
  {#if providerErrorReason(session)}
    <section class="provider-error" aria-label="Provider error">
      <strong>{providerErrorKind(session) || 'provider_error'}</strong>
      <p>{providerErrorReason(session)}</p>
      {#if providerErrorSuggestedAction(session)}
        <p>{providerErrorSuggestedAction(session)}</p>
      {/if}
      {#if providerErrorRetryPolicy(session)}
        <p>Retry policy: {providerErrorRetryPolicy(session)}</p>
      {/if}
    </section>
  {/if}
  {#if live && session.provider_controllable && session.agent === 'codex'}
    <div class="session-actions">
      <button type="button" on:click={() => onControlSession(session, 'continue', 'Continue from the current Core task context.')}>Continue</button>
      <button type="button" on:click={() => onControlSession(session, 'interrupt')}>Interrupt</button>
      <button type="button" on:click={() => onControlSession(session, 'cancel')}>Cancel</button>
    </div>
  {:else if sessionAllowsOperatorRelease(session)}
    <div class="session-actions">
      <button type="button" on:click={() => onReleaseSession(session, 'release', 'needs_review')}>Release → review</button>
      <button type="button" on:click={() => onReleaseSession(session, 'release', 'needs_rework')}>Release → rework</button>
      <button type="button" on:click={() => onReleaseSession(session, 'mark_dead', 'blocked')}>Mark dead</button>
      <button type="button" on:click={() => onReleaseSession(session, 'mark_provider_unavailable', 'blocked')}>Provider unavailable</button>
      <button type="button" on:click={() => onReleaseSession(session, 'cancel_without_provider_control', 'needs_rework')}>Cancel without control</button>
    </div>
  {/if}
  {#if session.result}
    <span class="session-result-chip" title={session.result.summary || ''}>result {session.result.outcome}</span>
  {/if}
  {#if session.result?.summary}
    <p class="session-message" title={session.result.summary}>{session.result.summary}</p>
  {:else if session.last_message}
    <p class="session-message">{session.last_message}</p>
  {/if}
  {#if session.blocking_reason}
    <p class="session-message">{session.blocking_reason}</p>
  {/if}
{/if}
