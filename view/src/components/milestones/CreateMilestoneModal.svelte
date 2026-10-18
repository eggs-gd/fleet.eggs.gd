<script lang="ts">
  import { focusDialog } from '../../lib/dialog';

  export let projectTitle = '';
  export let saving = false;
  export let createError = '';
  export let onCreate: (title: string, version: string) => Promise<void> = async () => {};
  export let onClose: () => void = () => {};

  let draftTitle = '';
  let draftVersion = '';

  function requestClose() {
    if (saving) return;
    if ((draftTitle.trim() || draftVersion.trim()) && !window.confirm('Discard this milestone?')) {
      return;
    }
    onClose();
  }

  function closeFromBackdrop(event: MouseEvent) {
    if (event.target === event.currentTarget) requestClose();
  }

  async function submit() {
    if (!draftTitle.trim()) return;
    await onCreate(draftTitle.trim(), draftVersion.trim());
  }
</script>

<section class="modal-backdrop" role="presentation" on:click={closeFromBackdrop}>
  <div
    class="task-modal create-milestone-modal"
    role="dialog"
    aria-modal="true"
    aria-labelledby="create-milestone-title"
    tabindex="-1"
    use:focusDialog={requestClose}
  >
    <header>
      <div>
        <p class="eyebrow">{projectTitle || 'New milestone'}</p>
        <h2 id="create-milestone-title">Add milestone</h2>
      </div>
      <div class="task-modal-actions">
        <button
          type="button"
          class="primary-action"
          disabled={saving || !draftTitle.trim()}
          on:click={submit}
        >
          {saving ? 'Creating…' : 'Create'}
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

    <p class="create-hint">
      Starts parked. Tasks opt in by drag-and-drop or by setting <code>milestone</code>; activate it
      later from the board.
    </p>

    {#if createError}
      <p class="edit-error">{createError}</p>
    {/if}
  </div>
</section>
