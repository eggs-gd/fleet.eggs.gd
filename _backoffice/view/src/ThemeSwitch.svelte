<script>
  import Icon from './Icon.svelte';

  export let pref = 'system';
  export let onChange = () => {};

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
