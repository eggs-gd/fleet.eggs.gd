<script>
  import {
    archiveProjectAccordionKey,
    archiveWorkspaceAccordionKey,
    isAccordionOpen,
    taskKey,
    updateAccordionOpen
  } from './lib/dashboardState.js';
  import { archiveListMode, groupedByWorkspaceAndProject } from './lib/taskDisplay.js';
  import Icon from './Icon.svelte';
  import TaskCard from './TaskCard.svelte';

  export let tasks = [];
  export let workspaces = [];
  export let projects = [];
  export let selectedProjectId = '';
  export let selectedWorkspaceId = '';
  export let accordionOpen = {};
  export let onOpenTask = () => {};
  export let transitions = null;
  export let onTransitionTask = null;

  $: groups = groupedByWorkspaceAndProject(tasks, workspaces, projects);
  $: mode = archiveListMode(selectedProjectId, selectedWorkspaceId);

  function toggle(key, currentlyOpen) {
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
                <TaskCard {task} {transitions} onOpen={onOpenTask} onTransition={onTransitionTask} />
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
                    <TaskCard {task} {transitions} onOpen={onOpenTask} onTransition={onTransitionTask} />
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
