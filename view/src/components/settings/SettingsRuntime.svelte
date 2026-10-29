<script lang="ts">
  import SettingsField from './SettingsField.svelte';
  import SettingsNotes from './SettingsNotes.svelte';
  import type { AnyRecord } from '../../lib/types';

  export let general: AnyRecord = {};
  export let projects: AnyRecord = {};

  const reachable = (root: AnyRecord) => (root.reachable ? 'reachable' : 'missing');
</script>

<section class="settings-block" aria-label="About this Fleet">
  <h3>About this Fleet</h3>
  <SettingsField label="Version" value={general.version || '—'} />
  <SettingsField label="Build" value={general.build_hash || '—'} mono />
  <SettingsField label="Mode" value={general.launch_mode || '—'} />
  <SettingsField label="Started at" value={general.started_at || '—'} />
  <SettingsField label="Uptime" value={general.uptime || '—'} />
  <SettingsField label="Address" value={general.api_endpoint || '—'} mono />
</section>

<section class="settings-block" aria-label="Folders Fleet reads">
  <h3>Folders Fleet reads</h3>
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

<section class="settings-block" aria-label="Where Fleet keeps its data">
  <h3>Where Fleet keeps its data</h3>
  <p class="settings-lede">
    Files marked missing are optional: Fleet creates them when it first needs them.
  </p>
  {#each projects.data_paths || [] as path (path.name)}
    <article class="settings-item">
      <div class="settings-item-head">
        <span
          class={`status-dot-inline is-${path.exists && path.writable ? 'ok' : path.exists ? 'warn' : 'muted'}`}
        ></span>
        <strong>{path.name}</strong>
        <span class="settings-pill"
          >{path.exists ? (path.writable ? 'writable' : 'not writable') : 'not created yet'}</span
        >
      </div>
      <code class="settings-path">{path.path}</code>
      <p class="settings-hint">{path.role}</p>
    </article>
  {/each}
</section>

<section class="settings-block" aria-label="Task storage">
  <h3>Task storage</h3>
  <SettingsField label="Backend" value={projects.task_backend?.active || '—'} />
  <SettingsField label="Config file" value={projects.task_backend?.config_file || '—'} mono />
  <SettingsField
    label="Config file exists"
    value={projects.task_backend?.config_file_exists ? 'yes' : 'no'}
  />
</section>

<section class="settings-block" aria-label="Startup">
  <h3>Startup</h3>
  <p class="settings-lede">What Fleet does each time it starts. These are not settings.</p>
  {#each general.startup || [] as fact (fact.id)}
    <SettingsField label={fact.label} value={fact.value} />
  {/each}
</section>

<SettingsNotes notes={[...(general.notes || []), ...(projects.notes || [])]} />
