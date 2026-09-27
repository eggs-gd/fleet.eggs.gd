<script lang="ts">
  import { taskKey } from './lib/dashboardState';
  import TaskCard from './TaskCard.svelte';
  import type { Task } from './lib/types';

  export let tasks: Task[] = [];
  export let selectedTask: Task | null = null;
  export let transitions: Record<string, string[]> | null = null;
  export let onOpenTask: (task: Task) => void = () => {};
  export let onTransitionTask: ((task: Task, status: string) => void) | null = null;
</script>

<section class="task-list" aria-label="Tasks">
  {#each tasks as task (taskKey(task))}
    <TaskCard
      {task}
      {transitions}
      selected={Boolean(selectedTask && taskKey(selectedTask) === taskKey(task))}
      onOpen={onOpenTask}
      onTransition={onTransitionTask}
    />
  {:else}
    <p class="empty">No tasks match the current filters.</p>
  {/each}
</section>

<style>
  .task-list {
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    gap: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    background: var(--surface);
    overflow: hidden;
  }

  .task-list > :global(.empty) {
    margin: 0;
    padding: var(--space-xl) var(--space-lg);
  }
</style>
