<script lang="ts">
  import {
    isAccordionOpen,
    milestoneAccordionKey,
    taskKey,
    updateAccordionOpen
  } from '../../lib/dashboardState';
  import {
    groupedByMilestone,
    shouldGroupByMilestone,
    milestoneGroupStatusCounts,
    taskProjectId
  } from '../../lib/taskDisplay';
  import Icon from '../shell/Icon.svelte';
  import TaskCard from '../tasks/TaskCard.svelte';
  import TaskList from '../tasks/TaskList.svelte';
  import type { Project, Task, Workspace } from '../../lib/types';

  export let tasks: Task[] = [];
  export let projects: Project[] = [];
  export let workspaces: Workspace[] = [];
  export let selectedProjectId = '';
  export let selectedWorkspaceId = '';
  export let selectedTask: Task | null = null;
  export let accordionOpen: Record<string, boolean> = {};
  export let transitions: Record<string, string[]> | null = null;
  export let onOpenTask: (task: Task) => void = () => {};
  export let onTransitionTask: ((task: Task, status: string) => void) | null = null;
  export let onDropTaskOnMilestone: ((taskPath: string, milestoneId: string) => void) | null = null;

  $: grouped = shouldGroupByMilestone(tasks, projects, selectedProjectId);
  $: groups = grouped
    ? groupedByMilestone(tasks, projects, workspaces, selectedProjectId, selectedWorkspaceId)
    : [];

  let dragOverKey = '';

  function toggle(key: string, currentlyOpen: boolean) {
    accordionOpen = updateAccordionOpen(accordionOpen, key, !currentlyOpen);
  }

  function statusSummary(groupTasks: Task[]): string {
    return milestoneGroupStatusCounts(groupTasks)
      .map((entry) => `${entry.count} ${entry.status}`)
      .join(' · ');
  }

  function handleDragStart(event: DragEvent, task: Task) {
    event.dataTransfer?.setData('text/task-path', task.path || task.relative_path || '');
    event.dataTransfer?.setData('text/task-project', taskProjectId(task));
    if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
  }

  function handleDragOver(event: DragEvent, key: string) {
    if (!onDropTaskOnMilestone) return;
    event.preventDefault();
    dragOverKey = key;
  }

  // A milestone belongs to one project, so a card only drops onto a group of
  // its own project; anything else is ignored instead of writing a foreign id.
  function handleDrop(event: DragEvent, group: { projectId: string; milestoneId: string }) {
    if (!onDropTaskOnMilestone) return;
    event.preventDefault();
    dragOverKey = '';
    const path = event.dataTransfer?.getData('text/task-path');
    const project = event.dataTransfer?.getData('text/task-project');
    if (path && project === group.projectId) onDropTaskOnMilestone(path, group.milestoneId);
  }
</script>

{#if !grouped}
  <TaskList {tasks} {selectedTask} {transitions} {onOpenTask} {onTransitionTask} />
{:else}
  <div class="milestone-list" role="list">
    {#each groups as group (group.key)}
      {@const key = milestoneAccordionKey(group.projectId, group.milestoneId)}
      {@const open = isAccordionOpen(accordionOpen, key, true)}
      <section class="milestone-group" class:is-drag-over={dragOverKey === key}>
        <button
          type="button"
          class="milestone-toggle"
          aria-expanded={open}
          on:click={() => toggle(key, open)}
          on:dragover={(event) => handleDragOver(event, key)}
          on:dragleave={() => (dragOverKey = '')}
          on:drop={(event) => handleDrop(event, group)}
        >
          <strong>{group.title}</strong>
          <span class="milestone-summary">{statusSummary(group.tasks)}</span>
          <span class="milestone-count">{group.tasks.length}</span>
          <span class="tree-toggle" class:is-open={open} aria-hidden="true">
            <Icon name="chevron-right" size={12} />
          </span>
        </button>
        {#if open}
          {#each group.tasks as task (taskKey(task))}
            <TaskCard
              {task}
              {transitions}
              selected={Boolean(selectedTask && taskKey(selectedTask) === taskKey(task))}
              onOpen={onOpenTask}
              onTransition={onTransitionTask}
              draggable={Boolean(onDropTaskOnMilestone)}
              onDragStart={handleDragStart}
            />
          {/each}
        {/if}
      </section>
    {/each}
  </div>
{/if}

<style>
  .milestone-list {
    display: grid;
    gap: var(--space-lg);
  }

  .milestone-group {
    display: grid;
    gap: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    overflow: hidden;
    background: var(--surface);
  }

  .milestone-group.is-drag-over {
    border-color: var(--accent);
    box-shadow: inset 0 0 0 1px var(--accent);
  }

  .milestone-toggle {
    display: flex;
    align-items: center;
    gap: var(--space-lg);
    width: 100%;
    min-height: 0;
    border: none;
    border-bottom: 1px solid var(--border);
    background: var(--surface-muted);
    color: var(--text);
    padding: var(--space-sm) var(--space-md);
    text-align: left;
    cursor: pointer;
  }

  .milestone-toggle:hover {
    background: var(--surface-muted);
    filter: brightness(0.98);
  }

  .milestone-toggle strong {
    min-width: 0;
    flex: 1 1 auto;
    font-size: var(--text-sm);
    line-height: 1.2;
  }

  .milestone-summary {
    flex: 0 0 auto;
    color: var(--text-muted);
    font-size: var(--text-xs);
  }

  .milestone-count {
    font-variant-numeric: tabular-nums;
    flex: 0 0 auto;
    min-width: 22px;
    border-radius: var(--radius-full);
    background: var(--surface);
    color: var(--text-muted);
    font-size: var(--text-xs);
    font-weight: 700;
    padding: var(--space-2xs) var(--space-sm);
    text-align: center;
  }
</style>
