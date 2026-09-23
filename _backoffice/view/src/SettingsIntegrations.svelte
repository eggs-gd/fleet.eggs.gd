<script>
  import SettingsField from './SettingsField.svelte';

  export let integrations = {};
</script>

<section class="settings-block" aria-label="MCP servers">
  <h3>MCP servers</h3>
  <SettingsField label="Source file" value={integrations.source_file || '—'} mono />
  <p class="settings-hint">{integrations.source_role || ''}</p>
  {#each integrations.mcp || [] as server (server.name)}
    <article class="settings-item">
      <div class="settings-item-head">
        <span class={`status-dot-inline is-${server.detected ? 'ok' : server.configured ? 'warn' : 'muted'}`}></span>
        <strong>{server.name}</strong>
        <span class="settings-pill">{server.detected ? 'detected' : 'not detected'}</span>
      </div>
      <SettingsField label="Configured" value={server.configured ? 'yes' : 'no'} />
      <SettingsField label="Detected" value={server.detected ? 'yes' : 'no'} />
      <SettingsField label="Reachable" value={server.reachable || 'unknown'} />
      <SettingsField label="Transport" value={server.transport || '—'} />
      <SettingsField label="Command" value={server.command || '—'} mono />
      {#if server.expected}
        <SettingsField label="Expected" value={server.expected} mono />
      {/if}
      {#if server.error}
        <p class="settings-warn">{server.error}</p>
      {/if}
      {#if !server.install_known}
        <p class="settings-hint">No Install button — Core has no deterministic installer for this server.</p>
      {/if}
    </article>
  {:else}
    <p class="settings-hint">No MCP servers in .mcp.json.</p>
  {/each}
</section>

{#if integrations.notes?.length}
  <ul class="settings-notes">
    {#each integrations.notes as note}<li>{note}</li>{/each}
  </ul>
{/if}
