<script lang="ts">
  import ArchiveView from '../components/board/ArchiveView.svelte';
  import NeedsAttention from '../components/board/NeedsAttention.svelte';
  import ProjectSettings from '../components/board/ProjectSettings.svelte';
  import RecentCompleted from '../components/board/RecentCompleted.svelte';
  import MilestoneTaskList from '../components/milestones/MilestoneTaskList.svelte';
  import RuntimeSessions from '../components/sessions/RuntimeSessions.svelte';
  import FiltersBar from '../components/shell/FiltersBar.svelte';
  import ProjectHeader from '../components/shell/ProjectHeader.svelte';
  import SplitGutter from '../components/shell/SplitGutter.svelte';
  import { PROJECT_SETTINGS_VIEW } from '../lib/projectSettings';
  import type { Project, Repository, Session, Task, Workspace } from '../lib/types';

  interface Chip {
    id: string;
    label: string;
    count: number;
  }

  interface RefreshStatus {
    label: string;
    warning: boolean;
    busy: boolean;
    busyLabel: string;
    updatedAt: string;
    updatedText: string;
  }

  export let runtimeSessions: Session[] = [];
  export let closedSessions: Session[] = [];
  export let orphanedTasks: Session[] = [];
  export let refreshStatus: RefreshStatus | null = null;
  export let sessionsTab = 'active';
  export let actionError = '';
  export let selectedProject: Project | null = null;
  export let activeView = 'board';
  export let chips: Chip[] = [];
  export let quickFilter = 'all';
  export let query = '';
  export let taskSort = 'updated';
  export let refreshing = false;
  export let selectedProjectId = '';
  export let selectedWorkspaceId = '';
  export let accordionOpen: Record<string, boolean> = {};
  export let archivedTasks: Task[] = [];
  export let workspaces: Workspace[] = [];
  export let projects: Project[] = [];
  export let transitions: Record<string, string[]> | null = null;
  export let repositories: Repository[] = [];
  export let filteredActiveTasks: Task[] = [];
  export let selectedTask: Task | null = null;
  export let generatedAt = '';
  export let repositoryCount = 0;
  export let attentionTasks: Task[] = [];
  export let tasks: Task[] = [];
  export let attentionOpen = true;
  export let recentOpen = true;

  export let onControlSession: (
    session: Session,
    action: string,
    input?: string,
    targetStatus?: string
  ) => Promise<void> = async () => {};
  export let onReleaseSession: (
    session: Session,
    action: string,
    targetStatus: string
  ) => Promise<void> = async () => {};
  export let onResolveOrphan: (session: Session, status: string) => Promise<void> = async () => {};
  export let onSessionsTab: (tab: string) => void = () => {};
  export let onChangeView: (view: string) => void = () => {};
  export let onBack: () => void = () => {};
  export let onCreateTask: () => void = () => {};
  export let onRefresh: () => void = () => {};
  export let onChipChange: (chipId: string) => void = () => {};
  export let onSortChange: (value: string) => void = () => {};
  export let onAddMilestone: (() => void) | null = null;
  export let onOpenTask: (task: Task) => void = () => {};
  export let onTransitionTask: (task: Task, status: string) => Promise<void> = async () => {};
  export let onDropTaskOnMilestone: (
    taskPath: string,
    milestoneId: string
  ) => Promise<void> = async () => {};
  export let onDragRight: (dx: number) => void = () => {};
  export let onToggleAttention: () => void = () => {};
  export let onToggleRecent: () => void = () => {};
  export let onShowAttention: () => void = () => {};
  export let onShowDone: () => void = () => {};
</script>

<div class="working-area">
  <RuntimeSessions
    {runtimeSessions}
    {closedSessions}
    {orphanedTasks}
    {refreshStatus}
    tab={sessionsTab}
    onTabChange={onSessionsTab}
    {onControlSession}
    {onReleaseSession}
    {onResolveOrphan}
  />

  {#if actionError}
    <section class="error action-error" aria-live="assertive">
      <strong>Task update failed.</strong>
      <span>{actionError}</span>
    </section>
  {/if}

  <div class="main-column">
    <ProjectHeader
      project={selectedProject}
      {activeView}
      {onChangeView}
      {onBack}
      {onCreateTask}
      {onRefresh}
      {refreshing}
    />

    {#if activeView === 'board'}
      <FiltersBar
        {chips}
        activeChip={quickFilter}
        bind:query
        sort={taskSort}
        {onChipChange}
        {onSortChange}
        {onAddMilestone}
      />
    {/if}

    <div class="main-scroll">
      {#if activeView === 'archive'}
        <ArchiveView
          tasks={archivedTasks}
          {workspaces}
          {projects}
          {selectedProjectId}
          {selectedWorkspaceId}
          bind:accordionOpen
          {transitions}
          {onOpenTask}
          {onTransitionTask}
        />
      {:else if activeView === PROJECT_SETTINGS_VIEW && selectedProject}
        <ProjectSettings project={selectedProject} {workspaces} {projects} {repositories} />
      {:else}
        <MilestoneTaskList
          tasks={filteredActiveTasks}
          {projects}
          {workspaces}
          {selectedProjectId}
          {selectedWorkspaceId}
          {selectedTask}
          bind:accordionOpen
          {transitions}
          {onOpenTask}
          {onTransitionTask}
          {onDropTaskOnMilestone}
        />
      {/if}

      <footer class="generated">
        Generated {generatedAt ? new Date(generatedAt).toLocaleString() : ''} from {repositoryCount} repositories.
      </footer>
    </div>
  </div>
</div>

<SplitGutter axis="x" onDrag={onDragRight} />

<aside class="sidebar sidebar-right">
  <NeedsAttention
    tasks={attentionTasks}
    open={attentionOpen}
    {transitions}
    onToggle={onToggleAttention}
    {onOpenTask}
    onShowAll={onShowAttention}
    {onTransitionTask}
  />
  <RecentCompleted
    {tasks}
    open={recentOpen}
    {transitions}
    onToggle={onToggleRecent}
    {onOpenTask}
    onShowAll={onShowDone}
    {onTransitionTask}
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
