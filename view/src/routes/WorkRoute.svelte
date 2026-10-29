<script lang="ts">
  import ArchiveView from '../components/board/ArchiveView.svelte';
  import NeedsAttention from '../components/board/NeedsAttention.svelte';
  import ProjectSettings from '../components/board/ProjectSettings.svelte';
  import RecentCompleted from '../components/board/RecentCompleted.svelte';
  import MilestoneTaskList from '../components/milestones/MilestoneTaskList.svelte';
  import RuntimeSessions from '../components/sessions/RuntimeSessions.svelte';
  import EmptyBoard from '../components/board/EmptyBoard.svelte';
  import DryRunBanner from '../components/shell/DryRunBanner.svelte';
  import FiltersBar from '../components/shell/FiltersBar.svelte';
  import ProjectHeader from '../components/shell/ProjectHeader.svelte';
  import SplitGutter from '../components/shell/SplitGutter.svelte';
  import { PROJECT_SETTINGS_VIEW } from '../lib/projectSettings';
  import type { Project } from '../lib/types';
  import {
    actionError,
    attentionTasks,
    closedSessions,
    data,
    launchMode,
    orphanedTasks,
    projects,
    refreshBusy,
    readyTodoCount,
    refreshStatus,
    requestManualRefresh,
    runtimeSessions,
    tasks,
    workspaces
  } from '../stores/board';
  import {
    accordionOpen,
    dragRight,
    rightPanels,
    toggleAttention,
    toggleRecent
  } from '../stores/layout';
  import {
    activeView,
    changeFilter,
    changeView,
    query,
    quickFilter,
    selectAllProjects,
    selectedProjectId,
    selectedWorkspaceId,
    selectQuickView,
    selectSettingsSection,
    sessionsTab,
    setSessionsTab,
    setSort,
    taskSort
  } from '../stores/location';
  import {
    milestoneCardId,
    openCreateMilestone,
    removeMilestone,
    startEditMilestone
  } from '../stores/milestones';
  import { releaseSession, resolveOrphan, controlSession } from '../stores/sessions';
  import {
    dropTaskOnMilestone,
    openCreateTask,
    openTask,
    selectedTask,
    transitionTask,
    transitions
  } from '../stores/tasks';
  import { archivedTasks, chips, filteredActiveTasks, selectedProject } from '../stores/views';
</script>

<div class="working-area">
  {#if $launchMode === 'dry-run'}
    <DryRunBanner readyCount={$readyTodoCount} onTurnOn={() => selectSettingsSection('general')} />
  {/if}

  <RuntimeSessions
    runtimeSessions={$runtimeSessions}
    closedSessions={$closedSessions}
    orphanedTasks={$orphanedTasks}
    refreshStatus={$refreshStatus}
    tab={$sessionsTab}
    onTabChange={setSessionsTab}
    onControlSession={controlSession}
    onReleaseSession={releaseSession}
    onResolveOrphan={resolveOrphan}
  />

  {#if $actionError}
    <section class="error action-error" aria-live="assertive">
      <strong>Task update failed.</strong>
      <span>{$actionError}</span>
    </section>
  {/if}

  <div class="main-column">
    <ProjectHeader
      project={$selectedProject as Project | null}
      activeView={$activeView}
      onChangeView={changeView}
      onBack={selectAllProjects}
      onCreateTask={openCreateTask}
      canCreateTask={$projects.length > 0}
      onRefresh={requestManualRefresh}
      refreshing={$refreshBusy}
    />

    {#if $activeView === 'board'}
      <FiltersBar
        chips={$chips}
        activeChip={$quickFilter}
        bind:query={$query}
        sort={$taskSort}
        onChipChange={changeFilter}
        onSortChange={setSort}
        onAddMilestone={$milestoneCardId ? openCreateMilestone : null}
      />
    {/if}

    <div class="main-scroll">
      {#if $activeView === 'archive'}
        <ArchiveView
          tasks={$archivedTasks}
          workspaces={$workspaces}
          projects={$projects}
          selectedProjectId={$selectedProjectId}
          selectedWorkspaceId={$selectedWorkspaceId}
          bind:accordionOpen={$accordionOpen}
          transitions={$transitions}
          onOpenTask={openTask}
          onTransitionTask={transitionTask}
        />
      {:else if $activeView === PROJECT_SETTINGS_VIEW && $selectedProject}
        <ProjectSettings
          project={$selectedProject as Project}
          workspaces={$workspaces}
          projects={$projects}
          repositories={$data?.registry?.repositories || []}
        />
      {:else if !$projects.length}
        <EmptyBoard />
      {:else}
        <MilestoneTaskList
          tasks={$filteredActiveTasks}
          projects={$projects}
          workspaces={$workspaces}
          selectedProjectId={$selectedProjectId}
          selectedWorkspaceId={$selectedWorkspaceId}
          selectedTask={$selectedTask}
          bind:accordionOpen={$accordionOpen}
          transitions={$transitions}
          onOpenTask={openTask}
          onTransitionTask={transitionTask}
          onDropTaskOnMilestone={dropTaskOnMilestone}
          onEditMilestone={startEditMilestone}
          onDeleteMilestone={removeMilestone}
        />
      {/if}

      <footer class="generated">
        Generated {$data?.generated_at ? new Date($data.generated_at).toLocaleString() : ''} from {$data
          ?.registry?.repositories_count || 0} repositories.
      </footer>
    </div>
  </div>
</div>

<SplitGutter axis="x" onDrag={dragRight} />

<aside class="sidebar sidebar-right">
  <NeedsAttention
    tasks={$attentionTasks}
    open={$rightPanels.attentionOpen}
    transitions={$transitions}
    onToggle={toggleAttention}
    onOpenTask={openTask}
    onShowAll={() => selectQuickView('attention')}
    onTransitionTask={transitionTask}
  />
  <RecentCompleted
    tasks={$tasks}
    open={$rightPanels.recentOpen}
    transitions={$transitions}
    onToggle={toggleRecent}
    onOpenTask={openTask}
    onShowAll={() => selectQuickView('done')}
    onTransitionTask={transitionTask}
  />
</aside>

<style>
  .working-area {
    flex: 1 1 auto;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-lg);
  }

  .main-column {
    flex: 1 1 auto;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-lg);
  }

  .sidebar-right {
    width: var(--right-w);
    flex: 0 0 var(--right-w);
    min-width: 0;
    max-width: var(--right-w);
    align-self: stretch;
  }

  .main-scroll {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    display: flex;
    flex-direction: column;
    gap: var(--space-lg);
    padding-right: var(--space-2xs);
  }

  .error {
    border: 1px solid var(--error-border);
    border-radius: var(--radius-md);
    background: var(--red-soft);
    color: var(--red-text);
  }

  .error span {
    display: block;
    margin-top: var(--space-sm);
  }

  .action-error {
    padding: var(--space-lg);
    flex: 0 0 auto;
  }

  .generated {
    flex: 0 0 auto;
    padding: var(--space-xs) 0 var(--space-2xs);
    color: var(--text-muted);
    font-size: var(--text-sm);
  }

  @media (max-width: 900px) {
    .working-area {
      display: flex;
      flex-direction: column;
      height: auto;
    }

    .sidebar-right,
    .main-column {
      width: 100%;
      flex: 1 1 auto;
    }

    .main-scroll {
      overflow-y: visible;
    }
  }
</style>
