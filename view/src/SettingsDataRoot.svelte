<script lang="ts">
  import { onMount } from 'svelte';
  import { apiFetch, readApiError } from './lib/api';
  import SettingsField from './SettingsField.svelte';
  import type { AnyRecord } from './lib/types';

  let config = $state<AnyRecord | null>(null);
  let input = $state('');
  let loading = $state(false);
  let saving = $state(false);
  let confirming = $state(false);
  let error = $state('');
  let restartRequired = $state(false);
  let previousBinding = $state<AnyRecord | null>(null);
  let recreateOffer = $state(false);
  let recreating = $state(false);
  let recreateError = $state('');

  async function load() {
    loading = true;
    try {
      const response = await apiFetch('/api/app-config');
      if (!response.ok) throw new Error(await response.text());
      config = await response.json();
      input = config!.dataRoot || config!.effectiveRoot || '';
      error = '';
    } catch (err: any) {
      error = err.message || String(err);
    } finally {
      loading = false;
    }
  }

  function requestMove() {
    confirming = true;
  }

  function cancelMove() {
    confirming = false;
  }

  async function confirmMove() {
    if (saving) return;
    saving = true;
    try {
      const bindingResponse = await apiFetch('/api/manager/binding', { cache: 'no-store' });
      previousBinding = bindingResponse.ok ? await bindingResponse.json() : null;
      const response = await apiFetch('/api/app-config', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dataRoot: input })
      });
      if (!response.ok) throw new Error(await response.text());
      config = await response.json();
      restartRequired = restartRequired || config!.restartRequired;
      confirming = false;
      error = '';
      recreateOffer = Boolean(previousBinding?.bound);
    } catch (err: any) {
      error = err.message || String(err);
    } finally {
      saving = false;
    }
  }

  function dismissRecreate() {
    recreateOffer = false;
    recreateError = '';
  }

  async function recreateManager() {
    if (recreating || !previousBinding?.agent) return;
    recreating = true;
    recreateError = '';
    try {
      const response = await apiFetch('/api/manager/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent: previousBinding.agent, workspace: input.trim() })
      });
      if (!response.ok) throw new Error(await readApiError(response));
      recreateOffer = false;
    } catch (err: any) {
      recreateError = err.message || String(err);
    } finally {
      recreating = false;
    }
  }

  let changed = $derived(
    Boolean(config) && input.trim() !== (config?.dataRoot || config?.effectiveRoot || '')
  );

  onMount(load);
</script>

<section class="settings-block" aria-label="Data root">
  <h3>Data root</h3>
  {#if loading && !config}
    <p class="settings-hint">Loading…</p>
  {:else if config}
    <SettingsField label="Currently serving" value={config.effectiveRoot} mono />
    <SettingsField
      label="Source"
      value={config.source || '—'}
      hint="flag = --root on the command line, config = ~/.fleet/app.json, default = first-run ~/.fleet/workspace."
    />
    <div class="settings-row">
      <span class="settings-key">Data root path</span>
      <span class="settings-val">
        <input
          class="settings-input"
          type="text"
          bind:value={input}
          disabled={saving}
          placeholder="~/.fleet/workspace"
        />
      </span>
    </div>
    <p class="settings-hint">
      Changing this moves the Data directory on disk to the new path, then updates
      ~/.fleet/app.json. Requires a server restart to take effect.
    </p>

    {#if !confirming}
      <button
        type="button"
        class="settings-btn"
        disabled={!changed || saving}
        onclick={requestMove}
      >
        Move data root…
      </button>
    {:else}
      <div class="settings-savebar">
        <span>Move Data from {config.effectiveRoot} to {input.trim()}?</span>
        <button type="button" class="settings-btn" disabled={saving} onclick={cancelMove}
          >Cancel</button
        >
        <button
          type="button"
          class="settings-btn is-primary"
          disabled={saving}
          onclick={confirmMove}
        >
          {saving ? 'Moving…' : 'Confirm move'}
        </button>
      </div>
    {/if}

    {#if restartRequired}
      <p class="settings-warn">Data was moved. Restart Fleet to serve from the new path.</p>
    {/if}
    {#if recreateOffer}
      <div class="settings-savebar">
        <span>
          The Manager session still uses the old folder. Fleet cannot change its root. Recreate it
          in the new folder? The old session stays until you confirm.
          {#if previousBinding?.agent === 'codex'}
            Trusting the new folder also allows Codex hooks and exec policy, not only the manager
            MCP server.
          {/if}
        </span>
        <button type="button" class="settings-btn" disabled={recreating} onclick={dismissRecreate}
          >Leave it</button
        >
        <button
          type="button"
          class="settings-btn is-primary"
          disabled={recreating}
          onclick={recreateManager}
        >
          {recreating ? 'Recreating…' : 'Recreate'}
        </button>
      </div>
      {#if recreateError}
        <p class="settings-error">{recreateError}</p>
      {/if}
    {/if}
    {#if error}
      <p class="settings-error">{error}</p>
    {/if}
  {/if}
</section>
