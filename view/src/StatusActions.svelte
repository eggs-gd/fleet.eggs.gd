<script lang="ts">
  import { actionsForStatus } from './lib/statusTransitions';
  import Icon from './Icon.svelte';
  import type { Task } from './lib/types';

  export let task: Task;
  export let compact = false;
  export let transitions: Record<string, string[]> | null = null;
  export let onTransition: (task: Task, status: string) => void = () => {};

  $: actions = actionsForStatus(task?.status || '', transitions);

  function run(event: Event, status: string) {
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

<style>
  .status-actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: var(--space-xs);
  }

  .status-action {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-xs);
    margin: 0;
    min-height: 0;
    height: 22px;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--surface);
    color: var(--text-muted);
    font-size: var(--text-xs);
    font-weight: 600;
    line-height: 1;
    padding: 0 var(--space-sm);
    cursor: pointer;
  }

  .status-action.is-compact {
    width: 22px;
    padding: 0;
  }

  .status-action:hover,
  .status-action:focus-visible {
    color: var(--text);
    outline: none;
  }

  .status-action--done {
    color: var(--green);
  }

  .status-action--todo,
  .status-action--backlog {
    color: var(--accent);
  }

  .status-action--needs_rework {
    color: var(--amber-text);
  }

  .status-action--archived {
    color: var(--text-muted);
  }
</style>
