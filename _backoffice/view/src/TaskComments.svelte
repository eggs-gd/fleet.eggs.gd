<script>
  export let comments = [];
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
