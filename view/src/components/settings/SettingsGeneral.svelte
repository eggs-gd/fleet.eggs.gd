<script lang="ts">
  import ThemeSwitch from '../shell/ThemeSwitch.svelte';
  import SettingsDataRoot from './SettingsDataRoot.svelte';
  import SettingsProjects from './SettingsProjects.svelte';
  import { AUTO_REFRESH_MS } from '../../lib/layoutPrefs';
  import type { AnyRecord } from '../../lib/types';

  export let general: AnyRecord = {};
  export let projects: AnyRecord = {};
  export let scanRoots: string[] = [];
  export let scanDirty = false;
  export let onScanRoots: (roots: string[]) => void = () => {};
  export let onRescan: () => void = () => {};
  export let sessionTimeout = '';
  export let onSessionTimeout: (value: string) => void = () => {};
  export let launch = '';
  export let onLaunch: (value: string) => void = () => {};
  export let themePref = 'system';
  export let onThemeChange: (pref: string) => void = () => {};
  export let refreshMs = 5000;
  export let onRefreshChange: (ms: number) => void = () => {};

  const label = (ms: number) => `${Math.round(ms / 1000)}s`;
  $: live = general.launch_mode === 'live';
</script>

<section class="settings-block" aria-label="Launching agents">
  <h3>Launching agents</h3>
  <p class="settings-lede">
    {#if live}
      Fleet is <strong>live</strong>: it starts your agents on tasks in <code>todo</code>. They edit
      files in your repositories without asking each time.
    {:else}
      Fleet is in <strong>dry-run</strong> mode: it plans what it would start, and starts nothing.
    {/if}
  </p>
  <div class="settings-row">
    <label class="settings-key" for="saved-launch-mode">Mode on next start</label>
    <span class="settings-val">
      <select
        id="saved-launch-mode"
        class="settings-input"
        name="launch_mode"
        autocomplete="off"
        value={launch}
        on:change={(e) => onLaunch(e.currentTarget.value)}
      >
        <option value="">Dry-run (default)</option>
        <option value="dry-run">Dry-run: plan launches only</option>
        <option value="live">Live: start agents for ready tasks</option>
      </select>
    </span>
  </div>
  <p class="settings-hint">
    {general.launch_config?.warning ||
      'Live mode starts AI agents that edit files without asking each time.'}
    {#if general.launch_config?.overridden_by}
      This run was started with {general.launch_config.overridden_by}, which takes priority.
    {/if}
  </p>
</section>

<SettingsProjects {projects} {scanRoots} {scanDirty} {onScanRoots} {onRescan} />

<SettingsDataRoot />

<section class="settings-block" aria-label="Appearance">
  <h3>Appearance</h3>
  <div class="settings-row">
    <span class="settings-key">Theme</span>
    <span class="settings-val">
      <ThemeSwitch pref={themePref} onChange={onThemeChange} />
    </span>
  </div>
  <div class="settings-row">
    <label class="settings-key" for="auto-refresh-interval">Refresh the board every</label>
    <span class="settings-val">
      <select
        id="auto-refresh-interval"
        class="settings-input"
        name="refresh_interval"
        autocomplete="off"
        value={refreshMs}
        on:change={(e) => onRefreshChange(Number(e.currentTarget.value))}
      >
        {#each AUTO_REFRESH_MS as ms}
          <option value={ms}>{label(ms)}</option>
        {/each}
      </select>
    </span>
  </div>
  <p class="settings-hint">These apply immediately and are remembered in this browser.</p>
</section>

<section class="settings-block" aria-label="Advanced">
  <details class="settings-tech">
    <summary>Advanced</summary>
    <div class="settings-row">
      <label class="settings-key" for="saved-session-timeout">Silent agent timeout</label>
      <span class="settings-val">
        <input
          id="saved-session-timeout"
          class="settings-input"
          name="session_timeout"
          autocomplete="off"
          value={sessionTimeout}
          placeholder="10m"
          on:input={(e) => onSessionTimeout(e.currentTarget.value)}
        />
      </span>
    </div>
    <p class="settings-hint">
      How long an agent can stay silent before Fleet asks for your attention. It is not a limit on
      how long a task may run. Applies to new sessions after a restart; currently {general.session_timeout ||
        '10m'}.
      {#if general.session_timeout_config?.overridden_by}
        This run was started with {general.session_timeout_config.overridden_by}, which takes
        priority.
      {/if}
    </p>
  </details>
</section>
