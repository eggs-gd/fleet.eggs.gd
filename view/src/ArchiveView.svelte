<script lang="ts">
  import {
    archiveProjectAccordionKey,
    archiveWorkspaceAccordionKey,
    isAccordionOpen,
    taskKey,
    updateAccordionOpen
  } from './lib/dashboardState';
  import { archiveListMode, groupedByWorkspaceAndProject } from './lib/taskDisplay';
  import Icon from './Icon.svelte';
  import TaskCard from './TaskCard.svelte';
  import type { Project, Task, Workspace } from './lib/types';

  export let tasks: Task[] = [];
  export let workspaces: Workspace[] = [];
  export let projects: Project[] = [];
  export let selectedProjectId = '';
  export let selectedWorkspaceId = '';
  export let accordionOpen: Record<string, boolean> = {};
  export let onOpenTask: (task: Task) => void = () => {};
  export let transitions: Record<string, string[]> | null = null;
  export let onTransitionTask: ((task: Task, status: string) => void) | null = null;

  $: groups = groupedByWorkspaceAndProject(tasks, workspaces, projects);
  $: mode = archiveListMode(selectedProjectId, selectedWorkspaceId);

  function toggle(key: string, currentlyOpen: boolean) {
    accordionOpen = updateAccordionOpen(accordionOpen, key, !currentlyOpen);
  }
</script>

<section class="archive-view" aria-label="Archived tasks">
  <header class="archive-view-header">
    <div>
      <h2>Archive</h2>
      <p>Historical tasks kept out of the active board. Restore returns a task to backlog.</p>
    </div>
    <span>{tasks.length}</span>
  </header>

  {#if tasks.length === 0}
    <p class="empty">No archived tasks match the current filters.</p>
  {:else if mode === 'flat'}
    <div class="archive-list" role="list">
      {#each tasks as task (taskKey(task))}
        <TaskCard {task} {transitions} onOpen={onOpenTask} onTransition={onTransitionTask} />
      {/each}
    </div>
  {:else if mode === 'projects'}
    <div class="archive-list" role="list">
      {#each groups as workspace (workspace.id)}
        {#each workspace.projects as project (project.id)}
          {@const projectKey = archiveProjectAccordionKey(project)}
          {@const projectOpen = isAccordionOpen(accordionOpen, projectKey, true)}
          <section class="workspace-task-group">
            <button
              type="button"
              class="archive-group-toggle"
              aria-expanded={projectOpen}
              on:click={() => toggle(projectKey, projectOpen)}
            >
              <strong>{project.title}</strong>
              <span>{project.tasks.length}</span>
              <span class="tree-toggle" class:is-open={projectOpen} aria-hidden="true">
                <Icon name="chevron-right" size={12} />
              </span>
            </button>
            {#if projectOpen}
              {#each project.tasks as task (taskKey(task))}
                <TaskCard
                  {task}
                  {transitions}
                  onOpen={onOpenTask}
                  onTransition={onTransitionTask}
                />
              {/each}
            {/if}
          </section>
        {/each}
      {/each}
    </div>
  {:else}
    <div class="archive-list" role="list">
      {#each groups as workspace (workspace.id)}
        {@const wsKey = archiveWorkspaceAccordionKey(workspace)}
        {@const wsOpen = isAccordionOpen(accordionOpen, wsKey, true)}
        {@const nestProjects = workspace.isGroup}
        <section class="workspace-task-group">
          <button
            type="button"
            class="archive-group-toggle"
            aria-expanded={wsOpen}
            on:click={() => toggle(wsKey, wsOpen)}
          >
            <strong>{workspace.title}</strong>
            <span>{workspace.tasks.length}</span>
            <span class="tree-toggle" class:is-open={wsOpen} aria-hidden="true">
              <Icon name="chevron-right" size={12} />
            </span>
          </button>

          {#if wsOpen && !nestProjects}
            {#each workspace.tasks as task (taskKey(task))}
              <TaskCard {task} {transitions} onOpen={onOpenTask} onTransition={onTransitionTask} />
            {/each}
          {:else if wsOpen}
            {#each workspace.projects as project (project.id)}
              {@const projectKey = archiveProjectAccordionKey(project)}
              {@const projectOpen = isAccordionOpen(accordionOpen, projectKey, true)}
              <section class="task-group">
                <button
                  type="button"
                  class="archive-group-toggle archive-group-toggle--project"
                  aria-expanded={projectOpen}
                  on:click={() => toggle(projectKey, projectOpen)}
                >
                  <strong>{project.title}</strong>
                  <span>{project.tasks.length}</span>
                  <span class="tree-toggle" class:is-open={projectOpen} aria-hidden="true">
                    <Icon name="chevron-right" size={12} />
                  </span>
                </button>
                {#if projectOpen}
                  {#each project.tasks as task (taskKey(task))}
                    <TaskCard
                      {task}
                      {transitions}
                      onOpen={onOpenTask}
                      onTransition={onTransitionTask}
                    />
                  {/each}
                {/if}
              </section>
            {/each}
          {/if}
        </section>
      {/each}
    </div>
  {/if}
</section>

<style>
  .archive-view {
    display: grid;
    gap: var(--space-lg);
  }

  .archive-view-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-lg);
  }

  .archive-view-header h2 {
    margin: 0 0 var(--space-xs);
    font-size: var(--text-xl);
  }

  .archive-view-header p {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--text-md);
    max-width: 42rem;
  }

  .archive-view-header > span {
    color: var(--text-muted);
    font-size: var(--text-md);
    font-weight: 700;
  }

  .archive-list {
    display: grid;
    gap: var(--space-lg);
  }

  .archive-list:not(:has(.workspace-task-group)) {
    gap: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    overflow: hidden;
    background: var(--surface);
  }

  .workspace-task-group {
    display: grid;
    gap: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    overflow: hidden;
    background: var(--surface);
    padding: 0;
  }

  .workspace-task-group:first-child {
    border-top: 1px solid var(--border);
    padding-top: 0;
  }

  .workspace-task-group > .archive-group-toggle {
    display: flex;
    align-items: center;
    gap: var(--space-lg);
    width: 100%;
    min-height: 0;
    border: none;
    border-bottom: 1px solid var(--border);
    border-radius: 0;
    background: var(--surface-muted);
    color: var(--text);
    padding: var(--space-sm) var(--space-md);
    text-align: left;
  }

  .archive-group-toggle strong {
    min-width: 0;
    flex: 1 1 auto;
    font-size: var(--text-sm);
    line-height: 1.2;
  }

  .archive-group-toggle > span:not(.tree-toggle) {
    min-width: 22px;
    border-radius: var(--radius-full);
    background: var(--surface);
    color: var(--text-muted);
    font-size: var(--text-xs);
    font-weight: 700;
    padding: var(--space-2xs) var(--space-sm);
    text-align: center;
  }

  .task-group {
    display: grid;
    gap: var(--space-md);
    padding-left: var(--space-md);
  }

  .archive-group-toggle--project {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-lg);
    width: 100%;
    min-height: 0;
    border: none;
    border-left: 3px solid var(--border-strong);
    border-radius: 0;
    background: transparent;
    padding: 0 var(--space-2xs) 0 var(--space-sm);
    text-align: left;
  }

  .workspace-task-group > .archive-group-toggle:hover {
    background: var(--surface-muted);
  }

  .archive-group-toggle--project:hover {
    background: var(--surface-muted);
  }

  .archive-group-toggle--project > span:not(.tree-toggle) {
    min-width: 22px;
    border-radius: var(--radius-full);
    background: var(--surface-muted);
    color: var(--text-muted);
    font-weight: 700;
    padding: var(--space-2xs) var(--space-sm);
    text-align: center;
  }
</style>
