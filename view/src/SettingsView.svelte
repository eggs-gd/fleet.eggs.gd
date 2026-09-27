<script context="module" lang="ts">
  import type { AnyRecord } from './lib/types';

  let cachedSnapshot: AnyRecord | null = null;
</script>

<script lang="ts">
  import { onMount } from 'svelte';
  import { apiFetch } from './lib/api';
  import { settingsSectionLabel } from './lib/settingsNav';
  import { draftFromSnapshot, patchFromDraft } from './lib/settingsDraft';
  import SettingsAgents from './SettingsAgents.svelte';
  import SettingsDiagnostics from './SettingsDiagnostics.svelte';
  import SettingsGeneral from './SettingsGeneral.svelte';
  import SettingsIntegrations from './SettingsIntegrations.svelte';
  import SettingsManager from './SettingsManager.svelte';
  import SettingsWorkflow from './SettingsWorkflow.svelte';
  import type { SettingsDraft, SettingsDraftAgent } from './lib/types';

  export let section = 'general';
  export let title = '';
  export let themePref = 'system';
  export let onThemeChange: (pref: string) => void = () => {};
  export let refreshMs = 5000;
  export let onRefreshChange: (ms: number) => void = () => {};

  let snapshot: AnyRecord | null = cachedSnapshot;
  let error = '';
  let warnings: string[] = [];
  let loading = false;
  let saving = false;
  let baseline: SettingsDraft | null = null;
  let draft: SettingsDraft | null = null;

  $: scanKey = draft
    ? `${JSON.stringify(draft.scanRoots)}|${draft.sessionTimeout || ''}|${draft.launch || ''}|${JSON.stringify(draft.agents)}|${JSON.stringify(draft.manager)}`
    : '';
  $: baseKey = baseline
    ? `${JSON.stringify(baseline.scanRoots)}|${baseline.sessionTimeout || ''}|${baseline.launch || ''}|${JSON.stringify(baseline.agents)}|${JSON.stringify(baseline.manager)}`
    : '';
  $: dirty = Boolean(draft && baseline && scanKey !== baseKey);

  function acceptSnapshot(next: AnyRecord) {
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
      const response = await apiFetch('/api/settings');
      if (!response.ok) {
        throw new Error(await response.text());
      }
      acceptSnapshot(await response.json());
      error = '';
    } catch (err: any) {
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
      const response = await apiFetch('/api/settings', {
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
    } catch (err: any) {
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
      const response = await apiFetch('/api/settings/recheck', { method: 'POST' });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      acceptSnapshot(await response.json());
      error = '';
    } catch (err: any) {
      error = err.message || String(err);
    } finally {
      loading = false;
    }
  }

  let scanTimer = 0;

  async function rescan() {
    if (dirty) return;
    try {
      const response = await apiFetch('/api/settings/rescan', { method: 'POST' });
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
    } catch (err: any) {
      error = err.message || String(err);
    }
  }

  function updateAgent(id: string, patch: SettingsDraftAgent) {
    draft = {
      ...draft,
      agents: {
        ...draft?.agents,
        [id]: { ...draft?.agents?.[id], ...patch }
      }
    };
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
      <button type="button" class="settings-btn" disabled={saving} on:click={discardSettings}
        >Discard</button
      >
      <button
        type="button"
        class="settings-btn is-primary"
        disabled={saving}
        on:click={saveSettings}
      >
        {saving ? 'Saving…' : 'Save'}
      </button>
    </div>
  {/if}

  {#each warnings as warning}
    <p class="settings-warn">{warning}</p>
  {/each}

  {#if !snapshot && !error}
    <p class="settings-hint">Loading Fleet configuration…</p>
  {:else if snapshot && draft}
    {#if section === 'general'}
      <SettingsGeneral
        general={snapshot.general}
        projects={snapshot.projects}
        scanRoots={draft.scanRoots}
        scanDirty={JSON.stringify(draft.scanRoots) !== JSON.stringify(baseline?.scanRoots)}
        onScanRoots={(value) => (draft = { ...draft, scanRoots: value })}
        sessionTimeout={draft.sessionTimeout || ''}
        onSessionTimeout={(value) => (draft = { ...draft, sessionTimeout: value })}
        launch={draft.launch || ''}
        onLaunch={(value) => (draft = { ...draft, launch: value })}
        onRescan={rescan}
        {themePref}
        {onThemeChange}
        {refreshMs}
        {onRefreshChange}
      />
    {:else if section === 'agents'}
      <SettingsManager manager={snapshot.manager} onReload={loadSettings} />
      <SettingsAgents
        agents={snapshot.agents}
        {draft}
        onUpdate={updateAgent}
        onRecheck={recheck}
        {dirty}
      />
    {:else if section === 'workflow'}
      <SettingsWorkflow workflow={snapshot.workflow} />
    {:else if section === 'integrations'}
      <SettingsIntegrations integrations={snapshot.integrations} />
    {:else if section === 'diagnostics'}
      <SettingsDiagnostics diagnostics={snapshot.diagnostics} inventory={snapshot.inventory} />
    {/if}
  {/if}
</section>
