<script>
  import { taskKey } from './lib/dashboardState.js';
  import TaskCard from './TaskCard.svelte';

  export let tasks = [];
  export let selectedTask = null;
  export let transitions = null;
  export let onOpenTask = () => {};
  export let onTransitionTask = null;
</script>

<section class="task-list" aria-label="Tasks">
  {#each tasks as task (taskKey(task))}
    <TaskCard
      {task}
      {transitions}
      selected={selectedTask && taskKey(selectedTask) === taskKey(task)}
      onOpen={onOpenTask}
      onTransition={onTransitionTask}
    />
  {:else}
    <p class="empty">No tasks match the current filters.</p>
  {/each}
</section>
