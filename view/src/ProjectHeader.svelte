<script>
  import { favoriteIds, toggleFavorite } from './lib/favorites.js';
  import { PROJECT_SETTINGS_VIEW } from './lib/projectSettings.js';
  import { hasTechnology, itemTechnologyView, repoWebUrl, techRepositories } from './lib/technologyDisplay.js';
  import TechnologyTags from './TechnologyTags.svelte';
  import Icon from './Icon.svelte';

  export let project = null;
  export let activeView = 'board';
  export let onChangeView = () => {};
  export let onBack = () => {};
  export let onCreateTask = () => {};
  export let onRefresh = () => {};
  export let refreshing = false;

  $: views = [
    { id: 'board', label: 'Tasks' },
    { id: 'archive', label: 'Archive' }
  ];

  $: primaryRemote = project ? techRepositories(project).map((repo) => repo.remote).find(Boolean) || '' : '';
  $: primaryRemoteUrl = repoWebUrl(primaryRemote);

  $: isFavorite = Boolean(project) && $favoriteIds.includes(project.id);
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
        <button
          type="button"
          class="project-header-favorite"
          class:is-active={activeView === PROJECT_SETTINGS_VIEW}
          title="Project settings"
          on:click={() => onChangeView(PROJECT_SETTINGS_VIEW)}
        >
          <Icon name="gear" size={16} />
        </button>
        <h2>{project.title || project.id}</h2>
        {#if project.relative_path || project.path}
          <code class="project-header-path">~/{project.relative_path || project.path}</code>
        {/if}
        {#if primaryRemoteUrl}
          <a class="project-header-icon-link" href={primaryRemoteUrl} target="_blank" rel="noreferrer" title={primaryRemote}>
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
    <button type="button" class:active={activeView === view.id} on:click={() => onChangeView(view.id)}>
      {view.label}
    </button>
  {/each}
</nav>
