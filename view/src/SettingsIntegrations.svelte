<script>
  import SettingsField from './SettingsField.svelte';

  export let integrations = {};
</script>

<section class="settings-block" aria-label="Fleet MCP">
  <h3>Fleet MCP</h3>
  {#if integrations.fleet_mcp?.provider}
    <SettingsField label="Provider" value={integrations.fleet_mcp.provider} />
    <SettingsField label="File" value={integrations.fleet_mcp.path || '—'} mono />
    <SettingsField label="URL" value={integrations.fleet_mcp.url || 'not written'} mono />
    <SettingsField
      label="Matches this process"
      value={integrations.fleet_mcp.matches ? 'yes' : 'no'}
    />
    {#if integrations.fleet_mcp.trusted != null}
      <SettingsField
        label="Codex trust"
        value={integrations.fleet_mcp.trusted ? 'trusted' : 'not trusted'}
      />
      <p class="settings-hint">
        Trusting this folder also allows Codex hooks and exec policy, not only the manager MCP
        server.
      </p>
    {/if}
  {:else}
    <p class="settings-hint">
      No Manager session is recorded, so Fleet MCP is not checked against a provider file.
    </p>
  {/if}
</section>

<section class="settings-block" aria-label="MCP servers">
  <h3>Data root .mcp.json</h3>
  <SettingsField label="Source file" value={integrations.source_file || '—'} mono />
  <p class="settings-hint">{integrations.source_role || ''}</p>
  {#each integrations.mcp || [] as server (server.name)}
    <article class="settings-item">
      <div class="settings-item-head">
        <span
          class={`status-dot-inline is-${server.detected ? 'ok' : server.configured ? 'warn' : 'muted'}`}
        ></span>
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
        <p class="settings-hint">
          No Install button — Fleet has no deterministic installer for this server.
        </p>
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
