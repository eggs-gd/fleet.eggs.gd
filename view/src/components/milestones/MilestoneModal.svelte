<script lang="ts">
  import { focusDialog } from '../../lib/dialog';

  export let projectTitle = '';
  export let saving = false;
  export let createError = '';
  /** Set to edit an existing milestone; leave unset to create one. */
  export let initial: { title: string; version: string } | null = null;
  export let onSubmit: (title: string, version: string) => Promise<void> = async () => {};
  export let onClose: () => void = () => {};

  let draftTitle = initial?.title ?? '';
  let draftVersion = initial?.version ?? '';
  const editing = initial !== null;

  function requestClose() {
    if (saving) return;
    if (changed() && !window.confirm('Discard your changes?')) {
      return;
    }
    onClose();
  }

  function changed() {
    if (initial)
      return draftTitle.trim() !== initial.title || draftVersion.trim() !== initial.version;
    return Boolean(draftTitle.trim() || draftVersion.trim());
  }

  function closeFromBackdrop(event: MouseEvent) {
    if (event.target === event.currentTarget) requestClose();
  }

  async function submit() {
    if (!draftTitle.trim()) return;
    await onSubmit(draftTitle.trim(), draftVersion.trim());
  }
</script>

<section class="modal-backdrop" role="presentation" on:click={closeFromBackdrop}>
  <div
    class="task-modal milestone-modal"
    role="dialog"
    aria-modal="true"
    aria-labelledby="milestone-modal-title"
    tabindex="-1"
    use:focusDialog={requestClose}
  >
    <header>
      <div>
        <p class="eyebrow">{projectTitle || 'New milestone'}</p>
        <h2 id="milestone-modal-title">{editing ? 'Edit milestone' : 'Add milestone'}</h2>
      </div>
      <div class="task-modal-actions">
        <button
          type="button"
          class="primary-action"
          disabled={saving || !draftTitle.trim()}
          on:click={submit}
        >
          {saving ? 'Saving…' : editing ? 'Save' : 'Create'}
        </button>
        <button type="button" disabled={saving} on:click={requestClose}>Cancel</button>
      </div>
    </header>

    <label class="body-editor">
      <span>Title</span>
      <input
        bind:value={draftTitle}
        name="title"
        autocomplete="off"
        placeholder="Short milestone title…"
      />
    </label>

    <label class="body-editor">
      <span>Version (optional)</span>
      <input bind:value={draftVersion} placeholder="v0.1.0" />
    </label>

    {#if !editing}
      <p class="create-hint">
        Starts parked. Tasks opt in by drag-and-drop or by setting <code>milestone</code>; activate
        it later from the board.
      </p>
    {/if}

    {#if createError}
      <p class="edit-error">{createError}</p>
    {/if}
  </div>
</section>
