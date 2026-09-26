<script>
  import { onMount, tick } from 'svelte';
  import {
    applyTaskUpdate,
    dedupeTasksByIdentity,
    flattenClosedSessions,
    isArchivedStatus,
    pruneAccordionOpen,
    reconcileSelectedTaskFromState,
    REFRESH_BUSY_DELAY_MS,
    refreshStatusPresentation,
    sameTaskRecord,
    sessionIsActive
  } from './lib/dashboardState.js';
  import { canTransitionStatus } from './lib/statusTransitions.js';
  import {
    assigneeLabel,
    dependsOnDraft,
    parseDependsOnDraft,
    operatorPause,
    priorityValue,
    taskInBacklog,
    taskPickupCompare,
    taskProjectId,
    taskUpdatedCompare
  } from './lib/taskDisplay.js';
  import {
    LEFT_WIDTH,
    RIGHT_WIDTH,
    loadAccordionOpen,
    loadLeftWidth,
    loadRightPanels,
    loadRightWidth,
    loadSidebarScroll,
    loadThemePref,
    loadAutoRefreshMs,
    saveAccordionOpen,
    saveLeftWidth,
    saveRightPanels,
    saveRightWidth,
    saveSidebarScroll,
    saveThemePref,
    saveAutoRefreshMs
  } from './lib/layoutPrefs.js';
  import { apiFetch } from './lib/api.js';
  import { applyTheme, watchSystemTheme } from './lib/theme.js';
  import ArchiveView from './ArchiveView.svelte';
  import CreateTaskModal from './CreateTaskModal.svelte';
  import FiltersBar from './FiltersBar.svelte';
  import ManagerSetup from './ManagerSetup.svelte';
  import NeedsAttention from './NeedsAttention.svelte';
  import ProjectHeader from './ProjectHeader.svelte';
  import ProjectSettings from './ProjectSettings.svelte';
  import { PROJECT_SETTINGS_VIEW } from './lib/projectSettings.js';
  import RecentCompleted from './RecentCompleted.svelte';
  import RuntimeSessions from './RuntimeSessions.svelte';
  import Sidebar from './Sidebar.svelte';
  import SplitGutter from './SplitGutter.svelte';
  import SettingsView from './SettingsView.svelte';
  import TaskList from './TaskList.svelte';
  import { isOperatorAssignee, operatorAssignee } from './lib/operator.js';
  import { isSettingsNav } from './lib/settingsNav.js';
  import TaskModal from './TaskModal.svelte';

  let data = null;
  let error = '';
  let refreshError = '';
  let actionError = '';
  let refreshing = false;
  let refreshBusy = false;
  let refreshBusyTimer = null;
  let lastRefreshAt = '';
  let selectedProjectId = '';
  let selectedWorkspaceId = '';
  let activeView = 'board';
  let globalNav = 'work';
  let settingsSection = 'general';
  let quickFilter = 'all';
  let taskSort = 'updated';
  let query = '';
  let selectedTask = null;
  let creatingTask = false;
  let draftStatus = '';
  let draftPriority = 5;
  let draftAssignee = '';
  let draftProject = '';
  let draftDependsOn = '';
  let draftBody = '';
  let draftComment = '';
  let saving = false;
  let creating = false;
  let editError = '';
  let createError = '';
  let accordionOpen = loadAccordionOpen();
  let leftWidth = loadLeftWidth();
  let rightWidth = loadRightWidth();
  let rightPanels = loadRightPanels();
  let sidebarScrollTop = loadSidebarScroll();
  let sidebarScrollEl = null;
  let themePref = loadThemePref();
  let refreshMs = loadAutoRefreshMs();
  let refreshTimer = null;
  $: refreshStatus = refreshStatusPresentation({
    refreshing: refreshBusy,
    refreshError,
    lastRefreshAt
  });

  const reconcileSelectedTask = (state) => {
    if (!selectedTask) return;
    const latestTask = reconcileSelectedTaskFromState(selectedTask, state);
    if (latestTask) {
      selectedTask = latestTask;
    } else {
      selectedTask = null;
      editError = '';
    }
  };

  function clearRefreshBusyTimer() {
    if (refreshBusyTimer == null) return;
    window.clearTimeout(refreshBusyTimer);
    refreshBusyTimer = null;
  }

  async function loadData({ manual = false } = {}) {
    if (refreshing) return;
    refreshing = true;
    clearRefreshBusyTimer();
    if (manual) {
      refreshBusy = true;
    } else {
      // Fast auto-polls stay visually quiet so the top menu does not flash every 5s.
      refreshBusyTimer = window.setTimeout(() => {
        refreshBusy = true;
        refreshBusyTimer = null;
      }, REFRESH_BUSY_DELAY_MS);
    }
    try {
      const response = await apiFetch('/api/state', { cache: 'no-store' });
      if (!response.ok) {
        throw new Error(`Failed to load Core state (${response.status})`);
      }
      const nextData = await response.json();
      nextData.tasks = dedupeTasksByIdentity(nextData.tasks ?? []);
      if (sidebarScrollEl) sidebarScrollTop = sidebarScrollEl.scrollTop;
      accordionOpen = pruneAccordionOpen(accordionOpen, nextData);
      saveAccordionOpen(accordionOpen);
      reconcileSelectedTask(nextData);
      data = nextData;
      error = '';
      refreshError = '';
      actionError = '';
      lastRefreshAt = new Date().toISOString();
      await tick();
      if (sidebarScrollEl) sidebarScrollEl.scrollTop = sidebarScrollTop;
    } catch (err) {
      if (data) {
        refreshError = err.message;
      } else {
        error = err.message;
      }
    } finally {
      clearRefreshBusyTimer();
      refreshing = false;
      refreshBusy = false;
    }
  }

  function requestManualRefresh() {
    return loadData({ manual: true });
  }

  function updateTaskInState(updatedTask, fallbackPath = '') {
    if (!data) return;
    const sameOrphan = (orphan) =>
      (fallbackPath &&
        (orphan.task_path === fallbackPath || orphan.task_path === updatedTask.relative_path)) ||
      (updatedTask.relative_path && orphan.task_path === updatedTask.relative_path) ||
      (updatedTask.id && orphan.task_id === updatedTask.id) ||
      (updatedTask.ref && orphan.task_ref === updatedTask.ref);
    data = {
      ...data,
      tasks: applyTaskUpdate(data.tasks ?? [], updatedTask, fallbackPath),
      orphaned_tasks:
        updatedTask.status && updatedTask.status !== 'doing'
          ? (data.orphaned_tasks ?? []).filter((orphan) => !sameOrphan(orphan))
          : (data.orphaned_tasks ?? [])
    };
    if (selectedTask && sameTaskRecord(selectedTask, updatedTask, fallbackPath)) {
      selectedTask = { ...selectedTask, ...updatedTask };
      draftStatus = selectedTask.status || '';
      draftPriority = priorityValue(selectedTask);
      draftAssignee = selectedTask.assignee || '';
      draftProject = selectedTask.project_id || selectedTask.project || '';
      draftDependsOn = dependsOnDraft(selectedTask);
      draftBody = selectedTask.body || '';
      draftComment = '';
    }
  }

  async function patchTask(path, patch) {
    const response = await apiFetch('/api/tasks', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path, ...patch })
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    const updatedTask = await response.json();
    const localTask = {
      ...updatedTask,
      ...(patch.status ? { status: patch.status } : {}),
      ...(patch.priority !== undefined ? { priority: patch.priority } : {}),
      ...(patch.assignee !== undefined ? { assignee: patch.assignee } : {}),
      ...(patch.depends_on !== undefined ? { depends_on: patch.depends_on } : {}),
      ...(patch.project
        ? { project: patch.project, project_id: updatedTask.project_id || patch.project }
        : {})
    };
    updateTaskInState(localTask, path);
    return localTask;
  }

  async function createTask(payload) {
    const response = await apiFetch('/api/tasks', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    const createdTask = await response.json();
    updateTaskInState(createdTask);
    return createdTask;
  }

  async function controlSession(session, action, input = '', targetStatus = '') {
    const payload = { claim_id: session.claim_id, action, input };
    if (targetStatus) {
      payload.target_status = targetStatus;
    }
    const response = await apiFetch('/api/sessions/control', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    await loadData();
  }

  async function releaseSession(session, action = 'release', targetStatus = 'needs_review') {
    try {
      error = '';
      await controlSession(session, action, '', targetStatus);
    } catch (err) {
      error = err.message;
    }
  }

  async function resolveOrphan(orphan, status) {
    const path = orphan.task_path;
    if (!path) return;
    try {
      await patchTask(path, {
        status,
        comment:
          status === 'blocked'
            ? `Marked blocked from orphaned runtime state: ${orphan.blocking_reason || orphan.reason || 'runtime session is not active'}`
            : `Resolved orphaned runtime state by moving task to ${status}.`,
        comment_author: operatorAssignee
      });
      await loadData();
    } catch (err) {
      error = err.message;
    }
  }

  function openTask(task) {
    selectedTask = task;
    creatingTask = false;
    draftStatus = task.status || '';
    draftPriority = priorityValue(task);
    draftAssignee = task.assignee || '';
    draftProject = task.project_id || task.project || '';
    draftDependsOn = dependsOnDraft(task);
    draftBody = task.body || '';
    draftComment = '';
    editError = '';
  }

  function openCreateTask() {
    selectedTask = null;
    creatingTask = true;
    createError = '';
  }

  function selectProject(projectId) {
    selectedProjectId = projectId;
    selectedWorkspaceId = '';
    if (globalNav !== 'work') globalNav = 'work';
    activeView = 'board';
  }

  function selectWorkspace(workspaceId) {
    selectedWorkspaceId = workspaceId;
    selectedProjectId = '';
    if (globalNav !== 'work') globalNav = 'work';
    // A workspace-level selection has no single project's Settings tab to
    // stay on — without this, switching from a project's Settings view
    // straight to a workspace kept activeView pinned to PROJECT_SETTINGS_VIEW.
    if (activeView === PROJECT_SETTINGS_VIEW) activeView = 'board';
  }

  function selectAllProjects() {
    selectedProjectId = '';
    selectedWorkspaceId = '';
    if (activeView === PROJECT_SETTINGS_VIEW) activeView = 'board';
  }

  function changeView(view) {
    activeView = view;
  }

  function selectQuickView(viewId) {
    selectedProjectId = '';
    selectedWorkspaceId = '';
    globalNav = 'work';
    if (viewId === 'archived') {
      activeView = 'archive';
      quickFilter = 'all';
    } else {
      activeView = 'board';
      quickFilter = viewId;
    }
  }

  function changeChip(chipId) {
    quickFilter = chipId;
    if (activeView !== 'board') activeView = 'board';
  }

  function selectNav(nav) {
    if (nav === 'agents') {
      settingsSection = 'agents';
      globalNav = 'settings';
      return;
    }
    if (nav === 'settings') {
      globalNav = nav;
      return;
    }
    globalNav = nav;
    if (
      nav === 'work' &&
      activeView !== 'board' &&
      activeView !== 'archive' &&
      activeView !== PROJECT_SETTINGS_VIEW
    ) {
      activeView = 'board';
    }
  }

  function selectSettingsSection(id) {
    settingsSection = id;
    globalNav = 'settings';
  }

  function dragLeft(dx) {
    leftWidth = Math.min(LEFT_WIDTH.max, Math.max(LEFT_WIDTH.min, leftWidth + dx));
    saveLeftWidth(leftWidth);
  }

  function dragRight(dx) {
    rightWidth = Math.min(RIGHT_WIDTH.max, Math.max(RIGHT_WIDTH.min, rightWidth - dx));
    saveRightWidth(rightWidth);
  }

  function toggleAttention() {
    rightPanels = { ...rightPanels, attentionOpen: !rightPanels.attentionOpen };
    saveRightPanels(rightPanels);
  }

  function toggleRecent() {
    rightPanels = { ...rightPanels, recentOpen: !rightPanels.recentOpen };
    saveRightPanels(rightPanels);
  }

  function persistSidebarScroll(value) {
    sidebarScrollTop = value;
    saveSidebarScroll(value);
  }

  $: saveAccordionOpen(accordionOpen);

  function closeTask() {
    selectedTask = null;
    editError = '';
  }

  function closeCreateTask() {
    creatingTask = false;
    createError = '';
  }

  async function saveTaskDetails() {
    if (!selectedTask || saving) return;
    const comment = draftComment.trim();
    // blocked -> needs_review is operator recovery; require a review reason.
    if (selectedTask.status === 'blocked' && draftStatus === 'needs_review' && !comment) {
      editError =
        'Add a review comment explaining why this blocked task is ready for needs_review.';
      return;
    }
    saving = true;
    editError = '';
    try {
      const currentProject = selectedTask.project_id || selectedTask.project || '';
      const nextProject = draftProject.trim();
      const patch = {
        status: draftStatus,
        priority: Number(draftPriority),
        assignee: draftAssignee.trim(),
        depends_on: parseDependsOnDraft(draftDependsOn),
        body: draftBody,
        comment,
        comment_author: operatorAssignee
      };
      if (nextProject && nextProject !== currentProject) {
        const project = projects.find((item) => item.id === nextProject);
        const repositoryOptions = project?.repositories?.length
          ? project.repositories
          : workspaces.find((workspace) => workspace.id === (project?.workspace_id || nextProject))
              ?.repositories || [];
        patch.project = nextProject;
        patch.repository = repositoryOptions[0] || '';
      }
      await patchTask(selectedTask.path, patch);
      closeTask();
    } catch (err) {
      editError = err.message;
    } finally {
      saving = false;
    }
  }

  async function saveCreatedTask(payload) {
    if (creating) return;
    creating = true;
    createError = '';
    try {
      const created = await createTask(payload);
      closeCreateTask();
      openTask(created);
    } catch (err) {
      createError = err.message;
    } finally {
      creating = false;
    }
  }

  async function transitionTask(task, status) {
    if (
      !task?.path ||
      !status ||
      !canTransitionStatus(task.status, status, data?.status_transitions)
    )
      return;
    const previous = task;
    try {
      actionError = '';
      updateTaskInState({ ...task, status }, task.path);
      await patchTask(task.path, { status });
    } catch (err) {
      actionError = err.message;
      updateTaskInState(previous, previous.path);
    }
  }

  function setThemePref(pref) {
    themePref = saveThemePref(pref);
    applyTheme(themePref);
  }

  function setRefreshMs(ms) {
    refreshMs = saveAutoRefreshMs(ms);
    restartRefresh();
  }

  function restartRefresh() {
    if (refreshTimer != null) window.clearInterval(refreshTimer);
    refreshTimer = window.setInterval(() => loadData(), refreshMs);
  }

  let managerSetupNeeded = false;

  async function loadManagerSetup() {
    try {
      const response = await apiFetch('/api/manager/setup', { cache: 'no-store' });
      if (!response.ok) return;
      const body = await response.json();
      managerSetupNeeded = Boolean(body?.needed);
    } catch {
      managerSetupNeeded = false;
    }
  }

  onMount(() => {
    applyTheme(themePref);
    loadData();
    loadManagerSetup();
    restartRefresh();
    const stopWatchingTheme = watchSystemTheme(() => {
      if (loadThemePref() === 'system') applyTheme('system');
    });
    return () => {
      if (refreshTimer != null) window.clearInterval(refreshTimer);
      clearRefreshBusyTimer();
      stopWatchingTheme();
    };
  });

  $: workspaces = data?.workspaces ?? [];
  $: projects = data?.projects ?? [];
  $: tasks = data?.tasks ?? [];
  $: sessionGroups = data?.session_groups ?? [];
  $: runtimeSessions = (data?.runtime_sessions ?? []).filter(sessionIsActive);
  $: orphanedTasks = data?.orphaned_tasks ?? [];
  $: closedSessions = flattenClosedSessions(sessionGroups, tasks);
  $: blockedTasks = tasks.filter((task) => task.status === 'blocked');
  $: reviewTasks = tasks.filter((task) => task.status === 'needs_review');
  $: waitingTasks = tasks.filter((task) => operatorPause(task).waiting);
  $: attentionTasks = [...waitingTasks, ...blockedTasks, ...reviewTasks];
  $: selectedProject =
    projects.find((project) => project.id === selectedProjectId) ||
    workspaces.find((workspace) => workspace.id === selectedWorkspaceId) ||
    null;
  $: assignees = [
    'all',
    ...Array.from(new Set(tasks.map((task) => assigneeLabel(task.assignee)))).sort()
  ];
  $: visibleTasks = tasks
    .filter((task) => {
      const identityMatch = selectedProjectId
        ? taskProjectId(task) === selectedProjectId
        : !selectedWorkspaceId || task.workspace_id === selectedWorkspaceId;
      const haystack =
        `${task.ref} ${task.title} ${task.type} ${task.project_id} ${task.project} ${task.repositories?.join(' ') ?? ''}`.toLowerCase();
      const queryMatch = !query.trim() || haystack.includes(query.trim().toLowerCase());
      return identityMatch && queryMatch;
    })
    .sort(taskSort === 'priority' ? taskPickupCompare : taskUpdatedCompare);
  $: activeTasks = visibleTasks.filter((task) => !isArchivedStatus(task.status));
  $: archivedTasks = visibleTasks.filter((task) => isArchivedStatus(task.status));
  $: liveSessions = [...runtimeSessions, ...orphanedTasks];
  $: backlogTasks = visibleTasks.filter((task) => taskInBacklog(task, liveSessions));
  $: fixedChips = [
    { id: 'all', label: 'All', count: backlogTasks.length },
    {
      id: 'open',
      label: 'Open',
      count: backlogTasks.filter((task) => task.status !== 'done').length
    },
    {
      id: 'rework',
      label: 'Rework',
      count: backlogTasks.filter((task) => task.status === 'needs_rework').length
    },
    {
      id: 'mine',
      label: 'Mine',
      count: backlogTasks.filter((task) => isOperatorAssignee(task.assignee)).length
    },
    {
      id: 'high',
      label: 'High',
      count: backlogTasks.filter((task) => priorityValue(task) <= 2).length
    },
    { id: 'bug', label: 'Bug', count: backlogTasks.filter((task) => task.type === 'bug').length },
    {
      id: 'feature',
      label: 'Feature',
      count: backlogTasks.filter((task) => task.type === 'feature').length
    }
  ];
  // The Sidebar's Views list can set quickFilter to a value with no matching
  // toolbar chip (blocked/review/done/attention) — without this, the chip row
  // showed no active state at all while the list was silently filtered.
  $: sidebarOnlyChipLabel = {
    blocked: 'Blocked',
    review: 'In Review',
    done: 'Done',
    attention: 'Needs Attention'
  }[quickFilter];
  $: chips = sidebarOnlyChipLabel
    ? [
        ...fixedChips,
        { id: quickFilter, label: sidebarOnlyChipLabel, count: filteredActiveTasks.length }
      ]
    : fixedChips;
  $: listTasks = ['blocked', 'review', 'done', 'attention'].includes(quickFilter)
    ? activeTasks
    : backlogTasks;
  $: filteredActiveTasks = listTasks.filter((task) => {
    switch (quickFilter) {
      case 'open':
        return task.status !== 'done';
      case 'rework':
        return task.status === 'needs_rework';
      case 'mine':
        return isOperatorAssignee(task.assignee);
      case 'high':
        return priorityValue(task) <= 2;
      case 'bug':
        return task.type === 'bug';
      case 'feature':
        return task.type === 'feature';
      case 'blocked':
        return task.status === 'blocked';
      case 'review':
        return task.status === 'needs_review';
      case 'done':
        return task.status === 'done';
      case 'attention':
        return (
          task.status === 'blocked' || task.status === 'needs_review' || operatorPause(task).waiting
        );
      default:
        return true;
    }
  });
</script>

<svelte:head>
  <title>Core Backoffice</title>
</svelte:head>

{#if managerSetupNeeded}
  <ManagerSetup onDone={() => (managerSetupNeeded = false)} />
{/if}

<main class="shell">
  {#if error}
    <section class="error">
      <strong>Backoffice data is not available.</strong>
      <span>{error}</span>
    </section>
  {:else if !data}
    <section class="loading">Loading Core state…</section>
  {:else}
    <div class="app-shell" style="--left-w: {leftWidth}px; --right-w: {rightWidth}px">
      <Sidebar
        {workspaces}
        {projects}
        {tasks}
        {selectedProjectId}
        {selectedWorkspaceId}
        {globalNav}
        {settingsSection}
        {themePref}
        quickView={activeView === 'archive' ? 'archived' : quickFilter}
        bind:accordionOpen
        onSelectProject={selectProject}
        onSelectWorkspace={selectWorkspace}
        onSelectAll={selectAllProjects}
        onSelectView={selectQuickView}
        onSelectNav={selectNav}
        onSelectSettingsSection={selectSettingsSection}
        onTreeScroll={persistSidebarScroll}
        onScrollEl={(el) => (sidebarScrollEl = el)}
        onThemeChange={setThemePref}
      />
      <SplitGutter axis="x" onDrag={dragLeft} />

      {#if isSettingsNav(globalNav)}
        <SettingsView
          section={settingsSection}
          {themePref}
          onThemeChange={setThemePref}
          {refreshMs}
          onRefreshChange={setRefreshMs}
        />
      {:else}
        <div class="working-area">
          <RuntimeSessions
            {runtimeSessions}
            {closedSessions}
            {orphanedTasks}
            {refreshStatus}
            onControlSession={controlSession}
            onReleaseSession={releaseSession}
            onResolveOrphan={resolveOrphan}
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
              onChangeView={changeView}
              onBack={selectAllProjects}
              onCreateTask={openCreateTask}
              onRefresh={requestManualRefresh}
              refreshing={refreshBusy}
            />

            {#if activeView === 'board'}
              <FiltersBar
                {chips}
                activeChip={quickFilter}
                bind:query
                sort={taskSort}
                onChipChange={changeChip}
                onSortChange={(value) => (taskSort = value)}
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
                  transitions={data.status_transitions}
                  onOpenTask={openTask}
                  onTransitionTask={transitionTask}
                />
              {:else if activeView === PROJECT_SETTINGS_VIEW && selectedProject}
                <ProjectSettings
                  project={selectedProject}
                  {workspaces}
                  {projects}
                  repositories={data.registry?.repositories || []}
                />
              {:else}
                <TaskList
                  tasks={filteredActiveTasks}
                  {selectedTask}
                  transitions={data.status_transitions}
                  onOpenTask={openTask}
                  onTransitionTask={transitionTask}
                />
              {/if}

              <footer class="generated">
                Generated {new Date(data.generated_at).toLocaleString()} from {data.registry
                  .repositories_count} repositories.
              </footer>
            </div>
          </div>
        </div>

        <SplitGutter axis="x" onDrag={dragRight} />

        <aside class="sidebar sidebar-right">
          <NeedsAttention
            tasks={attentionTasks}
            open={rightPanels.attentionOpen}
            transitions={data.status_transitions}
            onToggle={toggleAttention}
            onOpenTask={openTask}
            onShowAll={() => selectQuickView('attention')}
            onTransitionTask={transitionTask}
          />
          <RecentCompleted
            {tasks}
            open={rightPanels.recentOpen}
            transitions={data.status_transitions}
            onToggle={toggleRecent}
            onOpenTask={openTask}
            onShowAll={() => selectQuickView('done')}
            onTransitionTask={transitionTask}
          />
        </aside>
      {/if}
    </div>
  {/if}
</main>

{#if creatingTask && data}
  <CreateTaskModal
    {projects}
    {workspaces}
    statuses={data.statuses ?? []}
    {assignees}
    preferredProject={selectedProjectId || selectedWorkspaceId}
    saving={creating}
    {createError}
    onCreate={saveCreatedTask}
    onClose={closeCreateTask}
  />
{/if}

{#if selectedTask && data}
  <TaskModal
    task={selectedTask}
    {workspaces}
    {projects}
    statuses={data.statuses ?? []}
    {assignees}
    bind:draftStatus
    bind:draftPriority
    bind:draftAssignee
    bind:draftProject
    bind:draftDependsOn
    bind:draftBody
    bind:draftComment
    {saving}
    {editError}
    onSave={saveTaskDetails}
    onClose={closeTask}
  />
{/if}
