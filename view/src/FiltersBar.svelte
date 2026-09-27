<script lang="ts">
  import Icon from './Icon.svelte';

  interface Chip {
    id: string;
    label: string;
    count: number;
  }

  export let chips: Chip[] = [];
  export let activeChip = 'all';
  export let query = '';
  export let sort = 'updated';
  export let onChipChange: (chipId: string) => void = () => {};
  export let onSortChange: (value: string) => void = () => {};
</script>

<section class="task-toolbar" aria-label="Task toolbar">
  <div class="chip-row" role="tablist">
    {#each chips as chip (chip.id)}
      <button
        type="button"
        class="chip"
        class:active={activeChip === chip.id}
        class:chip-high={chip.id === 'high'}
        role="tab"
        aria-selected={activeChip === chip.id}
        on:click={() => onChipChange(chip.id)}
      >
        {chip.label}
        <span class="chip-count">{chip.count}</span>
      </button>
    {/each}
  </div>

  <div class="task-toolbar-tools">
    <label class="search-field">
      <Icon name="search" size={15} />
      <input bind:value={query} placeholder="Search tasks…" />
    </label>
    <label class="sort-field">
      <span class="visually-hidden">Sort</span>
      <select value={sort} on:change={(event) => onSortChange(event.currentTarget.value)}>
        <option value="updated">Updated ↓</option>
        <option value="priority">Priority</option>
      </select>
    </label>
  </div>
</section>

<style>
  .task-toolbar {
    flex: 0 0 auto;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-lg);
  }

  .chip-row {
    display: flex;
    flex: 1 1 auto;
    flex-wrap: wrap;
    min-width: 200px;
    gap: var(--space-sm);
  }

  .chip {
    display: inline-flex;
    align-items: center;
    gap: var(--space-sm);
    min-height: 30px;
    border-color: var(--border);
    border-radius: 6px;
    background: var(--surface);
    color: var(--text-muted);
    font-size: var(--text-sm);
    font-weight: 600;
    padding: 0 var(--space-lg);
  }

  .chip:hover {
    background: var(--surface-muted);
  }

  .chip.active {
    border-color: var(--accent);
    background: var(--accent-soft);
    color: var(--accent-soft-text);
  }

  .chip-count {
    border-radius: var(--radius-full);
    background: var(--chip-count-bg);
    font-size: var(--text-xs);
    padding: 0px var(--space-sm);
  }

  .chip.active .chip-count {
    background: var(--chip-count-active-bg);
  }

  .chip.chip-high .chip-count {
    background: var(--red-soft);
    color: var(--red-text);
  }

  .task-toolbar-tools {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    flex: 0 0 auto;
  }

  .search-field {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    width: 220px;
    flex: 0 0 auto;
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-sm);
    background: var(--surface);
    padding: 0 var(--space-lg);
    color: var(--text-faint);
  }

  .sort-field select {
    min-height: 34px;
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-sm);
    background: var(--surface);
    color: var(--text-muted);
    font-size: var(--text-sm);
    font-weight: 600;
  }

  .search-field input {
    border: none;
    padding: 0;
    min-height: 34px;
  }

  .search-field input:focus {
    outline: none;
  }
</style>
