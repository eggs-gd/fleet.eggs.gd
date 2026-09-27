<script lang="ts">
  import { apiFetch, readApiError } from './lib/api';
  import type { AnyRecord } from './lib/types';

  // What the server found on this computer: [{ id, name, available }].
  export let providers: AnyRecord[] = [];
  export let onDone: () => void = () => {};
  export let onSkip: () => void = () => {};

  $: available = providers.filter((provider) => provider.available);
  let agent = '';
  $: if (!available.some((provider) => provider.id === agent)) agent = available[0]?.id || '';
  let creating = false;
  let error = '';

  async function create() {
    if (creating || !agent) return;
    creating = true;
    error = '';
    try {
      const response = await apiFetch('/api/manager/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent })
      });
      if (!response.ok) throw new Error(await readApiError(response));
      onDone();
    } catch (err: any) {
      error = err.message || String(err);
    } finally {
      creating = false;
    }
  }
</script>

<div class="modal-backdrop">
  <section class="task-modal" aria-label="Create manager session">
    <header>
      <h2>Create a Manager session</h2>
    </header>
    <p class="settings-lede">
      The Manager is a session in an agent you already use, with this data root as its folder. You
      talk to it in that app. Fleet does not keep a chat here. You can skip this and set it up later
      in Settings.
    </p>
    {#if available.length}
      <div class="settings-row">
        <span class="settings-key">Agent</span>
        <span class="settings-val">
          <select class="settings-input" bind:value={agent} disabled={creating}>
            {#each available as option (option.id)}
              <option value={option.id}>{option.name}</option>
            {/each}
          </select>
        </span>
      </div>
      {#if agent === 'codex'}
        <p class="settings-hint">
          Trusting this folder also allows Codex hooks and exec policy, not only the manager MCP
          server.
        </p>
      {/if}
      {#if agent === 'cursor'}
        <p class="settings-hint">
          Fleet does not start a Cursor process. Open this data root in Cursor. That chat stays on
          this computer.
        </p>
      {/if}
      <p class="settings-hint">
        The agent must already be signed in. If it is not, open it once and sign in first.
      </p>
    {:else}
      <p class="settings-hint">
        Fleet found no supported agent on this computer. Install Claude, Codex, Cursor, or Gemini
        and sign in, then reload this page.
      </p>
    {/if}
    {#if error}
      <p class="settings-error">{error}</p>
    {/if}
    <footer>
      {#if available.length}
        <button
          type="button"
          class="settings-btn is-primary"
          disabled={creating || !agent}
          on:click={create}
        >
          {creating ? 'Creating…' : 'Create session'}
        </button>
      {/if}
      <button type="button" class="settings-btn" disabled={creating} on:click={onSkip}>
        Set up later
      </button>
    </footer>
  </section>
</div>
