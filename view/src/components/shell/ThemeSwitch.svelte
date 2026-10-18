<script lang="ts">
  import Icon from './Icon.svelte';

  export let pref = 'system';
  export let onChange: (pref: string) => void = () => {};

  const options = [
    { id: 'light', icon: 'sun', label: 'Light' },
    { id: 'dark', icon: 'moon', label: 'Dark' },
    { id: 'system', icon: 'desktop', label: 'System' }
  ];
</script>

<div class="theme-switch" role="radiogroup" aria-label="Color theme">
  {#each options as option (option.id)}
    <button
      type="button"
      class="theme-switch-btn"
      class:is-active={pref === option.id}
      role="radio"
      aria-checked={pref === option.id}
      title={option.label}
      aria-label={option.label}
      on:click={() => onChange(option.id)}
    >
      <Icon name={option.icon} size={13} />
    </button>
  {/each}
</div>

<style>
  .theme-switch {
    display: inline-flex;
    flex: 0 0 auto;
    gap: 1px;
    margin-left: auto;
    padding: var(--space-2xs);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--surface-muted);
  }

  .theme-switch-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    min-height: 0;
    height: 22px;
    margin: 0;
    padding: 0;
    border: none;
    border-radius: 3px;
    background: transparent;
    color: var(--text-faint);
  }

  .theme-switch-btn:hover {
    background: var(--surface);
    color: var(--text-muted);
  }

  .theme-switch-btn.is-active {
    background: var(--surface);
    color: var(--text);
    box-shadow: 0 0 0 1px var(--border);
  }
</style>
