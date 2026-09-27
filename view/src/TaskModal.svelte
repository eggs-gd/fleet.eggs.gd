<script lang="ts">
  import { tick } from 'svelte';
  import {
    blockedReason,
    codexRemoteInstruction,
    copyText,
    dependsOnList,
    execution,
    executionClass,
    executionIdentity,
    executionLabel,
    executionReason,
    launchEvaluation,
    launchOutcome,
    launchReason,
    providerErrorKind,
    providerErrorReason,
    providerErrorRetryPolicy,
    providerErrorSuggestedAction,
    showExecutionSignal,
    toolUsageWarning,
    workspaceTitle
  } from './lib/taskDisplay';
  import TaskComments from './TaskComments.svelte';
  import type { Project, Task, Workspace } from './lib/types';

  export let task: Task;
  export let workspaces: Workspace[] = [];
  export let projects: Project[] = [];
  export let statuses: string[] = [];
  export let assignees: string[] = [];
  export let draftStatus = '';
  export let draftPriority = 5;
  export let draftAssignee = '';
  export let draftProject = '';
  export let draftDependsOn = '';
  export let draftBody = '';
  export let draftComment = '';
  export let saving = false;
  export let editError = '';
  export let onSave: () => Promise<void> = async () => {};
  export let onClose: () => void = () => {};

  let commentField: HTMLTextAreaElement;

  $: sortedProjects = [...projects].sort((a, b) =>
    (a.title || a.id).localeCompare(b.title || b.id)
  );
  $: selectedProject = sortedProjects.find((project) => project.id === draftProject) || null;
  $: selectedWorkspaceId =
    selectedProject?.workspace_id || selectedProject?.id || task?.workspace_id || '';
  $: currentDependsOn = dependsOnList(task);

  function resizeCommentField() {
    if (!commentField) {
      return;
    }
    commentField.style.height = 'auto';
    commentField.style.height = `${commentField.scrollHeight}px`;
  }

  // Grow on typing/paste and when an existing longer draft is restored.
  $: (draftComment, tick().then(resizeCommentField));

  function closeFromBackdrop(event: MouseEvent) {
    if (event.target === event.currentTarget) {
      onClose();
    }
  }
</script>

<section class="modal-backdrop" role="presentation" on:click={closeFromBackdrop}>
  <div
    class="task-modal"
    role="dialog"
    aria-modal="true"
    aria-labelledby="task-title"
    tabindex="-1"
  >
    <header>
      <div>
        <p class="eyebrow">{task.ref || task.type}</p>
        <h2 id="task-title">{task.title}</h2>
      </div>
      <div class="task-modal-actions">
        <button type="button" class="primary-action" disabled={saving} on:click={onSave}>
          {saving ? 'Saving...' : 'Save'}
        </button>
        <button type="button" disabled={saving} on:click={onClose}>Close</button>
      </div>
    </header>

    <section class="detail-grid">
      <label>
        <span>Status</span>
        <select bind:value={draftStatus}>
          {#each statuses as status (status)}
            <option value={status}>{status}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Priority</span>
        <select bind:value={draftPriority}>
          {#each [1, 2, 3, 4, 5] as priority (priority)}
            <option value={priority}>P{priority}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Project</span>
        <select bind:value={draftProject}>
          {#each sortedProjects as project (project.id)}
            <option value={project.id}>{project.title || project.id}</option>
          {/each}
          {#if draftProject && !sortedProjects.some((project) => project.id === draftProject)}
            <option value={draftProject}>{draftProject}</option>
          {/if}
        </select>
      </label>
      <div>
        <span>Workspace</span>
        <strong>{workspaceTitle(workspaces, selectedWorkspaceId)}</strong>
      </div>
      <div>
        <span>Launch</span>
        <strong>{launchOutcome(task) || 'not evaluated'}</strong>
      </div>
      <div>
        <span>Execution</span>
        <strong>{executionLabel(task)}</strong>
      </div>
      <div>
        <span>Agent</span>
        <strong
          >{launchEvaluation(task).agent ||
            task.launch?.agent ||
            task.assignee ||
            'unassigned'}</strong
        >
      </div>
      <label>
        <span>Assignee</span>
        <select bind:value={draftAssignee}>
          {#each assignees.filter((assignee) => assignee !== 'all') as assignee (assignee)}
            <option value={assignee}>{assignee}</option>
          {/each}
          {#if !assignees.includes(draftAssignee)}
            <option value={draftAssignee}>{draftAssignee || 'unassigned'}</option>
          {/if}
        </select>
      </label>
    </section>

    <label class="body-editor depends-editor">
      <span>Depends on (hard launch blockers)</span>
      <input bind:value={draftDependsOn} placeholder="FLET-144, FLET-145" spellcheck="false" />
      <p class="depends-hint">
        Comma-separated prerequisite refs. Daemon will not claim/launch this task until each
        dependency is <code>done</code>. Leave empty for no gate. Waiting stays in launch evaluation
        — it does not flip status to <code>blocked</code>.
      </p>
      {#if currentDependsOn.length}
        <div class="depends-chips" aria-label="Current depends_on">
          {#each currentDependsOn as ref (ref)}
            <code>{ref}</code>
          {/each}
        </div>
      {/if}
    </label>

    {#if launchReason(task)}
      <section class="launch-detail">
        <span>Launch evaluation</span>
        <p>{launchReason(task)}</p>
      </section>
    {/if}

    {#if showExecutionSignal(task)}
      <section class={`execution-detail ${executionClass(task)}`}>
        <span>Execution identity</span>
        <div class="session-meta">
          {#each executionIdentity(execution(task)) as item (item)}
            <code>{item}</code>
          {/each}
          {#if execution(task).backend}
            <span>{execution(task).backend}</span>
          {/if}
          {#if execution(task).last_event}
            <span>{execution(task).last_event}</span>
          {/if}
          {#if execution(task).last_activity_at}
            <span>last {new Date(execution(task).last_activity_at!).toLocaleString()}</span>
          {/if}
          {#if execution(task).last_event_at}
            <span>event {new Date(execution(task).last_event_at!).toLocaleString()}</span>
          {/if}
          {#if execution(task).last_output_at}
            <span>output {new Date(execution(task).last_output_at!).toLocaleString()}</span>
          {/if}
          {#if execution(task).last_status_change_at}
            <span>status {new Date(execution(task).last_status_change_at!).toLocaleString()}</span>
          {/if}
        </div>
        {#if execution(task).provider === 'codex' || execution(task).backend === 'codex-app-server'}
          <p>{codexRemoteInstruction(execution(task))}</p>
          <div class="session-actions">
            {#if execution(task).thread_id}
              <button type="button" on:click={() => copyText(execution(task).thread_id)}
                >Copy thread id</button
              >
            {/if}
            {#if execution(task).turn_id}
              <button type="button" on:click={() => copyText(execution(task).turn_id)}
                >Copy turn id</button
              >
            {/if}
            {#if execution(task).thread_title}
              <button type="button" on:click={() => copyText(execution(task).thread_title)}
                >Copy title</button
              >
            {/if}
          </div>
        {/if}
        {#if executionReason(task)}
          <p>{executionReason(task)}</p>
        {/if}
        {#if providerErrorReason(execution(task))}
          <section class="provider-error" aria-label="Provider error">
            <strong>{providerErrorKind(execution(task)) || 'provider_error'}</strong>
            <p>{providerErrorReason(execution(task))}</p>
            {#if providerErrorSuggestedAction(execution(task))}
              <p>{providerErrorSuggestedAction(execution(task))}</p>
            {/if}
            {#if providerErrorRetryPolicy(execution(task))}
              <p>Retry policy: {providerErrorRetryPolicy(execution(task))}</p>
            {/if}
          </section>
        {/if}
        {#if toolUsageWarning(execution(task))}
          <section class="tool-evidence" aria-label="Tool usage warning">
            <strong>Tool evidence</strong>
            <p class="tool-warning-text">{toolUsageWarning(execution(task))}</p>
          </section>
        {/if}
      </section>
    {/if}

    {#if blockedReason(task)}
      <section class="blocked-detail">
        <span>Needs you</span>
        <p>{blockedReason(task)}</p>
      </section>
    {/if}

    <TaskComments comments={task.comments} />

    <label class="body-editor comment-editor">
      <span>
        Add Review Comment
        {#if task.status === 'blocked' && draftStatus === 'needs_review'}
          <em>(required for blocked → needs_review)</em>
        {/if}
      </span>
      <textarea
        bind:this={commentField}
        bind:value={draftComment}
        rows="3"
        class="comment-input"
        placeholder={task.status === 'blocked' && draftStatus === 'needs_review'
          ? 'Why is this blocked task ready for review without re-running?'
          : 'What should the next worker fix?'}
        on:input={resizeCommentField}></textarea>
    </label>

    <label class="body-editor">
      <span>Description</span>
      <textarea bind:value={draftBody} rows="18"></textarea>
    </label>

    {#if editError}
      <p class="edit-error">{editError}</p>
    {/if}

    <footer>
      <code>{task.relative_path}</code>
    </footer>
  </div>
</section>

<style>
  .depends-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-sm);
    margin-top: var(--space-md);
    align-items: center;
  }

  .depends-chips code {
    font-size: var(--text-sm);
    padding: var(--space-2xs) var(--space-sm);
    border: 1px solid var(--border);
    border-radius: 4px;
    background: var(--surface-muted);
  }

  .comment-editor textarea.comment-input {
    min-height: calc(1.4em * 3 + 20px);
    height: auto;
    max-height: min(40vh, 280px);
    overflow-y: auto;
    resize: none;
    line-height: 1.4;
    field-sizing: content;
  }

  .launch-detail {
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--surface-muted);
    margin: 0 0 var(--space-lg);
    padding: var(--space-lg);
  }

  .launch-detail span,
  .execution-detail > span {
    display: block;
    color: var(--text-muted);
    font-size: var(--text-sm);
    font-weight: 700;
    margin-bottom: var(--space-sm);
    text-transform: uppercase;
  }

  .launch-detail p {
    margin: 0;
  }

  .execution-detail {
    border: 1px solid var(--border);
    border-left-width: 3px;
    border-radius: var(--radius-sm);
    background: var(--surface-muted);
    margin: 0 0 var(--space-lg);
    padding: var(--space-lg);
  }

  .execution-detail p {
    margin: var(--space-md) 0 0;
    color: var(--text);
    line-height: 1.45;
  }

  .execution-live {
    border-left-color: var(--green);
    background: var(--green-soft);
  }

  .execution-waiting {
    border-left-color: var(--amber);
    background: var(--amber-soft);
  }

  .execution-orphaned,
  .execution-failed {
    border-left-color: var(--red);
    background: var(--red-soft);
  }

  .execution-complete {
    border-left-color: var(--blue);
    background: var(--blue-soft);
  }

  .execution-none {
    border-left-color: var(--border-strong);
  }

  .blocked-detail {
    border: 1px solid var(--error-border);
    border-radius: var(--radius-sm);
    background: var(--red-soft);
    margin: 0 0 var(--space-lg);
    padding: var(--space-lg);
  }

  .blocked-detail span {
    display: block;
    color: var(--red-text);
    font-size: var(--text-sm);
    font-weight: 700;
    margin-bottom: var(--space-sm);
    text-transform: uppercase;
  }

  .blocked-detail p {
    margin: 0;
    color: var(--red-text);
    line-height: 1.45;
  }
</style>
