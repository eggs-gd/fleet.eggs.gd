<script>
  import ThemeSwitch from './ThemeSwitch.svelte';
  import SettingsField from './SettingsField.svelte';
  import SettingsDataRoot from './SettingsDataRoot.svelte';
  import SettingsProjects from './SettingsProjects.svelte';
  import { AUTO_REFRESH_MS } from './lib/layoutPrefs.js';

  export let general = {};
  export let projects = {};
  export let scanRoots = [];
  export let scanDirty = false;
  export let onScanRoots = () => {};
  export let onRescan = () => {};
  export let sessionTimeout = '';
  export let onSessionTimeout = () => {};
  export let launch = '';
  export let onLaunch = () => {};
  export let themePref = 'system';
  export let onThemeChange = () => {};
  export let refreshMs = 5000;
  export let onRefreshChange = () => {};

  const label = (ms) => `${Math.round(ms / 1000)}s`;
</script>

<section class="settings-block" aria-label="Runtime">
  <h3>Runtime</h3>
  <SettingsField label="Fleet version" value={general.version || '—'} />
  <SettingsField label="Build hash" value={general.build_hash || '—'} mono />
  <SettingsField label="Runtime status" value={general.runtime_status || '—'} />
  <SettingsField label="Launch mode" value={general.launch_mode || '—'} />
  <div class="settings-row">
    <span class="settings-key">Saved launch mode</span>
    <span class="settings-val">
      <select class="settings-input" value={launch} on:change={(e) => onLaunch(e.target.value)}>
        <option value="">Default (dry-run)</option>
        <option value="dry-run">Dry-run: plan launches only</option>
        <option value="live">Live: start agents for ready tasks</option>
      </select>
    </span>
  </div>
  <p class="settings-hint">
    {general.launch_config?.warning ||
      'Live mode starts AI agents that edit files without asking each time.'}
    {#if general.launch_config?.overridden_by}
      This process was started with {general.launch_config.overridden_by}.
    {/if}
  </p>
  <SettingsField label="Started at" value={general.started_at || '—'} />
  <SettingsField label="Uptime" value={general.uptime || '—'} />
  <SettingsField label="Host" value={general.host || '—'} />
  <SettingsField label="API endpoint" value={general.api_endpoint || '—'} mono />
  <SettingsField
    label="Session timeout"
    value={general.session_timeout || '—'}
    hint="Running process. Idle attention threshold, not a total session cap."
  />
  <div class="settings-row">
    <span class="settings-key">Saved timeout</span>
    <span class="settings-val">
      <input
        class="settings-input"
        value={sessionTimeout}
        placeholder="10m"
        on:input={(e) => onSessionTimeout(e.target.value)}
      />
    </span>
  </div>
  <p class="settings-hint">
    Written to core.local.yaml. New sessions use it after restart. An explicit --session-timeout
    overrides the file.
    {#if general.session_timeout_config?.overridden_by}
      This process was started with {general.session_timeout_config.overridden_by}.
    {/if}
  </p>
</section>

<SettingsDataRoot />

<SettingsProjects {projects} {scanRoots} {scanDirty} {onScanRoots} {onRescan} />

<section class="settings-block" aria-label="Browser preferences">
  <h3>Browser preferences</h3>
  <div class="settings-row">
    <span class="settings-key">Default theme</span>
    <span class="settings-val">
      <ThemeSwitch pref={themePref} onChange={onThemeChange} />
    </span>
  </div>
  <p class="settings-hint">
    {general.theme?.source ||
      'Same localStorage state as the sidebar switcher. Applies immediately.'}
  </p>
  <div class="settings-row">
    <span class="settings-key">Auto-refresh interval</span>
    <span class="settings-val">
      <select
        class="settings-input"
        value={refreshMs}
        on:change={(e) => onRefreshChange(Number(e.target.value))}
      >
        {#each AUTO_REFRESH_MS as ms}
          <option value={ms}>{label(ms)}</option>
        {/each}
      </select>
    </span>
  </div>
  <p class="settings-hint">
    Work board poll of /api/state. Stored in localStorage core.autoRefreshMs. Applies immediately.
  </p>
</section>

<section class="settings-block" aria-label="Startup behaviour">
  <h3>Startup behaviour</h3>
  <p class="settings-lede">
    Observed bootstrap. These are not toggles — Fleet does not expose startup flags yet.
  </p>
  {#each general.startup || [] as fact (fact.id)}
    <SettingsField label={fact.label} value={fact.value} hint={fact.source} />
  {/each}
</section>

{#if general.notes?.length}
  <ul class="settings-notes">
    {#each general.notes as note}<li>{note}</li>{/each}
  </ul>
{/if}
