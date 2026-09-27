<script lang="ts">
  export let title = '';
  export let body = '';
</script>

<section class="stub-view" aria-label={title}>
  <h2>{title}</h2>
  <p>{body}</p>
</section>

<style>
  .stub-view {
    flex: 1 1 auto;
    min-width: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    background: var(--surface);
    padding: var(--space-2xl);
  }

  .stub-view h2 {
    margin: 0 0 var(--space-md);
  }

  .stub-view p {
    margin: 0;
    color: var(--text-muted);
  }
</style>
