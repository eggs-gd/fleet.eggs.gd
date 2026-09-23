<script context="module">
  let cachedSnapshot = null;
</script>
<script>
  import { onMount } from 'svelte';
  import { settingsSectionLabel } from './lib/settingsNav.js';
  import { draftFromSnapshot, patchFromDraft } from './lib/settingsDraft.js';
  import SettingsAgents from './SettingsAgents.svelte';
  import SettingsDiagnostics from './SettingsDiagnostics.svelte';
  import SettingsGeneral from './SettingsGeneral.svelte';
  import SettingsIntegrations from './SettingsIntegrations.svelte';
  import SettingsManager from './SettingsManager.svelte';
  import SettingsProjects from './SettingsProjects.svelte';
  import SettingsWorkflow from './SettingsWorkflow.svelte';

  export let section = 'general';
  export let title = '';
  export let themePref = 'system';
  export let onThemeChange = () => {};
  export let refreshMs = 5000;
  export let onRefreshChange = () => {};

  let snapshot = cachedSnapshot;
  let error = '';
  let warnings = [];
  let loading = false;
  let saving = false;
  let baseline = null;
  let draft = null;

  $: scanKey = draft ? `${draft.scanRoot}|${JSON.stringify(draft.agents)}|${JSON.stringify(draft.manager)}` : '';
  $: baseKey = baseline ? `${baseline.scanRoot}|${JSON.stringify(baseline.agents)}|${JSON.stringify(baseline.manager)}` : '';
  $: dirty = Boolean(draft && baseline && scanKey !== baseKey);

  function acceptSnapshot(next) {
    snapshot = next;
    cachedSnapshot = next;
    if (!dirty) {
      baseline = draftFromSnapshot(next);
      draft = draftFromSnapshot(next);
    }
  }

  async function loadSettings() {
    if (loading || dirty) return;
    loading = true;
    try {
      const response = await fetch('/api/settings');
      if (!response.ok) {
        throw new Error(await response.text());
      }
      acceptSnapshot(await response.json());
      error = '';
    } catch (err) {
      error = err.message || String(err);
    } finally {
      loading = false;
    }
  }

  async function saveSettings() {
    if (!dirty || saving) return;
    saving = true;
    warnings = [];
    try {
      const response = await fetch('/api/settings', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(patchFromDraft(draft, baseline))
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      const result = await response.json();
      warnings = result.warnings || [];
      baseline = draftFromSnapshot(result.snapshot);
      draft = draftFromSnapshot(result.snapshot);
      snapshot = result.snapshot;
      cachedSnapshot = result.snapshot;
      error = '';
    } catch (err) {
      error = err.message || String(err);
    } finally {
      saving = false;
    }
  }

  function discardSettings() {
    if (!snapshot) return;
    baseline = draftFromSnapshot(snapshot);
    draft = draftFromSnapshot(snapshot);
    error = '';
  }

  async function recheck() {
    if (dirty) return;
    loading = true;
    try {
      const response = await fetch('/api/settings/recheck', { method: 'POST' });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      acceptSnapshot(await response.json());
      error = '';
    } catch (err) {
      error = err.message || String(err);
    } finally {
      loading = false;
    }
  }

  let scanTimer = 0;

  async function rescan() {
    if (dirty) return;
    try {
      const response = await fetch('/api/settings/rescan', { method: 'POST' });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      error = '';
      await loadSettings();
      window.clearInterval(scanTimer);
      scanTimer = window.setInterval(async () => {
        await loadSettings();
        if (snapshot?.projects?.scan?.state !== 'scanning') {
          window.clearInterval(scanTimer);
          scanTimer = 0;
        }
      }, 1000);
    } catch (err) {
      error = err.message || String(err);
    }
  }

  function updateAgent(id, patch) {
    draft = {
      ...draft,
      agents: {
        ...draft.agents,
        [id]: { ...draft.agents[id], ...patch }
      }
    };
  }

  function updateManager(patch) {
    draft = { ...draft, manager: { ...draft.manager, ...patch } };
  }

  onMount(() => {
    loadSettings();
    const timer = window.setInterval(loadSettings, 5000);
    return () => {
      window.clearInterval(timer);
      window.clearInterval(scanTimer);
    };
  });
</script>

<section class="settings-view" aria-label="Settings">
  <header class="settings-head">
    <div>
      <h2>{title || settingsSectionLabel(section)}</h2>
      <p class="settings-mode">{dirty ? 'Unsaved changes' : 'Effective configuration'}</p>
    </div>
    {#if error}
      <p class="settings-error">{error}</p>
    {/if}
  </header>

  {#if dirty}
    <div class="settings-savebar">
      <span>Save writes gitignored core.local.yaml. Browser prefs are not included.</span>
      <button type="button" class="settings-btn" disabled={saving} on:click={discardSettings}>Discard</button>
      <button type="button" class="settings-btn is-primary" disabled={saving} on:click={saveSettings}>
        {saving ? 'Saving…' : 'Save'}
      </button>
    </div>
  {/if}

  {#each warnings as warning}
    <p class="settings-warn">{warning}</p>
  {/each}

  {#if !snapshot && !error}
    <p class="settings-hint">Loading Core configuration…</p>
  {:else if snapshot && draft}
    {#if section === 'general'}
      <SettingsGeneral general={snapshot.general} {themePref} {onThemeChange} {refreshMs} {onRefreshChange} />
    {:else if section === 'projects'}
      <SettingsProjects
        projects={snapshot.projects}
        bind:scanRoot={draft.scanRoot}
        scanDirty={draft.scanRoot !== baseline.scanRoot}
        onScanRoot={(value) => (draft = { ...draft, scanRoot: value })}
        onRescan={rescan}
      />
    {:else if section === 'agents'}
      <SettingsAgents agents={snapshot.agents} {draft} onUpdate={updateAgent} onRecheck={recheck} {dirty} />
    {:else if section === 'manager'}
      <SettingsManager manager={snapshot.manager} draft={draft.manager} onUpdate={updateManager} />
    {:else if section === 'workflow'}
      <SettingsWorkflow workflow={snapshot.workflow} />
    {:else if section === 'integrations'}
      <SettingsIntegrations integrations={snapshot.integrations} />
    {:else if section === 'diagnostics'}
      <SettingsDiagnostics diagnostics={snapshot.diagnostics} inventory={snapshot.inventory} />
    {/if}
  {/if}
</section>
