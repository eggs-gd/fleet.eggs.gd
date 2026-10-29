<script lang="ts">
  import { onMount } from 'svelte';
  import { apiFetch, readApiError } from '../../lib/api';
  import SettingsField from './SettingsField.svelte';
  import type { AnyRecord } from '../../lib/types';

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

  const SOURCE_LABELS: Record<string, string> = {
    flag: '--root flag',
    config: 'saved setting',
    default: 'default location'
  };

  onMount(load);
</script>

<section class="settings-block" aria-label="Your data">
  <h3>Your data</h3>
  <p class="settings-lede">
    Tasks, projects and notes are plain Markdown files in one folder. It is private to you; keep it
    out of public repositories.
  </p>
  {#if loading && !config}
    <p class="settings-hint">Loading…</p>
  {:else if config}
    <SettingsField label="Data folder" value={config.effectiveRoot} mono />
    <SettingsField
      label="Chosen by"
      value={SOURCE_LABELS[config.source || ''] || config.source || '—'}
    />
    <div class="settings-row">
      <span class="settings-key">Move it to</span>
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
      Moving copies your data to the new folder and remembers it. Restart Fleet afterwards.
    </p>

    {#if !confirming}
      <button
        type="button"
        class="settings-btn"
        disabled={!changed || saving}
        onclick={requestMove}
      >
        Move data folder…
      </button>
    {:else}
      <div class="settings-savebar">
        <span>Move your data from {config.effectiveRoot} to {input.trim()}?</span>
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
      <p class="settings-warn">Your data was moved. Restart Fleet to use the new folder.</p>
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
