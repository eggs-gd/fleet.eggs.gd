<script>
  const taskTypes = ['feature', 'bug', 'research', 'review', 'maintenance', 'decision'];
  const defaultAssignees = ['unassigned', 'owner', 'claude', 'codex', 'cursor', 'gemini'];

  export let projects = [];
  export let workspaces = [];
  export let statuses = [];
  export let assignees = [];
  export let preferredProject = '';
  export let saving = false;
  export let createError = '';
  export let onCreate = async () => {};
  export let onClose = () => {};

  let draftTitle = '';
  let draftRequest = '';
  let draftProject = preferredProject || '';
  let draftRepository = '';
  let draftStatus = 'backlog';
  let draftType = 'feature';
  let draftAssignee = 'unassigned';
  let draftPriority = 5;
  let draftDependsOn = '';

  $: sortedProjects = [...projects].sort((a, b) =>
    (a.title || a.id).localeCompare(b.title || b.id)
  );
  $: selectedProject = sortedProjects.find((project) => project.id === draftProject) || null;
  $: repositoryOptions = selectedProject?.repositories?.length
    ? selectedProject.repositories
    : workspaces.find(
        (workspace) => workspace.id === (selectedProject?.workspace_id || draftProject)
      )?.repositories || [];
  $: assigneeOptions = Array.from(
    new Set([
      ...defaultAssignees,
      ...assignees.filter((assignee) => assignee && assignee !== 'all')
    ])
  );
  $: statusOptions = statuses.length
    ? statuses
    : ['backlog', 'todo', 'needs_rework', 'doing', 'blocked', 'needs_review', 'done', 'archived'];
  $: if (!draftProject && sortedProjects.length) {
    const preferred =
      preferredProject &&
      sortedProjects.find(
        (project) => project.id === preferredProject || project.workspace_id === preferredProject
      );
    draftProject = preferred?.id || sortedProjects[0].id;
  }
  $: if (draftProject && repositoryOptions.length && !repositoryOptions.includes(draftRepository)) {
    draftRepository = repositoryOptions[0] || '';
  }
  $: if (!repositoryOptions.length) {
    draftRepository = '';
  }

  function closeFromBackdrop(event) {
    if (event.target === event.currentTarget) {
      onClose();
    }
  }

  function parseDependsOn(text) {
    const seen = new Set();
    const out = [];
    for (const part of String(text || '').split(/[\n,]+/)) {
      const ref = part.trim();
      if (!ref) continue;
      const key = ref.toUpperCase();
      if (seen.has(key)) continue;
      seen.add(key);
      out.push(ref);
    }
    return out;
  }

  async function submit() {
    await onCreate({
      title: draftTitle.trim(),
      request: draftRequest.trim(),
      project: draftProject,
      repository: draftRepository,
      status: draftStatus,
      type: draftType,
      assignee: draftAssignee,
      priority: Number(draftPriority),
      depends_on: parseDependsOn(draftDependsOn)
    });
  }
</script>

<section class="modal-backdrop" role="presentation" on:click={closeFromBackdrop}>
  <div
    class="task-modal create-task-modal"
    role="dialog"
    aria-modal="true"
    aria-labelledby="create-task-title"
    tabindex="-1"
  >
    <header>
      <div>
        <p class="eyebrow">New task</p>
        <h2 id="create-task-title">Create Core task</h2>
      </div>
      <div class="task-modal-actions">
        <button type="button" class="primary-action" disabled={saving} on:click={submit}>
          {saving ? 'Creating...' : 'Create'}
        </button>
        <button type="button" disabled={saving} on:click={onClose}>Cancel</button>
      </div>
    </header>

    <label class="body-editor">
      <span>Title</span>
      <input bind:value={draftTitle} placeholder="Short task title" />
    </label>

    <label class="body-editor">
      <span>Request / description</span>
      <textarea bind:value={draftRequest} rows="8" placeholder="What should be done?"></textarea>
    </label>

    <section class="detail-grid create-detail-grid">
      <label>
        <span>Project / workspace</span>
        <select bind:value={draftProject}>
          {#each sortedProjects as project (project.id)}
            <option value={project.id}>{project.title || project.id}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Repository</span>
        <select bind:value={draftRepository} disabled={!repositoryOptions.length}>
          {#if !repositoryOptions.length}
            <option value="">No repositories listed</option>
          {:else}
            {#each repositoryOptions as repository (repository)}
              <option value={repository}>{repository}</option>
            {/each}
          {/if}
        </select>
      </label>
      <label>
        <span>Status</span>
        <select bind:value={draftStatus}>
          {#each statusOptions as status (status)}
            <option value={status}>{status}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Type</span>
        <select bind:value={draftType}>
          {#each taskTypes as type (type)}
            <option value={type}>{type}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Assignee</span>
        <select bind:value={draftAssignee}>
          {#each assigneeOptions as assignee (assignee)}
            <option value={assignee}>{assignee}</option>
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
    </section>

    <label class="body-editor depends-editor">
      <span>Depends on (optional hard blockers)</span>
      <input bind:value={draftDependsOn} placeholder="FLET-144, FLET-145" spellcheck="false" />
      <p class="depends-hint">
        Prerequisite refs that must reach <code>done</code> before daemon launch. Written as
        <code>depends_on</code> frontmatter — not advisory prose.
      </p>
    </label>

    <p class="create-hint">
      Creates a durable Markdown task under <code>Work/&lt;project&gt;/tasks/</code> with the next
      <code>TAG-N</code> ref. Defaults stay off the launch queue (<code>backlog</code>) and do not
      commit, push, or open a PR.
    </p>

    {#if createError}
      <p class="edit-error">{createError}</p>
    {/if}
  </div>
</section>
