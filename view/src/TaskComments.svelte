<script lang="ts">
  import type { TaskComment } from './lib/types';

  export let comments: TaskComment[] = [];
</script>

<section class="comments">
  <header>
    <h3>Review Comments</h3>
    <span>{comments?.length ?? 0}</span>
  </header>
  {#each comments ?? [] as comment, index (`${comment.created_at || ''}:${comment.author || ''}:${index}`)}
    <article>
      <header>
        <strong>{comment.author || 'unknown'}</strong>
        <span>{comment.created_at ? new Date(comment.created_at).toLocaleString() : ''}</span>
      </header>
      <p>{comment.text}</p>
    </article>
  {:else}
    <p class="empty">No review comments yet.</p>
  {/each}
</section>

<style>
  .comments {
    display: grid;
    gap: var(--space-lg);
    margin: var(--space-xs) 0 var(--space-lg);
  }

  .comments > header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-lg);
  }

  .comments > header span {
    display: block;
    color: var(--text-muted);
    font-size: var(--text-sm);
    font-weight: 700;
    margin-bottom: var(--space-sm);
    text-transform: uppercase;
  }

  .comments h3 {
    margin: 0;
    font-size: var(--text-lg);
  }

  .comments article {
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--surface-muted);
    padding: var(--space-lg);
  }

  .comments article header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-lg);
  }

  .comments article span {
    color: var(--text-muted);
    font-size: var(--text-sm);
  }

  .comments article p {
    margin: var(--space-md) 0 0;
    color: var(--text);
    font-size: var(--text-md);
    line-height: 1.45;
  }
</style>
