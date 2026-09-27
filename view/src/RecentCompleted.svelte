<script lang="ts">
  import { agentColorClass, timeAgo, assigneeLabel } from './lib/taskDisplay';
  import { taskKey } from './lib/dashboardState';
  import Icon from './Icon.svelte';
  import StatusActions from './StatusActions.svelte';
  import TypePill from './TypePill.svelte';
  import type { Task } from './lib/types';

  export let tasks: Task[] = [];
  export let open = true;
  export let transitions: Record<string, string[]> | null = null;
  export let onToggle: () => void = () => {};
  export let onOpenTask: (task: Task) => void = () => {};
  export let onShowAll: () => void = () => {};
  export let onTransitionTask: ((task: Task, status: string) => void) | null = null;
  export let limit = 14;

  $: recent = [...tasks]
    .filter((task) => task.status === 'done')
    .sort((a, b) => (b.updated_at || '').localeCompare(a.updated_at || ''))
    .slice(0, limit);
  $: totalDone = tasks.filter((task) => task.status === 'done').length;
</script>

<section class="recent-completed" class:is-collapsed={!open} aria-label="Recently completed">
  <header>
    <button type="button" class="panel-toggle" aria-expanded={open} on:click={onToggle}>
      <h3>Recent Completed</h3>
      <span>{totalDone}</span>
    </button>
    <button type="button" class="panel-show-all" on:click={onShowAll}>Show all</button>
  </header>
  {#if open}
    <div class="recent-completed-list">
      {#each recent as task (taskKey(task))}
        <div class="recent-item">
          <button type="button" class="attention-item-open" on:click={() => onOpenTask(task)}>
            <div class="attention-item-head">
              <strong class="attention-item-title">
                <span class="attention-mark is-done" aria-hidden="true">
                  <Icon name="checks-circle" size={14} />
                </span>
                <span class="attention-item-title-text">{task.title}</span>
              </strong>
              <span class="attention-agent">
                <span class={`agent-dot ${agentColorClass(task.assignee)}`}></span>
                <span class="attention-agent-name">{assigneeLabel(task.assignee)}</span>
              </span>
            </div>
            {#if task.summary}
              <span class="attention-item-summary">{task.summary}</span>
            {/if}
          </button>
          <div class="attention-item-footer">
            <button type="button" class="attention-item-meta" on:click={() => onOpenTask(task)}>
              <TypePill type={task.type} />
              <code>{task.ref}</code>
              <span>{timeAgo(task.updated_at)}</span>
            </button>
            {#if onTransitionTask}
              <StatusActions {task} {transitions} compact onTransition={onTransitionTask} />
            {/if}
          </div>
        </div>
      {:else}
        <p class="empty">Nothing completed yet.</p>
      {/each}
    </div>
  {/if}
</section>
