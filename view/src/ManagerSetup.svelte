<script>
  const providers = [
    { id: 'codex', label: 'Codex' },
    { id: 'claude', label: 'Claude' },
    { id: 'gemini', label: 'Gemini' }
  ];

  export let onDone = () => {};

  let agent = 'codex';
  let creating = false;
  let error = '';

  async function create() {
    if (creating) return;
    creating = true;
    error = '';
    try {
      const response = await fetch('/api/manager/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent })
      });
      if (!response.ok) throw new Error(await response.text());
      onDone();
    } catch (err) {
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
      Fleet will start a session in the provider you already use, with this data root as its folder.
      You talk to it in that app. Fleet does not keep a chat here.
    </p>
    <div class="settings-row">
      <span class="settings-key">Provider</span>
      <span class="settings-val">
        <select class="settings-input" bind:value={agent} disabled={creating}>
          {#each providers as option (option.id)}
            <option value={option.id}>{option.label}</option>
          {/each}
        </select>
      </span>
    </div>
    {#if agent === 'codex'}
      <p class="settings-hint">Trusting this folder also allows Codex hooks and exec policy, not only the manager MCP server.</p>
    {/if}
    {#if error}
      <p class="settings-error">{error}</p>
    {/if}
    <footer>
      <button type="button" class="settings-btn is-primary" disabled={creating} on:click={create}>
        {creating ? 'Creating…' : 'Create session'}
      </button>
    </footer>
  </section>
</div>
