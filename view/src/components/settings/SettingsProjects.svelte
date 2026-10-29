<script lang="ts">
  import type { AnyRecord } from '../../lib/types';

  export let projects: AnyRecord = {};
  export let scanRoots: string[] = [];
  export let scanDirty = false;
  export let onScanRoots: (roots: string[]) => void = () => {};
  export let onRescan: () => void = () => {};

  $: scan = projects.scan || {};
  $: scanning = scan.state === 'scanning';

  function editRoot(index: number, value: string) {
    const next = scanRoots.slice();
    next[index] = value;
    onScanRoots(next);
  }

  function removeRoot(index: number) {
    onScanRoots(scanRoots.filter((_, i) => i !== index));
  }

  function addRoot() {
    onScanRoots([...scanRoots, '']);
  }
</script>

<section class="settings-block" aria-label="Project folders">
  <h3>Project folders</h3>
  <p class="settings-lede">
    Folders that hold your git repositories. Fleet scans them for projects and keeps watching: new
    repositories appear on their own. Save after changing the list.
  </p>
  {#each scanRoots as root, index (index)}
    <div class="settings-row">
      <span class="settings-key">Folder {index + 1}</span>
      <span class="settings-val">
        <input
          class="settings-input is-mono"
          type="text"
          value={root}
          on:input={(event) => editRoot(index, event.currentTarget.value)}
        />
        <button type="button" class="settings-btn" on:click={() => removeRoot(index)}>Remove</button
        >
      </span>
    </div>
  {/each}
  <div class="settings-row">
    <span class="settings-key"></span>
    <span class="settings-val">
      <button type="button" class="settings-btn" on:click={addRoot}>Add folder</button>
    </span>
  </div>
  <div class="settings-row">
    <span class="settings-key">Scan now</span>
    <span class="settings-val">
      <button
        type="button"
        class="settings-btn"
        disabled={scanning || scanDirty}
        on:click={onRescan}
      >
        {scanning ? 'Scanning…' : 'Scan now'}
      </button>
      <span class="settings-item-meta"
        >{scan.state || 'idle'}{scan.finished_at ? ` · ${scan.finished_at}` : ''}</span
      >
    </span>
  </div>
  {#if scanDirty}
    <p class="settings-hint">Save the folders before scanning.</p>
  {/if}
  {#if scan.error}
    <p class="settings-warn">{scan.error}</p>
  {/if}
</section>
