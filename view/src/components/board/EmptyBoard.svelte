<script lang="ts">
  import { addScanRoot } from '../../lib/scanRoots';
  import { requestManualRefresh } from '../../stores/board';

  let path = '';
  let adding = false;
  let error = '';
  let warnings: string[] = [];

  async function add() {
    if (adding) return;
    adding = true;
    error = '';
    warnings = [];
    try {
      warnings = await addScanRoot(path);
      path = '';
      await requestManualRefresh();
    } catch (err: any) {
      error = err.message || String(err);
    } finally {
      adding = false;
    }
  }
</script>

<section class="empty-board" aria-labelledby="empty-title">
  <h2 id="empty-title">Add the folder that holds your projects</h2>
  <p>
    Fleet finds projects by scanning a folder of git repositories, for example
    <code>~/Projects</code>. Once a project shows up here you can create tasks for it.
  </p>
  <form on:submit|preventDefault={add}>
    <label>
      <span>Folder path</span>
      <input
        class="settings-input"
        type="text"
        bind:value={path}
        placeholder="/Users/you/Projects"
        spellcheck="false"
        disabled={adding}
      />
    </label>
    <button type="submit" class="settings-btn is-primary" disabled={adding || !path.trim()}>
      {adding ? 'Scanning…' : 'Add folder'}
    </button>
  </form>
  {#if error}
    <p class="settings-error">{error}</p>
  {/if}
  {#each warnings as warning}
    <p class="settings-hint">{warning}</p>
  {/each}
</section>

<style>
  .empty-board {
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    background: var(--surface);
    padding: var(--space-2xl);
    display: flex;
    flex-direction: column;
    gap: var(--space-lg);
  }

  h2 {
    margin: 0;
    font-size: var(--text-xl);
  }

  p {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--text-md);
  }

  form {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: var(--space-md);
  }

  label {
    flex: 1 1 320px;
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
    font-size: var(--text-sm);
    color: var(--text-muted);
  }
</style>
