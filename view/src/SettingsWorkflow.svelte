<script>
  import SettingsField from './SettingsField.svelte';

  export let workflow = {};
</script>

<section class="settings-block" aria-label="Concurrency">
  <h3>Concurrency</h3>
  <p class="settings-lede">Actual lock scopes from launch policy + session hub. Not a generic concurrency engine.</p>
  {#each workflow.concurrency || [] as scope (scope.scope)}
    <SettingsField label={scope.scope} value={String(scope.limit)} hint={scope.source} />
  {/each}
</section>

<section class="settings-block" aria-label="HITL">
  <h3>HITL</h3>
  <SettingsField label="Task" value={workflow.hitl?.task_status || '—'} />
  <SettingsField label="Session" value={workflow.hitl?.session_status || '—'} />
  <SettingsField label="Releases slot" value={workflow.hitl?.releases_slot ? 'yes' : 'no'} />
  <SettingsField label="Equals closed" value={workflow.hitl?.equals_closed ? 'yes' : 'no'} />
  <SettingsField label="Source" value={workflow.hitl?.source || '—'} />
  <SettingsField label="If Finalizer published" value={workflow.hitl?.finalizer_if_published || '—'} />
</section>

<section class="settings-block" aria-label="Runtime outcomes">
  <h3>Runtime outcomes</h3>
  <p class="settings-lede">Generated from live host + taskflow tables. Read-only.</p>
  <div class="settings-table-wrap">
    <table class="settings-table">
      <thead>
        <tr>
          <th>Outcome</th>
          <th>Task</th>
          <th>Session</th>
          <th>Releases slot</th>
        </tr>
      </thead>
      <tbody>
        {#each workflow.runtime_outcomes || [] as row (row.outcome)}
          <tr>
            <td>{row.outcome}</td>
            <td>{row.task}</td>
            <td>{row.session}</td>
            <td>{row.releases_slot ? 'yes' : 'no'}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</section>

<section class="settings-block" aria-label="Operator transitions">
  <h3>Operator transitions</h3>
  {#each Object.entries(workflow.operator_transitions || {}) as [from, next] (from)}
    <SettingsField label={from} value={(next || []).join(' → ') || '—'} />
  {/each}
</section>

{#if workflow.notes?.length}
  <ul class="settings-notes">
    {#each workflow.notes as note}<li>{note}</li>{/each}
  </ul>
{/if}
