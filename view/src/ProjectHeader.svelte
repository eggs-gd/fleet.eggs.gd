<script lang="ts">
  import { favoriteIds, toggleFavorite } from './lib/favorites';
  import { PROJECT_SETTINGS_VIEW } from './lib/projectSettings';
  import {
    hasTechnology,
    itemTechnologyView,
    repoWebUrl,
    techRepositories
  } from './lib/technologyDisplay';
  import TechnologyTags from './TechnologyTags.svelte';
  import Icon from './Icon.svelte';
  import type { Project } from './lib/types';

  export let project: Project | null = null;
  export let activeView = 'board';
  export let onChangeView: (viewId: string) => void = () => {};
  export let onBack: () => void = () => {};
  export let onCreateTask: () => void = () => {};
  export let onRefresh: () => void = () => {};
  export let refreshing = false;

  $: views = project
    ? [
        { id: 'board', label: 'Tasks' },
        { id: 'archive', label: 'Archive' },
        { id: PROJECT_SETTINGS_VIEW, label: 'Settings' }
      ]
    : [
        { id: 'board', label: 'Tasks' },
        { id: 'archive', label: 'Archive' }
      ];

  $: primaryRemote = project
    ? techRepositories(project)
        .map((repo) => repo.remote)
        .find(Boolean) || ''
    : '';
  $: primaryRemoteUrl = repoWebUrl(primaryRemote);

  $: isFavorite = Boolean(project && $favoriteIds.includes(project.id));
</script>

<header class="project-header">
  <div class="project-header-row">
    {#if project}
      <div class="project-header-title">
        <button type="button" class="project-header-back" on:click={onBack} title="All Projects">
          <Icon name="chevron-left" size={16} />
        </button>
        <button
          type="button"
          class="project-header-favorite"
          class:is-favorite={isFavorite}
          title={isFavorite ? 'Remove from favorites' : 'Add to favorites'}
          on:click={() => toggleFavorite(project.id)}
        >
          <Icon name="star" size={17} filled={isFavorite} />
        </button>
        <h2>{project.title || project.id}</h2>
        {#if project.relative_path || project.path}
          <code class="project-header-path">~/{project.relative_path || project.path}</code>
        {/if}
        {#if primaryRemoteUrl}
          <a
            class="project-header-icon-link"
            href={primaryRemoteUrl}
            target="_blank"
            rel="noreferrer"
            title={primaryRemote}
          >
            <Icon name="github" size={16} />
          </a>
        {/if}
      </div>
    {:else}
      <div class="project-header-title">
        <h2>All Projects</h2>
      </div>
    {/if}
    <div class="project-header-actions">
      <button type="button" class="primary-action" on:click={onCreateTask}>
        <Icon name="plus" size={15} /> New Task
      </button>
      <button
        type="button"
        class="icon-button"
        class:is-busy={refreshing}
        aria-busy={refreshing}
        title="Refresh"
        on:click={onRefresh}
      >
        <Icon name="refresh" size={16} />
      </button>
    </div>
  </div>
  {#if project && hasTechnology(project)}
    {@const techView = itemTechnologyView(project)}
    <div class="technology-strip single-row" aria-label="Project technology">
      <TechnologyTags
        groups={[
          { tags: techView.primary, variant: 'primary' },
          { tags: techView.tooling, variant: 'tooling' }
        ]}
        limit={14}
      />
    </div>
  {/if}
</header>

<nav class="project-tabs" aria-label="Project view">
  {#each views as view (view.id)}
    <button
      type="button"
      class:active={activeView === view.id}
      on:click={() => onChangeView(view.id)}
    >
      {view.label}
    </button>
  {/each}
</nav>

<style>
  .technology-strip {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs);
    margin-top: var(--space-sm);
    max-height: 92px;
    overflow: auto;
  }

  .technology-strip.single-row {
    flex-wrap: nowrap;
    align-items: center;
    max-height: 28px;
    overflow-x: auto;
    overflow-y: hidden;
    scrollbar-width: thin;
  }

  /* .tech-more is TechnologyTags.svelte's own overflow chip, rendered as a
     child of this strip. */
  .technology-strip.single-row :global(.tech-more) {
    flex: 0 0 auto;
  }

  .project-header {
    flex: 0 0 auto;
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    background: var(--surface);
    padding: var(--space-lg) var(--space-lg);
  }

  .project-header-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-lg);
  }

  .project-header-title {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-md);
    min-width: 0;
  }

  .project-header-back {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    min-height: 30px;
    border-color: var(--border);
    color: var(--text-muted);
    padding: 0;
  }

  .project-header-favorite {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    min-height: 28px;
    border-color: transparent;
    background: transparent;
    color: var(--text-faint);
    padding: 0;
  }

  .project-header-favorite:hover {
    background: var(--surface-muted);
    color: var(--text-muted);
  }

  .project-header-favorite.is-favorite {
    color: var(--amber);
  }

  .project-header-title h2 {
    margin: 0;
    font-size: var(--text-2xl);
    white-space: nowrap;
  }

  .project-header-path {
    overflow: hidden;
    color: var(--text-faint);
    font-size: var(--text-sm);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .project-header-icon-link {
    display: inline-flex;
    align-items: center;
    color: var(--text-muted);
  }

  .project-header-icon-link:hover {
    color: var(--text);
  }

  .project-header-actions {
    display: flex;
    flex: 0 0 auto;
    gap: var(--space-md);
  }

  .project-tabs {
    display: flex;
    gap: var(--space-xl);
    flex: 0 0 auto;
    border-bottom: 1px solid var(--border);
    padding: 0 var(--space-2xs);
  }

  .project-tabs button {
    min-height: 0;
    border: none;
    border-bottom: 2px solid transparent;
    border-radius: 0;
    background: transparent;
    color: var(--text-muted);
    font-size: var(--text-md);
    font-weight: 600;
    padding: var(--space-md) var(--space-2xs);
    margin-bottom: -1px;
  }

  .project-tabs button:hover {
    color: var(--text);
    background: transparent;
  }

  .project-tabs button.active {
    border-bottom-color: var(--accent);
    color: var(--accent);
  }
</style>
