<script>
  export let label = '';
  export let value = '';
  export let hint = '';
  export let tone = '';
  export let mono = false;
</script>

<div class="settings-row">
  <span class="settings-key">{label}</span>
  <span class={`settings-val ${tone ? `is-${tone}` : ''}`} class:is-mono={mono}>{value}</span>
</div>
{#if hint}
  <p class="settings-hint">{hint}</p>
{/if}
