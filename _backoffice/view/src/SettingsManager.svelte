<script>
  import SettingsField from './SettingsField.svelte';
  import { timeAgo } from './lib/taskDisplay.js';

  const MANAGER_AGENTS = [
    { id: 'codex', label: 'Codex' },
    { id: 'claude', label: 'Claude' },
    { id: 'cursor', label: 'Cursor' }
  ];

  export let manager = {};
  export let draft = { agent: '', threadId: '' };
  export let onUpdate = () => {};

  $: bound = Boolean(draft?.agent && draft?.threadId);

  let threads = [];
  let threadsLoading = false;
  let threadsError = '';
  let loadedForAgent = '';

  async function loadThreads(agent) {
    loadedForAgent = agent;
    threadsLoading = true;
    threadsError = '';
    threads = [];
    try {
      const response = await fetch(`/api/manager/threads?agent=${encodeURIComponent(agent)}`);
      const body = await response.text();
      if (!response.ok) throw new Error(body || `Failed to list sessions (${response.status})`);
      threads = body ? JSON.parse(body) : [];
    } catch (err) {
      threadsError = err.message || String(err);
    } finally {
      threadsLoading = false;
    }
  }

  $: if (draft?.agent && draft.agent !== loadedForAgent) loadThreads(draft.agent);

  function unbind() {
    onUpdate({ agent: '', threadId: '' });
    threads = [];
    loadedForAgent = '';
  }
</script>

<section class="settings-block" aria-label="Manager">
  <h3>Manager</h3>
  <p class="settings-lede">Manager is a role, not another executor provider.</p>
  <SettingsField label="Role" value={manager.role || '—'} />

  <div class="settings-row">
    <span class="settings-key">Bind to session</span>
    <span class="settings-val">
      <select
        class="settings-input"
        value={draft?.agent || ''}
        on:change={(e) => onUpdate({ agent: e.target.value })}
      >
        <option value="">not bound</option>
        {#each MANAGER_AGENTS as option (option.id)}
          <option value={option.id}>{option.label}</option>
        {/each}
      </select>
    </span>
  </div>
  {#if draft?.agent}
    <div class="settings-row">
      <span class="settings-key">Pick a recent session</span>
      <span class="settings-val">
        {#if threadsLoading}
          <span class="settings-hint">Loading sessions…</span>
        {:else if threads.length}
          <select
            class="settings-input"
            value={threads.some((t) => t.id === draft?.threadId) ? draft.threadId : ''}
            on:change={(e) => onUpdate({ threadId: e.target.value })}
          >
            <option value="">choose one…</option>
            {#each threads as thread (thread.id)}
              <option value={thread.id}>{thread.name || '(untitled)'} — {timeAgo(thread.updatedAt) || thread.updatedAt}</option>
            {/each}
          </select>
        {:else if threadsError}
          <span class="settings-hint">{threadsError}</span>
        {:else}
          <span class="settings-hint">No sessions found.</span>
        {/if}
      </span>
    </div>
    <div class="settings-row">
      <span class="settings-key">Session / thread id</span>
      <span class="settings-val">
        <input
          class="settings-input is-mono"
          type="text"
          value={draft?.threadId || ''}
          placeholder="existing session id to route Manager Bar messages into"
          on:input={(e) => onUpdate({ threadId: e.target.value })}
        />
      </span>
    </div>
  {/if}
  {#if bound}
    <p class="settings-hint">
      Manager Bar text/voice routes directly into this session. Save to apply.
      <button type="button" class="settings-btn" on:click={unbind}>Unbind</button>
    </p>
  {:else}
    <p class="settings-hint">
      No manager session bound — Manager Bar text goes through fast-path command parsing only.
      Paste the id of an already-running session above to bind it.
    </p>
  {/if}

  <SettingsField label="Provider" value={manager.provider || '—'} />
  <SettingsField label="Session" value={manager.session ? `${manager.session.status} ${manager.session.id || ''}`.trim() : 'none'} />
  <SettingsField label="STT" value={manager.stt || '—'} />
  <SettingsField label="Classifier" value={manager.classifier || '—'} />
  <SettingsField label="Fast-path commands" value={manager.fast_path ? 'yes' : 'no'} />
  <SettingsField label="Endpoints" value={(manager.endpoints || []).join('  ')} mono />
  {#if manager.instructions}
    <p class="settings-lede">Instructions ({manager.instructions_source})</p>
    <pre class="settings-pre">{manager.instructions}</pre>
  {/if}
</section>

{#if manager.notes?.length}
  <ul class="settings-notes">
    {#each manager.notes as note}<li>{note}</li>{/each}
  </ul>
{/if}
