<script lang="ts">
  /** Developer-level remarks from the server, kept out of the way until asked for. */
  export let notes: string[] = [];
</script>

{#if notes?.length}
  <details class="settings-tech">
    <summary>Technical notes</summary>
    <ul class="settings-notes">
      {#each notes as note}<li>{note}</li>{/each}
    </ul>
  </details>
{/if}
