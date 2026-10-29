<script lang="ts">
  /** How many `todo` tasks are ready, so a dry run shows what it is holding back. */
  export let readyCount = 0;
  export let onTurnOn: () => void = () => {};
</script>

<section class="dry-run" role="status">
  <p>
    <strong>Dry-run mode.</strong>
    Fleet plans work but starts no agents.
    {#if readyCount > 0}
      {readyCount} task{readyCount === 1 ? '' : 's'} in <code>todo</code>
      {readyCount === 1 ? 'is' : 'are'} ready and would start in live mode.
    {:else}
      Move a task to <code>todo</code> to see what would start.
    {/if}
  </p>
  <button type="button" class="settings-btn" on:click={onTurnOn}>Turn on live mode…</button>
</section>

<style>
  .dry-run {
    flex: 0 0 auto;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-md) var(--space-xl);
    padding: var(--space-md) var(--space-lg);
    border: 1px solid var(--warn-border);
    border-radius: var(--radius-md);
    background: var(--amber-soft);
    color: var(--amber-text);
    font-size: var(--text-md);
  }

  p {
    margin: 0;
  }

  code {
    font-size: var(--text-sm);
  }
</style>
