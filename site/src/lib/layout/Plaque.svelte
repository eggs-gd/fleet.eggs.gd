<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    id,
    class: className = '',
    'aria-labelledby': ariaLabelledby,
    children
  }: {
    id?: string;
    class?: string;
    'aria-labelledby'?: string;
    children: Snippet;
  } = $props();
</script>

<article class="plaque {className}" {id} aria-labelledby={ariaLabelledby}>
  {@render children()}
</article>
