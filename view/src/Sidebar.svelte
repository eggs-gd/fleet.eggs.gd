<script lang="ts">
  import {
    isAccordionOpen,
    updateAccordionOpen,
    workspaceAccordionKey
  } from './lib/dashboardState';
  import { SETTINGS_SECTIONS } from './lib/settingsNav';
  import { favoriteIds } from './lib/favorites';
  import {
    buildSidebarTree,
    projectColor,
    projectInitial,
    projectOpenCount
  } from './lib/projectTree';
  import { isOperatorAssignee } from './lib/operator';
  import { operatorPause } from './lib/taskDisplay';
  import Icon from './Icon.svelte';
  import SidebarBranch from './SidebarBranch.svelte';
  import ThemeSwitch from './ThemeSwitch.svelte';
  import type { Project, Task, Workspace } from './lib/types';

  export let workspaces: Workspace[] = [];
  export let projects: Project[] = [];
  export let tasks: Task[] = [];
  export let selectedProjectId = '';
  export let selectedWorkspaceId = '';
  export let quickView = 'all';
  export let accordionOpen: Record<string, boolean> = {};
  export let globalNav = 'work';
  export let settingsSection = 'general';
  export let onSelectProject: (projectId: string) => void = () => {};
  export let onSelectWorkspace: (workspaceId: string) => void = () => {};
  export let onSelectAll: () => void = () => {};
  export let onSelectView: (viewId: string) => void = () => {};
  export let onSelectNav: (nav: string) => void = () => {};
  export let onSelectSettingsSection: (id: string) => void = () => {};
  export let onTreeScroll: (value: number) => void = () => {};
  export let onScrollEl: (el: HTMLElement | null) => void = () => {};
  export let themePref = 'system';
  export let onThemeChange: (pref: string) => void = () => {};

  let treeEl: HTMLElement;

  $: tree = buildSidebarTree(workspaces, projects, tasks);
  $: bookmarks = $favoriteIds
    .map((id) => projects.find((project) => project.id === id))
    .filter((project): project is Project => Boolean(project));
  $: views = [
    { id: 'all', label: 'All tasks', icon: 'folder', count: tasks.length },
    {
      id: 'mine',
      label: 'Assigned to me',
      icon: 'users',
      count: tasks.filter((task) => isOperatorAssignee(task.assignee)).length
    },
    {
      id: 'attention',
      label: 'Needs attention',
      icon: 'help-circle',
      count: tasks.filter(
        (task) =>
          task.status === 'blocked' || task.status === 'needs_review' || operatorPause(task).waiting
      ).length
    },
    {
      id: 'blocked',
      label: 'Blocked',
      icon: 'x-circle',
      count: tasks.filter((task) => task.status === 'blocked').length
    },
    {
      id: 'review',
      label: 'In review',
      icon: 'check-circle',
      count: tasks.filter((task) => task.status === 'needs_review').length
    },
    {
      id: 'done',
      label: 'Done',
      icon: 'checks-circle',
      count: tasks.filter((task) => task.status === 'done').length
    },
    {
      id: 'archived',
      label: 'Archived',
      icon: 'folder',
      count: tasks.filter((task) => task.status === 'archived').length
    }
  ];

  function toggle(key: string, open: boolean) {
    accordionOpen = updateAccordionOpen(accordionOpen, key, open);
  }

  function activateWorkspace(workspaceId: string, key: string, isOpen: boolean) {
    const reclick = selectedWorkspaceId === workspaceId && !selectedProjectId;
    onSelectWorkspace(workspaceId);
    if (reclick) toggle(key, !isOpen);
    else if (!isOpen) toggle(key, true);
  }

  function activateProject(projectId: string, key: string, isOpen: boolean, hasChildren: boolean) {
    const reclick = selectedProjectId === projectId;
    onSelectProject(projectId);
    if (!hasChildren) return;
    if (reclick) toggle(key, !isOpen);
    else if (!isOpen) toggle(key, true);
  }

  $: onScrollEl(treeEl);
</script>

<nav class="sidebar sidebar-left" aria-label="Navigation">
  <div class="sidebar-brand">
    <span class="sidebar-logo">F</span>
    <strong>Fleet</strong>
    <ThemeSwitch pref={themePref} onChange={onThemeChange} />
  </div>

  <div class="sidebar-rail">
    <button
      type="button"
      class="sidebar-rail-item"
      class:active={globalNav === 'work'}
      on:click={() => onSelectNav('work')}
    >
      <Icon name="home" size={16} /> Work
    </button>
    <button
      type="button"
      class="sidebar-rail-item"
      class:active={globalNav === 'settings'}
      on:click={() => onSelectNav('settings')}
    >
      <Icon name="gear" size={16} /> Settings
    </button>
  </div>

  {#if globalNav === 'settings'}
    <div class="sidebar-section">
      <p class="sidebar-section-title">Settings</p>
      <ul class="sidebar-views">
        {#each SETTINGS_SECTIONS as item (item.id)}
          <li>
            <button
              type="button"
              class="sidebar-view"
              class:active={settingsSection === item.id}
              on:click={() => onSelectSettingsSection(item.id)}
            >
              <span class="sidebar-view-title">{item.label}</span>
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {:else if globalNav === 'work'}
    <div class="sidebar-section sidebar-section--projects">
      <p class="sidebar-section-title">Projects</p>
      <button
        type="button"
        class="sidebar-all"
        class:active={!selectedProjectId && !selectedWorkspaceId}
        on:click={onSelectAll}
      >
        <span>All projects</span>
        <span class="sidebar-view-count">{projects.length}</span>
      </button>
      {#if bookmarks.length}
        <p class="sidebar-section-title">Bookmarks</p>
        {#each bookmarks as project (project.id)}
          <button
            type="button"
            class="sidebar-project sidebar-project--root"
            class:active={selectedProjectId === project.id}
            on:click={() => onSelectProject(project.id)}
          >
            <span
              class="project-mark project-mark--project"
              style="background: {projectColor(project.id)}"
              >{projectInitial(project.title, project.id)}</span
            >
            <span class="sidebar-project-title">{project.title || project.id}</span>
            <span class="sidebar-project-count">{projectOpenCount(tasks, project.id)}</span>
          </button>
        {/each}
      {/if}
      <div
        bind:this={treeEl}
        class="sidebar-workspaces"
        on:scroll={(event) => onTreeScroll(event.currentTarget.scrollTop)}
      >
        {#each tree as node (node.workspace.id)}
          {@const wsKey = workspaceAccordionKey(node.workspace)}
          {@const wsOpen = isAccordionOpen(accordionOpen, wsKey, node.taskCount > 0)}
          {#if node.isGroup}
            <div class="sidebar-workspace">
              <button
                type="button"
                class="sidebar-space"
                class:active={selectedWorkspaceId === node.workspace.id && !selectedProjectId}
                aria-expanded={wsOpen}
                on:click={() => activateWorkspace(node.workspace.id, wsKey, wsOpen)}
              >
                <span
                  class="project-mark project-mark--space"
                  style="background: {projectColor(node.workspace.id)}"
                  >{projectInitial(node.workspace.title, node.workspace.id)}</span
                >
                <span class="sidebar-project-title">{node.workspace.title}</span>
                <span class="sidebar-project-count">{node.taskCount}</span>
                <span class="tree-toggle" class:is-open={wsOpen} aria-hidden="true">
                  <Icon name="chevron-right" size={12} />
                </span>
              </button>
              {#if wsOpen}
                <ul>
                  {#if node.rootNode}
                    <SidebarBranch
                      node={node.rootNode}
                      depth={1}
                      {selectedProjectId}
                      {accordionOpen}
                      onActivate={activateProject}
                    />
                  {/if}
                  {#each node.children as child (child.project.id)}
                    <SidebarBranch
                      node={child}
                      depth={1}
                      {selectedProjectId}
                      {accordionOpen}
                      onActivate={activateProject}
                    />
                  {/each}
                </ul>
              {/if}
            </div>
          {:else if node.leafProject}
            <button
              type="button"
              class="sidebar-project sidebar-project--root"
              class:active={selectedProjectId === node.leafProject.id ||
                selectedWorkspaceId === node.workspace.id}
              on:click={() => onSelectProject(node.leafProject!.id)}
            >
              <span
                class="project-mark project-mark--project"
                style="background: {projectColor(node.leafProject.id)}"
                >{projectInitial(node.leafProject.title, node.leafProject.id)}</span
              >
              <span class="sidebar-project-title"
                >{node.leafProject.title || node.leafProject.id}</span
              >
              <span class="sidebar-project-count">{node.taskCount}</span>
            </button>
          {/if}
        {:else}
          <p class="empty">No projects yet.</p>
          <p class="empty">
            Fleet finds projects by scanning folders that hold your git repositories.
          </p>
          <button
            type="button"
            class="settings-btn"
            on:click={() => onSelectSettingsSection('general')}
          >
            Add a folder to scan
          </button>
        {/each}
      </div>
    </div>

    <div class="sidebar-section">
      <p class="sidebar-section-title">Views</p>
      <ul class="sidebar-views">
        {#each views as view (view.id)}
          <li>
            <button
              type="button"
              class="sidebar-view"
              class:active={quickView === view.id}
              on:click={() => onSelectView(view.id)}
            >
              <Icon name={view.icon} size={15} />
              <span class="sidebar-view-title">{view.label}</span>
              <span class="sidebar-view-count">{view.count}</span>
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  <p class="sidebar-footer">
    {#if globalNav === 'settings'}
      Settings
    {:else}
      {workspaces.length} workspaces &middot; {projects.length} projects
    {/if}
  </p>
</nav>

<style>
  .sidebar-left {
    width: var(--left-w);
    flex: 0 0 var(--left-w);
    min-width: 0;
    min-height: 0;
  }

  @media (max-width: 900px) {
    .sidebar-left {
      width: 100%;
      flex: 1 1 auto;
    }
  }

  .sidebar-brand {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    padding: var(--space-2xs) var(--space-xs) var(--space-xs);
  }

  .sidebar-logo {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    flex: 0 0 auto;
    border-radius: 50%;
    background: var(--accent);
    color: var(--on-accent);
    font-size: var(--text-md);
    font-weight: 800;
  }

  .sidebar-brand strong {
    flex: 1 1 auto;
    min-width: 0;
    overflow: hidden;
    font-size: var(--text-lg);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sidebar-footer {
    margin: auto 0 0;
    padding: var(--space-2xs) var(--space-xs) 0;
    color: var(--text-faint);
    font-size: var(--text-xs);
  }

  .sidebar-rail {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-2xs);
  }

  .sidebar-rail-item {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    width: 100%;
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    padding: var(--space-sm) var(--space-md);
    color: var(--text-muted);
    font-size: var(--text-md);
    font-weight: 600;
    text-align: left;
    min-height: 0;
    white-space: nowrap;
  }

  .sidebar-rail-item.active {
    background: var(--accent-soft);
    color: var(--accent-soft-text);
  }

  .sidebar-section {
    display: grid;
    gap: var(--space-sm);
    flex: 0 0 auto;
  }

  .sidebar-section--projects {
    flex: 1 1 auto;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .sidebar-section-title {
    margin: 0;
    padding: 0 var(--space-xs);
    color: var(--text-faint);
    font-size: var(--text-xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .sidebar-all {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-md);
    width: 100%;
    border: none;
    background: transparent;
    color: var(--text);
    text-align: left;
    font-weight: 700;
    font-size: var(--text-md);
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-sm);
    min-height: 0;
  }

  .sidebar-all:hover {
    background: var(--surface-muted);
  }

  .sidebar-all.active {
    background: var(--accent-soft);
    color: var(--accent-soft-text);
  }

  .sidebar-workspaces {
    flex: 1 1 auto;
    min-height: 0;
    min-width: 0;
    overflow-y: auto;
    overflow-x: hidden;
    display: grid;
    align-content: start;
    gap: var(--space-xs);
    padding-right: var(--space-2xs);
  }

  .sidebar-views {
    display: grid;
    gap: 1px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .sidebar-view {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    width: 100%;
    border: none;
    background: transparent;
    text-align: left;
    font-size: var(--text-md);
    color: var(--text-muted);
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-sm);
    min-height: 0;
  }

  .sidebar-view:hover {
    background: var(--surface-muted);
    color: var(--text);
  }

  .sidebar-view.active {
    background: var(--accent-soft);
    color: var(--accent-soft-text);
    font-weight: 600;
  }

  .sidebar-view-title {
    flex: 1 1 auto;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sidebar-view-count {
    flex: 0 0 auto;
    color: var(--text-faint);
    font-size: var(--text-sm);
  }
</style>
