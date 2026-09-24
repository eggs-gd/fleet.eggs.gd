<script>
  import Icon from './Icon.svelte';

  export let chips = [];
  export let activeChip = 'all';
  export let query = '';
  export let sort = 'updated';
  export let onChipChange = () => {};
  export let onSortChange = () => {};
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
