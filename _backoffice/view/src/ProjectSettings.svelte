<script>
  import SettingsField from './SettingsField.svelte';
  import { projectSettingsInspect } from './lib/projectSettings.js';

  export let project = null;
  export let workspaces = [];
  export let projects = [];
  export let repositories = [];

  $: inspect = projectSettingsInspect(project, { workspaces, projects, repositories });
  $: showNestCols = (inspect?.repositories || []).some((row) => row.nestedUnder || row.nests.length);
</script>

{#if inspect}
  <section class="settings-block" aria-label="Project identity">
    <h3>Identity</h3>
    <p class="settings-lede">Inspect-only facts from the board snapshot. Title, path, and tech stay in the header.</p>
    <SettingsField label="Kind" value={inspect.kind || '—'} />
    <SettingsField label="Source" value={inspect.source || '—'} />
    {#if inspect.status}
      <SettingsField label="Discovered" value={inspect.status} />
    {/if}
    {#if inspect.reviewStatus}
      <SettingsField label="Review" value={inspect.reviewStatus} />
    {/if}
    <SettingsField label="PROJECT.md" value={inspect.projectMd || '—'} mono />
    {#if inspect.membership}
      <SettingsField label="Workspace" value={`${inspect.membership.title} · ${inspect.membership.id}`} />
    {/if}
    {#if inspect.parent}
      <SettingsField label="Parent" value={`${inspect.parent.title} · ${inspect.parent.id}`} />
    {/if}
    {#if inspect.summary}
      <p class="settings-lede">{inspect.summary}</p>
    {/if}
  </section>

  {#if inspect.nested.length}
    <section class="settings-block" aria-label="Nested projects">
      <h3>Nested projects</h3>
      <ul class="settings-notes">
        {#each inspect.nested as child (child.id)}
          <li>{child.title} · {child.id}</li>
        {/each}
      </ul>
    </section>
  {/if}

  <section class="settings-block" aria-label="Repositories">
    <h3>Repositories</h3>
    {#if inspect.repositories.length}
      <div class="settings-table-wrap">
        <table class="settings-table">
          <thead>
            <tr>
              <th>Path</th>
              <th>Branch</th>
              <th>Remote</th>
              {#if showNestCols}
                <th>Nested under</th>
                <th>Nests</th>
              {/if}
            </tr>
          </thead>
          <tbody>
            {#each inspect.repositories as row (row.id || row.relativePath)}
              <tr>
                <td class="settings-val is-mono">{row.relativePath || '—'}</td>
                <td>{row.branch || '—'}</td>
                <td>
                  {#if row.webUrl}
                    <a href={row.webUrl} target="_blank" rel="noreferrer">{row.remote}</a>
                  {:else}
                    {row.remote || '—'}
                  {/if}
                </td>
                {#if showNestCols}
                  <td class="settings-val is-mono">{row.nestedUnder || '—'}</td>
                  <td class="settings-val is-mono">{row.nests.join(', ') || '—'}</td>
                {/if}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else}
      <p class="settings-hint">No repositories on this card.</p>
    {/if}
  </section>
{/if}
