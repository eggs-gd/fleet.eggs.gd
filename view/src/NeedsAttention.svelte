<script>
  import { blockedReason, launchReason, operatorPause, priorityLabel, timeAgo, agentColorClass, assigneeLabel } from './lib/taskDisplay.js';
  import { taskKey } from './lib/dashboardState.js';
  import Icon from './Icon.svelte';
  import StatusActions from './StatusActions.svelte';
  import TypePill from './TypePill.svelte';

  export let tasks = [];
  export let open = true;
  export let transitions = null;
  export let onToggle = () => {};
  export let onOpenTask = () => {};
  export let onShowAll = () => {};
  export let onTransitionTask = null;

  $: items = [...tasks].sort((a, b) => {
    const rank = (task) => (operatorPause(task).waiting ? 0 : task.status === 'blocked' ? 1 : 2);
    const diff = rank(a) - rank(b);
    if (diff) return diff;
    return (b.updated_at || '').localeCompare(a.updated_at || '');
  });
</script>

<section class="status-panel status-panel--attention" class:is-collapsed={!open} aria-label="Needs attention">
  <header>
    <button type="button" class="panel-toggle" aria-expanded={open} on:click={onToggle}>
      <h3>Needs Attention</h3>
      <span>{items.length}</span>
    </button>
    <button type="button" class="panel-show-all" on:click={onShowAll}>Show all</button>
  </header>
  {#if open}
    <div class="attention-list">
      {#each items as task (taskKey(task))}
        {@const pause = operatorPause(task)}
        {@const summary = (pause.waiting && pause.question) || blockedReason(task) || launchReason(task) || task.summary}
        <div
          class="attention-item"
          class:attention-item--waiting={pause.waiting}
          class:attention-item--blocked={task.status === 'blocked'}
          class:attention-item--review={task.status === 'needs_review'}
        >
          <button type="button" class="attention-item-open" on:click={() => onOpenTask(task)}>
            <div class="attention-item-head">
              <strong class="attention-item-title">
                {#if pause.waiting}
                  <span class="attention-mark is-waiting" aria-hidden="true">
                    <Icon name="help-circle" size={14} />
                  </span>
                {:else if task.status === 'blocked'}
                  <span class="attention-mark is-blocked" aria-hidden="true">
                    <Icon name="x-circle" size={14} />
                  </span>
                {:else if task.status === 'needs_review'}
                  <span class="attention-mark is-review" aria-hidden="true">
                    <Icon name="check-circle" size={14} />
                  </span>
                {/if}
                <span class="attention-item-title-text">{task.title}</span>
              </strong>
              <span class="attention-agent">
                <span class={`agent-dot ${agentColorClass(task.assignee)}`}></span>
                <span class="attention-agent-name">{assigneeLabel(task.assignee)}</span>
              </span>
            </div>
            {#if summary}
              <span class="attention-item-summary">{summary}</span>
            {/if}
          </button>
          <div class="attention-item-footer">
            <button type="button" class="attention-item-meta" on:click={() => onOpenTask(task)}>
              <span class="pill pill-priority">{priorityLabel(task)}</span>
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
        <p class="empty">Nothing needs attention.</p>
      {/each}
    </div>
  {/if}
</section>
