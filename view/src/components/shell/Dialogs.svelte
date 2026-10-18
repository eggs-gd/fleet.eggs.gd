<script lang="ts">
  import MilestoneModal from '../milestones/MilestoneModal.svelte';
  import CreateTaskModal from '../tasks/CreateTaskModal.svelte';
  import TaskModal from '../tasks/TaskModal.svelte';
  import { projects, statuses, workspaces } from '../../stores/board';
  import { selectedProjectId, selectedWorkspaceId } from '../../stores/location';
  import {
    closeCreateMilestone,
    closeEditMilestone,
    creatingMilestone,
    editingMilestone,
    milestoneCardId,
    milestoneError,
    saveCreatedMilestone,
    saveEditedMilestone,
    savingMilestone
  } from '../../stores/milestones';
  import {
    closeCreateTask,
    closeTask,
    createError,
    creating,
    creatingTask,
    draft,
    editError,
    saveCreatedTask,
    saveTaskDetails,
    saving,
    selectedTask
  } from '../../stores/tasks';
  import { assigneeChoices, selectedProject } from '../../stores/views';
</script>

{#if $creatingTask}
  <CreateTaskModal
    projects={$projects}
    workspaces={$workspaces}
    statuses={$statuses}
    assignees={$assigneeChoices}
    preferredProject={$selectedProjectId || $selectedWorkspaceId}
    saving={$creating}
    createError={$createError}
    onCreate={saveCreatedTask}
    onClose={closeCreateTask}
  />
{/if}

{#if $creatingMilestone && $milestoneCardId}
  <MilestoneModal
    projectTitle={$selectedProject?.title || $selectedProjectId}
    saving={$savingMilestone}
    createError={$milestoneError}
    onSubmit={saveCreatedMilestone}
    onClose={closeCreateMilestone}
  />
{:else if $editingMilestone}
  <MilestoneModal
    projectTitle={$editingMilestone.cardId}
    initial={{ title: $editingMilestone.title, version: $editingMilestone.version }}
    saving={$savingMilestone}
    createError={$milestoneError}
    onSubmit={saveEditedMilestone}
    onClose={closeEditMilestone}
  />
{/if}

{#if $selectedTask}
  <TaskModal
    task={$selectedTask}
    workspaces={$workspaces}
    projects={$projects}
    statuses={$statuses}
    assignees={$assigneeChoices}
    bind:draft={$draft}
    saving={$saving}
    editError={$editError}
    onSave={saveTaskDetails}
    onClose={closeTask}
  />
{/if}
