<script>
  import {
    isAccordionOpen,
    updateAccordionOpen,
    workspaceAccordionKey
  } from './lib/dashboardState.js';
  import { SETTINGS_SECTIONS } from './lib/settingsNav.js';
  import { favoriteIds } from './lib/favorites.js';
  import {
    buildSidebarTree,
    projectColor,
    projectInitial,
    projectOpenCount
  } from './lib/projectTree.js';
  import { isOperatorAssignee } from './lib/operator.js';
  import { operatorPause } from './lib/taskDisplay.js';
  import Icon from './Icon.svelte';
  import SidebarBranch from './SidebarBranch.svelte';
  import ThemeSwitch from './ThemeSwitch.svelte';

  export let workspaces = [];
  export let projects = [];
  export let tasks = [];
  export let selectedProjectId = '';
  export let selectedWorkspaceId = '';
  export let quickView = 'all';
  export let accordionOpen = {};
  export let globalNav = 'work';
  export let settingsSection = 'general';
  export let onSelectProject = () => {};
  export let onSelectWorkspace = () => {};
  export let onSelectAll = () => {};
  export let onSelectView = () => {};
  export let onSelectNav = () => {};
  export let onSelectSettingsSection = () => {};
  export let onTreeScroll = () => {};
  export let onScrollEl = () => {};
  export let themePref = 'system';
  export let onThemeChange = () => {};

  let treeEl;

  $: tree = buildSidebarTree(workspaces, projects, tasks);
  $: bookmarks = $favoriteIds
    .map((id) => projects.find((project) => project.id === id))
    .filter(Boolean);
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

  function toggle(key, open) {
    accordionOpen = updateAccordionOpen(accordionOpen, key, open);
  }

  function activateWorkspace(workspaceId, key, isOpen) {
    const reclick = selectedWorkspaceId === workspaceId && !selectedProjectId;
    onSelectWorkspace(workspaceId);
    if (reclick) toggle(key, !isOpen);
    else if (!isOpen) toggle(key, true);
  }

  function activateProject(projectId, key, isOpen, hasChildren) {
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
              on:click={() => onSelectProject(node.leafProject.id)}
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
