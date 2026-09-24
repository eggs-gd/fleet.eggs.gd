<script>
  import { agentColorClass, assigneeLabel, launchOutcome, priorityLabel, showLaunchSignal, taskAgent, timeAgo } from './lib/taskDisplay.js';
  import { pillClass } from './lib/taskTags.js';
  import { actionsForStatus } from './lib/statusTransitions.js';
  import Icon from './Icon.svelte';
  import StatusActions from './StatusActions.svelte';
  import TypePill from './TypePill.svelte';

  export let task;
  export let selected = false;
  export let transitions = null;
  export let onOpen = () => {};
  export let onTransition = null;

  $: done = task.status === 'done' || task.status === 'archived';
  $: agent = taskAgent(task);
  $: actions = typeof onTransition === 'function' ? actionsForStatus(task.status, transitions) : [];
</script>

<div class="task-row-shell" class:is-selected={selected} class:is-done={done}>
  <button type="button" class="task-row-open" on:click={() => onOpen(task)}>
    <div class="task-row-head">
      <span class="task-row-priority pill pill-priority">{priorityLabel(task)}</span>
      <strong class="task-row-title">
        {#if done}
          <span class="task-row-done-mark" aria-hidden="true"><Icon name="checks-circle" size={16} /></span>
        {/if}
        <span class="task-row-title-text">{task.title}</span>
      </strong>
      <span class="task-row-time">{timeAgo(task.updated_at)}</span>
      <span class="task-row-assignee">
        <span class={`agent-dot ${agentColorClass(agent)}`}></span>
        <span>{assigneeLabel(agent)}</span>
      </span>
    </div>
    <span class="task-row-summary">{task.summary || 'No summary yet.'}</span>
  </button>
  <div class="task-row-footer">
    <button type="button" class="task-row-tags" on:click={() => onOpen(task)}>
      {#if task.status === 'needs_rework'}
        <span class={pillClass('amber')}>rework</span>
      {/if}
      <TypePill type={task.type} />
      <code class="task-row-ref">{task.ref}</code>
      {#if showLaunchSignal(task)}
        <span class={pillClass(launchOutcome(task) === 'waiting' ? 'amber' : 'slate')}>{launchOutcome(task)}</span>
      {/if}
    </button>
    {#if actions.length}
      <StatusActions {task} {transitions} onTransition={onTransition} />
    {/if}
  </div>
</div>
