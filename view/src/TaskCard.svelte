<script lang="ts">
  import {
    agentColorClass,
    assigneeLabel,
    cardStatusLabel,
    launchOutcome,
    operatorPause,
    priorityLabel,
    showLaunchSignal,
    taskAgent,
    timeAgo
  } from './lib/taskDisplay';
  import { pillClass } from './lib/taskTags';
  import { actionsForStatus } from './lib/statusTransitions';
  import Icon from './Icon.svelte';
  import StatusActions from './StatusActions.svelte';
  import TypePill from './TypePill.svelte';
  import type { Task } from './lib/types';

  export let task: Task;
  export let selected = false;
  export let transitions: Record<string, string[]> | null = null;
  export let onOpen: (task: Task) => void = () => {};
  export let onTransition: ((task: Task, status: string) => void) | null = null;

  $: done = task.status === 'done' || task.status === 'archived';
  $: agent = taskAgent(task);
  $: actions =
    typeof onTransition === 'function' ? actionsForStatus(task.status || '', transitions) : [];
  $: pause = operatorPause(task);
  $: statusLabel = cardStatusLabel(task);
</script>

<div class="task-row-shell" class:is-selected={selected} class:is-done={done}>
  <button type="button" class="task-row-open" on:click={() => onOpen(task)}>
    <div class="task-row-head">
      <span class="task-row-priority pill pill-priority">{priorityLabel(task)}</span>
      <strong class="task-row-title">
        {#if done}
          <span class="task-row-done-mark" aria-hidden="true"
            ><Icon name="checks-circle" size={16} /></span
          >
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
    {#if pause.waiting && pause.question}
      <span class="task-row-summary">{pause.question}</span>
    {/if}
  </button>
  <div class="task-row-footer">
    <button type="button" class="task-row-tags" on:click={() => onOpen(task)}>
      {#if statusLabel}
        <span class={pillClass(pause.waiting || task.status === 'needs_rework' ? 'amber' : 'slate')}
          >{statusLabel}</span
        >
      {/if}
      <TypePill type={task.type} />
      <code class="task-row-ref">{task.ref}</code>
      {#if showLaunchSignal(task)}
        <span class={pillClass(launchOutcome(task) === 'waiting' ? 'amber' : 'slate')}
          >{launchOutcome(task)}</span
        >
      {/if}
    </button>
    {#if actions.length}
      <StatusActions {task} {transitions} onTransition={onTransition ?? (() => {})} />
    {/if}
  </div>
</div>

<style>
  .task-row-shell {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
    min-width: 0;
    border: none;
    border-bottom: 1px solid var(--border);
    border-radius: 0;
    background: transparent;
    padding: var(--space-lg) var(--space-lg);
  }

  .task-row-shell:last-child {
    border-bottom: none;
  }

  .task-row-shell:hover {
    background: var(--surface-muted);
  }

  .task-row-shell.is-selected {
    background: var(--accent-soft);
    box-shadow: inset 2px 0 0 var(--accent);
  }

  .task-row-shell.is-done {
    background: transparent;
  }

  .task-row-shell.is-done:hover {
    background: var(--surface-muted);
  }

  .task-row-shell.is-done.is-selected,
  .task-row-shell.is-selected:hover {
    background: var(--accent-soft);
  }

  .task-row-open {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: var(--space-xs);
    width: 100%;
    min-width: 0;
    min-height: 0;
    margin: 0;
    padding: 0;
    border: none;
    border-radius: 0;
    background: transparent;
    text-align: left;
    cursor: pointer;
    color: inherit;
  }

  .task-row-head {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    min-width: 0;
  }

  .task-row-priority {
    flex: 0 0 auto;
  }

  .task-row-title {
    display: inline-flex;
    align-items: center;
    gap: var(--space-sm);
    flex: 1 1 auto;
    min-width: 0;
    font-size: var(--text-md);
    line-height: 1.35;
  }

  .task-row-done-mark {
    flex: 0 0 auto;
    display: inline-flex;
    color: var(--green);
  }

  .task-row-title-text {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .task-row-shell.is-done .task-row-title-text,
  .task-row-shell.is-done .task-row-summary,
  .task-row-shell.is-done .task-row-assignee,
  .task-row-shell.is-done .task-row-time {
    color: var(--text-muted);
  }

  .task-row-time {
    flex: 0 0 auto;
    color: var(--text-faint);
    font-size: var(--text-xs);
    white-space: nowrap;
  }

  .task-row-ref {
    color: var(--text-faint);
    font-size: var(--text-xs);
  }

  .task-row-summary {
    display: block;
    width: 100%;
    min-width: 0;
    overflow: hidden;
    color: var(--text-muted);
    font-size: var(--text-sm);
    line-height: 1.35;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .task-row-footer {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    min-width: 0;
  }

  .task-row-tags {
    flex: 1 1 auto;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-sm);
    min-width: 0;
    margin: 0;
    padding: 0;
    border: none;
    border-radius: 0;
    background: transparent;
    text-align: left;
    cursor: pointer;
    color: inherit;
  }

  .task-row-assignee {
    flex: 0 0 auto;
    display: inline-flex;
    align-items: center;
    gap: var(--space-sm);
    max-width: 6.5rem;
    min-width: 0;
    overflow: hidden;
    color: var(--text-muted);
    font-size: var(--text-sm);
    white-space: nowrap;
  }

  .task-row-assignee span:last-child {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .task-row-shell :global(.status-actions) {
    position: static;
    flex: 0 0 auto;
    flex-wrap: nowrap;
  }
</style>
