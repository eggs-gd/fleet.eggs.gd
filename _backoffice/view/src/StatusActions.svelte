<script>
  import { actionsForStatus } from './lib/statusTransitions.js';
  import Icon from './Icon.svelte';

  export let task;
  export let compact = false;
  export let transitions = null;
  export let onTransition = () => {};

  $: actions = actionsForStatus(task?.status, transitions);

  function run(event, status) {
    event.preventDefault();
    event.stopPropagation();
    onTransition(task, status);
  }
</script>

{#if actions.length}
  <div class="status-actions" class:is-compact={compact} role="group" aria-label="Status actions">
    {#each actions as action (action.status)}
      <button
        type="button"
        class={`status-action status-action--${action.status}`}
        class:is-compact={compact}
        title={action.title}
        aria-label={action.label}
        on:click={(event) => run(event, action.status)}
      >
        <Icon name={action.icon} size={compact ? 13 : 12} />
        {#if !compact}
          <span>{action.label}</span>
        {/if}
      </button>
    {/each}
  </div>
{/if}
