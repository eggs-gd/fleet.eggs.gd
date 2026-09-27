<script lang="ts">
  import SettingsField from './SettingsField.svelte';
  import type { AnyRecord } from './lib/types';

  export let projects: AnyRecord = {};
  export let scanRoots: string[] = [];
  export let scanDirty = false;
  export let onScanRoots: (roots: string[]) => void = () => {};
  export let onRescan: () => void = () => {};

  const reachable = (root: AnyRecord) => (root.reachable ? 'reachable' : 'missing');
  $: scan = projects.scan || {};
  $: scanning = scan.state === 'scanning';

  function editRoot(index: number, value: string) {
    const next = scanRoots.slice();
    next[index] = value;
    onScanRoots(next);
  }

  function removeRoot(index: number) {
    onScanRoots(scanRoots.filter((_, i) => i !== index));
  }

  function addRoot() {
    onScanRoots([...scanRoots, '']);
  }
</script>

<section class="settings-block" aria-label="Project scan roots">
  <h3>Project scan roots</h3>
  <p class="settings-lede">
    Each path is a directory of git checkouts. Save, then the running server watches all of them and
    refreshes the registry when repositories appear or disappear. Rescan runs a pass now. Recheck is
    for agents, not this tree. Per-project repositories and PROJECT.md open from the Settings tab on
    that project.
  </p>
  {#each scanRoots as root, index (index)}
    <div class="settings-row">
      <span class="settings-key">Root {index + 1}</span>
      <span class="settings-val">
        <input
          class="settings-input is-mono"
          type="text"
          value={root}
          on:input={(event) => editRoot(index, event.currentTarget.value)}
        />
        <button type="button" class="settings-btn" on:click={() => removeRoot(index)}>Remove</button
        >
      </span>
    </div>
  {/each}
  <div class="settings-row">
    <span class="settings-key">Add</span>
    <span class="settings-val">
      <button type="button" class="settings-btn" on:click={addRoot}>Add root</button>
    </span>
  </div>
  {#if projects.scan_roots_source}
    <p class="settings-hint">{projects.scan_roots_source}</p>
  {/if}
  <div class="settings-row">
    <span class="settings-key">Rescan</span>
    <span class="settings-val">
      <button
        type="button"
        class="settings-btn"
        disabled={scanning || scanDirty}
        on:click={onRescan}
      >
        {scanning ? 'Scanning…' : 'Rescan'}
      </button>
      <span class="settings-item-meta"
        >{scan.state || 'idle'}{scan.finished_at ? ` · ${scan.finished_at}` : ''}</span
      >
    </span>
  </div>
  {#if scanDirty}
    <p class="settings-hint">Save the scan roots before Rescan.</p>
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

<section class="settings-block" aria-label="Fleet data">
  <h3>Fleet data</h3>
  {#each projects.data_paths || [] as path (path.name)}
    <article class="settings-item">
      <div class="settings-item-head">
        <span
          class={`status-dot-inline is-${path.exists && path.writable ? 'ok' : path.exists ? 'warn' : 'fail'}`}
        ></span>
        <strong>{path.name}</strong>
        <span class="settings-pill"
          >{path.exists ? (path.writable ? 'writable' : 'not writable') : 'missing'}</span
        >
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
</section>

{#if projects.notes?.length}
  <ul class="settings-notes">
    {#each projects.notes as note}<li>{note}</li>{/each}
  </ul>
{/if}
