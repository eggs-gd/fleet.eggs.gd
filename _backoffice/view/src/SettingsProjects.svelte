<script>
  import SettingsField from './SettingsField.svelte';
  import { fieldHint } from './lib/settingsDraft.js';

  export let projects = {};
  export let scanRoot = '';
  export let scanDirty = false;
  export let onScanRoot = () => {};
  export let onRescan = () => {};

  const reachable = (root) => (root.reachable ? 'reachable' : 'missing');
  $: scan = projects.scan || {};
  $: scanning = scan.state === 'scanning';
</script>

<section class="settings-block" aria-label="Project scan root">
  <h3>Project scan root</h3>
  <p class="settings-lede">
    One scan root. Multiple roots are {projects.multi_root_supported ? 'supported' : 'not implemented'}.
    Save, then Rescan — Recheck is for agents, not this tree.
    Per-project repositories and PROJECT.md are on the selected project's Settings tab.
  </p>
  <div class="settings-row">
    <span class="settings-key">Scan root</span>
    <span class="settings-val">
      <input
        class="settings-input is-mono"
        type="text"
        bind:value={scanRoot}
        on:input={() => onScanRoot(scanRoot)}
      />
    </span>
  </div>
  <p class="settings-hint">{fieldHint(projects.scan_root)}</p>
  <div class="settings-row">
    <span class="settings-key">Rescan</span>
    <span class="settings-val">
      <button type="button" class="settings-btn" disabled={scanning || scanDirty} on:click={onRescan}>
        {scanning ? 'Scanning…' : 'Rescan'}
      </button>
      <span class="settings-item-meta">{scan.state || 'idle'}{scan.finished_at ? ` · ${scan.finished_at}` : ''}</span>
    </span>
  </div>
  {#if scanDirty}
    <p class="settings-hint">Save the scan root before Rescan.</p>
  {/if}
  {#if scan.error}
    <p class="settings-warn">{scan.error}</p>
  {/if}
</section>

<section class="settings-block" aria-label="Project roots">
  <h3>Observed roots</h3>
  {#each projects.roots || [] as root (root.kind + root.path)}
    <article class="settings-item">
      <div class="settings-item-head">
        <span class={`status-dot-inline is-${root.reachable ? 'ok' : 'warn'}`}></span>
        <strong>{root.kind}</strong>
        <span class="settings-pill">{reachable(root)}</span>
      </div>
      <code class="settings-path">{root.path}</code>
      <SettingsField label="Source" value={root.source} />
      {#if root.repository_count != null}
        <SettingsField label="Repositories" value={String(root.repository_count)} />
      {/if}
      {#if root.workspace_count}
        <SettingsField label="Workspaces" value={String(root.workspace_count)} />
      {/if}
      {#if root.project_count}
        <SettingsField label="Projects" value={String(root.project_count)} />
      {/if}
      {#if root.last_scan}
        <SettingsField label="Last scan" value={root.last_scan} />
      {/if}
      {#if root.notes}
        <p class="settings-hint">{root.notes}</p>
      {/if}
    </article>
  {/each}
</section>

<section class="settings-block" aria-label="Core data">
  <h3>Core data</h3>
  {#each projects.data_paths || [] as path (path.name)}
    <article class="settings-item">
      <div class="settings-item-head">
        <span class={`status-dot-inline is-${path.exists && path.writable ? 'ok' : path.exists ? 'warn' : 'fail'}`}></span>
        <strong>{path.name}</strong>
        <span class="settings-pill">{path.exists ? (path.writable ? 'writable' : 'not writable') : 'missing'}</span>
      </div>
      <code class="settings-path">{path.path}</code>
      <p class="settings-hint">{path.role}</p>
    </article>
  {/each}
</section>

<section class="settings-block" aria-label="Task backend">
  <h3>Task backend</h3>
  <SettingsField label="Active" value={projects.task_backend?.active || '—'} />
  <SettingsField label="Config file" value={projects.task_backend?.config_file || '—'} mono />
  <SettingsField label="Exists" value={projects.task_backend?.config_file_exists ? 'yes' : 'no'} />
  <SettingsField label="Implementations in code" value={(projects.task_backend?.implementations || []).join(', ')} />
  {#if projects.task_backend?.plane}
    <p class="settings-lede">Plane (declared in config, not a fake selector)</p>
    <SettingsField label="Active" value={projects.task_backend.plane.active ? 'yes' : 'no'} />
    <SettingsField label="Workspace" value={projects.task_backend.plane.workspace || '—'} />
    <SettingsField label="Base URL" value={projects.task_backend.plane.base_url || '—'} mono />
    <SettingsField label="Token env" value={projects.task_backend.plane.token_env || '—'} />
    <SettingsField label="Token env set" value={projects.task_backend.plane.token_env_set ? 'yes' : 'no'} />
    <SettingsField label="Core project" value={projects.task_backend.plane.core_project || '—'} />
    {#if projects.task_backend.plane.notes}
      <p class="settings-hint">{projects.task_backend.plane.notes}</p>
    {/if}
  {/if}
</section>

{#if projects.notes?.length}
  <ul class="settings-notes">
    {#each projects.notes as note}<li>{note}</li>{/each}
  </ul>
{/if}
