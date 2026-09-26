<script>
  import SettingsField from './SettingsField.svelte';

  export let diagnostics = {};
  export let inventory = [];

  const tone = (status) =>
    status === 'ok' ? 'ok' : status === 'warn' ? 'warn' : status === 'error' ? 'fail' : 'muted';
</script>

<section class="settings-block" aria-label="System health">
  <h3>System health</h3>
  {#each diagnostics.health || [] as item (item.id)}
    <div class="settings-row">
      <span class="settings-key">
        <span class={`status-dot-inline is-${tone(item.status)}`}></span>
        {item.label}
      </span>
      <span class={`settings-val is-${tone(item.status)}`}>{item.detail || item.status}</span>
    </div>
  {/each}
</section>

<section class="settings-block" aria-label="Session diagnostics">
  <h3>Session diagnostics</h3>
  <SettingsField label="Active" value={String(diagnostics.sessions?.active ?? 0)} />
  <SettingsField label="HITL" value={String(diagnostics.sessions?.hitl ?? 0)} />
  <SettingsField label="Closed" value={String(diagnostics.sessions?.closed ?? 0)} />
  <SettingsField label="Failed" value={String(diagnostics.sessions?.failed ?? 0)} />
  <SettingsField label="Orphaned" value={String(diagnostics.sessions?.orphaned ?? 0)} />
  <SettingsField
    label="Orphaned resumable"
    value={String(diagnostics.sessions?.orphaned_resumable ?? 0)}
  />
</section>

<section class="settings-block" aria-label="Configuration issues">
  <h3>Configuration issues</h3>
  {#each diagnostics.issues || [] as issue (issue.code + issue.message)}
    <p class={`settings-issue is-${issue.severity}`}>{issue.message}</p>
  {:else}
    <p class="settings-hint">No configuration issues reported.</p>
  {/each}
</section>

<section class="settings-block" aria-label="Configuration inventory">
  <h3>Configuration inventory</h3>
  <p class="settings-lede">Where each setting lives today. GUI-editable now only where marked.</p>
  <div class="settings-table-wrap">
    <table class="settings-table">
      <thead>
        <tr>
          <th>Setting</th>
          <th>Source</th>
          <th>Section</th>
          <th>GUI now</th>
        </tr>
      </thead>
      <tbody>
        {#each inventory || [] as row (row.setting)}
          <tr>
            <td>{row.setting}</td>
            <td>{row.current_source}</td>
            <td>{row.section}</td>
            <td>{row.configurable_now ? 'yes' : 'no'}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</section>

{#if diagnostics.notes?.length}
  <ul class="settings-notes">
    {#each diagnostics.notes as note}<li>{note}</li>{/each}
  </ul>
{/if}
